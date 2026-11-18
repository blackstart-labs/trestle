CREATE TABLE documents (
 id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
 workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
 creation_key UUID NOT NULL UNIQUE,
 created_by UUID REFERENCES users(id) ON DELETE SET NULL,
 title TEXT NOT NULL CHECK (char_length(trim(title)) BETWEEN 1 AND 160),
 content TEXT NOT NULL DEFAULT '' CHECK (char_length(content) <= 100000),
 tags TEXT[] NOT NULL DEFAULT '{}',
 parent_id UUID,
 version BIGINT NOT NULL DEFAULT 1 CHECK (version > 0),
 archived BOOLEAN NOT NULL DEFAULT false,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 UNIQUE(workspace_id,id),
 CHECK(parent_id IS DISTINCT FROM id),
 FOREIGN KEY(workspace_id,parent_id) REFERENCES documents(workspace_id,id)
);
CREATE INDEX documents_listing ON documents(workspace_id,archived,updated_at DESC,id);
CREATE INDEX documents_parent ON documents(workspace_id,parent_id);
CREATE TABLE document_versions (
 workspace_id UUID NOT NULL,
 document_id UUID NOT NULL,
 version BIGINT NOT NULL,
 title TEXT NOT NULL,
 content TEXT NOT NULL,
 tags TEXT[] NOT NULL,
 parent_id UUID,
 archived BOOLEAN NOT NULL,
 actor_id UUID REFERENCES users(id) ON DELETE SET NULL,
 created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
 PRIMARY KEY(document_id,version),
 FOREIGN KEY(workspace_id,document_id) REFERENCES documents(workspace_id,id) ON DELETE CASCADE
);
CREATE TABLE document_links (
 workspace_id UUID NOT NULL,
 source_id UUID NOT NULL,
 target_id UUID NOT NULL,
 PRIMARY KEY(source_id,target_id),
 FOREIGN KEY(workspace_id,source_id) REFERENCES documents(workspace_id,id) ON DELETE CASCADE,
 FOREIGN KEY(workspace_id,target_id) REFERENCES documents(workspace_id,id) ON DELETE CASCADE
);
CREATE INDEX document_backlinks ON document_links(workspace_id,target_id);
