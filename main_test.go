package main

import (
	"bytes"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/maruf-pfc/trestle/internal/config"
	"github.com/maruf-pfc/trestle/internal/database"
	"github.com/maruf-pfc/trestle/internal/httpapi"
)

var configKeys = []string{
	"DATABASE_URL", "TRESTLE_BIND_ADDR", "TRESTLE_MAX_DB_CONNECTIONS", "TRESTLE_DB_TIMEOUT_MS",
	"TRESTLE_DEV_MODE", "TRESTLE_ARGON2_M_COST", "TRESTLE_ARGON2_T_COST", "TRESTLE_ARGON2_PARALLELISM",
}

// runWithEnv runs the executable's entry point in-process with only env set,
// and returns its error plus everything it logged (what main would print).
func runWithEnv(t *testing.T, env map[string]string, args ...string) (error, string) {
	t.Helper()
	for _, k := range configKeys {
		t.Setenv(k, "") // restores the original value when the test ends
		os.Unsetenv(k)
	}
	for k, v := range env {
		t.Setenv(k, v)
	}
	var logs bytes.Buffer
	previous := slog.Default()
	slog.SetDefault(slog.New(slog.NewJSONHandler(&logs, nil)))
	defer slog.SetDefault(previous)
	err := run(args)
	if err != nil {
		slog.Error("application stopped with an error", "message", err.Error())
	}
	return err, logs.String()
}

func TestExecutableRejectsBadEnvironmentWithoutLoggingSecretValues(t *testing.T) {
	err, diagnostics := runWithEnv(t, map[string]string{
		"DATABASE_URL":               "postgres://user:private-password@localhost/db",
		"TRESTLE_MAX_DB_CONNECTIONS": "private-password",
	})
	if err == nil {
		t.Fatal("run succeeded")
	}
	if !strings.Contains(diagnostics, "TRESTLE_MAX_DB_CONNECTIONS") {
		t.Errorf("diagnostics do not name the setting: %s", diagnostics)
	}
	if strings.Contains(diagnostics, "private-password") {
		t.Errorf("diagnostics leak the value: %s", diagnostics)
	}
}

func TestNonUnicodeConfigurationNamesTheSettingWithoutExposingBytes(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Windows environment strings are UTF-16 and cannot hold invalid UTF-8")
	}
	err, diagnostics := runWithEnv(t, map[string]string{"DATABASE_URL": "private-password\xff"})
	if err == nil {
		t.Fatal("run succeeded")
	}
	if !strings.Contains(diagnostics, "DATABASE_URL must be valid Unicode") {
		t.Errorf("diagnostics: %s", diagnostics)
	}
	if strings.Contains(diagnostics, "private-password") {
		t.Errorf("diagnostics leak the value: %s", diagnostics)
	}
}

func TestUnavailableDatabaseMakesMigrationCommandFailWithoutCredentials(t *testing.T) {
	err, diagnostics := runWithEnv(t, map[string]string{
		"DATABASE_URL":          "postgres://user:private-password@127.0.0.1:1/trestle",
		"TRESTLE_DB_TIMEOUT_MS": "100",
	}, "migrate")
	if err == nil {
		t.Fatal("migrate succeeded")
	}
	if !strings.Contains(diagnostics, "database migration failed or timed out") {
		t.Errorf("diagnostics: %s", diagnostics)
	}
	for _, secret := range []string{"private-password", "postgres://"} {
		if strings.Contains(diagnostics, secret) {
			t.Errorf("diagnostics contain %q: %s", secret, diagnostics)
		}
	}
}

// embeddedApp serves the real embedded web/dist with no reachable database.
func embeddedApp(t *testing.T) *httptest.Server {
	t.Helper()
	cfg, err := config.FromPairs(map[string]string{"DATABASE_URL": "postgres://user:secret@127.0.0.1:1/trestle"})
	if err != nil {
		t.Fatal(err)
	}
	assets, err := fs.Sub(dist, "web/dist")
	if err != nil {
		t.Fatal(err)
	}
	pool := database.Pool(cfg)
	t.Cleanup(pool.Close)
	srv := httptest.NewServer(httpapi.Router(pool, cfg.DBTimeout, cfg, assets))
	t.Cleanup(srv.Close)
	return srv
}

func fetch(t *testing.T, method, url string) (*http.Response, []byte) {
	t.Helper()
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		t.Fatal(err)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(res.Body)
	return res, body
}

const csp = "default-src 'none'; script-src 'self'; style-src 'self'; font-src 'self'; connect-src 'self'; base-uri 'none'; object-src 'none'; frame-ancestors 'none'; form-action 'none'"

func TestEmbeddedRootDeliversTheBuiltFrontendWithoutADatabase(t *testing.T) {
	srv := embeddedApp(t)
	res, body := fetch(t, "GET", srv.URL+"/")
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	for k, v := range map[string]string{
		"Content-Type": "text/html; charset=utf-8", "Cache-Control": "no-store", "X-Content-Type-Options": "nosniff",
	} {
		if got := res.Header.Get(k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}
	html := string(body)
	if !strings.Contains(html, "<title>Trestle</title>") || !strings.Contains(html, "/assets/") {
		t.Errorf("unexpected index.html: %s", html)
	}
}

func TestEmbeddedReferencedAssetsHaveCorrectTypesAndSecurityHeaders(t *testing.T) {
	srv := embeddedApp(t)
	_, body := fetch(t, "GET", srv.URL+"/")
	var paths []string
	for _, v := range strings.Split(string(body), `"`) {
		if strings.HasPrefix(v, "/assets/") {
			paths = append(paths, v)
		}
	}
	var js, css bool
	for _, p := range paths {
		js = js || strings.HasSuffix(p, ".js")
		css = css || strings.HasSuffix(p, ".css")
	}
	if !js || !css {
		t.Fatalf("index.html must reference a script and a stylesheet: %v", paths)
	}
	for _, p := range paths {
		res, body := fetch(t, "GET", srv.URL+p)
		if res.StatusCode != http.StatusOK {
			t.Fatalf("%s: status %d", p, res.StatusCode)
		}
		wantType := "text/css; charset=utf-8"
		if strings.HasSuffix(p, ".js") {
			wantType = "text/javascript; charset=utf-8"
		}
		for k, v := range map[string]string{
			"Content-Type": wantType, "Cache-Control": "no-store", "X-Content-Type-Options": "nosniff",
			"Content-Security-Policy": csp,
		} {
			if got := res.Header.Get(k); got != v {
				t.Errorf("%s: %s = %q, want %q", p, k, got, v)
			}
		}
		if len(body) == 0 {
			t.Errorf("%s: empty body", p)
		}
		head, headBody := fetch(t, "HEAD", srv.URL+p)
		if head.StatusCode != http.StatusOK || len(headBody) != 0 {
			t.Errorf("%s: HEAD %d with %d bytes", p, head.StatusCode, len(headBody))
		}
		if post, _ := fetch(t, "POST", srv.URL+p); post.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("%s: POST %d", p, post.StatusCode)
		}
	}
}
