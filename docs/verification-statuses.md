# Custom Status Administration Verification (M05a)

> Recorded against the former Rust backend ([ADR 0005](adr/0005-go-backend.md)). Go equivalents: `cargo fmt` → `gofmt -l .`, `cargo clippy` → `go vet ./...`, `cargo test` → `go test ./...` (set `DATABASE_URL`), `cargo build --release` → `CGO_ENABLED=0 go build -trimpath -o trestle .`, `cargo deny check` → `govulncheck ./...`.

Record for milestone M05a: Custom Status Administration and Consistent Transitions.

Date: 2026-11-02. Linux x86_64; Node 24.21.0/npm 11.19.0, Rust 1.99.0; PostgreSQL.
Scope: [Custom Status Administration Specification](features/custom-statuses.md).

## Executed Verifications

- **Database Migration**:
  - `migrations/202611020005_custom_statuses.sql` adds `task_statuses` table with project scoping, unique names per project, valid hex color check, canonical category check, position ordering, and `tasks.status_id` foreign key.
  - Backward compatibility: migration automatically backfills existing projects with standard 5 statuses and updates existing tasks to point to their corresponding `status_id`.

- **Backend Logic & Permissions**:
  - `src/projects/store.rs` defines status CRUD, reordering, and atomic deletion with safe task remapping.
  - Last-status protection: deleting the only status in a project returns `400 Bad Request`.
  - Active task protection: deleting a status with active tasks requires a valid `remap_to` status in the same project, reassigning tasks and logging audit activity entries.
  - Cross-tenant boundaries: status operations verify workspace and project ownership; members and guests cannot mutate statuses (403 Forbidden).

- **Rust Test Suite**:
  - `tests/custom_statuses.rs` exercises end-to-end status lifecycle: default 5 seed, 403 member rejection, 201 admin creation, duplicate name 409 conflict, update, position reordering, safe deletion with task remap requirement, and atomic reassignment.
  - All 71 Rust unit and integration tests pass (`cargo test`).
  - Formatting and Clippy clean (`cargo fmt --check`, `cargo clippy --all-targets -- -D warnings`).
  - Cargo deny clean (`cargo deny check`).

- **Frontend & Design System Integration**:
  - Svelte 5 runes (`$state`, `$derived`, `$props`, `$effect`) throughout frontend components.
  - `StatusManagerModal.svelte` provides visual status administration:
    - Canonical category pill badges (`backlog`, `unstarted`, `started`, `completed`, `cancelled`).
    - Curated color swatches with contrast validation.
    - Accessible reordering buttons with keyboard support and accessible aria-labels.
    - Inline edit mode for renaming and color modification.
    - Safe deletion prompt requiring target remap dropdown when tasks are assigned.
  - `TasksView.svelte` integrates custom statuses into task list, status selection dropdowns, and filtering.
  - `TaskDetailDrawer.svelte` displays custom status badges with color dots and allows transitions across project statuses.
  - `Icon.svelte` updated with SVG vector icons: `settings`, `arrow-up`, `arrow-down`, `pencil`, `alert-triangle`. Zero emojis used for functional UI.
  - Vitest test suite (`StatusManagerModal.test.ts`): all 8 test files and 38 frontend tests pass (`npm test`).
  - `svelte-check` reports 0 errors and 0 warnings. Prettier format passes (`npm run format:check`).
  - Production build succeeds (`npm run build`).

## Limitations & Next Milestones

- Custom status administration completes M05a.
- Milestone M05b will build on these statuses to provide accessible Kanban board drag-and-drop ordering, inline table editing, and saved view configurations.
