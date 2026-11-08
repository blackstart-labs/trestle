-- Forward migration: project milestones (planning targets for grouping tasks)

CREATE TABLE milestones (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    title TEXT NOT NULL CHECK (char_length(trim(title)) >= 1 AND char_length(title) <= 255),
    description TEXT NOT NULL DEFAULT '' CHECK (char_length(description) <= 10000),
    due_date DATE,
    status TEXT NOT NULL DEFAULT 'open' CHECK (status IN ('open', 'closed')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_milestones_project ON milestones (workspace_id, project_id, status);

ALTER TABLE tasks
    ADD COLUMN milestone_id UUID REFERENCES milestones (id) ON DELETE SET NULL;

CREATE INDEX idx_tasks_milestone_id ON tasks (milestone_id);
