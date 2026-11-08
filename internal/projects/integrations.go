package projects

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Integration struct {
	Provider      string     `json:"provider"`
	Secret        string     `json:"-"`
	ChatID        string     `json:"chat_id"`
	TopicID       string     `json:"topic_id,omitempty"`
	LastAttemptAt *time.Time `json:"last_attempt_at"`
	LastError     string     `json:"last_error"`
}

func ListIntegrations(ctx context.Context, db Querier, ws, project uuid.UUID) ([]Integration, error) {
	rows, err := db.Query(ctx, `SELECT provider, secret, chat_id, topic_id, last_attempt_at, last_error
 FROM project_integrations WHERE workspace_id=$1 AND project_id=$2 ORDER BY provider`, ws, project)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (Integration, error) {
		var i Integration
		err := r.Scan(&i.Provider, &i.Secret, &i.ChatID, &i.TopicID, &i.LastAttemptAt, &i.LastError)
		return i, err
	})
}

func SaveIntegration(ctx context.Context, db Querier, ws, project uuid.UUID, i Integration) error {
	tag, err := db.Exec(ctx, `INSERT INTO project_integrations (workspace_id, project_id, provider, secret, chat_id, topic_id)
 SELECT workspace_id, id, $3, $4, $5, $6 FROM projects WHERE workspace_id=$1 AND id=$2 AND archived_at IS NULL
 ON CONFLICT (workspace_id, project_id, provider) DO UPDATE SET secret=EXCLUDED.secret,
 chat_id=EXCLUDED.chat_id, topic_id=EXCLUDED.topic_id, last_attempt_at=NULL, last_error=''`, ws, project, i.Provider, i.Secret, i.ChatID, i.TopicID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func DeleteIntegration(ctx context.Context, db Querier, ws, project uuid.UUID, provider string) error {
	_, err := db.Exec(ctx, `DELETE FROM project_integrations WHERE workspace_id=$1 AND project_id=$2 AND provider=$3`, ws, project, provider)
	return err
}

func RecordIntegrationAttempt(ctx context.Context, db Querier, ws, project uuid.UUID, i Integration, message string) error {
	tag, err := db.Exec(ctx, `UPDATE project_integrations SET last_attempt_at=now(), last_error=$6
 WHERE workspace_id=$1 AND project_id=$2 AND provider=$3 AND secret=$4 AND chat_id=$5`, ws, project, i.Provider, i.Secret, i.ChatID, message)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("integration changed during delivery")
	}
	return nil
}
