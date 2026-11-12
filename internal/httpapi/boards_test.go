package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/google/uuid"
)

func TestBoardLifecycleIsolationAndConflicts(t *testing.T) {
	app := newApp(t)
	_, ws := ptBootstrap(app, "owner", "board-password", "Owner", "Studio", "studio")
	c := app.login("owner", "board-password")
	base := fmt.Sprintf("/api/v1/workspaces/%s/boards", ws)
	scene := map[string]any{"version": 1, "elements": []any{map[string]any{"id": "shape", "kind": "rectangle", "x": 40, "y": 40, "width": 200, "height": 100, "text": "Review", "color": "ink"}}}
	body := map[string]any{"title": "Release flow", "kind": "diagram", "scene": scene, "creation_key": uuid.NewString()}
	ptExpect(t, app.anon().do("GET", base, nil), http.StatusUnauthorized)
	ptExpect(t, (&client{app: app, cookies: c.cookies}).do("POST", base, body), http.StatusForbidden)
	created := ptExpect(t, c.do("POST", base, body), http.StatusCreated).JSON(t)
	id := created["id"].(string)
	ptEq(t, "version", created["version"], 1)
	// A lost creation response can be retried without another board or audit event.
	replayed := ptExpect(t, c.do("POST", base, body), http.StatusCreated).JSON(t)
	ptEq(t, "replayed id", replayed["id"], id)
	var auditCount int
	if err := app.pool.QueryRow(context.Background(), "SELECT count(*) FROM security_audit_events WHERE event_type='board.created' AND metadata->>'board_id'=$1", id).Scan(&auditCount); err != nil {
		t.Fatal(err)
	}
	if auditCount != 1 {
		t.Fatalf("creation retry recorded %d audit events", auditCount)
	}
	list := ptExpect(t, c.do("GET", base, nil), http.StatusOK).JSON(t)
	if len(list["boards"].([]any)) != 1 {
		t.Fatal("duplicate board on creation retry")
	}
	if list["boards"].([]any)[0].(map[string]any)["scene"] != nil {
		t.Fatal("list includes full scene")
	}
	ptExpect(t, c.do("GET", base+"?offset=-1", nil), http.StatusBadRequest)
	ptExpect(t, c.do("POST", base, map[string]any{"title": "Invalid", "kind": "iframe", "scene": scene}), http.StatusBadRequest)
	path := base + "/" + id
	ptExpect(t, c.do("GET", path, nil), http.StatusOK)
	outsider := app.createUser("outsider", "other-password", "Outsider")
	other := app.login("outsider", "other-password")
	ptExpect(t, other.do("GET", path, nil), http.StatusNotFound)
	otherWS := ptExpect(t, other.do("POST", "/api/v1/workspaces", map[string]any{"name": "Other", "slug": "other"}), http.StatusCreated).JSON(t)
	ptExpect(t, other.do("GET", "/api/v1/workspaces/"+otherWS["id"].(string)+"/boards/"+id, nil), http.StatusNotFound)
	// Add the same user as a guest; read/export is allowed, writes are not.
	ptExpect(t, c.do("POST", fmt.Sprintf("/api/v1/workspaces/%s/members", ws), map[string]any{"username": outsider.Username, "role": "guest"}), http.StatusCreated)
	ptExpect(t, other.do("GET", path, nil), http.StatusOK)
	ptExpect(t, other.do("PUT", path, body), http.StatusForbidden)
	ptExpect(t, other.do("POST", base, body), http.StatusForbidden)
	ptExpect(t, c.do("PATCH", fmt.Sprintf("/api/v1/workspaces/%s/members/%s", ws, outsider.ID), map[string]any{"role": "member"}), http.StatusOK)
	ptExpect(t, other.do("PUT", path, map[string]any{"title": "Release flow", "kind": "diagram", "scene": scene, "version": 1, "archived": true}), http.StatusForbidden)
	body["version"] = 1
	body["title"] = "Updated flow"
	updated := ptExpect(t, c.do("PUT", path, body), http.StatusOK).JSON(t)
	ptEq(t, "updated version", updated["version"], 2)
	ptExpect(t, c.do("PUT", path, body), http.StatusConflict)
	ptEq(t, "conflict did not overwrite", ptExpect(t, c.do("GET", path, nil), http.StatusOK).JSON(t)["title"], "Updated flow")
	body["version"] = 2
	body["archived"] = true
	ptExpect(t, c.do("PUT", path, body), http.StatusOK)
	active := ptExpect(t, c.do("GET", base, nil), http.StatusOK).JSON(t)
	if len(active["boards"].([]any)) != 0 {
		t.Fatal("archive stayed active")
	}
	archived := ptExpect(t, c.do("GET", base+"?archived=true", nil), http.StatusOK).JSON(t)
	if len(archived["boards"].([]any)) != 1 {
		t.Fatal("missing archived board")
	}
	exported := ptExpect(t, c.do("GET", fmt.Sprintf("/api/v1/workspaces/%s/export", ws), nil), http.StatusOK).JSON(t)
	exportedBoards := exported["boards"].([]any)
	if len(exportedBoards) != 1 || exportedBoards[0].(map[string]any)["scene"] == nil {
		t.Fatal("Workspace export omitted the archived board scene")
	}
	body["version"] = 3
	ptExpect(t, c.do("PUT", path, body), http.StatusConflict)
	body["archived"] = false
	ptExpect(t, c.do("PUT", path, body), http.StatusOK)
	// A failed history write rolls the board mutation back.
	_, err := app.pool.Exec(context.Background(), `CREATE FUNCTION fail_board_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.event_type='board.updated' THEN RAISE EXCEPTION 'test audit failure'; END IF; RETURN NEW; END $$; CREATE TRIGGER fail_board_audit BEFORE INSERT ON security_audit_events FOR EACH ROW EXECUTE FUNCTION fail_board_audit()`)
	if err != nil {
		t.Fatal(err)
	}
	body["version"] = 4
	body["title"] = "Must roll back"
	ptExpect(t, c.do("PUT", path, body), http.StatusInternalServerError)
	got := ptExpect(t, c.do("GET", path, nil), http.StatusOK).JSON(t)
	ptEq(t, "rollback title", got["title"], "Updated flow")
	ptEq(t, "rollback version", got["version"], 4)
}

func TestConcurrentBoardSaves(t *testing.T) {
	app := newApp(t)
	_, ws := ptBootstrap(app, "owner", "board-password", "Owner", "Studio", "studio")
	c := app.login("owner", "board-password")
	base := fmt.Sprintf("/api/v1/workspaces/%s/boards", ws)
	body := map[string]any{"title": "Sketch", "kind": "whiteboard", "scene": map[string]any{"version": 1, "elements": []any{}}, "creation_key": uuid.NewString()}
	b := ptExpect(t, c.do("POST", base, body), http.StatusCreated).JSON(t)
	body["version"] = 1
	results := make(chan int, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); results <- c.do("PUT", base+"/"+b["id"].(string), body).StatusCode }()
	}
	wg.Wait()
	close(results)
	counts := map[int]int{}
	for code := range results {
		counts[code]++
	}
	if counts[http.StatusOK] != 1 || counts[http.StatusConflict] != 1 {
		t.Fatalf("concurrent results: %v", counts)
	}
}
