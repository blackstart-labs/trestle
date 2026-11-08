# Database Architecture & Schema

Trestle uses PostgreSQL (15+) as its single source of truth for application state. Database access is managed in Go via `pgx/v5` (`pgxpool.Pool`) with compile-time embedded forward migrations.

---

## 1. Connection Management (`pgxpool`)

Database connections are managed in `internal/database/db.go`.

- **Lazy Connection**: The application creates the connection pool without immediately blocking on connection establishment. This ensures that the HTTP server can bind ports and respond to liveness probes (`/health/live`) even if PostgreSQL is temporarily starting up or recovering.
- **Connection Bounds**:
  - `MinConns = 0`: Allows the pool to contract completely during idle periods.
  - `MaxConns`: Defaults to 25 (or configured via environment), preventing connection exhaustion on the PostgreSQL server.
  - `ConnectTimeout`: Configured (default 5s) to fail fast on network isolation.
- **Session Protections**:
  - `statement_timeout = 5000`: Protects against runaway queries locking CPU and connection slots.
  - `lock_timeout = 5000`: Aborts transactions that cannot acquire table or row locks within 5 seconds, preventing cascading deadlocks.

---

## 2. Migration Engine & Lifecycle

Schema migrations are managed in `internal/database/migrate.go` and stored as SQL files in `migrations/`:

```text
migrations/
├── 202610150001_foundation.sql
├── 202611010002_identity.sql
├── 202611020003_projects_and_tasks.sql
├── 202611020004_task_collaboration.sql
├── 202611020005_custom_statuses.sql
├── 202611070006_project_integrations.sql
├── 202611080007_project_milestones.sql
├── 202611080008_project_avatar.sql
├── 202611080009_default_six_statuses.sql
├── 202611080010_milestone_color_dates.sql
└── embed.go
```

### Invariants:
1. **Explicit Forward Execution**: Migrations are applied via `trestle migrate` prior to serving traffic. The HTTP server does not apply migrations automatically at runtime.
2. **Distributed Advisory Lock**: Concurrent migration attempts (e.g. multi-replica deployments) synchronize via PostgreSQL session advisory locking (`pg_advisory_lock`), guaranteeing single-instance execution.
3. **Tracking & Integrity**: The `_sqlx_migrations` table records:
   - `version` (`BIGINT`): 12-digit timestamp identifier.
   - `description` (`TEXT`): Human-readable migration purpose.
   - `installed_on` (`TIMESTAMPTZ`): Execution timestamp.
   - `success` (`BOOLEAN`): Execution outcome.
   - `checksum` (`BYTEA`): SHA-384 hash of the file contents.
4. **Immutability Contract**: Never modify an already-applied migration file; doing so invalidates the SHA-384 checksum and causes startup validation failure. Always append a new forward migration.

---

## 3. Core Tables & Relationships

```text
users
  ├── workspace_members ── workspaces
  │                           └── projects
  │                                 ├── task_statuses
  │                                 ├── project_milestones
  │                                 ├── project_integrations
  │                                 └── tasks
  │                                       ├── task_assignees ── users
  │                                       ├── task_labels ───── labels
  │                                       ├── task_comments ─── users
  │                                       └── task_activity ─── users
  └── sessions
```

### Table Definitions

| Table | Primary Key | Key Columns & Indexes | Description |
| --- | --- | --- | --- |
| `users` | `id` (UUID) | `username` (UNIQUE), `email` (UNIQUE) | Operator and member accounts with Argon2id password hashes. |
| `sessions` | `id` (UUID) | `user_id` (FK), `token_hash` (UNIQUE), `expires_at` | Active user login sessions. |
| `workspaces` | `id` (UUID) | `slug` (UNIQUE) | Primary tenant boundary. |
| `workspace_members` | `(workspace_id, user_id)` | `role` (`owner`, `admin`, `member`, `guest`) | RBAC mapping of users to workspaces. |
| `projects` | `id` (UUID) | `(workspace_id, slug)` (UNIQUE), `(workspace_id, key)` (UNIQUE) | Work container with human prefix (e.g. `CORE`) and task counter. |
| `task_statuses` | `id` (UUID) | `(project_id, name)` (UNIQUE), `(project_id, position)` | Custom workflow statuses mapped to canonical categories. |
| `project_milestones` | `id` (UUID) | `project_id` (FK), `status` (`open`, `closed`), `due_date` | Deliverable groupings with start/due dates and progress tracking. |
| `tasks` | `id` (UUID) | `(project_id, task_number)` (UNIQUE), `human_key` (UNIQUE) | Actionable work items. `status_id` references `task_statuses`. |
| `task_assignees` | `(task_id, user_id)` | `task_id` (FK), `user_id` (FK) | Multiple assignees per task. |
| `labels` | `id` (UUID) | `(project_id, name)` (UNIQUE) | Project-specific taxonomy tags with color metadata. |
| `task_labels` | `(task_id, label_id)` | `task_id` (FK), `label_id` (FK) | Association of labels to tasks. |
| `task_comments` | `id` (UUID) | `task_id` (FK), `created_at` | Task discussion thread. |
| `task_activity` | `id` (UUID) | `task_id` (FK), `created_at` | Audit log of property changes and task lifecycle events. |
| `project_integrations` | `(project_id, provider)` | `project_id` (FK), `provider` (`telegram`, `slack`, `discord`) | Webhook delivery configuration and encrypted credentials. |

---

## 4. Key Schema Patterns

### Atomic Sequential Key Allocation
Human-readable task keys (e.g. `CORE-42`) are strictly monotonic per project. Within a transaction:
```sql
UPDATE projects
SET task_counter = task_counter + 1
WHERE id = $1
RETURNING task_counter;
```
The resulting number is formatted with the project key to form `human_key` (`CORE-42`), preventing race conditions and duplicate keys under concurrent task creation.

### Status Category Mapping
Custom statuses always roll up into one of five immutable categories:
- `backlog`: Pending, not yet scheduled.
- `unstarted`: Scheduled or ready to be pulled into a sprint.
- `started`: Active in-progress or in-review work.
- `completed`: Successfully finished.
- `cancelled`: Discarded or abandoned work.

Completion metrics consistently use:
$$\text{Completion Rate} = \frac{\text{Completed}}{\text{All Tasks} - \text{Cancelled}}$$

### Foreign Key Cascades & Soft Deletion
- Deleting a Workspace cascades and purges its projects, tasks, and memberships.
- Deleting a Project purges its tasks, statuses, milestones, and integrations.
- Deleting a Task cascades to its assignees, labels, comments, and activity audit logs.
- Archive operations (`archived = true` on `projects`) are preferred over destructive deletion.

