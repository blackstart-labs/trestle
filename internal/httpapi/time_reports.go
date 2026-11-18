package httpapi

import (
	"encoding/csv"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"net/http"
	"strings"
	"time"
)

func safeCSV(s string) string {
	if s != "" && strings.ContainsRune("=+-@\t\r\n", rune(s[0])) {
		return "'" + s
	}
	return s
}
func (s *Server) timeReport(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, role, ok := s.scope(w, r, session, false)
	if !ok {
		return
	}
	if role == "guest" {
		forbidden(w)
		return
	}
	q := r.URL.Query()
	from, e := time.Parse(time.DateOnly, q.Get("from"))
	if e != nil {
		badRequest(w, "Choose a start date")
		return
	}
	to, e := time.Parse(time.DateOnly, q.Get("to"))
	if e != nil || to.Before(from) || to.Sub(from) > 366*24*time.Hour {
		badRequest(w, "Choose a date range of at most 367 days")
		return
	}
	offset, ok := documentOffset(w, r)
	if !ok {
		return
	}
	project, ok := queryUUID(w, r, "project_id")
	if !ok {
		return
	}
	user, ok := queryUUID(w, r, "user_id")
	if !ok {
		return
	}
	if !isManager(role) {
		user = &session.User.ID
	}
	tx, e := s.Pool.BeginTx(r.Context(), pgx.TxOptions{IsoLevel: pgx.RepeatableRead, AccessMode: pgx.ReadOnly})
	if e != nil {
		internal(w, "failed to start time report", e)
		return
	}
	defer tx.Rollback(r.Context())
	w.Header().Set("Cache-Control", "private, no-store")
	where := ` FROM task_time_logs l JOIN tasks t ON t.id=l.task_id JOIN projects p ON p.id=l.project_id LEFT JOIN users u ON u.id=l.user_id WHERE l.workspace_id=$1 AND l.logged_date BETWEEN $2 AND $3 AND ($4::uuid IS NULL OR l.project_id=$4) AND ($5::uuid IS NULL OR l.user_id=$5)`
	args := []any{ids[0], from.Format(time.DateOnly), to.Format(time.DateOnly), project, user}
	var hours float64
	var count int
	if e = tx.QueryRow(r.Context(), "SELECT COALESCE(sum(l.hours),0),count(*)"+where, args...).Scan(&hours, &count); e != nil {
		internal(w, "time totals failed", e)
		return
	}
	export := q.Get("format") == "csv"
	limit := 101
	if export {
		if count > 10000 {
			badRequest(w, "Narrow the date or project filter to export at most 10000 entries")
			return
		}
		limit = 10001
		offset = 0
	}
	rows, e := tx.Query(r.Context(), `SELECT l.id,l.project_id,l.task_id,COALESCE(l.user_id,'00000000-0000-0000-0000-000000000000'::uuid),COALESCE(u.display_name,'Former member'),p.name,p.key||'-'||t.task_number,t.title,l.hours,l.logged_date::text,l.description`+where+` ORDER BY l.logged_date DESC,l.created_at DESC,l.id LIMIT $6 OFFSET $7`, append(args, limit, offset)...)
	if e != nil {
		internal(w, "time report failed", e)
		return
	}
	defer rows.Close()
	type entry struct {
		ID          uuid.UUID `json:"id"`
		ProjectID   uuid.UUID `json:"project_id"`
		TaskID      uuid.UUID `json:"task_id"`
		UserID      uuid.UUID `json:"user_id"`
		User        string    `json:"user_name"`
		Project     string    `json:"project_name"`
		Key         string    `json:"task_key"`
		Title       string    `json:"task_title"`
		Hours       float64   `json:"hours"`
		Date        string    `json:"date"`
		Description string    `json:"description"`
	}
	list := []entry{}
	for rows.Next() {
		var x entry
		if e = rows.Scan(&x.ID, &x.ProjectID, &x.TaskID, &x.UserID, &x.User, &x.Project, &x.Key, &x.Title, &x.Hours, &x.Date, &x.Description); e != nil {
			internal(w, "time row failed", e)
			return
		}
		list = append(list, x)
	}
	if e = rows.Err(); e != nil {
		internal(w, "time report failed", e)
		return
	}
	if export {
		if len(list) > 10000 {
			badRequest(w, "More entries were added. Narrow the report and retry export")
			return
		}
		w.Header().Set("Content-Type", "text/csv; charset=utf-8")
		w.Header().Set("Content-Disposition", `attachment; filename="trestle-timesheet.csv"`)
		writer := csv.NewWriter(w)
		writer.Write([]string{"Date", "Member", "Project", "Task", "Title", "Hours", "Description"})
		for _, x := range list {
			writer.Write([]string{x.Date, safeCSV(x.User), safeCSV(x.Project), safeCSV(x.Key), safeCSV(x.Title), fmt.Sprintf("%.2f", x.Hours), safeCSV(x.Description)})
		}
		writer.Flush()
		return
	}
	more := len(list) > 100
	if more {
		list = list[:100]
	}
	writeJSON(w, 200, map[string]any{"entries": list, "total_hours": hours, "total_entries": count, "has_more": more, "offset": offset})
}
