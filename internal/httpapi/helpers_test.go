package httpapi

// Shared helpers for the httpapi tests (ported from the Rust integration tests).

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/maruf-pfc/trestle/internal/config"
	"github.com/maruf-pfc/trestle/internal/identity"
	"github.com/maruf-pfc/trestle/internal/testdb"
)

// testAssets stands in for the built frontend.
var testAssets = fstest.MapFS{
	"index.html":         {Data: []byte("<!doctype html><title>Trestle</title>")},
	"assets/index-x.js":  {Data: []byte("console.log(1)")},
	"assets/index-x.css": {Data: []byte("body{}")},
}

type testApp struct {
	t       *testing.T
	pool    *pgxpool.Pool
	cfg     config.Config
	handler http.Handler
}

// newApp returns a router on a fresh, migrated database.
func newApp(t *testing.T) *testApp {
	t.Helper()
	pool, cfg := testdb.Pool(t, true)
	return &testApp{t: t, pool: pool, cfg: cfg, handler: Router(pool, 2*time.Second, cfg, testAssets)}
}

type testResponse struct {
	*http.Response
	Body []byte
}

// JSON decodes the response body into a generic value.
func (r testResponse) JSON(t *testing.T) map[string]any {
	t.Helper()
	var v map[string]any
	if err := json.Unmarshal(r.Body, &v); err != nil {
		t.Fatalf("decode %q: %v", r.Body, err)
	}
	return v
}

// client carries a signed-in session's cookies and CSRF token.
type client struct {
	app     *testApp
	cookies string
	csrf    string
}

// do sends a request; body is JSON-encoded unless nil.
func (c *client) do(method, path string, body any) testResponse {
	c.app.t.Helper()
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			c.app.t.Fatal(err)
		}
		rdr = bytes.NewReader(b)
	}
	req := httptest.NewRequest(method, path, rdr)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	if c.cookies != "" {
		req.Header.Set("Cookie", c.cookies)
	}
	if c.csrf != "" {
		req.Header.Set("X-CSRF-Token", c.csrf)
	}
	rec := httptest.NewRecorder()
	c.app.handler.ServeHTTP(rec, req)
	res := rec.Result()
	b, _ := io.ReadAll(res.Body)
	return testResponse{res, b}
}

// anon is an unauthenticated client.
func (a *testApp) anon() *client { return &client{app: a} }

// createUser provisions a user directly through the store.
func (a *testApp) createUser(username, password, displayName string) identity.UserProjection {
	a.t.Helper()
	hasher, _ := identity.NewHasher(a.cfg.Argon2MCost, a.cfg.Argon2TCost, a.cfg.Argon2Parallelism)
	hash, err := hasher.Hash([]byte(password))
	if err != nil {
		a.t.Fatal(err)
	}
	u, err := identity.CreateUser(context.Background(), a.pool, username, hash, displayName)
	if err != nil {
		a.t.Fatal(err)
	}
	return u
}

// login signs in over HTTP and returns a client holding the session cookie and CSRF token.
func (a *testApp) login(username, password string) *client {
	a.t.Helper()
	res := a.anon().do("POST", "/api/v1/session", map[string]string{"username": username, "password": password})
	if res.StatusCode != http.StatusOK {
		a.t.Fatalf("login %s: %d %s", username, res.StatusCode, res.Body)
	}
	c := &client{app: a}
	var parts []string
	for _, sc := range res.Header.Values("Set-Cookie") {
		pair, _, _ := strings.Cut(sc, ";")
		parts = append(parts, pair)
		if v, ok := strings.CutPrefix(pair, "trestle-csrf="); ok {
			c.csrf = v
		}
	}
	c.cookies = strings.Join(parts, "; ")
	return c
}

// extractCookie returns the value of name from a Set-Cookie header value.
func extractCookie(setCookie, name string) (string, bool) {
	pair, _, _ := strings.Cut(setCookie, ";")
	return strings.CutPrefix(strings.TrimSpace(pair), name+"=")
}
