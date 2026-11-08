-- Forward migration: Task time tracking logs
CREATE TABLE IF NOT EXISTS task_time_logs (
    id           UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID         NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    project_id   UUID         NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    task_id      UUID         NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    user_id      UUID         REFERENCES users (id) ON DELETE SET NULL,
    hours        NUMERIC(7,2) NOT NULL CHECK (hours > 0),
    logged_date  DATE         NOT NULL DEFAULT CURRENT_DATE,
    description  TEXT         NOT NULL DEFAULT '',
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS task_time_logs_task_id ON task_time_logs (task_id);
CREATE INDEX IF NOT EXISTS task_time_logs_project_id ON task_time_logs (project_id);
CREATE INDEX IF NOT EXISTS task_time_logs_workspace_id ON task_time_logs (workspace_id);

