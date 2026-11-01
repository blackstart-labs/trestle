-- Identity, workspaces and security audit tables for M03a.1.
-- Users, sessions, workspaces, memberships and audit events.
-- Login-attempt counters are kept in application memory; only persisted state is here.

-- Users -----------------------------------------------------------------------

CREATE TABLE users (
    id             UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    -- ASCII-lowercase canonical form enforced by application before insert.
    -- Stored lowercase; uniqueness checked on this column.
    username       TEXT        NOT NULL,
    password_hash  TEXT        NOT NULL,
    display_name   TEXT        NOT NULL,
    locale         TEXT        NOT NULL DEFAULT 'en',
    timezone       TEXT        NOT NULL DEFAULT 'UTC',
    disabled_at    TIMESTAMPTZ,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT users_username_nonempty       CHECK (length(username) >= 3),
    CONSTRAINT users_username_maxlen         CHECK (length(username) <= 64),
    CONSTRAINT users_username_ascii          CHECK (username ~ '^[a-z0-9._-]+$'),
    CONSTRAINT users_display_name_nonempty   CHECK (length(display_name) >= 1),
    CONSTRAINT users_display_name_maxlen     CHECK (length(display_name) <= 120),
    CONSTRAINT users_locale_nonempty         CHECK (length(locale) >= 2),
    CONSTRAINT users_locale_maxlen           CHECK (length(locale) <= 35),
    CONSTRAINT users_timezone_nonempty       CHECK (length(timezone) >= 1),
    CONSTRAINT users_timezone_maxlen         CHECK (length(timezone) <= 64)
);

CREATE UNIQUE INDEX users_username_unique ON users (username);

-- Sessions --------------------------------------------------------------------
-- token_hash: SHA-256 hex digest of the raw opaque session token.
-- csrf_token_hash: SHA-256 hex digest of the raw CSRF token (stored to verify).
-- Raw tokens are never stored. Tokens are 32 cryptographically random bytes,
-- base64url encoded for transmission.

CREATE TABLE sessions (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    token_hash      TEXT        NOT NULL,
    user_id         UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    csrf_token_hash TEXT        NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_active_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    -- Absolute expiry: creation + 7 days. Cannot be extended by activity.
    expires_at      TIMESTAMPTZ NOT NULL,
    -- Idle expiry: last_active_at + 12 hours, checked at session resolution time.
    -- Revocation: set to now() to invalidate without waiting for natural expiry.
    revoked_at      TIMESTAMPTZ,

    CONSTRAINT sessions_token_hash_len    CHECK (length(token_hash) = 64),
    CONSTRAINT sessions_csrf_hash_len     CHECK (length(csrf_token_hash) = 64),
    CONSTRAINT sessions_expires_after     CHECK (expires_at > created_at)
);

CREATE UNIQUE INDEX sessions_token_hash_unique ON sessions (token_hash);
-- Used to clean up expired/revoked sessions and to bound active session counts.
CREATE INDEX sessions_user_id ON sessions (user_id);
CREATE INDEX sessions_expires_at ON sessions (expires_at);

-- Workspaces ------------------------------------------------------------------

CREATE TABLE workspaces (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT        NOT NULL,
    -- URL-safe identifier, lowercase alphanumeric and hyphens, 2-48 characters.
    slug        TEXT        NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT workspaces_name_nonempty    CHECK (length(name) >= 1),
    CONSTRAINT workspaces_name_maxlen      CHECK (length(name) <= 120),
    CONSTRAINT workspaces_slug_format      CHECK (slug ~ '^[a-z0-9][a-z0-9-]{0,46}[a-z0-9]$' OR slug ~ '^[a-z0-9]{2}$'),
    CONSTRAINT workspaces_slug_len         CHECK (length(slug) >= 2 AND length(slug) <= 48)
);

CREATE UNIQUE INDEX workspaces_slug_unique ON workspaces (slug);

-- Workspace memberships -------------------------------------------------------
-- role: owner, admin, member, guest.
-- Invariant (enforced at application layer): every workspace has at least one
-- active owner (user whose disabled_at IS NULL and membership role = 'owner').
-- Database cannot enforce this with a simple CHECK; concurrency tests are required.

CREATE TABLE workspace_memberships (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID        NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    user_id      UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role         TEXT        NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT workspace_memberships_role_values
        CHECK (role IN ('owner', 'admin', 'member', 'guest')),
    CONSTRAINT workspace_memberships_unique
        UNIQUE (workspace_id, user_id)
);

CREATE INDEX workspace_memberships_user ON workspace_memberships (user_id);
CREATE INDEX workspace_memberships_workspace ON workspace_memberships (workspace_id);

-- Security audit events -------------------------------------------------------
-- Records security-relevant actions. actor_id may be NULL for pre-auth events.
-- metadata is structured but never contains passwords, raw tokens or hashes.

CREATE TABLE security_audit_events (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    actor_id    UUID        REFERENCES users (id) ON DELETE SET NULL,
    event_type  TEXT        NOT NULL,
    metadata    JSONB       NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT audit_event_type_nonempty CHECK (length(event_type) >= 1),
    CONSTRAINT audit_event_type_maxlen   CHECK (length(event_type) <= 64)
);

CREATE INDEX security_audit_events_actor ON security_audit_events (actor_id)
    WHERE actor_id IS NOT NULL;
CREATE INDEX security_audit_events_created ON security_audit_events (created_at);
CREATE INDEX security_audit_events_type ON security_audit_events (event_type);
