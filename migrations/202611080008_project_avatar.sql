-- Project avatar / image storage.
-- Adds binary image payload, MIME type, and update timestamp directly to projects.

ALTER TABLE projects
    ADD COLUMN IF NOT EXISTS avatar_data BYTEA,
    ADD COLUMN IF NOT EXISTS avatar_mime TEXT,
    ADD COLUMN IF NOT EXISTS avatar_updated_at TIMESTAMPTZ;

