package httpapi

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// idServe runs the handler on a real server so HEAD bodies are dropped as in production.
func idServe(t *testing.T, app *testApp, method, path string) (*http.Response, []byte) {
	t.Helper()
	srv := httptest.NewServer(app.handler)
	defer srv.Close()
	req, err := http.NewRequest(method, srv.URL+path, nil)
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

// The embedded production build is checked in the root package (main_test.go);
// here testAssets stands in for it.
func TestReferencedAssetsHaveCorrectTypesAndSecurityHeaders(t *testing.T) {
	app := idNoDBApp(t)
	for path, wantType := range map[string]string{
		"/assets/index-x.js":  "text/javascript; charset=utf-8",
		"/assets/index-x.css": "text/css; charset=utf-8",
	} {
		res := app.anon().do("GET", path, nil)
		if res.StatusCode != http.StatusOK {
			t.Fatalf("%s: status %d", path, res.StatusCode)
		}
		idWantHeaders(t, res, map[string]string{
			"Content-Type":            wantType,
			"Cache-Control":           "no-store",
			"X-Content-Type-Options":  "nosniff",
			"Content-Security-Policy": csp,
		})
		if len(res.Body) == 0 {
			t.Errorf("%s: empty body", path)
		}
		head, body := idServe(t, app, "HEAD", path)
		if head.StatusCode != http.StatusOK || len(body) != 0 {
			t.Errorf("%s: HEAD %d with %d bytes", path, head.StatusCode, len(body))
		}
		if post := app.anon().do("POST", path, nil); post.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("%s: POST %d", path, post.StatusCode)
		}
	}
}

func TestUnknownPathsNeverFallBackToHTML(t *testing.T) {
	app := idNoDBApp(t)
	for _, path := range []string{
		"/assets/missing.js",
		"/api/v1/nonexistent",
		"/src/App.svelte",
		"/assets/../index.html",
		"/assets/%2e%2e/index.html",
		"/assets/%2fetc/passwd",
		"/index.html",
	} {
		res := app.anon().do("GET", path, nil)
		if res.StatusCode != http.StatusNotFound {
			t.Errorf("%s: status %d", path, res.StatusCode)
		}
		if got := res.Header.Get("Content-Type"); got != "application/json" {
			t.Errorf("%s: content type %q", path, got)
		}
		if string(res.Body) != `{"error":{"code":"not_found","message":"Not found"}}` {
			t.Errorf("%s: body %s", path, res.Body)
		}
	}
}

func TestRootSupportsHeadButNotMutation(t *testing.T) {
	app := idNoDBApp(t)
	head, body := idServe(t, app, "HEAD", "/")
	if head.StatusCode != http.StatusOK || len(body) != 0 {
		t.Errorf("HEAD / %d with %d bytes", head.StatusCode, len(body))
	}
	if post := app.anon().do("POST", "/", nil); post.StatusCode != http.StatusMethodNotAllowed {
		t.Errorf("POST / %d", post.StatusCode)
	}
}

func TestRootDeliversTheBuiltFrontendWithoutADatabase(t *testing.T) {
	res := idNoDBApp(t).anon().do("GET", "/", nil)
	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d", res.StatusCode)
	}
	idWantHeaders(t, res, map[string]string{
		"Content-Type": "text/html; charset=utf-8", "Cache-Control": "no-store", "X-Content-Type-Options": "nosniff",
	})
	if !strings.Contains(string(res.Body), "<title>Trestle</title>") {
		t.Errorf("body %s", res.Body)
	}
}
