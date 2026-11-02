-- Forward migration: task collaboration primitives (labels, assignees, subtasks, comments, activity)

-- 1. Add parent_id for hierarchical subtasks within the same project
ALTER TABLE tasks
    ADD COLUMN parent_id UUID REFERENCES tasks (id) ON DELETE CASCADE,
    ADD CONSTRAINT tasks_parent_not_self CHECK (parent_id IS NULL OR parent_id != id);

CREATE INDEX idx_tasks_parent_id ON tasks (parent_id) WHERE parent_id IS NOT NULL;

-- 2. Labels table scoped to workspace and project
CREATE TABLE labels (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK (char_length(trim(name)) >= 1 AND char_length(name) <= 50),
    color TEXT NOT NULL DEFAULT '#315b3b' CHECK (color ~* '^#[0-9a-f]{6}$'),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_labels_project_name UNIQUE (project_id, name)
);

CREATE INDEX idx_labels_workspace_project ON labels (workspace_id, project_id);

-- 3. Task label attachments
CREATE TABLE task_label_assignments (
    task_id UUID NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    label_id UUID NOT NULL REFERENCES labels (id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (task_id, label_id)
);

CREATE INDEX idx_task_label_assignments_label_id ON task_label_assignments (label_id);

-- 4. Task assignees
CREATE TABLE task_assignees (
    task_id UUID NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    assigned_by UUID REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (task_id, user_id)
);

CREATE INDEX idx_task_assignees_user_id ON task_assignees (user_id);

-- 5. Threaded task comments
CREATE TABLE task_comments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    content TEXT NOT NULL CHECK (char_length(trim(content)) >= 1 AND char_length(content) <= 5000),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_task_comments_task_id ON task_comments (task_id, created_at ASC);

-- 6. Task activity history
CREATE TABLE task_activities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    task_id UUID NOT NULL REFERENCES tasks (id) ON DELETE CASCADE,
    actor_id UUID REFERENCES users (id) ON DELETE SET NULL,
    action TEXT NOT NULL,
    details JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_task_activities_task_id ON task_activities (task_id, created_at DESC);
