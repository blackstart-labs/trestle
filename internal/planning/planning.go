// Package planning provides measurable goals and bounded task groupings without
// introducing a second task identity or a second tenant boundary.
package planning

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"math"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrProjectLinked = errors.New("Remove linked tasks before changing the Project")
var ErrConflict = errors.New("This plan changed. Reload the saved item before saving your draft.")
var ErrInvalid = errors.New("Choose active references from this Workspace. Objectives need a goal; key results need an objective; cycles and modules need a Project.")

type Item struct {
	ID          uuid.UUID  `json:"id"`
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	Kind        string     `json:"kind"`
	Title       string     `json:"title"`
	Description string     `json:"description"`
	Status      string     `json:"status"`
	ParentID    *uuid.UUID `json:"parent_id"`
	ProjectID   *uuid.UUID `json:"project_id"`
	OwnerID     *uuid.UUID `json:"owner_id"`
	StartDate   *string    `json:"start_date"`
	DueDate     *string    `json:"due_date"`
	Target      float64    `json:"target"`
	Actual      float64    `json:"actual"`
	Archived    bool       `json:"archived"`
	Version     int64      `json:"version"`
	UpdatedAt   time.Time  `json:"updated_at"`
	Total       int        `json:"total_tasks"`
	Completed   int        `json:"completed_tasks"`
	Cancelled   int        `json:"cancelled_tasks"`
}
type Input struct {
	Item
	CreationKey uuid.UUID `json:"creation_key"`
}

func Kind(k string) bool {
	return k == "goal" || k == "objective" || k == "key_result" || k == "cycle" || k == "module"
}
func Validate(in Input) error {
	if !Kind(in.Kind) || strings.TrimSpace(in.Title) == "" || utf8.RuneCountInString(in.Title) > 160 || utf8.RuneCountInString(in.Description) > 15000 {
		return errors.New("Choose a planning type, a title of 1–160 characters and a description of at most 15000 characters")
	}
	if !strings.Contains("|planned|active|at_risk|blocked|completed|cancelled|", "|"+in.Status+"|") || in.Status == "" {
		return errors.New("Choose a valid plan status")
	}
	if math.IsNaN(in.Target) || math.IsInf(in.Target, 0) || in.Target <= 0 || in.Target > 1e9 || math.IsNaN(in.Actual) || math.IsInf(in.Actual, 0) || in.Actual < 0 || in.Actual > 1e9 {
		return errors.New("Target must be positive; actual must be nonnegative; both must be at most one billion")
	}
	for _, d := range []*string{in.StartDate, in.DueDate} {
		if d != nil {
			if _, e := time.Parse(time.DateOnly, *d); e != nil {
				return errors.New("Use valid dates")
			}
		}
	}
	if in.StartDate != nil && in.DueDate != nil && *in.StartDate > *in.DueDate {
		return errors.New("Due date must not precede start date")
	}
	return nil
}

const cols = `i.id,i.workspace_id,i.kind,i.title,i.description,i.status,i.parent_id,i.project_id,i.owner_id,i.start_date::text,i.due_date::text,i.target,i.actual,i.archived,i.version,i.updated_at`
const counts = `,(SELECT count(*) FROM planning_tasks l JOIN tasks t ON t.id=l.task_id WHERE l.item_id=i.id),(SELECT count(*) FROM planning_tasks l JOIN tasks t ON t.id=l.task_id WHERE l.item_id=i.id AND t.status='completed'),(SELECT count(*) FROM planning_tasks l JOIN tasks t ON t.id=l.task_id WHERE l.item_id=i.id AND t.status='cancelled')`

func scan(row pgx.Row) (Item, error) {
	var i Item
	e := row.Scan(&i.ID, &i.WorkspaceID, &i.Kind, &i.Title, &i.Description, &i.Status, &i.ParentID, &i.ProjectID, &i.OwnerID, &i.StartDate, &i.DueDate, &i.Target, &i.Actual, &i.Archived, &i.Version, &i.UpdatedAt, &i.Total, &i.Completed, &i.Cancelled)
	return i, e
}
func Get(ctx context.Context, pool *pgxpool.Pool, ws, id uuid.UUID) (Item, error) {
	return scan(pool.QueryRow(ctx, "SELECT "+cols+counts+" FROM planning_items i WHERE i.workspace_id=$1 AND i.id=$2", ws, id))
}
func List(ctx context.Context, pool *pgxpool.Pool, ws uuid.UUID, kind, q string, archived bool, offset int) ([]Item, bool, error) {
	rows, e := pool.Query(ctx, "SELECT "+cols+counts+" FROM planning_items i WHERE i.workspace_id=$1 AND ($2='' OR i.kind=$2) AND i.archived=$3 AND ($4='' OR strpos(lower(i.title),lower($4))>0) ORDER BY i.updated_at DESC,i.id LIMIT 51 OFFSET $5", ws, kind, archived, q, offset)
	if e != nil {
		return nil, false, e
	}
	defer rows.Close()
	out := []Item{}
	for rows.Next() {
		i, e := scan(rows)
		if e != nil {
			return nil, false, e
		}
		out = append(out, i)
	}
	more := len(out) > 50
	if more {
		out = out[:50]
	}
	return out, more, rows.Err()
}
func Save(ctx context.Context, pool *pgxpool.Pool, actor, ws, id uuid.UUID, in Input, manager bool) (Item, error) {
	if e := Validate(in); e != nil {
		return Item{}, e
	}
	tx, e := pool.Begin(ctx)
	if e != nil {
		return Item{}, e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SELECT id FROM workspaces WHERE id=$1 FOR UPDATE", ws); e != nil {
		return Item{}, e
	}
	if id != uuid.Nil {
		var version int64
		var kind string
		var archived bool
		var project *uuid.UUID
		e = tx.QueryRow(ctx, "SELECT version,kind,archived,project_id FROM planning_items WHERE workspace_id=$1 AND id=$2 FOR UPDATE", ws, id).Scan(&version, &kind, &archived, &project)
		if e != nil {
			return Item{}, e
		}
		if version != in.Version || kind != in.Kind || (archived && in.Archived) {
			return Item{}, ErrConflict
		}
		if archived != in.Archived && !manager {
			return Item{}, ErrInvalid
		}
		if (project == nil) != (in.ProjectID == nil) || (project != nil && in.ProjectID != nil && *project != *in.ProjectID) {
			var linked bool
			if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM planning_tasks WHERE item_id=$1)", id).Scan(&linked); e != nil {
				return Item{}, e
			}
			if linked {
				return Item{}, ErrProjectLinked
			}
		}
	}
	if (in.Kind == "cycle" || in.Kind == "module") && in.ProjectID == nil {
		return Item{}, ErrInvalid
	}
	if in.ProjectID != nil {
		var exists bool
		e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM projects WHERE workspace_id=$1 AND id=$2 AND archived_at IS NULL)", ws, *in.ProjectID).Scan(&exists)
		if e != nil {
			return Item{}, e
		}
		if !exists {
			return Item{}, ErrInvalid
		}
	}
	if in.OwnerID != nil {
		var exists bool
		e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM workspace_members WHERE workspace_id=$1 AND user_id=$2)", ws, *in.OwnerID).Scan(&exists)
		if e != nil {
			return Item{}, e
		}
		if !exists {
			return Item{}, ErrInvalid
		}
	}
	expected := ""
	if in.Kind == "objective" {
		expected = "goal"
	}
	if in.Kind == "key_result" {
		expected = "objective"
	}
	if expected != "" {
		if in.ParentID == nil {
			return Item{}, ErrInvalid
		}
		var exists bool
		e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM planning_items WHERE workspace_id=$1 AND id=$2 AND kind=$3 AND NOT archived)", ws, *in.ParentID, expected).Scan(&exists)
		if e != nil {
			return Item{}, e
		}
		if !exists {
			return Item{}, ErrInvalid
		}
	} else if in.ParentID != nil {
		return Item{}, ErrInvalid
	}
	event := "plan.updated"
	if id == uuid.Nil {
		if in.CreationKey == uuid.Nil || in.Archived {
			return Item{}, ErrInvalid
		}
		event = "plan.created"
		e = tx.QueryRow(ctx, `INSERT INTO planning_items(workspace_id,creation_key,created_by,kind,title,description,status,parent_id,project_id,owner_id,start_date,due_date,target,actual) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) ON CONFLICT(creation_key) DO NOTHING RETURNING id`, ws, in.CreationKey, actor, in.Kind, strings.TrimSpace(in.Title), in.Description, in.Status, in.ParentID, in.ProjectID, in.OwnerID, in.StartDate, in.DueDate, in.Target, in.Actual).Scan(&id)
		if errors.Is(e, pgx.ErrNoRows) {
			e = tx.QueryRow(ctx, `SELECT id FROM planning_items WHERE workspace_id=$1 AND creation_key=$2 AND created_by=$3 AND version=1 AND kind=$4 AND title=$5 AND description=$6 AND status=$7 AND parent_id IS NOT DISTINCT FROM $8 AND project_id IS NOT DISTINCT FROM $9 AND owner_id IS NOT DISTINCT FROM $10 AND start_date IS NOT DISTINCT FROM $11::date AND due_date IS NOT DISTINCT FROM $12::date AND target=$13 AND actual=$14`, ws, in.CreationKey, actor, in.Kind, strings.TrimSpace(in.Title), in.Description, in.Status, in.ParentID, in.ProjectID, in.OwnerID, in.StartDate, in.DueDate, in.Target, in.Actual).Scan(&id)
			if errors.Is(e, pgx.ErrNoRows) {
				return Item{}, ErrConflict
			}
			if e != nil {
				return Item{}, e
			}
			if e = tx.Commit(ctx); e != nil {
				return Item{}, e
			}
			return Get(ctx, pool, ws, id)
		}
	} else {
		_, e = tx.Exec(ctx, `UPDATE planning_items SET title=$3,description=$4,status=$5,parent_id=$6,project_id=$7,owner_id=$8,start_date=$9,due_date=$10,target=$11,actual=$12,archived=$13,version=version+1,updated_at=now() WHERE workspace_id=$1 AND id=$2`, ws, id, strings.TrimSpace(in.Title), in.Description, in.Status, in.ParentID, in.ProjectID, in.OwnerID, in.StartDate, in.DueDate, in.Target, in.Actual, in.Archived)
	}
	if e != nil {
		return Item{}, e
	}
	meta, _ := json.Marshal(map[string]any{"workspace_id": ws, "plan_id": id, "kind": in.Kind, "status": in.Status, "archived": in.Archived})
	if _, e = tx.Exec(ctx, "INSERT INTO security_audit_events(actor_id,event_type,metadata) VALUES($1,$2,$3)", actor, event, meta); e != nil {
		return Item{}, e
	}
	if e = tx.Commit(ctx); e != nil {
		return Item{}, e
	}
	return Get(ctx, pool, ws, id)
}

type Task struct {
	ID        uuid.UUID `json:"id"`
	ProjectID uuid.UUID `json:"project_id"`
	Key       string    `json:"human_key"`
	Title     string    `json:"title"`
	Status    string    `json:"status"`
}

func Tasks(ctx context.Context, pool *pgxpool.Pool, ws, id uuid.UUID, offset int) ([]Task, bool, error) {
	rows, e := pool.Query(ctx, `SELECT t.id,t.project_id,p.key||'-'||t.task_number,t.title,t.status FROM planning_tasks l JOIN tasks t ON t.id=l.task_id JOIN projects p ON p.id=t.project_id WHERE l.workspace_id=$1 AND l.item_id=$2 ORDER BY t.task_number,t.id LIMIT 51 OFFSET $3`, ws, id, offset)
	if e != nil {
		return nil, false, e
	}
	defer rows.Close()
	out := []Task{}
	for rows.Next() {
		var t Task
		if e = rows.Scan(&t.ID, &t.ProjectID, &t.Key, &t.Title, &t.Status); e != nil {
			return nil, false, e
		}
		out = append(out, t)
	}
	more := len(out) > 50
	if more {
		out = out[:50]
	}
	return out, more, rows.Err()
}
func Link(ctx context.Context, pool *pgxpool.Pool, actor, ws, id, task uuid.UUID, remove bool) error {
	tx, e := pool.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	var kind string
	var project *uuid.UUID
	var archived bool
	e = tx.QueryRow(ctx, "SELECT kind,project_id,archived FROM planning_items WHERE workspace_id=$1 AND id=$2 FOR UPDATE", ws, id).Scan(&kind, &project, &archived)
	if e != nil {
		return e
	}
	if archived {
		return ErrConflict
	}
	var taskProject uuid.UUID
	e = tx.QueryRow(ctx, "SELECT t.project_id FROM tasks t JOIN projects p ON p.id=t.project_id WHERE t.workspace_id=$1 AND t.id=$2 AND p.archived_at IS NULL FOR UPDATE OF t", ws, task).Scan(&taskProject)
	if e != nil {
		return e
	}
	if !remove && project != nil && *project != taskProject {
		return ErrInvalid
	}
	if remove {
		_, e = tx.Exec(ctx, "DELETE FROM planning_tasks WHERE workspace_id=$1 AND item_id=$2 AND task_id=$3", ws, id, task)
	} else {
		if kind == "cycle" || kind == "module" {
			_, e = tx.Exec(ctx, "DELETE FROM planning_tasks WHERE workspace_id=$1 AND task_id=$2 AND kind=$3", ws, task, kind)
			if e != nil {
				return e
			}
		}
		_, e = tx.Exec(ctx, "INSERT INTO planning_tasks(workspace_id,item_id,task_id,kind) VALUES($1,$2,$3,$4) ON CONFLICT DO NOTHING", ws, id, task, kind)
	}
	if e != nil {
		return e
	}
	meta, _ := json.Marshal(map[string]any{"workspace_id": ws, "plan_id": id, "task_id": task, "removed": remove})
	if _, e = tx.Exec(ctx, "INSERT INTO security_audit_events(actor_id,event_type,metadata) VALUES($1,'plan.task_changed',$2)", actor, meta); e != nil {
		return e
	}
	return tx.Commit(ctx)
}
func Children(ctx context.Context, pool *pgxpool.Pool, ws, id uuid.UUID, offset int) ([]Item, bool, error) {
	rows, e := pool.Query(ctx, "SELECT "+cols+counts+" FROM planning_items i WHERE workspace_id=$1 AND parent_id=$2 AND NOT archived ORDER BY updated_at DESC,id LIMIT 51 OFFSET $3", ws, id, offset)
	if e != nil {
		return nil, false, e
	}
	defer rows.Close()
	out := []Item{}
	for rows.Next() {
		i, e := scan(rows)
		if e != nil {
			return nil, false, e
		}
		out = append(out, i)
	}
	more := len(out) > 50
	if more {
		out = out[:50]
	}
	return out, more, rows.Err()
}
func Measurement(ctx context.Context, pool *pgxpool.Pool, ws, id uuid.UUID) (*float64, error) {
	var progress *float64
	e := pool.QueryRow(ctx, `WITH RECURSIVE descendants AS (SELECT id,kind,target,actual FROM planning_items WHERE workspace_id=$1 AND id=$2 AND NOT archived UNION ALL SELECT c.id,c.kind,c.target,c.actual FROM planning_items c JOIN descendants p ON c.parent_id=p.id WHERE c.workspace_id=$1 AND NOT c.archived AND c.status<>'cancelled') SELECT avg(LEAST(100,actual*100/target))::double precision FROM descendants WHERE kind='key_result'`, ws, id).Scan(&progress)
	return progress, e
}

// Export includes archived plans and canonical task associations. The caller
// supplies its repeatable-read transaction so the entire Workspace is coherent.
type ExportItem struct {
	Item
	TaskIDs []uuid.UUID `json:"task_ids"`
}

func Export(ctx context.Context, tx pgx.Tx, ws uuid.UUID) ([]ExportItem, error) {
	rows, err := tx.Query(ctx, "SELECT "+cols+counts+" FROM planning_items i WHERE i.workspace_id=$1 ORDER BY i.created_at,i.id", ws)
	if err != nil {
		return nil, err
	}
	items := []ExportItem{}
	indices := map[uuid.UUID]int{}
	for rows.Next() {
		item, err := scan(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		indices[item.ID] = len(items)
		items = append(items, ExportItem{Item: item, TaskIDs: []uuid.UUID{}})
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	links, err := tx.Query(ctx, "SELECT item_id,task_id FROM planning_tasks WHERE workspace_id=$1 ORDER BY item_id,task_id", ws)
	if err != nil {
		return nil, err
	}
	defer links.Close()
	for links.Next() {
		var item, task uuid.UUID
		if err = links.Scan(&item, &task); err != nil {
			return nil, err
		}
		if index, ok := indices[item]; ok {
			items[index].TaskIDs = append(items[index].TaskIDs, task)
		}
	}
	return items, links.Err()
}
