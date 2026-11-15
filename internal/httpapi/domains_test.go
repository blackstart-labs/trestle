package httpapi

import (
	"context"
	"net/http"
	"testing"

	"github.com/maruf-pfc/trestle/internal/identity"
)

func TestInstanceDomainRequiresExplicitOperatorAndGuardsConcurrentEdits(t *testing.T) {
	app := newApp(t)
	const password = "valid-password-123"
	const path = "/api/v1/instance/domain"
	app.idBootstrap("operator", password, "Operator", "Original", "original")
	other := app.createUser("other-owner", password, "Other Owner")
	if _, err := identity.CreateWorkspace(context.Background(), app.pool, other.ID, "Other", "other"); err != nil {
		t.Fatal(err)
	}
	operator := app.login("operator", password)
	owner := app.login("other-owner", password)
	input := map[string]string{"previous_origin": "", "public_origin": "https://Trestle.Example/"}
	idWantStatus(t, "anonymous read", app.anon().do("GET", path, nil), http.StatusUnauthorized)
	idWantStatus(t, "workspace owner read", owner.do("GET", path, nil), http.StatusForbidden)
	idWantStatus(t, "workspace owner write", owner.do("PUT", path, input), http.StatusForbidden)
	noCSRF := *operator
	noCSRF.csrf = ""
	idWantStatus(t, "operator write without CSRF", noCSRF.do("PUT", path, input), http.StatusForbidden)
	idWantStatus(t, "operator read", operator.do("GET", path, nil), http.StatusOK)
	res := operator.do("PUT", path, input)
	idWantStatus(t, "operator save", res, http.StatusOK)
	if res.JSON(t)["public_origin"] != "https://trestle.example" {
		t.Fatalf("unexpected response: %s", res.Body)
	}
	idWantStatus(t, "stale save", operator.do("PUT", path, input), http.StatusConflict)
	if err := identity.SetInstanceOperator(context.Background(), app.pool, other.Username, true); err != nil {
		t.Fatal(err)
	}
	idWantStatus(t, "granted access in existing session", owner.do("GET", path, nil), http.StatusOK)
	if err := identity.SetInstanceOperator(context.Background(), app.pool, other.Username, false); err != nil {
		t.Fatal(err)
	}
	idWantStatus(t, "revoked access in existing session", owner.do("GET", path, nil), http.StatusForbidden)
}
