package httpapi

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestProjectEditingPreservesTaskIdentityAndValidatesMetadata(t *testing.T) {
	app := newApp(t)
	_, wsID := ptBootstrap(app, "owner", "project-edit-pw", "Owner", "Engineering", "engineering")
	c := app.login("owner", "project-edit-pw")
	base := fmt.Sprintf("/api/v1/workspaces/%s/projects", wsID)
	create := func(name, slug, key string) map[string]any {
		return ptExpect(t, c.do("POST", base, map[string]any{"name": name, "slug": slug, "key": key, "description": "Original description"}), http.StatusCreated).JSON(t)
	}
	project := create("Core", "core", "CORE")
	create("Other", "other", "OTHER")
	path := base + "/" + project["id"].(string)
	original := ptExpect(t, c.do("GET", path, nil), http.StatusOK).JSON(t)
	task := ptExpect(t, c.do("POST", path+"/tasks", map[string]any{"title": "Keep my key"}), http.StatusCreated).JSON(t)
	updated := ptExpect(t, c.do("PATCH", path, map[string]any{"name": " Core Platform ", "slug": " PLATFORM ", "description": "Updated description"}), http.StatusOK).JSON(t)
	ptEq(t, "name", updated["name"], "Core Platform")
	ptEq(t, "slug", updated["slug"], "platform")
	ptEq(t, "description", updated["description"], "Updated description")
	ptEq(t, "key", updated["key"], "CORE")
	ptEq(t, "task_counter", updated["task_counter"], 1)
	ptEq(t, "created_at", updated["created_at"], original["created_at"])
	gotTask := ptExpect(t, c.do("GET", path+"/tasks/"+task["id"].(string), nil), http.StatusOK).JSON(t)
	ptEq(t, "human_key", gotTask["human_key"], task["human_key"])
	ptEq(t, "id", gotTask["id"], task["id"])
	ptExpect(t, c.do("PATCH", path, map[string]any{"name": "Must roll back", "slug": "other"}), http.StatusConflict)
	for _, patch := range []map[string]any{
		{}, {"name": " "}, {"name": strings.Repeat("x", 121)},
		{"slug": "x"}, {"slug": "-bad"}, {"slug": "bad-"}, {"slug": "bad slug"}, {"slug": strings.Repeat("x", 49)},
		{"description": strings.Repeat("x", 2001)}, {"name": "Must not save", "key": "NEW"},
	} {
		ptExpect(t, c.do("PATCH", path, patch), http.StatusBadRequest)
	}
	got := ptExpect(t, c.do("GET", path, nil), http.StatusOK).JSON(t)
	ptEq(t, "name", got["name"], "Core Platform")
	ptEq(t, "key", got["key"], "CORE")
	cleared := ptExpect(t, c.do("PATCH", path, map[string]any{"description": ""}), http.StatusOK).JSON(t)
	ptEq(t, "description", cleared["description"], "")
	ptEq(t, "name", cleared["name"], "Core Platform")
	ptEq(t, "slug", cleared["slug"], "platform")
	ptExpect(t, c.do("PATCH", base+"/"+uuid.NewString(), map[string]any{"name": "Missing"}), http.StatusNotFound)
	var audits int
	if err := app.pool.QueryRow(t.Context(), "SELECT count(*) FROM security_audit_events WHERE event_type = 'project.updated'").Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if audits != 2 {
		t.Fatalf("project.updated audits = %d, want 2 successful updates", audits)
	}
}

func TestProjectEditingAuthorizationAndIsolation(t *testing.T) {
	app := newApp(t)
	owner, wsID := ptBootstrap(app, "owner", "project-edit-pw", "Owner", "Engineering", "engineering")
	admin := app.createUser("admin", "project-edit-pw", "Admin")
	member := app.createUser("member", "project-edit-pw", "Member")
	outsider := app.createUser("outsider", "project-edit-pw", "Outsider")
	ptAddMember(app, owner.ID, wsID, admin.ID, "admin")
	ptAddMember(app, owner.ID, wsID, member.ID, "member")
	foreign := ptForeignWorkspace(app, outsider.ID, "Other workspace", "other-workspace")
	c := app.login("owner", "project-edit-pw")
	base := fmt.Sprintf("/api/v1/workspaces/%s/projects", wsID)
	project := ptExpect(t, c.do("POST", base, map[string]any{"name": "Core", "slug": "core", "key": "CORE"}), http.StatusCreated).JSON(t)
	path := base + "/" + project["id"].(string)
	patch := map[string]any{"name": "Updated"}
	ptExpect(t, app.anon().do("PATCH", path, patch), http.StatusUnauthorized)
	ptExpect(t, app.login("member", "project-edit-pw").do("PATCH", path, patch), http.StatusForbidden)
	ptExpect(t, app.login("outsider", "project-edit-pw").do("PATCH", path, patch), http.StatusNotFound)
	foreignPath := fmt.Sprintf("/api/v1/workspaces/%s/projects/%s", foreign, project["id"])
	ptExpect(t, app.login("outsider", "project-edit-pw").do("PATCH", foreignPath, patch), http.StatusNotFound)
	noCSRF := app.login("owner", "project-edit-pw")
	noCSRF.csrf = ""
	ptExpect(t, noCSRF.do("PATCH", path, patch), http.StatusForbidden)
	ptExpect(t, app.login("admin", "project-edit-pw").do("PATCH", path, patch), http.StatusOK)
}

func TestProjectImageUploadAndRetrieval(t *testing.T) {
	app := newApp(t)
	_, wsID := ptBootstrap(app, "owner", "project-edit-pw", "Owner", "Engineering", "engineering")
	c := app.login("owner", "project-edit-pw")
	base := fmt.Sprintf("/api/v1/workspaces/%s/projects", wsID)

	samplePNG := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="

	projectWithImg := ptExpect(t, c.do("POST", base, map[string]any{
		"name": "Image Project", "slug": "image-project", "key": "IMG", "image": samplePNG,
	}), http.StatusCreated).JSON(t)

	imgURL, ok := projectWithImg["image_url"].(string)
	if !ok || imgURL == "" {
		t.Fatalf("expected image_url to be present on created project, got %v", projectWithImg["image_url"])
	}

	path := base + "/" + projectWithImg["id"].(string)
	res := ptExpect(t, c.do("GET", path+"/image", nil), http.StatusOK)
	if ct := res.Header.Get("Content-Type"); ct != "image/png" {
		t.Fatalf("expected content-type image/png, got %s", ct)
	}

	updated := ptExpect(t, c.do("PATCH", path, map[string]any{
		"name": "Updated Image Project",
	}), http.StatusOK).JSON(t)
	if updated["image_url"] == nil {
		t.Fatalf("expected image_url to remain, got nil")
	}

	ptExpect(t, c.do("DELETE", path+"/image", nil), http.StatusOK)
	deleted := ptExpect(t, c.do("GET", path, nil), http.StatusOK).JSON(t)
	if deleted["image_url"] != nil {
		t.Fatalf("expected image_url to be nil after deletion, got %v", deleted["image_url"])
	}

	ptExpect(t, c.do("GET", path+"/image", nil), http.StatusNotFound)
}
