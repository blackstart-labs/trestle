package httpapi

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/maruf-pfc/trestle/internal/projects"
)

// ---------------------------------------------------------------------------
// Response shapes
// ---------------------------------------------------------------------------

func ts(t time.Time) string { return t.UTC().Format(time.RFC3339Nano) }

func idPtr(id *uuid.UUID) *string {
	if id == nil {
		return nil
	}
	s := id.String()
	return &s
}

type projectItem struct {
	ID          string  `json:"id"`
	WorkspaceID string  `json:"workspace_id"`
	Name        string  `json:"name"`
	Slug        string  `json:"slug"`
	Key         string  `json:"key"`
	Description string  `json:"description"`
	TaskCounter int64   `json:"task_counter"`
	Archived    bool    `json:"archived"`
	ImageURL    *string `json:"image_url"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

func toProject(p projects.Project) projectItem {
	var imageURL *string
	if p.HasAvatar {
		t := ""
		if p.AvatarUpdatedAt != nil {
			t = fmt.Sprintf("?t=%d", p.AvatarUpdatedAt.Unix())
		}
		url := fmt.Sprintf("/api/v1/workspaces/%s/projects/%s/image%s", p.WorkspaceID, p.ID, t)
		imageURL = &url
	}
	return projectItem{
		ID:          p.ID.String(),
		WorkspaceID: p.WorkspaceID.String(),
		Name:        p.Name,
		Slug:        p.Slug,
		Key:         p.Key,
		Description: p.Description,
		TaskCounter: p.TaskCounter,
		Archived:    p.Archived,
		ImageURL:    imageURL,
		CreatedAt:   ts(p.CreatedAt),
		UpdatedAt:   ts(p.UpdatedAt),
	}
}

func parseImageData(raw string) ([]byte, string, error) {
	mimeType := "image/png"
	b64 := raw
	if idx := strings.Index(raw, ","); idx != -1 {
		header := raw[:idx]
		b64 = raw[idx+1:]
		if strings.HasPrefix(header, "data:") {
			mimeType = strings.TrimPrefix(header, "data:")
			if semi := strings.Index(mimeType, ";"); semi != -1 {
				mimeType = mimeType[:semi]
			}
		}
	}
	data, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		return nil, "", errors.New("invalid base64 image data")
	}
	if len(data) > 5*1024*1024 {
		return nil, "", errors.New("image size exceeds 5MB")
	}
	validMimes := map[string]bool{
		"image/png":     true,
		"image/jpeg":    true,
		"image/jpg":     true,
		"image/webp":    true,
		"image/gif":     true,
		"image/svg+xml": true,
	}
	detected := http.DetectContentType(data)
	if detected == "image/png" || detected == "image/jpeg" || detected == "image/gif" || detected == "image/webp" {
		mimeType = detected
	} else if strings.HasPrefix(strings.TrimSpace(string(data)), "<svg") || strings.HasPrefix(strings.TrimSpace(string(data)), "<?xml") {
		mimeType = "image/svg+xml"
	}
	if !validMimes[mimeType] {
		return nil, "", errors.New("unsupported image format. Allowed: PNG, JPEG, WebP, GIF, SVG")
	}
	return data, mimeType, nil
}

type labelItem struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	ProjectID   string `json:"project_id"`
	Name        string `json:"name"`
	Color       string `json:"color"`
	CreatedAt   string `json:"created_at"`
}

func toLabel(l projects.Label) labelItem {
	return labelItem{l.ID.String(), l.WorkspaceID.String(), l.ProjectID.String(), l.Name, l.Color, ts(l.CreatedAt)}
}

type assigneeItem struct {
	TaskID      string  `json:"task_id"`
	UserID      string  `json:"user_id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	AvatarURL   *string `json:"avatar_url"`
	AssignedBy  *string `json:"assigned_by"`
	CreatedAt   string  `json:"created_at"`
}

func toAssignee(a projects.TaskAssignee) assigneeItem {
	return assigneeItem{a.TaskID.String(), a.UserID.String(), a.Username, a.DisplayName, a.AvatarURL, idPtr(a.AssignedBy), ts(a.CreatedAt)}
}

type commentItem struct {
	ID             string `json:"id"`
	WorkspaceID    string `json:"workspace_id"`
	TaskID         string `json:"task_id"`
	UserID         string `json:"user_id"`
	AuthorName     string `json:"author_name"`
	AuthorUsername string `json:"author_username"`
	Content        string `json:"content"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

func toComment(c projects.TaskComment) commentItem {
	return commentItem{c.ID.String(), c.WorkspaceID.String(), c.TaskID.String(), c.UserID.String(),
		c.AuthorName, c.AuthorUsername, c.Content, ts(c.CreatedAt), ts(c.UpdatedAt)}
}

type activityItem struct {
	ID          string          `json:"id"`
	WorkspaceID string          `json:"workspace_id"`
	TaskID      string          `json:"task_id"`
	ActorID     *string         `json:"actor_id"`
	ActorName   *string         `json:"actor_name"`
	Action      string          `json:"action"`
	Details     json.RawMessage `json:"details"`
	CreatedAt   string          `json:"created_at"`
}

type taskItem struct {
	ID             string         `json:"id"`
	WorkspaceID    string         `json:"workspace_id"`
	ProjectID      string         `json:"project_id"`
	ProjectKey     string         `json:"project_key"`
	TaskNumber     int64          `json:"task_number"`
	HumanKey       string         `json:"human_key"`
	Title          string         `json:"title"`
	Description    string         `json:"description"`
	Status         string         `json:"status"`
	Priority       string         `json:"priority"`
	DueDate        *string        `json:"due_date"`
	CreatedBy      *string        `json:"created_by"`
	ParentID       *string        `json:"parent_id"`
	StatusID       *string        `json:"status_id"`
	StatusName     *string        `json:"status_name"`
	StatusColor    *string        `json:"status_color"`
	MilestoneID    *string        `json:"milestone_id"`
	MilestoneTitle *string        `json:"milestone_title"`
	MilestoneColor *string        `json:"milestone_color"`
	Labels         []labelItem    `json:"labels"`
	Assignees      []assigneeItem `json:"assignees"`
	CreatedAt      string         `json:"created_at"`
	UpdatedAt      string         `json:"updated_at"`
}

func toTask(t projects.Task) taskItem {
	item := taskItem{
		ID: t.ID.String(), WorkspaceID: t.WorkspaceID.String(), ProjectID: t.ProjectID.String(),
		ProjectKey: t.ProjectKey, TaskNumber: t.TaskNumber, HumanKey: t.HumanKey, Title: t.Title,
		Description: t.Description, Status: t.Status, Priority: t.Priority,
		CreatedBy: idPtr(t.CreatedBy), ParentID: idPtr(t.ParentID), StatusID: idPtr(t.StatusID),
		StatusName: t.StatusName, StatusColor: t.StatusColor,
		MilestoneID: idPtr(t.MilestoneID), MilestoneTitle: t.MilestoneTitle, MilestoneColor: t.MilestoneColor,
		Labels: mapSlice(t.Labels, toLabel), Assignees: mapSlice(t.Assignees, toAssignee),
		CreatedAt: ts(t.CreatedAt), UpdatedAt: ts(t.UpdatedAt),
	}
	if t.DueDate != nil {
		d := t.DueDate.Format(time.DateOnly)
		item.DueDate = &d
	}
	return item
}

type statusItem struct {
	ID          string `json:"id"`
	WorkspaceID string `json:"workspace_id"`
	ProjectID   string `json:"project_id"`
	Name        string `json:"name"`
	Category    string `json:"category"`
	Color       string `json:"color"`
	Position    int32  `json:"position"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

func toStatus(s projects.TaskStatus) statusItem {
	return statusItem{s.ID.String(), s.WorkspaceID.String(), s.ProjectID.String(), s.Name, s.Category, s.Color,
		s.Position, ts(s.CreatedAt), ts(s.UpdatedAt)}
}

// mapSlice always returns a non-nil slice so empty lists encode as [].
func mapSlice[T, U any](in []T, f func(T) U) []U {
	out := make([]U, len(in))
	for i, v := range in {
		out[i] = f(v)
	}
	return out
}

// ---------------------------------------------------------------------------
// Validation
// ---------------------------------------------------------------------------

func validProjectSlug(s string) bool {
	if len(s) < 2 || len(s) > 48 || s[0] == '-' || s[len(s)-1] == '-' {
		return false
	}
	for _, c := range s {
		if !(c >= 'a' && c <= 'z' || c >= '0' && c <= '9' || c == '-') {
			return false
		}
	}
	return true
}

func validProjectKey(k string) bool {
	if len(k) < 2 || len(k) > 8 {
		return false
	}
	for _, c := range k {
		if !(c >= 'A' && c <= 'Z' || c >= '0' && c <= '9') {
			return false
		}
	}
	return true
}

func validStatus(s string) bool {
	switch s {
	case "backlog", "unstarted", "started", "completed", "cancelled":
		return true
	}
	return false
}

func validPriority(p string) bool {
	switch p {
	case "none", "low", "medium", "high", "urgent":
		return true
	}
	return false
}

func validHexColor(c string) bool {
	c = strings.TrimSpace(c)
	if len(c) != 7 || c[0] != '#' {
		return false
	}
	for _, ch := range c[1:] {
		if !(ch >= '0' && ch <= '9' || ch >= 'a' && ch <= 'f' || ch >= 'A' && ch <= 'F') {
			return false
		}
	}
	return true
}

func parseDueDate(s string) (*time.Time, bool) {
	d, err := time.Parse(time.DateOnly, strings.TrimSpace(s))
	return &d, err == nil
}

// storeError maps store errors to responses; conflictMsg is used for ErrConflict.
func storeError(w http.ResponseWriter, err error, logMsg, conflictMsg string) {
	switch {
	case errors.Is(err, projects.ErrNotFound):
		notFound(w)
	case errors.Is(err, projects.ErrForbidden):
		forbidden(w)
	case errors.Is(err, projects.ErrConflict) && conflictMsg != "":
		conflict(w, conflictMsg)
	default:
		internal(w, logMsg, err)
	}
}

// scope parses the path IDs and checks membership; managerOnly also requires owner or admin.
func (s *Server) scope(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession, managerOnly bool, names ...string) ([]uuid.UUID, string, bool) {
	ids, ok := pathIDs(w, r, append([]string{"workspace_id"}, names...)...)
	if !ok {
		return nil, "", false
	}
	role, ok := s.membership(w, r, ids[0], session.User.ID)
	if !ok {
		return nil, "", false
	}
	if managerOnly && !isManager(role) {
		forbidden(w)
		return nil, "", false
	}
	return ids, role, true
}

// ---------------------------------------------------------------------------
// Projects
// ---------------------------------------------------------------------------

func (s *Server) listProjects(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, ok := pathIDs(w, r, "workspace_id")
	if !ok {
		return
	}
	includeArchived := false
	if q := r.URL.Query(); q.Has("include_archived") {
		switch q.Get("include_archived") {
		case "true":
			includeArchived = true
		case "false":
		default:
			badRequest(w, "Invalid query parameter: include_archived")
			return
		}
	}
	if _, ok := s.membership(w, r, ids[0], session.User.ID); !ok {
		return
	}
	list, err := projects.ListProjects(r.Context(), s.Pool, ids[0], includeArchived)
	if err != nil {
		internal(w, "failed to list projects", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"projects": mapSlice(list, toProject)})
}

func (s *Server) createProject(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	var req struct {
		Name        *string `json:"name"`
		Slug        *string `json:"slug"`
		Key         *string `json:"key"`
		Description *string `json:"description"`
		Image       *string `json:"image"`
	}
	if _, ok := pathIDs(w, r, "workspace_id"); !ok {
		return
	}
	if !decodeJSON(w, r, &req) || !required(w, req.Name, req.Slug, req.Key) {
		return
	}
	ids, _, ok := s.scope(w, r, session, true)
	if !ok {
		return
	}
	name := strings.TrimSpace(*req.Name)
	if name == "" || len(name) > 120 {
		badRequest(w, "Project name must be between 1 and 120 characters")
		return
	}
	slug := strings.ToLower(strings.TrimSpace(*req.Slug))
	if !validProjectSlug(slug) {
		badRequest(w, "Project slug must be 2-48 lowercase alphanumeric characters or hyphens")
		return
	}
	key := strings.ToUpper(strings.TrimSpace(*req.Key))
	if !validProjectKey(key) {
		badRequest(w, "Project key must be 2-8 uppercase alphanumeric characters")
		return
	}
	description := ""
	if req.Description != nil {
		description = *req.Description
	}
	if len(description) > 2000 {
		badRequest(w, "Project description cannot exceed 2000 characters")
		return
	}

	var avatarData []byte
	var avatarMIME string
	if req.Image != nil && strings.TrimSpace(*req.Image) != "" {
		data, mime, err := parseImageData(*req.Image)
		if err != nil {
			badRequest(w, err.Error())
			return
		}
		avatarData = data
		avatarMIME = mime
	}

	p, err := projects.CreateProject(r.Context(), s.Pool, &session.User.ID, ids[0], projects.CreateProjectParams{
		Name: name, Slug: slug, Key: key, Description: description,
		AvatarData: avatarData, AvatarMIME: avatarMIME,
	})
	if err != nil {
		storeError(w, err, "failed to create project", "A project with this slug or key already exists in the workspace")
		return
	}
	writeJSON(w, http.StatusCreated, toProject(p))
}

func (s *Server) getProject(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "project_id")
	if !ok {
		return
	}
	p, err := projects.GetProject(r.Context(), s.Pool, ids[0], ids[1])
	if err != nil {
		storeError(w, err, "failed to get project", "")
		return
	}
	writeJSON(w, http.StatusOK, toProject(p))
}

func (s *Server) updateProject(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, true, "project_id")
	if !ok {
		return
	}
	var req struct {
		Name        *string `json:"name"`
		Slug        *string `json:"slug"`
		Description *string `json:"description"`
		Key         *string `json:"key"`
		Image       *string `json:"image"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if req.Key != nil {
		badRequest(w, "Project key cannot be changed because existing task keys use it")
		return
	}
	if req.Name == nil && req.Slug == nil && req.Description == nil && req.Image == nil {
		badRequest(w, "Provide a project name, slug, description, or image to update")
		return
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" || len(name) > 120 {
			badRequest(w, "Project name must be between 1 and 120 characters")
			return
		}
		req.Name = &name
	}
	if req.Slug != nil {
		slug := strings.ToLower(strings.TrimSpace(*req.Slug))
		if !validProjectSlug(slug) {
			badRequest(w, "Project slug must be 2-48 lowercase alphanumeric characters or hyphens")
			return
		}
		req.Slug = &slug
	}
	if req.Description != nil && len(*req.Description) > 2000 {
		badRequest(w, "Project description cannot exceed 2000 characters")
		return
	}

	var avatarData *[]byte
	var avatarMIME *string
	removeAvatar := false
	if req.Image != nil {
		if strings.TrimSpace(*req.Image) == "" {
			removeAvatar = true
		} else {
			data, mime, err := parseImageData(*req.Image)
			if err != nil {
				badRequest(w, err.Error())
				return
			}
			avatarData = &data
			avatarMIME = &mime
		}
	}

	p, err := projects.UpdateProject(r.Context(), s.Pool, &session.User.ID, ids[0], ids[1], projects.UpdateProjectParams{
		Name: req.Name, Slug: req.Slug, Description: req.Description,
		AvatarData: avatarData, AvatarMIME: avatarMIME, RemoveAvatar: removeAvatar,
	})
	if err != nil {
		storeError(w, err, "failed to update project", "A project with this slug already exists in the workspace")
		return
	}
	writeJSON(w, http.StatusOK, toProject(p))
}

func (s *Server) getProjectImage(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "project_id")
	if !ok {
		return
	}
	data, mime, updatedAt, err := projects.GetProjectAvatar(r.Context(), s.Pool, ids[0], ids[1])
	if err != nil {
		notFound(w)
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "public, max-age=86400")
	w.Header().Set("ETag", fmt.Sprintf("\"%s-%d\"", ids[1].String(), updatedAt.UnixNano()))
	w.WriteHeader(http.StatusOK)
	w.Write(data)
}

func (s *Server) setProjectImage(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, true, "project_id")
	if !ok {
		return
	}
	ct := r.Header.Get("Content-Type")
	var data []byte
	var mime string
	var err error
	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseMultipartForm(5 << 20); err != nil {
			badRequest(w, "Failed to parse upload form")
			return
		}
		file, header, ferr := r.FormFile("file")
		if ferr != nil {
			file, header, ferr = r.FormFile("image")
		}
		if ferr != nil {
			badRequest(w, "No image file provided in upload")
			return
		}
		defer file.Close()
		data, err = io.ReadAll(file)
		if err != nil {
			badRequest(w, "Failed to read uploaded image")
			return
		}
		mime = header.Header.Get("Content-Type")
	} else if strings.HasPrefix(ct, "application/json") {
		var req struct {
			Image string `json:"image"`
		}
		if !decodeJSON(w, r, &req) {
			return
		}
		data, mime, err = parseImageData(req.Image)
		if err != nil {
			badRequest(w, err.Error())
			return
		}
	} else {
		data, err = io.ReadAll(http.MaxBytesReader(w, r.Body, 5<<20))
		if err != nil {
			badRequest(w, "Image size exceeds 5MB")
			return
		}
		mime = ct
	}

	if len(data) == 0 {
		badRequest(w, "Image cannot be empty")
		return
	}

	validMimes := map[string]bool{
		"image/png":     true,
		"image/jpeg":    true,
		"image/jpg":     true,
		"image/webp":    true,
		"image/gif":     true,
		"image/svg+xml": true,
	}
	detected := http.DetectContentType(data)
	if detected == "image/png" || detected == "image/jpeg" || detected == "image/gif" || detected == "image/webp" {
		mime = detected
	} else if strings.HasPrefix(strings.TrimSpace(string(data)), "<svg") || strings.HasPrefix(strings.TrimSpace(string(data)), "<?xml") {
		mime = "image/svg+xml"
	}
	if !validMimes[mime] {
		badRequest(w, "Unsupported image format. Allowed: PNG, JPEG, WebP, GIF, SVG")
		return
	}

	p, err := projects.UpdateProject(r.Context(), s.Pool, &session.User.ID, ids[0], ids[1], projects.UpdateProjectParams{
		AvatarData: &data, AvatarMIME: &mime,
	})
	if err != nil {
		storeError(w, err, "failed to set project image", "")
		return
	}
	writeJSON(w, http.StatusOK, toProject(p))
}

func (s *Server) deleteProjectImage(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, true, "project_id")
	if !ok {
		return
	}
	p, err := projects.UpdateProject(r.Context(), s.Pool, &session.User.ID, ids[0], ids[1], projects.UpdateProjectParams{
		RemoveAvatar: true,
	})
	if err != nil {
		storeError(w, err, "failed to delete project image", "")
		return
	}
	writeJSON(w, http.StatusOK, toProject(p))
}

func (s *Server) archiveProject(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, true, "project_id")
	if !ok {
		return
	}
	p, err := projects.ArchiveProject(r.Context(), s.Pool, &session.User.ID, ids[0], ids[1])
	if err != nil {
		storeError(w, err, "failed to archive project", "")
		return
	}
	writeJSON(w, http.StatusOK, toProject(p))
}

// ---------------------------------------------------------------------------
// Tasks
// ---------------------------------------------------------------------------

func (s *Server) listTasks(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	if _, ok := pathIDs(w, r, "workspace_id", "project_id"); !ok {
		return
	}
	q := r.URL.Query()
	params := projects.ListTasksParams{Limit: 20}
	var ok bool
	if params.StatusID, ok = queryUUID(w, r, "status_id"); !ok {
		return
	}
	if params.LabelID, ok = queryUUID(w, r, "label_id"); !ok {
		return
	}
	if params.AssigneeID, ok = queryUUID(w, r, "assignee_id"); !ok {
		return
	}
	if params.MilestoneID, ok = queryUUID(w, r, "milestone_id"); !ok {
		return
	}
	for name, dst := range map[string]*int64{"limit": &params.Limit, "offset": &params.Offset} {
		if q.Has(name) {
			n, err := strconv.ParseInt(q.Get(name), 10, 64)
			if err != nil {
				badRequest(w, "Invalid query parameter: "+name)
				return
			}
			*dst = n
		}
	}
	if q.Has("status") {
		status := q.Get("status")
		params.Status = &status
	}
	ids, _, ok := s.scope(w, r, session, false, "project_id")
	if !ok {
		return
	}
	if params.Status != nil && !validStatus(*params.Status) {
		badRequest(w, "Invalid status filter")
		return
	}
	page, err := projects.ListTasks(r.Context(), s.Pool, ids[0], ids[1], params)
	if err != nil {
		storeError(w, err, "failed to list tasks", "")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tasks": mapSlice(page.Tasks, toTask), "total": page.Total, "limit": page.Limit, "offset": page.Offset,
	})
}

type createTaskRequest struct {
	Title       *string    `json:"title"`
	Description *string    `json:"description"`
	Status      *string    `json:"status"`
	StatusID    *uuid.UUID `json:"status_id"`
	Priority    *string    `json:"priority"`
	DueDate     *string    `json:"due_date"`
	MilestoneID *uuid.UUID `json:"milestone_id"`
	ParentID    *uuid.UUID `json:"parent_id"`
}

// createTaskOrSubtask serves both endpoints; noun is "Task" or "Subtask" in error messages.
func (s *Server) createTaskOrSubtask(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession, subtask bool) {
	names := []string{"project_id"}
	noun := "Task"
	if subtask {
		names, noun = append(names, "task_id"), "Subtask"
	}
	if _, ok := pathIDs(w, r, append([]string{"workspace_id"}, names...)...); !ok {
		return
	}
	var req createTaskRequest
	if !decodeJSON(w, r, &req) || !required(w, req.Title) {
		return
	}
	ids, _, ok := s.scope(w, r, session, false, names...)
	if !ok {
		return
	}
	lower := strings.ToLower(noun)
	title := strings.TrimSpace(*req.Title)
	if title == "" || len(title) > 255 {
		badRequest(w, noun+" title must be between 1 and 255 characters")
		return
	}
	status := "unstarted"
	if req.Status != nil {
		status = *req.Status
	}
	if req.StatusID == nil && !validStatus(status) {
		badRequest(w, "Invalid "+lower+" status")
		return
	}
	priority := "none"
	if req.Priority != nil {
		priority = *req.Priority
	}
	if !validPriority(priority) {
		badRequest(w, "Invalid "+lower+" priority")
		return
	}
	description := ""
	if req.Description != nil {
		description = *req.Description
	}
	if len(description) > 10000 {
		badRequest(w, noun+" description cannot exceed 10000 characters")
		return
	}
	var due *time.Time
	if req.DueDate != nil && strings.TrimSpace(*req.DueDate) != "" {
		if due, ok = parseDueDate(*req.DueDate); !ok {
			badRequest(w, "Invalid due_date format, expected YYYY-MM-DD")
			return
		}
	}
	parent := req.ParentID
	if subtask {
		parent = &ids[2]
	}
	t, err := projects.CreateTask(r.Context(), s.Pool, &session.User.ID, ids[0], ids[1], projects.CreateTaskParams{
		Title: title, Description: description, Status: status, Priority: priority, DueDate: due,
		ParentID: parent, StatusID: req.StatusID, MilestoneID: req.MilestoneID,
	})
	if err != nil {
		storeError(w, err, "failed to create "+lower, "")
		return
	}
	s.notifyTask(r, ids[0], ids[1], "created", t)
	writeJSON(w, http.StatusCreated, toTask(t))
}

func (s *Server) createTask(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	s.createTaskOrSubtask(w, r, session, false)
}

func (s *Server) createSubtask(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	s.createTaskOrSubtask(w, r, session, true)
}

func (s *Server) getTask(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "project_id", "task_id")
	if !ok {
		return
	}
	t, err := projects.GetTask(r.Context(), s.Pool, ids[0], ids[1], ids[2])
	if err != nil {
		storeError(w, err, "failed to get task", "")
		return
	}
	writeJSON(w, http.StatusOK, toTask(t))
}

func (s *Server) updateTask(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	if _, ok := pathIDs(w, r, "workspace_id", "project_id", "task_id"); !ok {
		return
	}
	var req struct {
		Title       *string         `json:"title"`
		Description *string         `json:"description"`
		Status      *string         `json:"status"`
		StatusID    *uuid.UUID      `json:"status_id"`
		Priority    *string         `json:"priority"`
		DueDate     json.RawMessage `json:"due_date"`     // absent: unchanged; null or "": clear
		MilestoneID json.RawMessage `json:"milestone_id"` // absent: unchanged; null or "": clear
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	var dueDate *string
	if len(req.DueDate) > 0 && json.Unmarshal(req.DueDate, &dueDate) != nil {
		unprocessable(w)
		return
	}
	var rawMilestoneID *string
	if len(req.MilestoneID) > 0 && json.Unmarshal(req.MilestoneID, &rawMilestoneID) != nil {
		unprocessable(w)
		return
	}
	ids, _, ok := s.scope(w, r, session, false, "project_id", "task_id")
	if !ok {
		return
	}
	if req.Title != nil {
		t := strings.TrimSpace(*req.Title)
		if t == "" || len(t) > 255 {
			badRequest(w, "Task title must be between 1 and 255 characters")
			return
		}
		req.Title = &t
	}
	if req.Status != nil && req.StatusID == nil && !validStatus(*req.Status) {
		badRequest(w, "Invalid task status")
		return
	}
	if req.Priority != nil && !validPriority(*req.Priority) {
		badRequest(w, "Invalid task priority")
		return
	}
	if req.Description != nil && len(*req.Description) > 10000 {
		badRequest(w, "Task description cannot exceed 10000 characters")
		return
	}
	params := projects.UpdateTaskParams{Title: req.Title, Description: req.Description, Status: req.Status,
		StatusID: req.StatusID, Priority: req.Priority, SetDueDate: len(req.DueDate) > 0,
		SetMilestoneID: len(req.MilestoneID) > 0}
	if dueDate != nil && strings.TrimSpace(*dueDate) != "" {
		if params.DueDate, ok = parseDueDate(*dueDate); !ok {
			badRequest(w, "Invalid due_date format, expected YYYY-MM-DD")
			return
		}
	}
	if rawMilestoneID != nil && strings.TrimSpace(*rawMilestoneID) != "" {
		mid, err := uuid.Parse(strings.TrimSpace(*rawMilestoneID))
		if err != nil {
			badRequest(w, "Invalid milestone_id format")
			return
		}
		params.MilestoneID = &mid
	}
	t, err := projects.UpdateTask(r.Context(), s.Pool, &session.User.ID, ids[0], ids[1], ids[2], params)
	if err != nil {
		storeError(w, err, "failed to update task", "")
		return
	}
	s.notifyTask(r, ids[0], ids[1], "updated", t)
	writeJSON(w, http.StatusOK, toTask(t))
}

func (s *Server) deleteTask(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, true, "project_id", "task_id")
	if !ok {
		return
	}
	t, err := projects.GetTask(r.Context(), s.Pool, ids[0], ids[1], ids[2])
	if err != nil {
		storeError(w, err, "failed to get task for deletion", "")
		return
	}
	if err := projects.DeleteTask(r.Context(), s.Pool, &session.User.ID, ids[0], ids[1], ids[2]); err != nil {
		storeError(w, err, "failed to delete task", "")
		return
	}
	s.notifyTask(r, ids[0], ids[1], "deleted", t)
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Labels
// ---------------------------------------------------------------------------

func (s *Server) listLabels(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "project_id")
	if !ok {
		return
	}
	list, err := projects.ListLabels(r.Context(), s.Pool, ids[0], ids[1])
	if err != nil {
		storeError(w, err, "failed to list labels", "")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"labels": mapSlice(list, toLabel)})
}

func (s *Server) createLabel(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	if _, ok := pathIDs(w, r, "workspace_id", "project_id"); !ok {
		return
	}
	var req struct {
		Name  *string `json:"name"`
		Color *string `json:"color"`
	}
	if !decodeJSON(w, r, &req) || !required(w, req.Name) {
		return
	}
	ids, _, ok := s.scope(w, r, session, false, "project_id")
	if !ok {
		return
	}
	name := strings.TrimSpace(*req.Name)
	if name == "" || len(name) > 50 {
		badRequest(w, "Label name must be between 1 and 50 characters")
		return
	}
	color := "#315b3b"
	if req.Color != nil {
		color = *req.Color
	}
	if !validHexColor(color) {
		badRequest(w, "Label color must be a valid 6-digit hex code (e.g. #315b3b)")
		return
	}
	l, err := projects.CreateLabel(r.Context(), s.Pool, &session.User.ID, ids[0], ids[1], name, color)
	if err != nil {
		storeError(w, err, "failed to create label", "A label with this name already exists in this project")
		return
	}
	writeJSON(w, http.StatusCreated, toLabel(l))
}

func (s *Server) deleteLabel(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, true, "project_id", "label_id")
	if !ok {
		return
	}
	if err := projects.DeleteLabel(r.Context(), s.Pool, &session.User.ID, ids[0], ids[1], ids[2]); err != nil {
		storeError(w, err, "failed to delete label", "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) attachLabel(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	if _, ok := pathIDs(w, r, "workspace_id", "project_id", "task_id"); !ok {
		return
	}
	var req struct {
		LabelID *uuid.UUID `json:"label_id"`
	}
	if !decodeJSON(w, r, &req) || !required(w, req.LabelID) {
		return
	}
	ids, _, ok := s.scope(w, r, session, false, "project_id", "task_id")
	if !ok {
		return
	}
	if err := projects.AttachLabel(r.Context(), s.Pool, &session.User.ID, ids[0], ids[1], ids[2], *req.LabelID); err != nil {
		storeError(w, err, "failed to attach label", "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) detachLabel(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "project_id", "task_id", "label_id")
	if !ok {
		return
	}
	if err := projects.DetachLabel(r.Context(), s.Pool, &session.User.ID, ids[0], ids[1], ids[2], ids[3]); err != nil {
		storeError(w, err, "failed to detach label", "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Assignees
// ---------------------------------------------------------------------------

func (s *Server) assignUser(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	if _, ok := pathIDs(w, r, "workspace_id", "project_id", "task_id"); !ok {
		return
	}
	var req struct {
		UserID *uuid.UUID `json:"user_id"`
	}
	if !decodeJSON(w, r, &req) || !required(w, req.UserID) {
		return
	}
	ids, _, ok := s.scope(w, r, session, false, "project_id", "task_id")
	if !ok {
		return
	}
	a, err := projects.AssignUser(r.Context(), s.Pool, &session.User.ID, ids[0], ids[1], ids[2], *req.UserID)
	if err != nil {
		storeError(w, err, "failed to assign user", "")
		return
	}
	writeJSON(w, http.StatusCreated, toAssignee(a))
}

func (s *Server) unassignUser(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "project_id", "task_id", "user_id")
	if !ok {
		return
	}
	if err := projects.UnassignUser(r.Context(), s.Pool, &session.User.ID, ids[0], ids[1], ids[2], ids[3]); err != nil {
		storeError(w, err, "failed to unassign user", "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// ---------------------------------------------------------------------------
// Subtasks, comments, activity
// ---------------------------------------------------------------------------

func (s *Server) listSubtasks(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "project_id", "task_id")
	if !ok {
		return
	}
	list, err := projects.ListSubtasks(r.Context(), s.Pool, ids[0], ids[1], ids[2])
	if err != nil {
		storeError(w, err, "failed to list subtasks", "")
		return
	}
	writeJSON(w, http.StatusOK, mapSlice(list, toTask))
}

func (s *Server) listComments(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "project_id", "task_id")
	if !ok {
		return
	}
	list, err := projects.ListComments(r.Context(), s.Pool, ids[0], ids[1], ids[2])
	if err != nil {
		storeError(w, err, "failed to list comments", "")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"comments": mapSlice(list, toComment)})
}

func (s *Server) createComment(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	if _, ok := pathIDs(w, r, "workspace_id", "project_id", "task_id"); !ok {
		return
	}
	var req struct {
		Content *string `json:"content"`
	}
	if !decodeJSON(w, r, &req) || !required(w, req.Content) {
		return
	}
	ids, _, ok := s.scope(w, r, session, false, "project_id", "task_id")
	if !ok {
		return
	}
	content := strings.TrimSpace(*req.Content)
	if content == "" || len(content) > 5000 {
		badRequest(w, "Comment content must be between 1 and 5000 characters")
		return
	}
	c, err := projects.CreateComment(r.Context(), s.Pool, session.User.ID, ids[0], ids[1], ids[2], content)
	if err != nil {
		storeError(w, err, "failed to create comment", "")
		return
	}
	writeJSON(w, http.StatusCreated, toComment(c))
}

func (s *Server) deleteComment(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, role, ok := s.scope(w, r, session, false, "project_id", "task_id", "comment_id")
	if !ok {
		return
	}
	if err := projects.DeleteComment(r.Context(), s.Pool, session.User.ID, ids[0], ids[1], ids[2], ids[3], isManager(role)); err != nil {
		storeError(w, err, "failed to delete comment", "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) listActivity(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "project_id", "task_id")
	if !ok {
		return
	}
	list, err := projects.ListTaskActivities(r.Context(), s.Pool, ids[0], ids[1], ids[2])
	if err != nil {
		storeError(w, err, "failed to list task activities", "")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"activities": mapSlice(list, func(a projects.TaskActivity) activityItem {
		return activityItem{a.ID.String(), a.WorkspaceID.String(), a.TaskID.String(), idPtr(a.ActorID), a.ActorName, a.Action, a.Details, ts(a.CreatedAt)}
	})})
}

// ---------------------------------------------------------------------------
// Statuses
// ---------------------------------------------------------------------------

func (s *Server) listStatuses(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "project_id")
	if !ok {
		return
	}
	list, err := projects.ListProjectStatuses(r.Context(), s.Pool, ids[0], ids[1])
	if err != nil {
		storeError(w, err, "failed to list project statuses", "")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"statuses": mapSlice(list, toStatus)})
}

const statusNameTaken = "Status name already exists in this project"

func (s *Server) createStatus(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	if _, ok := pathIDs(w, r, "workspace_id", "project_id"); !ok {
		return
	}
	var req struct {
		Name     *string `json:"name"`
		Category *string `json:"category"`
		Color    *string `json:"color"`
	}
	if !decodeJSON(w, r, &req) || !required(w, req.Name, req.Category) {
		return
	}
	ids, _, ok := s.scope(w, r, session, true, "project_id")
	if !ok {
		return
	}
	name := strings.TrimSpace(*req.Name)
	if name == "" || len(name) > 50 {
		badRequest(w, "Status name must be between 1 and 50 characters")
		return
	}
	if !validStatus(*req.Category) {
		badRequest(w, "Invalid status category")
		return
	}
	if req.Color != nil && !validHexColor(*req.Color) {
		badRequest(w, "Color must be a valid hex code (e.g. #315b3b)")
		return
	}
	st, err := projects.CreateProjectStatus(r.Context(), s.Pool, &session.User.ID, ids[0], ids[1], projects.CreateStatusParams{
		Name: name, Category: *req.Category, Color: req.Color,
	})
	if err != nil {
		storeError(w, err, "failed to create status", statusNameTaken)
		return
	}
	writeJSON(w, http.StatusCreated, toStatus(st))
}

func (s *Server) updateStatus(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	if _, ok := pathIDs(w, r, "workspace_id", "project_id", "status_id"); !ok {
		return
	}
	var req struct {
		Name  *string `json:"name"`
		Color *string `json:"color"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	ids, _, ok := s.scope(w, r, session, true, "project_id", "status_id")
	if !ok {
		return
	}
	if req.Name != nil {
		if n := strings.TrimSpace(*req.Name); n == "" || len(n) > 50 {
			badRequest(w, "Status name must be between 1 and 50 characters")
			return
		}
	}
	if req.Color != nil && !validHexColor(*req.Color) {
		badRequest(w, "Color must be a valid hex code (e.g. #315b3b)")
		return
	}
	st, err := projects.UpdateProjectStatus(r.Context(), s.Pool, &session.User.ID, ids[0], ids[1], ids[2], projects.UpdateStatusParams{
		Name: req.Name, Color: req.Color,
	})
	if err != nil {
		storeError(w, err, "failed to update status", statusNameTaken)
		return
	}
	writeJSON(w, http.StatusOK, toStatus(st))
}

func (s *Server) reorderStatuses(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	if _, ok := pathIDs(w, r, "workspace_id", "project_id"); !ok {
		return
	}
	var req struct {
		StatusIDs *[]uuid.UUID `json:"status_ids"`
	}
	if !decodeJSON(w, r, &req) || !required(w, req.StatusIDs) {
		return
	}
	ids, _, ok := s.scope(w, r, session, true, "project_id")
	if !ok {
		return
	}
	if err := projects.ReorderProjectStatuses(r.Context(), s.Pool, &session.User.ID, ids[0], ids[1], *req.StatusIDs); err != nil {
		storeError(w, err, "failed to reorder statuses", "")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s *Server) deleteStatus(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	if _, ok := pathIDs(w, r, "workspace_id", "project_id", "status_id"); !ok {
		return
	}
	remapTo, ok := queryUUID(w, r, "remap_to")
	if !ok {
		return
	}
	ids, _, ok := s.scope(w, r, session, true, "project_id", "status_id")
	if !ok {
		return
	}
	if err := projects.DeleteProjectStatus(r.Context(), s.Pool, &session.User.ID, ids[0], ids[1], ids[2], remapTo); err != nil {
		storeError(w, err, "failed to delete status",
			"Cannot delete status: it may be the last remaining status, or active tasks require a valid remap_to target")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
