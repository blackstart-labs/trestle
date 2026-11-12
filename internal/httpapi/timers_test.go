package httpapi

import (
	"context"
	"net/http"
	"sync"
	"testing"

	"github.com/maruf-pfc/trestle/internal/projects"
)

func TestTimerLifecycleAndIdempotentStop(t *testing.T) {
	app := newApp(t)
	owner, ws := ptBootstrap(app, "owner", "timer-password", "Owner", "Studio", "studio")
	project, err := projects.CreateProject(context.Background(), app.pool, &owner.ID, ws, projects.CreateProjectParams{Name: "Core", Slug: "core", Key: "CORE"})
	if err != nil {
		t.Fatal(err)
	}
	first, err := projects.CreateTask(context.Background(), app.pool, &owner.ID, ws, project.ID, projects.CreateTaskParams{Title: "First task", Status: "unstarted", Priority: "none"})
	if err != nil {
		t.Fatal(err)
	}
	second, err := projects.CreateTask(context.Background(), app.pool, &owner.ID, ws, project.ID, projects.CreateTaskParams{Title: "Second task", Status: "unstarted", Priority: "none"})
	if err != nil {
		t.Fatal(err)
	}
	c := app.login("owner", "timer-password")

	ptExpect(t, app.anon().do("GET", "/api/v1/timer", nil), http.StatusUnauthorized)
	empty := ptExpect(t, c.do("GET", "/api/v1/timer", nil), http.StatusOK).JSON(t)
	if empty["timer"] != nil {
		t.Fatal("new user has an active timer")
	}

	started := ptExpect(t, c.do("POST", "/api/v1/timer", map[string]any{
		"task_id": first.ID.String(), "note": "Writing",
	}), http.StatusCreated).JSON(t)
	timer := started["timer"].(map[string]any)
	timerID := timer["id"].(string)
	ptEq(t, "task key", timer["task_key"], "CORE-1")
	ptEq(t, "note", timer["note"], "Writing")

	// Repeating the same start is idempotent.
	replayed := ptExpect(t, c.do("POST", "/api/v1/timer", map[string]any{
		"task_id": first.ID.String(), "note": "Writing",
	}), http.StatusCreated).JSON(t)
	ptEq(t, "replayed timer", replayed["timer"].(map[string]any)["id"], timerID)

	// A different task needs an explicit replacement decision.
	conflictRes := ptExpect(t, c.do("POST", "/api/v1/timer", map[string]any{
		"task_id": second.ID.String(),
	}), http.StatusConflict).JSON(t)
	ptEq(t, "conflict timer", conflictRes["timer"].(map[string]any)["id"], timerID)
	ptEq(t, "conflict code", conflictRes["error"].(map[string]any)["code"], "timer_active")

	replaced := ptExpect(t, c.do("POST", "/api/v1/timer", map[string]any{
		"task_id": second.ID.String(), "replace_active": true,
	}), http.StatusCreated).JSON(t)
	secondID := replaced["timer"].(map[string]any)["id"].(string)
	if secondID == timerID {
		t.Fatal("replacement reused the old timer")
	}
	var discarded string
	if err := app.pool.QueryRow(context.Background(), "SELECT state FROM task_timers WHERE id = $1", timerID).Scan(&discarded); err != nil {
		t.Fatal(err)
	}
	ptEq(t, "replaced timer state", discarded, "discarded")

	updated := ptExpect(t, c.do("PATCH", "/api/v1/timer", map[string]any{
		"timer_id": secondID, "note": "Reviewing",
	}), http.StatusOK).JSON(t)
	ptEq(t, "updated note", updated["timer"].(map[string]any)["note"], "Reviewing")

	stopped := ptExpect(t, c.do("POST", "/api/v1/timer/stop", map[string]any{
		"timer_id": secondID, "note": "Reviewed",
	}), http.StatusOK).JSON(t)
	log := stopped["time_log"].(map[string]any)
	logID := log["id"].(string)
	ptEq(t, "log note", log["description"], "Reviewed")
	ptEq(t, "spent hours", stopped["spent_hours"], log["hours"])

	// A lost stop response is retried without creating another time log.
	again := ptExpect(t, c.do("POST", "/api/v1/timer/stop", map[string]any{
		"timer_id": secondID,
	}), http.StatusOK).JSON(t)
	ptEq(t, "retried log", again["time_log"].(map[string]any)["id"], logID)
	var logCount int
	if err := app.pool.QueryRow(context.Background(), "SELECT count(*) FROM task_time_logs WHERE task_id = $1", second.ID).Scan(&logCount); err != nil {
		t.Fatal(err)
	}
	if logCount != 1 {
		t.Fatalf("time logs = %d, want 1", logCount)
	}

	after := ptExpect(t, c.do("GET", "/api/v1/timer", nil), http.StatusOK).JSON(t)
	if after["timer"] != nil {
		t.Fatal("stopped timer still active")
	}
	ptExpect(t, c.do("POST", "/api/v1/timer/discard", map[string]any{"timer_id": secondID}), http.StatusConflict)
}

func TestTimerDiscardIsIdempotent(t *testing.T) {
	app := newApp(t)
	owner, ws := ptBootstrap(app, "owner", "discard-password", "Owner", "Studio", "studio")
	project, err := projects.CreateProject(context.Background(), app.pool, &owner.ID, ws, projects.CreateProjectParams{Name: "Core", Slug: "core", Key: "CORE"})
	if err != nil {
		t.Fatal(err)
	}
	task, err := projects.CreateTask(context.Background(), app.pool, &owner.ID, ws, project.ID, projects.CreateTaskParams{Title: "Task", Status: "unstarted", Priority: "none"})
	if err != nil {
		t.Fatal(err)
	}
	c := app.login("owner", "discard-password")
	started := ptExpect(t, c.do("POST", "/api/v1/timer", map[string]any{"task_id": task.ID.String()}), http.StatusCreated).JSON(t)
	timerID := started["timer"].(map[string]any)["id"].(string)
	ptExpect(t, c.do("POST", "/api/v1/timer/discard", map[string]any{"timer_id": timerID}), http.StatusOK)
	ptExpect(t, c.do("POST", "/api/v1/timer/discard", map[string]any{"timer_id": timerID}), http.StatusOK)
	ptExpect(t, c.do("POST", "/api/v1/timer/stop", map[string]any{"timer_id": timerID}), http.StatusConflict)
	var count int
	if err := app.pool.QueryRow(context.Background(), "SELECT count(*) FROM task_time_logs WHERE task_id = $1", task.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("discarded timer created %d time logs", count)
	}
}

func TestTimerPermissionsAndArchivedProjects(t *testing.T) {
	app := newApp(t)
	owner, ws := ptBootstrap(app, "owner", "timer-perm-password", "Owner", "Studio", "studio")
	guest := app.createUser("guest", "timer-perm-password", "Guest")
	ptAddMember(app, owner.ID, ws, guest.ID, "guest")
	app.createUser("outsider", "timer-perm-password", "Outsider")
	project, err := projects.CreateProject(context.Background(), app.pool, &owner.ID, ws, projects.CreateProjectParams{Name: "Core", Slug: "core", Key: "CORE"})
	if err != nil {
		t.Fatal(err)
	}
	task, err := projects.CreateTask(context.Background(), app.pool, &owner.ID, ws, project.ID, projects.CreateTaskParams{Title: "Task", Status: "unstarted", Priority: "none"})
	if err != nil {
		t.Fatal(err)
	}
	guestClient := app.login("guest", "timer-perm-password")
	ptExpect(t, guestClient.do("POST", "/api/v1/timer", map[string]any{"task_id": task.ID.String()}), http.StatusForbidden)

	outsiderClient := app.login("outsider", "timer-perm-password")
	ptExpect(t, outsiderClient.do("POST", "/api/v1/timer", map[string]any{"task_id": task.ID.String()}), http.StatusNotFound)

	if _, err := app.pool.Exec(context.Background(), "UPDATE projects SET archived_at = now() WHERE id = $1", project.ID); err != nil {
		t.Fatal(err)
	}
	ownerClient := app.login("owner", "timer-perm-password")
	ptExpect(t, ownerClient.do("POST", "/api/v1/timer", map[string]any{"task_id": task.ID.String()}), http.StatusConflict)
}

func TestTimerConcurrentStopCreatesOneLog(t *testing.T) {
	app := newApp(t)
	owner, ws := ptBootstrap(app, "owner", "timer-race-password", "Owner", "Studio", "studio")
	project, err := projects.CreateProject(context.Background(), app.pool, &owner.ID, ws, projects.CreateProjectParams{Name: "Core", Slug: "core", Key: "CORE"})
	if err != nil {
		t.Fatal(err)
	}
	task, err := projects.CreateTask(context.Background(), app.pool, &owner.ID, ws, project.ID, projects.CreateTaskParams{Title: "Task", Status: "unstarted", Priority: "none"})
	if err != nil {
		t.Fatal(err)
	}
	c := app.login("owner", "timer-race-password")
	started := ptExpect(t, c.do("POST", "/api/v1/timer", map[string]any{"task_id": task.ID.String()}), http.StatusCreated).JSON(t)
	timerID := started["timer"].(map[string]any)["id"].(string)

	var wg sync.WaitGroup
	results := make(chan testResponse, 2)
	for range 2 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			results <- c.do("POST", "/api/v1/timer/stop", map[string]any{"timer_id": timerID})
		}()
	}
	wg.Wait()
	close(results)
	var ids []string
	for res := range results {
		if res.StatusCode != http.StatusOK {
			t.Fatalf("concurrent stop = %d %s", res.StatusCode, res.Body)
		}
		ids = append(ids, res.JSON(t)["time_log"].(map[string]any)["id"].(string))
	}
	if len(ids) != 2 || ids[0] != ids[1] {
		t.Fatalf("concurrent stops returned different logs: %v", ids)
	}
	var count int
	if err := app.pool.QueryRow(context.Background(), "SELECT count(*) FROM task_time_logs WHERE task_id = $1", task.ID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("time logs = %d, want 1", count)
	}
}

func TestTimerValidationAndUnknownTimer(t *testing.T) {
	app := newApp(t)
	_, _ = ptBootstrap(app, "owner", "timer-valid-password", "Owner", "Studio", "studio")
	c := app.login("owner", "timer-valid-password")
	ptExpect(t, c.do("POST", "/api/v1/timer", map[string]any{}), http.StatusUnprocessableEntity)
	ptExpect(t, c.do("POST", "/api/v1/timer", map[string]any{"task_id": "not-a-uuid"}), http.StatusBadRequest)
	ptExpect(t, c.do("POST", "/api/v1/timer/stop", map[string]any{}), http.StatusUnprocessableEntity)
	ptExpect(t, c.do("POST", "/api/v1/timer/stop", map[string]any{"timer_id": "00000000-0000-0000-0000-000000000000"}), http.StatusNotFound)
	ptExpect(t, c.do("PATCH", "/api/v1/timer", map[string]any{"timer_id": "00000000-0000-0000-0000-000000000000", "note": "x"}), http.StatusNotFound)
}

func TestTimerLoggedDateUsesUserTimezone(t *testing.T) {
	app := newApp(t)
	owner, ws := ptBootstrap(app, "owner", "timer-tz-password", "Owner", "Studio", "studio")
	project, err := projects.CreateProject(context.Background(), app.pool, &owner.ID, ws, projects.CreateProjectParams{Name: "Core", Slug: "core", Key: "CORE"})
	if err != nil {
		t.Fatal(err)
	}
	task, err := projects.CreateTask(context.Background(), app.pool, &owner.ID, ws, project.ID, projects.CreateTaskParams{Title: "Task", Status: "unstarted", Priority: "none"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := app.pool.Exec(context.Background(), "UPDATE users SET timezone = 'Asia/Dhaka' WHERE id = $1", owner.ID); err != nil {
		t.Fatal(err)
	}
	c := app.login("owner", "timer-tz-password")
	started := ptExpect(t, c.do("POST", "/api/v1/timer", map[string]any{"task_id": task.ID.String()}), http.StatusCreated).JSON(t)
	timerID := started["timer"].(map[string]any)["id"].(string)
	if _, err := app.pool.Exec(context.Background(), "UPDATE task_timers SET started_at = now() - interval '1 hour' WHERE id = $1", timerID); err != nil {
		t.Fatal(err)
	}
	stopped := ptExpect(t, c.do("POST", "/api/v1/timer/stop", map[string]any{"timer_id": timerID}), http.StatusOK).JSON(t)
	log := stopped["time_log"].(map[string]any)
	var want string
	if err := app.pool.QueryRow(context.Background(), "SELECT (started_at AT TIME ZONE 'Asia/Dhaka')::date::text FROM task_timers WHERE id = $1", timerID).Scan(&want); err != nil {
		t.Fatal(err)
	}
	ptEq(t, "logged date", log["logged_date"], want)
	ptEq(t, "rounded hours", log["hours"], 1.0)
}
