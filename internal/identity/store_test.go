package identity

import (
	"context"
	"errors"
	"testing"

	"github.com/maruf-pfc/trestle/internal/testdb"
)

func TestBootstrapCreatesUserWorkspaceAndOwnerMembershipAtomically(t *testing.T) {
	pool, _ := testdb.Pool(t, true)
	ctx := context.Background()

	if exists, err := AnyUserExists(ctx, pool); err != nil || exists {
		t.Fatalf("users exist before bootstrap: %v %v", exists, err)
	}
	hash := mustHash(t, testHasher(t), "correct-horse-battery-staple")
	user, err := Bootstrap(ctx, pool, "admin", hash, "Administrator", "Acme Corp", "acme")
	if err != nil {
		t.Fatal(err)
	}
	if user.Username != "admin" || user.DisplayName != "Administrator" {
		t.Errorf("user %+v", user)
	}
	workspaces, err := ListUserWorkspaces(ctx, pool, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(workspaces) != 1 || workspaces[0].Slug != "acme" || workspaces[0].Name != "Acme Corp" || workspaces[0].Role != "owner" {
		t.Errorf("workspaces %+v", workspaces)
	}
	if exists, err := AnyUserExists(ctx, pool); err != nil || !exists {
		t.Errorf("no user after bootstrap: %v %v", exists, err)
	}
	// A second bootstrap must fail with ErrConflict.
	if _, err := Bootstrap(ctx, pool, "operator", hash, "Operator", "Second Org", "second"); !errors.Is(err, ErrConflict) {
		t.Errorf("second bootstrap: %v", err)
	}
}

func TestSessionCreationResolutionAndRevocation(t *testing.T) {
	pool, _ := testdb.Pool(t, true)
	ctx := context.Background()
	hash := mustHash(t, testHasher(t), "password123")
	user, err := Bootstrap(ctx, pool, "alice", hash, "Alice Smith", "Alice Workspace", "alice-ws")
	if err != nil {
		t.Fatal(err)
	}
	sessionToken, csrfToken, err := CreateSession(ctx, pool, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolveSession(ctx, pool, sessionToken)
	if err != nil || resolved == nil {
		t.Fatalf("session must resolve: %v", err)
	}
	if resolved.User.ID != user.ID || resolved.User.Username != "alice" {
		t.Errorf("resolved user %+v", resolved.User)
	}
	wantCSRF, err := HashClientToken(csrfToken)
	if err != nil || resolved.CSRFTokenHash != wantCSRF {
		t.Errorf("csrf hash %q, want %q (%v)", resolved.CSRFTokenHash, wantCSRF, err)
	}
	if err := RevokeSession(ctx, pool, resolved.SessionID); err != nil {
		t.Fatal(err)
	}
	if after, err := ResolveSession(ctx, pool, sessionToken); err != nil || after != nil {
		t.Errorf("revoked session resolves: %+v %v", after, err)
	}
}

func TestWorkspaceListIsScopedToUserMemberships(t *testing.T) {
	pool, _ := testdb.Pool(t, true)
	ctx := context.Background()
	hash := mustHash(t, testHasher(t), "password123")
	user, err := Bootstrap(ctx, pool, "bob", hash, "Bob Jones", "Bob Org", "bob-org")
	if err != nil {
		t.Fatal(err)
	}
	workspaces, err := ListUserWorkspaces(ctx, pool, user.ID)
	if err != nil {
		t.Fatal(err)
	}
	if len(workspaces) != 1 || workspaces[0].Slug != "bob-org" || workspaces[0].Role != "owner" {
		t.Errorf("workspaces %+v", workspaces)
	}
}

func TestConcurrentLastOwnerRemovalRacePrevention(t *testing.T) {
	pool, _ := testdb.Pool(t, true)
	ctx := context.Background()
	hash := mustHash(t, testHasher(t), "password-123")
	owner1, err := Bootstrap(ctx, pool, "owner_one", hash, "Owner One", "Dual Owner Workspace", "dual-owner")
	if err != nil {
		t.Fatal(err)
	}
	workspaces, err := ListUserWorkspaces(ctx, pool, owner1.ID)
	if err != nil {
		t.Fatal(err)
	}
	wsID := workspaces[0].ID
	owner2, err := CreateUser(ctx, pool, "owner_two", hash, "Owner Two")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := pool.Exec(ctx, "INSERT INTO workspace_memberships (workspace_id, user_id, role) VALUES ($1, $2, 'owner')", wsID, owner2.ID); err != nil {
		t.Fatal(err)
	}

	results := make(chan error, 2)
	for _, owner := range []UserProjection{owner1, owner2} {
		go func() { results <- RemoveWorkspaceMember(ctx, pool, &owner.ID, wsID, owner.ID) }()
	}
	var ok, conflicts int
	for range 2 {
		switch err := <-results; {
		case err == nil:
			ok++
		case errors.Is(err, ErrConflict):
			conflicts++
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	if ok != 1 {
		t.Errorf("exactly one removal should succeed, got %d", ok)
	}
	if conflicts != 1 {
		t.Errorf("one removal must fail with ErrConflict to preserve the last owner, got %d", conflicts)
	}

	members, err := ListWorkspaceMembers(ctx, pool, wsID)
	if err != nil {
		t.Fatal(err)
	}
	owners := 0
	for _, m := range members {
		if m.Role == "owner" {
			owners++
		}
	}
	if owners != 1 {
		t.Errorf("workspace must retain exactly 1 active owner, has %d", owners)
	}
}
