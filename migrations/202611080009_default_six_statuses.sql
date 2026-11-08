-- Forward migration: standard 6 default statuses: Backlog, Todo, In Progress, In Review, Done, Cancelled

-- 1. Rename existing "To Do" to "Todo" where "Todo" does not exist yet
UPDATE task_statuses
SET name = 'Todo', updated_at = now()
WHERE name = 'To Do'
  AND NOT EXISTS (
      SELECT 1 FROM task_statuses ts2
      WHERE ts2.project_id = task_statuses.project_id AND ts2.name = 'Todo'
  );

-- 2. Ensure "In Review" exists for every project
INSERT INTO task_statuses (id, workspace_id, project_id, name, category, color, position)
SELECT gen_random_uuid(), p.workspace_id, p.id, 'In Review', 'started', '#6366f1', 3
FROM projects p
WHERE NOT EXISTS (
    SELECT 1 FROM task_statuses ts
    WHERE ts.project_id = p.id AND ts.name = 'In Review'
)
ON CONFLICT (project_id, name) DO NOTHING;

-- 3. Adjust standard status positions
UPDATE task_statuses SET position = 0 WHERE name = 'Backlog';
UPDATE task_statuses SET position = 1 WHERE name = 'Todo';
UPDATE task_statuses SET position = 2 WHERE name = 'In Progress';
UPDATE task_statuses SET position = 3 WHERE name = 'In Review';
UPDATE task_statuses SET position = 4 WHERE name = 'Done';
UPDATE task_statuses SET position = 5 WHERE name = 'Cancelled';

