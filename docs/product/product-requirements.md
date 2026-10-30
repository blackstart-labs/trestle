# Product requirements

Status: approved high-level direction; feature contracts are specified
incrementally. See [capability status](feature-matrix.md) for implementation.

## Outcomes and requirement identifiers

| ID | Requirement | Acceptance surface |
| --- | --- | --- |
| PR-01 | One workspace tenant model with server-enforced membership and permissions | Identity/authorization integration tests, negative two-workspace journeys |
| PR-02 | Projects organize a canonical task model shared by all views | Create/edit/transition task through list and board without divergent records |
| PR-03 | Custom statuses carry stable completion semantics | Rename completed status without changing progress meaning; safe deletion/remap |
| PR-04 | Work/time/report data remain reconcilable under retries and concurrency | Numbering, ordering, timers, conflict and fixed-fixture report tests |
| PR-05 | Markdown knowledge is safe, linked and exportable | Rename/link/export round-trip, malicious-content and authorization tests |
| PR-06 | External effects are scoped, bounded and inspectable | Signature, SSRF, retry, idempotency and revoked-permission tests |
| PR-07 | Everyday workflows support keyboard, touch and narrow screens | Accessible, responsive end-to-end acceptance journeys |
| PR-08 | Core deployment uses no proprietary infrastructure | Direct and container deployment, upgrade and restore evidence |
| PR-09 | Resource use and responsiveness are measured, not asserted | Reproducible baselines and approved budgets before performance sign-off |
| PR-10 | Documentation, capability status and tests match shipped behavior | Same-change traceability and release checklist review |

## Delivery scope

First useful release: identity/session lifecycle, isolated workspaces, projects,
tasks, configurable status semantics and a reliable list workflow. Board and
table capabilities follow the same task interfaces; they are not independent
CRUD systems. Time and Markdown follow once ownership rules are stable.

Later scope includes planning, advanced time modes, budgets, approvals, team
capacity, structured collections, connected-knowledge graphs, developer
integrations and bounded automation. Inclusion in the inventory does not make
these a dependency of the first useful release.

## Decision gates

Before identity implementation resolve bootstrap, enrollment/invitation,
recovery, owner transfer, session expiry and permission-change semantics in its
feature spec. Before financial functionality approve currencies, rate precedence,
rounding, tax and audit requirements. Before public sharing approve a separate
public-access model; default workspace content remains private.

Offline/local-first synchronization, native applications, SQLite and Termux
server support require separate evidence and decisions; Markdown portability
and mobile-browser access do not imply any of those capabilities.

The complete domain is not to be materialized in migrations up front. Each
feature introduces only its approved tables, constraints and tests.
