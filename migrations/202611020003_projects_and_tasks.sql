-- Projects and canonical tasks schema for M04a.
-- Establishes workspace-scoped projects, canonical tasks, and human-readable task keys.

CREATE TABLE projects (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id  UUID        NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    name          TEXT        NOT NULL,
    slug          TEXT        NOT NULL,
    key           TEXT        NOT NULL,
    description   TEXT        NOT NULL DEFAULT '',
    task_counter  BIGINT      NOT NULL DEFAULT 0,
    archived_at   TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT projects_name_nonempty CHECK (length(name) >= 1),
    CONSTRAINT projects_name_maxlen   CHECK (length(name) <= 120),
    CONSTRAINT projects_slug_format   CHECK (slug ~ '^[a-z0-9][a-z0-9-]{0,46}[a-z0-9]$' OR slug ~ '^[a-z0-9]{2}$'),
    CONSTRAINT projects_slug_len      CHECK (length(slug) >= 2 AND length(slug) <= 48),
    CONSTRAINT projects_key_format    CHECK (key ~ '^[A-Z0-9]{2,8}$'),
    CONSTRAINT projects_unique_slug   UNIQUE (workspace_id, slug),
    CONSTRAINT projects_unique_key    UNIQUE (workspace_id, key)
);

CREATE INDEX projects_workspace_id ON projects (workspace_id);

CREATE TABLE tasks (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID        NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    project_id   UUID        NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    task_number  BIGINT      NOT NULL,
    title        TEXT        NOT NULL,
    description  TEXT        NOT NULL DEFAULT '',
    status       TEXT        NOT NULL DEFAULT 'unstarted',
    priority     TEXT        NOT NULL DEFAULT 'none',
    due_date     DATE,
    created_by   UUID        REFERENCES users (id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT tasks_title_nonempty CHECK (length(title) >= 1),
    CONSTRAINT tasks_title_maxlen   CHECK (length(title) <= 255),
    CONSTRAINT tasks_status_values  CHECK (status IN ('backlog', 'unstarted', 'started', 'completed', 'cancelled')),
    CONSTRAINT tasks_priority_values CHECK (priority IN ('none', 'low', 'medium', 'high', 'urgent')),
    CONSTRAINT tasks_unique_number  UNIQUE (project_id, task_number)
);

CREATE INDEX tasks_project_status ON tasks (project_id, status, created_at DESC);
CREATE INDEX tasks_workspace_id ON tasks (workspace_id);
