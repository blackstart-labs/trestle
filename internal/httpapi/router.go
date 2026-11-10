// Package httpapi serves the JSON API, health checks and the compiled frontend.
package httpapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"mime"
	"net"
	"net/http"
	"path"
	"reflect"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/maruf-pfc/trestle/internal/config"
	"github.com/maruf-pfc/trestle/internal/identity"
)

// Server is the shared application state injected into all handlers.
type Server struct {
	Pool    *pgxpool.Pool
	Timeout time.Duration
	Config  config.Config
	Hasher  identity.Hasher
	Limiter *identity.LoginLimiter
}

const csp = "default-src 'none'; script-src 'self'; style-src 'self'; font-src 'self'; img-src 'self' blob:; connect-src 'self'; base-uri 'none'; object-src 'none'; frame-ancestors 'none'; form-action 'none'"

// maxBody matches the previous framework's default JSON body limit.
const maxBody = 2 << 20

// Router builds the full handler. assets is the compiled frontend (index.html and assets/).
func Router(pool *pgxpool.Pool, timeout time.Duration, cfg config.Config, assets fs.FS) http.Handler {
	hasher, err := identity.NewHasher(cfg.Argon2MCost, cfg.Argon2TCost, cfg.Argon2Parallelism)
	if err != nil {
		slog.Warn("argon2 parameters invalid, using defaults")
		hasher = identity.DefaultHasher()
	}
	s := &Server{Pool: pool, Timeout: timeout, Config: cfg, Hasher: hasher, Limiter: identity.NewLoginLimiter()}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", serveAsset(assets, "index.html"))
	mux.HandleFunc("GET /assets/{file}", func(w http.ResponseWriter, r *http.Request) {
		serveAsset(assets, "assets/"+r.PathValue("file"))(w, r)
	})
	mux.HandleFunc("GET /health/live", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("GET /health/ready", s.readiness)

	// Identity
	mux.HandleFunc("POST /api/v1/session", s.login)
	mux.HandleFunc("DELETE /api/v1/session", s.auth(s.logout))
	mux.HandleFunc("GET /api/v1/me", s.auth(s.me))
	mux.HandleFunc("PATCH /api/v1/me", s.auth(s.updateProfile))
	mux.HandleFunc("GET /api/v1/me/avatar", s.getUserAvatar)
	mux.HandleFunc("PUT /api/v1/me/avatar", s.auth(s.setUserAvatar))
	mux.HandleFunc("DELETE /api/v1/me/avatar", s.auth(s.deleteUserAvatar))
	mux.HandleFunc("GET /api/v1/users/{user_id}/avatar", s.getUserAvatar)
	mux.HandleFunc("GET /api/v1/workspaces", s.auth(s.listWorkspaces))
	mux.HandleFunc("POST /api/v1/workspaces", s.auth(s.createWorkspace))
	const ws = "/api/v1/workspaces/{workspace_id}"
	mux.HandleFunc("PATCH "+ws, s.auth(s.updateWorkspace))
	mux.HandleFunc("GET "+ws+"/image", s.auth(s.getWorkspaceImage))
	mux.HandleFunc("PUT "+ws+"/image", s.auth(s.setWorkspaceImage))
	mux.HandleFunc("DELETE "+ws+"/image", s.auth(s.deleteWorkspaceImage))
	mux.HandleFunc("GET "+ws+"/export", s.auth(s.exportWorkspace))
	mux.HandleFunc("GET "+ws+"/members", s.auth(s.listMembers))
	mux.HandleFunc("POST "+ws+"/members", s.auth(s.addMember))
	mux.HandleFunc("PATCH "+ws+"/members/{user_id}", s.auth(s.updateMemberRole))
	mux.HandleFunc("DELETE "+ws+"/members/{user_id}", s.auth(s.removeMember))
	mux.HandleFunc("POST "+ws+"/transfer-ownership", s.auth(s.transferOwnership))

	// Projects and tasks
	const p = ws + "/projects/{project_id}"
	const t = p + "/tasks/{task_id}"
	mux.HandleFunc("GET "+ws+"/projects", s.auth(s.listProjects))
	mux.HandleFunc("POST "+ws+"/projects", s.auth(s.createProject))
	mux.HandleFunc("GET "+p, s.auth(s.getProject))
	mux.HandleFunc("PATCH "+p, s.auth(s.updateProject))
	mux.HandleFunc("POST "+p+"/archive", s.auth(s.archiveProject))
	mux.HandleFunc("GET "+p+"/image", s.auth(s.getProjectImage))
	mux.HandleFunc("PUT "+p+"/image", s.auth(s.setProjectImage))
	mux.HandleFunc("DELETE "+p+"/image", s.auth(s.deleteProjectImage))
	mux.HandleFunc("GET "+p+"/export", s.auth(s.exportProject))
	mux.HandleFunc("GET "+p+"/integrations", s.auth(s.listIntegrations))
	mux.HandleFunc("PUT "+p+"/integrations/{provider}", s.auth(s.saveIntegration))
	mux.HandleFunc("DELETE "+p+"/integrations/{provider}", s.auth(s.deleteIntegration))
	mux.HandleFunc("POST "+p+"/integrations/{provider}/test", s.auth(s.testIntegration))
	mux.HandleFunc("GET "+p+"/tasks", s.auth(s.listTasks))
	mux.HandleFunc("POST "+p+"/tasks", s.auth(s.createTask))
	mux.HandleFunc("GET "+t, s.auth(s.getTask))
	mux.HandleFunc("PATCH "+t, s.auth(s.updateTask))
	mux.HandleFunc("DELETE "+t, s.auth(s.deleteTask))
	mux.HandleFunc("GET "+t+"/time-logs", s.auth(s.listTaskTimeLogs))
	mux.HandleFunc("POST "+t+"/time-logs", s.auth(s.createTaskTimeLog))
	mux.HandleFunc("DELETE "+t+"/time-logs/{log_id}", s.auth(s.deleteTaskTimeLog))

	// Custom statuses
	mux.HandleFunc("GET "+p+"/statuses", s.auth(s.listStatuses))
	mux.HandleFunc("POST "+p+"/statuses", s.auth(s.createStatus))
	mux.HandleFunc("PUT "+p+"/statuses/positions", s.auth(s.reorderStatuses))
	mux.HandleFunc("PATCH "+p+"/statuses/{status_id}", s.auth(s.updateStatus))
	mux.HandleFunc("DELETE "+p+"/statuses/{status_id}", s.auth(s.deleteStatus))

	// Milestones
	mux.HandleFunc("GET "+p+"/milestones", s.auth(s.listMilestones))
	mux.HandleFunc("POST "+p+"/milestones", s.auth(s.createMilestone))
	mux.HandleFunc("GET "+p+"/milestones/{milestone_id}", s.auth(s.getMilestone))
	mux.HandleFunc("PATCH "+p+"/milestones/{milestone_id}", s.auth(s.updateMilestone))
	mux.HandleFunc("DELETE "+p+"/milestones/{milestone_id}", s.auth(s.deleteMilestone))

	// Labels, assignees, subtasks, comments, activity
	mux.HandleFunc("GET "+p+"/labels", s.auth(s.listLabels))
	mux.HandleFunc("POST "+p+"/labels", s.auth(s.createLabel))
	mux.HandleFunc("DELETE "+p+"/labels/{label_id}", s.auth(s.deleteLabel))
	mux.HandleFunc("POST "+t+"/labels", s.auth(s.attachLabel))
	mux.HandleFunc("DELETE "+t+"/labels/{label_id}", s.auth(s.detachLabel))
	mux.HandleFunc("POST "+t+"/assignees", s.auth(s.assignUser))
	mux.HandleFunc("DELETE "+t+"/assignees/{user_id}", s.auth(s.unassignUser))
	mux.HandleFunc("GET "+t+"/subtasks", s.auth(s.listSubtasks))
	mux.HandleFunc("POST "+t+"/subtasks", s.auth(s.createSubtask))
	mux.HandleFunc("GET "+t+"/comments", s.auth(s.listComments))
	mux.HandleFunc("POST "+t+"/comments", s.auth(s.createComment))
	mux.HandleFunc("DELETE "+t+"/comments/{comment_id}", s.auth(s.deleteComment))
	mux.HandleFunc("GET "+t+"/activity", s.auth(s.listActivity))

	return diagnostics(s.resolveSession(s.csrf(jsonFallbacks(mux))))
}

// ---------------------------------------------------------------------------
// Responses
// ---------------------------------------------------------------------------

func writeJSON(w http.ResponseWriter, status int, v any) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		slog.Error("failed to encode response", "error", err)
		status, buf = http.StatusInternalServerError, *bytes.NewBufferString(`{"error":{"code":"internal_error","message":"Internal server error"}}`)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	w.Write(bytes.TrimSuffix(buf.Bytes(), []byte("\n")))
}

// writeError never carries raw database or secret-bearing messages.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, map[string]map[string]string{"error": {"code": code, "message": message}})
}

func notFound(w http.ResponseWriter)  { writeError(w, http.StatusNotFound, "not_found", "Not found") }
func forbidden(w http.ResponseWriter) { writeError(w, http.StatusForbidden, "forbidden", "Forbidden") }
func badRequest(w http.ResponseWriter, msg string) {
	writeError(w, http.StatusBadRequest, "bad_request", msg)
}
func conflict(w http.ResponseWriter, msg string) { writeError(w, http.StatusConflict, "conflict", msg) }
func internal(w http.ResponseWriter, msg string, err error) {
	slog.Error(msg, "error", err)
	writeError(w, http.StatusInternalServerError, "internal_error", "Internal server error")
}

// ---------------------------------------------------------------------------
// Request decoding
// ---------------------------------------------------------------------------

// decodeJSON reads a JSON body. It answers 415 for a non-JSON content type,
// 400 for malformed JSON and 422 for a wrong shape, then returns false.
func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if mt != "application/json" && !(strings.HasPrefix(mt, "application/") && strings.HasSuffix(mt, "+json")) {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type", "Expected request with `Content-Type: application/json`")
		return false
	}
	body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxBody))
	if err != nil {
		writeError(w, http.StatusRequestEntityTooLarge, "payload_too_large", "Request body too large")
		return false
	}
	if err := json.Unmarshal(body, dst); err != nil {
		var syntax *json.SyntaxError
		if errors.As(err, &syntax) || !json.Valid(body) {
			badRequest(w, "Malformed JSON request body")
		} else {
			unprocessable(w)
		}
		return false
	}
	return true
}

func unprocessable(w http.ResponseWriter) {
	writeError(w, http.StatusUnprocessableEntity, "unprocessable_entity", "Request body has missing or invalid fields")
}

// required answers 422 and returns false when any required field pointer is nil.
func required(w http.ResponseWriter, fields ...any) bool {
	for _, f := range fields {
		if reflect.ValueOf(f).IsNil() {
			unprocessable(w)
			return false
		}
	}
	return true
}

// pathIDs parses the named UUID path parameters, answering 400 on failure.
func pathIDs(w http.ResponseWriter, r *http.Request, names ...string) ([]uuid.UUID, bool) {
	ids := make([]uuid.UUID, len(names))
	for i, name := range names {
		id, err := uuid.Parse(r.PathValue(name))
		if err != nil {
			badRequest(w, "Invalid path parameter: "+name)
			return nil, false
		}
		ids[i] = id
	}
	return ids, true
}

// queryUUID parses an optional UUID query parameter, answering 400 on failure.
func queryUUID(w http.ResponseWriter, r *http.Request, name string) (*uuid.UUID, bool) {
	q := r.URL.Query()
	if !q.Has(name) {
		return nil, true
	}
	id, err := uuid.Parse(q.Get(name))
	if err != nil {
		badRequest(w, "Invalid query parameter: "+name)
		return nil, false
	}
	return &id, true
}

func peerIP(r *http.Request) string {
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return "127.0.0.1"
	}
	return host
}

// ---------------------------------------------------------------------------
// Health and assets
// ---------------------------------------------------------------------------

func (s *Server) readiness(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), s.Timeout)
	defer cancel()
	if _, err := s.Pool.Exec(ctx, "SELECT 1"); err != nil {
		slog.Warn("database readiness check failed")
		writeError(w, http.StatusServiceUnavailable, "unavailable", "Service unavailable")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

var assetTypes = map[string]string{
	".html":  "text/html; charset=utf-8",
	".js":    "text/javascript; charset=utf-8",
	".css":   "text/css; charset=utf-8",
	".txt":   "text/plain; charset=utf-8",
	".woff2": "font/woff2",
	".svg":   "image/svg+xml",
}

func serveAsset(assets fs.FS, name string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		mimeType, ok := assetTypes[path.Ext(name)]
		if !ok || assets == nil {
			notFound(w)
			return
		}
		body, err := fs.ReadFile(assets, name)
		if err != nil {
			notFound(w)
			return
		}
		w.Header().Set("Content-Type", mimeType)
		w.Write(body)
	}
}

// ---------------------------------------------------------------------------
// Middleware
// ---------------------------------------------------------------------------

type routeKey struct{}

type statusWriter struct {
	http.ResponseWriter
	status int
}

func (w *statusWriter) WriteHeader(code int) {
	if w.status == 0 {
		w.status = code
	}
	w.ResponseWriter.WriteHeader(code)
}

func (w *statusWriter) Write(b []byte) (int, error) {
	if w.status == 0 {
		w.status = http.StatusOK
	}
	return w.ResponseWriter.Write(b)
}

// diagnostics logs each request by route pattern (never the raw path or query)
// and sets the security headers on every response.
func diagnostics(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("Cache-Control", "no-store")
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("Content-Security-Policy", csp)
		started := time.Now()
		sw := &statusWriter{ResponseWriter: w}
		route := "<unmatched>"
		next.ServeHTTP(sw, r.WithContext(context.WithValue(r.Context(), routeKey{}, &route)))
		slog.Info("request complete", "method", r.Method, "route", route, "status", sw.status, "elapsed_ms", time.Since(started).Milliseconds())
	})
}

// fallbackWriter replaces the mux's plain-text 404/405 bodies with the JSON error shape.
type fallbackWriter struct {
	http.ResponseWriter
	done bool
}

func (w *fallbackWriter) WriteHeader(code int) {
	w.done = true
	if code == http.StatusMethodNotAllowed {
		writeError(w.ResponseWriter, code, "method_not_allowed", "Method not allowed")
	} else {
		notFound(w.ResponseWriter)
	}
}

func (w *fallbackWriter) Write(b []byte) (int, error) {
	if !w.done {
		w.WriteHeader(http.StatusNotFound)
	}
	return len(b), nil
}

func unauthorized(w http.ResponseWriter) {
	writeError(w, http.StatusUnauthorized, "unauthorized", "Authentication required")
}

func extractUploadedImage(w http.ResponseWriter, r *http.Request) ([]byte, string, bool) {
	ct := r.Header.Get("Content-Type")
	var data []byte
	var mime string
	var err error
	if strings.HasPrefix(ct, "multipart/form-data") {
		if err := r.ParseMultipartForm(5 << 20); err != nil {
			badRequest(w, "Failed to parse upload form")
			return nil, "", false
		}
		file, header, ferr := r.FormFile("file")
		if ferr != nil {
			file, header, ferr = r.FormFile("image")
		}
		if ferr != nil {
			badRequest(w, "No image file provided in upload")
			return nil, "", false
		}
		defer file.Close()
		data, err = io.ReadAll(file)
		if err != nil {
			badRequest(w, "Failed to read uploaded image")
			return nil, "", false
		}
		mime = header.Header.Get("Content-Type")
	} else if strings.HasPrefix(ct, "application/json") {
		var req struct {
			Image string `json:"image"`
		}
		if !decodeJSON(w, r, &req) {
			return nil, "", false
		}
		raw := req.Image
		if idx := strings.Index(raw, ";base64,"); idx != -1 {
			if strings.HasPrefix(raw, "data:") {
				mime = raw[5:idx]
			}
			raw = raw[idx+8:]
		}
		b, err := base64.StdEncoding.DecodeString(raw)
		if err != nil {
			badRequest(w, "Invalid base64 image data")
			return nil, "", false
		}
		data = b
	} else if ct != "" && r.Body != nil {
		b, err := io.ReadAll(http.MaxBytesReader(w, r.Body, 5<<20))
		if err != nil {
			badRequest(w, "Failed to read image payload")
			return nil, "", false
		}
		data = b
		mime = ct
	} else {
		badRequest(w, "Unsupported upload Content-Type")
		return nil, "", false
	}

	if len(data) == 0 {
		badRequest(w, "Empty image uploaded")
		return nil, "", false
	}
	if len(data) > 5<<20 {
		badRequest(w, "Image exceeds maximum allowed size of 5MB")
		return nil, "", false
	}
	if mime == "" || mime == "application/octet-stream" {
		mime = http.DetectContentType(data)
	}
	if !strings.HasPrefix(mime, "image/") {
		badRequest(w, "Uploaded file is not a supported image")
		return nil, "", false
	}
	return data, mime, true
}

func jsonFallbacks(mux *http.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, pattern := mux.Handler(r)
		if route, ok := r.Context().Value(routeKey{}).(*string); ok && pattern != "" {
			*route = pattern
		}
		if pattern == "" {
			mux.ServeHTTP(&fallbackWriter{ResponseWriter: w}, r)
			return
		}
		mux.ServeHTTP(w, r)
	})
}
