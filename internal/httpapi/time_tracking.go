package httpapi

import (
	"net/http"

	"github.com/maruf-pfc/trestle/internal/projects"
)

type createTimeLogRequest struct {
	Hours       *float64 `json:"hours"`
	LoggedDate  *string  `json:"logged_date"`
	Description *string  `json:"description"`
}

func (s *Server) listTaskTimeLogs(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "project_id", "task_id")
	if !ok {
		return
	}
	wsID, pID, tID := ids[0], ids[1], ids[2]

	logs, err := projects.ListTaskTimeLogs(r.Context(), s.Pool, wsID, pID, tID)
	if err != nil {
		storeError(w, err, "failed to list task time logs", "")
		return
	}
	if logs == nil {
		logs = []projects.TimeLog{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"time_logs": logs})
}

func (s *Server) createTaskTimeLog(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "project_id", "task_id")
	if !ok {
		return
	}
	wsID, pID, tID := ids[0], ids[1], ids[2]

	var req createTimeLogRequest
	if !decodeJSON(w, r, &req) || !required(w, req.Hours) {
		return
	}
	if *req.Hours <= 0 || *req.Hours > 240 {
		badRequest(w, "Hours must be between 0.01 and 240")
		return
	}

	desc := ""
	if req.Description != nil {
		desc = *req.Description
	}
	loggedDate := ""
	if req.LoggedDate != nil {
		loggedDate = *req.LoggedDate
	}

	log, err := projects.LogTaskTime(r.Context(), s.Pool, &session.User.ID, wsID, pID, tID, projects.CreateTimeLogParams{
		Hours:       *req.Hours,
		LoggedDate:  loggedDate,
		Description: desc,
	})
	if err != nil {
		storeError(w, err, "failed to log task time", "")
		return
	}

	totalHours, _ := projects.GetTaskTotalSpentHours(r.Context(), s.Pool, tID)
	writeJSON(w, http.StatusCreated, map[string]any{
		"time_log":    log,
		"spent_hours": totalHours,
	})
}

func (s *Server) deleteTaskTimeLog(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "project_id", "task_id", "log_id")
	if !ok {
		return
	}
	wsID, pID, tID, logID := ids[0], ids[1], ids[2], ids[3]

	if err := projects.DeleteTaskTimeLog(r.Context(), s.Pool, &session.User.ID, wsID, pID, tID, logID); err != nil {
		storeError(w, err, "failed to delete task time log", "")
		return
	}

	totalHours, _ := projects.GetTaskTotalSpentHours(r.Context(), s.Pool, tID)
	writeJSON(w, http.StatusOK, map[string]any{
		"status":      "deleted",
		"spent_hours": totalHours,
	})
}
