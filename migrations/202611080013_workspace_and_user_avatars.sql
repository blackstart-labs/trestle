-- Forward migration: workspace and user avatars, and allow email characters in username
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS avatar_data BYTEA;
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS avatar_mime TEXT;
ALTER TABLE workspaces ADD COLUMN IF NOT EXISTS avatar_updated_at TIMESTAMPTZ;

ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_data BYTEA;
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_mime TEXT;
ALTER TABLE users ADD COLUMN IF NOT EXISTS avatar_updated_at TIMESTAMPTZ;

ALTER TABLE users DROP CONSTRAINT IF EXISTS users_username_ascii;
ALTER TABLE users ADD CONSTRAINT users_username_ascii CHECK (username ~ '^[a-z0-9._@+-]+$');

