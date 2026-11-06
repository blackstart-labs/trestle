package httpapi

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/maruf-pfc/trestle/internal/identity"
)

// idHash hashes a password with the app's (fast, test) Argon2 parameters.
func (a *testApp) idHash(password string) string {
	a.t.Helper()
	h, err := identity.NewHasher(a.cfg.Argon2MCost, a.cfg.Argon2TCost, a.cfg.Argon2Parallelism)
	if err != nil {
		a.t.Fatal(err)
	}
	hash, err := h.Hash([]byte(password))
	if err != nil {
		a.t.Fatal(err)
	}
	return hash
}

func (a *testApp) idBootstrap(username, password, displayName, wsName, wsSlug string) identity.UserProjection {
	a.t.Helper()
	u, err := identity.Bootstrap(context.Background(), a.pool, username, a.idHash(password), displayName, wsName, wsSlug)
	if err != nil {
		a.t.Fatal(err)
	}
	return u
}

func idLoginStatus(a *testApp, username, password string) int {
	a.t.Helper()
	return a.anon().do("POST", "/api/v1/session", map[string]string{"username": username, "password": password}).StatusCode
}

func idWantStatus(t *testing.T, step string, res testResponse, want int) {
	t.Helper()
	if res.StatusCode != want {
		t.Fatalf("%s: status %d, want %d (%s)", step, res.StatusCode, want, res.Body)
	}
}

func TestHTTPLoginSuccessAndLogoutFlow(t *testing.T) {
	app := newApp(t)
	app.idBootstrap("carol", "valid-password-123", "Carol Danvers", "Carol Base", "carol-base")

	idWantStatus(t, "anonymous /me", app.anon().do("GET", "/api/v1/me", nil), http.StatusUnauthorized)
	idWantStatus(t, "anonymous /workspaces", app.anon().do("GET", "/api/v1/workspaces", nil), http.StatusUnauthorized)

	// Mixed case normalizes to lowercase.
	res := app.anon().do("POST", "/api/v1/session", map[string]string{"username": "Carol", "password": "valid-password-123"})
	idWantStatus(t, "login", res, http.StatusOK)
	var session, csrf string
	for _, sc := range res.Header.Values("Set-Cookie") {
		if v, ok := extractCookie(sc, "trestle-session"); ok {
			session = v
		}
		if v, ok := extractCookie(sc, "trestle-csrf"); ok {
			csrf = v
		}
	}
	if session == "" || csrf == "" {
		t.Fatalf("session and csrf cookies must be set: %v", res.Header.Values("Set-Cookie"))
	}
	noCSRF := &client{app: app, cookies: "trestle-session=" + session + "; trestle-csrf=" + csrf}
	withCSRF := &client{app: app, cookies: noCSRF.cookies, csrf: csrf}

	me := noCSRF.do("GET", "/api/v1/me", nil)
	idWantStatus(t, "/me", me, http.StatusOK)
	if j := me.JSON(t); j["username"] != "carol" || j["display_name"] != "Carol Danvers" {
		t.Errorf("/me %v", j)
	}

	ws := noCSRF.do("GET", "/api/v1/workspaces", nil)
	idWantStatus(t, "/workspaces", ws, http.StatusOK)
	list, _ := ws.JSON(t)["workspaces"].([]any)
	if len(list) == 0 {
		t.Fatalf("no workspaces: %s", ws.Body)
	}
	if first := list[0].(map[string]any); first["slug"] != "carol-base" || first["role"] != "owner" {
		t.Errorf("workspace %v", first)
	}

	idWantStatus(t, "logout without csrf", noCSRF.do("DELETE", "/api/v1/session", nil), http.StatusForbidden)
	logout := withCSRF.do("DELETE", "/api/v1/session", nil)
	if logout.StatusCode != http.StatusOK && logout.StatusCode != http.StatusNoContent {
		t.Fatalf("logout: %d %s", logout.StatusCode, logout.Body)
	}
	idWantStatus(t, "/me after logout", noCSRF.do("GET", "/api/v1/me", nil), http.StatusUnauthorized)
}

func TestHTTPLoginRateLimiting(t *testing.T) {
	app := newApp(t)
	body := map[string]string{"username": "ratelimited_user", "password": "wrong-password"}
	for range 5 {
		idWantStatus(t, "failed login", app.anon().do("POST", "/api/v1/session", body), http.StatusUnauthorized)
	}
	sixth := app.anon().do("POST", "/api/v1/session", body)
	idWantStatus(t, "sixth attempt", sixth, http.StatusTooManyRequests)
	if sixth.Header.Get("Retry-After") == "" {
		t.Error("Retry-After missing")
	}
}

func TestOperatorUserProvisioningAndRecovery(t *testing.T) {
	app := newApp(t)
	ctx := context.Background()

	user := app.createUser("operator_user", "initial-password-123", "Operator User")
	if user.Username != "operator_user" || user.Disabled {
		t.Errorf("user %+v", user)
	}
	if _, err := identity.CreateUser(ctx, app.pool, "operator_user", app.idHash("initial-password-123"), "Another User"); !errors.Is(err, identity.ErrConflict) {
		t.Errorf("duplicate user: %v", err)
	}
	if ws, err := identity.ListUserWorkspaces(ctx, app.pool, user.ID); err != nil || len(ws) != 0 {
		t.Errorf("new user has workspaces: %+v %v", ws, err)
	}

	first := app.login("operator_user", "initial-password-123")
	idWantStatus(t, "/me", first.do("GET", "/api/v1/me", nil), http.StatusOK)

	// Reset revokes every active session.
	reset, err := identity.ResetUserPassword(ctx, app.pool, "operator_user", app.idHash("recovered-password-456"))
	if err != nil || reset.ID != user.ID {
		t.Fatalf("reset: %+v %v", reset, err)
	}
	idWantStatus(t, "/me after reset", first.do("GET", "/api/v1/me", nil), http.StatusUnauthorized)
	if s := idLoginStatus(app, "operator_user", "initial-password-123"); s != http.StatusUnauthorized {
		t.Errorf("old password: %d", s)
	}
	second := app.login("operator_user", "recovered-password-456")

	// Disabling invalidates the session and prevents login.
	if _, err := identity.SetUserDisabled(ctx, app.pool, "operator_user", true); err != nil {
		t.Fatal(err)
	}
	idWantStatus(t, "/me when disabled", second.do("GET", "/api/v1/me", nil), http.StatusUnauthorized)
	if s := idLoginStatus(app, "operator_user", "recovered-password-456"); s != http.StatusUnauthorized {
		t.Errorf("login when disabled: %d", s)
	}

	if _, err := identity.SetUserDisabled(ctx, app.pool, "operator_user", false); err != nil {
		t.Fatal(err)
	}
	if s := idLoginStatus(app, "operator_user", "recovered-password-456"); s != http.StatusOK {
		t.Errorf("login when re-enabled: %d", s)
	}
}

func TestWorkspaceMembershipsLifecycleAndAuthorization(t *testing.T) {
	app := newApp(t)
	ctx := context.Background()

	owner := app.idBootstrap("owner_user", "owner-password-123", "Workspace Owner", "Trestle HQ", "trestle-hq")
	workspaces, err := identity.ListUserWorkspaces(ctx, app.pool, owner.ID)
	if err != nil {
		t.Fatal(err)
	}
	wsID := workspaces[0].ID
	alice := app.createUser("alice", "alice-password-123", "Alice Cooper")
	bob := app.createUser("bob", "bob-password-123", "Bob Builder")
	app.createUser("charlie", "charlie-password-123", "Charlie Outsider")

	ownerC := app.login("owner_user", "owner-password-123")
	charlieC := app.login("charlie", "charlie-password-123")
	members := "/api/v1/workspaces/" + wsID.String() + "/members"
	ownerMember := members + "/" + owner.ID.String()

	// A non-member gets a concealed 404.
	idWantStatus(t, "charlie lists members", charlieC.do("GET", members, nil), http.StatusNotFound)
	idWantStatus(t, "owner lists members", ownerC.do("GET", members, nil), http.StatusOK)

	addAlice := map[string]string{"username": "alice", "role": "admin"}
	idWantStatus(t, "owner adds alice", ownerC.do("POST", members, addAlice), http.StatusCreated)
	idWantStatus(t, "owner adds alice again", ownerC.do("POST", members, addAlice), http.StatusConflict)

	aliceC := app.login("alice", "alice-password-123")
	idWantStatus(t, "admin adds bob", aliceC.do("POST", members, map[string]string{"username": "bob", "role": "member"}), http.StatusCreated)
	idWantStatus(t, "admin adds charlie as owner", aliceC.do("POST", members, map[string]string{"username": "charlie", "role": "owner"}), http.StatusBadRequest)

	demote := map[string]string{"role": "member"}
	idWantStatus(t, "admin demotes owner", aliceC.do("PATCH", ownerMember, demote), http.StatusForbidden)
	idWantStatus(t, "admin removes owner", aliceC.do("DELETE", ownerMember, nil), http.StatusForbidden)

	// Last active owner protection.
	idWantStatus(t, "sole owner demotes self", ownerC.do("PATCH", ownerMember, demote), http.StatusConflict)
	idWantStatus(t, "sole owner removes self", ownerC.do("DELETE", ownerMember, nil), http.StatusConflict)
	if _, err := identity.SetUserDisabled(ctx, app.pool, "owner_user", true); !errors.Is(err, identity.ErrConflict) {
		t.Errorf("disable sole owner: %v", err)
	}

	// Ownership transfer requires reauthentication.
	transfer := "/api/v1/workspaces/" + wsID.String() + "/transfer-ownership"
	idWantStatus(t, "transfer with bad password",
		ownerC.do("POST", transfer, map[string]any{"target_user_id": alice.ID, "password": "wrong-password"}), http.StatusUnauthorized)
	idWantStatus(t, "transfer",
		ownerC.do("POST", transfer, map[string]any{"target_user_id": alice.ID, "password": "owner-password-123"}), http.StatusOK)

	for _, c := range []struct {
		user identity.UserProjection
		want string
	}{{alice, "owner"}, {owner, "admin"}} {
		if role, err := identity.GetUserMembership(ctx, app.pool, wsID, c.user.ID); err != nil || role != c.want {
			t.Errorf("role of %s = %q (%v), want %q", c.user.Username, role, err, c.want)
		}
	}

	idWantStatus(t, "new owner removes bob", aliceC.do("DELETE", members+"/"+bob.ID.String(), nil), http.StatusNoContent)
	if role, err := identity.GetUserMembership(ctx, app.pool, wsID, bob.ID); err != nil || role != "" {
		t.Errorf("bob still a member: %q %v", role, err)
	}
}
