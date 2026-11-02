-- Forward migration: custom task statuses with stable category semantics (M05a)

-- 1. Create task_statuses table
CREATE TABLE task_statuses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK (char_length(trim(name)) >= 1 AND char_length(name) <= 50),
    category TEXT NOT NULL CHECK (category IN ('backlog', 'unstarted', 'started', 'completed', 'cancelled')),
    color TEXT NOT NULL DEFAULT '#315b3b' CHECK (color ~* '^#[0-9a-f]{6}$'),
    position INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_task_statuses_project_name UNIQUE (project_id, name)
);

CREATE INDEX idx_task_statuses_project ON task_statuses (workspace_id, project_id, position);

-- 2. Backfill default statuses for all existing projects
INSERT INTO task_statuses (id, workspace_id, project_id, name, category, color, position)
SELECT gen_random_uuid(), p.workspace_id, p.id, s.name, s.category, s.color, s.position
FROM projects p
CROSS JOIN (
    VALUES
        ('Backlog', 'backlog', '#4c544c', 0),
        ('To Do', 'unstarted', '#1e4d79', 1),
        ('In Progress', 'started', '#775300', 2),
        ('Done', 'completed', '#26633b', 3),
        ('Cancelled', 'cancelled', '#9f302b', 4)
) AS s(name, category, color, position)
ON CONFLICT (project_id, name) DO NOTHING;

-- 3. Add status_id to tasks referencing task_statuses
ALTER TABLE tasks
    ADD COLUMN status_id UUID REFERENCES task_statuses (id) ON DELETE RESTRICT;

-- 4. Backfill status_id for existing tasks matching project and canonical status
UPDATE tasks t
SET status_id = ts.id
FROM task_statuses ts
WHERE ts.project_id = t.project_id
  AND ts.category = t.status
  AND t.status_id IS NULL;

-- 5. Index status_id for task queries and filtering
CREATE INDEX idx_tasks_status_id ON tasks (status_id);
