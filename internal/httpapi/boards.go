package httpapi

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/maruf-pfc/trestle/internal/boards"
)

func (s *Server) listBoards(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false)
	if !ok {
		return
	}
	offset := 0
	if raw := r.URL.Query().Get("offset"); raw != "" {
		var err error
		offset, err = strconv.Atoi(raw)
		if err != nil || offset < 0 || offset > 100000 {
			badRequest(w, "Offset must be between 0 and 100000")
			return
		}
	}
	list, more, err := boards.List(r.Context(), s.Pool, ids[0], r.URL.Query().Get("archived") == "true", offset)
	if err != nil {
		internal(w, "failed to list boards", err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"boards": list, "has_more": more, "limit": 50, "offset": offset})
}

func (s *Server) getBoard(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, _, ok := s.scope(w, r, session, false, "board_id")
	if !ok {
		return
	}
	b, err := boards.Get(r.Context(), s.Pool, ids[0], ids[1])
	if errors.Is(err, pgx.ErrNoRows) {
		notFound(w)
		return
	}
	if err != nil {
		internal(w, "failed to get board", err)
		return
	}
	writeJSON(w, http.StatusOK, b)
}

func (s *Server) saveBoard(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	names := []string{}
	if r.Method == http.MethodPut {
		names = append(names, "board_id")
	}
	ids, role, ok := s.scope(w, r, session, false, names...)
	if !ok {
		return
	}
	var req struct {
		Title       string       `json:"title"`
		Kind        string       `json:"kind"`
		Scene       boards.Scene `json:"scene"`
		Version     int64        `json:"version"`
		Archived    bool         `json:"archived"`
		CreationKey uuid.UUID    `json:"creation_key"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	if err := boards.Validate(req.Title, req.Scene); err != nil {
		badRequest(w, err.Error())
		return
	}
	if req.Kind != "diagram" && req.Kind != "whiteboard" {
		badRequest(w, "Choose diagram or whiteboard")
		return
	}
	id := uuid.Nil
	status := http.StatusCreated
	if len(ids) == 1 && req.CreationKey == uuid.Nil {
		badRequest(w, "A unique creation_key UUID is required to safely retry creation")
		return
	}
	if len(ids) > 1 {
		id = ids[1]
		status = http.StatusOK
		current, err := boards.Get(r.Context(), s.Pool, ids[0], id)
		if errors.Is(err, pgx.ErrNoRows) {
			notFound(w)
			return
		}
		if err != nil {
			internal(w, "failed to get board", err)
			return
		}
		if current.Archived != req.Archived && !isManager(role) {
			forbidden(w)
			return
		}
		if current.Kind != req.Kind {
			badRequest(w, "Board type cannot change")
			return
		}
		if current.Archived && req.Archived {
			conflict(w, "Restore this board before editing it.")
			return
		}
	}
	b, err := boards.Save(r.Context(), s.Pool, session.User.ID, ids[0], id, req.CreationKey, req.Title, req.Kind, req.Scene, req.Version, req.Archived)
	if errors.Is(err, boards.ErrConflict) {
		if id == uuid.Nil {
			conflict(w, "A previous board creation may already have succeeded. Export this draft, then check Boards before importing a new copy.")
			return
		}
		conflict(w, "This board changed. Export your draft, then reload the latest version before editing again.")
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		notFound(w)
		return
	}
	if err != nil {
		internal(w, "failed to save board", err)
		return
	}
	writeJSON(w, status, b)
}
