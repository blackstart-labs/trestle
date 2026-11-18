package httpapi

import (
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/maruf-pfc/trestle/internal/planning"
	"net/http"
	"unicode/utf8"
)

func planningError(w http.ResponseWriter, e error) {
	switch {
	case errors.Is(e, pgx.ErrNoRows):
		notFound(w)
	case errors.Is(e, planning.ErrConflict), errors.Is(e, planning.ErrProjectLinked):
		conflict(w, e.Error())
	case errors.Is(e, planning.ErrInvalid):
		badRequest(w, e.Error())
	default:
		internal(w, "planning operation failed", e)
	}
}
func (s *Server) listPlans(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false)
	if !ok {
		return
	}
	offset, ok := documentOffset(w, r)
	if !ok {
		return
	}
	q := r.URL.Query()
	kind := q.Get("kind")
	if (kind != "" && !planning.Kind(kind)) || utf8.RuneCountInString(q.Get("q")) > 200 || (q.Get("archived") != "" && q.Get("archived") != "true" && q.Get("archived") != "false") {
		badRequest(w, "Invalid planning filters")
		return
	}
	items, more, e := planning.List(r.Context(), s.Pool, ids[0], kind, q.Get("q"), q.Get("archived") == "true", offset)
	if e != nil {
		planningError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"items": items, "has_more": more, "offset": offset})
}
func (s *Server) getPlan(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "plan_id")
	if !ok {
		return
	}
	item, e := planning.Get(r.Context(), s.Pool, ids[0], ids[1])
	if e != nil {
		planningError(w, e)
		return
	}
	progress, e := planning.Measurement(r.Context(), s.Pool, ids[0], ids[1])
	if e != nil {
		planningError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"item": item, "measurement_progress": progress})
}
func (s *Server) savePlan(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	names := []string{}
	if r.Method == http.MethodPut {
		names = append(names, "plan_id")
	}
	ids, role, ok := s.scope(w, r, session, false, names...)
	if !ok {
		return
	}
	var in planning.Input
	if !decodeJSON(w, r, &in) {
		return
	}
	if e := planning.Validate(in); e != nil {
		badRequest(w, e.Error())
		return
	}
	id := uuid.Nil
	status := 201
	if len(ids) > 1 {
		id = ids[1]
		status = 200
		old, e := planning.Get(r.Context(), s.Pool, ids[0], id)
		if e != nil {
			planningError(w, e)
			return
		}
		if old.Archived != in.Archived && !isManager(role) {
			forbidden(w)
			return
		}
	}
	item, e := planning.Save(r.Context(), s.Pool, session.User.ID, ids[0], id, in, isManager(role))
	if e != nil {
		planningError(w, e)
		return
	}
	writeJSON(w, status, item)
}
func (s *Server) planRelations(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "plan_id")
	if !ok {
		return
	}
	if _, e := planning.Get(r.Context(), s.Pool, ids[0], ids[1]); e != nil {
		planningError(w, e)
		return
	}
	offset, ok := documentOffset(w, r)
	if !ok {
		return
	}
	if r.PathValue("relation") == "children" {
		items, more, e := planning.Children(r.Context(), s.Pool, ids[0], ids[1], offset)
		if e != nil {
			planningError(w, e)
			return
		}
		writeJSON(w, 200, map[string]any{"items": items, "has_more": more})
		return
	}
	if r.PathValue("relation") != "tasks" {
		notFound(w)
		return
	}
	items, more, e := planning.Tasks(r.Context(), s.Pool, ids[0], ids[1], offset)
	if e != nil {
		planningError(w, e)
		return
	}
	writeJSON(w, 200, map[string]any{"items": items, "has_more": more})
}
func (s *Server) linkPlanTask(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "plan_id", "task_id")
	if !ok {
		return
	}
	e := planning.Link(r.Context(), s.Pool, session.User.ID, ids[0], ids[1], ids[2], r.Method == http.MethodDelete)
	if e != nil {
		planningError(w, e)
		return
	}
	writeJSON(w, 200, map[string]bool{"ok": true})
}
