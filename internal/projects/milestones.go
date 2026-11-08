package projects

import (
	"context"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Milestone struct {
	ID             uuid.UUID
	WorkspaceID    uuid.UUID
	ProjectID      uuid.UUID
	Title          string
	Description    string
	StartDate      *time.Time
	DueDate        *time.Time
	Color          string
	Status         string // "open" | "closed"
	TotalTasks     int64
	CompletedTasks int64
	OpenTasks      int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CreateMilestoneParams struct {
	Title       string
	Description string
	StartDate   *time.Time
	DueDate     *time.Time
	Color       string
}

type UpdateMilestoneParams struct {
	Title        *string
	Description  *string
	StartDate    *time.Time
	SetStartDate bool
	DueDate      *time.Time
	SetDueDate   bool
	Status       *string
	Color        *string
}

func scanMilestone(row pgx.Row) (Milestone, error) {
	var m Milestone
	err := row.Scan(
		&m.ID, &m.WorkspaceID, &m.ProjectID, &m.Title, &m.Description,
		&m.StartDate, &m.DueDate, &m.Status, &m.Color, &m.TotalTasks, &m.CompletedTasks, &m.OpenTasks,
		&m.CreatedAt, &m.UpdatedAt,
	)
	return m, mapErr(err)
}

const milestoneColumns = `m.id, m.workspace_id, m.project_id, m.title, m.description,
	m.start_date, m.due_date, m.status, m.color,
	COALESCE(COUNT(t.id), 0) AS total_tasks,
	COALESCE(COUNT(CASE WHEN t.status = 'completed' THEN 1 END), 0) AS completed_tasks,
	COALESCE(COUNT(CASE WHEN t.status IN ('backlog', 'unstarted', 'started') THEN 1 END), 0) AS open_tasks,
	m.created_at, m.updated_at
	FROM milestones m
	LEFT JOIN tasks t ON t.milestone_id = m.id AND t.workspace_id = m.workspace_id`

func CreateMilestone(ctx context.Context, pool *pgxpool.Pool, actorID *uuid.UUID, workspaceID, projectID uuid.UUID, params CreateMilestoneParams) (Milestone, error) {
	if _, err := GetProject(ctx, pool, workspaceID, projectID); err != nil {
		return Milestone{}, err
	}
	now := time.Now().UTC()
	color := strings.TrimSpace(params.Color)
	if color == "" {
		color = "#6366f1"
	}
	m := Milestone{
		ID:          uuid.New(),
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		Title:       params.Title,
		Description: params.Description,
		StartDate:   params.StartDate,
		DueDate:     params.DueDate,
		Color:       color,
		Status:      "open",
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		return Milestone{}, err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `INSERT INTO milestones
		(id, workspace_id, project_id, title, description, start_date, due_date, status, color, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		m.ID, workspaceID, projectID, m.Title, m.Description, m.StartDate, m.DueDate, m.Status, m.Color, now, now); err != nil {
		return Milestone{}, mapErr(err)
	}

	if err := audit(ctx, tx, actorID, "milestone.created", map[string]any{
		"workspace_id": workspaceID, "project_id": projectID, "milestone_id": m.ID, "title": m.Title,
	}); err != nil {
		return Milestone{}, err
	}

	return m, tx.Commit(ctx)
}

func ListMilestones(ctx context.Context, db Querier, workspaceID, projectID uuid.UUID) ([]Milestone, error) {
	if _, err := GetProject(ctx, db, workspaceID, projectID); err != nil {
		return nil, err
	}
	rows, err := db.Query(ctx, `SELECT `+milestoneColumns+`
		WHERE m.workspace_id = $1 AND m.project_id = $2
		GROUP BY m.id
		ORDER BY CASE WHEN m.status = 'open' THEN 0 ELSE 1 END ASC, m.due_date ASC NULLS LAST, m.created_at DESC`,
		workspaceID, projectID)
	if err != nil {
		return nil, mapErr(err)
	}
	defer rows.Close()

	var list []Milestone
	for rows.Next() {
		m, err := scanMilestone(rows)
		if err != nil {
			return nil, err
		}
		list = append(list, m)
	}
	return list, rows.Err()
}

func GetMilestone(ctx context.Context, db Querier, workspaceID, projectID, milestoneID uuid.UUID) (Milestone, error) {
	row := db.QueryRow(ctx, `SELECT `+milestoneColumns+`
		WHERE m.id = $1 AND m.workspace_id = $2 AND m.project_id = $3
		GROUP BY m.id`,
		milestoneID, workspaceID, projectID)
	return scanMilestone(row)
}

func UpdateMilestone(ctx context.Context, pool *pgxpool.Pool, actorID *uuid.UUID, workspaceID, projectID, milestoneID uuid.UUID, params UpdateMilestoneParams) (Milestone, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return Milestone{}, err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `UPDATE milestones
		SET updated_at = $1,
		    title = COALESCE($2, title),
		    description = COALESCE($3, description),
		    start_date = CASE WHEN $4 THEN $5 ELSE start_date END,
		    due_date = CASE WHEN $6 THEN $7 ELSE due_date END,
		    status = COALESCE($8, status),
		    color = COALESCE($9, color)
		WHERE id = $10 AND workspace_id = $11 AND project_id = $12`,
		time.Now().UTC(), params.Title, params.Description, params.SetStartDate, params.StartDate, params.SetDueDate, params.DueDate, params.Status, params.Color,
		milestoneID, workspaceID, projectID)
	if err != nil {
		return Milestone{}, mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return Milestone{}, ErrNotFound
	}

	if err := audit(ctx, tx, actorID, "milestone.updated", map[string]any{
		"workspace_id": workspaceID, "project_id": projectID, "milestone_id": milestoneID,
	}); err != nil {
		return Milestone{}, err
	}

	if err := tx.Commit(ctx); err != nil {
		return Milestone{}, err
	}

	return GetMilestone(ctx, pool, workspaceID, projectID, milestoneID)
}

func DeleteMilestone(ctx context.Context, pool *pgxpool.Pool, actorID *uuid.UUID, workspaceID, projectID, milestoneID uuid.UUID) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	tag, err := tx.Exec(ctx, `DELETE FROM milestones WHERE id = $1 AND workspace_id = $2 AND project_id = $3`,
		milestoneID, workspaceID, projectID)
	if err != nil {
		return mapErr(err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}

	if err := audit(ctx, tx, actorID, "milestone.deleted", map[string]any{
		"workspace_id": workspaceID, "project_id": projectID, "milestone_id": milestoneID,
	}); err != nil {
		return err
	}

	return tx.Commit(ctx)
}

