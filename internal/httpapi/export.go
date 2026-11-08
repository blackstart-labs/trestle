package httpapi

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

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

func (s *Server) exportProject(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "project_id")
	if !ok {
		return
	}
	wsID, pID := ids[0], ids[1]

	proj, err := projects.GetProject(r.Context(), s.Pool, wsID, pID)
	if err != nil {
		internal(w, "failed to get project", err)
		return
	}

	statuses, err := projects.ListProjectStatuses(r.Context(), s.Pool, wsID, pID)
	if err != nil {
		internal(w, "failed to list statuses", err)
		return
	}
	stItems := make([]statusItem, len(statuses))
	for i, st := range statuses {
		stItems[i] = toStatus(st)
	}

	milestones, err := projects.ListMilestones(r.Context(), s.Pool, wsID, pID)
	if err != nil {
		internal(w, "failed to list milestones", err)
		return
	}
	msItems := make([]milestoneItem, len(milestones))
	for i, ms := range milestones {
		msItems[i] = toMilestone(ms)
	}

	paginated, err := projects.ListTasks(r.Context(), s.Pool, wsID, pID, projects.ListTasksParams{Limit: 10000})
	if err != nil {
		internal(w, "failed to list tasks", err)
		return
	}

	taskItems := make([]taskExportItem, len(paginated.Tasks))
	for i, t := range paginated.Tasks {
		tItem := toTask(t)
		logs, _ := projects.ListTaskTimeLogs(r.Context(), s.Pool, wsID, pID, t.ID)
		if logs == nil {
			logs = []projects.TimeLog{}
		}
		taskItems[i] = taskExportItem{
			taskItem: tItem,
			TimeLogs: logs,
		}
	}

	export := projectExportData{
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Project:    toProject(proj),
		Statuses:   stItems,
		Milestones: msItems,
		Tasks:      taskItems,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-export.json"`, proj.Key))
	_ = json.NewEncoder(w).Encode(export)
}

func (s *Server) exportWorkspace(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, ok := pathIDs(w, r, "workspace_id")
	if !ok {
		return
	}
	wsID := ids[0]
	role, ok := s.membership(w, r, wsID, session.User.ID)
	if !ok {
		return
	}
	if !isManager(role) {
		forbidden(w)
		return
	}

	workspaces, err := identity.ListUserWorkspaces(r.Context(), s.Pool, session.User.ID)
	if err != nil {
		internal(w, "failed to get workspace", err)
		return
	}
	var currentWS *workspaceItem
	for _, ws := range workspaces {
		if ws.ID == wsID {
			item := toWorkspaceItem(ws)
			currentWS = &item
			break
		}
	}
	if currentWS == nil {
		notFound(w)
		return
	}

	members, err := identity.ListWorkspaceMembers(r.Context(), s.Pool, wsID)
	if err != nil {
		internal(w, "failed to list members", err)
		return
	}
	mItems := make([]memberItem, len(members))
	for i, m := range members {
		mItems[i] = toMember(m)
	}

	projList, err := projects.ListProjects(r.Context(), s.Pool, wsID, false)
	if err != nil {
		internal(w, "failed to list projects", err)
		return
	}

	pExports := make([]projectExportData, 0, len(projList))
	for _, proj := range projList {
		statuses, _ := projects.ListProjectStatuses(r.Context(), s.Pool, wsID, proj.ID)
		stItems := make([]statusItem, len(statuses))
		for i, st := range statuses {
			stItems[i] = toStatus(st)
		}

		milestones, _ := projects.ListMilestones(r.Context(), s.Pool, wsID, proj.ID)
		msItems := make([]milestoneItem, len(milestones))
		for i, ms := range milestones {
			msItems[i] = toMilestone(ms)
		}

		paginated, _ := projects.ListTasks(r.Context(), s.Pool, wsID, proj.ID, projects.ListTasksParams{Limit: 10000})
		taskItems := make([]taskExportItem, len(paginated.Tasks))
		for i, t := range paginated.Tasks {
			tItem := toTask(t)
			logs, _ := projects.ListTaskTimeLogs(r.Context(), s.Pool, wsID, proj.ID, t.ID)
			if logs == nil {
				logs = []projects.TimeLog{}
			}
			taskItems[i] = taskExportItem{
				taskItem: tItem,
				TimeLogs: logs,
			}
		}

		pExports = append(pExports, projectExportData{
			ExportedAt: time.Now().UTC().Format(time.RFC3339),
			Project:    toProject(proj),
			Statuses:   stItems,
			Milestones: msItems,
			Tasks:      taskItems,
		})
	}

	export := workspaceExportData{
		ExportedAt: time.Now().UTC().Format(time.RFC3339),
		Workspace:  *currentWS,
		Members:    mItems,
		Projects:   pExports,
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf(`attachment; filename="%s-export.json"`, currentWS.Slug))
	_ = json.NewEncoder(w).Encode(export)
}
