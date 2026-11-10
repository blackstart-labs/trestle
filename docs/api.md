# REST API Specification (`/api/v1`)

Trestle exposes a JSON REST API under `/api/v1` served by the Go backend. All requests and responses use `application/json` unless serving raw binary media (such as project avatars).

---

## 1. Request Conventions

### Authentication
Authentication is session-based via HTTP cookies:
- Cookie Name: `trestle_session`
- Flags: `HttpOnly`, `SameSite=Lax`, `Path=/`, `Secure` (when TLS enabled).
- Lifespan: Configured via `SESSION_TTL_HOURS` (default 336 hours / 14 days).

### CSRF Defense
All unsafe state-changing HTTP methods (`POST`, `PUT`, `PATCH`, `DELETE`) require a CSRF token:
1. The server sets a non-HttpOnly readable cookie: `trestle-csrf=<token>`.
2. The client transmits this token in the header:
   ```http
   X-CSRF-Token: <token>
   ```
3. The server compares both tokens using constant-time comparison before processing the request.

### Envelopes & Formatting
- Request bodies must be valid UTF-8 JSON.
- Response bodies return JSON objects with data payloads or standard pagination containers.
- Dates and timestamps are represented in ISO 8601 / RFC 3339 format (`YYYY-MM-DDTHH:MM:SSZ`).
- Date-only properties (due dates, milestone targets) are represented as `YYYY-MM-DD`.

### Standard Error Envelope
On failure (HTTP 4xx or 5xx), the API returns a structured error object:
```json
{
  "error": {
    "code": "invalid_input",
    "message": "Title is required and must not exceed 255 characters"
  }
}
```

Standard error codes:
- `unauthorized`: Session cookie is missing, expired, or invalid.
- `forbidden`: Insufficient workspace or project role permissions.
- `not_found`: Requested resource does not exist or user lacks visibility.
- `conflict`: Unique constraint violation (e.g. duplicate username, slug, or status name).
- `invalid_input`: Request body failed validation.
- `rate_limited`: Operation throttled (e.g. outbound webhook testing).
- `internal_error`: Unexpected server error (credentials and stack traces redacted).

### Pagination Convention
Collection endpoints accept two query parameters:
- `limit`: Number of items per page (default `20`, maximum `100`).
- `offset`: Zero-based item offset.

Response envelope:
```json
{
  "tasks": [...],
  "total": 42,
  "limit": 20,
  "offset": 0
}
```

---

## 2. API Endpoints

### 2.1 Identity & Session

| Method | Endpoint | Description |
| --- | --- | --- |
| `POST` | `/api/v1/session` | Authenticate with username/email and password. Returns user object and sets `trestle_session` & `trestle-csrf` cookies. |
| `DELETE` | `/api/v1/session` | Invalidate current session and clear cookies. |
| `GET` | `/api/v1/me` | Return the profile of the currently signed-in user. |

### 2.2 Workspaces & Members

`POST /api/v1/workspaces` accepts `{ "name": "Engineering", "slug": "engineering" }`.
It requires a signed-in session and CSRF token. The creator becomes owner in the same transaction.
Slugs are globally unique (409 on conflict), 2–48 lowercase letters, numbers or hyphens, with no leading/trailing hyphen.
Names are 1–120 characters. `PATCH /api/v1/workspaces/{workspace_id}` accepts the same fields for owners/admins and returns the caller's role.

Task creation accepts optional `assignee_ids` (up to 100 active Workspace member UUIDs) and `milestone_id`.
All metadata is validated and written in the Task transaction; invalid references roll back creation.
Task creation, subtask creation and Task PATCH accept descriptions up to 15,000 Unicode characters.
Task responses include `subtask_count`, the number of direct children in the same Workspace and Project, not just those on the current page.

Project and Workspace JSON exports read every task page from a stable database snapshot.
Workspace exports include archived Projects. A failed read returns an error instead of a successful partial backup.

| Method | Endpoint | Description |
| --- | --- | --- |
| `GET` | `/api/v1/workspaces` | List all workspaces where the current user is a member. |
| `GET` | `/api/v1/workspaces/{id}/members` | List members of a workspace with roles (`owner`, `admin`, `member`, `guest`). |
| `POST` | `/api/v1/workspaces/{id}/members` | Add an existing user to the workspace with a designated role. |
| `PATCH` | `/api/v1/workspaces/{id}/members/{user_id}` | Update a member's role (`admin` or `owner` permission required). |
| `DELETE` | `/api/v1/workspaces/{id}/members/{user_id}` | Remove a member from the workspace. |
| `POST` | `/api/v1/workspaces/{id}/transfer-ownership` | Transfer workspace ownership (`owner` permission required). |

### 2.3 Projects

| Method | Endpoint | Description |
| --- | --- | --- |
| `GET` | `/api/v1/workspaces/{ws_id}/projects` | List projects in the workspace. |
| `POST` | `/api/v1/workspaces/{ws_id}/projects` | Create a new project (name, key, description). |
| `GET` | `/api/v1/projects/{id}` | Retrieve project details, task counters, and live completion stats. |
| `PATCH` | `/api/v1/projects/{id}` | Update project name, description, or slug. |
| `POST` | `/api/v1/projects/{id}/archive` | Archive or unarchive a project. |
| `GET` | `/api/v1/projects/{id}/image` | Retrieve project avatar image. |
| `PUT` | `/api/v1/projects/{id}/image` | Upload project avatar image (PNG, JPEG, WebP, SVG). |
| `DELETE` | `/api/v1/projects/{id}/image` | Remove project avatar image. |

### 2.4 Workflow Statuses

| Method | Endpoint | Description |
| --- | --- | --- |
| `GET` | `/api/v1/projects/{id}/statuses` | List custom workflow statuses ordered by position. |
| `POST` | `/api/v1/projects/{id}/statuses` | Create a new workflow status with category and color token. |
| `PUT` | `/api/v1/projects/{id}/statuses/positions` | Atomically reorder project statuses by passing ordered status IDs. |
| `PATCH` | `/api/v1/projects/{id}/statuses/{status_id}` | Update status name, color, or category. |
| `DELETE` | `/api/v1/projects/{id}/statuses/{status_id}` | Delete a status (disallowed if tasks still use it). |

### 2.5 Milestones

| Method | Endpoint | Description |
| --- | --- | --- |
| `GET` | `/api/v1/projects/{id}/milestones` | List project milestones with progress rollups (`completed_tasks`, `total_tasks`). |
| `POST` | `/api/v1/projects/{id}/milestones` | Create a milestone with title, description, start date, due date, and color. |
| `GET` | `/api/v1/projects/{id}/milestones/{milestone_id}` | Get milestone details and task associations. |
| `PATCH` | `/api/v1/projects/{id}/milestones/{milestone_id}` | Update milestone attributes or toggle status (`open`, `closed`). |
| `DELETE` | `/api/v1/projects/{id}/milestones/{milestone_id}` | Delete milestone (detaches all assigned tasks). |

### 2.6 Tasks & Collaboration

| Method | Endpoint | Description |
| --- | --- | --- |
| `GET` | `/api/v1/projects/{id}/tasks` | List tasks filtered by `status_id`, `priority`, `assignee`, or `milestone_id`. |
| `POST` | `/api/v1/projects/{id}/tasks` | Create task; automatically allocates sequential `human_key` (e.g. `CORE-14`). |
| `GET` | `/api/v1/tasks/{id}` | Get full task detail including assignees, labels, subtasks, and milestone. |
| `PATCH` | `/api/v1/tasks/{id}` | Update task title, description, status, priority, due date, or milestone. |
| `DELETE` | `/api/v1/tasks/{id}` | Delete task and cascade child records. |
| `GET` | `/api/v1/tasks/{id}/comments` | List discussion comments on a task. |
| `POST` | `/api/v1/tasks/{id}/comments` | Add a comment to a task thread. |
| `DELETE` | `/api/v1/tasks/{id}/comments/{comment_id}` | Delete comment (author or admin only). |
| `GET` | `/api/v1/tasks/{id}/activity` | Retrieve property change audit history for a task. |
| `POST` | `/api/v1/tasks/{id}/assignees` | Assign a user to the task. |
| `DELETE` | `/api/v1/tasks/{id}/assignees/{user_id}` | Unassign a user from the task. |
| `GET` | `/api/v1/projects/{id}/labels` | List taxonomy labels available in the project. |
| `POST` | `/api/v1/projects/{id}/labels` | Create a new project label with name and color. |
| `DELETE` | `/api/v1/projects/{id}/labels/{label_id}` | Delete a label definition. |
| `POST` | `/api/v1/tasks/{id}/labels` | Attach a label to a task. |
| `DELETE` | `/api/v1/tasks/{id}/labels/{label_id}` | Detach a label from a task. |
| `GET` | `/api/v1/tasks/{id}/subtasks` | List subtasks belonging to a parent task. |
| `POST` | `/api/v1/tasks/{id}/subtasks` | Create a child subtask under this task. |

### 2.7 Project Integrations

| Method | Endpoint | Description |
| --- | --- | --- |
| `GET` | `/api/v1/projects/{id}/integrations` | List configured integration providers (`telegram`, `slack`, `discord`). |
| `PUT` | `/api/v1/projects/{id}/integrations/{provider}` | Configure webhook URL or bot credentials (write-only storage). |
| `DELETE` | `/api/v1/projects/{id}/integrations/{provider}` | Disconnect and purge integration credentials. |
| `POST` | `/api/v1/projects/{id}/integrations/{provider}/test` | Trigger a test notification to verify connectivity. |

### 2.8 System & Probes

| Method | Endpoint | Description |
| --- | --- | --- |
| `GET` | `/health/live` | Liveness probe: returns `200 OK` immediately if HTTP server is running. |
| `GET` | `/health/ready` | Readiness probe: checks database ping and connection pool health. Returns `200 OK` or `503 Service Unavailable`. |
