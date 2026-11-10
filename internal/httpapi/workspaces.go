package httpapi

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"unicode/utf8"

	"github.com/google/uuid"

	"github.com/maruf-pfc/trestle/internal/identity"
)

type workspaceItem struct {
	ID        string  `json:"id"`
	Name      string  `json:"name"`
	Slug      string  `json:"slug"`
	Role      string  `json:"role"`
	AvatarURL *string `json:"avatar_url,omitempty"`
}

type memberItem struct {
	UserID      string  `json:"user_id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	Role        string  `json:"role"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

func toWorkspaceItem(ws identity.WorkspaceProjection) workspaceItem {
	item := workspaceItem{ID: ws.ID.String(), Name: ws.Name, Slug: ws.Slug, Role: ws.Role}
	if ws.HasAvatar {
		u := "/api/v1/workspaces/" + ws.ID.String() + "/image"
		item.AvatarURL = &u
	}
	return item
}

func toMember(m identity.MemberProjection) memberItem {
	item := memberItem{UserID: m.UserID.String(), Username: m.Username, DisplayName: m.DisplayName, Role: m.Role}
	if m.HasAvatar {
		u := "/api/v1/users/" + m.UserID.String() + "/avatar"
		item.AvatarURL = &u
	}
	return item
}

// membership returns the caller's role, answering 404 (concealing the workspace) when not a member.
func (s *Server) membership(w http.ResponseWriter, r *http.Request, workspaceID, userID uuid.UUID) (string, bool) {
	role, err := identity.GetUserMembership(r.Context(), s.Pool, workspaceID, userID)
	if err != nil {
		internal(w, "failed to check workspace membership", err)
		return "", false
	}
	if role == "" {
		notFound(w)
		return "", false
	}
	return role, true
}

func isManager(role string) bool { return role == "owner" || role == "admin" }

func validRole(role string) bool { return role == "admin" || role == "member" || role == "guest" }

func (s *Server) listWorkspaces(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	list, err := identity.ListUserWorkspaces(r.Context(), s.Pool, session.User.ID)
	if err != nil {
		internal(w, "failed to list workspaces", err)
		return
	}
	items := make([]workspaceItem, len(list))
	for i, ws := range list {
		items[i] = toWorkspaceItem(ws)
	}
	writeJSON(w, http.StatusOK, map[string]any{"workspaces": items})
}

func (s *Server) createWorkspace(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	var req struct {
		Name *string `json:"name"`
		Slug *string `json:"slug"`
	}
	if !decodeJSON(w, r, &req) || !required(w, req.Name, req.Slug) {
		return
	}
	name, slug := strings.TrimSpace(*req.Name), strings.ToLower(strings.TrimSpace(*req.Slug))
	if name == "" || utf8.RuneCountInString(name) > 120 || !validProjectSlug(slug) {
		badRequest(w, "Enter a workspace name of 1–120 characters and a slug of 2–48 lowercase letters, numbers or hyphens")
		return
	}
	ws, err := identity.CreateWorkspace(r.Context(), s.Pool, session.User.ID, name, slug)
	if errors.Is(err, identity.ErrConflict) {
		conflict(w, "A workspace with this slug already exists. Choose another slug.")
		return
	}
	if err != nil {
		internal(w, "failed to create workspace", err)
		return
	}
	writeJSON(w, http.StatusCreated, toWorkspaceItem(ws))
}

func (s *Server) listMembers(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, ok := pathIDs(w, r, "workspace_id")
	if !ok {
		return
	}
	if _, ok := s.membership(w, r, ids[0], session.User.ID); !ok {
		return
	}
	list, err := identity.ListWorkspaceMembers(r.Context(), s.Pool, ids[0])
	if err != nil {
		internal(w, "failed to list workspace members", err)
		return
	}
	items := make([]memberItem, len(list))
	for i, m := range list {
		items[i] = toMember(m)
	}
	writeJSON(w, http.StatusOK, map[string]any{"members": items})
}

func (s *Server) addMember(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, ok := pathIDs(w, r, "workspace_id")
	if !ok {
		return
	}
	var req struct {
		Username *string `json:"username"`
		Role     *string `json:"role"`
	}
	if !decodeJSON(w, r, &req) || !required(w, req.Username, req.Role) {
		return
	}
	role, ok := s.membership(w, r, ids[0], session.User.ID)
	if !ok {
		return
	}
	if !isManager(role) {
		forbidden(w)
		return
	}
	if !validRole(*req.Role) {
		badRequest(w, "Role must be admin, member, or guest")
		return
	}
	username, err := identity.ParseUsername(*req.Username)
	if err != nil {
		badRequest(w, "Invalid username format")
		return
	}
	target, err := identity.FindUserProjectionByUsername(r.Context(), s.Pool, username)
	if errors.Is(err, identity.ErrNotFound) {
		notFound(w)
		return
	} else if err != nil {
		internal(w, "failed to find user by username", err)
		return
	}
	if target.Disabled {
		badRequest(w, "Cannot add a disabled user")
		return
	}
	m, err := identity.AddWorkspaceMember(r.Context(), s.Pool, &session.User.ID, ids[0], target.ID, *req.Role)
	switch {
	case err == nil:
		writeJSON(w, http.StatusCreated, toMember(m))
	case errors.Is(err, identity.ErrConflict):
		conflict(w, "User is already a member of this workspace")
	case errors.Is(err, identity.ErrForbidden):
		forbidden(w)
	case errors.Is(err, identity.ErrNotFound):
		notFound(w)
	default:
		internal(w, "failed to add member", err)
	}
}

// adminMayNotTouchOwner answers 403 when an admin targets an owner; it returns false when the request is done.
func (s *Server) adminMayNotTouchOwner(w http.ResponseWriter, r *http.Request, callerRole string, workspaceID, targetID uuid.UUID) bool {
	if callerRole != "admin" {
		return true
	}
	targetRole, ok := s.membership(w, r, workspaceID, targetID)
	if !ok {
		return false
	}
	if targetRole == "owner" {
		forbidden(w)
		return false
	}
	return true
}

func (s *Server) updateMemberRole(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, ok := pathIDs(w, r, "workspace_id", "user_id")
	if !ok {
		return
	}
	var req struct {
		Role *string `json:"role"`
	}
	if !decodeJSON(w, r, &req) || !required(w, req.Role) {
		return
	}
	role, ok := s.membership(w, r, ids[0], session.User.ID)
	if !ok {
		return
	}
	if !isManager(role) {
		forbidden(w)
		return
	}
	if !validRole(*req.Role) {
		badRequest(w, "Role must be admin, member, or guest")
		return
	}
	if !s.adminMayNotTouchOwner(w, r, role, ids[0], ids[1]) {
		return
	}
	m, err := identity.UpdateWorkspaceMemberRole(r.Context(), s.Pool, &session.User.ID, ids[0], ids[1], *req.Role)
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, toMember(m))
	case errors.Is(err, identity.ErrConflict):
		conflict(w, "Cannot demote the last active owner")
	case errors.Is(err, identity.ErrNotFound):
		notFound(w)
	case errors.Is(err, identity.ErrForbidden):
		forbidden(w)
	default:
		internal(w, "failed to update member role", err)
	}
}

func (s *Server) removeMember(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, ok := pathIDs(w, r, "workspace_id", "user_id")
	if !ok {
		return
	}
	role, ok := s.membership(w, r, ids[0], session.User.ID)
	if !ok {
		return
	}
	if session.User.ID != ids[1] {
		if !isManager(role) {
			forbidden(w)
			return
		}
		if !s.adminMayNotTouchOwner(w, r, role, ids[0], ids[1]) {
			return
		}
	}
	err := identity.RemoveWorkspaceMember(r.Context(), s.Pool, &session.User.ID, ids[0], ids[1])
	switch {
	case err == nil:
		w.WriteHeader(http.StatusNoContent)
	case errors.Is(err, identity.ErrConflict):
		conflict(w, "Cannot remove the last active owner")
	case errors.Is(err, identity.ErrNotFound):
		notFound(w)
	default:
		internal(w, "failed to remove member", err)
	}
}

func (s *Server) transferOwnership(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, ok := pathIDs(w, r, "workspace_id")
	if !ok {
		return
	}
	var req struct {
		TargetUserID *uuid.UUID `json:"target_user_id"`
		Password     *string    `json:"password"`
	}
	if !decodeJSON(w, r, &req) || !required(w, req.TargetUserID, req.Password) {
		return
	}
	role, ok := s.membership(w, r, ids[0], session.User.ID)
	if !ok {
		return
	}
	if role != "owner" {
		forbidden(w)
		return
	}
	// Reauthenticate with the caller's password.
	_, hash, found, err := identity.FindUserByUsername(r.Context(), s.Pool, session.User.Username)
	if err != nil || !found {
		internal(w, "failed to load caller for reauthentication", err)
		return
	}
	if valid, _ := s.Hasher.Verify([]byte(*req.Password), hash); !valid {
		writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid password confirmation")
		return
	}
	err = identity.TransferWorkspaceOwnership(r.Context(), s.Pool, &session.User.ID, ids[0], session.User.ID, *req.TargetUserID)
	switch {
	case err == nil:
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	case errors.Is(err, identity.ErrNotFound):
		notFound(w)
	case errors.Is(err, identity.ErrForbidden):
		forbidden(w)
	default:
		internal(w, "failed to transfer ownership", err)
	}
}

func (s *Server) updateWorkspace(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, ok := pathIDs(w, r, "workspace_id")
	if !ok {
		return
	}
	role, ok := s.membership(w, r, ids[0], session.User.ID)
	if !ok {
		return
	}
	if !isManager(role) {
		forbidden(w)
		return
	}
	var req struct {
		Name *string `json:"name"`
		Slug *string `json:"slug"`
	}
	if !decodeJSON(w, r, &req) || !required(w, req.Name, req.Slug) {
		return
	}
	name, slug := strings.TrimSpace(*req.Name), strings.ToLower(strings.TrimSpace(*req.Slug))
	if name == "" || utf8.RuneCountInString(name) > 120 || !validProjectSlug(slug) {
		badRequest(w, "Enter a workspace name of 1–120 characters and a slug of 2–48 lowercase letters, numbers or hyphens")
		return
	}
	ws, err := identity.UpdateWorkspace(r.Context(), s.Pool, ids[0], name, slug)
	if errors.Is(err, identity.ErrConflict) {
		conflict(w, "A workspace with this slug already exists. Choose another slug.")
		return
	}
	if err != nil {
		internal(w, "failed to update workspace", err)
		return
	}
	ws.Role = role
	writeJSON(w, http.StatusOK, toWorkspaceItem(ws))
}

func (s *Server) getWorkspaceImage(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, ok := pathIDs(w, r, "workspace_id")
	if !ok {
		return
	}
	if _, ok := s.membership(w, r, ids[0], session.User.ID); !ok {
		return
	}
	data, mime, updated, err := identity.GetWorkspaceAvatar(r.Context(), s.Pool, ids[0])
	if err != nil || len(data) == 0 {
		notFound(w)
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("Last-Modified", updated.Format(http.TimeFormat))
	_, _ = w.Write(data)
}

func (s *Server) setWorkspaceImage(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, ok := pathIDs(w, r, "workspace_id")
	if !ok {
		return
	}
	role, ok := s.membership(w, r, ids[0], session.User.ID)
	if !ok {
		return
	}
	if !isManager(role) {
		forbidden(w)
		return
	}
	data, mime, ok := extractUploadedImage(w, r)
	if !ok {
		return
	}
	if err := identity.SetWorkspaceAvatar(r.Context(), s.Pool, ids[0], data, mime); err != nil {
		internal(w, "failed to save workspace image", err)
		return
	}
	url := fmt.Sprintf("/api/v1/workspaces/%s/image", ids[0])
	writeJSON(w, http.StatusOK, map[string]string{"image_url": url})
}

func (s *Server) deleteWorkspaceImage(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ids, ok := pathIDs(w, r, "workspace_id")
	if !ok {
		return
	}
	role, ok := s.membership(w, r, ids[0], session.User.ID)
	if !ok {
		return
	}
	if !isManager(role) {
		forbidden(w)
		return
	}
	if err := identity.DeleteWorkspaceAvatar(r.Context(), s.Pool, ids[0]); err != nil {
		internal(w, "failed to delete workspace image", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
