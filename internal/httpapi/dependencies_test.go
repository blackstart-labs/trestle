package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/maruf-pfc/trestle/internal/projects"
)

type depFixture struct {
	app      *testApp
	ownerID  uuid.UUID
	ws       uuid.UUID
	projectA uuid.UUID
	projectB uuid.UUID
	taskA1   uuid.UUID
	taskA2   uuid.UUID
	taskA3   uuid.UUID
	taskA4   uuid.UUID
	taskB1   uuid.UUID
	client   *client
}

func (f *depFixture) depBase(task uuid.UUID) string {
	return fmt.Sprintf("/api/v1/workspaces/%s/projects/%s/tasks/%s/dependencies",
		f.ws, f.projectA, task)
}

func (f *depFixture) taskBase(project, task uuid.UUID) string {
	return fmt.Sprintf("/api/v1/workspaces/%s/projects/%s/tasks/%s/dependencies",
		f.ws, project, task)
}

func newDependencyFixture(t *testing.T) *depFixture {
	t.Helper()
	app := newApp(t)
	owner, ws := ptBootstrap(app, "owner", "dep-password", "Owner", "Studio", "studio")
	password := "dep-password"
	newProject := func(name, slug, key string) uuid.UUID {
		p, err := projects.CreateProject(context.Background(), app.pool, &owner.ID, ws,
			projects.CreateProjectParams{Name: name, Slug: slug, Key: key})
		if err != nil {
			t.Fatal(err)
		}
		return p.ID
	}
	f := &depFixture{app: app, ownerID: owner.ID, ws: ws}
	f.projectA = newProject("Core", "core", "CORE")
	f.projectB = newProject("Web", "web", "WEB")
	newTask := func(project uuid.UUID, title string) uuid.UUID {
		task, err := projects.CreateTask(context.Background(), app.pool, &owner.ID, ws, project,
			projects.CreateTaskParams{Title: title, Status: "unstarted", Priority: "none"})
		if err != nil {
			t.Fatal(err)
		}
		return task.ID
	}
	f.taskA1 = newTask(f.projectA, "Design schema")
	f.taskA2 = newTask(f.projectA, "Build API")
	f.taskA3 = newTask(f.projectA, "Write docs")
	f.taskA4 = newTask(f.projectA, "Ship release")
	f.taskB1 = newTask(f.projectB, "Build page")
	f.client = app.login("owner", password)
	return f
}

func TestTaskDependenciesLifecycle(t *testing.T) {
	f := newDependencyFixture(t)
	base := f.depBase(f.taskA1)

	empty := ptExpect(t, f.client.do("GET", base, nil), http.StatusOK).JSON(t)
	ptLen(t, "blocked_by", empty["blocked_by"], 0)
	ptLen(t, "blocks", empty["blocks"], 0)

	added := ptExpect(t, f.client.do("POST", base, map[string]any{
		"related_task_id": f.taskB1.String(), "direction": "blocked_by",
	}), http.StatusCreated).JSON(t)
	dep := added["dependency"].(map[string]any)
	rel := dep["task"].(map[string]any)
	ptEq(t, "related key", rel["human_key"], "WEB-1")
	ptEq(t, "related project", rel["project_key"], "WEB")
	ptEq(t, "related title", rel["title"], "Build page")
	depID := dep["id"].(string)

	a1 := ptExpect(t, f.client.do("GET", base, nil), http.StatusOK).JSON(t)
	blockedBy := ptLen(t, "blocked_by", a1["blocked_by"], 1)
	ptEq(t, "blocked by key", blockedBy[0].(map[string]any)["task"].(map[string]any)["human_key"], "WEB-1")
	ptLen(t, "blocks", a1["blocks"], 0)

	// The other side sees the inverse direction.
	bBase := f.taskBase(f.projectB, f.taskB1)
	b1 := ptExpect(t, f.client.do("GET", bBase, nil), http.StatusOK).JSON(t)
	blocks := ptLen(t, "blocks", b1["blocks"], 1)
	ptEq(t, "blocker key", blocks[0].(map[string]any)["task"].(map[string]any)["human_key"], "CORE-1")
	ptLen(t, "blocked_by", b1["blocked_by"], 0)

	// The same edge in the same direction is a conflict.
	ptExpect(t, f.client.do("POST", base, map[string]any{
		"related_task_id": f.taskB1.String(), "direction": "blocked_by",
	}), http.StatusConflict)
	// The reverse direction is a cycle.
	ptExpect(t, f.client.do("POST", bBase, map[string]any{
		"related_task_id": f.taskA1.String(), "direction": "blocked_by",
	}), http.StatusConflict)
	// Self-links are refused.
	ptExpect(t, f.client.do("POST", base, map[string]any{
		"related_task_id": f.taskA1.String(), "direction": "blocked_by",
	}), http.StatusBadRequest)

	// Removal works from the inverse side and is then gone from both.
	ptExpect(t, f.client.do("DELETE", bBase+"/"+depID, nil), http.StatusNoContent)
	ptExpect(t, f.client.do("DELETE", bBase+"/"+depID, nil), http.StatusNotFound)
	after := ptExpect(t, f.client.do("GET", base, nil), http.StatusOK).JSON(t)
	ptLen(t, "blocked_by", after["blocked_by"], 0)
}

func TestTaskDependenciesRejectIndirectCycles(t *testing.T) {
	f := newDependencyFixture(t)
	link := func(task, related uuid.UUID, direction string) testResponse {
		return f.client.do("POST", f.depBase(task), map[string]any{
			"related_task_id": related.String(), "direction": direction,
		})
	}
	// CORE-1 waits for CORE-2, CORE-2 waits for CORE-3.
	ptExpect(t, link(f.taskA1, f.taskA2, "blocked_by"), http.StatusCreated)
	ptExpect(t, link(f.taskA2, f.taskA3, "blocked_by"), http.StatusCreated)
	// CORE-3 waiting for CORE-1 would close the loop.
	ptExpect(t, link(f.taskA3, f.taskA1, "blocked_by"), http.StatusConflict)
	// A legal edge in the same graph still works: CORE-1 waits for CORE-3 too.
	ptExpect(t, link(f.taskA1, f.taskA3, "blocked_by"), http.StatusCreated)

	// The blocks direction validates the same cycle: CORE-4 waits for CORE-1,
	// so CORE-4 blocking CORE-1 would close the loop.
	ptExpect(t, link(f.taskA4, f.taskA1, "blocked_by"), http.StatusCreated)
	ptExpect(t, link(f.taskA4, f.taskA1, "blocks"), http.StatusConflict)
	// A legal blocks edge: CORE-4 unblocks WEB-1 (WEB-1 waits for CORE-4).
	ptExpect(t, link(f.taskA4, f.taskB1, "blocks"), http.StatusCreated)
}

func TestTaskDependenciesPermissionsAndScope(t *testing.T) {
	f := newDependencyFixture(t)
	guest := f.app.createUser("guest", "dep-password", "Guest")
	ptAddMember(f.app, f.ownerID, f.ws, guest.ID, "guest")
	f.app.createUser("outsider", "dep-password", "Outsider")

	base := f.depBase(f.taskA1)
	guestClient := f.app.login("guest", "dep-password")
	ptExpect(t, guestClient.do("GET", base, nil), http.StatusOK)
	ptExpect(t, guestClient.do("POST", base, map[string]any{
		"related_task_id": f.taskB1.String(), "direction": "blocked_by",
	}), http.StatusForbidden)

	outsiderClient := f.app.login("outsider", "dep-password")
	ptExpect(t, outsiderClient.do("GET", base, nil), http.StatusNotFound)
	ptExpect(t, outsiderClient.do("POST", base, map[string]any{
		"related_task_id": f.taskB1.String(), "direction": "blocked_by",
	}), http.StatusNotFound)

	// A related task in another workspace is concealed.
	foreignWS := ptForeignWorkspace(f.app, f.ownerID, "Elsewhere", "elsewhere")
	foreign, err := projects.CreateProject(context.Background(), f.app.pool, nil, foreignWS,
		projects.CreateProjectParams{Name: "Foreign", Slug: "foreign", Key: "FOR"})
	if err != nil {
		t.Fatal(err)
	}
	foreignTask, err := projects.CreateTask(context.Background(), f.app.pool, nil, foreignWS, foreign.ID,
		projects.CreateTaskParams{Title: "Secret", Status: "unstarted", Priority: "none"})
	if err != nil {
		t.Fatal(err)
	}
	ptExpect(t, f.client.do("POST", base, map[string]any{
		"related_task_id": foreignTask.ID.String(), "direction": "blocked_by",
	}), http.StatusNotFound)

	// A task id that is not in the project in the URL is concealed.
	wrongPath := f.taskBase(f.projectA, f.taskB1)
	ptExpect(t, f.client.do("GET", wrongPath, nil), http.StatusNotFound)
	ptExpect(t, f.client.do("POST", wrongPath, map[string]any{
		"related_task_id": f.taskA1.String(), "direction": "blocked_by",
	}), http.StatusNotFound)
}

func TestTaskDependencyConcurrentOppositeEdgesCannotCycle(t *testing.T) {
	f := newDependencyFixture(t)
	var wg sync.WaitGroup
	results := make(chan int, 2)
	add := func(from, to uuid.UUID) {
		defer wg.Done()
		res := f.client.do("POST", f.depBase(from), map[string]any{
			"related_task_id": to.String(), "direction": "blocked_by",
		})
		results <- res.StatusCode
	}
	wg.Add(2)
	go add(f.taskA1, f.taskA2)
	go add(f.taskA2, f.taskA1)
	wg.Wait()
	close(results)

	created, conflicts := 0, 0
	for code := range results {
		switch code {
		case http.StatusCreated:
			created++
		case http.StatusConflict:
			conflicts++
		default:
			t.Fatalf("unexpected status %d", code)
		}
	}
	if created != 1 || conflicts != 1 {
		t.Fatalf("created=%d conflicts=%d, want one of each", created, conflicts)
	}
	var count int
	if err := f.app.pool.QueryRow(context.Background(),
		"SELECT count(*) FROM task_dependencies WHERE workspace_id = $1", f.ws).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("dependency rows = %d, want 1", count)
	}
}

func TestWorkspaceTaskSearch(t *testing.T) {
	f := newDependencyFixture(t)
	search := func(query string) []any {
		res := ptExpect(t, f.client.do("GET",
			fmt.Sprintf("/api/v1/workspaces/%s/tasks?query=%s", f.ws, query), nil), http.StatusOK).JSON(t)
		return res["tasks"].([]any)
	}

	all := search("")
	if len(all) < 4 {
		t.Fatalf("recent tasks = %d, want at least 4", len(all))
	}
	byTitle := search("Build")
	if len(byTitle) != 2 {
		t.Fatalf("title search = %d, want 2", len(byTitle))
	}
	byKey := search("WEB-1")
	if len(byKey) != 1 {
		t.Fatalf("key search = %d, want 1", len(byKey))
	}
	ptEq(t, "key search title", byKey[0].(map[string]any)["title"], "Build page")

	// Archived projects are not offered for new links.
	if _, err := f.app.pool.Exec(context.Background(),
		"UPDATE projects SET archived_at = now() WHERE id = $1", f.projectB); err != nil {
		t.Fatal(err)
	}
	if got := search("WEB-1"); len(got) != 0 {
		t.Fatalf("archived project task still searchable: %d", len(got))
	}

	// Guests can search; non-members cannot.
	guest := f.app.createUser("guest", "dep-password", "Guest")
	ptAddMember(f.app, f.ownerID, f.ws, guest.ID, "guest")
	guestClient := f.app.login("guest", "dep-password")
	ptExpect(t, guestClient.do("GET", fmt.Sprintf("/api/v1/workspaces/%s/tasks", f.ws), nil), http.StatusOK)

	f.app.createUser("outsider", "dep-password", "Outsider")
	outsiderClient := f.app.login("outsider", "dep-password")
	ptExpect(t, outsiderClient.do("GET", fmt.Sprintf("/api/v1/workspaces/%s/tasks", f.ws), nil), http.StatusNotFound)

	// Overlong queries are refused.
	long := ""
	for range 101 {
		long += "a"
	}
	ptExpect(t, f.client.do("GET",
		fmt.Sprintf("/api/v1/workspaces/%s/tasks?query=%s", f.ws, long), nil), http.StatusBadRequest)
}
