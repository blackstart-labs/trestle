-- Forward migration: add start_date and color to milestones

ALTER TABLE milestones
    ADD COLUMN IF NOT EXISTS start_date DATE,
    ADD COLUMN IF NOT EXISTS color TEXT NOT NULL DEFAULT '#6366f1' CHECK (color ~* '^#[0-9a-f]{6}$');
