# Projects and canonical tasks

Status: **Specification and execution plan for M04a**. Linked requirements: PR-02,
PR-07, PR-08; UC-01; NFR-SEC and NFR-DATA.

## 1. Purpose

Provide workspace members with authorized projects and canonical actionable tasks.
Establish tenant-isolated projects, transactional task key allocation (e.g. `ENG-14`),
canonical workflow status categories, and bounded, paginated lists end-to-end.

## 2. Definitions and Scope

Per [CONTEXT](../../CONTEXT.md):
- **Project**: Work organized within one workspace with its own settings, members, tasks, and workflow.
- **Task**: The canonical actionable work item (including bugs, features, and chores as classifications; never separate issue/ticket identities).
- **Status category**: Stable semantic grouping: `backlog`, `unstarted`, `started`, `completed`, `cancelled`.
- **Priority**: `none`, `low`, `medium`, `high`, `urgent`.

M04a delivers:
1. Scoped projects within an authorized workspace.
2. Canonical tasks with transactional human keys (`<KEY>-<NUMBER>`).
3. Status category lifecycle and transitions.
4. Bounded paginated task lists with status filters.
5. Svelte project and task management UI views.

Deferred to later milestones:
- Labels, assignees, comments, subtasks, activity history (M04b).
- Custom workflow status administration and remap rules (M05a).
- Interactive Kanban drag-and-drop boards and table view (M05b).

## 3. Security and Permissions

Following [permissions matrix](../security/permissions-matrix.md):
- Workspace Owner & Admin: Create projects, archive projects, view/edit all projects and tasks.
- Workspace Member & Guest: View visible projects in their workspace, create tasks, update tasks in authorized projects.
- Non-members: Concealed `404 Not Found` on any workspace or project lookup (prevents tenant enumeration).
- Cross-tenant boundaries: Every query enforces `workspace_id` matching the authenticated membership.

## 4. Proposed Database Schema

### `projects`
- `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
- `workspace_id UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE`
- `name TEXT NOT NULL` (1–120 chars)
- `slug TEXT NOT NULL` (2–48 chars, lowercase alphanumeric + hyphen)
- `key TEXT NOT NULL` (2–8 uppercase ASCII chars, e.g. `TREST`, `ENG`)
- `description TEXT NOT NULL DEFAULT ''`
- `task_counter BIGINT NOT NULL DEFAULT 0`
- `archived_at TIMESTAMPTZ`
- `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- Constraints:
  - `UNIQUE (workspace_id, slug)`
  - `UNIQUE (workspace_id, key)`

### `tasks`
- `id UUID PRIMARY KEY DEFAULT gen_random_uuid()`
- `workspace_id UUID NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE`
- `project_id UUID NOT NULL REFERENCES projects (id) ON DELETE CASCADE`
- `task_number BIGINT NOT NULL`
- `title TEXT NOT NULL` (1–255 chars)
- `description TEXT NOT NULL DEFAULT ''`
- `status TEXT NOT NULL DEFAULT 'unstarted'`
- `priority TEXT NOT NULL DEFAULT 'none'`
- `due_date DATE`
- `created_by UUID REFERENCES users (id) ON DELETE SET NULL`
- `created_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- `updated_at TIMESTAMPTZ NOT NULL DEFAULT now()`
- Constraints:
  - `status IN ('backlog', 'unstarted', 'started', 'completed', 'cancelled')`
  - `priority IN ('none', 'low', 'medium', 'high', 'urgent')`
  - `UNIQUE (project_id, task_number)`

## 5. HTTP Interface Contracts

| Method | Path | Auth | Description |
|---|---|---|---|
| `GET` | `/api/v1/workspaces/{workspace_id}/projects` | Member | List active projects |
| `POST` | `/api/v1/workspaces/{workspace_id}/projects` | Admin/Owner | Create project |
| `GET` | `/api/v1/workspaces/{workspace_id}/projects/{project_id}` | Member | Get project details |
| `PATCH` | `/api/v1/workspaces/{workspace_id}/projects/{project_id}` | Admin/Owner | Update project name, slug, or description; project key stays fixed |
| `POST` | `/api/v1/workspaces/{workspace_id}/projects/{project_id}/archive` | Admin/Owner | Archive project |
| `GET` | `/api/v1/workspaces/{workspace_id}/projects/{project_id}/tasks` | Member | Paginated tasks (`?status=&limit=&offset=`) |
| `POST` | `/api/v1/workspaces/{workspace_id}/projects/{project_id}/tasks` | Member | Create task |
| `GET` | `/api/v1/workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id}` | Member | Get task details |
| `PATCH` | `/api/v1/workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id}` | Member | Update task (status, priority, title, description) |
| `DELETE` | `/api/v1/workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id}` | Admin/Owner | Delete task |

All endpoints require active session authentication and validate membership in `workspace_id`.
All mutations require a valid `X-CSRF-Token` header.
Pagination parameters are bounded: default `limit = 20`, max `limit = 100`.

Project editing accepts any non-empty subset of `name`, `slug`, and `description`.
Names are trimmed (1–120 bytes); slugs are trimmed, lowercased, and validated with
the creation rules (2–48 ASCII letters, numbers or hyphens, no leading/trailing
hyphen). Descriptions allow 0–2000 bytes; an empty string clears the description.
Duplicate workspace slugs return 409 and leave all fields unchanged. The update
and `project.updated` audit event commit together. Project IDs, keys, task numbers,
archive state, and creation timestamps remain unchanged. Supplying a `key` returns
400; task references must stay stable. A missing/cross-workspace project returns
404, and members cannot edit project metadata.

Owners/admins open **Edit** from a project card or **Edit project** from its task
header. The shared dialog preserves input on failure, saves only changed fields,
disables duplicate submission, and confirms before discarding unsaved changes.
