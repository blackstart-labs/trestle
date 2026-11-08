package httpapi

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/google/uuid"

	"github.com/maruf-pfc/trestle/internal/identity"
)

type sessionKey struct{}

// AuthenticatedSession is present only for a valid, unexpired, unrevoked session of an enabled user.
type AuthenticatedSession struct {
	SessionID     uuid.UUID
	User          identity.UserProjection
	CSRFTokenHash string
}

func sessionFrom(r *http.Request) *AuthenticatedSession {
	s, _ := r.Context().Value(sessionKey{}).(*AuthenticatedSession)
	return s
}

// resolveSession attaches the session named by the trestle-session cookie, if valid.
func (s *Server) resolveSession(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if c, err := r.Cookie("trestle-session"); err == nil {
			resolved, err := identity.ResolveSession(r.Context(), s.Pool, c.Value)
			if err != nil {
				slog.Warn("session resolution database error", "error", err)
			} else if resolved != nil {
				r = r.WithContext(context.WithValue(r.Context(), sessionKey{}, &AuthenticatedSession{
					SessionID: resolved.SessionID, User: resolved.User, CSRFTokenHash: resolved.CSRFTokenHash,
				}))
			}
		}
		next.ServeHTTP(w, r)
	})
}

// csrf requires a valid X-CSRF-Token on mutating requests that carry a session.
// Requests without a session are left to the handler, which rejects them if needed.
func (s *Server) csrf(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
			if session := sessionFrom(r); session != nil {
				hash, err := identity.HashClientToken(r.Header.Get("X-CSRF-Token"))
				if err != nil || !identity.ConstantTimeEqual(hash, session.CSRFTokenHash) {
					writeError(w, http.StatusForbidden, "csrf_invalid", "Invalid or missing CSRF token")
					return
				}
			}
		}
		next.ServeHTTP(w, r)
	})
}

type authedHandler func(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession)

// auth answers 401 when no session is present.
func (s *Server) auth(h authedHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		session := sessionFrom(r)
		if session == nil {
			writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
			return
		}
		h(w, r, session)
	}
}

type userResponse struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	DisplayName string  `json:"display_name"`
	Locale      string  `json:"locale"`
	Timezone    string  `json:"timezone"`
	AvatarURL   *string `json:"avatar_url,omitempty"`
}

func toUserResponse(u identity.UserProjection) userResponse {
	resp := userResponse{ID: u.ID.String(), Username: u.Username, DisplayName: u.DisplayName, Locale: u.Locale, Timezone: u.Timezone}
	if u.HasAvatar {
		url := "/api/v1/users/" + u.ID.String() + "/avatar"
		resp.AvatarURL = &url
	}
	return resp
}

func (s *Server) me(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	writeJSON(w, http.StatusOK, toUserResponse(session.User))
}

func (s *Server) updateProfile(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	var req struct {
		DisplayName *string `json:"display_name"`
		Timezone    *string `json:"timezone"`
		Locale      *string `json:"locale"`
	}
	if !decodeJSON(w, r, &req) {
		return
	}
	name := session.User.DisplayName
	if req.DisplayName != nil && *req.DisplayName != "" {
		name = *req.DisplayName
	}
	tz := session.User.Timezone
	if req.Timezone != nil {
		tz = *req.Timezone
	}
	loc := session.User.Locale
	if req.Locale != nil {
		loc = *req.Locale
	}
	u, err := identity.UpdateUserProfile(r.Context(), s.Pool, session.User.ID, name, tz, loc)
	if err != nil {
		badRequest(w, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, toUserResponse(u))
}

func (s *Server) getUserAvatar(w http.ResponseWriter, r *http.Request) {
	userIDStr := r.PathValue("user_id")
	var userID uuid.UUID
	var err error
	if userIDStr == "" || userIDStr == "me" {
		sess := sessionFrom(r)
		if sess == nil {
			unauthorized(w)
			return
		}
		userID = sess.User.ID
	} else {
		userID, err = uuid.Parse(userIDStr)
		if err != nil {
			badRequest(w, "invalid user id")
			return
		}
	}
	data, mime, updated, err := identity.GetUserAvatar(r.Context(), s.Pool, userID)
	if err != nil || len(data) == 0 {
		notFound(w)
		return
	}
	w.Header().Set("Content-Type", mime)
	w.Header().Set("Cache-Control", "public, max-age=3600")
	w.Header().Set("Last-Modified", updated.Format(http.TimeFormat))
	_, _ = w.Write(data)
}

func (s *Server) setUserAvatar(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	data, mime, ok := extractUploadedImage(w, r)
	if !ok {
		return
	}
	if err := identity.SetUserAvatar(r.Context(), s.Pool, session.User.ID, data, mime); err != nil {
		internal(w, "failed to save avatar", err)
		return
	}
	url := "/api/v1/users/" + session.User.ID.String() + "/avatar"
	writeJSON(w, http.StatusOK, map[string]string{"avatar_url": url})
}

func (s *Server) deleteUserAvatar(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	if err := identity.DeleteUserAvatar(r.Context(), s.Pool, session.User.ID); err != nil {
		internal(w, "failed to delete avatar", err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func invalidCredentials(w http.ResponseWriter) {
	writeError(w, http.StatusUnauthorized, "unauthorized", "Invalid credentials")
}

// login answers 401 identically for unknown users and wrong passwords (with dummy
// verification to equalise timing) and 429 while a rate limit is active.
func (s *Server) login(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Username *string `json:"username"`
		Password *string `json:"password"`
	}
	if !decodeJSON(w, r, &req) || !required(w, req.Username, req.Password) {
		return
	}
	username, err := identity.ParseUsername(*req.Username)
	if err != nil {
		s.Hasher.DummyVerify()
		invalidCredentials(w)
		return
	}
	ip := peerIP(r)
	if s.Limiter.Exceeded(ip, username) {
		w.Header().Set("Retry-After", "900")
		writeError(w, http.StatusTooManyRequests, "rate_limited", "Too many failed login attempts, please wait before trying again")
		return
	}
	ctx := r.Context()
	userID, hash, found, err := identity.FindUserByUsername(ctx, s.Pool, username)
	if err != nil {
		s.Hasher.DummyVerify()
		invalidCredentials(w)
		return
	}
	if !found {
		s.Hasher.DummyVerify()
		s.Limiter.RecordFailure(ip, username)
		invalidCredentials(w)
		return
	}
	ok, err := s.Hasher.Verify([]byte(*req.Password), hash)
	if err != nil || !ok {
		s.Limiter.RecordFailure(ip, username)
		invalidCredentials(w)
		return
	}
	sessionToken, csrfToken, err := identity.CreateSession(ctx, s.Pool, userID)
	if err != nil {
		internal(w, "failed to create session", err)
		return
	}
	user, err := identity.FindUserByID(ctx, s.Pool, userID)
	if err != nil {
		internal(w, "failed to fetch user after login", err)
		return
	}
	_ = identity.RecordAuditEvent(ctx, s.Pool, &userID, "session.created", map[string]any{"method": "password"})

	w.Header().Add("Set-Cookie", "trestle-session="+sessionToken+"; HttpOnly"+s.secure()+"; SameSite=Lax; Path=/; Max-Age=604800")
	// Not HttpOnly: the frontend reads it and sends it back as X-CSRF-Token.
	w.Header().Add("Set-Cookie", "trestle-csrf="+csrfToken+s.secure()+"; SameSite=Lax; Path=/; Max-Age=604800")
	writeJSON(w, http.StatusOK, toUserResponse(user))
}

// logout revokes the current session; CSRF is validated by middleware.
func (s *Server) logout(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	ctx := r.Context()
	_ = identity.RevokeSession(ctx, s.Pool, session.SessionID)
	_ = identity.RecordAuditEvent(ctx, s.Pool, &session.User.ID, "session.revoked", map[string]any{})
	w.Header().Add("Set-Cookie", "trestle-session=; HttpOnly"+s.secure()+"; SameSite=Lax; Path=/; Max-Age=0")
	w.Header().Add("Set-Cookie", "trestle-csrf=; SameSite=Lax"+s.secure()+"; Path=/; Max-Age=0")
	w.WriteHeader(http.StatusNoContent)
}

// secure omits the Secure cookie flag only in loopback development mode.
func (s *Server) secure() string {
	if s.Config.DevMode {
		return ""
	}
	return "; Secure"
}
