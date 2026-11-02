# Feature matrix

Last reviewed: 2026-10-15. This file owns capability status. `Implemented` means
the scoped behavior has implementation and executed verification; it does not
imply production readiness. `Partial` means an explicit subset exists. `Planned`
means no executable product capability. `Deprecated` requires a replacement and
migration policy; nothing is deprecated currently.

| Capability / requirement | Status | Scope and implementation | Evidence / prerequisite |
| --- | --- | --- | --- |
| Backend foundation / PR-08, PR-10 | Implemented | Rust config, live/ready, explicit metadata migration, logs/errors | [Approved scope](../foundation-milestone.md), `src/`, [verification](../verification.md), commit `79478d7` |
| CI and deployment / PR-08 | Partial | Workflow, Docker image and dev database configuration | Local lint/build/runtime pass; hosted CI, production upgrades and full notices remain unverified |
| Product documentation / PR-10 | Partial | Vision, requirements, inventory, roadmap, core architecture and initial identity spec | This documentation milestone; detailed later feature specs not written yet |
| Frontend foundation / PR-07/08/10 | Implemented | Svelte/Vite/TypeScript; embedded assets; visible readiness/retry, no product UI | [Spec](../features/frontend-foundation.md), [M01b evidence](../verification-frontend.md) |
| Application shell / PR-07 | Implemented | Real section navigation, skip/focus, local appearance preference and existing readiness | [M02 spec](../features/application-shell.md), [evidence](../verification-shell.md) |
| Design system and mobile workflows / PR-07 | Partial | Semantic light/dark tokens; native button/select and feedback; responsive public shell | [Tokens](../design/design-tokens.md), [components](../design/components.md); no product/mobile workflow or full accessibility certification |
| Identity/workspaces / PR-01 | Implemented | One-time bootstrap, Argon2id credentials, session/CSRF tokens, login limits, operator provisioning/recovery CLI, multi-user workspace memberships, owner transfer with reauthentication, and last-owner preservation | [M03a.1 evidence](../verification-identity.md) and M03a.2 implementation |
| Projects/tasks / PR-02 | Implemented | Tenant-isolated projects, atomic sequential task keys, canonical status categories (backlog, unstarted, started, completed, cancelled), priorities, due dates, bounded pagination, concealed 404 isolation, Svelte Projects & Tasks views | [M04a spec](../features/projects-and-tasks.md), tests/projects_and_tasks.rs, web/src/projects/ |
| Task collaboration & detail / PR-02, PR-07 | Implemented | Labels, project member assignees, hierarchical subtasks, threaded discussion comments, activity history timeline, multi-dimensional filtering, and TaskDetailDrawer responsive panel | [M04b spec](../features/task-details-and-collaboration.md), tests/task_collaboration.rs, web/src/projects/TaskDetailDrawer.svelte |
| Custom statuses / PR-03 | Implemented | Custom statuses per project with immutable canonical category mapping (backlog, unstarted, started, completed, cancelled), atomic deletion with task remap, drag/button reordering, hex color validation, and Svelte StatusManagerModal | [M05a spec](../features/custom-statuses.md), `tests/custom_statuses.rs`, `web/src/projects/StatusManagerModal.svelte` |
| Kanban/table/list/saved views / PR-02, PR-04 | Planned | No product views or ordering persistence | Task interfaces, ordering ADR and concurrency tests |
| Milestones/cycles/roadmaps / PR-04 | Planned | Milestones distinct from tasks | Progress/date rules and planning specification |
| Timers/manual entries/timesheets / PR-04 | Planned | No time-entry tables or browser timers | Timer overlap/idempotency/DST specification |
| Reports/rates/budgets/capacity / PR-04 | Planned | No aggregates or financial behavior | Approved formulas and reliable underlying records |
| Markdown/connected knowledge / PR-05 | Planned | No editor, renderer, search or graph | Ownership, portability, sanitization and link-identity rules |
| Structured collections / PR-05 | Planned | Not a generic database engine | Real use cases, bounded properties and typed constraints |
| Integrations/notifications/automation / PR-06 | Planned | No credentials, events or workers | Stable authorized operations and durable execution design |
| Performance budgets / PR-09 | Planned | Methodology specified, no measured budgets | Representative workloads and baseline run |
| Android/Termux, other server OSs | Planned evaluation | No compatibility promise | Toolchain, PostgreSQL, signals/storage/recovery testing |

Suites `tests/config.rs`, `tests/http.rs`, and `tests/database.rs` establish the
foundation only; executed counts/results belong in the linked verification
record. M02 additionally tests the public shell's contrast/accessibility subset.
No current tests establish login, authorization, tenant isolation, product
throughput or financial correctness; shell checks are not full WCAG certification.
