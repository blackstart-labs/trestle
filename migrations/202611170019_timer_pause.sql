-- Pausing retains the active slot; it does not create a second timer or time log.
ALTER TABLE task_timers ADD COLUMN IF NOT EXISTS paused_at TIMESTAMPTZ;
ALTER TABLE task_timers ADD COLUMN IF NOT EXISTS paused_seconds DOUBLE PRECISION NOT NULL DEFAULT 0 CHECK(paused_seconds>=0 AND paused_seconds<'Infinity'::float8);
