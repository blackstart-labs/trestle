package projects

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type TimeLog struct {
	ID          uuid.UUID  `json:"id"`
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	ProjectID   uuid.UUID  `json:"project_id"`
	TaskID      uuid.UUID  `json:"task_id"`
	UserID      *uuid.UUID `json:"user_id"`
	UserName    string     `json:"user_name"`
	Hours       float64    `json:"hours"`
	LoggedDate  string     `json:"logged_date"`
	Description string     `json:"description"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

type CreateTimeLogParams struct {
	Hours       float64
	LoggedDate  string
	Description string
}

func LogTaskTime(ctx context.Context, db Querier, actorID *uuid.UUID, workspaceID, projectID, taskID uuid.UUID, params CreateTimeLogParams) (TimeLog, error) {
	if params.Hours <= 0 {
		return TimeLog{}, errors.New("Hours must be greater than zero")
	}
	var loggedDate time.Time
	if params.LoggedDate != "" {
		d, err := time.Parse(time.DateOnly, params.LoggedDate)
		if err != nil {
			return TimeLog{}, errors.New("Enter a valid date in YYYY-MM-DD format")
		}
		loggedDate = d
	} else {
		loggedDate = time.Now().UTC()
	}

	var exists bool
	if err := db.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM tasks WHERE id = $1 AND project_id = $2 AND workspace_id = $3)",
		taskID, projectID, workspaceID).Scan(&exists); err != nil {
		return TimeLog{}, err
	}
	if !exists {
		return TimeLog{}, ErrNotFound
	}

	id := uuid.New()
	now := time.Now().UTC()
	var userName string
	if actorID != nil {
		_ = db.QueryRow(ctx, "SELECT display_name FROM users WHERE id = $1", *actorID).Scan(&userName)
	}

	_, err := db.Exec(ctx, `INSERT INTO task_time_logs
		(id, workspace_id, project_id, task_id, user_id, hours, logged_date, description, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $9)`,
		id, workspaceID, projectID, taskID, actorID, params.Hours, loggedDate.Format(time.DateOnly), params.Description, now)
	if err != nil {
		return TimeLog{}, err
	}

	return TimeLog{
		ID:          id,
		WorkspaceID: workspaceID,
		ProjectID:   projectID,
		TaskID:      taskID,
		UserID:      actorID,
		UserName:    userName,
		Hours:       params.Hours,
		LoggedDate:  loggedDate.Format(time.DateOnly),
		Description: params.Description,
		CreatedAt:   now,
		UpdatedAt:   now,
	}, nil
}

func ListTaskTimeLogs(ctx context.Context, db Querier, workspaceID, projectID, taskID uuid.UUID) ([]TimeLog, error) {
	rows, err := db.Query(ctx, `SELECT ttl.id, ttl.workspace_id, ttl.project_id, ttl.task_id,
		ttl.user_id, COALESCE(u.display_name, ''), ttl.hours, ttl.logged_date, ttl.description, ttl.created_at, ttl.updated_at
		FROM task_time_logs ttl
		LEFT JOIN users u ON u.id = ttl.user_id
		WHERE ttl.workspace_id = $1 AND ttl.project_id = $2 AND ttl.task_id = $3
		ORDER BY ttl.logged_date DESC, ttl.created_at DESC`, workspaceID, projectID, taskID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []TimeLog
	for rows.Next() {
		var l TimeLog
		var lDate time.Time
		if err := rows.Scan(&l.ID, &l.WorkspaceID, &l.ProjectID, &l.TaskID,
			&l.UserID, &l.UserName, &l.Hours, &lDate, &l.Description, &l.CreatedAt, &l.UpdatedAt); err != nil {
			return nil, err
		}
		l.LoggedDate = lDate.Format(time.DateOnly)
		logs = append(logs, l)
	}
	return logs, rows.Err()
}

func DeleteTaskTimeLog(ctx context.Context, db Querier, actorID *uuid.UUID, workspaceID, projectID, taskID, logID uuid.UUID) error {
	tag, err := db.Exec(ctx, `DELETE FROM task_time_logs
		WHERE id = $1 AND workspace_id = $2 AND project_id = $3 AND task_id = $4`,
		logID, workspaceID, projectID, taskID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func GetTaskTotalSpentHours(ctx context.Context, db Querier, taskID uuid.UUID) (float64, error) {
	var total float64
	err := db.QueryRow(ctx, "SELECT COALESCE(SUM(hours), 0) FROM task_time_logs WHERE task_id = $1", taskID).Scan(&total)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return 0, err
	}
	return total, nil
}
