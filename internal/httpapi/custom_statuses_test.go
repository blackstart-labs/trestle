package httpapi

// Ported from tests/custom_statuses.rs.

import (
	"fmt"
	"net/http"
	"slices"
	"testing"
)

func TestCustomStatusesLifecycleAndRemap(t *testing.T) {
	app := newApp(t)
	const pw = "test-status-pw"

	// 1-2. Owner Alice and member Bob.
	alice, wsID := ptBootstrap(app, "alice_status", pw, "Alice Admin", "Status Workspace", "status-ws")
	bob := app.createUser("bob_status", pw, "Bob Member")
	ptAddMember(app, alice.ID, wsID, bob.ID, "member")
	ac := app.login("alice_status", pw)
	bc := app.login("bob_status", pw)

	// 3. Create a project.
	base := fmt.Sprintf("/api/v1/workspaces/%s/projects", wsID)
	projID := ptExpect(t, ac.do("POST", base, map[string]any{"name": "Status Test Project", "slug": "status-proj", "key": "STAT"}), http.StatusCreated).JSON(t)["id"].(string)
	proj := base + "/" + projID

	// 4. Five default statuses, in order.
	statuses := ptLen(t, "statuses", ptExpect(t, bc.do("GET", proj+"/statuses", nil), http.StatusOK).JSON(t)["statuses"], 5)
	var names []string
	for _, s := range statuses {
		names = append(names, s.(map[string]any)["name"].(string))
	}
	if want := []string{"Backlog", "To Do", "In Progress", "Done", "Cancelled"}; !slices.Equal(names, want) {
		t.Fatalf("default statuses = %v, want %v", names, want)
	}
	todoID := statuses[1].(map[string]any)["id"].(string)
	doneID := statuses[3].(map[string]any)["id"].(string)

	// 5. Member Bob cannot create a status.
	ptExpect(t, bc.do("POST", proj+"/statuses", map[string]any{"name": "Blocked", "category": "started", "color": "#9f302b"}), http.StatusForbidden)

	// 6. Alice creates "In QA".
	qa := ptExpect(t, ac.do("POST", proj+"/statuses", map[string]any{"name": "In QA", "category": "started", "color": "#775300"}), http.StatusCreated).JSON(t)
	qaID := qa["id"].(string)
	ptEq(t, "name", qa["name"], "In QA")
	ptEq(t, "category", qa["category"], "started")
	ptEq(t, "position", qa["position"], 5)

	// 7. Duplicate name conflicts.
	ptExpect(t, ac.do("POST", proj+"/statuses", map[string]any{"name": "In QA", "category": "started"}), http.StatusConflict)

	// 8. Rename and recolour.
	upd := ptExpect(t, ac.do("PATCH", proj+"/statuses/"+qaID, map[string]any{"name": "QA Testing", "color": "#8c6200"}), http.StatusOK).JSON(t)
	ptEq(t, "name", upd["name"], "QA Testing")
	ptEq(t, "color", upd["color"], "#8c6200")

	// 9. Bob creates a task in QA Testing.
	task := ptExpect(t, bc.do("POST", proj+"/tasks", map[string]any{"title": "Verify status workflow", "status_id": qaID}), http.StatusCreated).JSON(t)
	taskID := task["id"].(string)
	ptEq(t, "status_id", task["status_id"], qaID)
	ptEq(t, "status_name", task["status_name"], "QA Testing")
	ptEq(t, "status", task["status"], "started")

	// 10. Deleting a status in use without remap_to conflicts.
	ptExpect(t, ac.do("DELETE", proj+"/statuses/"+qaID, nil), http.StatusConflict)

	// 11. With remap_to it reassigns tasks and deletes.
	ptExpect(t, ac.do("DELETE", proj+"/statuses/"+qaID+"?remap_to="+doneID, nil), http.StatusNoContent)

	// 12. The task now sits in Done.
	check := ptExpect(t, bc.do("GET", proj+"/tasks/"+taskID, nil), http.StatusOK).JSON(t)
	ptEq(t, "status_id", check["status_id"], doneID)
	ptEq(t, "status_name", check["status_name"], "Done")
	ptEq(t, "status", check["status"], "completed")

	// 13. Reorder.
	ptExpect(t, ac.do("PUT", proj+"/statuses/positions", map[string]any{"status_ids": []string{doneID, todoID}}), http.StatusNoContent)
}
