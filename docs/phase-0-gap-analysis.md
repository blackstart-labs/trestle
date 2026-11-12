# Phase 0: verified inventory and implementation plan

Date: 2026-11-11. Baseline: `a373909`, the PR #52 conflict-resolution commit, including upstream `main` at `2dd1f03`. Planning branch: `docs/phase-0-product-plan`.

Input: the maintainer's `trestle.txt` product brief. This is an implementation proposal, not a statement that the requested product already exists. The published `0.2.1` Docker image was built from `a7e1c24`, before the upstream merge; it is not the exact baseline reviewed here.

This is a frozen inventory of that baseline. Subsequent implementation adds
[Workspace diagrams and whiteboards](boards.md); consult that document for the
new visual tools rather than treating this snapshot as the current feature list.

## What this repo does

Trestle is a working, self-hosted Workspace for Projects, Tasks, milestones, discussions and basic time logs. A Go HTTP server embeds a Svelte 5 frontend and uses PostgreSQL. It is not yet the six-area platform described in the brief.

Assumed load: small and medium-sized teams on one instance, with concurrent browser sessions and unreliable mobile connections. There is no measured capacity claim. Preserve the modular monolith, existing data, task identities and public behavior; expand with tested vertical slices, not a replacement application.

## 1. Repository map and evidence

| Area | Current responsibility | Evidence |
| --- | --- | --- |
| Entry point and operator recovery | Serve, migrate, bootstrap, provision, reset passwords, disable/enable users | `main.go`, `main_test.go` |
| Configuration and database | Validated environment, bounded pool, forward SQL migrations with advisory locking and checksums | `internal/config/`, `internal/database/`, `migrations/` |
| Identity | Password hashes, database sessions, Workspace membership/ownership, profile images | `internal/identity/`, `internal/httpapi/session.go`, `workspaces.go` |
| Work domain | Projects, canonical Tasks, statuses, labels, assignees, parent links, comments, activity | `internal/projects/store.go`, `internal/httpapi/projects.go` |
| Planning and time | Milestone CRUD/rollups; task-scoped decimal-hour logs | `internal/projects/milestones.go`, `time_tracking.go`, corresponding HTTP handlers |
| Notifications | Three fixed providers, stored connection settings, latest delivery result | `internal/integrations/`, `internal/projects/integrations.go`, `internal/httpapi/integrations.go` |
| Application | Shell, auth, Settings, dashboard, Project/Task/milestone views and shared UI | `web/src/` |
| Public documentation | Separate Vite/Svelte documentation site, not an in-app document system | `docs-site/` |
| Delivery | Non-root image, embedded assets, CI, local runner scripts | `Dockerfile`, `compose.yaml`, `.github/workflows/ci.yml`, `run.*`, `tests/` |

The complete registered runtime API is in `internal/httpapi/router.go:61–136`. The 13 SQL migrations define users, sessions, Workspaces, memberships, security audit events, Projects, Tasks, labels, label attachments, assignees, comments, task activities, statuses, provider settings, milestones and time logs. They do not define the future modules below.

## 2. Feature inventory against the brief

**Implemented** means a concrete existing workflow, not completion of every requested capability in that area. **Partial** means usable pieces exist but important requirements are missing. **Missing** means no matching application schema, registered endpoints and complete user workflow were found. A documentation page is not evidence of implementation.

| Brief area | State | Verified behavior and remaining gap |
| --- | --- | --- |
| Workspace lifecycle | Implemented, limited scope | Create, choose, rename, change slug/icon; backend owner/admin management. No Workspace delete/archive API. `workspaces.go`, `WorkspaceChooser.svelte`, `WorkspaceSettings.svelte`. |
| Membership and ownership | Partial | Add existing operator-provisioned users, change roles, remove/leave, transfer ownership with password confirmation. No invitation tokens, email enrollment, teams or groups. `identity/store.go`, `WorkspaceMembers.svelte`. |
| Project management | Partial | Create/edit name, slug, description/icon; immutable key; archive and export. No dedicated owner, Project membership/visibility, dates, priority, duplication or templates. `CreateProjectParams`, `UpdateProjectParams`, `ProjectsView.svelte`. |
| Goals, objectives, key results | Missing | No persistence, endpoints or runtime screen. |
| Core Tasks | Implemented, limited scope | UUID and sequential human key, title, Markdown description, Status/category, priority, due date, creator, multiple assignees, labels, milestone and parent. Creation metadata is transactional. `CreateTask`, `UpdateTask`, `TasksView.svelte`, `TaskDetailDrawer.svelte`. |
| Advanced Tasks | Missing/partial | Direct subtasks and parent links exist; a complete nested-tree workflow does not. No start date, estimate, archive/restore, duplicate, bulk operations, dependencies, attachments, custom fields, recurrence, templates or watchers. |
| Workflow | Implemented | Custom statuses, canonical categories, ordering, inline changes and remapping when deleting a used Status. Status management is owner/admin only. |
| Milestones and roadmap | Partial | Milestone CRUD, start/due dates, colors, task assignment, cards and timeline. Not a task-dependency Gantt or Workspace/goal roadmap. Cancellation math needs correction (finding 3). |
| Cycles and modules/epics | Missing | No cycle/module persistence, assignments, burndown or historical snapshots. |
| Views and filters | Partial | Task List/Board, milestone cards/timeline; category/custom Status, label, assignee and milestone filters; remembered browser layout. No saved/shared query model, priority/date/search filters, configurable sort/group/columns, Task calendar or full Gantt. |
| Personal and Workspace dashboards | Partial | Real task-derived summaries, due dates and assigned work; bounded sample of 12 Projects × 100 Tasks. “My work” is a sampled dashboard section, not an exhaustive cross-Project Task view. `Dashboard.svelte`. |
| Collaboration | Partial | Flat comments, assignees, profile/avatar display and Task activity. No comment replies, mentions, subscriptions, announcements, presence or in-app notification center. The comment schema has no parent-comment field despite its migration comment saying “threaded.” |
| Manual time entries | Partial | Task-scoped hours, date, note, creator, list/delete and aggregate hours. No edit endpoint, interval timestamps, approvals, activities, customers or billing fields. `task_time_logs`, `time_tracking.go`. |
| Running timer | Partial; not server-authoritative | Start/display/stop/discard and notes persist in browser localStorage. One instance's save guard is not cross-tab, cross-device or network-retry idempotency. No persisted running timer, pause or resume. `lib/timer.svelte.ts`. |
| Timesheets, customers, activities, billing | Missing | No corresponding schema or runtime workflows. Task activity history is not a billable Activity classification. |
| Documents | Missing | Task descriptions have a safe Write/Preview editor. No Document entity, ownership, page tree, revisions, history restoration, favorites or document permissions. |
| Markdown/block editor | Partial | Task Markdown supports formatting, lists, links, code and tables. Raw HTML is escaped and image Markdown becomes text; no block storage/editor, file/image attachments, syntax-highlighting workflow, autosave recovery or collaborative editing. `ui/MarkdownEditor.svelte`, `lib/markdown.ts`. |
| Connected knowledge and graph | Missing | No document links, backlinks, wiki links, graph or broken-link detection. Do not confuse a source-code analysis tool with a shipped product feature. |
| Structured collections | Missing | No schema/properties/records, formulas, relations, rollups or collection views. |
| Automation | Missing | No rules, durable event delivery, conditions/actions, execution history or worker. Existing task notifications are hard-coded behavior, not an automation engine. |
| GitHub integration | Missing | No GitHub App/OAuth connection, repository model, inbound signature verification or PR/commit/branch association. GitHub CI for this repository is not an in-product integration. |
| Public integration API and webhooks | Missing/partial | The browser's session/CSRF-protected JSON API exists. No scoped personal/integration tokens, generic outbound webhook subscriptions or inbound integration event API were found. Provider notifications are not a general webhook platform. |
| Telegram/Slack/Discord | Partial | Project connection save/replace/test/disconnect, Topic ID, task create/update/delete delivery and last result. No durable retry, per-event settings, preferences or append-only attempt history. |
| Unified search | Partial | Command palette searches available navigation, Projects and theme commands. No backend permission-aware search across Tasks, comments, users or future entities. `App.svelte`, `CommandPalette.svelte`. |
| Analytics/reports | Partial/missing | Real bounded dashboard aggregates and JSON exports exist. No comprehensive server-side reporting, historical trends, resource workload, CSV/PDF reports or billing summaries. |
| Design/accessibility | Implemented foundation | Semantic tokens, Mona Sans, light/dark/system, five accent presets, shared controls, keyboard paths, phone layouts and automated contrast/axe/reflow coverage. New modules still need their own complete state and permission coverage. |
| Security and operations | Partial foundation | Cookie auth, CSRF, Argon2id, membership checks, audit primitives, explicit migrations, Docker/CI and PostgreSQL backup guidance. Not a completed enterprise authorization, secret-management, attachment or integration-token system. |

## 3. Concrete findings before expansion

These are source-verified cases, not a claim that production incidents were observed. No implementation fixes are included in this Phase 0 branch. Priorities reflect the requested reliable shared-team product; proposed policies are labeled separately from proven behavior.

### Must fix before relying on expanded workflows

1. **A timer can create duplicate logs after a lost response** (`web/src/lib/timer.svelte.ts:138–190`, `internal/projects/time_tracking.go:32–83`).
   - **What this is:** The browser converts elapsed local time into a new time-log POST.
   - **Problem:** If the server commits but the response is lost, the timer remains active; retry inserts a second log. Another browser tab/device has an independent guard.
   - **Fix:** Persist user-owned timer state and make stopping it one transaction with an idempotent result. Do not silently import untrusted local timers into the server.
   - **If skipped:** Hours can be lost, duplicated or attributed using the wrong local state. This blocks credible timesheets and billing.

2. **Writes can succeed while their activity write fails** (`internal/projects/store.go:698–706,753–815,847–868`).
   - **What this is:** Several helpers perform a data mutation and then insert history through a pool; creation-time calls inside a transaction are already safer.
   - **Problem:** A standalone comment/assignment can commit before its activity insert fails. The API returns an error although the user's mutation already happened; retrying a comment duplicates it.
   - **Fix:** Wrap each logical mutation and its required audit/activity records in the same transaction, reusing the existing Querier/transaction pattern. Add a failure-injection regression through the API.
   - **If skipped:** History becomes unreliable and error retries change data twice. Automation cannot safely depend on these events.

3. **Milestone completion counts cancelled work in the denominator** (`internal/projects/milestones.go:59–66`, `web/src/projects/MilestoneTimeline.svelte:34–36`, `TasksView.svelte:getMilestonePercent`).
   - **What this is:** Milestone views compute completed Tasks divided by total Tasks.
   - **Problem:** With one completed and one cancelled Task, a milestone shows 50%; the project-wide contract requires 100%: completed ÷ (all − cancelled).
   - **Fix:** Return a cancelled or counted total and use one shared completion definition across cards, timeline, dashboard and reports. Test an all-cancelled milestone too.
   - **If skipped:** Different screens disagree about whether the same work is finished.

4. **JSON exports are not complete recovery backups** (`internal/httpapi/export.go:15–32,35–72`).
   - **What this is:** Exports now include all Task pages and logged hours in a stable snapshot.
   - **Problem:** A Workspace with comments or uploaded icons loses that content from the JSON: comments, activity and image bytes are absent, and there is no import/restore endpoint.
   - **Fix:** Document PostgreSQL dump/restore as the recovery path now. Define a versioned portable-export format and round-trip tests before claiming complete JSON portability; bound memory and replace the per-Task log query when export size warrants it.
   - **If skipped:** Users may mistake a successful JSON download for a recoverable full backup.

### Should fix before the next feature release

5. **A failed Project read becomes a zero on the dashboard** (`web/src/projects/Dashboard.svelte:229–245`).
   - **What this is:** The dashboard requests sampled Tasks separately for each Project.
   - **Problem:** A Project returning HTTP 503 becomes an empty array and total zero. The screen can say nothing is assigned without reporting the missing data.
   - **Fix:** Track failed Projects, show partial-data feedback and retry, and never present unknown counts as zero. Sampling itself is a disclosed design decision, not a bug.
   - **If skipped:** Users can miss assigned work during transient failures.

6. **Professional time ownership is not enforced** (`internal/httpapi/time_tracking.go:75–91`, `internal/projects/time_tracking.go:112–122`).
   - **What this is:** A non-guest Workspace member can delete a Task's time log; the store accepts but does not use actorID.
   - **Problem:** Member B can delete Member A's logged hours. There is no record-level owner/admin restriction or deletion activity.
   - **Fix:** Adopt an explicit own-entry-or-manager policy before approvals/billing, enforce it in the backend and test both same-Workspace denial and cross-Workspace isolation. This is a proposed policy change, not a claim that an existing documented time-entry policy was implemented.
   - **If skipped:** Billable history can be altered by unrelated contributors.

7. **Parser and database disagree about usernames** (`internal/identity/primitives.go:20–49`, `migrations/202611010002_identity.sql:20–25`).
   - **What this is:** The parser accepts `@` and `+`, while the database username constraint allows only letters, digits, dot, underscore and hyphen.
   - **Problem:** `person+ops` passes parser validation but fails user insertion. There is no separate email identity/verification model.
   - **Fix:** Choose the intended username contract, align parser/schema with a forward migration if expanding it, and test provisioning plus login. Do not advertise email enrollment based on parser characters alone.
   - **If skipped:** Invitation or enrollment flows built on this assumption fail for accepted input.

8. **The data model does not provide Project-level access grants** (`internal/httpapi/projects.go:328–350`, `migrations/202611010002_identity.sql:86–101`).
   - **What this is:** Authorization is Workspace membership plus manager/guest rules.
   - **Problem:** Adding a guest to a Workspace gives read access to its Projects; there is no Project membership table to support “granted Projects only.” This is a capability gap and documentation mismatch, not proof of cross-Workspace read access.
   - **Fix:** Document current Workspace-wide visibility. Introduce explicit Project visibility and memberships before private Documents, customers, restricted Projects or shared views depend on them; preserve existing Projects' visibility by default.
   - **If skipped:** New features may rely on a permission boundary that does not exist.

9. **Provider credentials are write-only, not encrypted by the application** (`migrations/202611070006_project_integrations.sql:3–12`, `internal/projects/integrations.go:21–38`).
   - **What this is:** Secrets are hidden from API serialization but stored directly in a TEXT column.
   - **Problem:** A database dump contains usable provider tokens. There is no application key, encryption version or rotation path.
   - **Fix:** Correct the storage claim now; decide the threat model and operator-managed key/rotation policy before storing GitHub installation/private-key credentials. Protect database backups regardless.
   - **If skipped:** Operators may distribute unprotected backups under a false encryption assumption.

10. **Tenant consistency is only partly enforced by database relationships** (`migrations/202611020003_projects_and_tasks.sql:27–45`, `202611080007_project_milestones.sql:3–18`; contrast the composite integration FK).
    - **What this is:** Many rows store both Workspace and Project IDs but reference them separately.
    - **Problem:** Direct SQL can insert a Task with Workspace A and a Project belonging to Workspace B; both foreign keys exist independently. No HTTP exploit is claimed here.
    - **Fix:** Preflight inconsistent existing rows, then add scoped unique keys/composite FKs and validate them in forward migrations before new import/worker paths. Never delete mismatched data automatically.
    - **If skipped:** New non-HTTP writers can corrupt tenant relationships even if current handlers check scope correctly.

## 4. Architecture to preserve and extend

- Keep Go/net/http, pgx, PostgreSQL and embedded Svelte. Continue the existing HTTP → domain store → database boundaries. No Redis, Kafka, Elasticsearch, microservice split or replacement ORM is justified by this inventory.
- Keep a Task as the canonical work item. “Issue,” subtask, cycle membership, module membership, time and document links must refer to that identity, not duplicate it.
- Keep tenant authorization in backend queries and mutations. Factor the existing policy only when the first real Project-visibility consumer needs it; do not build a generic policy engine now.
- Reuse existing tokens, Avatar, Modal, Select, DatePicker, MultiPicker, MarkdownEditor, Feedback, Toast and motion helpers. Add destinations to the sidebar and command palette only when their end-to-end workflow exists.
- Preserve portable Markdown as Document source. Implement a reliable text editor and revision model before blocks or real-time collaboration. `docs-site/` remains deployment documentation, not tenant-owned content.
- Use PostgreSQL full-text search with explicit authorized Workspace/Project scope first. Rank bounded queries and add indexes only after inspecting plans and representative data. Do not search before checking visibility.
- When delivery reliability becomes a requirement, write an outbox record in the same transaction as the mutation and consume it with a bounded PostgreSQL-backed worker. A durable outbox is not an excuse to introduce a general event framework without a concrete consumer.
- Use typed domain columns for stable properties; JSONB only for intentionally flexible rule/property definitions. Do not use arbitrary SQL, JavaScript evaluation or unbounded formulas supplied by users.
- Keep history distinct from the domain noun **Activity**: task activity events describe changes; a time-entry Activity classifies the work performed.

## 5. Forward migration plan (proposals, not migrations shipped)

Never edit migrations 001–013, rename away legacy data or silently purge records. Each feature PR must provide a reviewed migration, index plan, old-data/backfill handling and rollback/recovery notes.

| Slice | Minimal schema direction | Compatibility and integrity gate |
| --- | --- | --- |
| Foundation | Scoped unique constraints/FKs; existing write/audit transactions; only needed aggregate indexes | Detect inconsistent data first; validate constraints without deleting it. Preserve UUIDs and human keys. |
| Access and invitations | Project visibility + Project memberships; invitations only once enrollment semantics are agreed | Existing Projects remain Workspace-visible; single-use hashed invitation tokens, expiry/revocation and atomic acceptance. No implied SMTP dependency. |
| Tasks/planning | Nullable task start/estimate/archive fields; dependency edges, saved views, cycles and modules introduced separately | Same-scope references, cycle prevention, bounded bulk changes; old clients retain current defaults. Version saved-filter schemas rather than storing raw SQL. |
| Strategic planning | Workspace Goals/Objectives/Key Results and explicit Project links | Do not use a second tenant entity. Separate measured results from automatic task-derived progress. |
| Reliable time | User-owned running timer and interval/elapsed-duration fields; idempotent stop result | Add to or compatibly extend `task_time_logs`; preserve existing decimal-hour logs. Never invent precise timestamps for historical hours-only entries. Audit any backfill and identify legacy precision. |
| Professional time | Customers, work Activities, rates and later timesheets/approvals | Integer duration and exact money representation; historical rate/currency snapshots, explicit rounding/timezone/overnight rules before billing. |
| Knowledge | Documents, revision snapshots and scoped links; optional parent and Project association | Revision conflict handling, safe parent trees and permission-aware links/backlinks. Images/attachments require a separate secure storage design. |
| Collections | Collection/property/record rows, then explicit relations and rollups | Validate typed values and relation scope; constrain computation depth and query cost. Formula work is a later bounded slice. |
| GitHub | Installations/repositories, external-object links and deduplicated webhook receipts | Scoped installation permissions, verified signatures, delivery-ID uniqueness, replay-safe processing and disconnect behavior. |
| Automation | Outbox/delivery attempts, then rule definitions and execution records | Transactional event creation, unique execution keys, retries with bounds, loop prevention, actor authorization rechecked at execution time. |

No exact future migration numbers are reserved; allocate the next forward number when its feature PR is ready.

## 6. API and interaction contracts for the first slices

All paths below are **proposed**, not registered today. Continue `/api/v1/workspaces/{workspace_id}` scoping, session/CSRF protections, JSON errors and concealed cross-tenant 404s. Add idempotency/versioning only where the workflow needs it.

- **Server timer:** `GET/POST /api/v1/workspaces/{ws}/me/running-timer`, with `POST .../pause`, `.../resume`, `.../stop` and `DELETE` for confirmed discard. Enforce one active timer per user across Workspaces by a database invariant. Starting another returns a conflict with a deliberate switch flow. Repeated stop keys return the same time entry, including after a lost response. The server owns elapsed time; UI tick rendering does not write each second.
- **Saved views:** `GET/POST .../views`, `GET/PATCH/DELETE .../views/{id}`. Store an allowlisted typed filter/sort/group definition, owner and sharing scope. Recheck access when executing a view; sharing never grants access to underlying Tasks.
- **Task dependencies:** `GET/POST .../projects/{project}/tasks/{task}/dependencies` plus explicit removal. Model direction once, reject self/cyclic edges and cross-scope references, and offer keyboard/touch parity instead of drag-only graph editing.
- **Documents:** `GET/POST .../documents`, `GET/PATCH .../documents/{id}` with a revision precondition, explicit archive/restore, and separate revision/link reads. A stale save must preserve the user's draft and offer conflict recovery, not overwrite another author.
- **Search:** `GET .../search?q=...&types=...&cursor=...` after a common visibility policy exists. Limit query length, result count and snippets; enforce permissions before results/counts are returned. Scope saved search history to the user.

Every slice includes loading, empty, filtered-empty, retryable failure, success, saving/disabled, permission-limited and stale/conflict states. Errors are persistent inline messages, not disappearing toasts. Long text remains explicit-save with draft protection until autosave is demonstrably safe.

## 7. Delivery order and release gates

Phases are ordered deliverables, not a commitment to complete the entire brief in one release. Within each phase, ship one complete workflow per PR: migration → domain/authorization → API → UI → tests → documentation. No empty navigation destinations or placeholder controls.

| Phase | Small PRs, in dependency order | Exit gate |
| --- | --- | --- |
| 0 — audit (this work) | Inventory, source-backed findings, documentation corrections and this plan | Maintainer can distinguish working, partial and missing capabilities; no application rewrite. |
| Foundation gate | Fix findings 1–10 in separate reviewed changes; resolve time-entry policy and Project visibility; add a real browser-to-PostgreSQL core journey | No known silent data loss/duplicate log path; scoped writes and trustworthy audit events; a backup restoration drill. The durable timer is pulled forward because it already affects existing work. |
| 1 — core planning | Task dates/estimates/archive/duplicate → saved filters + complete My Tasks → dependencies → cycles → modules → calendar → dependency-aware Gantt → recurrence/templates/bulk actions; add Project visibility/invitations and simple strategic planning as separate slices | Each view uses the same authorized Tasks and dates; cycles have explicit carryover and report rules; dependencies cannot form cycles; recurring jobs are idempotent. |
| 2 — professional time | Complete pause/resume/server synchronization → editable authorized entries → daily/weekly/monthly timesheets → Customers/Activities → approvals → exact rates/billable summaries → invoice/CSV/PDF slices | Concurrent-device and lost-response tests pass; timezone/DST and rate-history tests pass. No billing based on sampled dashboards or guessed legacy intervals. |
| 3 — knowledge | Portable Markdown Documents → nested pages and safe attachments → Task/Project links → backlinks and permission-aware search → revisions/restoration → favorites/templates; block editor only after this foundation | Two authors cannot silently overwrite revisions; links cannot leak private titles/snippets; Markdown exports round-trip. Graph is supplementary and follows link integrity, not precedes it. |
| 4 — collections | Typed properties/records + table → filter/sort/group → board/calendar/gallery → relations → bounded rollups → constrained formulas | Property changes preserve/validate data; all views enforce the same schema and permissions; expensive computations have limits. |
| 5 — developer integrations | GitHub App installation/disconnect → repository selection → signed, deduplicated webhook processing → scoped task-key linking → PR/review/CI status | Minimal installation permissions, safe token lifecycle and replay tests; disconnect stops future access and explains retained history. |
| 6 — automation | Reuse durable event delivery → one create/status trigger and one bounded action → conditions → execution history/retry UI → controlled cross-module events | Retries do not repeat effects; loops and run budgets are enforced; disabled users/removed memberships cannot keep executing privileged actions. |
| 7 — analytics and polish | Exact server aggregates → historical snapshots/trends → role-specific reports and exports → performance/accessibility/operations work | Unsampled totals agree across screens; historical reports do not reconstruct past state from today's rows. Quality gates remain mandatory in every earlier phase too. |

Dependencies: tenant policy + atomic writes → all new modules; reliable time → timesheets → rates → invoices; document revisions + links → backlinks → graph; delivery persistence → webhook retries and automation. PostgreSQL search can expand as each authorized entity becomes real; it does not need to wait until the end.

### First three implementation PRs recommended

1. **Atomic time-entry stop and ownership.** Add server persistence, unique active-user constraint, idempotent stop, own-entry/manager mutation checks and preserved browser draft recovery. Acceptance: simultaneous starts/stops across two sessions and a lost-response retry produce one authorized entry; failure leaves recoverable state. Do not migrate historical hours into fabricated intervals.
2. **Atomic mutations and truthful progress.** Fix write+history transaction boundaries, milestone cancellation math and partial-dashboard error feedback, with API failure injection and shared UI math tests. Acceptance: no committed comment after a reported failed transaction; one done + one cancelled is 100%; a failed Project query is “unknown,” never zero.
3. **Tenant/data-contract alignment.** Reconcile username validation, scoped relationships and Project visibility policy; document JSON export limits and run a PostgreSQL restore drill. Acceptance: invalid scope relationships fail at the database boundary; existing users/Projects retain access and identity; invitations are not advertised before enrollment works.

These are deliberately narrower than a complete timesheet, authorization framework or generic event bus. Detailed endpoint/migration review belongs in each PR, not speculative runtime scaffolding in Phase 0.

## 8. Documentation corrections and decisions

This branch corrects the capability/CSP summary in `PRODUCT.md`, fixes ADR links
and flags reference drift in `docs/README.md`. The API, database, operations and
security reference rewrites remain follow-up documentation work; they are not
silently certified by this audit. The following source-backed corrections take
precedence over their outdated implementation claims:

- Time logging is implemented, but a server-authoritative Running timer and professional time system are not.
- Sessions use `trestle-session`, a fixed seven-day absolute lifetime and twelve-hour idle limit; `SESSION_TTL_HOURS` is not read. Production Secure cookies are disabled only by validated loopback development mode, not by automatically detecting TLS termination.
- Many old API examples omit the required Workspace/Project path. Priority filtering and unarchive are not registered capabilities. The reference must match `router.go`.
- The schema uses `workspace_memberships`, `milestones`, `task_label_assignments` and `task_activities`. Task `human_key` is derived, not a stored globally unique column. There is no `users.email` column.
- Pool defaults are five connections and a two-second connection/readiness timeout; authoritative code is `internal/database/database.go`, not nonexistent `db.go`/`migrate.go` files.
- Provider secrets are plaintext in PostgreSQL, even though API reads omit them. Backup encryption/key handling is an operator responsibility today.
- Roles are Workspace-wide; there are no private-Project grants or workspace-delete endpoint. Status management and milestone management have different role requirements.
- `TRESTLE_ORIGIN`, `SESSION_TTL_HOURS` and `LOG_LEVEL` are not active configuration keys. Notification links currently use request/forwarded host information; a future trusted-origin/proxy policy is a real change, not a documented environment knob already in use.

Open product decisions for their first implementation PR: private-Project guest policy, invitation enrollment/delivery, timer switching and overnight allocation, approval authority, rate/currency rounding, attachment quotas/storage, document sharing/revision retention, and secret-key recovery. Default to existing public Workspace behavior and do not block this inventory on those future decisions. Customer and Activity vocabulary comes from `CONTEXT.md`; do not introduce another tenant layer.

## 9. Verification, coverage and limits

For this documentation-only change, `git diff --check` passed and all 19 local
Markdown link targets in the three changed documents were checked and exist.
Runtime tests, builds and visual checks were not rerun: no application, schema,
dependency or UI files changed. The test results below are prior baseline
results, not new test executions for this audit.

Immediately before this docs-only work, the exact baseline passed 82 frontend unit tests, 74 Playwright browser tests, real PostgreSQL backend tests with the race detector, Go vet, design lint, Svelte checks (zero errors/warnings), formatting and production/release builds during PR conflict resolution. This is evidence for existing behavior, not proof that the newly identified gaps are fixed.

Current browser tests run a real Go-served production bundle but use API fixtures and an unavailable database; backend tests separately exercise real isolated PostgreSQL. Add a separate real browser/API/PostgreSQL journey before calling the full enterprise workflow end-to-end tested. Do not replace the existing deterministic accessibility/fixture suite.

For every feature: include member/guest/outsider checks, cross-Workspace reference attempts, transaction rollback, duplicate requests, concurrency where relevant, actual SQL migration/backfill tests, keyboard/touch parity, both themes, 320px/200% reflow, persistent errors and draft recovery. Add multi-browser coverage when claiming cross-browser support. Integration tests use controlled providers; real deliveries require separate safe credentials and cannot be inferred from mocks.

This review maps the whole repository and inspects the main identity, Task, planning, time, export and notification flows. It is not a line-by-line review of every stylesheet, runner, docs-site page or test. Not checked: production database contents, load benchmarks, restored backups, hostile-provider live tests, multi-process rate limits, external customer UX research, or commercial accounting/legal requirements. New findings are source-verified; no new failure-injection implementation was added in Phase 0. No feature dates or performance numbers are invented.

**Verdict:** preserve the current product. First make shared data and timers dependable, then extend the six areas incrementally. Phase 0 is a plan and honest inventory, not completion of the product vision.
