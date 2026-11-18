package projects

import (
	"context"
	"errors"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	// Embed the IANA database so time-log dates work in the slim runtime image
	// and in tests that do not have system zoneinfo.
	_ "time/tzdata"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// One active timer per user is enforced by a partial unique index. A stopped
// timer keeps the time log it produced so a retried stop is idempotent.
type Timer struct {
	PausedAt      *time.Time `json:"paused_at"`
	PausedSeconds float64    `json:"paused_seconds"`
	ID            uuid.UUID  `json:"id"`
	WorkspaceID   uuid.UUID  `json:"workspace_id"`
	ProjectID     uuid.UUID  `json:"project_id"`
	TaskID        uuid.UUID  `json:"task_id"`
	TaskKey       string     `json:"task_key"`
	TaskTitle     string     `json:"task_title"`
	Note          string     `json:"note"`
	StartedAt     time.Time  `json:"started_at"`
}

// TimerTask is the minimum task context needed before checking membership.
type TimerTask struct {
	TaskID          uuid.UUID
	WorkspaceID     uuid.UUID
	ProjectID       uuid.UUID
	HumanKey        string
	Title           string
	ProjectArchived bool
}

var (
	ErrTimerActive          = errors.New("a timer is already running")
	ErrTimerStopped         = errors.New("time is already logged for this timer")
	ErrTimerDiscarded       = errors.New("this timer was discarded")
	ErrTimerTooLong         = errors.New("timer has run longer than 240 hours")
	ErrTimerTaskUnavailable = errors.New("the project is archived")
)

const timerColumns = `SELECT tm.id, tm.workspace_id, tm.project_id, tm.task_id,
	p.key || '-' || t.task_number, t.title, tm.note, tm.started_at, tm.paused_at, tm.paused_seconds
	FROM task_timers tm
	JOIN tasks t ON t.id = tm.task_id
	JOIN projects p ON p.id = tm.project_id`

func scanTimer(row pgx.Row) (Timer, error) {
	var tm Timer
	err := row.Scan(&tm.ID, &tm.WorkspaceID, &tm.ProjectID, &tm.TaskID,
		&tm.TaskKey, &tm.TaskTitle, &tm.Note, &tm.StartedAt, &tm.PausedAt, &tm.PausedSeconds)
	return tm, err
}

// LookupTimerTask resolves a task before the HTTP layer checks membership.
func LookupTimerTask(ctx context.Context, db Querier, taskID uuid.UUID) (TimerTask, error) {
	var t TimerTask
	err := db.QueryRow(ctx, `SELECT t.id, t.workspace_id, t.project_id,
		p.key || '-' || t.task_number, t.title, p.archived_at IS NOT NULL
		FROM tasks t JOIN projects p ON p.id = t.project_id
		WHERE t.id = $1`, taskID).
		Scan(&t.TaskID, &t.WorkspaceID, &t.ProjectID, &t.HumanKey, &t.Title, &t.ProjectArchived)
	if errors.Is(err, pgx.ErrNoRows) {
		return TimerTask{}, ErrNotFound
	}
	return t, err
}

// GetActiveTimer returns nil when the user has no running timer.
func GetActiveTimer(ctx context.Context, db Querier, userID uuid.UUID) (*Timer, error) {
	tm, err := scanTimer(db.QueryRow(ctx, timerColumns+`
		WHERE tm.user_id = $1 AND tm.state = 'active'`, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &tm, nil
}

func getTimerByID(ctx context.Context, db Querier, userID, timerID uuid.UUID) (Timer, error) {
	tm, err := scanTimer(db.QueryRow(ctx, timerColumns+`
		WHERE tm.id = $1 AND tm.user_id = $2`, timerID, userID))
	if errors.Is(err, pgx.ErrNoRows) {
		return Timer{}, ErrNotFound
	}
	return tm, err
}

// StartTimer starts one timer for the user. Starting the same task again is
// idempotent. A different running timer is returned as ErrTimerActive unless
// replace is set, and the previous timer is then discarded without logging.
func StartTimer(ctx context.Context, pool *pgxpool.Pool, userID, workspaceID, taskID uuid.UUID, note string, replace bool) (Timer, error) {
	if utf8.RuneCountInString(note) > 2000 {
		return Timer{}, errors.New("timer note must be at most 2000 characters")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return Timer{}, err
	}
	defer tx.Rollback(ctx)

	existing, err := scanTimer(tx.QueryRow(ctx, timerColumns+`
		WHERE tm.user_id = $1 AND tm.state = 'active' FOR UPDATE`, userID))
	switch {
	case err == nil && existing.TaskID == taskID:
		if err := tx.Commit(ctx); err != nil {
			return Timer{}, err
		}
		return existing, nil
	case err == nil && !replace:
		return Timer{}, ErrTimerActive
	case err == nil:
		if _, err := tx.Exec(ctx, `UPDATE task_timers
			SET state = 'discarded', ended_at = now(), updated_at = now()
			WHERE id = $1`, existing.ID); err != nil {
			return Timer{}, err
		}
	case !errors.Is(err, pgx.ErrNoRows):
		return Timer{}, err
	}

	var projectID uuid.UUID
	var archived bool
	err = tx.QueryRow(ctx, `SELECT t.project_id, p.archived_at IS NOT NULL
		FROM tasks t JOIN projects p ON p.id = t.project_id
		WHERE t.id = $1 AND t.workspace_id = $2`, taskID, workspaceID).Scan(&projectID, &archived)
	if errors.Is(err, pgx.ErrNoRows) {
		return Timer{}, ErrNotFound
	}
	if err != nil {
		return Timer{}, err
	}
	if archived {
		return Timer{}, ErrTimerTaskUnavailable
	}

	now := time.Now().UTC()
	id := uuid.New()
	_, err = tx.Exec(ctx, `INSERT INTO task_timers
		(id, user_id, workspace_id, project_id, task_id, note, started_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $7, $7)`,
		id, userID, workspaceID, projectID, taskID, note, now)
	if err != nil {
		// A concurrent start won the unique active-timer index.
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			tx.Rollback(ctx)
			active, readErr := GetActiveTimer(ctx, pool, userID)
			if readErr != nil {
				return Timer{}, readErr
			}
			if active != nil && active.TaskID == taskID {
				return *active, nil
			}
			return Timer{}, ErrTimerActive
		}
		return Timer{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return Timer{}, err
	}
	return getTimerByID(ctx, pool, userID, id)
}

// UpdateTimerNote persists the note of a running timer.
func UpdateTimerNote(ctx context.Context, pool *pgxpool.Pool, userID, timerID uuid.UUID, note string) (Timer, error) {
	if utf8.RuneCountInString(note) > 2000 {
		return Timer{}, errors.New("timer note must be at most 2000 characters")
	}
	tag, err := pool.Exec(ctx, `UPDATE task_timers SET note = $3, updated_at = now()
		WHERE id = $1 AND user_id = $2 AND state = 'active'`, timerID, userID, note)
	if err != nil {
		return Timer{}, err
	}
	if tag.RowsAffected() == 0 {
		var state string
		err := pool.QueryRow(ctx, `SELECT state FROM task_timers WHERE id = $1 AND user_id = $2`, timerID, userID).Scan(&state)
		if errors.Is(err, pgx.ErrNoRows) {
			return Timer{}, ErrNotFound
		}
		if err != nil {
			return Timer{}, err
		}
		if state == "stopped" {
			return Timer{}, ErrTimerStopped
		}
		return Timer{}, ErrTimerDiscarded
	}
	return getTimerByID(ctx, pool, userID, timerID)
}

// StopTimer runs one transaction: it computes elapsed time on the server,
// writes the time log and marks the timer stopped. Repeating the call returns
// the original log instead of creating a duplicate.
func StopTimer(ctx context.Context, pool *pgxpool.Pool, userID, timerID uuid.UUID, note *string) (TimeLog, float64, error) {
	if note != nil && utf8.RuneCountInString(*note) > 2000 {
		return TimeLog{}, 0, errors.New("timer note must be at most 2000 characters")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return TimeLog{}, 0, err
	}
	defer tx.Rollback(ctx)

	var workspaceID, projectID, taskID uuid.UUID
	var state, storedNote string
	var startedAt time.Time
	var pausedAt *time.Time
	var pausedSeconds float64
	var logID *uuid.UUID
	err = tx.QueryRow(ctx, `SELECT workspace_id, project_id, task_id, state, note, started_at, time_log_id, paused_at, paused_seconds
		FROM task_timers WHERE id = $1 AND user_id = $2 FOR UPDATE`, timerID, userID).
		Scan(&workspaceID, &projectID, &taskID, &state, &storedNote, &startedAt, &logID, &pausedAt, &pausedSeconds)
	if errors.Is(err, pgx.ErrNoRows) {
		return TimeLog{}, 0, ErrNotFound
	}
	if err != nil {
		return TimeLog{}, 0, err
	}

	if state == "stopped" {
		if logID == nil {
			return TimeLog{}, 0, ErrTimerStopped
		}
		log, err := loadTimeLog(ctx, tx, *logID)
		if err != nil {
			return TimeLog{}, 0, err
		}
		total, err := GetTaskTotalSpentHours(ctx, tx, taskID)
		if err != nil {
			return TimeLog{}, 0, err
		}
		return log, total, nil
	}
	if state == "discarded" {
		return TimeLog{}, 0, ErrTimerDiscarded
	}

	if note != nil {
		storedNote = strings.TrimSpace(*note)
	}
	now := time.Now().UTC()
	end := now
	if pausedAt != nil {
		end = *pausedAt
	}
	hours := math.Round(math.Max(0, end.Sub(startedAt).Seconds()-pausedSeconds)/36) / 100
	if hours < 0.01 {
		hours = 0.01
	}
	if hours > 240 {
		return TimeLog{}, 0, ErrTimerTooLong
	}

	timezone := "UTC"
	if err := tx.QueryRow(ctx, `SELECT timezone FROM users WHERE id = $1`, userID).Scan(&timezone); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return TimeLog{}, 0, err
	}
	loc, err := time.LoadLocation(timezone)
	if err != nil {
		loc = time.UTC
	}

	log, err := LogTaskTime(ctx, tx, &userID, workspaceID, projectID, taskID, CreateTimeLogParams{
		Hours:       hours,
		LoggedDate:  startedAt.In(loc).Format(time.DateOnly),
		Description: storedNote,
	})
	if err != nil {
		return TimeLog{}, 0, err
	}
	if _, err := tx.Exec(ctx, `UPDATE task_timers
		SET state = 'stopped', ended_at = $2, time_log_id = $3, note = $4, updated_at = $2
		WHERE id = $1`, timerID, now, log.ID, storedNote); err != nil {
		return TimeLog{}, 0, err
	}
	total, err := GetTaskTotalSpentHours(ctx, tx, taskID)
	if err != nil {
		return TimeLog{}, 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return TimeLog{}, 0, err
	}
	return log, total, nil
}

func loadTimeLog(ctx context.Context, db Querier, id uuid.UUID) (TimeLog, error) {
	var l TimeLog
	var loggedDate time.Time
	err := db.QueryRow(ctx, `SELECT ttl.id, ttl.workspace_id, ttl.project_id, ttl.task_id,
		ttl.user_id, COALESCE(u.display_name, ''), ttl.hours, ttl.logged_date, ttl.description,
		ttl.created_at, ttl.updated_at
		FROM task_time_logs ttl LEFT JOIN users u ON u.id = ttl.user_id
		WHERE ttl.id = $1`, id).
		Scan(&l.ID, &l.WorkspaceID, &l.ProjectID, &l.TaskID, &l.UserID, &l.UserName,
			&l.Hours, &loggedDate, &l.Description, &l.CreatedAt, &l.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return TimeLog{}, ErrNotFound
	}
	l.LoggedDate = loggedDate.Format(time.DateOnly)
	return l, err
}

// DiscardTimer drops unlogged time. Discarding twice is a no-op; discarding a
// stopped timer is refused because its time is already recorded.
func DiscardTimer(ctx context.Context, pool *pgxpool.Pool, userID, timerID uuid.UUID) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	var state string
	err = tx.QueryRow(ctx, `SELECT state FROM task_timers WHERE id = $1 AND user_id = $2 FOR UPDATE`,
		timerID, userID).Scan(&state)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	switch state {
	case "active":
		if _, err := tx.Exec(ctx, `UPDATE task_timers
			SET state = 'discarded', ended_at = now(), updated_at = now() WHERE id = $1`, timerID); err != nil {
			return err
		}
	case "stopped":
		return ErrTimerStopped
	}
	return tx.Commit(ctx)
}

// SetTimerPaused uses an explicit desired state so retries never toggle twice.
func SetTimerPaused(ctx context.Context, pool *pgxpool.Pool, user, id uuid.UUID, paused bool) (Timer, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return Timer{}, err
	}
	defer tx.Rollback(ctx)
	var state string
	var wasPaused *time.Time
	err = tx.QueryRow(ctx, `SELECT tm.state,tm.paused_at FROM task_timers tm JOIN workspace_members m ON m.workspace_id=tm.workspace_id AND m.user_id=tm.user_id WHERE tm.id=$1 AND tm.user_id=$2 AND m.role<>'guest' FOR UPDATE OF tm`, id, user).Scan(&state, &wasPaused)
	if errors.Is(err, pgx.ErrNoRows) {
		return Timer{}, ErrNotFound
	}
	if err != nil {
		return Timer{}, err
	}
	if state != "active" {
		return Timer{}, ErrTimerStopped
	}
	if paused && wasPaused == nil {
		_, err = tx.Exec(ctx, "UPDATE task_timers SET paused_at=clock_timestamp(),updated_at=clock_timestamp() WHERE id=$1", id)
	}
	if !paused && wasPaused != nil {
		_, err = tx.Exec(ctx, "UPDATE task_timers SET paused_seconds=paused_seconds+GREATEST(0,EXTRACT(EPOCH FROM clock_timestamp()-paused_at)),paused_at=NULL,updated_at=clock_timestamp() WHERE id=$1", id)
	}
	if err != nil {
		return Timer{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Timer{}, err
	}
	return getTimerByID(ctx, pool, user, id)
}
