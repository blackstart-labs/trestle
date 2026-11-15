-- Server-authoritative running timers.
-- One active timer per user. Stopped timers keep the idempotent link to the
-- time log they produced, so a lost stop response can be retried safely.

CREATE TABLE task_timers (
    id           UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    workspace_id UUID        NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    project_id   UUID        NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    task_id      UUID        NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    note         TEXT        NOT NULL DEFAULT '',
    state        TEXT        NOT NULL DEFAULT 'active',
    started_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    ended_at     TIMESTAMPTZ,
    time_log_id  UUID        REFERENCES task_time_logs (id) ON DELETE SET NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT task_timers_state  CHECK (state IN ('active', 'stopped', 'discarded')),
    CONSTRAINT task_timers_note   CHECK (length(note) <= 2000),
    CONSTRAINT task_timers_ended  CHECK ((state = 'active') = (ended_at IS NULL)),
    CONSTRAINT task_timers_log    CHECK (time_log_id IS NULL OR state = 'stopped')
);

-- The database, not a browser, enforces the single running timer.
CREATE UNIQUE INDEX task_timers_one_active_per_user
    ON task_timers (user_id) WHERE state = 'active';

CREATE INDEX task_timers_user_started ON task_timers (user_id, started_at DESC);
CREATE INDEX task_timers_task_id      ON task_timers (task_id);
