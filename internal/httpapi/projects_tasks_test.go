package httpapi

// Ported from tests/projects_and_tasks.rs.

import (
	"context"
	"fmt"
	"net/http"
	"slices"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/maruf-pfc/trestle/internal/identity"
	"github.com/maruf-pfc/trestle/internal/projects"
)

// ptHash hashes a password with the app's fast test parameters.
func ptHash(a *testApp, password string) string {
	a.t.Helper()
	hasher, _ := identity.NewHasher(a.cfg.Argon2MCost, a.cfg.Argon2TCost, a.cfg.Argon2Parallelism)
	h, err := hasher.Hash([]byte(password))
	if err != nil {
		a.t.Fatal(err)
	}
	return h
}

// ptBootstrap creates the first user and workspace, returning the user and workspace ID.
func ptBootstrap(a *testApp, username, password, displayName, wsName, wsSlug string) (identity.UserProjection, uuid.UUID) {
	a.t.Helper()
	ctx := context.Background()
	u, err := identity.Bootstrap(ctx, a.pool, username, ptHash(a, password), displayName, wsName, wsSlug)
	if err != nil {
		a.t.Fatal(err)
	}
	wss, err := identity.ListUserWorkspaces(ctx, a.pool, u.ID)
	if err != nil || len(wss) == 0 {
		a.t.Fatalf("list workspaces: %v", err)
	}
	return u, wss[0].ID
}

// ptAddMember adds userID to the workspace with role, acting as actorID.
func ptAddMember(a *testApp, actorID, wsID, userID uuid.UUID, role string) {
	a.t.Helper()
	if _, err := identity.AddWorkspaceMember(context.Background(), a.pool, &actorID, wsID, userID, role); err != nil {
		a.t.Fatal(err)
	}
}

// ptForeignWorkspace inserts a separate workspace owned by ownerID, as the Rust tests do with raw SQL.
func ptForeignWorkspace(a *testApp, ownerID uuid.UUID, name, slug string) uuid.UUID {
	a.t.Helper()
	ctx := context.Background()
	id := uuid.New()
	if _, err := a.pool.Exec(ctx, "INSERT INTO workspaces (id, name, slug) VALUES ($1, $2, $3)", id, name, slug); err != nil {
		a.t.Fatal(err)
	}
	if _, err := a.pool.Exec(ctx, "INSERT INTO workspace_memberships (workspace_id, user_id, role) VALUES ($1, $2, 'owner')", id, ownerID); err != nil {
		a.t.Fatal(err)
	}
	return id
}

// ptExpect fails unless res has the wanted status.
func ptExpect(t *testing.T, res testResponse, want int) testResponse {
	t.Helper()
	if res.StatusCode != want {
		t.Fatalf("status = %d, want %d; body %s", res.StatusCode, want, res.Body)
	}
	return res
}

// ptEq fails unless got equals want (JSON numbers compare as float64).
func ptEq(t *testing.T, field string, got, want any) {
	t.Helper()
	if n, ok := want.(int); ok {
		want = float64(n)
	}
	if got != want {
		t.Fatalf("%s = %#v, want %#v", field, got, want)
	}
}

// ptLen fails unless v is a JSON array of length want, and returns it.
func ptLen(t *testing.T, field string, v any, want int) []any {
	t.Helper()
	arr, ok := v.([]any)
	if !ok || len(arr) != want {
		t.Fatalf("%s = %#v, want array of %d", field, v, want)
	}
	return arr
}

func TestTaskActivityOnlyRecordsActualPropertyChanges(t *testing.T) {
	app := newApp(t)
	_, wsID := ptBootstrap(app, "alice", "secure-pass-123", "Alice", "Engineering", "engineering")
	c := app.login("alice", "secure-pass-123")
	base := fmt.Sprintf("/api/v1/workspaces/%s/projects", wsID)
	project := ptExpect(t, c.do("POST", base, map[string]any{"name": "Core", "slug": "core", "key": "CORE"}), http.StatusCreated).JSON(t)
	base += "/" + project["id"].(string)
	task := ptExpect(t, c.do("POST", base+"/tasks", map[string]any{"title": "Review progress", "status": "unstarted", "priority": "none"}), http.StatusCreated).JSON(t)
	path := base + "/tasks/" + task["id"].(string)
	for i := 0; i < 3; i++ {
		ptExpect(t, c.do("PATCH", path, map[string]any{"status": "unstarted", "priority": "low"}), http.StatusOK)
	}
	markdown := "  **Structure**\n\n- Next step  \n"
	got := ptExpect(t, c.do("PATCH", path, map[string]any{"description": markdown}), http.StatusOK).JSON(t)
	ptEq(t, "description", got["description"], markdown)
	ptEq(t, "status", got["status"], "unstarted")
	ptEq(t, "priority", got["priority"], "low")
	// A distinct Status in the same category still counts as a real change.
	status := ptExpect(t, c.do("POST", base+"/statuses", map[string]any{"name": "Ready", "category": "unstarted", "color": "#315b3b"}), http.StatusCreated).JSON(t)
	for i := 0; i < 3; i++ {
		ptExpect(t, c.do("PATCH", path, map[string]any{"status_id": status["id"]}), http.StatusOK)
	}
	var wg sync.WaitGroup
	for i := 0; i < 4; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			ptExpect(t, c.do("PATCH", path, map[string]any{"priority": "medium"}), http.StatusOK)
		}()
	}
	wg.Wait()
	activities := ptExpect(t, c.do("GET", path+"/activity", nil), http.StatusOK).JSON(t)["activities"].([]any)
	counts := map[string]int{}
	for _, entry := range activities {
		counts[entry.(map[string]any)["action"].(string)]++
	}
	if counts["task.status_changed"] != 1 || counts["task.priority_changed"] != 2 {
		t.Fatalf("unexpected activity counts: %v", counts)
	}
}

func TestProjectAndTaskLifecycleEndToEnd(t *testing.T) {
	app := newApp(t)
	_, wsID := ptBootstrap(app, "alice", "secure-pass-123", "Alice Smith", "Acme Engineering", "acme-eng")
	c := app.login("alice", "secure-pass-123")
	base := fmt.Sprintf("/api/v1/workspaces/%s/projects", wsID)

	// 1. List projects initially — empty.
	ptLen(t, "projects", ptExpect(t, c.do("GET", base, nil), http.StatusOK).JSON(t)["projects"], 0)

	// 2. Create project.
	payload := map[string]any{"name": "Core Platform", "slug": "core-platform", "key": "CORE", "description": "Main platform infrastructure"}
	proj := ptExpect(t, c.do("POST", base, payload), http.StatusCreated).JSON(t)
	projID := proj["id"].(string)
	ptEq(t, "key", proj["key"], "CORE")
	ptEq(t, "slug", proj["slug"], "core-platform")
	ptEq(t, "task_counter", proj["task_counter"], 0)

	// 3. Conflict on duplicate slug or key.
	ptExpect(t, c.do("POST", base, payload), http.StatusConflict)

	// 4. Get project by ID.
	got := ptExpect(t, c.do("GET", base+"/"+projID, nil), http.StatusOK).JSON(t)
	ptEq(t, "name", got["name"], "Core Platform")

	tasks := base + "/" + projID + "/tasks"
	// 5. Create Task 1.
	t1 := ptExpect(t, c.do("POST", tasks, map[string]any{
		"title": "Set up database connection pool", "description": "Configure pooling and timeouts",
		"status": "unstarted", "priority": "high", "due_date": "2026-11-15",
	}), http.StatusCreated).JSON(t)
	t1ID := t1["id"].(string)
	ptEq(t, "human_key", t1["human_key"], "CORE-1")
	ptEq(t, "task_number", t1["task_number"], 1)
	ptEq(t, "priority", t1["priority"], "high")
	ptEq(t, "status", t1["status"], "unstarted")
	ptEq(t, "due_date", t1["due_date"], "2026-11-15")

	// 6. Create Task 2.
	t2 := ptExpect(t, c.do("POST", tasks, map[string]any{
		"title": "Implement healthcheck endpoints", "description": "Live and ready checks",
		"status": "backlog", "priority": "medium",
	}), http.StatusCreated).JSON(t)
	t2ID := t2["id"].(string)
	ptEq(t, "human_key", t2["human_key"], "CORE-2")
	ptEq(t, "task_number", t2["task_number"], 2)

	// 7. List tasks — ordered by task_number DESC.
	list := ptExpect(t, c.do("GET", tasks, nil), http.StatusOK).JSON(t)
	ptEq(t, "total", list["total"], 2)
	arr := ptLen(t, "tasks", list["tasks"], 2)
	ptEq(t, "tasks[0].human_key", arr[0].(map[string]any)["human_key"], "CORE-2")
	ptEq(t, "tasks[1].human_key", arr[1].(map[string]any)["human_key"], "CORE-1")

	// 8. Filter by status=backlog.
	filtered := ptExpect(t, c.do("GET", tasks+"?status=backlog", nil), http.StatusOK).JSON(t)
	ptEq(t, "total", filtered["total"], 1)
	ptEq(t, "tasks[0].human_key", ptLen(t, "tasks", filtered["tasks"], 1)[0].(map[string]any)["human_key"], "CORE-2")

	// 9. Update Task 1 status and priority; title is unchanged.
	upd := ptExpect(t, c.do("PATCH", tasks+"/"+t1ID, map[string]any{"status": "started", "priority": "urgent"}), http.StatusOK).JSON(t)
	ptEq(t, "status", upd["status"], "started")
	ptEq(t, "priority", upd["priority"], "urgent")
	ptEq(t, "title", upd["title"], "Set up database connection pool")

	// 10. Delete Task 2; one task remains.
	ptExpect(t, c.do("DELETE", tasks+"/"+t2ID, nil), http.StatusNoContent)
	ptEq(t, "total", c.do("GET", tasks, nil).JSON(t)["total"], 1)

	// 11. Archive project; it leaves the default list but stays in include_archived.
	arch := ptExpect(t, c.do("POST", base+"/"+projID+"/archive", nil), http.StatusOK).JSON(t)
	ptEq(t, "archived", arch["archived"], true)
	ptLen(t, "projects", c.do("GET", base, nil).JSON(t)["projects"], 0)
	ptLen(t, "projects", c.do("GET", base+"?include_archived=true", nil).JSON(t)["projects"], 1)
}

func TestProjectAndTaskAuthorizationAndIsolation(t *testing.T) {
	app := newApp(t)
	const pw = "test-pass-word"
	alice, wsA := ptBootstrap(app, "alice", pw, "Alice Smith", "Workspace A", "ws-a")
	bob := app.createUser("bob", pw, "Bob Jones")
	ptForeignWorkspace(app, bob.ID, "Workspace B", "ws-b")
	charlie := app.createUser("charlie", pw, "Charlie Brown")
	ptAddMember(app, alice.ID, wsA, charlie.ID, "member")

	ac := app.login("alice", pw)
	base := fmt.Sprintf("/api/v1/workspaces/%s/projects", wsA)
	projID := ptExpect(t, ac.do("POST", base, map[string]any{
		"name": "Project Alpha", "slug": "proj-alpha", "key": "ALPHA", "description": "Secret Workspace A Project",
	}), http.StatusCreated).JSON(t)["id"].(string)
	proj := base + "/" + projID
	taskID := ptExpect(t, ac.do("POST", proj+"/tasks", map[string]any{"title": "Secret Task in A"}), http.StatusCreated).JSON(t)["id"].(string)
	task := proj + "/tasks/" + taskID

	// Bob is not a member of Workspace A: everything is a concealed 404.
	bc := app.login("bob", pw)
	ptExpect(t, bc.do("GET", base, nil), http.StatusNotFound)
	ptExpect(t, bc.do("GET", proj, nil), http.StatusNotFound)
	ptExpect(t, bc.do("GET", proj+"/tasks", nil), http.StatusNotFound)
	ptExpect(t, bc.do("POST", proj+"/tasks", map[string]any{"title": "Intruder task"}), http.StatusNotFound)
	ptExpect(t, bc.do("PATCH", task, map[string]any{"title": "Hacked"}), http.StatusNotFound)
	ptExpect(t, bc.do("DELETE", task, nil), http.StatusNotFound)

	// Charlie is a plain member of Workspace A.
	cc := app.login("charlie", pw)
	ptExpect(t, cc.do("GET", base, nil), http.StatusOK)
	ptExpect(t, cc.do("POST", base, map[string]any{"name": "Charlie Proj", "slug": "charlie-p", "key": "CP"}), http.StatusForbidden)
	ptExpect(t, cc.do("POST", proj+"/archive", nil), http.StatusForbidden)
	ptExpect(t, cc.do("POST", proj+"/tasks", map[string]any{"title": "Charlie's task"}), http.StatusCreated)
	ptExpect(t, cc.do("PATCH", task, map[string]any{"status": "started"}), http.StatusOK)
	ptExpect(t, cc.do("DELETE", task, nil), http.StatusForbidden)
}

func TestConcurrentTaskCounterAllocation(t *testing.T) {
	app := newApp(t)
	user, wsID := ptBootstrap(app, "dave", "test-pass-word", "Dave Miller", "Concurrent Org", "conc-org")
	ctx := context.Background()
	project, err := projects.CreateProject(ctx, app.pool, &user.ID, wsID, projects.CreateProjectParams{
		Name: "Concurrency Test", Slug: "conc-test", Key: "CONC", Description: "Testing concurrent task key counter",
	})
	if err != nil {
		t.Fatal(err)
	}

	const taskCount = 20
	numbers := make([]int64, taskCount)
	errs := make([]error, taskCount)
	var wg sync.WaitGroup
	for i := range taskCount {
		wg.Add(1)
		go func() {
			defer wg.Done()
			task, err := projects.CreateTask(ctx, app.pool, &user.ID, wsID, project.ID, projects.CreateTaskParams{
				Title: fmt.Sprintf("Task %d", i), Description: "Concurrent creation", Status: "unstarted", Priority: "none",
			})
			numbers[i], errs[i] = task.TaskNumber, err
		}()
	}
	wg.Wait()
	for i, err := range errs {
		if err != nil {
			t.Fatalf("task %d: %v", i, err)
		}
	}

	slices.Sort(numbers)
	for i, n := range numbers {
		if n != int64(i+1) {
			t.Fatalf("task numbers = %v, want 1..%d", numbers, taskCount)
		}
	}

	updated, err := projects.GetProject(ctx, app.pool, wsID, project.ID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.TaskCounter != taskCount {
		t.Fatalf("task_counter = %d, want %d", updated.TaskCounter, taskCount)
	}
}
