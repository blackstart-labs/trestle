# Task details, labels, assignees, subtasks, and comments

Status: **Specification and execution plan for M04b**. Linked requirements: PR-02,
PR-04, PR-07, PR-08; UC-01; NFR-SEC and NFR-DATA.

## 1. Purpose

Extend canonical tasks with rich collaboration and organization primitives:
project labels, task assignees, hierarchical subtasks, threaded comments,
and activity audit history. Provide responsive, accessible UI for inspecting
and editing task details in a dedicated detail drawer or view.

## 2. Definitions and Scope

Per [CONTEXT](../../CONTEXT.md):
- **Task**: The single canonical work item (including bugs, features, and chores as classifications; never separate ticket types).
- **Subtask**: A child task linked to a parent task within the same project. Subtasks retain standard task identities, numbers, and keys.
- **Label**: Project-scoped tags with curated, accessible color palettes for categorization and filtering.
- **Assignee**: Workspace member assigned to execute the work item.
- **Comment**: Author-attributed threaded remark on a task with timestamp.
- **Activity**: Append-only log of changes to a task's lifecycle (status changes, re-assignments, label attachments).

M04b delivers:
1. Project-scoped labels creation, listing, attachment, and deletion.
2. Member assignment and unassignment on tasks.
3. Subtask creation, listing, and completion tracking.
4. Threaded task comments (create, list, delete).
5. Task activity history recording and display.
6. Multi-dimensional task list filtering (by status, label, assignee).
7. Accessible, high-polish Svelte task detail drawer.

## 3. Security and Permissions

Following [permissions matrix](../security/permissions-matrix.md):
- **Workspace Owner & Admin**: Manage labels, assign any member, edit/delete any comment, delete subtasks.
- **Workspace Member**: Attach/detach labels, assign/unassign themselves or teammates, create subtasks, post comments, delete own comments.
- **Workspace Guest**: View assigned tasks, post comments, update assigned task status.
- **Non-members**: Concealed `404 Not Found` on any workspace, project, or task lookup (tenant enumeration prevention).
- **Tenant Integrity**: All queries enforce `workspace_id` matching authenticated membership. All subtasks and labels must belong to the same `project_id`.

## 4. Proposed Database Schema

### `labels`
- `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
- `workspace_id UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE`
- `project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE`
- `name TEXT NOT NULL` (1–50 chars)
- `color TEXT NOT NULL DEFAULT '#315b3b'` (hex color)
- `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- Constraints:
  - `UNIQUE (project_id, name)`

### `task_label_assignments`
- `task_id UUID NOT NULL REFERENCES tasks (id) ON DELETE CASCADE`
- `label_id UUID NOT NULL REFERENCES labels (id) ON DELETE CASCADE`
- `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- `PRIMARY KEY (task_id, label_id)`

### `task_assignees`
- `task_id UUID NOT NULL REFERENCES tasks (id) ON DELETE CASCADE`
- `user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE`
- `assigned_by UUID REFERENCES users (id) ON DELETE SET NULL`
- `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- `PRIMARY KEY (task_id, user_id)`

### `tasks` (alterations)
- `parent_id UUID REFERENCES tasks (id) ON DELETE CASCADE`
- Constraint: `parent_id != id`

### `task_comments`
- `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
- `workspace_id UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE`
- `task_id UUID NOT NULL REFERENCES tasks (id) ON DELETE CASCADE`
- `user_id UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE`
- `content TEXT NOT NULL` (1–5000 chars)
- `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`

### `task_activities`
- `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
- `workspace_id UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE`
- `task_id UUID NOT NULL REFERENCES tasks (id) ON DELETE CASCADE`
- `actor_id UUID REFERENCES users (id) ON DELETE SET NULL`
- `action TEXT NOT NULL`
- `details JSONB NOT NULL DEFAULT '{}'`
- `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`

## 5. HTTP Interface Contracts

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/api/v1/workspaces/{w_id}/projects/{p_id}/labels` | Member | List project labels |
| `POST` | `/api/v1/workspaces/{w_id}/projects/{p_id}/labels` | Member | Create label |
| `DELETE` | `/api/v1/workspaces/{w_id}/projects/{p_id}/labels/{label_id}` | Admin/Owner | Delete label |
| `POST` | `/api/v1/workspaces/{w_id}/projects/{p_id}/tasks/{t_id}/labels` | Member | Attach label |
| `DELETE` | `/api/v1/workspaces/{w_id}/projects/{p_id}/tasks/{t_id}/labels/{label_id}` | Member | Detach label |
| `POST` | `/api/v1/workspaces/{w_id}/projects/{p_id}/tasks/{t_id}/assignees` | Member | Assign user |
| `DELETE` | `/api/v1/workspaces/{w_id}/projects/{p_id}/tasks/{t_id}/assignees/{user_id}` | Member | Unassign user |
| `GET` | `/api/v1/workspaces/{w_id}/projects/{p_id}/tasks/{t_id}/subtasks` | Member | List subtasks |
| `POST` | `/api/v1/workspaces/{w_id}/projects/{p_id}/tasks/{t_id}/subtasks` | Member | Create subtask |
| `GET` | `/api/v1/workspaces/{w_id}/projects/{p_id}/tasks/{t_id}/comments` | Member | List task comments |
| `POST` | `/api/v1/workspaces/{w_id}/projects/{p_id}/tasks/{t_id}/comments` | Member | Post comment |
| `DELETE` | `/api/v1/workspaces/{w_id}/projects/{p_id}/tasks/{t_id}/comments/{c_id}` | Author/Admin | Delete comment |
| `GET` | `/api/v1/workspaces/{w_id}/projects/{p_id}/tasks/{t_id}/activity` | Member | Get activity history |
