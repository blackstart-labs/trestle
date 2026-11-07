ALTER TABLE projects ADD CONSTRAINT projects_workspace_identity UNIQUE (workspace_id, id);

CREATE TABLE project_integrations (
    workspace_id UUID NOT NULL,
    project_id UUID NOT NULL,
    provider TEXT NOT NULL CHECK (provider IN ('telegram', 'slack', 'discord')),
    secret TEXT NOT NULL CHECK (length(secret) BETWEEN 1 AND 512),
    chat_id TEXT NOT NULL DEFAULT '',
    last_attempt_at TIMESTAMPTZ,
    last_error TEXT NOT NULL DEFAULT '',
    PRIMARY KEY (workspace_id, project_id, provider),
    FOREIGN KEY (workspace_id, project_id) REFERENCES projects (workspace_id, id) ON DELETE CASCADE
);
