package httpapi

import (
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/maruf-pfc/trestle/internal/projects"
)

type milestoneItem struct {
	ID             string  `json:"id"`
	WorkspaceID    string  `json:"workspace_id"`
	ProjectID      string  `json:"project_id"`
	Title          string  `json:"title"`
	Description    string  `json:"description"`
	StartDate      *string `json:"start_date"`
	DueDate        *string `json:"due_date"`
	EndDate        *string `json:"end_date"`
	Color          string  `json:"color"`
	Status         string  `json:"status"`
	TotalTasks     int64   `json:"total_tasks"`
	CompletedTasks int64   `json:"completed_tasks"`
	OpenTasks      int64   `json:"open_tasks"`
	CreatedAt      string  `json:"created_at"`
	UpdatedAt      string  `json:"updated_at"`
}

func toMilestone(m projects.Milestone) milestoneItem {
	item := milestoneItem{
		ID:             m.ID.String(),
		WorkspaceID:    m.WorkspaceID.String(),
		ProjectID:      m.ProjectID.String(),
		Title:          m.Title,
		Description:    m.Description,
		Color:          m.Color,
		Status:         m.Status,
		TotalTasks:     m.TotalTasks,
		CompletedTasks: m.CompletedTasks,
		OpenTasks:      m.OpenTasks,
		CreatedAt:      ts(m.CreatedAt),
		UpdatedAt:      ts(m.UpdatedAt),
	}
	if m.StartDate != nil {
		d := m.StartDate.Format(time.DateOnly)
		item.StartDate = &d
	}
	if m.DueDate != nil {
		d := m.DueDate.Format(time.DateOnly)
		item.DueDate = &d
		item.EndDate = &d
	}
	return item
}

func isValidHexColor(c string) bool {
	if len(c) != 7 || c[0] != '#' {
		return false
	}
	for i := 1; i < 7; i++ {
		b := c[i]
		if !((b >= '0' && b <= '9') || (b >= 'a' && b <= 'f') || (b >= 'A' && b <= 'F')) {
			return false
		}
	}
	return true
}

func (s *Server) listMilestones(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "project_id")
	if !ok {
		return
	}
	list, err := projects.ListMilestones(r.Context(), s.Pool, ids[0], ids[1])
	if err != nil {
		storeError(w, err, "failed to list milestones", "")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"milestones": mapSlice(list, toMilestone)})
}

func (s *Server) getMilestone(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "project_id", "milestone_id")
	if !ok {
		return
	}
	m, err := projects.GetMilestone(r.Context(), s.Pool, ids[0], ids[1], ids[2])
	if err != nil {
		storeError(w, err, "failed to get milestone", "")
		return
	}
	writeJSON(w, http.StatusOK, toMilestone(m))
}

func (s *Server) createMilestone(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	if _, ok := pathIDs(w, r, "workspace_id", "project_id"); !ok {
		return
	}
	var req struct {
		Title       *string `json:"title"`
		Description *string `json:"description"`
		StartDate   *string `json:"start_date"`
		DueDate     *string `json:"due_date"`
		EndDate     *string `json:"end_date"`
		Color       *string `json:"color"`
	}
	if !decodeJSON(w, r, &req) || !required(w, req.Title) {
		return
	}
	ids, _, ok := s.scope(w, r, session, false, "project_id")
	if !ok {
		return
	}
	title := strings.TrimSpace(*req.Title)
	if title == "" || len(title) > 255 {
		badRequest(w, "Milestone title must be between 1 and 255 characters")
		return
	}
	desc := ""
	if req.Description != nil {
		desc = *req.Description
	}
	if len(desc) > 10000 {
		badRequest(w, "Milestone description cannot exceed 10000 characters")
		return
	}
	color := "#6366f1"
	if req.Color != nil && strings.TrimSpace(*req.Color) != "" {
		c := strings.ToLower(strings.TrimSpace(*req.Color))
		if !isValidHexColor(c) {
			badRequest(w, "Milestone color must be a valid 6-character hex color (e.g. #6366f1)")
			return
		}
		color = c
	}
	var startDate *time.Time
	if req.StartDate != nil && strings.TrimSpace(*req.StartDate) != "" {
		if d, valid := parseDueDate(*req.StartDate); valid {
			startDate = d
		} else {
			badRequest(w, "Invalid start_date format, expected YYYY-MM-DD")
			return
		}
	}
	var due *time.Time
	targetDate := req.DueDate
	if targetDate == nil || strings.TrimSpace(*targetDate) == "" {
		targetDate = req.EndDate
	}
	if targetDate != nil && strings.TrimSpace(*targetDate) != "" {
		if d, valid := parseDueDate(*targetDate); valid {
			due = d
		} else {
			badRequest(w, "Invalid due_date format, expected YYYY-MM-DD")
			return
		}
	}

	m, err := projects.CreateMilestone(r.Context(), s.Pool, &session.User.ID, ids[0], ids[1], projects.CreateMilestoneParams{
		Title:       title,
		Description: desc,
		StartDate:   startDate,
		DueDate:     due,
		Color:       color,
	})
	if err != nil {
		storeError(w, err, "failed to create milestone", "")
		return
	}
	writeJSON(w, http.StatusCreated, toMilestone(m))
}

func (s *Server) updateMilestone(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	if _, ok := pathIDs(w, r, "workspace_id", "project_id", "milestone_id"); !ok {
		return
	}
	var req struct {
		Title       *string         `json:"title"`
		Description *string         `json:"description"`
		StartDate   json.RawMessage `json:"start_date"`
		DueDate     json.RawMessage `json:"due_date"`
		EndDate     json.RawMessage `json:"end_date"`
		Status      *string         `json:"status"`
		Color       *string         `json:"color"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	var startDateRaw, dueDateRaw, endDateRaw *string
	if len(req.StartDate) > 0 && json.Unmarshal(req.StartDate, &startDateRaw) != nil {
		unprocessable(w)
		return
	}
	if len(req.DueDate) > 0 && json.Unmarshal(req.DueDate, &dueDateRaw) != nil {
		unprocessable(w)
		return
	}
	if len(req.EndDate) > 0 && json.Unmarshal(req.EndDate, &endDateRaw) != nil {
		unprocessable(w)
		return
	}
	ids, _, ok := s.scope(w, r, session, false, "project_id", "milestone_id")
	if !ok {
		return
	}
	params := projects.UpdateMilestoneParams{}
	if req.Title != nil {
		t := strings.TrimSpace(*req.Title)
		if t == "" || len(t) > 255 {
			badRequest(w, "Milestone title must be between 1 and 255 characters")
			return
		}
		params.Title = &t
	}
	if req.Description != nil {
		if len(*req.Description) > 10000 {
			badRequest(w, "Milestone description cannot exceed 10000 characters")
			return
		}
		params.Description = req.Description
	}
	if req.Status != nil {
		st := strings.ToLower(strings.TrimSpace(*req.Status))
		if st != "open" && st != "closed" {
			badRequest(w, "Milestone status must be 'open' or 'closed'")
			return
		}
		params.Status = &st
	}
	if req.Color != nil {
		c := strings.ToLower(strings.TrimSpace(*req.Color))
		if !isValidHexColor(c) {
			badRequest(w, "Milestone color must be a valid 6-character hex color (e.g. #6366f1)")
			return
		}
		params.Color = &c
	}
	if len(req.StartDate) > 0 {
		params.SetStartDate = true
		if startDateRaw != nil && strings.TrimSpace(*startDateRaw) != "" {
			d, valid := parseDueDate(*startDateRaw)
			if !valid {
				badRequest(w, "Invalid start_date format, expected YYYY-MM-DD")
				return
			}
			params.StartDate = d
		}
	}
	targetDueDateRaw := dueDateRaw
	hasDueDate := len(req.DueDate) > 0
	if !hasDueDate && len(req.EndDate) > 0 {
		hasDueDate = true
		targetDueDateRaw = endDateRaw
	}
	if hasDueDate {
		params.SetDueDate = true
		if targetDueDateRaw != nil && strings.TrimSpace(*targetDueDateRaw) != "" {
			d, valid := parseDueDate(*targetDueDateRaw)
			if !valid {
				badRequest(w, "Invalid due_date format, expected YYYY-MM-DD")
				return
			}
			params.DueDate = d
		}
	}

	m, err := projects.UpdateMilestone(r.Context(), s.Pool, &session.User.ID, ids[0], ids[1], ids[2], params)
	if err != nil {
		storeError(w, err, "failed to update milestone", "")
		return
	}
	writeJSON(w, http.StatusOK, toMilestone(m))
}

func (s *Server) deleteMilestone(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "project_id", "milestone_id")
	if !ok {
		return
	}
	if err := projects.DeleteMilestone(r.Context(), s.Pool, &session.User.ID, ids[0], ids[1], ids[2]); err != nil {
		storeError(w, err, "failed to delete milestone", "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
