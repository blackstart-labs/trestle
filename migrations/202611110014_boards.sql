-- Workspace-owned visual work; no public sharing or arbitrary embedded content.
CREATE TABLE boards (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    creation_key UUID NOT NULL UNIQUE,
    workspace_id UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    title TEXT NOT NULL CHECK (char_length(trim(title)) BETWEEN 1 AND 120),
    kind TEXT NOT NULL CHECK (kind IN ('diagram', 'whiteboard')),
    scene JSONB NOT NULL CHECK (jsonb_typeof(scene) = 'object'),
    version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
    archived BOOLEAN NOT NULL DEFAULT false,
    created_by UUID REFERENCES users (id) ON DELETE SET NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX boards_workspace_list ON boards (workspace_id, archived, updated_at DESC, id);
