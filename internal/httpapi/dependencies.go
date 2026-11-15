package httpapi

import (
	"errors"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/maruf-pfc/trestle/internal/projects"
)

type dependencyTaskItem struct {
	ID              string  `json:"id"`
	WorkspaceID     string  `json:"workspace_id"`
	ProjectID       string  `json:"project_id"`
	ProjectName     string  `json:"project_name"`
	ProjectKey      string  `json:"project_key"`
	TaskNumber      int64   `json:"task_number"`
	HumanKey        string  `json:"human_key"`
	Title           string  `json:"title"`
	Status          string  `json:"status"`
	StatusID        *string `json:"status_id"`
	StatusName      *string `json:"status_name"`
	StatusColor     *string `json:"status_color"`
	ProjectArchived bool    `json:"project_archived"`
}

type dependencyItem struct {
	ID        string             `json:"id"`
	Task      dependencyTaskItem `json:"task"`
	CreatedAt string             `json:"created_at"`
}

func toDependencyItem(l projects.DependencyLink) dependencyItem {
	return dependencyItem{
		ID: l.ID.String(),
		Task: dependencyTaskItem{
			ID: l.Task.ID.String(), WorkspaceID: l.Task.WorkspaceID.String(),
			ProjectID: l.Task.ProjectID.String(), ProjectName: l.Task.ProjectName,
			ProjectKey: l.Task.ProjectKey, TaskNumber: l.Task.TaskNumber,
			HumanKey: l.Task.HumanKey, Title: l.Task.Title, Status: l.Task.Status,
			StatusID: idPtr(l.Task.StatusID), StatusName: l.Task.StatusName,
			StatusColor: l.Task.StatusColor, ProjectArchived: l.Task.ProjectArchived,
		},
		CreatedAt: ts(l.CreatedAt),
	}
}

func (s *Server) listTaskDependencies(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "project_id", "task_id")
	if !ok {
		return
	}
	deps, err := projects.ListTaskDependencies(r.Context(), s.Pool, ids[0], ids[1], ids[2])
	if err != nil {
		storeError(w, err, "failed to list task dependencies", "")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"blocked_by": mapSlice(deps.BlockedBy, toDependencyItem),
		"blocks":     mapSlice(deps.Blocks, toDependencyItem),
	})
}

type addDependencyRequest struct {
	RelatedTaskID *string `json:"related_task_id"`
	Direction     *string `json:"direction"`
}

func (s *Server) addTaskDependency(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "project_id", "task_id")
	if !ok {
		return
	}
	var req addDependencyRequest
	if !decodeJSON(w, r, &req) || !required(w, req.RelatedTaskID, req.Direction) {
		return
	}
	relatedID, err := uuid.Parse(strings.TrimSpace(*req.RelatedTaskID))
	if err != nil {
		badRequest(w, "Invalid related task id")
		return
	}
	direction := strings.TrimSpace(*req.Direction)
	if direction != projects.DependencyBlockedBy && direction != projects.DependencyBlocks {
		badRequest(w, "Direction must be blocked_by or blocks")
		return
	}

	link, err := projects.AddTaskDependency(r.Context(), s.Pool, &session.User.ID,
		ids[0], ids[1], ids[2], relatedID, direction)
	switch {
	case errors.Is(err, projects.ErrNotFound):
		notFound(w)
		return
	case errors.Is(err, projects.ErrDependencySelf):
		badRequest(w, "A task cannot depend on itself")
		return
	case errors.Is(err, projects.ErrDependencyExists):
		conflict(w, "These tasks are already linked in that direction.")
		return
	case errors.Is(err, projects.ErrDependencyCycle):
		conflict(w, "That dependency would create a cycle. Remove the reverse link first.")
		return
	case err != nil:
		internal(w, "failed to add task dependency", err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"dependency": toDependencyItem(link)})
}

func (s *Server) removeTaskDependency(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "project_id", "task_id", "dependency_id")
	if !ok {
		return
	}
	if err := projects.RemoveTaskDependency(r.Context(), s.Pool, &session.User.ID,
		ids[0], ids[1], ids[2], ids[3]); err != nil {
		storeError(w, err, "failed to remove task dependency", "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
