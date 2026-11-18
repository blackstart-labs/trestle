package httpapi

import (
	"context"
	"fmt"
	"github.com/google/uuid"
	"net/http"
	"sync"
	"testing"
)

func TestDocumentsLifecycleSecurityHistoryAndHierarchy(t *testing.T) {
	app := newApp(t)
	_, ws := ptBootstrap(app, "owner", "document-password", "Owner", "Knowledge", "knowledge")
	c := app.login("owner", "document-password")
	base := fmt.Sprintf("/api/v1/workspaces/%s/documents", ws)
	body := map[string]any{"title": "Architecture", "content": "  # Design\n\nOriginal body  ", "tags": []string{"design"}, "creation_key": uuid.NewString()}
	ptExpect(t, app.anon().do("GET", base, nil), 401)
	ptExpect(t, (&client{app: app, cookies: c.cookies}).do("POST", base, body), 403)
	d := ptExpect(t, c.do("POST", base, body), 201).JSON(t)
	id := d["id"].(string)
	path := base + "/" + id
	replay := ptExpect(t, c.do("POST", base, body), 201).JSON(t)
	ptEq(t, "replay", replay["id"], id)
	ptEq(t, "whitespace", d["content"], body["content"])
	child := ptExpect(t, c.do("POST", base, map[string]any{"title": "Child", "content": "[[" + id + "|Architecture]]", "parent_id": id, "creation_key": uuid.NewString()}), 201).JSON(t)
	back := ptExpect(t, c.do("GET", path+"/backlinks", nil), 200).JSON(t)
	if len(back["documents"].([]any)) != 1 {
		t.Fatal("missing backlink")
	}
	body["parent_id"] = child["id"]
	body["version"] = 1
	ptExpect(t, c.do("PUT", path, body), 400)
	delete(body, "parent_id")
	body["title"] = "Updated architecture"
	body["content"] = "New body"
	ptExpect(t, c.do("PUT", path, body), 200)
	ptExpect(t, c.do("PUT", path, body), 409)
	first := ptExpect(t, c.do("GET", path+"/versions/1", nil), 200).JSON(t)
	ptEq(t, "old revision", first["content"], "  # Design\n\nOriginal body  ")
	versions := ptExpect(t, c.do("GET", path+"/versions", nil), 200).JSON(t)
	if len(versions["documents"].([]any)) != 2 {
		t.Fatal("incorrect revision count")
	}
	ptExpect(t, c.do("GET", base+"?offset=-1", nil), 400)
	ptExpect(t, c.do("GET", base+"?archived=garbage", nil), 400)
	search := ptExpect(t, c.do("GET", base+"?q=New%20body&tag=design", nil), 200).JSON(t)
	if len(search["documents"].([]any)) != 1 {
		t.Fatal("search failed")
	}
	otherUser := app.createUser("outsider", "document-password", "Outsider")
	other := app.login("outsider", "document-password")
	otherWS := ptExpect(t, other.do("POST", "/api/v1/workspaces", map[string]any{"name": "Other", "slug": "other"}), 201).JSON(t)
	for _, suffix := range []string{"", "/versions", "/versions/1", "/backlinks"} {
		ptExpect(t, other.do("GET", path+suffix, nil), 404)
		ptExpect(t, other.do("GET", "/api/v1/workspaces/"+otherWS["id"].(string)+"/documents/"+id+suffix, nil), 404)
	}
	ptExpect(t, other.do("POST", "/api/v1/workspaces/"+otherWS["id"].(string)+"/documents", map[string]any{"title": "Bad parent", "parent_id": id, "creation_key": uuid.NewString()}), 400)
	ptExpect(t, c.do("POST", fmt.Sprintf("/api/v1/workspaces/%s/members", ws), map[string]any{"username": otherUser.Username, "role": "guest"}), 201)
	ptExpect(t, other.do("GET", path, nil), 200)
	ptExpect(t, other.do("PUT", path, body), 403)
	ptExpect(t, other.do("POST", base, body), 403)
	body["version"] = 2
	body["archived"] = true
	ptExpect(t, c.do("PUT", path, body), 200)
	active := ptExpect(t, c.do("GET", base, nil), 200).JSON(t)
	if len(active["documents"].([]any)) != 1 {
		t.Fatal("archive should retain child")
	}
	export := ptExpect(t, c.do("GET", fmt.Sprintf("/api/v1/workspaces/%s/export", ws), nil), 200).JSON(t)
	if len(export["documents"].([]any)) != 2 {
		t.Fatal("export omitted pages")
	}
	body["version"] = 3
	ptExpect(t, c.do("PUT", path, body), 409)
	body["archived"] = false
	ptExpect(t, c.do("PUT", path, body), 200)
	// Audit failure must roll back content, revision and links together.
	_, err := app.pool.Exec(context.Background(), `CREATE FUNCTION fail_doc_audit() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN IF NEW.event_type='document.updated' THEN RAISE EXCEPTION 'test'; END IF; RETURN NEW; END $$; CREATE TRIGGER fail_doc_audit BEFORE INSERT ON security_audit_events FOR EACH ROW EXECUTE FUNCTION fail_doc_audit()`)
	if err != nil {
		t.Fatal(err)
	}
	body["version"] = 4
	body["title"] = "Must roll back"
	ptExpect(t, c.do("PUT", path, body), 500)
	ptEq(t, "rollback", ptExpect(t, c.do("GET", path, nil), 200).JSON(t)["version"], 4)
}
func TestDocumentConcurrentEdits(t *testing.T) {
	app := newApp(t)
	_, ws := ptBootstrap(app, "owner", "document-password", "Owner", "Knowledge", "knowledge")
	c := app.login("owner", "document-password")
	base := fmt.Sprintf("/api/v1/workspaces/%s/documents", ws)
	body := map[string]any{"title": "Page", "creation_key": uuid.NewString()}
	d := ptExpect(t, c.do("POST", base, body), 201).JSON(t)
	body["version"] = 1
	results := make(chan int, 2)
	var wg sync.WaitGroup
	for range 2 {
		wg.Add(1)
		go func() { defer wg.Done(); results <- c.do("PUT", base+"/"+d["id"].(string), body).StatusCode }()
	}
	wg.Wait()
	close(results)
	counts := map[int]int{}
	for code := range results {
		counts[code]++
	}
	if counts[http.StatusOK] != 1 || counts[http.StatusConflict] != 1 {
		t.Fatalf("concurrency %v", counts)
	}
}

func TestDocumentPaginationAndRestore(t *testing.T) {
	app := newApp(t)
	_, ws := ptBootstrap(app, "owner", "document-password", "Owner", "Knowledge", "knowledge")
	c := app.login("owner", "document-password")
	base := fmt.Sprintf("/api/v1/workspaces/%s/documents", ws)
	parent := ptExpect(t, c.do("POST", base, map[string]any{"title": "Parent", "creation_key": uuid.NewString()}), 201).JSON(t)
	for i := 0; i < 51; i++ {
		ptExpect(t, c.do("POST", base, map[string]any{"title": fmt.Sprintf("Child %02d", i), "parent_id": parent["id"], "creation_key": uuid.NewString()}), 201)
	}
	children := base + "?parent_id=" + parent["id"].(string)
	first := ptExpect(t, c.do("GET", children, nil), 200).JSON(t)
	if len(first["documents"].([]any)) != 50 || first["has_more"] != true {
		t.Fatal("first page incorrect")
	}
	second := ptExpect(t, c.do("GET", children+"&offset=50", nil), 200).JSON(t)
	if len(second["documents"].([]any)) != 1 || second["has_more"] != false {
		t.Fatal("second page incorrect")
	}
	path := base + "/" + parent["id"].(string)
	parent["content"] = "Changed"
	ptExpect(t, c.do("PUT", path, parent), 200)
	revision := ptExpect(t, c.do("GET", path+"/versions/1", nil), 200).JSON(t)
	revision["version"] = 2
	restored := ptExpect(t, c.do("PUT", path, revision), 200).JSON(t)
	ptEq(t, "restored content", restored["content"], "")
	ptEq(t, "new version", restored["version"], 3)
}

func TestDocumentDelete(t *testing.T) {
	app := newApp(t)
	_, ws := ptBootstrap(app, "owner", "doc-password", "Owner", "Studio", "studio")
	c := app.login("owner", "doc-password")
	base := fmt.Sprintf("/api/v1/workspaces/%s/documents", ws)
	parent := ptExpect(t, c.do("POST", base, map[string]any{"title": "Parent Doc", "creation_key": uuid.NewString()}), 201).JSON(t)
	child := ptExpect(t, c.do("POST", base, map[string]any{"title": "Child Doc", "parent_id": parent["id"], "creation_key": uuid.NewString()}), 201).JSON(t)

	// Delete parent; child should be re-parented to nil
	ptExpect(t, c.do("DELETE", base+"/"+parent["id"].(string), nil), 204)
	ptExpect(t, c.do("GET", base+"/"+parent["id"].(string), nil), 404)

	childDoc := ptExpect(t, c.do("GET", base+"/"+child["id"].(string), nil), 200).JSON(t)
	if childDoc["parent_id"] != nil {
		t.Fatalf("expected child to have nil parent_id after parent deletion, got %v", childDoc["parent_id"])
	}

	ptExpect(t, c.do("DELETE", base+"/"+child["id"].(string), nil), 204)
	ptExpect(t, c.do("GET", base+"/"+child["id"].(string), nil), 404)
}
