package httpapi

import (
	"fmt"
	"net/http"
	"testing"
)

func TestTimeLogDeletionIsLimitedToOwnerOrManager(t *testing.T) {
	a := newApp(t)
	owner, ws := ptBootstrap(a, "owner", "time-password", "Owner", "Workspace", "workspace")
	member := a.createUser("member", "time-password", "Member")
	ptAddMember(a, owner.ID, ws, member.ID, "member")

	c := a.login("owner", "time-password")
	m := a.login("member", "time-password")

	base := fmt.Sprintf("/api/v1/workspaces/%s/projects", ws)
	p := ptExpect(t, c.do("POST", base, map[string]any{"name": "Core", "slug": "core", "key": "CORE"}), http.StatusCreated).JSON(t)
	task := ptExpect(t, c.do("POST", base+"/"+p["id"].(string)+"/tasks", map[string]any{"title": "Tracked"}), http.StatusCreated).JSON(t)
	path := base + "/" + p["id"].(string) + "/tasks/" + task["id"].(string) + "/time-logs"

	ownerLog := ptExpect(t, c.do("POST", path, map[string]any{"hours": 2, "logged_date": "2026-11-10"}), http.StatusCreated).JSON(t)["time_log"].(map[string]any)
	memberLog := ptExpect(t, m.do("POST", path, map[string]any{"hours": 1, "logged_date": "2026-11-10"}), http.StatusCreated).JSON(t)["time_log"].(map[string]any)

	ptExpect(t, m.do("DELETE", path+"/"+ownerLog["id"].(string), nil), http.StatusForbidden)
	ptExpect(t, c.do("DELETE", path+"/"+ownerLog["id"].(string), nil), http.StatusOK)
	ptExpect(t, m.do("DELETE", path+"/"+memberLog["id"].(string), nil), http.StatusOK)
	ptExpect(t, m.do("DELETE", path+"/"+memberLog["id"].(string), nil), http.StatusNotFound)

	activity := ptExpect(t, c.do("GET", base+"/"+p["id"].(string)+"/tasks/"+task["id"].(string)+"/activity", nil), http.StatusOK).JSON(t)
	found := false
	for _, raw := range activity["activities"].([]any) {
		if raw.(map[string]any)["action"] == "time_log.deleted" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected a time_log.deleted activity record")
	}
}
