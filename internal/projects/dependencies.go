package projects

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// A dependency edge means "task_id is blocked by depends_on_task_id". Both
// tasks share a workspace but may belong to different projects.
const (
	DependencyBlockedBy = "blocked_by"
	DependencyBlocks    = "blocks"
)

var (
	ErrDependencySelf   = errors.New("a task cannot depend on itself")
	ErrDependencyExists = errors.New("this dependency already exists")
	ErrDependencyCycle  = errors.New("this dependency would create a cycle")
)

// DependencyTask is the compact task summary shown in a dependency list.
type DependencyTask struct {
	ID              uuid.UUID
	WorkspaceID     uuid.UUID
	ProjectID       uuid.UUID
	ProjectName     string
	ProjectKey      string
	TaskNumber      int64
	HumanKey        string
	Title           string
	Status          string
	StatusID        *uuid.UUID
	StatusName      *string
	StatusColor     *string
	ProjectArchived bool
}

// DependencyLink is one edge as seen from a particular task. Task is the task
// at the other end of the edge.
type DependencyLink struct {
	ID        uuid.UUID
	Task      DependencyTask
	CreatedAt time.Time
}

// TaskDependencies groups a task's edges by direction. BlockedBy lists the
// tasks this task waits for; Blocks lists the tasks waiting for this one.
type TaskDependencies struct {
	BlockedBy []DependencyLink
	Blocks    []DependencyLink
}

const dependencyTaskColumns = `t.id, t.workspace_id, t.project_id, p.name, p.key, t.task_number,
	t.title, t.status, t.status_id, ts.name, ts.color, (p.archived_at IS NOT NULL)`

func scanDependencyLink(row pgx.Row) (DependencyLink, error) {
	var l DependencyLink
	err := row.Scan(&l.ID, &l.Task.ID, &l.Task.WorkspaceID, &l.Task.ProjectID,
		&l.Task.ProjectName, &l.Task.ProjectKey, &l.Task.TaskNumber, &l.Task.Title,
		&l.Task.Status, &l.Task.StatusID, &l.Task.StatusName, &l.Task.StatusColor,
		&l.Task.ProjectArchived, &l.CreatedAt)
	l.Task.HumanKey = humanKey(l.Task.ProjectKey, l.Task.TaskNumber)
	return l, err
}

func collectDependencyLinks(ctx context.Context, db Querier, sql string, args ...any) ([]DependencyLink, error) {
	rows, err := db.Query(ctx, sql, args...)
	if err != nil {
		return nil, err
	}
	links, err := pgx.CollectRows(rows, func(r pgx.CollectableRow) (DependencyLink, error) {
		return scanDependencyLink(r)
	})
	if links == nil {
		links = []DependencyLink{}
	}
	return links, err
}

// ListTaskDependencies returns both directions for one task. It returns
// ErrNotFound when the task is not in the project and workspace.
func ListTaskDependencies(ctx context.Context, db Querier, workspaceID, projectID, taskID uuid.UUID) (TaskDependencies, error) {
	var exists bool
	if err := db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM tasks
		WHERE id = $1 AND project_id = $2 AND workspace_id = $3)`,
		taskID, projectID, workspaceID).Scan(&exists); err != nil {
		return TaskDependencies{}, err
	}
	if !exists {
		return TaskDependencies{}, ErrNotFound
	}

	blockedBy, err := collectDependencyLinks(ctx, db, `SELECT d.id, `+dependencyTaskColumns+`, d.created_at
		FROM task_dependencies d
		JOIN tasks t ON t.id = d.depends_on_task_id
		JOIN projects p ON p.id = t.project_id
		LEFT JOIN task_statuses ts ON ts.id = t.status_id
		WHERE d.task_id = $1 AND d.workspace_id = $2
		ORDER BY p.key ASC, t.task_number ASC`, taskID, workspaceID)
	if err != nil {
		return TaskDependencies{}, err
	}
	blocks, err := collectDependencyLinks(ctx, db, `SELECT d.id, `+dependencyTaskColumns+`, d.created_at
		FROM task_dependencies d
		JOIN tasks t ON t.id = d.task_id
		JOIN projects p ON p.id = t.project_id
		LEFT JOIN task_statuses ts ON ts.id = t.status_id
		WHERE d.depends_on_task_id = $1 AND d.workspace_id = $2
		ORDER BY p.key ASC, t.task_number ASC`, taskID, workspaceID)
	if err != nil {
		return TaskDependencies{}, err
	}
	return TaskDependencies{BlockedBy: blockedBy, Blocks: blocks}, nil
}

func getDependencyTask(ctx context.Context, db Querier, workspaceID, taskID uuid.UUID) (DependencyTask, error) {
	var t DependencyTask
	err := db.QueryRow(ctx, `SELECT `+dependencyTaskColumns+`
		FROM tasks t
		JOIN projects p ON p.id = t.project_id
		LEFT JOIN task_statuses ts ON ts.id = t.status_id
		WHERE t.id = $1 AND t.workspace_id = $2`, taskID, workspaceID).
		Scan(&t.ID, &t.WorkspaceID, &t.ProjectID, &t.ProjectName, &t.ProjectKey,
			&t.TaskNumber, &t.Title, &t.Status, &t.StatusID, &t.StatusName,
			&t.StatusColor, &t.ProjectArchived)
	if errors.Is(err, pgx.ErrNoRows) {
		return DependencyTask{}, ErrNotFound
	}
	t.HumanKey = humanKey(t.ProjectKey, t.TaskNumber)
	return t, err
}

// AddTaskDependency creates one edge. The transaction takes a workspace-scoped
// advisory lock so two concurrent requests cannot both pass the cycle check.
func AddTaskDependency(ctx context.Context, pool *pgxpool.Pool, actorID *uuid.UUID,
	workspaceID, projectID, taskID, relatedTaskID uuid.UUID, direction string) (DependencyLink, error) {
	var blockedID, blockerID uuid.UUID
	switch direction {
	case DependencyBlockedBy:
		blockedID, blockerID = taskID, relatedTaskID
	case DependencyBlocks:
		blockedID, blockerID = relatedTaskID, taskID
	default:
		return DependencyLink{}, errors.New("invalid dependency direction")
	}
	if blockedID == blockerID {
		return DependencyLink{}, ErrDependencySelf
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return DependencyLink{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock(hashtextextended($1::text, 0))`,
		workspaceID.String()); err != nil {
		return DependencyLink{}, err
	}

	// The task in the URL must belong to this project; the related task may
	// live in another project of the same workspace.
	var taskProject, relatedProject uuid.UUID
	err = tx.QueryRow(ctx, `SELECT project_id FROM tasks WHERE id = $1 AND workspace_id = $2`,
		taskID, workspaceID).Scan(&taskProject)
	if errors.Is(err, pgx.ErrNoRows) {
		return DependencyLink{}, ErrNotFound
	}
	if err != nil {
		return DependencyLink{}, err
	}
	if taskProject != projectID {
		return DependencyLink{}, ErrNotFound
	}
	err = tx.QueryRow(ctx, `SELECT project_id FROM tasks WHERE id = $1 AND workspace_id = $2`,
		relatedTaskID, workspaceID).Scan(&relatedProject)
	if errors.Is(err, pgx.ErrNoRows) {
		return DependencyLink{}, ErrNotFound
	}
	if err != nil {
		return DependencyLink{}, err
	}

	var duplicate uuid.UUID
	err = tx.QueryRow(ctx, `SELECT id FROM task_dependencies
		WHERE task_id = $1 AND depends_on_task_id = $2`, blockedID, blockerID).Scan(&duplicate)
	switch {
	case err == nil:
		return DependencyLink{}, ErrDependencyExists
	case !errors.Is(err, pgx.ErrNoRows):
		return DependencyLink{}, err
	}

	// Adding blocked -> blocker is a cycle when the blocker already depends,
	// directly or transitively, on the blocked task.
	var cycle bool
	err = tx.QueryRow(ctx, `WITH RECURSIVE chain AS (
		SELECT depends_on_task_id FROM task_dependencies WHERE task_id = $1
		UNION
		SELECT d.depends_on_task_id FROM task_dependencies d
		JOIN chain c ON d.task_id = c.depends_on_task_id
	) SELECT EXISTS (SELECT 1 FROM chain WHERE depends_on_task_id = $2)`,
		blockerID, blockedID).Scan(&cycle)
	if err != nil {
		return DependencyLink{}, err
	}
	if cycle {
		return DependencyLink{}, ErrDependencyCycle
	}

	projectForBlocked := taskProject
	if direction == DependencyBlocks {
		projectForBlocked = relatedProject
	}
	id := uuid.New()
	var createdAt time.Time
	if err := tx.QueryRow(ctx, `INSERT INTO task_dependencies
		(id, workspace_id, project_id, task_id, depends_on_task_id, created_by)
		VALUES ($1, $2, $3, $4, $5, $6) RETURNING created_at`,
		id, workspaceID, projectForBlocked, blockedID, blockerID, actorID).Scan(&createdAt); err != nil {
		return DependencyLink{}, err
	}

	otherID := relatedTaskID
	other, err := getDependencyTask(ctx, tx, workspaceID, otherID)
	if err != nil {
		return DependencyLink{}, err
	}
	if err := RecordTaskActivity(ctx, tx, workspaceID, blockedID, actorID, "dependency.added",
		map[string]any{"task_key": other.HumanKey, "task_title": other.Title}); err != nil {
		return DependencyLink{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return DependencyLink{}, err
	}
	return DependencyLink{ID: id, Task: other, CreatedAt: createdAt}, nil
}

// RemoveTaskDependency deletes an edge touching the task, from either side.
func RemoveTaskDependency(ctx context.Context, pool *pgxpool.Pool, actorID *uuid.UUID,
	workspaceID, projectID, taskID, dependencyID uuid.UUID) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var exists bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM tasks
		WHERE id = $1 AND project_id = $2 AND workspace_id = $3)`,
		taskID, projectID, workspaceID).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}

	var blockedID, blockerID uuid.UUID
	err = tx.QueryRow(ctx, `SELECT task_id, depends_on_task_id FROM task_dependencies
		WHERE id = $1 AND workspace_id = $2 AND (task_id = $3 OR depends_on_task_id = $3)
		FOR UPDATE`, dependencyID, workspaceID, taskID).Scan(&blockedID, &blockerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM task_dependencies WHERE id = $1`, dependencyID); err != nil {
		return err
	}

	blocker, err := getDependencyTask(ctx, tx, workspaceID, blockerID)
	if err != nil {
		return err
	}
	if err := RecordTaskActivity(ctx, tx, workspaceID, blockedID, actorID, "dependency.removed",
		map[string]any{"task_key": blocker.HumanKey, "task_title": blocker.Title}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
