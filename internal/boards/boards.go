// Package boards owns bounded, versioned diagram and whiteboard scenes.
package boards

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Point struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type Element struct {
	ID          string  `json:"id"`
	Kind        string  `json:"kind"`
	X           float64 `json:"x"`
	Y           float64 `json:"y"`
	Width       float64 `json:"width"`
	Height      float64 `json:"height"`
	Text        string  `json:"text"`
	Color       string  `json:"color"`
	From        string  `json:"from,omitempty"`
	To          string  `json:"to,omitempty"`
	Points      []Point `json:"points,omitempty"`
	Fill        string  `json:"fill,omitempty"`
	StrokeWidth float64 `json:"strokeWidth,omitempty"`
	Dashed      bool    `json:"dashed,omitempty"`
	FontSize    float64 `json:"fontSize,omitempty"`
	Bold        bool    `json:"bold,omitempty"`
	Align       string  `json:"align,omitempty"`
	Route       string  `json:"route,omitempty"`
	FromPort    string  `json:"fromPort,omitempty"`
	ToPort      string  `json:"toPort,omitempty"`
	Heads       string  `json:"heads,omitempty"`
	Group       string  `json:"group,omitempty"`
	Locked      bool    `json:"locked,omitempty"`
}

type Scene struct {
	Version  int       `json:"version"`
	Elements []Element `json:"elements"`
}

type Board struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Title       string    `json:"title"`
	Kind        string    `json:"kind"`
	Scene       *Scene    `json:"scene,omitempty"`
	Version     int64     `json:"version"`
	Archived    bool      `json:"archived"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

var ErrConflict = errors.New("board changed since it was opened")

func coordinate(n float64) bool { return !math.IsNaN(n) && !math.IsInf(n, 0) && n >= 0 && n <= 4000 }

// Validate is also the import boundary: no raw SVG, URLs, HTML or arbitrary styles.
func Validate(title string, scene Scene) error {
	if strings.TrimSpace(title) == "" || utf8.RuneCountInString(title) > 120 {
		return errors.New("Enter a board title of 1–120 characters")
	}
	if scene.Version != 1 || scene.Elements == nil || len(scene.Elements) > 500 {
		return errors.New("Use scene version 1 with at most 500 objects")
	}
	ids := map[string]string{}
	points := 0
	for _, e := range scene.Elements {
		if e.ID == "" || len(e.ID) > 64 || ids[e.ID] != "" {
			return errors.New("Object IDs must be unique and 1–64 bytes")
		}
		ids[e.ID] = e.Kind
		switch e.Kind {
		case "rectangle", "ellipse", "diamond", "text", "stroke", "arrow", "terminator", "document", "database", "parallelogram", "note", "frame", "swimlane":
		default:
			return errors.New("Unknown object type")
		}
		switch e.Color {
		case "ink", "accent", "success", "warning", "danger", "info":
		default:
			return errors.New("Unknown object color")
		}
		if !coordinate(e.X) || !coordinate(e.Y) || !coordinate(e.Width) || !coordinate(e.Height) || e.X+e.Width > 4000 || e.Y+e.Height > 4000 || utf8.RuneCountInString(e.Text) > 200 {
			return errors.New("Objects must fit within 4000 × 4000 with labels up to 200 characters")
		}
		if e.Kind != "arrow" && e.Kind != "stroke" && (e.Width < 20 || e.Height < 20) {
			return errors.New("Shapes must be at least 20 × 20")
		}
		if e.Kind != "stroke" && len(e.Points) != 0 {
			return errors.New("Only strokes can contain points")
		}
		if e.Kind != "arrow" && (e.From != "" || e.To != "") {
			return errors.New("Only arrows can connect objects")
		}
		if e.Kind == "stroke" && len(e.Points) < 2 {
			return errors.New("A stroke needs at least two points")
		}
		points += len(e.Points)
		if e.Fill != "" && e.Fill != "none" && e.Fill != "ink" && e.Fill != "accent" && e.Fill != "success" && e.Fill != "warning" && e.Fill != "danger" && e.Fill != "info" {
			return errors.New("Unknown shape fill")
		}
		if (e.StrokeWidth != 0 && (e.StrokeWidth < 1 || e.StrokeWidth > 8 || math.IsNaN(e.StrokeWidth) || math.IsInf(e.StrokeWidth, 0))) || (e.FontSize != 0 && (e.FontSize < 12 || e.FontSize > 72 || math.IsNaN(e.FontSize) || math.IsInf(e.FontSize, 0))) {
			return errors.New("Use line widths of 1–8 and text sizes of 12–72")
		}
		if e.Align != "" && e.Align != "left" && e.Align != "center" && e.Align != "right" {
			return errors.New("Unknown text alignment")
		}
		if e.Route != "" && e.Route != "straight" && e.Route != "elbow" && e.Route != "curve" {
			return errors.New("Unknown connector routing")
		}
		for _, port := range []string{e.FromPort, e.ToPort} {
			if port != "" && port != "auto" && port != "top" && port != "right" && port != "bottom" && port != "left" {
				return errors.New("Unknown connector port")
			}
		}
		if e.Heads != "" && e.Heads != "end" && e.Heads != "both" && e.Heads != "none" {
			return errors.New("Unknown connector arrowhead")
		}
		if len(e.Group) > 64 {
			return errors.New("Group IDs must be at most 64 bytes")
		}
		for _, p := range e.Points {
			if !coordinate(p.X) || !coordinate(p.Y) {
				return errors.New("Stroke points must fit within 4000 × 4000")
			}
		}
	}
	if points > 20000 {
		return errors.New("A board can contain at most 20000 stroke points")
	}
	for _, e := range scene.Elements {
		if e.Kind == "arrow" && (e.From == e.To || ids[e.From] == "" || ids[e.To] == "" || ids[e.From] == "arrow" || ids[e.To] == "arrow" || ids[e.From] == "stroke" || ids[e.To] == "stroke") {
			return errors.New("Connect two different shapes with each arrow")
		}
	}
	return nil
}

const columns = "id, workspace_id, title, kind, version, archived, created_at, updated_at"

func scan(row pgx.Row, withScene bool) (Board, error) {
	var b Board
	args := []any{&b.ID, &b.WorkspaceID, &b.Title, &b.Kind, &b.Version, &b.Archived, &b.CreatedAt, &b.UpdatedAt}
	if withScene {
		b.Scene = &Scene{}
		args = append(args, b.Scene)
	}
	err := row.Scan(args...)
	return b, err
}

func List(ctx context.Context, pool *pgxpool.Pool, ws uuid.UUID, archived bool, offset int) ([]Board, bool, error) {
	rows, err := pool.Query(ctx, "SELECT "+columns+" FROM boards WHERE workspace_id=$1 AND archived=$2 ORDER BY updated_at DESC, id LIMIT 51 OFFSET $3", ws, archived, offset)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	list := []Board{}
	for rows.Next() {
		b, err := scan(rows, false)
		if err != nil {
			return nil, false, err
		}
		list = append(list, b)
	}
	more := len(list) > 50
	if more {
		list = list[:50]
	}
	return list, more, rows.Err()
}

func Get(ctx context.Context, pool *pgxpool.Pool, ws, id uuid.UUID) (Board, error) {
	return scan(pool.QueryRow(ctx, "SELECT "+columns+", scene FROM boards WHERE workspace_id=$1 AND id=$2", ws, id), true)
}

// Export participates in the Workspace export's repeatable-read snapshot.
func Export(ctx context.Context, db interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, ws uuid.UUID) ([]Board, error) {
	// shortcut: snapshots buffer scenes; stream when collections outgrow small-team memory.
	rows, err := db.Query(ctx, "SELECT "+columns+", scene FROM boards WHERE workspace_id=$1 ORDER BY id", ws)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	list := []Board{}
	for rows.Next() {
		b, err := scan(rows, true)
		if err != nil {
			return nil, err
		}
		list = append(list, b)
	}
	return list, rows.Err()
}

// Save keeps the mutation and audit atomic. A version precondition prevents lost edits.
func Save(ctx context.Context, pool *pgxpool.Pool, actor, ws, id, creationKey uuid.UUID, title, kind string, scene Scene, version int64, archived bool) (Board, error) {
	if err := Validate(title, scene); err != nil {
		return Board{}, err
	}
	if kind != "diagram" && kind != "whiteboard" {
		return Board{}, errors.New("Choose diagram or whiteboard")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return Board{}, err
	}
	defer tx.Rollback(ctx)
	var b Board
	event := "board.updated"
	if id == uuid.Nil {
		event = "board.created"
		if creationKey == uuid.Nil {
			return Board{}, errors.New("A creation key is required")
		}
		b, err = scan(tx.QueryRow(ctx, "INSERT INTO boards (workspace_id,title,kind,scene,created_by,creation_key) VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (creation_key) DO NOTHING RETURNING "+columns+", scene", ws, strings.TrimSpace(title), kind, scene, actor, creationKey), true)
		if errors.Is(err, pgx.ErrNoRows) {
			b, err = scan(tx.QueryRow(ctx, "SELECT "+columns+", scene FROM boards WHERE creation_key=$1 AND workspace_id=$2 AND created_by=$3 AND title=$4 AND kind=$5 AND scene=$6 AND version=1", creationKey, ws, actor, strings.TrimSpace(title), kind, scene), true)
			if errors.Is(err, pgx.ErrNoRows) {
				return Board{}, ErrConflict
			}
			if err != nil {
				return Board{}, err
			}
			return b, tx.Commit(ctx)
		}
	} else {
		var current Board
		current, err = scan(tx.QueryRow(ctx, "SELECT "+columns+" FROM boards WHERE workspace_id=$1 AND id=$2 FOR UPDATE", ws, id), false)
		if err != nil {
			return Board{}, err
		}
		if current.Version != version {
			return Board{}, ErrConflict
		}
		if current.Kind != kind {
			return Board{}, errors.New("Board type cannot change")
		}
		b, err = scan(tx.QueryRow(ctx, "UPDATE boards SET title=$3,scene=$4,archived=$5,version=version+1,updated_at=now() WHERE workspace_id=$1 AND id=$2 RETURNING "+columns+", scene", ws, id, strings.TrimSpace(title), scene, archived), true)
	}
	if err != nil {
		return Board{}, err
	}
	metadata, err := json.Marshal(map[string]any{"workspace_id": ws, "board_id": b.ID, "version": b.Version, "archived": b.Archived})
	if err != nil {
		return Board{}, err
	}
	if _, err = tx.Exec(ctx, "INSERT INTO security_audit_events (actor_id,event_type,metadata) VALUES ($1,$2,$3)", actor, event, metadata); err != nil {
		return Board{}, fmt.Errorf("record board change: %w", err)
	}
	return b, tx.Commit(ctx)
}
