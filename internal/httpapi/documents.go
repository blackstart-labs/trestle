package httpapi

import (
	"errors"
	"net/http"
	"strconv"
	"unicode/utf8"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/maruf-pfc/trestle/internal/documents"
)

func documentOffset(w http.ResponseWriter, r *http.Request) (int, bool) {
	raw := r.URL.Query().Get("offset")
	if raw == "" {
		return 0, true
	}
	v, e := strconv.Atoi(raw)
	if e != nil || v < 0 || v > 100000 {
		badRequest(w, "Offset must be between 0 and 100000")
		return 0, false
	}
	return v, true
}
func documentError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, pgx.ErrNoRows):
		notFound(w)
	case errors.Is(err, documents.ErrConflict):
		conflict(w, err.Error())
	case errors.Is(err, documents.ErrInvalid):
		badRequest(w, err.Error())
	default:
		internal(w, "document operation failed", err)
	}
}
func (s *Server) listDocuments(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false)
	if !ok {
		return
	}
	offset, ok := documentOffset(w, r)
	if !ok {
		return
	}
	q, tag := r.URL.Query().Get("q"), r.URL.Query().Get("tag")
	if utf8.RuneCountInString(q) > 200 || utf8.RuneCountInString(tag) > 40 {
		badRequest(w, "Search text is too long")
		return
	}
	archived := r.URL.Query().Get("archived")
	if archived != "" && archived != "true" && archived != "false" {
		badRequest(w, "Archived must be true or false")
		return
	}
	parent, ok := queryUUID(w, r, "parent_id")
	if !ok {
		return
	}
	list, more, err := documents.List(r.Context(), s.Pool, ids[0], archived == "true", q, tag, parent, offset)
	if err != nil {
		documentError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"documents": list, "has_more": more, "offset": offset, "limit": 50})
}
func (s *Server) getDocument(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "document_id")
	if !ok {
		return
	}
	d, err := documents.Get(r.Context(), s.Pool, ids[0], ids[1])
	if err != nil {
		documentError(w, err)
		return
	}
	writeJSON(w, 200, d)
}
func (s *Server) saveDocument(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	names := []string{}
	if r.Method == http.MethodPut {
		names = append(names, "document_id")
	}
	ids, role, ok := s.scope(w, r, session, false, names...)
	if !ok {
		return
	}
	var in documents.Input
	if !decodeJSON(w, r, &in) {
		return
	}
	if err := documents.Validate(in); err != nil {
		badRequest(w, err.Error())
		return
	}
	id := uuid.Nil
	status := http.StatusCreated
	if len(ids) > 1 {
		id = ids[1]
		status = http.StatusOK
		current, err := documents.Get(r.Context(), s.Pool, ids[0], id)
		if err != nil {
			documentError(w, err)
			return
		}
		if current.Archived != in.Archived && !isManager(role) {
			forbidden(w)
			return
		}
	}
	d, err := documents.Save(r.Context(), s.Pool, session.User.ID, ids[0], id, in, isManager(role))
	if err != nil {
		documentError(w, err)
		return
	}
	writeJSON(w, status, d)
}
func (s *Server) documentHistory(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "document_id")
	if !ok {
		return
	}
	if _, err := documents.Get(r.Context(), s.Pool, ids[0], ids[1]); err != nil {
		documentError(w, err)
		return
	}
	if raw := r.PathValue("version"); raw != "" {
		version, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || version < 1 {
			badRequest(w, "Invalid revision")
			return
		}
		d, err := documents.Revision(r.Context(), s.Pool, ids[0], ids[1], version)
		if err != nil {
			documentError(w, err)
			return
		}
		writeJSON(w, 200, d)
		return
	}
	offset, ok := documentOffset(w, r)
	if !ok {
		return
	}
	list, more, err := documents.History(r.Context(), s.Pool, ids[0], ids[1], offset)
	if err != nil {
		documentError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"documents": list, "has_more": more, "offset": offset})
}
func (s *Server) documentBacklinks(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "document_id")
	if !ok {
		return
	}
	if _, err := documents.Get(r.Context(), s.Pool, ids[0], ids[1]); err != nil {
		documentError(w, err)
		return
	}
	offset, ok := documentOffset(w, r)
	if !ok {
		return
	}
	list, more, err := documents.Backlinks(r.Context(), s.Pool, ids[0], ids[1], offset)
	if err != nil {
		documentError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"documents": list, "has_more": more, "offset": offset})
}

func (s *Server) documentGraph(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false)
	if !ok {
		return
	}
	q, tag := r.URL.Query().Get("q"), r.URL.Query().Get("tag")
	if utf8.RuneCountInString(q) > 200 || utf8.RuneCountInString(tag) > 40 {
		badRequest(w, "Search filter is too long")
		return
	}
	nodes, edges, more, err := documents.Graph(r.Context(), s.Pool, ids[0], q, tag)
	if err != nil {
		documentError(w, err)
		return
	}
	writeJSON(w, 200, map[string]any{"nodes": nodes, "edges": edges, "truncated": more})
}

func (s *Server) deleteDocument(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "document_id")
	if !ok {
		return
	}
	err := documents.Delete(r.Context(), s.Pool, session.User.ID, ids[0], ids[1])
	if errors.Is(err, pgx.ErrNoRows) {
		notFound(w)
		return
	}
	if err != nil {
		documentError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
