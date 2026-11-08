-- Forward migration: add topic_id to project_integrations for Telegram forum supergroups
ALTER TABLE project_integrations ADD COLUMN topic_id TEXT NOT NULL DEFAULT '';
