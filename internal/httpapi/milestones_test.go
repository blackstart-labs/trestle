package httpapi

import (
	"fmt"
	"net/http"
	"strings"
	"testing"
)

func TestMilestoneLifecycleAndTaskRollup(t *testing.T) {
	app := newApp(t)
	_, wsID := ptBootstrap(app, "owner", "milestone-pw", "Owner", "Engineering", "engineering")
	c := app.login("owner", "milestone-pw")

	base := fmt.Sprintf("/api/v1/workspaces/%s/projects", wsID)
	proj := ptExpect(t, c.do("POST", base, map[string]any{"name": "Core", "slug": "core", "key": "CORE"}), http.StatusCreated).JSON(t)
	projectPath := base + "/" + proj["id"].(string)

	// Validation checks
	ptExpect(t, c.do("POST", projectPath+"/milestones", map[string]any{"title": ""}), http.StatusBadRequest)
	ptExpect(t, c.do("POST", projectPath+"/milestones", map[string]any{"title": strings.Repeat("m", 256)}), http.StatusBadRequest)
	ptExpect(t, c.do("POST", projectPath+"/milestones", map[string]any{"title": "Valid", "due_date": "not-a-date"}), http.StatusBadRequest)
	ptExpect(t, c.do("POST", projectPath+"/milestones", map[string]any{"title": "Valid", "start_date": "not-a-date"}), http.StatusBadRequest)
	ptExpect(t, c.do("POST", projectPath+"/milestones", map[string]any{"title": "Valid", "color": "invalid-color"}), http.StatusBadRequest)

	// Create milestone
	created := ptExpect(t, c.do("POST", projectPath+"/milestones", map[string]any{
		"title":       "v1.0 Release",
		"description": "First major release targets",
		"start_date":  "2026-11-01",
		"end_date":    "2026-12-31",
		"color":       "#1e4d79",
	}), http.StatusCreated).JSON(t)

	mid := created["id"].(string)
	ptEq(t, "title", created["title"], "v1.0 Release")
	ptEq(t, "description", created["description"], "First major release targets")
	ptEq(t, "start_date", created["start_date"], "2026-11-01")
	ptEq(t, "due_date", created["due_date"], "2026-12-31")
	ptEq(t, "end_date", created["end_date"], "2026-12-31")
	ptEq(t, "color", created["color"], "#1e4d79")
	ptEq(t, "status", created["status"], "open")
	ptEq(t, "total_tasks", created["total_tasks"], 0)
	ptEq(t, "completed_tasks", created["completed_tasks"], 0)
	ptEq(t, "open_tasks", created["open_tasks"], 0)

	// Create tasks associated with this milestone
	t1 := ptExpect(t, c.do("POST", projectPath+"/tasks", map[string]any{
		"title":        "Task 1",
		"status":       "unstarted",
		"milestone_id": mid,
	}), http.StatusCreated).JSON(t)
	ptEq(t, "t1 milestone_id", t1["milestone_id"], mid)
	ptEq(t, "t1 milestone_title", t1["milestone_title"], "v1.0 Release")
	ptEq(t, "t1 milestone_color", t1["milestone_color"], "#1e4d79")

	t2 := ptExpect(t, c.do("POST", projectPath+"/tasks", map[string]any{
		"title":        "Task 2",
		"status":       "completed",
		"milestone_id": mid,
	}), http.StatusCreated).JSON(t)
	ptEq(t, "t2 milestone_id", t2["milestone_id"], mid)

	// Unrelated task
	ptExpect(t, c.do("POST", projectPath+"/tasks", map[string]any{
		"title": "Unrelated Task",
	}), http.StatusCreated)

	// Get milestone rollup
	got := ptExpect(t, c.do("GET", projectPath+"/milestones/"+mid, nil), http.StatusOK).JSON(t)
	ptEq(t, "total_tasks", got["total_tasks"], 2)
	ptEq(t, "completed_tasks", got["completed_tasks"], 1)
	ptEq(t, "open_tasks", got["open_tasks"], 1)

	// Filter tasks by milestone_id
	listFiltered := ptExpect(t, c.do("GET", projectPath+"/tasks?milestone_id="+mid, nil), http.StatusOK).JSON(t)
	tasks := listFiltered["tasks"].([]any)
	if len(tasks) != 2 {
		t.Fatalf("expected 2 tasks for milestone, got %d", len(tasks))
	}

	// Update task milestone (clear milestone_id)
	updatedTask := ptExpect(t, c.do("PATCH", projectPath+"/tasks/"+t1["id"].(string), map[string]any{
		"milestone_id": nil,
	}), http.StatusOK).JSON(t)
	if updatedTask["milestone_id"] != nil {
		t.Fatalf("expected nil milestone_id after clear, got %v", updatedTask["milestone_id"])
	}

	// Update milestone status to closed
	updatedM := ptExpect(t, c.do("PATCH", projectPath+"/milestones/"+mid, map[string]any{
		"status":     "closed",
		"title":      "v1.0 Final",
		"color":      "#26633b",
		"start_date": "2026-11-05",
	}), http.StatusOK).JSON(t)
	ptEq(t, "status", updatedM["status"], "closed")
	ptEq(t, "title", updatedM["title"], "v1.0 Final")
	ptEq(t, "color", updatedM["color"], "#26633b")
	ptEq(t, "start_date", updatedM["start_date"], "2026-11-05")
	ptEq(t, "total_tasks", updatedM["total_tasks"], 1)

	// Delete milestone
	ptExpect(t, c.do("DELETE", projectPath+"/milestones/"+mid, nil), http.StatusNoContent)

	// Verify t2 still exists and its milestone_id is now nil
	t2After := ptExpect(t, c.do("GET", projectPath+"/tasks/"+t2["id"].(string), nil), http.StatusOK).JSON(t)
	if t2After["milestone_id"] != nil {
		t.Fatalf("expected nil milestone_id after milestone deletion, got %v", t2After["milestone_id"])
	}
}

