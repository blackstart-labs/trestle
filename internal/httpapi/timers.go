package httpapi

import (
	"errors"
	"net/http"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/maruf-pfc/trestle/internal/projects"
)

type startTimerRequest struct {
	TaskID        *string `json:"task_id"`
	Note          *string `json:"note"`
	ReplaceActive *bool   `json:"replace_active"`
}

type timerIDRequest struct {
	TimerID *string `json:"timer_id"`
}

type stopTimerRequest struct {
	TimerID *string `json:"timer_id"`
	Note    *string `json:"note"`
}

type timerNoteRequest struct {
	TimerID *string `json:"timer_id"`
	Note    *string `json:"note"`
}

func parseTimerID(w http.ResponseWriter, raw *string) (uuid.UUID, bool) {
	if raw == nil {
		badRequest(w, "Timer id is required")
		return uuid.Nil, false
	}
	id, err := uuid.Parse(*raw)
	if err != nil {
		badRequest(w, "Invalid timer id")
		return uuid.Nil, false
	}
	return id, true
}

func validTimerNote(note string) bool {
	return utf8.RuneCountInString(note) <= 2000
}

func (s *Server) getTimer(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	timer, err := projects.GetActiveTimer(r.Context(), s.Pool, session.User.ID)
	if err != nil {
		internal(w, "failed to read running timer", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"timer":      timer,
		"server_now": time.Now().UTC(),
	})
}

func (s *Server) startTimer(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	var req startTimerRequest
	if !decodeJSON(w, r, &req) || !required(w, req.TaskID) {
		return
	}
	taskID, err := uuid.Parse(*req.TaskID)
	if err != nil {
		badRequest(w, "Invalid task id")
		return
	}
	note := ""
	if req.Note != nil {
		note = strings.TrimSpace(*req.Note)
	}
	if !validTimerNote(note) {
		badRequest(w, "Timer note must be at most 2000 characters")
		return
	}
	replace := req.ReplaceActive != nil && *req.ReplaceActive

	task, err := projects.LookupTimerTask(r.Context(), s.Pool, taskID)
	if errors.Is(err, projects.ErrNotFound) {
		notFound(w)
		return
	}
	if err != nil {
		internal(w, "failed to read task for timer", err)
		return
	}
	role, ok := s.membership(w, r, task.WorkspaceID, session.User.ID)
	if !ok {
		return
	}
	if role == "guest" {
		forbidden(w)
		return
	}
	if task.ProjectArchived {
		conflict(w, "This project is archived. Restore it before tracking time.")
		return
	}

	timer, err := projects.StartTimer(r.Context(), s.Pool, session.User.ID, task.WorkspaceID, taskID, note, replace)
	switch {
	case errors.Is(err, projects.ErrTimerActive):
		active, readErr := projects.GetActiveTimer(r.Context(), s.Pool, session.User.ID)
		body := map[string]any{
			"error": map[string]string{
				"code":    "timer_active",
				"message": "Another timer is already running. Stop it or choose to replace it.",
			},
		}
		if readErr == nil && active != nil {
			body["timer"] = active
		}
		writeJSON(w, http.StatusConflict, body)
		return
	case errors.Is(err, projects.ErrTimerTaskUnavailable):
		conflict(w, "This project is archived. Restore it before tracking time.")
		return
	case errors.Is(err, projects.ErrNotFound):
		notFound(w)
		return
	case err != nil:
		internal(w, "failed to start timer", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{
		"timer":      timer,
		"server_now": time.Now().UTC(),
	})
}

func (s *Server) stopTimer(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	var req stopTimerRequest
	if !decodeJSON(w, r, &req) || !required(w, req.TimerID) {
		return
	}
	timerID, ok := parseTimerID(w, req.TimerID)
	if !ok {
		return
	}
	var note *string
	if req.Note != nil {
		trimmed := strings.TrimSpace(*req.Note)
		if !validTimerNote(trimmed) {
			badRequest(w, "Timer note must be at most 2000 characters")
			return
		}
		note = &trimmed
	}

	log, total, err := projects.StopTimer(r.Context(), s.Pool, session.User.ID, timerID, note)
	switch {
	case errors.Is(err, projects.ErrNotFound):
		notFound(w)
		return
	case errors.Is(err, projects.ErrTimerDiscarded):
		conflict(w, "This timer was discarded. Start a new timer to log time.")
		return
	case errors.Is(err, projects.ErrTimerStopped):
		conflict(w, "Time is already recorded for this timer.")
		return
	case errors.Is(err, projects.ErrTimerTooLong):
		writeError(w, http.StatusUnprocessableEntity, "timer_too_long",
			"This timer has run longer than 240 hours. Log the time manually, then discard the timer.")
		return
	case err != nil:
		internal(w, "failed to stop timer", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"time_log":    log,
		"spent_hours": total,
	})
}

func (s *Server) discardTimer(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	var req timerIDRequest
	if !decodeJSON(w, r, &req) || !required(w, req.TimerID) {
		return
	}
	timerID, ok := parseTimerID(w, req.TimerID)
	if !ok {
		return
	}
	err := projects.DiscardTimer(r.Context(), s.Pool, session.User.ID, timerID)
	switch {
	case errors.Is(err, projects.ErrNotFound):
		notFound(w)
		return
	case errors.Is(err, projects.ErrTimerStopped):
		conflict(w, "Time is already recorded for this timer.")
		return
	case err != nil:
		internal(w, "failed to discard timer", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "discarded"})
}

func (s *Server) updateTimerNote(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	var req timerNoteRequest
	if !decodeJSON(w, r, &req) || !required(w, req.TimerID, req.Note) {
		return
	}
	timerID, ok := parseTimerID(w, req.TimerID)
	if !ok {
		return
	}
	note := strings.TrimSpace(*req.Note)
	if !validTimerNote(note) {
		badRequest(w, "Timer note must be at most 2000 characters")
		return
	}
	timer, err := projects.UpdateTimerNote(r.Context(), s.Pool, session.User.ID, timerID, note)
	switch {
	case errors.Is(err, projects.ErrNotFound):
		notFound(w)
		return
	case errors.Is(err, projects.ErrTimerStopped):
		conflict(w, "Time is already recorded for this timer.")
		return
	case errors.Is(err, projects.ErrTimerDiscarded):
		conflict(w, "This timer was discarded.")
		return
	case err != nil:
		internal(w, "failed to update timer note", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"timer": timer})
}

func (s *Server) pauseTimer(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	var req struct {
		TimerID *string `json:"timer_id"`
		Paused  *bool   `json:"paused"`
	}
	if !decodeJSON(w, r, &req) || !required(w, req.TimerID, req.Paused) {
		return
	}
	id, ok := parseTimerID(w, req.TimerID)
	if !ok {
		return
	}
	timer, err := projects.SetTimerPaused(r.Context(), s.Pool, session.User.ID, id, *req.Paused)
	if errors.Is(err, projects.ErrNotFound) {
		notFound(w)
		return
	}
	if errors.Is(err, projects.ErrTimerStopped) {
		conflict(w, "This timer is no longer active. Reload timer state.")
		return
	}
	if err != nil {
		internal(w, "failed to pause or resume timer", err)
		return
	}
	writeJSON(w, 200, map[string]any{"timer": timer, "server_now": time.Now().UTC()})
}
