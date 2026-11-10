package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/maruf-pfc/trestle/internal/identity"
	"github.com/maruf-pfc/trestle/internal/projects"
)

type projectExportData struct {
	ExportedAt string           `json:"exported_at"`
	Project    projectItem      `json:"project"`
	Statuses   []statusItem     `json:"statuses"`
	Milestones []milestoneItem  `json:"milestones"`
	Tasks      []taskExportItem `json:"tasks"`
}

type taskExportItem struct {
	taskItem
	TimeLogs []projects.TimeLog `json:"time_logs"`
}

type workspaceExportData struct {
	ExportedAt string              `json:"exported_at"`
	Workspace  workspaceItem       `json:"workspace"`
	Members    []memberItem        `json:"members"`
	Projects   []projectExportData `json:"projects"`
}

func readProjectExport(ctx context.Context, db projects.Querier, ws, id uuid.UUID) (projectExportData, error) {
	// shortcut: snapshots fit small-team memory; stream exports when datasets outgrow it.
	out := projectExportData{ExportedAt: time.Now().UTC().Format(time.RFC3339), Tasks: []taskExportItem{}}
	p, err := projects.GetProject(ctx, db, ws, id)
	if err != nil {
		return out, err
	}
	out.Project = toProject(p)
	statuses, err := projects.ListProjectStatuses(ctx, db, ws, id)
	if err != nil {
		return out, err
	}
	out.Statuses = mapSlice(statuses, toStatus)
	milestones, err := projects.ListMilestones(ctx, db, ws, id)
	if err != nil {
		return out, err
	}
	out.Milestones = mapSlice(milestones, toMilestone)
	for offset := int64(0); ; {
		page, err := projects.ListTasks(ctx, db, ws, id, projects.ListTasksParams{Limit: 100, Offset: offset})
		if err != nil {
			return out, err
		}
		for _, task := range page.Tasks {
			logs, err := projects.ListTaskTimeLogs(ctx, db, ws, id, task.ID)
			if err != nil {
				return out, err
			}
			if logs == nil {
				logs = []projects.TimeLog{}
			}
			out.Tasks = append(out.Tasks, taskExportItem{taskItem: toTask(task), TimeLogs: logs})
		}
		offset += int64(len(page.Tasks))
		if offset >= page.Total || len(page.Tasks) == 0 {
			return out, nil
		}
	}
}

func (s *Server) exportProject(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "project_id")
	if !ok {
		return
	}
	// A stable snapshot prevents concurrent edits from skipping tasks between pages.
	tx, err := s.Pool.BeginTx(r.Context(), pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		internal(w, "failed to start export", err)
		return
	}
	defer tx.Rollback(r.Context())
	out, err := readProjectExport(r.Context(), tx, ids[0], ids[1])
	if err != nil {
		internal(w, "failed to export project", err)
		return
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-export.json"`, out.Project.Key))
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) exportWorkspace(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, ok := pathIDs(w, r, "workspace_id")
	if !ok {
		return
	}
	role, ok := s.membership(w, r, ids[0], session.User.ID)
	if !ok {
		return
	}
	if !isManager(role) {
		forbidden(w)
		return
	}
	tx, err := s.Pool.BeginTx(r.Context(), pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		internal(w, "failed to start export", err)
		return
	}
	defer tx.Rollback(r.Context())
	workspaces, err := identity.ListUserWorkspaces(r.Context(), tx, session.User.ID)
	if err != nil {
		internal(w, "failed to get workspace", err)
		return
	}
	out := workspaceExportData{ExportedAt: time.Now().UTC().Format(time.RFC3339), Projects: []projectExportData{}}
	for _, ws := range workspaces {
		if ws.ID == ids[0] {
			out.Workspace = toWorkspaceItem(ws)
		}
	}
	if out.Workspace.ID == "" {
		notFound(w)
		return
	}
	members, err := identity.ListWorkspaceMembers(r.Context(), tx, ids[0])
	if err != nil {
		internal(w, "failed to list members", err)
		return
	}
	out.Members = mapSlice(members, toMember)
	list, err := projects.ListProjects(r.Context(), tx, ids[0], true)
	if err != nil {
		internal(w, "failed to list projects", err)
		return
	}
	for _, p := range list {
		data, err := readProjectExport(r.Context(), tx, ids[0], p.ID)
		if err != nil {
			internal(w, "failed to export project", err)
			return
		}
		out.Projects = append(out.Projects, data)
	}
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-export.json"`, out.Workspace.Slug))
	writeJSON(w, http.StatusOK, out)
}
