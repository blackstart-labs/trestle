/// Database operations for identity, sessions, workspaces, and memberships.
///
/// All SQL is explicit and parameterized via SQLx. This module never returns
/// raw database errors to callers; it maps to domain-level results.
/// Token hashes and password hashes are opaque strings; this module does not
/// interpret their content beyond storing and retrieving them.
use sqlx::PgPool;
use time::OffsetDateTime;
use uuid::Uuid;

/// Session idle lifetime (12 hours from last activity).
pub const SESSION_IDLE_HOURS: i64 = 12;
/// Session absolute lifetime (7 days from creation).
pub const SESSION_MAX_DAYS: i64 = 7;

/// A user record projected for safe API and CLI consumption.
/// Never includes password_hash or any internal credential field.
#[derive(Debug, Clone, serde::Serialize)]
pub struct UserProjection {
    pub id: Uuid,
    pub username: String,
    pub display_name: String,
    pub locale: String,
    pub timezone: String,
    pub disabled: bool,
}

/// A workspace membership projection for authorized member views.
#[derive(Debug, Clone, serde::Serialize)]
pub struct MemberProjection {
    pub user_id: Uuid,
    pub username: String,
    pub display_name: String,
    pub role: String,
}

/// A workspace projection for the authorized workspace list.
#[derive(Debug, Clone, serde::Serialize)]
pub struct WorkspaceProjection {
    pub id: Uuid,
    pub name: String,
    pub slug: String,
    pub role: String,
}

/// An active session's resolved user and CSRF token hash.
#[derive(Debug, Clone)]
pub struct ResolvedSession {
    pub session_id: Uuid,
    pub user: UserProjection,
    /// The stored CSRF token hash for validation.
    pub csrf_token_hash: String,
}

/// Errors from identity store operations. Callers map these to HTTP/CLI responses.
#[derive(Debug)]
pub enum StoreError {
    /// A uniqueness constraint was violated (e.g., duplicate username or slug)
    /// or a domain invariant would be broken (e.g., removing the last owner).
    Conflict,
    /// No matching record was found.
    NotFound,
    /// The requested operation is forbidden by policy.
    Forbidden,
    /// An unexpected database error occurred. Details are logged, not exposed.
    Internal(sqlx::Error),
}

impl std::fmt::Display for StoreError {
    fn fmt(&self, f: &mut std::fmt::Formatter<'_>) -> std::fmt::Result {
        match self {
            Self::Conflict => f.write_str("record already exists or invariant violation"),
            Self::NotFound => f.write_str("record not found"),
            Self::Forbidden => f.write_str("operation forbidden"),
            Self::Internal(_) => f.write_str("internal database error"),
        }
    }
}

impl std::error::Error for StoreError {
    fn source(&self) -> Option<&(dyn std::error::Error + 'static)> {
        match self {
            Self::Internal(e) => Some(e),
            _ => None,
        }
    }
}

fn map_sqlx(err: sqlx::Error) -> StoreError {
    if let Some(db_err) = err.as_database_error() {
        // PostgreSQL unique violation code.
        if db_err.code().as_deref() == Some("23505") {
            return StoreError::Conflict;
        }
    }
    StoreError::Internal(err)
}

// ---------------------------------------------------------------------------
// Bootstrap
// ---------------------------------------------------------------------------

/// Check whether any user exists. Used by bootstrap to detect an initialized instance.
pub async fn any_user_exists(pool: &PgPool) -> Result<bool, StoreError> {
    let row = sqlx::query_scalar::<_, bool>("SELECT EXISTS (SELECT 1 FROM users LIMIT 1)")
        .fetch_one(pool)
        .await
        .map_err(StoreError::Internal)?;
    Ok(row)
}

/// Create the first user, a default workspace, and an owner membership atomically.
///
/// Fails if any user already exists (checked inside the transaction).
/// Returns the created user projection.
pub async fn bootstrap(
    pool: &PgPool,
    username: &str,
    password_hash: &str,
    display_name: &str,
    workspace_name: &str,
    workspace_slug: &str,
) -> Result<UserProjection, StoreError> {
    let mut tx = pool.begin().await.map_err(StoreError::Internal)?;

    // Atomic guard: refuse if any user exists.
    let exists: bool = sqlx::query_scalar("SELECT EXISTS (SELECT 1 FROM users LIMIT 1) FOR UPDATE")
        .fetch_one(&mut *tx)
        .await
        .map_err(StoreError::Internal)?;
    if exists {
        return Err(StoreError::Conflict);
    }

    let user_id = Uuid::new_v4();
    sqlx::query(
        "INSERT INTO users (id, username, password_hash, display_name)
         VALUES ($1, $2, $3, $4)",
    )
    .bind(user_id)
    .bind(username)
    .bind(password_hash)
    .bind(display_name)
    .execute(&mut *tx)
    .await
    .map_err(map_sqlx)?;

    let workspace_id = Uuid::new_v4();
    sqlx::query("INSERT INTO workspaces (id, name, slug) VALUES ($1, $2, $3)")
        .bind(workspace_id)
        .bind(workspace_name)
        .bind(workspace_slug)
        .execute(&mut *tx)
        .await
        .map_err(map_sqlx)?;

    sqlx::query(
        "INSERT INTO workspace_memberships (workspace_id, user_id, role)
         VALUES ($1, $2, 'owner')",
    )
    .bind(workspace_id)
    .bind(user_id)
    .execute(&mut *tx)
    .await
    .map_err(map_sqlx)?;

    record_audit_event_tx(&mut tx, None, "user.bootstrap", &serde_json::json!({})).await?;

    tx.commit().await.map_err(StoreError::Internal)?;

    Ok(UserProjection {
        id: user_id,
        username: username.to_owned(),
        display_name: display_name.to_owned(),
        locale: "en".to_owned(),
        timezone: "UTC".to_owned(),
        disabled: false,
    })
}

// ---------------------------------------------------------------------------
// Users
// ---------------------------------------------------------------------------

/// Look up a user by their canonical (lowercase) username.
///
/// Returns `None` if not found; callers must perform dummy verification before
/// responding to avoid timing side-channels.
pub async fn find_user_by_username(
    pool: &PgPool,
    username: &str,
) -> Result<Option<(Uuid, String)>, StoreError> {
    // Returns (user_id, password_hash). Never returns the raw hash to HTTP callers.
    let row = sqlx::query_as::<_, (Uuid, String)>(
        "SELECT id, password_hash FROM users
         WHERE username = $1 AND disabled_at IS NULL",
    )
    .bind(username)
    .fetch_optional(pool)
    .await
    .map_err(StoreError::Internal)?;
    Ok(row)
}

/// Look up a user projection by ID.
pub async fn find_user_by_id(pool: &PgPool, user_id: Uuid) -> Result<UserProjection, StoreError> {
    let row = sqlx::query_as::<_, (Uuid, String, String, String, String, Option<OffsetDateTime>)>(
        "SELECT id, username, display_name, locale, timezone, disabled_at
         FROM users WHERE id = $1",
    )
    .bind(user_id)
    .fetch_optional(pool)
    .await
    .map_err(StoreError::Internal)?
    .ok_or(StoreError::NotFound)?;
    Ok(UserProjection {
        id: row.0,
        username: row.1,
        display_name: row.2,
        locale: row.3,
        timezone: row.4,
        disabled: row.5.is_some(),
    })
}

/// Look up a user projection by canonical (lowercase) username.
pub async fn find_user_projection_by_username(
    pool: &PgPool,
    username: &str,
) -> Result<Option<UserProjection>, StoreError> {
    let row = sqlx::query_as::<_, (Uuid, String, String, String, String, Option<OffsetDateTime>)>(
        "SELECT id, username, display_name, locale, timezone, disabled_at
         FROM users WHERE username = $1",
    )
    .bind(username)
    .fetch_optional(pool)
    .await
    .map_err(StoreError::Internal)?;

    Ok(row.map(|r| UserProjection {
        id: r.0,
        username: r.1,
        display_name: r.2,
        locale: r.3,
        timezone: r.4,
        disabled: r.5.is_some(),
    }))
}

/// Provision an additional user account via operator CLI.
///
/// Does not grant membership in any workspace. Returns `StoreError::Conflict`
/// if username is already taken.
pub async fn create_user(
    pool: &PgPool,
    username: &str,
    password_hash: &str,
    display_name: &str,
) -> Result<UserProjection, StoreError> {
    let user_id = Uuid::new_v4();
    let mut tx = pool.begin().await.map_err(StoreError::Internal)?;

    sqlx::query(
        "INSERT INTO users (id, username, password_hash, display_name)
         VALUES ($1, $2, $3, $4)",
    )
    .bind(user_id)
    .bind(username)
    .bind(password_hash)
    .bind(display_name)
    .execute(&mut *tx)
    .await
    .map_err(map_sqlx)?;

    record_audit_event_tx(
        &mut tx,
        Some(user_id),
        "user.provision_cli",
        &serde_json::json!({ "username": username }),
    )
    .await?;

    tx.commit().await.map_err(StoreError::Internal)?;

    Ok(UserProjection {
        id: user_id,
        username: username.to_owned(),
        display_name: display_name.to_owned(),
        locale: "en".to_owned(),
        timezone: "UTC".to_owned(),
        disabled: false,
    })
}

/// Reset a user's password and atomically revoke all active sessions.
///
/// Used for operator emergency credential recovery.
pub async fn reset_user_password(
    pool: &PgPool,
    username: &str,
    new_password_hash: &str,
) -> Result<UserProjection, StoreError> {
    let mut tx = pool.begin().await.map_err(StoreError::Internal)?;

    let row = sqlx::query_as::<_, (Uuid, String, String, String, String, Option<OffsetDateTime>)>(
        "SELECT id, username, display_name, locale, timezone, disabled_at
         FROM users WHERE username = $1 FOR UPDATE",
    )
    .bind(username)
    .fetch_optional(&mut *tx)
    .await
    .map_err(StoreError::Internal)?
    .ok_or(StoreError::NotFound)?;

    let user_id = row.0;
    let now = OffsetDateTime::now_utc();

    sqlx::query("UPDATE users SET password_hash = $1, updated_at = $2 WHERE id = $3")
        .bind(new_password_hash)
        .bind(now)
        .bind(user_id)
        .execute(&mut *tx)
        .await
        .map_err(StoreError::Internal)?;

    // Atomically revoke all active sessions for this user.
    sqlx::query("UPDATE sessions SET revoked_at = $1 WHERE user_id = $2 AND revoked_at IS NULL")
        .bind(now)
        .bind(user_id)
        .execute(&mut *tx)
        .await
        .map_err(StoreError::Internal)?;

    record_audit_event_tx(
        &mut tx,
        Some(user_id),
        "user.password_reset_cli",
        &serde_json::json!({ "username": username }),
    )
    .await?;

    tx.commit().await.map_err(StoreError::Internal)?;

    Ok(UserProjection {
        id: user_id,
        username: row.1,
        display_name: row.2,
        locale: row.3,
        timezone: row.4,
        disabled: row.5.is_some(),
    })
}

/// Disable or enable a user account.
///
/// Disabling immediately revokes all active sessions for that user.
/// Refuses with StoreError::Conflict if this user is the last active owner of any workspace.
pub async fn set_user_disabled(
    pool: &PgPool,
    username: &str,
    disabled: bool,
) -> Result<UserProjection, StoreError> {
    let mut tx = pool.begin().await.map_err(StoreError::Internal)?;

    let row = sqlx::query_as::<_, (Uuid, String, String, String, String, Option<OffsetDateTime>)>(
        "SELECT id, username, display_name, locale, timezone, disabled_at
         FROM users WHERE username = $1 FOR UPDATE",
    )
    .bind(username)
    .fetch_optional(&mut *tx)
    .await
    .map_err(StoreError::Internal)?
    .ok_or(StoreError::NotFound)?;

    let user_id = row.0;
    let now = OffsetDateTime::now_utc();

    if disabled {
        // Enforce last-owner invariant across all workspaces:
        let is_sole_owner: bool = sqlx::query_scalar(
            "SELECT EXISTS (
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
             )",
        )
        .bind(user_id)
        .fetch_one(&mut *tx)
        .await
        .map_err(StoreError::Internal)?;

        if is_sole_owner {
            return Err(StoreError::Conflict);
        }

        sqlx::query("UPDATE users SET disabled_at = $1, updated_at = $1 WHERE id = $2")
            .bind(now)
            .bind(user_id)
            .execute(&mut *tx)
            .await
            .map_err(StoreError::Internal)?;

        sqlx::query(
            "UPDATE sessions SET revoked_at = $1 WHERE user_id = $2 AND revoked_at IS NULL",
        )
        .bind(now)
        .bind(user_id)
        .execute(&mut *tx)
        .await
        .map_err(StoreError::Internal)?;

        record_audit_event_tx(
            &mut tx,
            Some(user_id),
            "user.disabled_cli",
            &serde_json::json!({ "username": username }),
        )
        .await?;
    } else {
        sqlx::query("UPDATE users SET disabled_at = NULL, updated_at = $1 WHERE id = $2")
            .bind(now)
            .bind(user_id)
            .execute(&mut *tx)
            .await
            .map_err(StoreError::Internal)?;

        record_audit_event_tx(
            &mut tx,
            Some(user_id),
            "user.enabled_cli",
            &serde_json::json!({ "username": username }),
        )
        .await?;
    }

    tx.commit().await.map_err(StoreError::Internal)?;

    Ok(UserProjection {
        id: user_id,
        username: row.1,
        display_name: row.2,
        locale: row.3,
        timezone: row.4,
        disabled,
    })
}

// ---------------------------------------------------------------------------
// Sessions
// ---------------------------------------------------------------------------

/// Create a new session for a user and return the opaque token string.
///
/// The raw token is returned only here; the caller delivers it via cookie and
/// must not store or log it. The CSRF token is also generated here.
pub async fn create_session(pool: &PgPool, user_id: Uuid) -> Result<(String, String), StoreError> {
    let session_token = crate::identity::Token::generate()
        .map_err(|_| StoreError::Internal(sqlx::Error::RowNotFound))?;
    let csrf_token = crate::identity::Token::generate()
        .map_err(|_| StoreError::Internal(sqlx::Error::RowNotFound))?;

    let token_hash = session_token.hash();
    let csrf_hash = csrf_token.hash();

    let now = OffsetDateTime::now_utc();
    let expires_at = now + time::Duration::days(SESSION_MAX_DAYS);

    sqlx::query(
        "INSERT INTO sessions
            (token_hash, user_id, csrf_token_hash, created_at, last_active_at, expires_at)
         VALUES ($1, $2, $3, $4, $5, $6)",
    )
    .bind(token_hash.as_str())
    .bind(user_id)
    .bind(csrf_hash.as_str())
    .bind(now)
    .bind(now)
    .bind(expires_at)
    .execute(pool)
    .await
    .map_err(StoreError::Internal)?;

    Ok((
        session_token.as_str().to_owned(),
        csrf_token.as_str().to_owned(),
    ))
}

/// Resolve a session from a raw token string.
///
/// Returns the session data if the token is valid, not expired (absolute),
/// not idle-expired (last_active_at + SESSION_IDLE_HOURS), and not revoked.
/// Updates last_active_at on success.
pub async fn resolve_session(
    pool: &PgPool,
    raw_token: &str,
) -> Result<Option<ResolvedSession>, StoreError> {
    let hash = match crate::identity::hash_client_token(raw_token) {
        Ok(h) => h,
        Err(_) => return Ok(None),
    };
    let now = OffsetDateTime::now_utc();
    let idle_cutoff = now - time::Duration::hours(SESSION_IDLE_HOURS);

    // Fetch session and user in one query.
    let row = sqlx::query_as::<
        _,
        (
            Uuid,
            Uuid,
            String,
            String,
            String,
            String,
            Option<OffsetDateTime>,
        ),
    >(
        "SELECT s.id, u.id, u.username, u.display_name, u.locale, u.timezone, u.disabled_at
         FROM sessions s
         JOIN users u ON u.id = s.user_id
         WHERE s.token_hash = $1
           AND s.revoked_at IS NULL
           AND s.expires_at > $2
           AND s.last_active_at > $3",
    )
    .bind(hash.as_str())
    .bind(now)
    .bind(idle_cutoff)
    .fetch_optional(pool)
    .await
    .map_err(StoreError::Internal)?;

    let Some((session_id, user_id, username, display_name, locale, timezone, disabled_at)) = row
    else {
        return Ok(None);
    };

    // A disabled user's sessions are not valid.
    if disabled_at.is_some() {
        return Ok(None);
    }

    // Fetch CSRF token hash for this session.
    let csrf_token_hash: String =
        sqlx::query_scalar("SELECT csrf_token_hash FROM sessions WHERE id = $1")
            .bind(session_id)
            .fetch_one(pool)
            .await
            .map_err(StoreError::Internal)?;

    // Update last_active_at (best-effort; do not fail the request if this fails).
    let _ = sqlx::query("UPDATE sessions SET last_active_at = $1 WHERE id = $2")
        .bind(now)
        .bind(session_id)
        .execute(pool)
        .await;

    Ok(Some(ResolvedSession {
        session_id,
        csrf_token_hash,
        user: UserProjection {
            id: user_id,
            username,
            display_name,
            locale,
            timezone,
            disabled: false,
        },
    }))
}

/// Revoke a specific session. Returns Ok(()) whether or not the session existed.
pub async fn revoke_session(pool: &PgPool, session_id: Uuid) -> Result<(), StoreError> {
    let now = OffsetDateTime::now_utc();
    sqlx::query("UPDATE sessions SET revoked_at = $1 WHERE id = $2 AND revoked_at IS NULL")
        .bind(now)
        .bind(session_id)
        .execute(pool)
        .await
        .map_err(StoreError::Internal)?;
    Ok(())
}

// ---------------------------------------------------------------------------
// Workspaces
// ---------------------------------------------------------------------------

/// Return the workspaces the given user is a member of.
pub async fn list_user_workspaces(
    pool: &PgPool,
    user_id: Uuid,
) -> Result<Vec<WorkspaceProjection>, StoreError> {
    let rows = sqlx::query_as::<_, (Uuid, String, String, String)>(
        "SELECT w.id, w.name, w.slug, wm.role
         FROM workspaces w
         JOIN workspace_memberships wm ON wm.workspace_id = w.id
         WHERE wm.user_id = $1
         ORDER BY w.name",
    )
    .bind(user_id)
    .fetch_all(pool)
    .await
    .map_err(StoreError::Internal)?;

    Ok(rows
        .into_iter()
        .map(|(id, name, slug, role)| WorkspaceProjection {
            id,
            name,
            slug,
            role,
        })
        .collect())
}

/// Check if the user is a member of the workspace, returning their role.
pub async fn get_user_membership(
    pool: &PgPool,
    workspace_id: Uuid,
    user_id: Uuid,
) -> Result<Option<String>, StoreError> {
    let row = sqlx::query_scalar::<_, String>(
        "SELECT wm.role
         FROM workspace_memberships wm
         JOIN users u ON u.id = wm.user_id
         WHERE wm.workspace_id = $1 AND wm.user_id = $2 AND u.disabled_at IS NULL",
    )
    .bind(workspace_id)
    .bind(user_id)
    .fetch_optional(pool)
    .await
    .map_err(StoreError::Internal)?;

    Ok(row)
}

/// List all active members of a workspace.
pub async fn list_workspace_members(
    pool: &PgPool,
    workspace_id: Uuid,
) -> Result<Vec<MemberProjection>, StoreError> {
    let rows = sqlx::query_as::<_, (Uuid, String, String, String)>(
        "SELECT u.id, u.username, u.display_name, wm.role
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
            u.username ASC",
    )
    .bind(workspace_id)
    .fetch_all(pool)
    .await
    .map_err(StoreError::Internal)?;

    Ok(rows
        .into_iter()
        .map(|(user_id, username, display_name, role)| MemberProjection {
            user_id,
            username,
            display_name,
            role,
        })
        .collect())
}

/// Add an active user to a workspace with a specified role.
///
/// Fails with `StoreError::Conflict` if the user is already a member.
/// Fails with `StoreError::Forbidden` if attempting to add as 'owner'.
pub async fn add_workspace_member(
    pool: &PgPool,
    actor_id: Option<Uuid>,
    workspace_id: Uuid,
    target_user_id: Uuid,
    role: &str,
) -> Result<MemberProjection, StoreError> {
    if role != "admin" && role != "member" && role != "guest" {
        return Err(StoreError::Forbidden);
    }

    let mut tx = pool.begin().await.map_err(StoreError::Internal)?;

    // Verify target user exists and is active.
    let target_user = sqlx::query_as::<_, (Uuid, String, String)>(
        "SELECT id, username, display_name FROM users WHERE id = $1 AND disabled_at IS NULL",
    )
    .bind(target_user_id)
    .fetch_optional(&mut *tx)
    .await
    .map_err(StoreError::Internal)?
    .ok_or(StoreError::NotFound)?;

    sqlx::query(
        "INSERT INTO workspace_memberships (workspace_id, user_id, role)
         VALUES ($1, $2, $3)",
    )
    .bind(workspace_id)
    .bind(target_user_id)
    .bind(role)
    .execute(&mut *tx)
    .await
    .map_err(map_sqlx)?;

    record_audit_event_tx(
        &mut tx,
        actor_id,
        "workspace.member_added",
        &serde_json::json!({
            "workspace_id": workspace_id,
            "target_user_id": target_user_id,
            "role": role,
        }),
    )
    .await?;

    tx.commit().await.map_err(StoreError::Internal)?;

    Ok(MemberProjection {
        user_id: target_user.0,
        username: target_user.1,
        display_name: target_user.2,
        role: role.to_owned(),
    })
}

/// Update a member's role in a workspace.
///
/// Serialized per workspace row to prevent concurrent last-owner demotion races.
/// Fails with `StoreError::Conflict` if attempting to demote the last active owner.
pub async fn update_workspace_member_role(
    pool: &PgPool,
    actor_id: Option<Uuid>,
    workspace_id: Uuid,
    target_user_id: Uuid,
    new_role: &str,
) -> Result<MemberProjection, StoreError> {
    if new_role != "admin" && new_role != "member" && new_role != "guest" {
        return Err(StoreError::Forbidden);
    }

    let mut tx = pool.begin().await.map_err(StoreError::Internal)?;

    // Lock workspace row to prevent concurrent membership race conditions.
    let _ = sqlx::query("SELECT id FROM workspaces WHERE id = $1 FOR UPDATE")
        .bind(workspace_id)
        .fetch_optional(&mut *tx)
        .await
        .map_err(StoreError::Internal)?
        .ok_or(StoreError::NotFound)?;

    let current_role: String = sqlx::query_scalar(
        "SELECT role FROM workspace_memberships
         WHERE workspace_id = $1 AND user_id = $2 FOR UPDATE",
    )
    .bind(workspace_id)
    .bind(target_user_id)
    .fetch_optional(&mut *tx)
    .await
    .map_err(StoreError::Internal)?
    .ok_or(StoreError::NotFound)?;

    if current_role == "owner" && new_role != "owner" {
        let active_owners: i64 = sqlx::query_scalar(
            "SELECT COUNT(*)
             FROM workspace_memberships wm
             JOIN users u ON u.id = wm.user_id
             WHERE wm.workspace_id = $1 AND wm.role = 'owner'
               AND wm.user_id != $2 AND u.disabled_at IS NULL",
        )
        .bind(workspace_id)
        .bind(target_user_id)
        .fetch_one(&mut *tx)
        .await
        .map_err(StoreError::Internal)?;

        if active_owners == 0 {
            return Err(StoreError::Conflict);
        }
    }

    let now = OffsetDateTime::now_utc();
    sqlx::query(
        "UPDATE workspace_memberships SET role = $1, updated_at = $2
         WHERE workspace_id = $3 AND user_id = $4",
    )
    .bind(new_role)
    .bind(now)
    .bind(workspace_id)
    .bind(target_user_id)
    .execute(&mut *tx)
    .await
    .map_err(StoreError::Internal)?;

    let user_info = sqlx::query_as::<_, (String, String)>(
        "SELECT username, display_name FROM users WHERE id = $1",
    )
    .bind(target_user_id)
    .fetch_one(&mut *tx)
    .await
    .map_err(StoreError::Internal)?;

    record_audit_event_tx(
        &mut tx,
        actor_id,
        "workspace.member_role_updated",
        &serde_json::json!({
            "workspace_id": workspace_id,
            "target_user_id": target_user_id,
            "old_role": current_role,
            "new_role": new_role,
        }),
    )
    .await?;

    tx.commit().await.map_err(StoreError::Internal)?;

    Ok(MemberProjection {
        user_id: target_user_id,
        username: user_info.0,
        display_name: user_info.1,
        role: new_role.to_owned(),
    })
}

/// Remove a member from a workspace.
///
/// Serialized per workspace row to prevent concurrent last-owner removal races.
/// Fails with `StoreError::Conflict` if attempting to remove the last active owner.
pub async fn remove_workspace_member(
    pool: &PgPool,
    actor_id: Option<Uuid>,
    workspace_id: Uuid,
    target_user_id: Uuid,
) -> Result<(), StoreError> {
    let mut tx = pool.begin().await.map_err(StoreError::Internal)?;

    // Lock workspace row to prevent concurrent race conditions.
    let _ = sqlx::query("SELECT id FROM workspaces WHERE id = $1 FOR UPDATE")
        .bind(workspace_id)
        .fetch_optional(&mut *tx)
        .await
        .map_err(StoreError::Internal)?
        .ok_or(StoreError::NotFound)?;

    let current_role: String = sqlx::query_scalar(
        "SELECT role FROM workspace_memberships
         WHERE workspace_id = $1 AND user_id = $2 FOR UPDATE",
    )
    .bind(workspace_id)
    .bind(target_user_id)
    .fetch_optional(&mut *tx)
    .await
    .map_err(StoreError::Internal)?
    .ok_or(StoreError::NotFound)?;

    if current_role == "owner" {
        let active_owners: i64 = sqlx::query_scalar(
            "SELECT COUNT(*)
             FROM workspace_memberships wm
             JOIN users u ON u.id = wm.user_id
             WHERE wm.workspace_id = $1 AND wm.role = 'owner'
               AND wm.user_id != $2 AND u.disabled_at IS NULL",
        )
        .bind(workspace_id)
        .bind(target_user_id)
        .fetch_one(&mut *tx)
        .await
        .map_err(StoreError::Internal)?;

        if active_owners == 0 {
            return Err(StoreError::Conflict);
        }
    }

    sqlx::query("DELETE FROM workspace_memberships WHERE workspace_id = $1 AND user_id = $2")
        .bind(workspace_id)
        .bind(target_user_id)
        .execute(&mut *tx)
        .await
        .map_err(StoreError::Internal)?;

    record_audit_event_tx(
        &mut tx,
        actor_id,
        "workspace.member_removed",
        &serde_json::json!({
            "workspace_id": workspace_id,
            "target_user_id": target_user_id,
        }),
    )
    .await?;

    tx.commit().await.map_err(StoreError::Internal)?;
    Ok(())
}

/// Atomically transfer ownership of a workspace from the current owner to a target member.
///
/// Current owner is demoted to 'admin', and the target member is promoted to 'owner'.
pub async fn transfer_workspace_ownership(
    pool: &PgPool,
    actor_id: Option<Uuid>,
    workspace_id: Uuid,
    current_owner_id: Uuid,
    target_user_id: Uuid,
) -> Result<(), StoreError> {
    if current_owner_id == target_user_id {
        return Ok(());
    }

    let mut tx = pool.begin().await.map_err(StoreError::Internal)?;

    // Lock workspace row.
    let _ = sqlx::query("SELECT id FROM workspaces WHERE id = $1 FOR UPDATE")
        .bind(workspace_id)
        .fetch_optional(&mut *tx)
        .await
        .map_err(StoreError::Internal)?
        .ok_or(StoreError::NotFound)?;

    // Verify current owner is indeed owner.
    let owner_role: String = sqlx::query_scalar(
        "SELECT role FROM workspace_memberships
         WHERE workspace_id = $1 AND user_id = $2 FOR UPDATE",
    )
    .bind(workspace_id)
    .bind(current_owner_id)
    .fetch_optional(&mut *tx)
    .await
    .map_err(StoreError::Internal)?
    .ok_or(StoreError::Forbidden)?;

    if owner_role != "owner" {
        return Err(StoreError::Forbidden);
    }

    // Verify target user is an active member and not disabled.
    let target_active: bool = sqlx::query_scalar(
        "SELECT EXISTS (
             SELECT 1 FROM workspace_memberships wm
             JOIN users u ON u.id = wm.user_id
             WHERE wm.workspace_id = $1 AND wm.user_id = $2
               AND u.disabled_at IS NULL
         )",
    )
    .bind(workspace_id)
    .bind(target_user_id)
    .fetch_one(&mut *tx)
    .await
    .map_err(StoreError::Internal)?;

    if !target_active {
        return Err(StoreError::NotFound);
    }

    let now = OffsetDateTime::now_utc();

    // Demote current owner to admin.
    sqlx::query(
        "UPDATE workspace_memberships SET role = 'admin', updated_at = $1
         WHERE workspace_id = $2 AND user_id = $3",
    )
    .bind(now)
    .bind(workspace_id)
    .bind(current_owner_id)
    .execute(&mut *tx)
    .await
    .map_err(StoreError::Internal)?;

    // Promote target member to owner.
    sqlx::query(
        "UPDATE workspace_memberships SET role = 'owner', updated_at = $1
         WHERE workspace_id = $2 AND user_id = $3",
    )
    .bind(now)
    .bind(workspace_id)
    .bind(target_user_id)
    .execute(&mut *tx)
    .await
    .map_err(StoreError::Internal)?;

    record_audit_event_tx(
        &mut tx,
        actor_id,
        "workspace.ownership_transferred",
        &serde_json::json!({
            "workspace_id": workspace_id,
            "previous_owner_id": current_owner_id,
            "new_owner_id": target_user_id,
        }),
    )
    .await?;

    tx.commit().await.map_err(StoreError::Internal)?;
    Ok(())
}

// ---------------------------------------------------------------------------
// Audit
// ---------------------------------------------------------------------------

/// Record a security audit event.
///
/// `actor_id` is None for pre-authentication events (e.g., failed logins
/// where the user does not exist).
///
/// The metadata must not contain passwords, raw tokens, or hashes.
pub async fn record_audit_event(
    pool: &PgPool,
    actor_id: Option<Uuid>,
    event_type: &str,
    metadata: &serde_json::Value,
) -> Result<(), StoreError> {
    sqlx::query(
        "INSERT INTO security_audit_events (actor_id, event_type, metadata)
         VALUES ($1, $2, $3)",
    )
    .bind(actor_id)
    .bind(event_type)
    .bind(metadata)
    .execute(pool)
    .await
    .map_err(StoreError::Internal)?;
    Ok(())
}

/// Record a security audit event within an existing transaction.
async fn record_audit_event_tx(
    tx: &mut sqlx::Transaction<'_, sqlx::Postgres>,
    actor_id: Option<Uuid>,
    event_type: &str,
    metadata: &serde_json::Value,
) -> Result<(), StoreError> {
    sqlx::query(
        "INSERT INTO security_audit_events (actor_id, event_type, metadata)
         VALUES ($1, $2, $3)",
    )
    .bind(actor_id)
    .bind(event_type)
    .bind(metadata)
    .execute(&mut **tx)
    .await
    .map_err(StoreError::Internal)?;
    Ok(())
}
