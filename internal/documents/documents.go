// Package documents stores portable Markdown pages and immutable revision snapshots.
package documents

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrConflict = errors.New("This document changed. Export your draft, then reload before saving.")
var ErrInvalid = errors.New("Choose an active parent in this Workspace; a page cannot contain itself or an ancestor.")

type Document struct {
	ID          uuid.UUID  `json:"id"`
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	Title       string     `json:"title"`
	Content     string     `json:"content"`
	Tags        []string   `json:"tags"`
	ParentID    *uuid.UUID `json:"parent_id"`
	Version     int64      `json:"version"`
	Archived    bool       `json:"archived"`
	UpdatedAt   time.Time  `json:"updated_at"`
}
type Input struct {
	Title       string     `json:"title"`
	Content     string     `json:"content"`
	Tags        []string   `json:"tags"`
	ParentID    *uuid.UUID `json:"parent_id"`
	Version     int64      `json:"version"`
	Archived    bool       `json:"archived"`
	CreationKey uuid.UUID  `json:"creation_key"`
}

func Validate(in Input) error {
	if strings.TrimSpace(in.Title) == "" || utf8.RuneCountInString(in.Title) > 160 || strings.ContainsRune(in.Title, 0) {
		return errors.New("Enter a title of 1–160 characters")
	}
	if !utf8.ValidString(in.Content) || utf8.RuneCountInString(in.Content) > 100000 || strings.ContainsRune(in.Content, 0) {
		return errors.New("Use at most 100000 characters of valid Markdown")
	}
	if len(in.Tags) > 20 {
		return errors.New("Use at most 20 tags")
	}
	seen := map[string]bool{}
	for _, tag := range in.Tags {
		if tag != strings.TrimSpace(tag) || tag == "" || utf8.RuneCountInString(tag) > 40 || strings.ContainsRune(tag, 0) || seen[tag] {
			return errors.New("Tags must be unique, nonempty and at most 40 characters")
		}
		seen[tag] = true
	}
	return nil
}

const columns = "id,workspace_id,title,content,tags,parent_id,version,archived,updated_at"

func scan(row pgx.Row) (Document, error) {
	var d Document
	err := row.Scan(&d.ID, &d.WorkspaceID, &d.Title, &d.Content, &d.Tags, &d.ParentID, &d.Version, &d.Archived, &d.UpdatedAt)
	return d, err
}
func Get(ctx context.Context, pool *pgxpool.Pool, ws, id uuid.UUID) (Document, error) {
	return scan(pool.QueryRow(ctx, "SELECT "+columns+" FROM documents WHERE workspace_id=$1 AND id=$2", ws, id))
}
func List(ctx context.Context, pool *pgxpool.Pool, ws uuid.UUID, archived bool, q, tag string, parent *uuid.UUID, offset int) ([]Document, bool, error) {
	rows, err := pool.Query(ctx, `SELECT id,workspace_id,title,'' AS content,tags,parent_id,version,archived,updated_at FROM documents WHERE workspace_id=$1 AND archived=$2 AND ($3='' OR strpos(lower(title),lower($3))>0 OR strpos(lower(content),lower($3))>0) AND ($4='' OR $4=ANY(tags)) AND ($6::uuid IS NULL OR parent_id=$6) ORDER BY updated_at DESC,id LIMIT 51 OFFSET $5`, ws, archived, q, tag, offset, parent)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	out := []Document{}
	for rows.Next() {
		d, e := scan(rows)
		if e != nil {
			return nil, false, e
		}
		out = append(out, d)
	}
	more := len(out) > 50
	if more {
		out = out[:50]
	}
	return out, more, rows.Err()
}

// Links are stable UUID references, so renaming never breaks a connection.
var wiki = regexp.MustCompile(`\[\[([0-9a-fA-F-]{36})(?:\|[^\]\r\n]{0,160})?\]\]`)

func Links(content string) []uuid.UUID {
	out := []uuid.UUID{}
	seen := map[uuid.UUID]bool{}
	for _, m := range wiki.FindAllStringSubmatch(content, -1) {
		id, e := uuid.Parse(m[1])
		if e == nil && !seen[id] {
			seen[id] = true
			out = append(out, id)
		}
	}
	return out
}
func Save(ctx context.Context, pool *pgxpool.Pool, actor, ws, id uuid.UUID, in Input, manager bool) (Document, error) {
	if err := Validate(in); err != nil {
		return Document{}, err
	}
	if in.Tags == nil {
		in.Tags = []string{}
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return Document{}, err
	}
	defer tx.Rollback(ctx)
	// Serialize hierarchy changes within a Workspace to prevent concurrent reparent cycles.
	if _, err = tx.Exec(ctx, "SELECT id FROM workspaces WHERE id=$1 FOR UPDATE", ws); err != nil {
		return Document{}, err
	}
	var old Document
	if id != uuid.Nil {
		old, err = scan(tx.QueryRow(ctx, "SELECT "+columns+" FROM documents WHERE workspace_id=$1 AND id=$2 FOR UPDATE", ws, id))
		if err != nil {
			return Document{}, err
		}
		if old.Version != in.Version {
			return Document{}, ErrConflict
		}
		if old.Archived != in.Archived && !manager {
			return Document{}, ErrInvalid
		}
		if old.Archived && in.Archived {
			return Document{}, ErrConflict
		}
	}
	if in.ParentID != nil {
		var valid bool
		err = tx.QueryRow(ctx, `WITH RECURSIVE ancestors AS (SELECT id,parent_id FROM documents WHERE workspace_id=$1 AND id=$2 AND NOT archived UNION SELECT d.id,d.parent_id FROM documents d JOIN ancestors a ON d.id=a.parent_id WHERE d.workspace_id=$1) SELECT EXISTS(SELECT 1 FROM ancestors) AND NOT EXISTS(SELECT 1 FROM ancestors WHERE id=$3)`, ws, *in.ParentID, id).Scan(&valid)
		if err != nil {
			return Document{}, err
		}
		if !valid {
			return Document{}, ErrInvalid
		}
	}
	event := "document.updated"
	var d Document
	if id == uuid.Nil {
		if in.CreationKey == uuid.Nil || in.Archived {
			return Document{}, ErrInvalid
		}
		event = "document.created"
		d, err = scan(tx.QueryRow(ctx, "INSERT INTO documents(workspace_id,creation_key,created_by,title,content,tags,parent_id) VALUES($1,$2,$3,$4,$5,$6,$7) ON CONFLICT(creation_key) DO NOTHING RETURNING "+columns, ws, in.CreationKey, actor, strings.TrimSpace(in.Title), in.Content, in.Tags, in.ParentID))
		if errors.Is(err, pgx.ErrNoRows) {
			d, err = scan(tx.QueryRow(ctx, "SELECT "+columns+" FROM documents WHERE workspace_id=$1 AND creation_key=$2 AND created_by=$3 AND version=1 AND title=$4 AND content=$5 AND tags=$6 AND parent_id IS NOT DISTINCT FROM $7", ws, in.CreationKey, actor, strings.TrimSpace(in.Title), in.Content, in.Tags, in.ParentID))
			if errors.Is(err, pgx.ErrNoRows) {
				return Document{}, ErrConflict
			}
			if err != nil {
				return Document{}, err
			}
			return d, tx.Commit(ctx)
		}
	} else {
		d, err = scan(tx.QueryRow(ctx, "UPDATE documents SET title=$3,content=$4,tags=$5,parent_id=$6,archived=$7,version=version+1,updated_at=now() WHERE workspace_id=$1 AND id=$2 RETURNING "+columns, ws, id, strings.TrimSpace(in.Title), in.Content, in.Tags, in.ParentID, in.Archived))
	}
	if err != nil {
		return Document{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO document_versions(workspace_id,document_id,version,title,content,tags,parent_id,archived,actor_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9)`, ws, d.ID, d.Version, d.Title, d.Content, d.Tags, d.ParentID, d.Archived, actor); err != nil {
		return Document{}, err
	}
	if _, err = tx.Exec(ctx, "DELETE FROM document_links WHERE workspace_id=$1 AND source_id=$2", ws, d.ID); err != nil {
		return Document{}, err
	}
	for _, target := range Links(d.Content) {
		if _, err = tx.Exec(ctx, `INSERT INTO document_links(workspace_id,source_id,target_id) SELECT $1,$2,id FROM documents WHERE workspace_id=$1 AND id=$3 ON CONFLICT DO NOTHING`, ws, d.ID, target); err != nil {
			return Document{}, err
		}
	}
	metadata, _ := json.Marshal(map[string]any{"workspace_id": ws, "document_id": d.ID, "version": d.Version, "archived": d.Archived})
	if _, err = tx.Exec(ctx, "INSERT INTO security_audit_events(actor_id,event_type,metadata) VALUES($1,$2,$3)", actor, event, metadata); err != nil {
		return Document{}, err
	}
	return d, tx.Commit(ctx)
}
func History(ctx context.Context, pool *pgxpool.Pool, ws, id uuid.UUID, offset int) ([]Document, bool, error) {
	rows, err := pool.Query(ctx, `SELECT document_id,workspace_id,title,'' AS content,tags,parent_id,version,archived,created_at FROM document_versions WHERE workspace_id=$1 AND document_id=$2 ORDER BY version DESC LIMIT 51 OFFSET $3`, ws, id, offset)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	out := []Document{}
	for rows.Next() {
		d, e := scan(rows)
		if e != nil {
			return nil, false, e
		}
		out = append(out, d)
	}
	more := len(out) > 50
	if more {
		out = out[:50]
	}
	return out, more, rows.Err()
}
func Revision(ctx context.Context, pool *pgxpool.Pool, ws, id uuid.UUID, version int64) (Document, error) {
	return scan(pool.QueryRow(ctx, `SELECT document_id,workspace_id,title,content,tags,parent_id,version,archived,created_at FROM document_versions WHERE workspace_id=$1 AND document_id=$2 AND version=$3`, ws, id, version))
}
func Backlinks(ctx context.Context, pool *pgxpool.Pool, ws, id uuid.UUID, offset int) ([]Document, bool, error) {
	rows, err := pool.Query(ctx, `SELECT d.id,d.workspace_id,d.title,'' AS content,d.tags,d.parent_id,d.version,d.archived,d.updated_at FROM documents d JOIN document_links l ON l.source_id=d.id AND l.workspace_id=d.workspace_id WHERE l.workspace_id=$1 AND l.target_id=$2 AND NOT d.archived ORDER BY d.updated_at DESC,d.id LIMIT 51 OFFSET $3`, ws, id, offset)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	out := []Document{}
	for rows.Next() {
		d, e := scan(rows)
		if e != nil {
			return nil, false, e
		}
		out = append(out, d)
	}
	more := len(out) > 50
	if more {
		out = out[:50]
	}
	return out, more, rows.Err()
}
func Export(ctx context.Context, db interface {
	Query(context.Context, string, ...any) (pgx.Rows, error)
}, ws uuid.UUID) ([]Document, error) {
	rows, err := db.Query(ctx, "SELECT "+columns+" FROM documents WHERE workspace_id=$1 ORDER BY id", ws)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []Document{}
	for rows.Next() {
		d, e := scan(rows)
		if e != nil {
			return nil, e
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

type Edge struct {
	Source uuid.UUID `json:"source"`
	Target uuid.UUID `json:"target"`
}

func Graph(ctx context.Context, pool *pgxpool.Pool, ws uuid.UUID, q, tag string) ([]Document, []Edge, bool, error) {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if err != nil {
		return nil, nil, false, err
	}
	defer tx.Rollback(ctx)
	rows, err := tx.Query(ctx, `SELECT id,workspace_id,title,'' AS content,tags,parent_id,version,archived,updated_at FROM documents WHERE workspace_id=$1 AND NOT archived AND ($2='' OR strpos(lower(title),lower($2))>0) AND ($3='' OR $3=ANY(tags)) ORDER BY updated_at DESC,id LIMIT 201`, ws, q, tag)
	if err != nil {
		return nil, nil, false, err
	}
	nodes := []Document{}
	for rows.Next() {
		d, e := scan(rows)
		if e != nil {
			rows.Close()
			return nil, nil, false, e
		}
		nodes = append(nodes, d)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, nil, false, err
	}
	more := len(nodes) > 200
	if more {
		nodes = nodes[:200]
	}
	ids := []uuid.UUID{}
	for _, n := range nodes {
		ids = append(ids, n.ID)
	}
	rows, err = tx.Query(ctx, "SELECT source_id,target_id FROM document_links WHERE workspace_id=$1 AND source_id=ANY($2::uuid[]) AND target_id=ANY($2::uuid[]) ORDER BY source_id,target_id", ws, ids)
	if err != nil {
		return nil, nil, false, err
	}
	defer rows.Close()
	edges := []Edge{}
	for rows.Next() {
		var e Edge
		if err = rows.Scan(&e.Source, &e.Target); err != nil {
			return nil, nil, false, err
		}
		edges = append(edges, e)
	}
	return nodes, edges, more, rows.Err()
}

// Delete permanently removes a document, re-parenting its immediate child pages
// to its parent, and records the audit event.
func Delete(ctx context.Context, pool *pgxpool.Pool, actor, ws, id uuid.UUID) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	var parentID *uuid.UUID
	err = tx.QueryRow(ctx, "SELECT parent_id FROM documents WHERE workspace_id=$1 AND id=$2", ws, id).Scan(&parentID)
	if errors.Is(err, pgx.ErrNoRows) {
		return pgx.ErrNoRows
	}
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "UPDATE documents SET parent_id=$1 WHERE workspace_id=$2 AND parent_id=$3", parentID, ws, id); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "DELETE FROM documents WHERE workspace_id=$1 AND id=$2", ws, id); err != nil {
		return err
	}
	metadata, err := json.Marshal(map[string]any{"workspace_id": ws, "document_id": id})
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, "INSERT INTO security_audit_events (actor_id,event_type,metadata) VALUES ($1,$2,$3)", actor, "document.deleted", metadata); err != nil {
		return fmt.Errorf("record document delete: %w", err)
	}
	return tx.Commit(ctx)
}
