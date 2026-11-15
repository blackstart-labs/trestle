-- Task dependencies: directed "blocked by" edges between tasks.
-- A row means task_id cannot finish until depends_on_task_id is done. Both
-- tasks must share a workspace, but may belong to different projects.

CREATE TABLE task_dependencies (
    id                UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id      UUID        NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    project_id        UUID        NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    task_id           UUID        NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    depends_on_task_id UUID       NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    created_by        UUID        REFERENCES users (id) ON DELETE SET NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT task_dependencies_not_self CHECK (task_id <> depends_on_task_id),
    CONSTRAINT task_dependencies_unique   UNIQUE (task_id, depends_on_task_id)
);

CREATE INDEX task_dependencies_task       ON task_dependencies (task_id);
CREATE INDEX task_dependencies_depends_on ON task_dependencies (depends_on_task_id);
CREATE INDEX task_dependencies_workspace  ON task_dependencies (workspace_id);
