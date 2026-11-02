# Feature Specification: Custom Status Administration and Consistent Transitions (M05a)

Status: Approved specification  
Milestone: M05a (Phase 5)  
Requirements: PR-02, PR-03, PR-07, PR-10  
Dependencies: M04a, M04b  

---

## 1. Context and Problem Statement

In M04a and M04b, tasks rely on five hardcoded canonical status categories:
`backlog`, `unstarted`, `started`, `completed`, and `cancelled`.

Real development workflows require project-specific statuses (such as "In Review", "Blocked", "QA Testing", "Staging Verification", "Deployed") while strictly preserving stable completion semantics (PR-03). A status named "Deployed" or "Shipped" must have unambiguous completion meaning, and progress calculations, cycle burndowns, and metrics must not break when teams rename or customize statuses.

Furthermore, deleting a status must never leave tasks in an orphaned or undefined state. Safe deletion requires atomic remapping of existing tasks to a valid destination status in the same project.

---

## 2. Core Principles & Semantic Contracts

1. **Category Mapping Rule**:
   Every custom status belongs to a project and maps to exactly one immutable canonical category:
   - `backlog`: Work noted but not committed to an active backlog queue.
   - `unstarted`: Work triaged and ready for execution (e.g. "To Do", "Ready for Dev").
   - `started`: Active effort in progress (e.g. "In Development", "Code Review", "QA").
   - `completed`: Terminal success state; satisfies completion semantics (e.g. "Done", "Shipped", "Verified").
   - `cancelled`: Terminal dropped state; excluded from active completion (e.g. "Won't Do", "Duplicate").

2. **Default Project Seed**:
   When a project is created, it is seeded with 5 default statuses matching the canonical categories:
   - "Backlog" (`backlog`, `#4c544c`)
   - "To Do" (`unstarted`, `#1e4d79`)
   - "In Progress" (`started`, `#775300`)
   - "Done" (`completed`, `#26633b`)
   - "Cancelled" (`cancelled`, `#9f302b`)

3. **Safe Deletion & Remap Contract**:
   - A status cannot be deleted if it is the only status in the project.
   - If tasks are assigned to the status to be deleted, the client must supply a valid `remap_to` status ID belonging to the same project.
   - If tasks exist and no `remap_to` is provided, deletion fails with `400 Bad Request`.
   - The deletion and task reassignment execute atomically in a single PostgreSQL transaction.
   - A `task.status_changed` activity entry is recorded for remapped tasks.

4. **Tenant Isolation & Permissions**:
   - Statuses are scoped strictly by `workspace_id` and `project_id`.
   - Creating, editing, reordering, and deleting statuses requires `owner` or `admin` role in the workspace.
   - Members and guests can view statuses and transition tasks between them.

5. **Sorting & Ordering**:
   - Statuses are ordered within a project by `position ASC, created_at ASC`.
   - The API supports reordering positions within a project.

---

## 3. Database Schema

```sql
CREATE TABLE task_statuses (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workspace_id UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE,
    name TEXT NOT NULL CHECK (char_length(trim(name)) >= 1 AND char_length(name) <= 50),
    category TEXT NOT NULL CHECK (category IN ('backlog', 'unstarted', 'started', 'completed', 'cancelled')),
    color TEXT NOT NULL DEFAULT '#315b3b' CHECK (color ~* '^#[0-9a-f]{6}$'),
    position INT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_task_statuses_project_name UNIQUE (project_id, name)
);

CREATE INDEX idx_task_statuses_project ON task_statuses (workspace_id, project_id, position);

ALTER TABLE tasks
    ADD COLUMN status_id UUID REFERENCES task_statuses (id) ON DELETE RESTRICT;

CREATE INDEX idx_tasks_status_id ON tasks (status_id);
```

---

## 4. HTTP API Specifications

### `GET /api/v1/workspaces/{workspace_id}/projects/{project_id}/statuses`
- Returns array of `TaskStatusItem` sorted by `position ASC`.
- Requires workspace membership.

### `POST /api/v1/workspaces/{workspace_id}/projects/{project_id}/statuses`
- Payload: `{ "name": "In QA", "category": "started", "color": "#775300" }`
- Returns: `201 Created` with created status.
- Requires workspace `owner` or `admin`.

### `PATCH /api/v1/workspaces/{workspace_id}/projects/{project_id}/statuses/{status_id}`
- Payload: `{ "name": "Quality Assurance", "color": "#8c6200" }` (category cannot be changed once created).
- Returns: `200 OK` with updated status.

### `PUT /api/v1/workspaces/{workspace_id}/projects/{project_id}/statuses/positions`
- Payload: `{ "status_ids": ["uuid-1", "uuid-2", "uuid-3"] }`
- Reorders positions atomically.

### `DELETE /api/v1/workspaces/{workspace_id}/projects/{project_id}/statuses/{status_id}`
- Query parameter or body: `{ "remap_to": "uuid-destination" }`
- Returns: `204 No Content` on success.
- If tasks exist and `remap_to` is omitted: `400 Bad Request` with message `"Cannot delete status with active tasks without remap_to"`.

---

## 5. User Interface Specifications

- Status manager modal accessed from project header or settings.
- Grouped by category with distinct color badges.
- Visual status reordering with accessible up/down arrow buttons.
- Delete confirmation modal with target status select box when tasks are present.
- Integration into `TasksView` column/filters and `TaskDetailDrawer` status dropdowns.
