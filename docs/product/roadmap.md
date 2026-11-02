# Dependency-aware roadmap and implementation tracker

Status reviewed 2026-10-15. This file owns work ordering; the
[feature matrix](feature-matrix.md) owns shipped capability status.

| Work ID | Phase | Deliverable / exit condition | Depends on | State |
| --- | --- | --- | --- | --- |
| M00 | 0 | Product source-of-truth docs, architecture ADRs, domain/permission/design direction and measurable test plan | Reference extraction | Core documentation delivered; detailed specs and open security decisions remain explicit |
| M01a | 1 | Tested Rust backend, config, PostgreSQL migration path, diagnostics, health and local delivery | Approved foundation spec | Verified; commit `79478d7` |
| M01b | 1 | Frontend build/type/test/CI foundation with same-origin asset delivery | M00, frontend ADR | Implemented; see frontend verification record |
| M02 | 2 | Semantic tokens, accessible responsive shell, reusable forms/feedback | M01b, design acceptance criteria | Implemented scoped shell; see application-shell specification and verification |
| M03a | 3 | Bootstrap/login/logout/revocation and workspace membership authorization | M01a, M02, identity specification | Security policy approved; implementation not started |
| M03a.1 | 3 | One-time CLI bootstrap plus real browser login/current-user/logout and initial workspace access, with expiry/CSRF/login limits | M03a policy, executable contracts/hash measurements/test seams | Verified; one-time CLI bootstrap, Argon2id auth, session/CSRF tokens, login limits, and Svelte sign-in/workspace chooser implemented |
| M03a.2 | 3 | Operator provisioning/recovery, membership lifecycle and owner transfer/revocation races across workspaces | M03a.1, scoped permission contracts | Verified; operator CLI (adduser, resetpassword, disableuser, enableuser), workspace membership lifecycle, last-owner preservation under concurrency, and owner transfer with reauthentication implemented |
| M03b | 3 | Invitations and any self-service recovery via approved delivery channels | M03a, separate channel/recovery specification | Deferred; no SMTP or public enrollment in M03a |
| M04a | 4 | Authorized projects, canonical tasks and paginated list end to end | M03a, project/task specs | Not started |
| M04b | 4 | Labels/assignees/comments/history/subtasks and filtering | M04a | Not started |
| M05a | 5 | Custom status administration and consistent transitions | M04a, category/remap rules | Not started |
| M05b | 5 | Accessible board ordering, table editing, saved views | M05a, ordering/concurrency ADR | Not started |
| M05c | 5 | Milestones/progress, then cycles and roadmap | M04b, planning formulas | Not started |
| M06a | 6 | Retry-safe timers/manual entries and reconcilable timesheets | M04a, time rules and concurrency tests | Not started |
| M06b | 6 | Permission-aware reports, budgets, capacity and exports | M06a, exact calculation specs | Not started |
| M07a | 7 | Safe Markdown documents, organization, project links and history | M03a, knowledge spec | Not started |
| M07b | 7 | Wikilinks/backlinks/search/export; optional graph | M07a, link-identity rules | Not started |
| M07c | 7 | Structured collections and additional views | Validated use cases, M07a | Deferred specification |
| M08a | 8 | Scoped external interface, developer links, signed hooks and delivery records | M04b, authorized versioned events | Not started |
| M08b | 8 | Notifications, constrained automation and execution history | M08a, durable jobs/loop controls | Not started |
| M09 | 9 | Production release criteria verified on supported profiles | Relevant shipped slices, security/a11y/performance/restore evidence | Ongoing gate, not a future substitute for safe implementation |

Frontend infrastructure and shell do not constitute a usable project product.
The first usable pilot ends at M04a; M05–08 may ship independently as complete
vertical slices. Financial invoicing, full accounting rules, native apps,
offline sync and SQLite are outside the first-release critical path.

## Execution rule

For each work item: write the authoritative spec and acceptance criteria, close
material decisions, identify dependencies, design data/permissions/interactions,
implement one complete slice, test, verify, review, document, then make a focused
conventional commit when authorized. Never push/publish implicitly.

Each milestone report records implementation/docs, database/interface changes,
checks actually executed, measured observations (or absence of measurements),
security limits, commits, unresolved work and the next concrete increment.

## Next concrete milestone

M04a: Authorized projects, canonical tasks and paginated list end to end.
