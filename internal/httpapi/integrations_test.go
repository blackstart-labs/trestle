package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestProjectIntegrationsLifecycleAndIsolation(t *testing.T) {
	app := newApp(t)
	const pw = "test-pass-word"
	owner, ws := ptBootstrap(app, "owner", pw, "Owner", "Workspace", "workspace")
	member := app.createUser("member", pw, "Member")
	ptAddMember(app, owner.ID, ws, member.ID, "member")
	admin := app.createUser("admin", pw, "Admin")
	ptAddMember(app, owner.ID, ws, admin.ID, "admin")
	outsider := app.createUser("outsider", pw, "Outsider")
	foreignWS := ptForeignWorkspace(app, outsider.ID, "Foreign", "foreign")
	c := app.login("owner", pw)
	base := fmt.Sprintf("/api/v1/workspaces/%s/projects", ws)
	create := func(name, key string) string {
		return ptExpect(t, c.do("POST", base, map[string]any{"name": name, "slug": strings.ToLower(key), "key": key}), http.StatusCreated).JSON(t)["id"].(string)
	}
	project, otherProject := create("Alpha", "ALPHA"), create("Beta", "BETA")
	endpoint := base + "/" + project + "/integrations"
	const secret = "https://hooks.slack.com/services/TEAM/BOT/SECRETTOKEN"
	payload := map[string]any{"secret": secret}
	for _, provider := range []string{"telegram", "slack", "discord"} {
		config := payload
		if provider == "telegram" {
			config = map[string]any{"secret": "123456:abcdefghijklmnopqrstuvwxyz_123456789", "chat_id": "-100123"}
		}
		if provider == "discord" {
			config = map[string]any{"secret": "https://discord.com/api/webhooks/123/secret"}
		}
		saved := ptExpect(t, c.do("PUT", endpoint+"/"+provider, config), http.StatusOK)
		if strings.Contains(string(saved.Body), "secret") || strings.Contains(string(saved.Body), "SECRETTOKEN") {
			t.Fatal("credential in save response")
		}
	}
	list := ptExpect(t, c.do("GET", endpoint, nil), http.StatusOK)
	ptLen(t, "integrations", list.JSON(t)["integrations"], 3)
	if strings.Contains(string(list.Body), "SECRETTOKEN") || strings.Contains(string(list.Body), "secret\"") {
		t.Fatal("credential in list response")
	}
	ptLen(t, "integrations", ptExpect(t, c.do("GET", base+"/"+otherProject+"/integrations", nil), http.StatusOK).JSON(t)["integrations"], 0)
	for _, method := range []string{"GET", "PUT", "DELETE", "POST"} {
		suffix := "/slack"
		if method == "GET" {
			suffix = ""
		}
		if method == "POST" {
			suffix += "/test"
		}
		ptExpect(t, app.anon().do(method, endpoint+suffix, payload), http.StatusUnauthorized)
		ptExpect(t, app.login("member", pw).do(method, endpoint+suffix, payload), http.StatusForbidden)
		ptExpect(t, app.login("outsider", pw).do(method, endpoint+suffix, payload), http.StatusNotFound)
	}
	ptExpect(t, c.do("GET", fmt.Sprintf("/api/v1/workspaces/%s/projects/%s/integrations", foreignWS, project), nil), http.StatusNotFound)
	// A project from another Workspace cannot be configured through a Workspace the caller owns.
	ptExpect(t, app.login("outsider", pw).do("PUT", fmt.Sprintf("/api/v1/workspaces/%s/projects/%s/integrations/slack", foreignWS, project), payload), http.StatusNotFound)
	ptExpect(t, app.login("admin", pw).do("PUT", endpoint+"/slack", payload), http.StatusOK)
	ptExpect(t, c.do("PUT", endpoint+"/slack", map[string]any{"secret": "http://127.0.0.1/private"}), http.StatusBadRequest)
	ptExpect(t, c.do("POST", endpoint+"/unknown/test", nil), http.StatusNotFound)
	for _, provider := range []string{"telegram", "slack", "discord"} {
		ptExpect(t, c.do("DELETE", endpoint+"/"+provider, nil), http.StatusNoContent)
	}
	ptLen(t, "integrations", ptExpect(t, c.do("GET", endpoint, nil), http.StatusOK).JSON(t)["integrations"], 0)
	ptExpect(t, c.do("POST", endpoint+"/slack/test", nil), http.StatusNotFound)
	// Invalid stored credentials fail locally: task writes still succeed and record the failed attempt.
	_, err := app.pool.Exec(context.Background(), `INSERT INTO project_integrations (workspace_id,project_id,provider,secret) VALUES ($1,$2,'slack','invalid')`, ws, uuid.MustParse(project))
	if err != nil {
		t.Fatal(err)
	}
	taskBase := base + "/" + project + "/tasks"
	task := ptExpect(t, c.do("POST", taskBase, map[string]any{"title": "Notification task"}), http.StatusCreated).JSON(t)
	checkAttempt := func() {
		row := ptLen(t, "integrations", ptExpect(t, c.do("GET", endpoint, nil), http.StatusOK).JSON(t)["integrations"], 1)[0].(map[string]any)
		if row["last_attempt_at"] == nil || row["last_error"] == "" {
			t.Fatal("failed notification not recorded")
		}
		_, err := app.pool.Exec(context.Background(), `UPDATE project_integrations SET last_attempt_at=NULL,last_error='' WHERE workspace_id=$1 AND project_id=$2`, ws, uuid.MustParse(project))
		if err != nil {
			t.Fatal(err)
		}
	}
	checkAttempt()
	taskURL := taskBase + "/" + task["id"].(string)
	ptExpect(t, c.do("PATCH", taskURL, map[string]any{"status": "completed"}), http.StatusOK)
	checkAttempt()
	ptExpect(t, c.do("DELETE", taskURL, nil), http.StatusNoContent)
	checkAttempt()
	ptExpect(t, c.do("POST", base+"/"+project+"/archive", nil), http.StatusOK)
	ptExpect(t, c.do("PUT", endpoint+"/slack", payload), http.StatusNotFound)
	ptExpect(t, c.do("POST", endpoint+"/slack/test", nil), http.StatusConflict)
}
