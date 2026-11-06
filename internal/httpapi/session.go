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
	ID          string `json:"id"`
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Locale      string `json:"locale"`
	Timezone    string `json:"timezone"`
}

func toUserResponse(u identity.UserProjection) userResponse {
	return userResponse{u.ID.String(), u.Username, u.DisplayName, u.Locale, u.Timezone}
}

func (s *Server) me(w http.ResponseWriter, r *http.Request, session *AuthenticatedSession) {
	writeJSON(w, http.StatusOK, toUserResponse(session.User))
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
	if ok, err := s.Hasher.Verify([]byte(*req.Password), hash); err != nil || !ok {
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
