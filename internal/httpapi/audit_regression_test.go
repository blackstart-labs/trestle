package httpapi

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/maruf-pfc/trestle/internal/projects"
)

func TestWorkspaceCreationAndEditing(t *testing.T) {
	a := newApp(t)
	owner, original := ptBootstrap(a, "owner", "audit-password", "Owner", "Original", "original")
	c := a.login("owner", "audit-password")
	base := "/api/v1/workspaces"
	body := map[string]string{"name": " New workspace ", "slug": " NEW-WORKSPACE "}
	ptExpect(t, a.anon().do("POST", base, body), http.StatusUnauthorized)
	noCSRF := *c
	noCSRF.csrf = ""
	ptExpect(t, noCSRF.do("POST", base, body), http.StatusForbidden)
	ws := ptExpect(t, c.do("POST", base, body), http.StatusCreated).JSON(t)
	ptEq(t, "role", ws["role"], "owner")
	ptEq(t, "name", ws["name"], "New workspace")
	ptEq(t, "slug", ws["slug"], "new-workspace")
	ptExpect(t, c.do("POST", base, body), http.StatusConflict)
	path := base + "/" + ws["id"].(string)
	updated := ptExpect(t, c.do("PATCH", path, map[string]string{"name": "Renamed", "slug": "renamed"}), http.StatusOK).JSON(t)
	ptEq(t, "role", updated["role"], "owner")
	ptEq(t, "name", updated["name"], "Renamed")
	ptExpect(t, c.do("PATCH", path, map[string]string{"name": "Renamed", "slug": "original"}), http.StatusConflict)
	for _, slug := range []string{"x", "bad slug", "-bad", "bad-"} {
		ptExpect(t, c.do("PATCH", path, map[string]string{"name": "Renamed", "slug": slug}), http.StatusBadRequest)
	}
	member := a.createUser("member", "audit-password", "Member")
	ptAddMember(a, owner.ID, original, member.ID, "member")
	m := a.login("member", "audit-password")
	ptExpect(t, m.do("PATCH", fmt.Sprintf("%s/%s", base, original), body), http.StatusForbidden)
	ptExpect(t, m.do("PATCH", path, body), http.StatusNotFound)
}

func TestTaskCreationMetadataDescriptionAndSubtaskCounts(t *testing.T) {
	a := newApp(t)
	owner, ws := ptBootstrap(a, "owner", "audit-password", "Owner", "Workspace", "workspace")
	c := a.login("owner", "audit-password")
	base := fmt.Sprintf("/api/v1/workspaces/%s/projects", ws)
	p := ptExpect(t, c.do("POST", base, map[string]string{"name": "Core", "slug": "core", "key": "CORE"}), http.StatusCreated).JSON(t)
	projectPath := base + "/" + p["id"].(string)
	m := ptExpect(t, c.do("POST", projectPath+"/milestones", map[string]string{"title": "Release"}), http.StatusCreated).JSON(t)
	path := projectPath + "/tasks"
	description := strings.Repeat("界", 15000)
	created := ptExpect(t, c.do("POST", path, map[string]any{"title": "Parent", "description": description, "milestone_id": m["id"], "assignee_ids": []string{owner.ID.String()}}), http.StatusCreated).JSON(t)
	ptEq(t, "description", created["description"], description)
	ptEq(t, "milestone_id", created["milestone_id"], m["id"])
	ptLen(t, "assignees", created["assignees"], 1)
	taskPath := path + "/" + created["id"].(string)
	ptExpect(t, c.do("PATCH", taskPath, map[string]string{"description": description}), http.StatusOK)
	ptExpect(t, c.do("PATCH", taskPath, map[string]string{"description": description + "x"}), http.StatusBadRequest)
	ptExpect(t, c.do("POST", path, map[string]string{"title": "Too long", "description": description + "x"}), http.StatusBadRequest)
	for range 2 {
		ptExpect(t, c.do("POST", taskPath+"/subtasks", map[string]string{"title": "Child"}), http.StatusCreated)
	}
	parent := ptExpect(t, c.do("GET", taskPath, nil), http.StatusOK).JSON(t)
	ptEq(t, "subtask_count", parent["subtask_count"], 2)
	page := ptExpect(t, c.do("GET", path+"?limit=1&assignee_id="+owner.ID.String(), nil), http.StatusOK).JSON(t)
	rows := ptLen(t, "tasks", page["tasks"], 1)
	ptEq(t, "subtask_count", rows[0].(map[string]any)["subtask_count"], 2)
	ptExpect(t, c.do("POST", path, map[string]any{"title": "Invalid assignee", "assignee_ids": []string{uuid.NewString()}}), http.StatusNotFound)
	ptExpect(t, c.do("POST", path, map[string]any{"title": "Invalid status", "status_id": uuid.NewString()}), http.StatusNotFound)
	page = ptExpect(t, c.do("GET", path, nil), http.StatusOK).JSON(t)
	ptEq(t, "total", page["total"], 3)
}

func TestExportsIncludeEveryTaskAndArchivedProjects(t *testing.T) {
	a := newApp(t)
	owner, ws := ptBootstrap(a, "owner", "audit-password", "Owner", "Workspace", "workspace")
	p, err := projects.CreateProject(t.Context(), a.pool, &owner.ID, ws, projects.CreateProjectParams{Name: "Core", Slug: "core", Key: "CORE"})
	if err != nil {
		t.Fatal(err)
	}
	for i := range 105 {
		_, err := projects.CreateTask(t.Context(), a.pool, &owner.ID, ws, p.ID, projects.CreateTaskParams{Title: fmt.Sprintf("Task %d", i), Status: "unstarted", Priority: "none"})
		if err != nil {
			t.Fatal(err)
		}
	}
	c := a.login("owner", "audit-password")
	base := fmt.Sprintf("/api/v1/workspaces/%s", ws)
	path := fmt.Sprintf("%s/projects/%s", base, p.ID)
	out := ptExpect(t, c.do("GET", path+"/export", nil), http.StatusOK).JSON(t)
	ptLen(t, "tasks", out["tasks"], 105)
	ptExpect(t, c.do("POST", path+"/archive", nil), http.StatusOK)
	out = ptExpect(t, c.do("GET", base+"/export", nil), http.StatusOK).JSON(t)
	list := ptLen(t, "projects", out["projects"], 1)
	ptLen(t, "tasks", list[0].(map[string]any)["tasks"], 105)
}

func TestGuestsCannotMutateTasks(t *testing.T) {
	a := newApp(t)
	owner, ws := ptBootstrap(a, "owner", "audit-password", "Owner", "Workspace", "workspace")
	guest := a.createUser("guest", "audit-password", "Guest")
	ptAddMember(a, owner.ID, ws, guest.ID, "guest")
	p, err := projects.CreateProject(t.Context(), a.pool, &owner.ID, ws, projects.CreateProjectParams{Name: "Core", Slug: "core", Key: "CORE"})
	if err != nil {
		t.Fatal(err)
	}
	task, err := projects.CreateTask(t.Context(), a.pool, &owner.ID, ws, p.ID, projects.CreateTaskParams{Title: "Task", Status: "unstarted", Priority: "none"})
	if err != nil {
		t.Fatal(err)
	}
	base := fmt.Sprintf("/api/v1/workspaces/%s/projects/%s/tasks", ws, p.ID)
	c := a.login("guest", "audit-password")
	ptExpect(t, c.do("GET", base, nil), http.StatusOK)
	ptExpect(t, c.do("POST", base, map[string]string{"title": "Not allowed"}), http.StatusForbidden)
	path := base + "/" + task.ID.String()
	ptExpect(t, c.do("PATCH", path, map[string]string{"title": "Not allowed"}), http.StatusForbidden)
	ptExpect(t, c.do("POST", path+"/time-logs", map[string]any{"hours": 1}), http.StatusForbidden)
	ptExpect(t, c.do("POST", path+"/assignees", map[string]any{"user_id": guest.ID}), http.StatusForbidden)
	ptExpect(t, c.do("POST", path+"/comments", map[string]string{"content": "Allowed comment"}), http.StatusCreated)
}

func TestNotificationDescriptionsAreReadableAndBounded(t *testing.T) {
	for source, want := range map[string]string{
		`<p><span style="color:red">122. Buy &amp; sell</span></p><p>266. Palindrome</p>`: "122. Buy & sell\n266. Palindrome",
		"- First\n- Second": "- First\n- Second",
		"Less than < 5":     "Less than < 5",
		"<p>Safe</p><script>bad()</script><p>Text</p>": "Safe\nText",
	} {
		if got := notificationText(source); got != want {
			t.Errorf("notificationText(%q) = %q, want %q", source, got, want)
		}
	}
	msg := formatTaskNotification("telegram", projects.Project{Name: "Project"}, projects.Task{Title: "Title", Description: strings.Repeat("界", 15000)}, "created", "https://example.test/#task/ws/proj/task")
	if len([]rune(msg)) > 2000 || !strings.Contains(msg, "open task for full text") {
		t.Fatal("notification is not bounded")
	}
	msg = formatTaskNotification("slack", projects.Project{Name: "<@everyone>"}, projects.Task{Title: "<https://evil.test|bad>"}, "created", "https://example.test/#task/ws/proj/task")
	if strings.Contains(msg, "<@everyone>") || strings.Contains(msg, "<https://evil.test") {
		t.Fatal("unescaped Slack content")
	}
}
