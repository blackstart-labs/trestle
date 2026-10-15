# Capability inventory and reference extraction

Status: planning inventory. Authoritative implementation status lives in the
[feature matrix](feature-matrix.md); terminology lives in [CONTEXT](../../CONTEXT.md).

## Sources actually read

Local ignored files `audit-reports/Plane.txt`, `audit-reports/Leantime.txt`, and
`audit-reports/Kimai.txt` were available and read. They describe other projects
and propose Rust rewrites; they are not original Trestle code or independently
validated upstream audits. They are not committed, so this extraction records
the useful concepts without making future readers depend on local files.

| Source | Retain as Trestle capability candidates | Do not adopt blindly |
| --- | --- | --- |
| Plane | Workspace/project memberships; project-scoped task keys; semantic states; subtasks/labels/assignees; cycles; planning groups; saved views; triage | Complete schema, float ordering, missing composite tenant integrity, claimed lock-free counters or latency |
| Leantime | Planning and access scopes; custom workflow labels; swimlanes; dense tables/inline editing; milestones and progress; capacity and integrations | Numeric global-role hierarchy, milestone/task polymorphism, unbounded board reads, unauthenticated mutation examples |
| Kimai | Customer/project/activity distinctions; time modes, breaks and rounding; applied rates; quick entry; approvals, budgets, exports, lockdown periods | Non-unique active-timer index, update/insert concurrency assumptions, floating-point money, destructive historical-data cascades |

The reports' RAM, bundle, throughput, microsecond and rewrite-time estimates
are hypotheses and are not Trestle targets or measurements. PostgreSQL-specific
schemas do not establish SQLite compatibility. Reference code errors must not
be imported as implementation shortcuts.

## Domain inventory

| Domain | Required/core capabilities | Later extensions | Specification gate |
| --- | --- | --- | --- |
| Identity/workspaces | Local identity, sessions, workspace/project membership, settings, audit | Invitations, recovery, guests, external identity | Enrollment/recovery/owner-transfer threat model |
| Projects | Identifier, description, visibility, members, archive, linked work | Customers, estimates, budgets, integrations | Identifier and archival scope |
| Tasks/workflows | Title, Markdown, type, category-backed status, priority, creator, assignee, labels, history | Subtasks, dependencies, custom fields, attachments | Same-project invariants, transition/remap rules |
| Views | Paginated list; board with non-drag moves; filter/sort | Dense table, bulk actions, saved/grouped/calendar views | Shared queries, conflict and ordering strategy |
| Planning | Milestones and explicit progress formula | Cycles, roadmap/timeline, dependency scheduling | Eligibility and date/time semantics |
| Time | Timers, manual entries, task/project association, timesheet totals | Activities, duration/punch modes, breaks, approvals, quick entry, lockdown | Single-running-timer scope, retries, overlap policy, DST |
| Reporting/budgets | Authorized time/task aggregates and exports | Capacity, rate resolution, recurring budgets, invoices | Formulas, exact money, currency, rounding and historical rates |
| Knowledge | Markdown documents, project links, organization, history/search | Wikilinks/backlinks, tags, import/export, broken-link handling, optional graph | Link identity, sanitization, portable metadata |
| Structured collections | Separately identified records and typed properties | Relations, templates, computed properties, table/board/calendar | Proven use case and property/query limits |
| Integrations | Scoped credentials, external task/PR links, delivery records | GitHub/GitLab, signed webhooks, calendar feeds, imports | SSRF, idempotency, versioned events |
| Automation | Explicit triggers/conditions/actions, actor, run records | Schedules, reminders, bounded retries, notifications | Permission recheck, loop prevention, execution limits |
| Search/notifications | Permission-scoped PostgreSQL search, useful in-app signals | Cross-domain search, mentions and channels | No revoked/private-resource leakage |

This inventory is not a claim that every candidate has a complete feature
specification. The roadmap schedules specification work before implementation.
