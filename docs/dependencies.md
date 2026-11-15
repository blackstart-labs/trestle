# Task dependencies

A dependency is a directed link between two tasks: one task is **blocked by**
another. It records ordering, not parentage. A subtask is still a task with a
parent; a dependency says only that one task should finish before the other.

Links are workspace-scoped, so a task in `CORE` can be blocked by a task in
`WEB`. They never cross workspaces.

## What the product guarantees

- **Direction is explicit.** Each link has one blocked task and one blocker.
  The blocked task's page shows it under **Blocked by**; the blocker's page
  shows the same link under **Blocks**.
- **No self-links.** A task cannot block itself.
- **No duplicates.** The pair (blocked, blocker) is unique.
- **No cycles.** Adding a link is rejected when the blocker already depends,
  directly or transitively, on the blocked task. The database transaction takes
  a workspace-scoped advisory lock, so two concurrent opposite links cannot both
  pass the check.
- **Both sides can remove.** Removing a link from either task removes it from
  both.
- **Guests read only.** Guests see links but cannot add or remove them.

Editing a Task's dependencies appends to the blocked task's Activity:
`dependency.added` and `dependency.removed`, with the other task's key.

## Where it lives

**Task drawer → Dependencies.** It is below Time Tracking and above Discussion.

- **Blocked by** and **Blocks** list the other task's key, title, status and
  project (when it differs). Archiving a Project does not delete its links; the
  row says “Archived project” so stale context is visible.
- **Link a task that** is a `Select` with two choices: *is blocked by* or
  *blocks*. The adjacent search field matches a task title or human key across
  the workspace, excluding archived Projects. Results appear after a 250 ms
  pause; a result is one button that adds the link in the chosen direction.
- Loading uses a live `status` message, errors stay inline with Retry, and an
  empty search says “No matching tasks.” There is no placeholder or seeded data.
- A link is a relation, not a resource with a page. Rows are text, not fake
  links, so opening a task in another Project still goes through the normal
  task route.

## API

All routes require a signed-in Workspace member and live under the task:

```
/api/v1/workspaces/{workspace_id}/projects/{project_id}/tasks/{task_id}/dependencies
```

| Method | Path | Body | Success |
| --- | --- | --- | --- |
| GET | `…/dependencies` | — | `{blocked_by: Link[], blocks: Link[]}` |
| POST | `…/dependencies` | `{related_task_id, direction}` | `201 {dependency: Link}` |
| DELETE | `…/dependencies/{dependency_id}` | — | `204` |

`direction` is `blocked_by` (this task waits for the related task) or `blocks`
(the related task waits for this task).

`Link` is `{id, created_at, task}` where `task` is
`{id, workspace_id, project_id, project_name, project_key, task_number,
human_key, title, status, status_id, status_name, status_color,
project_archived}`.

Failures:

- `400` — self-link, missing field, invalid UUID or unknown `direction`.
- `403` — guest member tries to add or remove.
- `404` — the task is not in the Project in the URL, the related task is not in
  the Workspace, or the dependency is not on this task.
- `409` — the link exists, or it would create a cycle.

### Task search

The picker uses a workspace-level search:

```
GET /api/v1/workspaces/{workspace_id}/tasks?query=CORE-12&limit=20
```

It matches a title or human key with a literal substring (case-insensitive),
skips archived Projects and returns at most 50 tasks. An empty query returns the
most recently updated tasks. `query` is limited to 100 characters. Any member,
including a guest, can search; non-members get `404`.

## Schema

`task_dependencies` stores `workspace_id`, the blocked `task_id`, the
`depends_on_task_id`, the blocked task's `project_id`, `created_by` and
`created_at`. A check constraint rejects self-links, a unique constraint rejects
duplicates, and both task foreign keys cascade on delete. The table is forward
migration `202611130016_task_dependencies.sql`.

## Limits and non-goals

- Dependencies do not change completion math. Completion remains
  completed ÷ (all − cancelled).
- They do not schedule work, set dates, or block status changes. A task can be
  marked complete while its blocker is open; the lists expose the mismatch
  instead of silently rewriting it.
- There is no Gantt view, critical path, lead/lag time, or dependency graph
  screen yet.
- A dependency can point at a task in an archived Project so it can be removed;
  the picker will not offer archived Projects for new links.
- Deleting either task deletes its links. There is no restore of a dependency.

## Verification

From the repository root:

```bash
# API, PostgreSQL, permissions, cycles and the concurrent-race guard
set -a; source .env; set +a
go test ./internal/httpapi -run 'TestTaskDependenc|TestWorkspaceTaskSearch' -count=1

# Frontend unit and build checks
cd web
npm run check
npm test
npm run lint:design
npm run build

# Fixture browser journeys for the drawer
npm run test:e2e -- e2e/dependencies.spec.ts
```

The Go tests cover both directions, cross-Project links, duplicates, self-links,
direct and indirect cycles, the concurrent opposite-edge race, guest and
non-member access, cross-workspace concealment, archived-Project search and
query validation. The Playwright spec covers both directions in the picker,
cross-Project search, removal, filtered-empty and the guest read-only state.
