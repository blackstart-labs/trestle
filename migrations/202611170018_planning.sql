-- Shared scheduling metadata; canonical tasks remain in tasks.
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'projects_workspace_identity'
    ) THEN
        ALTER TABLE projects ADD CONSTRAINT projects_workspace_identity UNIQUE(workspace_id,id);
    END IF;
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'tasks_workspace_identity'
    ) THEN
        ALTER TABLE tasks ADD CONSTRAINT tasks_workspace_identity UNIQUE(workspace_id,id);
    END IF;
END $$;
CREATE TABLE IF NOT EXISTS planning_items (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 creation_key UUID NOT NULL UNIQUE,
 kind TEXT NOT NULL CHECK(kind IN ('goal','objective','key_result','cycle','module')),
 title TEXT NOT NULL CHECK(char_length(trim(title)) BETWEEN 1 AND 160),
 description TEXT NOT NULL DEFAULT '' CHECK(char_length(description)<=15000),
 status TEXT NOT NULL DEFAULT 'planned' CHECK(status IN ('planned','active','at_risk','blocked','completed','cancelled')),
 parent_id UUID,
 project_id UUID,
 owner_id UUID REFERENCES users(id) ON DELETE SET NULL,
 start_date DATE,
 due_date DATE,
 target NUMERIC NOT NULL DEFAULT 100 CHECK(target>0 AND target<=1000000000),
 actual NUMERIC NOT NULL DEFAULT 0 CHECK(actual>=0 AND actual<=1000000000),
 archived BOOLEAN NOT NULL DEFAULT false,
 version BIGINT NOT NULL DEFAULT 1,
 created_by UUID REFERENCES users(id) ON DELETE SET NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE(workspace_id,id),
 UNIQUE(workspace_id,id,kind),
 CHECK(start_date IS NULL OR due_date IS NULL OR start_date<=due_date),
 CHECK(parent_id IS DISTINCT FROM id),
 FOREIGN KEY(workspace_id,parent_id) REFERENCES planning_items(workspace_id,id),
 FOREIGN KEY(workspace_id,project_id) REFERENCES projects(workspace_id,id)
);
CREATE INDEX IF NOT EXISTS planning_workspace_list ON planning_items(workspace_id,kind,archived,updated_at DESC,id);
CREATE INDEX IF NOT EXISTS planning_parent ON planning_items(workspace_id,parent_id);
CREATE TABLE IF NOT EXISTS planning_tasks (
 workspace_id UUID NOT NULL,
 item_id UUID NOT NULL,
 task_id UUID NOT NULL,
 kind TEXT NOT NULL,
 PRIMARY KEY(item_id,task_id),
 FOREIGN KEY(workspace_id,item_id,kind) REFERENCES planning_items(workspace_id,id,kind) ON DELETE CASCADE,
 FOREIGN KEY(workspace_id,task_id) REFERENCES tasks(workspace_id,id) ON DELETE CASCADE
);
CREATE UNIQUE INDEX IF NOT EXISTS task_one_cycle ON planning_tasks(task_id) WHERE kind='cycle';
CREATE UNIQUE INDEX IF NOT EXISTS task_one_module ON planning_tasks(task_id) WHERE kind='module';
