package identity

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

const (
	SessionIdle   = 12 * time.Hour     // from last activity
	SessionMaxAge = 7 * 24 * time.Hour // from creation
)

// Store errors. Callers map these to HTTP/CLI responses; raw database errors are never exposed.
var (
	ErrConflict  = errors.New("record already exists or invariant violation")
	ErrNotFound  = errors.New("record not found")
	ErrForbidden = errors.New("operation forbidden")
)

// Querier is satisfied by both *pgxpool.Pool and pgx.Tx.
type Querier interface {
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// MapErr turns a unique violation into ErrConflict and no-rows into ErrNotFound.
func MapErr(err error) error {
	var pg *pgconn.PgError
	if errors.As(err, &pg) && pg.Code == "23505" {
		return ErrConflict
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	return err
}

// UserProjection never includes password_hash or other credential fields.
type UserProjection struct {
	ID          uuid.UUID
	Username    string
	DisplayName string
	Locale      string
	Timezone    string
	Disabled    bool
}

type MemberProjection struct {
	UserID      uuid.UUID
	Username    string
	DisplayName string
	Role        string
}

type WorkspaceProjection struct {
	ID   uuid.UUID
	Name string
	Slug string
	Role string
}

type ResolvedSession struct {
	SessionID     uuid.UUID
	User          UserProjection
	CSRFTokenHash string
}

func newUser(id uuid.UUID, username, displayName string) UserProjection {
	return UserProjection{ID: id, Username: username, DisplayName: displayName, Locale: "en", Timezone: "UTC"}
}

const userColumns = "id, username, display_name, locale, timezone, disabled_at"

func scanUser(row pgx.Row) (UserProjection, error) {
	var u UserProjection
	var disabledAt *time.Time
	err := row.Scan(&u.ID, &u.Username, &u.DisplayName, &u.Locale, &u.Timezone, &disabledAt)
	u.Disabled = disabledAt != nil
	return u, err
}

// ---------------------------------------------------------------------------
// Bootstrap
// ---------------------------------------------------------------------------

func AnyUserExists(ctx context.Context, db Querier) (bool, error) {
	var exists bool
	err := db.QueryRow(ctx, "SELECT EXISTS (SELECT 1 FROM users LIMIT 1)").Scan(&exists)
	return exists, err
}

// Bootstrap atomically creates the first user, a workspace and an owner membership.
func Bootstrap(ctx context.Context, pool *pgxpool.Pool, username, passwordHash, displayName, workspaceName, workspaceSlug string) (UserProjection, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return UserProjection{}, err
	}
	defer tx.Rollback(ctx)

	// Serialize concurrent bootstraps; the existence check below runs under this lock.
	if _, err := tx.Exec(ctx, "LOCK TABLE users IN SHARE ROW EXCLUSIVE MODE"); err != nil {
		return UserProjection{}, err
	}
	exists, err := AnyUserExists(ctx, tx)
	if err != nil {
		return UserProjection{}, err
	}
	if exists {
		return UserProjection{}, ErrConflict
	}

	userID := uuid.New()
	if _, err := tx.Exec(ctx, "INSERT INTO users (id, username, password_hash, display_name) VALUES ($1, $2, $3, $4)",
		userID, username, passwordHash, displayName); err != nil {
		return UserProjection{}, MapErr(err)
	}
	workspaceID := uuid.New()
	if _, err := tx.Exec(ctx, "INSERT INTO workspaces (id, name, slug) VALUES ($1, $2, $3)",
		workspaceID, workspaceName, workspaceSlug); err != nil {
		return UserProjection{}, MapErr(err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO workspace_memberships (workspace_id, user_id, role) VALUES ($1, $2, 'owner')",
		workspaceID, userID); err != nil {
		return UserProjection{}, MapErr(err)
	}
	if err := RecordAuditEvent(ctx, tx, nil, "user.bootstrap", map[string]any{}); err != nil {
		return UserProjection{}, err
	}
	return newUser(userID, username, displayName), tx.Commit(ctx)
}

// ---------------------------------------------------------------------------
// Users
// ---------------------------------------------------------------------------

// FindUserByUsername returns (id, password_hash) for an active user, or ok=false.
// Callers must run DummyVerify when ok is false to avoid timing side channels.
func FindUserByUsername(ctx context.Context, db Querier, username string) (id uuid.UUID, passwordHash string, ok bool, err error) {
	err = db.QueryRow(ctx, "SELECT id, password_hash FROM users WHERE username = $1 AND disabled_at IS NULL", username).
		Scan(&id, &passwordHash)
	if errors.Is(err, pgx.ErrNoRows) {
		return id, "", false, nil
	}
	return id, passwordHash, err == nil, err
}

func FindUserByID(ctx context.Context, db Querier, userID uuid.UUID) (UserProjection, error) {
	u, err := scanUser(db.QueryRow(ctx, "SELECT "+userColumns+" FROM users WHERE id = $1", userID))
	return u, MapErr(err)
}

// FindUserProjectionByUsername returns ErrNotFound when no user has that username.
func FindUserProjectionByUsername(ctx context.Context, db Querier, username string) (UserProjection, error) {
	u, err := scanUser(db.QueryRow(ctx, "SELECT "+userColumns+" FROM users WHERE username = $1", username))
	return u, MapErr(err)
}

// CreateUser provisions an account via the operator CLI, with no workspace membership.
func CreateUser(ctx context.Context, pool *pgxpool.Pool, username, passwordHash, displayName string) (UserProjection, error) {
	userID := uuid.New()
	tx, err := pool.Begin(ctx)
	if err != nil {
		return UserProjection{}, err
	}
	defer tx.Rollback(ctx)
	if _, err := tx.Exec(ctx, "INSERT INTO users (id, username, password_hash, display_name) VALUES ($1, $2, $3, $4)",
		userID, username, passwordHash, displayName); err != nil {
		return UserProjection{}, MapErr(err)
	}
	if err := RecordAuditEvent(ctx, tx, &userID, "user.provision_cli", map[string]any{"username": username}); err != nil {
		return UserProjection{}, err
	}
	return newUser(userID, username, displayName), tx.Commit(ctx)
}

// ResetUserPassword sets a new password hash and revokes all active sessions atomically.
func ResetUserPassword(ctx context.Context, pool *pgxpool.Pool, username, newPasswordHash string) (UserProjection, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return UserProjection{}, err
	}
	defer tx.Rollback(ctx)
	u, err := scanUser(tx.QueryRow(ctx, "SELECT "+userColumns+" FROM users WHERE username = $1 FOR UPDATE", username))
	if err != nil {
		return UserProjection{}, MapErr(err)
	}
	now := time.Now().UTC()
	if _, err := tx.Exec(ctx, "UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3", newPasswordHash, now, u.ID); err != nil {
		return UserProjection{}, err
	}
	if _, err := tx.Exec(ctx, "UPDATE sessions SET revoked_at = $1 WHERE user_id = $2 AND revoked_at IS NULL", now, u.ID); err != nil {
		return UserProjection{}, err
	}
	if err := RecordAuditEvent(ctx, tx, &u.ID, "user.password_reset_cli", map[string]any{"username": username}); err != nil {
		return UserProjection{}, err
	}
	return u, tx.Commit(ctx)
}

// SetUserDisabled disables (revoking all sessions) or enables an account.
// Disabling returns ErrConflict when the user is the last active owner of any workspace.
func SetUserDisabled(ctx context.Context, pool *pgxpool.Pool, username string, disabled bool) (UserProjection, error) {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return UserProjection{}, err
	}
	defer tx.Rollback(ctx)
	u, err := scanUser(tx.QueryRow(ctx, "SELECT "+userColumns+" FROM users WHERE username = $1 FOR UPDATE", username))
	if err != nil {
		return UserProjection{}, MapErr(err)
	}
	now := time.Now().UTC()
	if disabled {
		var soleOwner bool
		if err := tx.QueryRow(ctx, `SELECT EXISTS (
			 SELECT 1 FROM workspace_memberships wm
			 WHERE wm.user_id = $1 AND wm.role = 'owner'
			   AND (
			       SELECT COUNT(*)
			       FROM workspace_memberships wm2
			       JOIN users u2 ON u2.id = wm2.user_id
			       WHERE wm2.workspace_id = wm.workspace_id
			         AND wm2.role = 'owner'
			         AND wm2.user_id != $1
			         AND u2.disabled_at IS NULL
			   ) = 0
		 )`, u.ID).Scan(&soleOwner); err != nil {
			return UserProjection{}, err
		}
		if soleOwner {
			return UserProjection{}, ErrConflict
		}
		if _, err := tx.Exec(ctx, "UPDATE users SET disabled_at = $1, updated_at = $1 WHERE id = $2", now, u.ID); err != nil {
			return UserProjection{}, err
		}
		if _, err := tx.Exec(ctx, "UPDATE sessions SET revoked_at = $1 WHERE user_id = $2 AND revoked_at IS NULL", now, u.ID); err != nil {
			return UserProjection{}, err
		}
		if err := RecordAuditEvent(ctx, tx, &u.ID, "user.disabled_cli", map[string]any{"username": username}); err != nil {
			return UserProjection{}, err
		}
	} else {
		if _, err := tx.Exec(ctx, "UPDATE users SET disabled_at = NULL, updated_at = $1 WHERE id = $2", now, u.ID); err != nil {
			return UserProjection{}, err
		}
		if err := RecordAuditEvent(ctx, tx, &u.ID, "user.enabled_cli", map[string]any{"username": username}); err != nil {
			return UserProjection{}, err
		}
	}
	u.Disabled = disabled
	return u, tx.Commit(ctx)
}

// ---------------------------------------------------------------------------
// Sessions
// ---------------------------------------------------------------------------

// CreateSession returns the raw session and CSRF tokens. They are delivered by
// cookie and never stored or logged; only their hashes are persisted.
func CreateSession(ctx context.Context, db Querier, userID uuid.UUID) (sessionToken, csrfToken string, err error) {
	sessionToken, sessionHash, err := GenerateToken()
	if err != nil {
		return "", "", err
	}
	csrfToken, csrfHash, err := GenerateToken()
	if err != nil {
		return "", "", err
	}
	now := time.Now().UTC()
	_, err = db.Exec(ctx, `INSERT INTO sessions
		(token_hash, user_id, csrf_token_hash, created_at, last_active_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		sessionHash, userID, csrfHash, now, now, now.Add(SessionMaxAge))
	return sessionToken, csrfToken, err
}

// ResolveSession returns nil when the token is invalid, expired, idle-expired,
// revoked, or belongs to a disabled user. It refreshes last_active_at on success.
func ResolveSession(ctx context.Context, db Querier, rawToken string) (*ResolvedSession, error) {
	hash, err := HashClientToken(rawToken)
	if err != nil {
		return nil, nil
	}
	now := time.Now().UTC()
	var s ResolvedSession
	var disabledAt *time.Time
	var lastActive time.Time
	err = db.QueryRow(ctx, `SELECT s.id, s.csrf_token_hash, s.last_active_at, u.id, u.username, u.display_name, u.locale, u.timezone, u.disabled_at
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.token_hash = $1
		  AND s.revoked_at IS NULL
		  AND s.expires_at > $2
		  AND s.last_active_at > $3`, hash, now, now.Add(-SessionIdle)).
		Scan(&s.SessionID, &s.CSRFTokenHash, &lastActive, &s.User.ID, &s.User.Username, &s.User.DisplayName, &s.User.Locale, &s.User.Timezone, &disabledAt)
	if errors.Is(err, pgx.ErrNoRows) || disabledAt != nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	// Best-effort; never fail the request over this. Refreshing at most once a minute
	// avoids a row write per request; the 12-hour idle limit makes that precision enough.
	if now.Sub(lastActive) > time.Minute {
		_, _ = db.Exec(ctx, "UPDATE sessions SET last_active_at = $1 WHERE id = $2", now, s.SessionID)
	}
	return &s, nil
}

// RevokeSession succeeds whether or not the session existed.
func RevokeSession(ctx context.Context, db Querier, sessionID uuid.UUID) error {
	_, err := db.Exec(ctx, "UPDATE sessions SET revoked_at = $1 WHERE id = $2 AND revoked_at IS NULL", time.Now().UTC(), sessionID)
	return err
}

// ---------------------------------------------------------------------------
// Workspaces
// ---------------------------------------------------------------------------

func ListUserWorkspaces(ctx context.Context, db Querier, userID uuid.UUID) ([]WorkspaceProjection, error) {
	rows, err := db.Query(ctx, `SELECT w.id, w.name, w.slug, wm.role
		FROM workspaces w
		JOIN workspace_memberships wm ON wm.workspace_id = w.id
		WHERE wm.user_id = $1
		ORDER BY w.name`, userID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (WorkspaceProjection, error) {
		var w WorkspaceProjection
		return w, r.Scan(&w.ID, &w.Name, &w.Slug, &w.Role)
	})
}

// GetUserMembership returns the active user's role, or "" when they are not a member.
func GetUserMembership(ctx context.Context, db Querier, workspaceID, userID uuid.UUID) (string, error) {
	var role string
	err := db.QueryRow(ctx, `SELECT wm.role
		FROM workspace_memberships wm
		JOIN users u ON u.id = wm.user_id
		WHERE wm.workspace_id = $1 AND wm.user_id = $2 AND u.disabled_at IS NULL`, workspaceID, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", nil
	}
	return role, err
}

func ListWorkspaceMembers(ctx context.Context, db Querier, workspaceID uuid.UUID) ([]MemberProjection, error) {
	rows, err := db.Query(ctx, `SELECT u.id, u.username, u.display_name, wm.role
		FROM workspace_memberships wm
		JOIN users u ON u.id = wm.user_id
		WHERE wm.workspace_id = $1 AND u.disabled_at IS NULL
		ORDER BY
		   CASE wm.role
		       WHEN 'owner' THEN 1
		       WHEN 'admin' THEN 2
		       WHEN 'member' THEN 3
		       WHEN 'guest' THEN 4
		       ELSE 5
		   END,
		   u.username ASC`, workspaceID)
	if err != nil {
		return nil, err
	}
	return pgx.CollectRows(rows, func(r pgx.CollectableRow) (MemberProjection, error) {
		var m MemberProjection
		return m, r.Scan(&m.UserID, &m.Username, &m.DisplayName, &m.Role)
	})
}

func assignableRole(role string) bool { return role == "admin" || role == "member" || role == "guest" }

// AddWorkspaceMember returns ErrConflict if already a member, ErrForbidden for a non-assignable role.
func AddWorkspaceMember(ctx context.Context, pool *pgxpool.Pool, actorID *uuid.UUID, workspaceID, targetUserID uuid.UUID, role string) (MemberProjection, error) {
	if !assignableRole(role) {
		return MemberProjection{}, ErrForbidden
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return MemberProjection{}, err
	}
	defer tx.Rollback(ctx)
	m := MemberProjection{Role: role}
	if err := tx.QueryRow(ctx, "SELECT id, username, display_name FROM users WHERE id = $1 AND disabled_at IS NULL", targetUserID).
		Scan(&m.UserID, &m.Username, &m.DisplayName); err != nil {
		return MemberProjection{}, MapErr(err)
	}
	if _, err := tx.Exec(ctx, "INSERT INTO workspace_memberships (workspace_id, user_id, role) VALUES ($1, $2, $3)",
		workspaceID, targetUserID, role); err != nil {
		return MemberProjection{}, MapErr(err)
	}
	if err := RecordAuditEvent(ctx, tx, actorID, "workspace.member_added", map[string]any{
		"workspace_id": workspaceID, "target_user_id": targetUserID, "role": role,
	}); err != nil {
		return MemberProjection{}, err
	}
	return m, tx.Commit(ctx)
}

// lockMembership locks the workspace row (serializing last-owner checks) and returns the target's role.
func lockMembership(ctx context.Context, tx pgx.Tx, workspaceID, userID uuid.UUID, missing error) (string, error) {
	var id uuid.UUID
	if err := tx.QueryRow(ctx, "SELECT id FROM workspaces WHERE id = $1 FOR UPDATE", workspaceID).Scan(&id); err != nil {
		return "", MapErr(err)
	}
	var role string
	err := tx.QueryRow(ctx, "SELECT role FROM workspace_memberships WHERE workspace_id = $1 AND user_id = $2 FOR UPDATE",
		workspaceID, userID).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", missing
	}
	return role, err
}

func otherActiveOwners(ctx context.Context, tx pgx.Tx, workspaceID, userID uuid.UUID) (int64, error) {
	var n int64
	err := tx.QueryRow(ctx, `SELECT COUNT(*)
		FROM workspace_memberships wm
		JOIN users u ON u.id = wm.user_id
		WHERE wm.workspace_id = $1 AND wm.role = 'owner'
		  AND wm.user_id != $2 AND u.disabled_at IS NULL`, workspaceID, userID).Scan(&n)
	return n, err
}

// UpdateWorkspaceMemberRole returns ErrConflict when demoting the last active owner.
func UpdateWorkspaceMemberRole(ctx context.Context, pool *pgxpool.Pool, actorID *uuid.UUID, workspaceID, targetUserID uuid.UUID, newRole string) (MemberProjection, error) {
	if !assignableRole(newRole) {
		return MemberProjection{}, ErrForbidden
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return MemberProjection{}, err
	}
	defer tx.Rollback(ctx)
	current, err := lockMembership(ctx, tx, workspaceID, targetUserID, ErrNotFound)
	if err != nil {
		return MemberProjection{}, err
	}
	if current == "owner" {
		n, err := otherActiveOwners(ctx, tx, workspaceID, targetUserID)
		if err != nil {
			return MemberProjection{}, err
		}
		if n == 0 {
			return MemberProjection{}, ErrConflict
		}
	}
	if _, err := tx.Exec(ctx, "UPDATE workspace_memberships SET role = $1, updated_at = $2 WHERE workspace_id = $3 AND user_id = $4",
		newRole, time.Now().UTC(), workspaceID, targetUserID); err != nil {
		return MemberProjection{}, err
	}
	m := MemberProjection{UserID: targetUserID, Role: newRole}
	if err := tx.QueryRow(ctx, "SELECT username, display_name FROM users WHERE id = $1", targetUserID).Scan(&m.Username, &m.DisplayName); err != nil {
		return MemberProjection{}, err
	}
	if err := RecordAuditEvent(ctx, tx, actorID, "workspace.member_role_updated", map[string]any{
		"workspace_id": workspaceID, "target_user_id": targetUserID, "old_role": current, "new_role": newRole,
	}); err != nil {
		return MemberProjection{}, err
	}
	return m, tx.Commit(ctx)
}

// RemoveWorkspaceMember returns ErrConflict when removing the last active owner.
func RemoveWorkspaceMember(ctx context.Context, pool *pgxpool.Pool, actorID *uuid.UUID, workspaceID, targetUserID uuid.UUID) error {
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	current, err := lockMembership(ctx, tx, workspaceID, targetUserID, ErrNotFound)
	if err != nil {
		return err
	}
	if current == "owner" {
		n, err := otherActiveOwners(ctx, tx, workspaceID, targetUserID)
		if err != nil {
			return err
		}
		if n == 0 {
			return ErrConflict
		}
	}
	if _, err := tx.Exec(ctx, "DELETE FROM workspace_memberships WHERE workspace_id = $1 AND user_id = $2", workspaceID, targetUserID); err != nil {
		return err
	}
	if err := RecordAuditEvent(ctx, tx, actorID, "workspace.member_removed", map[string]any{
		"workspace_id": workspaceID, "target_user_id": targetUserID,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// TransferWorkspaceOwnership demotes the current owner to admin and promotes the target to owner.
func TransferWorkspaceOwnership(ctx context.Context, pool *pgxpool.Pool, actorID *uuid.UUID, workspaceID, currentOwnerID, targetUserID uuid.UUID) error {
	if currentOwnerID == targetUserID {
		return nil
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	role, err := lockMembership(ctx, tx, workspaceID, currentOwnerID, ErrForbidden)
	if err != nil {
		return err
	}
	if role != "owner" {
		return ErrForbidden
	}
	var targetActive bool
	if err := tx.QueryRow(ctx, `SELECT EXISTS (
		 SELECT 1 FROM workspace_memberships wm
		 JOIN users u ON u.id = wm.user_id
		 WHERE wm.workspace_id = $1 AND wm.user_id = $2
		   AND u.disabled_at IS NULL
	 )`, workspaceID, targetUserID).Scan(&targetActive); err != nil {
		return err
	}
	if !targetActive {
		return ErrNotFound
	}
	now := time.Now().UTC()
	if _, err := tx.Exec(ctx, "UPDATE workspace_memberships SET role = 'admin', updated_at = $1 WHERE workspace_id = $2 AND user_id = $3",
		now, workspaceID, currentOwnerID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, "UPDATE workspace_memberships SET role = 'owner', updated_at = $1 WHERE workspace_id = $2 AND user_id = $3",
		now, workspaceID, targetUserID); err != nil {
		return err
	}
	if err := RecordAuditEvent(ctx, tx, actorID, "workspace.ownership_transferred", map[string]any{
		"workspace_id": workspaceID, "previous_owner_id": currentOwnerID, "new_owner_id": targetUserID,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

// ---------------------------------------------------------------------------
// Audit
// ---------------------------------------------------------------------------

// RecordAuditEvent stores a security event. Metadata must never contain passwords, raw tokens or hashes.
func RecordAuditEvent(ctx context.Context, db Querier, actorID *uuid.UUID, eventType string, metadata map[string]any) error {
	b, err := json.Marshal(metadata)
	if err != nil {
		return err
	}
	_, err = db.Exec(ctx, "INSERT INTO security_audit_events (actor_id, event_type, metadata) VALUES ($1, $2, $3)",
		actorID, eventType, b)
	return err
}
