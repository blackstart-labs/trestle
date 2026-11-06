package httpapi

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/maruf-pfc/trestle/internal/config"
	"github.com/maruf-pfc/trestle/internal/database"
)

// idNoDBApp is a router whose database is unreachable (port 1), as in the Rust http/assets tests.
func idNoDBApp(t *testing.T) *testApp {
	t.Helper()
	cfg, err := config.FromPairs(map[string]string{
		"DATABASE_URL":          "postgres://user:secret@127.0.0.1:1/trestle",
		"TRESTLE_DB_TIMEOUT_MS": "100",
	})
	if err != nil {
		t.Fatal(err)
	}
	pool := database.Pool(cfg)
	t.Cleanup(pool.Close)
	return &testApp{t: t, pool: pool, cfg: cfg, handler: Router(pool, cfg.DBTimeout, cfg, testAssets)}
}

func idWantHeaders(t *testing.T, res testResponse, want map[string]string) {
	t.Helper()
	for k, v := range want {
		if got := res.Header.Get(k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}
}

func TestLivenessSucceedsWithoutADatabase(t *testing.T) {
	res := idNoDBApp(t).anon().do("GET", "/health/live", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	idWantHeaders(t, res, map[string]string{
		"Content-Type": "application/json", "Cache-Control": "no-store", "X-Content-Type-Options": "nosniff",
	})
	if string(res.Body) != `{"status":"ok"}` {
		t.Errorf("body %s", res.Body)
	}
}

func TestUnavailableDatabaseReturnsBoundedSecretSafeReadinessFailure(t *testing.T) {
	app := idNoDBApp(t)
	done := make(chan testResponse, 1)
	go func() { done <- app.anon().do("GET", "/health/ready", nil) }()
	var res testResponse
	select {
	case res = <-done:
	case <-time.After(2 * time.Second):
		t.Fatal("readiness must be bounded")
	}
	if res.StatusCode != http.StatusServiceUnavailable {
		t.Fatalf("status %d", res.StatusCode)
	}
	idWantHeaders(t, res, map[string]string{"Cache-Control": "no-store"})
	if string(res.Body) != `{"error":{"code":"unavailable","message":"Service unavailable"}}` {
		t.Errorf("body %s", res.Body)
	}
}

func TestUnknownRoutesAndUnsupportedMethodsReturnConsistentErrors(t *testing.T) {
	for _, c := range []struct {
		method, path string
		status       int
		expected     string
	}{
		{"GET", "/unknown?secret=never-log-this", http.StatusNotFound, `{"error":{"code":"not_found","message":"Not found"}}`},
		{"POST", "/health/live", http.StatusMethodNotAllowed, `{"error":{"code":"method_not_allowed","message":"Method not allowed"}}`},
	} {
		res := idNoDBApp(t).anon().do(c.method, c.path, nil)
		if res.StatusCode != c.status {
			t.Errorf("%s %s: status %d", c.method, c.path, res.StatusCode)
		}
		idWantHeaders(t, res, map[string]string{"Content-Type": "application/json", "Cache-Control": "no-store"})
		if string(res.Body) != c.expected {
			t.Errorf("%s %s: body %s", c.method, c.path, res.Body)
		}
	}
}

func TestRequestDiagnosticsIncludeStatusButNotRawURLOrHeaders(t *testing.T) {
	app := idNoDBApp(t)
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	defer slog.SetDefault(previous)

	req := httptest.NewRequest("GET", "/private-path?token=private-query", nil)
	req.Header.Set("Authorization", "Bearer private-header")
	app.handler.ServeHTTP(httptest.NewRecorder(), req)

	out := logs.String()
	for _, want := range []string{"404", "<unmatched>"} {
		if !bytes.Contains(logs.Bytes(), []byte(want)) {
			t.Errorf("logs lack %q: %s", want, out)
		}
	}
	for _, secret := range []string{"private-path", "private-query", "private-header"} {
		if bytes.Contains(logs.Bytes(), []byte(secret)) {
			t.Errorf("logs contain %q: %s", secret, out)
		}
	}
}

func TestInternalErrorsHaveAFixedPublicResponse(t *testing.T) {
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	defer slog.SetDefault(previous)

	rec := httptest.NewRecorder()
	internal(rec, "test failure", bytes.ErrTooLarge)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("status %d", rec.Code)
	}
	if got := rec.Body.String(); got != `{"error":{"code":"internal_error","message":"Internal server error"}}` {
		t.Errorf("body %s", got)
	}
}
