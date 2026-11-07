// Package projects persists projects, tasks, statuses, labels, assignees, subtasks, comments and activity.
package projects

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/maruf-pfc/trestle/internal/identity"
)

var (
	ErrConflict  = identity.ErrConflict
	ErrNotFound  = identity.ErrNotFound
	ErrForbidden = identity.ErrForbidden
	mapErr       = identity.MapErr
	audit        = identity.RecordAuditEvent
)

type Querier = identity.Querier

type Project struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Name        string
	Slug        string
	Key         string
	Description string
	TaskCounter int64
	Archived    bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type Label struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	Name        string
	Color       string
	CreatedAt   time.Time
}

type TaskAssignee struct {
	TaskID      uuid.UUID
	UserID      uuid.UUID
	Username    string
	DisplayName string
	AvatarURL   *string
	AssignedBy  *uuid.UUID
	CreatedAt   time.Time
}

type TaskComment struct {
	ID             uuid.UUID
	WorkspaceID    uuid.UUID
	TaskID         uuid.UUID
	UserID         uuid.UUID
	AuthorName     string
	AuthorUsername string
	Content        string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type TaskActivity struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	TaskID      uuid.UUID
	ActorID     *uuid.UUID
	ActorName   *string
	Action      string
	Details     json.RawMessage
	CreatedAt   time.Time
}

type Task struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	ProjectKey  string
	TaskNumber  int64
	HumanKey    string
	Title       string
	Description string
	Status      string
	StatusID    *uuid.UUID
	StatusName  *string
	StatusColor *string
	Priority    string
	DueDate     *time.Time
	CreatedBy   *uuid.UUID
	ParentID    *uuid.UUID
	Labels      []Label
	Assignees   []TaskAssignee
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type PaginatedTasks struct {
	Tasks  []Task
	Total  int64
	Limit  int64
	Offset int64
}

type CreateProjectParams struct {
	Name, Slug, Key, Description string
}

type CreateTaskParams struct {
	Title       string
	Description string
	Status      string
	Priority    string
	DueDate     *time.Time
	ParentID    *uuid.UUID
	StatusID    *uuid.UUID
}

// UpdateTaskParams: nil means "leave unchanged". SetDueDate with a nil DueDate clears it.
type UpdateTaskParams struct {
	Title       *string
	Description *string
	Status      *string
	StatusID    *uuid.UUID
	Priority    *string
	SetDueDate  bool
	DueDate     *time.Time
}

type ListTasksParams struct {
	Status     *string
	StatusID   *uuid.UUID
	LabelID    *uuid.UUID
	AssigneeID *uuid.UUID
	Limit      int64
	Offset     int64
}

func humanKey(key string, n int64) string { return fmt.Sprintf("%s-%d", key, n) }

// ---------------------------------------------------------------------------
// Projects
// ---------------------------------------------------------------------------

const projectColumns = "id, workspace_id, name, slug, key, description, task_counter, archived_at, created_at, updated_at"

func scanProject(row pgx.Row) (Project, error) {
	var p Project
	var archivedAt *time.Time
	err := row.Scan(&p.ID, &p.WorkspaceID, &p.Name, &p.Slug, &p.Key, &p.Description, &p.TaskCounter, &archivedAt, &p.CreatedAt, &p.UpdatedAt)
	p.Archived = archivedAt != nil
	return p, err
}

func CreateProject(ctx context.Context, pool *pgxpool.Pool, actorID *uuid.UUID, workspaceID uuid.UUID, params CreateProjectParams) (Project, error) {
	projectID := uuid.New()
	now := time.Now().UTC()
	tx, err := pool.Begin(ctx)
	if err != nil {
		return Project{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, `INSERT INTO projects (id, workspace_id, name, slug, key, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7)`,
		projectID, workspaceID, params.Name, params.Slug, params.Key, params.Description, now); err != nil {
		return Project{}, mapErr(err)
	}
	if err := audit(ctx, tx, actorID, "project.created", map[string]any{
		"workspace_id": workspaceID, "project_id": projectID, "slug": params.Slug, "key": params.Key,
	}); err != nil {
		return Project{}, err
	}
	// Seed the default statuses with stable categories.
	if _, err := tx.Exec(ctx, `INSERT INTO task_statuses (id, workspace_id, project_id, name, category, color, position, created_at, updated_at)
		VALUES
		   (gen_random_uuid(), $1, $2, 'Backlog', 'backlog', '#4c544c', 0, $3, $3),
		   (gen_random_uuid(), $1, $2, 'To Do', 'unstarted', '#1e4d79', 1, $3, $3),
		   (gen_random_uuid(), $1, $2, 'In Progress', 'started', '#775300', 2, $3, $3),
		   (gen_random_uuid(), $1, $2, 'Done', 'completed', '#26633b', 3, $3, $3),
		   (gen_random_uuid(), $1, $2, 'Cancelled', 'cancelled', '#9f302b', 4, $3, $3)`,
		workspaceID, projectID, now); err != nil {
		return Project{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Project{}, err
	}
	return Project{ID: projectID, WorkspaceID: workspaceID, Name: params.Name, Slug: params.Slug, Key: params.Key,
		Description: params.Description, CreatedAt: now, UpdatedAt: now}, nil
}

func ListProjects(ctx context.Context, db Querier, workspaceID uuid.UUID, includeArchived bool) ([]Project, error) {
	rows, err := db.Query(ctx, "SELECT "+projectColumns+` FROM projects
		WHERE workspace_id = $1 AND ($2 OR archived_at IS NULL)
		ORDER BY name ASC`, workspaceID, includeArchived)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Project, error) { return scanProject(r) })
}

// GetProject returns ErrNotFound when the project is not in the workspace.
func GetProject(ctx context.Context, db Querier, workspaceID, projectID uuid.UUID) (Project, error) {
	p, err := scanProject(db.QueryRow(ctx, "SELECT "+projectColumns+" FROM projects WHERE id = $1 AND workspace_id = $2", projectID, workspaceID))
	return p, mapErr(err)
}

func ArchiveProject(ctx context.Context, db Querier, actorID *uuid.UUID, workspaceID, projectID uuid.UUID) (Project, error) {
	p, err := scanProject(db.QueryRow(ctx, `UPDATE projects
		SET archived_at = $1, updated_at = $1
		WHERE id = $2 AND workspace_id = $3
		RETURNING `+projectColumns, time.Now().UTC(), projectID, workspaceID))
	if err != nil {
		return Project{}, mapErr(err)
	}
	return p, audit(ctx, db, actorID, "project.archived", map[string]any{"workspace_id": workspaceID, "project_id": projectID})
}

// ---------------------------------------------------------------------------
// Tasks
// ---------------------------------------------------------------------------

func CreateTask(ctx context.Context, pool *pgxpool.Pool, actorID *uuid.UUID, workspaceID, projectID uuid.UUID, params CreateTaskParams) (Task, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return Task{}, err
	}
	defer tx.Rollback(ctx)

	if params.ParentID != nil {
		var exists bool
		if err := tx.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM tasks WHERE id = $1 AND project_id = $2 AND workspace_id = $3)",
			*params.ParentID, projectID, workspaceID).Scan(&exists); err != nil {
			return Task{}, err
		}
		if !exists {
			return Task{}, ErrNotFound
		}
	}

	t := Task{ID: uuid.New(), WorkspaceID: workspaceID, ProjectID: projectID, Title: params.Title, Description: params.Description,
		Priority: params.Priority, DueDate: params.DueDate, CreatedBy: actorID, ParentID: params.ParentID,
		Labels: []Label{}, Assignees: []TaskAssignee{}}

	// Increment the project's task counter atomically.
	if err := tx.QueryRow(ctx, `UPDATE projects
		SET task_counter = task_counter + 1, updated_at = now()
		WHERE id = $1 AND workspace_id = $2 AND archived_at IS NULL
		RETURNING key, task_counter`, projectID, workspaceID).Scan(&t.ProjectKey, &t.TaskNumber); err != nil {
		return Task{}, mapErr(err)
	}
	now := time.Now().UTC()
	t.CreatedAt, t.UpdatedAt = now, now

	// Resolve status_id and its canonical category.
	var row pgx.Row
	if params.StatusID != nil {
		row = tx.QueryRow(ctx, "SELECT id, name, category, color FROM task_statuses WHERE id = $1 AND project_id = $2 AND workspace_id = $3",
			*params.StatusID, projectID, workspaceID)
	} else {
		row = tx.QueryRow(ctx, "SELECT id, name, category, color FROM task_statuses WHERE project_id = $1 AND workspace_id = $2 AND (category = $3 OR name = $3) ORDER BY position ASC LIMIT 1",
			projectID, workspaceID, params.Status)
	}
	var sid uuid.UUID
	var sname, scolor string
	switch err := row.Scan(&sid, &sname, &t.Status, &scolor); {
	case err == nil:
		t.StatusID, t.StatusName, t.StatusColor = &sid, &sname, &scolor
	case errors.Is(err, pgx.ErrNoRows):
		t.Status = params.Status
	default:
		return Task{}, err
	}

	if _, err := tx.Exec(ctx, `INSERT INTO tasks
		   (id, workspace_id, project_id, task_number, title, description, status, priority, due_date, created_by, parent_id, status_id, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $13)`,
		t.ID, workspaceID, projectID, t.TaskNumber, t.Title, t.Description, t.Status, t.Priority, t.DueDate, actorID, t.ParentID, t.StatusID, now); err != nil {
		return Task{}, mapErr(err)
	}
	t.HumanKey = humanKey(t.ProjectKey, t.TaskNumber)

	if err := audit(ctx, tx, actorID, "task.created", map[string]any{
		"workspace_id": workspaceID, "project_id": projectID, "task_id": t.ID, "human_key": t.HumanKey,
		"parent_id": t.ParentID, "status_id": t.StatusID,
	}); err != nil {
		return Task{}, err
	}
	if err := RecordTaskActivity(ctx, tx, workspaceID, t.ID, actorID, "task.created", map[string]any{
		"human_key": t.HumanKey, "title": t.Title, "status": t.Status, "status_id": t.StatusID, "priority": t.Priority,
	}); err != nil {
		return Task{}, err
	}
	if t.ParentID != nil {
		if err := RecordTaskActivity(ctx, tx, workspaceID, *t.ParentID, actorID, "subtask.created", map[string]any{
			"subtask_id": t.ID, "subtask_key": t.HumanKey, "title": t.Title,
		}); err != nil {
			return Task{}, err
		}
	}
	return t, tx.Commit(ctx)
}

func fetchLabels(ctx context.Context, db Querier, taskIDs []uuid.UUID) (map[uuid.UUID][]Label, error) {
	out := map[uuid.UUID][]Label{}
	if len(taskIDs) == 0 {
		return out, nil
	}
	rows, err := db.Query(ctx, `SELECT tla.task_id, l.id, l.workspace_id, l.project_id, l.name, l.color, l.created_at
		FROM task_label_assignments tla
		JOIN labels l ON l.id = tla.label_id
		WHERE tla.task_id = ANY($1)
		ORDER BY l.name ASC`, taskIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var taskID uuid.UUID
		var l Label
		if err := rows.Scan(&taskID, &l.ID, &l.WorkspaceID, &l.ProjectID, &l.Name, &l.Color, &l.CreatedAt); err != nil {
			return nil, err
		}
		out[taskID] = append(out[taskID], l)
	}
	return out, rows.Err()
}

func fetchAssignees(ctx context.Context, db Querier, taskIDs []uuid.UUID) (map[uuid.UUID][]TaskAssignee, error) {
	out := map[uuid.UUID][]TaskAssignee{}
	if len(taskIDs) == 0 {
		return out, nil
	}
	rows, err := db.Query(ctx, `SELECT ta.task_id, ta.user_id, u.username, u.display_name, ta.assigned_by, ta.created_at
		FROM task_assignees ta
		JOIN users u ON u.id = ta.user_id
		WHERE ta.task_id = ANY($1)
		ORDER BY u.display_name ASC`, taskIDs)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var a TaskAssignee
		if err := rows.Scan(&a.TaskID, &a.UserID, &a.Username, &a.DisplayName, &a.AssignedBy, &a.CreatedAt); err != nil {
			return nil, err
		}
		out[a.TaskID] = append(out[a.TaskID], a)
	}
	return out, rows.Err()
}

const taskColumns = `t.id, t.workspace_id, t.project_id, p.key, t.task_number, t.title, t.description,
	t.status, t.priority, t.due_date, t.created_by, t.parent_id,
	t.status_id, ts.name, ts.color, t.created_at, t.updated_at
	FROM tasks t
	JOIN projects p ON p.id = t.project_id
	LEFT JOIN task_statuses ts ON ts.id = t.status_id`

// queryTasks runs a task query and attaches labels and assignees.
func queryTasks(ctx context.Context, db Querier, sql string, args ...any) ([]Task, error) {
	rows, err := db.Query(ctx, "SELECT "+taskColumns+" "+sql, args...)
	if err != nil {
		return nil, err
	}
	tasks, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (Task, error) {
		var t Task
		err := r.Scan(&t.ID, &t.WorkspaceID, &t.ProjectID, &t.ProjectKey, &t.TaskNumber, &t.Title, &t.Description,
			&t.Status, &t.Priority, &t.DueDate, &t.CreatedBy, &t.ParentID,
			&t.StatusID, &t.StatusName, &t.StatusColor, &t.CreatedAt, &t.UpdatedAt)
		t.HumanKey = humanKey(t.ProjectKey, t.TaskNumber)
		return t, err
	})
	if err != nil {
		return nil, err
	}
	ids := make([]uuid.UUID, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
	}
	labels, err := fetchLabels(ctx, db, ids)
	if err != nil {
		return nil, err
	}
	assignees, err := fetchAssignees(ctx, db, ids)
	if err != nil {
		return nil, err
	}
	for i := range tasks {
		tasks[i].Labels = append([]Label{}, labels[tasks[i].ID]...)
		tasks[i].Assignees = append([]TaskAssignee{}, assignees[tasks[i].ID]...)
	}
	return tasks, nil
}

const taskFilter = `WHERE t.project_id = $1 AND t.workspace_id = $2
	  AND ($3::TEXT IS NULL OR t.status = $3)
	  AND ($4::UUID IS NULL OR t.status_id = $4)
	  AND ($5::UUID IS NULL OR EXISTS (
	      SELECT 1 FROM task_label_assignments tla WHERE tla.task_id = t.id AND tla.label_id = $5
	  ))
	  AND ($6::UUID IS NULL OR EXISTS (
	      SELECT 1 FROM task_assignees ta WHERE ta.task_id = t.id AND ta.user_id = $6
	  ))`

func ListTasks(ctx context.Context, db Querier, workspaceID, projectID uuid.UUID, params ListTasksParams) (PaginatedTasks, error) {
	if _, err := GetProject(ctx, db, workspaceID, projectID); err != nil {
		return PaginatedTasks{}, err
	}
	limit := min(max(params.Limit, 1), 100)
	offset := max(params.Offset, 0)
	args := []any{projectID, workspaceID, params.Status, params.StatusID, params.LabelID, params.AssigneeID}

	var total int64
	if err := db.QueryRow(ctx, "SELECT COUNT(*) FROM tasks t "+taskFilter, args...).Scan(&total); err != nil {
		return PaginatedTasks{}, err
	}
	tasks, err := queryTasks(ctx, db, taskFilter+" ORDER BY t.task_number DESC LIMIT $7 OFFSET $8", append(args, limit, offset)...)
	if err != nil {
		return PaginatedTasks{}, err
	}
	return PaginatedTasks{Tasks: tasks, Total: total, Limit: limit, Offset: offset}, nil
}

// GetTask returns ErrNotFound when the task is not in the project and workspace.
func GetTask(ctx context.Context, db Querier, workspaceID, projectID, taskID uuid.UUID) (Task, error) {
	tasks, err := queryTasks(ctx, db, "WHERE t.id = $1 AND t.project_id = $2 AND t.workspace_id = $3", taskID, projectID, workspaceID)
	if err != nil {
		return Task{}, err
	}
	if len(tasks) == 0 {
		return Task{}, ErrNotFound
	}
	return tasks[0], nil
}

// taskKey returns the task's human key with one query; ErrNotFound when it is not in the project and workspace.
func taskKey(ctx context.Context, db Querier, workspaceID, projectID, taskID uuid.UUID) (string, error) {
	var key string
	var n int64
	err := db.QueryRow(ctx, `SELECT p.key, t.task_number FROM tasks t JOIN projects p ON p.id = t.project_id
		WHERE t.id = $1 AND t.project_id = $2 AND t.workspace_id = $3`, taskID, projectID, workspaceID).Scan(&key, &n)
	return humanKey(key, n), mapErr(err)
}

func UpdateTask(ctx context.Context, pool *pgxpool.Pool, actorID *uuid.UUID, workspaceID, projectID, taskID uuid.UUID, params UpdateTaskParams) (Task, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return Task{}, err
	}
	defer tx.Rollback(ctx)
	// Compare under a row lock so concurrent identical writes log one change.
	var previousStatus, previousPriority string
	var previousStatusID *uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT status, status_id, priority FROM tasks
		WHERE id = $1 AND project_id = $2 AND workspace_id = $3 FOR UPDATE`, taskID, projectID, workspaceID).
		Scan(&previousStatus, &previousStatusID, &previousPriority); err != nil {
		return Task{}, mapErr(err)
	}
	var targetStatusID *uuid.UUID
	var targetCategory *string
	if params.StatusID != nil {
		var id uuid.UUID
		var category string
		if err := tx.QueryRow(ctx, "SELECT id, category FROM task_statuses WHERE id = $1 AND project_id = $2 AND workspace_id = $3",
			*params.StatusID, projectID, workspaceID).Scan(&id, &category); err != nil {
			return Task{}, mapErr(err)
		}
		targetStatusID, targetCategory = &id, &category
	} else if params.Status != nil {
		var id uuid.UUID
		var category string
		err := tx.QueryRow(ctx, "SELECT id, category FROM task_statuses WHERE project_id = $1 AND workspace_id = $2 AND (category = $3 OR name = $3) ORDER BY position ASC LIMIT 1",
			projectID, workspaceID, *params.Status).Scan(&id, &category)
		switch {
		case err == nil:
			targetStatusID, targetCategory = &id, &category
		case errors.Is(err, pgx.ErrNoRows):
			targetCategory = params.Status
		default:
			return Task{}, err
		}
	}

	tag, err := tx.Exec(ctx, `UPDATE tasks
		SET updated_at = $1,
		     title = COALESCE($2, title),
		     description = COALESCE($3, description),
		     status = CASE WHEN $4 THEN $5 ELSE status END,
		     status_id = CASE WHEN $6 THEN $7 ELSE status_id END,
		     priority = COALESCE($8, priority),
		     due_date = CASE WHEN $9 THEN $10 ELSE due_date END
		WHERE id = $11 AND project_id = $12 AND workspace_id = $13`,
		time.Now().UTC(), params.Title, params.Description, targetCategory != nil, targetCategory,
		targetStatusID != nil, targetStatusID, params.Priority, params.SetDueDate, params.DueDate,
		taskID, projectID, workspaceID)
	if err != nil {
		return Task{}, mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return Task{}, ErrNotFound
	}
	if err := audit(ctx, tx, actorID, "task.updated", map[string]any{
		"workspace_id": workspaceID, "project_id": projectID, "task_id": taskID,
	}); err != nil {
		return Task{}, err
	}
	statusIDChanged := targetStatusID != nil && (previousStatusID == nil || *previousStatusID != *targetStatusID)
	if targetCategory != nil && (*targetCategory != previousStatus || statusIDChanged) {
		if err := RecordTaskActivity(ctx, tx, workspaceID, taskID, actorID, "task.status_changed", map[string]any{
			"status": *targetCategory, "status_id": targetStatusID,
		}); err != nil {
			return Task{}, err
		}
	}
	if params.Priority != nil && *params.Priority != previousPriority {
		if err := RecordTaskActivity(ctx, tx, workspaceID, taskID, actorID, "task.priority_changed", map[string]any{"priority": *params.Priority}); err != nil {
			return Task{}, err
		}
	}
	updated, err := GetTask(ctx, tx, workspaceID, projectID, taskID)
	if err != nil {
		return Task{}, err
	}
	return updated, tx.Commit(ctx)
}

func DeleteTask(ctx context.Context, db Querier, actorID *uuid.UUID, workspaceID, projectID, taskID uuid.UUID) error {
	tag, err := db.Exec(ctx, "DELETE FROM tasks WHERE id = $1 AND project_id = $2 AND workspace_id = $3", taskID, projectID, workspaceID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return audit(ctx, db, actorID, "task.deleted", map[string]any{"workspace_id": workspaceID, "project_id": projectID, "task_id": taskID})
}

// ---------------------------------------------------------------------------
// Labels
// ---------------------------------------------------------------------------

func CreateLabel(ctx context.Context, db Querier, actorID *uuid.UUID, workspaceID, projectID uuid.UUID, name, color string) (Label, error) {
	if _, err := GetProject(ctx, db, workspaceID, projectID); err != nil {
		return Label{}, err
	}
	l := Label{ID: uuid.New(), WorkspaceID: workspaceID, ProjectID: projectID, Name: name, Color: color, CreatedAt: time.Now().UTC()}
	if _, err := db.Exec(ctx, "INSERT INTO labels (id, workspace_id, project_id, name, color, created_at) VALUES ($1, $2, $3, $4, $5, $6)",
		l.ID, workspaceID, projectID, name, color, l.CreatedAt); err != nil {
		return Label{}, mapErr(err)
	}
	return l, audit(ctx, db, actorID, "label.created", map[string]any{
		"workspace_id": workspaceID, "project_id": projectID, "label_id": l.ID, "name": name,
	})
}

func ListLabels(ctx context.Context, db Querier, workspaceID, projectID uuid.UUID) ([]Label, error) {
	if _, err := GetProject(ctx, db, workspaceID, projectID); err != nil {
		return nil, err
	}
	rows, err := db.Query(ctx, `SELECT id, workspace_id, project_id, name, color, created_at
		FROM labels WHERE workspace_id = $1 AND project_id = $2 ORDER BY name ASC`, workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Label, error) {
		var l Label
		return l, r.Scan(&l.ID, &l.WorkspaceID, &l.ProjectID, &l.Name, &l.Color, &l.CreatedAt)
	})
}

func DeleteLabel(ctx context.Context, db Querier, actorID *uuid.UUID, workspaceID, projectID, labelID uuid.UUID) error {
	tag, err := db.Exec(ctx, "DELETE FROM labels WHERE id = $1 AND workspace_id = $2 AND project_id = $3", labelID, workspaceID, projectID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return audit(ctx, db, actorID, "label.deleted", map[string]any{"workspace_id": workspaceID, "project_id": projectID, "label_id": labelID})
}

func AttachLabel(ctx context.Context, db Querier, actorID *uuid.UUID, workspaceID, projectID, taskID, labelID uuid.UUID) error {
	key, err := taskKey(ctx, db, workspaceID, projectID, taskID)
	if err != nil {
		return err
	}
	var name, color string
	if err := db.QueryRow(ctx, "SELECT name, color FROM labels WHERE id = $1 AND project_id = $2 AND workspace_id = $3",
		labelID, projectID, workspaceID).Scan(&name, &color); err != nil {
		return mapErr(err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO task_label_assignments (task_id, label_id, created_at)
		VALUES ($1, $2, now()) ON CONFLICT (task_id, label_id) DO NOTHING`, taskID, labelID); err != nil {
		return err
	}
	return RecordTaskActivity(ctx, db, workspaceID, taskID, actorID, "label.attached", map[string]any{
		"label_id": labelID, "name": name, "color": color, "task_key": key,
	})
}

func DetachLabel(ctx context.Context, db Querier, actorID *uuid.UUID, workspaceID, projectID, taskID, labelID uuid.UUID) error {
	key, err := taskKey(ctx, db, workspaceID, projectID, taskID)
	if err != nil {
		return err
	}
	tag, err := db.Exec(ctx, "DELETE FROM task_label_assignments WHERE task_id = $1 AND label_id = $2", taskID, labelID)
	if err != nil || tag.RowsAffected() == 0 {
		return err
	}
	return RecordTaskActivity(ctx, db, workspaceID, taskID, actorID, "label.detached", map[string]any{
		"label_id": labelID, "task_key": key,
	})
}

// ---------------------------------------------------------------------------
// Assignees
// ---------------------------------------------------------------------------

func AssignUser(ctx context.Context, db Querier, actorID *uuid.UUID, workspaceID, projectID, taskID, userID uuid.UUID) (TaskAssignee, error) {
	key, err := taskKey(ctx, db, workspaceID, projectID, taskID)
	if err != nil {
		return TaskAssignee{}, err
	}
	a := TaskAssignee{TaskID: taskID, UserID: userID, AssignedBy: actorID, CreatedAt: time.Now().UTC()}
	if err := db.QueryRow(ctx, `SELECT u.username, u.display_name
		FROM workspace_memberships wm
		JOIN users u ON u.id = wm.user_id
		WHERE wm.workspace_id = $1 AND wm.user_id = $2`, workspaceID, userID).Scan(&a.Username, &a.DisplayName); err != nil {
		return TaskAssignee{}, mapErr(err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO task_assignees (task_id, user_id, assigned_by, created_at)
		VALUES ($1, $2, $3, $4)
		ON CONFLICT (task_id, user_id) DO UPDATE SET assigned_by = $3`, taskID, userID, actorID, a.CreatedAt); err != nil {
		return TaskAssignee{}, err
	}
	return a, RecordTaskActivity(ctx, db, workspaceID, taskID, actorID, "user.assigned", map[string]any{
		"user_id": userID, "display_name": a.DisplayName, "username": a.Username, "task_key": key,
	})
}

func UnassignUser(ctx context.Context, db Querier, actorID *uuid.UUID, workspaceID, projectID, taskID, userID uuid.UUID) error {
	key, err := taskKey(ctx, db, workspaceID, projectID, taskID)
	if err != nil {
		return err
	}
	tag, err := db.Exec(ctx, "DELETE FROM task_assignees WHERE task_id = $1 AND user_id = $2", taskID, userID)
	if err != nil || tag.RowsAffected() == 0 {
		return err
	}
	return RecordTaskActivity(ctx, db, workspaceID, taskID, actorID, "user.unassigned", map[string]any{
		"user_id": userID, "task_key": key,
	})
}

// ---------------------------------------------------------------------------
// Subtasks
// ---------------------------------------------------------------------------

func ListSubtasks(ctx context.Context, db Querier, workspaceID, projectID, parentID uuid.UUID) ([]Task, error) {
	if _, err := taskKey(ctx, db, workspaceID, projectID, parentID); err != nil {
		return nil, err
	}
	return queryTasks(ctx, db, "WHERE t.parent_id = $1 AND t.project_id = $2 AND t.workspace_id = $3 ORDER BY t.task_number ASC",
		parentID, projectID, workspaceID)
}

// ---------------------------------------------------------------------------
// Comments
// ---------------------------------------------------------------------------

func CreateComment(ctx context.Context, db Querier, actorID, workspaceID, projectID, taskID uuid.UUID, content string) (TaskComment, error) {
	key, err := taskKey(ctx, db, workspaceID, projectID, taskID)
	if err != nil {
		return TaskComment{}, err
	}
	now := time.Now().UTC()
	c := TaskComment{ID: uuid.New(), WorkspaceID: workspaceID, TaskID: taskID, UserID: actorID, Content: content, CreatedAt: now, UpdatedAt: now}
	if err := db.QueryRow(ctx, "SELECT username, display_name FROM users WHERE id = $1", actorID).Scan(&c.AuthorUsername, &c.AuthorName); err != nil {
		return TaskComment{}, mapErr(err)
	}
	if _, err := db.Exec(ctx, `INSERT INTO task_comments (id, workspace_id, task_id, user_id, content, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $6)`, c.ID, workspaceID, taskID, actorID, content, now); err != nil {
		return TaskComment{}, err
	}
	return c, RecordTaskActivity(ctx, db, workspaceID, taskID, &actorID, "comment.added", map[string]any{
		"comment_id": c.ID, "author_name": c.AuthorName, "task_key": key,
	})
}

func ListComments(ctx context.Context, db Querier, workspaceID, projectID, taskID uuid.UUID) ([]TaskComment, error) {
	if _, err := taskKey(ctx, db, workspaceID, projectID, taskID); err != nil {
		return nil, err
	}
	rows, err := db.Query(ctx, `SELECT c.id, c.workspace_id, c.task_id, c.user_id, u.display_name, u.username, c.content, c.created_at, c.updated_at
		FROM task_comments c
		JOIN users u ON u.id = c.user_id
		WHERE c.task_id = $1 AND c.workspace_id = $2
		ORDER BY c.created_at ASC`, taskID, workspaceID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (TaskComment, error) {
		var c TaskComment
		return c, r.Scan(&c.ID, &c.WorkspaceID, &c.TaskID, &c.UserID, &c.AuthorName, &c.AuthorUsername, &c.Content, &c.CreatedAt, &c.UpdatedAt)
	})
}

// DeleteComment lets authors delete their own comments, and admins or owners delete any.
func DeleteComment(ctx context.Context, db Querier, actorID, workspaceID, projectID, taskID, commentID uuid.UUID, isAdminOrOwner bool) error {
	if _, err := taskKey(ctx, db, workspaceID, projectID, taskID); err != nil {
		return err
	}
	var authorID, commentWorkspace uuid.UUID
	if err := db.QueryRow(ctx, "SELECT user_id, workspace_id FROM task_comments WHERE id = $1 AND task_id = $2", commentID, taskID).
		Scan(&authorID, &commentWorkspace); err != nil {
		return mapErr(err)
	}
	if commentWorkspace != workspaceID {
		return ErrNotFound
	}
	if !isAdminOrOwner && authorID != actorID {
		return ErrForbidden
	}
	_, err := db.Exec(ctx, "DELETE FROM task_comments WHERE id = $1", commentID)
	return err
}

// ---------------------------------------------------------------------------
// Activity
// ---------------------------------------------------------------------------

func RecordTaskActivity(ctx context.Context, db Querier, workspaceID, taskID uuid.UUID, actorID *uuid.UUID, action string, details map[string]any) error {
	b, err := json.Marshal(details)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, `INSERT INTO task_activities (id, workspace_id, task_id, actor_id, action, details, created_at)
		VALUES (gen_random_uuid(), $1, $2, $3, $4, $5, now())`, workspaceID, taskID, actorID, action, b)
	return err
}

func ListTaskActivities(ctx context.Context, db Querier, workspaceID, projectID, taskID uuid.UUID) ([]TaskActivity, error) {
	if _, err := taskKey(ctx, db, workspaceID, projectID, taskID); err != nil {
		return nil, err
	}
	rows, err := db.Query(ctx, `SELECT a.id, a.workspace_id, a.task_id, a.actor_id, u.display_name, a.action, a.details, a.created_at
		FROM task_activities a
		LEFT JOIN users u ON u.id = a.actor_id
		WHERE a.task_id = $1 AND a.workspace_id = $2
		ORDER BY a.created_at DESC
		LIMIT 50`, taskID, workspaceID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (TaskActivity, error) {
		var a TaskActivity
		var details []byte
		err := r.Scan(&a.ID, &a.WorkspaceID, &a.TaskID, &a.ActorID, &a.ActorName, &a.Action, &details, &a.CreatedAt)
		a.Details = details
		return a, err
	})
}

// ---------------------------------------------------------------------------
// Task statuses
// ---------------------------------------------------------------------------

type TaskStatus struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	ProjectID   uuid.UUID
	Name        string
	Category    string
	Color       string
	Position    int32
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

type CreateStatusParams struct {
	Name     string
	Category string
	Color    *string
}

type UpdateStatusParams struct {
	Name  *string
	Color *string
}

const statusColumns = "id, workspace_id, project_id, name, category, color, position, created_at, updated_at"

func scanStatus(row pgx.Row) (TaskStatus, error) {
	var s TaskStatus
	return s, row.Scan(&s.ID, &s.WorkspaceID, &s.ProjectID, &s.Name, &s.Category, &s.Color, &s.Position, &s.CreatedAt, &s.UpdatedAt)
}

func ListProjectStatuses(ctx context.Context, db Querier, workspaceID, projectID uuid.UUID) ([]TaskStatus, error) {
	if _, err := GetProject(ctx, db, workspaceID, projectID); err != nil {
		return nil, err
	}
	rows, err := db.Query(ctx, "SELECT "+statusColumns+` FROM task_statuses
		WHERE workspace_id = $1 AND project_id = $2
		ORDER BY position ASC, created_at ASC`, workspaceID, projectID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (TaskStatus, error) { return scanStatus(r) })
}

var validCategories = map[string]bool{"backlog": true, "unstarted": true, "started": true, "completed": true, "cancelled": true}

func CreateProjectStatus(ctx context.Context, pool *pgxpool.Pool, actorID *uuid.UUID, workspaceID, projectID uuid.UUID, params CreateStatusParams) (TaskStatus, error) {
	if _, err := GetProject(ctx, pool, workspaceID, projectID); err != nil {
		return TaskStatus{}, err
	}
	name := strings.TrimSpace(params.Name)
	if name == "" || len(name) > 50 || !validCategories[params.Category] {
		return TaskStatus{}, ErrConflict
	}
	color := "#315b3b"
	if params.Color != nil {
		color = *params.Color
	}
	now := time.Now().UTC()
	s := TaskStatus{ID: uuid.New(), WorkspaceID: workspaceID, ProjectID: projectID, Name: name, Category: params.Category, Color: color, CreatedAt: now, UpdatedAt: now}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return TaskStatus{}, err
	}
	defer tx.Rollback(ctx)
	if err := tx.QueryRow(ctx, "SELECT COALESCE(MAX(position) + 1, 0) FROM task_statuses WHERE workspace_id = $1 AND project_id = $2",
		workspaceID, projectID).Scan(&s.Position); err != nil {
		return TaskStatus{}, err
	}
	if _, err := tx.Exec(ctx, `INSERT INTO task_statuses (id, workspace_id, project_id, name, category, color, position, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)`, s.ID, workspaceID, projectID, name, s.Category, color, s.Position, now); err != nil {
		return TaskStatus{}, mapErr(err)
	}
	if err := audit(ctx, tx, actorID, "status.created", map[string]any{
		"workspace_id": workspaceID, "project_id": projectID, "status_id": s.ID, "name": name, "category": s.Category,
	}); err != nil {
		return TaskStatus{}, err
	}
	return s, tx.Commit(ctx)
}

func UpdateProjectStatus(ctx context.Context, pool *pgxpool.Pool, actorID *uuid.UUID, workspaceID, projectID, statusID uuid.UUID, params UpdateStatusParams) (TaskStatus, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return TaskStatus{}, err
	}
	defer tx.Rollback(ctx)
	s, err := scanStatus(tx.QueryRow(ctx, "SELECT "+statusColumns+" FROM task_statuses WHERE id = $1 AND workspace_id = $2 AND project_id = $3",
		statusID, workspaceID, projectID))
	if err != nil {
		return TaskStatus{}, mapErr(err)
	}
	if params.Name != nil && strings.TrimSpace(*params.Name) != "" {
		s.Name = strings.TrimSpace(*params.Name)
	}
	if params.Color != nil && strings.TrimSpace(*params.Color) != "" {
		s.Color = strings.TrimSpace(*params.Color)
	}
	s.UpdatedAt = time.Now().UTC()
	if _, err := tx.Exec(ctx, `UPDATE task_statuses SET name = $1, color = $2, updated_at = $3
		WHERE id = $4 AND workspace_id = $5 AND project_id = $6`, s.Name, s.Color, s.UpdatedAt, statusID, workspaceID, projectID); err != nil {
		return TaskStatus{}, mapErr(err)
	}
	if err := audit(ctx, tx, actorID, "status.updated", map[string]any{
		"workspace_id": workspaceID, "project_id": projectID, "status_id": statusID, "name": s.Name, "color": s.Color,
	}); err != nil {
		return TaskStatus{}, err
	}
	return s, tx.Commit(ctx)
}

func ReorderProjectStatuses(ctx context.Context, pool *pgxpool.Pool, actorID *uuid.UUID, workspaceID, projectID uuid.UUID, statusIDs []uuid.UUID) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	for pos, sid := range statusIDs {
		tag, err := tx.Exec(ctx, `UPDATE task_statuses SET position = $1, updated_at = now()
			WHERE id = $2 AND workspace_id = $3 AND project_id = $4`, int32(pos), sid, workspaceID, projectID)
		if err != nil {
			return err
		}
		if tag.RowsAffected() == 0 {
			return ErrNotFound
		}
	}
	if err := audit(ctx, tx, actorID, "status.reordered", map[string]any{
		"workspace_id": workspaceID, "project_id": projectID, "status_ids": statusIDs,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// DeleteProjectStatus refuses (ErrConflict) to delete the last status, or one still in use without a valid remapTo.
func DeleteProjectStatus(ctx context.Context, pool *pgxpool.Pool, actorID *uuid.UUID, workspaceID, projectID, statusID uuid.UUID, remapTo *uuid.UUID) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var name string
	if err := tx.QueryRow(ctx, "SELECT name FROM task_statuses WHERE id = $1 AND workspace_id = $2 AND project_id = $3",
		statusID, workspaceID, projectID).Scan(&name); err != nil {
		return mapErr(err)
	}
	var count, taskCount int64
	if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM task_statuses WHERE workspace_id = $1 AND project_id = $2", workspaceID, projectID).Scan(&count); err != nil {
		return err
	}
	if count <= 1 {
		return ErrConflict
	}
	if err := tx.QueryRow(ctx, "SELECT COUNT(*) FROM tasks WHERE status_id = $1 AND workspace_id = $2 AND project_id = $3",
		statusID, workspaceID, projectID).Scan(&taskCount); err != nil {
		return err
	}
	if taskCount > 0 {
		if remapTo == nil || *remapTo == statusID {
			return ErrConflict
		}
		var destID uuid.UUID
		var destCategory string
		if err := tx.QueryRow(ctx, "SELECT id, category FROM task_statuses WHERE id = $1 AND workspace_id = $2 AND project_id = $3",
			*remapTo, workspaceID, projectID).Scan(&destID, &destCategory); err != nil {
			return mapErr(err)
		}
		if _, err := tx.Exec(ctx, `UPDATE tasks SET status_id = $1, status = $2, updated_at = now()
			WHERE status_id = $3 AND workspace_id = $4 AND project_id = $5`, destID, destCategory, statusID, workspaceID, projectID); err != nil {
			return err
		}
	}
	if _, err := tx.Exec(ctx, "DELETE FROM task_statuses WHERE id = $1 AND workspace_id = $2 AND project_id = $3", statusID, workspaceID, projectID); err != nil {
		return err
	}
	if err := audit(ctx, tx, actorID, "status.deleted", map[string]any{
		"workspace_id": workspaceID, "project_id": projectID, "status_id": statusID, "name": name, "remap_to": remapTo,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
