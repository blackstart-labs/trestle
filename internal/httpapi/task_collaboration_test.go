package httpapi

// Ported from tests/task_collaboration.rs.

import (
	"encoding/json"
	"fmt"
	"net/http"
	"slices"
	"testing"
)

func TestTaskCollaborationEndToEnd(t *testing.T) {
	app := newApp(t)
	const pw = "test-collab-pw"

	// 1-3. Workspace A with owner Alice and member Bob; Mallory owns a separate Workspace B.
	alice, wsA := ptBootstrap(app, "alice_collab", pw, "Alice Smith", "Collab Workspace", "collab-ws")
	bob := app.createUser("bob_collab", pw, "Bob Developer")
	ptAddMember(app, alice.ID, wsA, bob.ID, "member")
	mallory := app.createUser("mallory_collab", pw, "Mallory Attacker")
	ptForeignWorkspace(app, mallory.ID, "Mallory Workspace", "mallory-ws")

	ac := app.login("alice_collab", pw)
	bc := app.login("bob_collab", pw)
	mc := app.login("mallory_collab", pw)

	// 4. Alice creates a project.
	base := fmt.Sprintf("/api/v1/workspaces/%s/projects", wsA)
	projID := ptExpect(t, ac.do("POST", base, map[string]any{
		"name": "Platform Redesign", "slug": "platform-redesign", "key": "PLAT",
		"description": "Modern frontend & collaboration tools",
	}), http.StatusCreated).JSON(t)["id"].(string)
	proj := base + "/" + projID

	// 5. Labels.
	label1 := ptExpect(t, ac.do("POST", proj+"/labels", map[string]any{"name": "frontend", "color": "#315b3b"}), http.StatusCreated).JSON(t)
	label1ID := label1["id"].(string)
	ptEq(t, "name", label1["name"], "frontend")
	ptEq(t, "color", label1["color"], "#315b3b")
	ptExpect(t, ac.do("POST", proj+"/labels", map[string]any{"name": "frontend", "color": "#112233"}), http.StatusConflict)
	label2ID := ptExpect(t, ac.do("POST", proj+"/labels", map[string]any{"name": "bug", "color": "#9f302b"}), http.StatusCreated).JSON(t)["id"].(string)
	ptLen(t, "labels", ptExpect(t, ac.do("GET", proj+"/labels", nil), http.StatusOK).JSON(t)["labels"], 2)

	// 6. Main task.
	taskJSON := ptExpect(t, ac.do("POST", proj+"/tasks", map[string]any{
		"title": "Implement task detail drawer", "description": "Rich drawer showing details, assignees, and comments",
		"status": "started", "priority": "high",
	}), http.StatusCreated).JSON(t)
	taskID := taskJSON["id"].(string)
	ptEq(t, "human_key", taskJSON["human_key"], "PLAT-1")
	task := proj + "/tasks/" + taskID

	// 7. Attach label.
	ptExpect(t, ac.do("POST", task+"/labels", map[string]any{"label_id": label1ID}), http.StatusNoContent)

	// 8. Assign Bob; assigning non-member Mallory is a concealed 404.
	assignee := ptExpect(t, ac.do("POST", task+"/assignees", map[string]any{"user_id": bob.ID}), http.StatusCreated).JSON(t)
	ptEq(t, "username", assignee["username"], "bob_collab")
	ptExpect(t, ac.do("POST", task+"/assignees", map[string]any{"user_id": mallory.ID}), http.StatusNotFound)

	detail := ptExpect(t, ac.do("GET", task, nil), http.StatusOK).JSON(t)
	ptEq(t, "labels[0].name", ptLen(t, "labels", detail["labels"], 1)[0].(map[string]any)["name"], "frontend")
	ptEq(t, "assignees[0].username", ptLen(t, "assignees", detail["assignees"], 1)[0].(map[string]any)["username"], "bob_collab")

	// 9. Subtasks.
	sub1 := ptExpect(t, ac.do("POST", task+"/subtasks", map[string]any{
		"title": "Build drawer layout", "status": "completed", "priority": "medium",
	}), http.StatusCreated).JSON(t)
	ptEq(t, "human_key", sub1["human_key"], "PLAT-2")
	ptEq(t, "parent_id", sub1["parent_id"], taskID)
	ptExpect(t, ac.do("POST", task+"/subtasks", map[string]any{
		"title": "Add keyboard focus trap and ESC handler", "status": "unstarted", "priority": "high",
	}), http.StatusCreated)
	// The subtask list is a bare JSON array.
	subRes := ptExpect(t, ac.do("GET", task+"/subtasks", nil), http.StatusOK)
	var subs []any
	if err := json.Unmarshal(subRes.Body, &subs); err != nil || len(subs) != 2 {
		t.Fatalf("subtasks = %s, want array of 2 (%v)", subRes.Body, err)
	}

	// 10. Comments.
	bobComment := ptExpect(t, bc.do("POST", task+"/comments", map[string]any{"content": "I am working on the drawer subtasks today."}), http.StatusCreated).JSON(t)
	bobCommentID := bobComment["id"].(string)
	ptEq(t, "author_username", bobComment["author_username"], "bob_collab")
	aliceCommentID := ptExpect(t, ac.do("POST", task+"/comments", map[string]any{"content": "Sounds good! Make sure to verify keyboard ESC behavior."}), http.StatusCreated).JSON(t)["id"].(string)
	ptLen(t, "comments", ptExpect(t, ac.do("GET", task+"/comments", nil), http.StatusOK).JSON(t)["comments"], 2)
	ptExpect(t, bc.do("DELETE", task+"/comments/"+aliceCommentID, nil), http.StatusForbidden)
	ptExpect(t, bc.do("DELETE", task+"/comments/"+bobCommentID, nil), http.StatusNoContent)

	// 11. Activity feed.
	acts, _ := ptExpect(t, ac.do("GET", task+"/activity", nil), http.StatusOK).JSON(t)["activities"].([]any)
	if len(acts) == 0 {
		t.Fatal("activities is empty")
	}
	var actions []string
	for _, a := range acts {
		actions = append(actions, a.(map[string]any)["action"].(string))
	}
	for _, want := range []string{"task.created", "label.attached", "user.assigned", "subtask.created", "comment.added"} {
		if !slices.Contains(actions, want) {
			t.Fatalf("activities %v missing %q", actions, want)
		}
	}

	// 12. Filtering the task list.
	byLabel := ptExpect(t, ac.do("GET", proj+"/tasks?label_id="+label1ID, nil), http.StatusOK).JSON(t)
	ptEq(t, "total", byLabel["total"], 1)
	ptEq(t, "tasks[0].id", ptLen(t, "tasks", byLabel["tasks"], 1)[0].(map[string]any)["id"], taskID)
	ptEq(t, "total", ptExpect(t, ac.do("GET", proj+"/tasks?label_id="+label2ID, nil), http.StatusOK).JSON(t)["total"], 0)
	ptEq(t, "total", ptExpect(t, ac.do("GET", proj+"/tasks?assignee_id="+bob.ID.String(), nil), http.StatusOK).JSON(t)["total"], 1)

	// 13. Tenant isolation: Mallory gets concealed 404s.
	ptExpect(t, mc.do("GET", task, nil), http.StatusNotFound)
	ptExpect(t, mc.do("GET", proj+"/labels", nil), http.StatusNotFound)
	ptExpect(t, mc.do("GET", task+"/subtasks", nil), http.StatusNotFound)
}
