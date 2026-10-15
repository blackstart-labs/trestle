# Architecture overview

Status: foundation implemented; product topology proposed within the approved
Rust/PostgreSQL modular-monolith direction. [Current modules](../architecture.md)
remain authoritative for executable behavior.

```text
Browser ── same-origin HTTPS ── operator reverse proxy
                                      │
                               Rust application
                         HTTP → application interfaces
                                domain modules
                             SQLx / explicit SQL
                                      │
                                 PostgreSQL
```

The frontend will be built to static assets; Node is a development/build tool,
not a required production server. Direct binary and Docker/Podman deployment
share the same configuration and explicit migration entry point. No separate
worker, broker, Redis, object-storage service or search cluster is required by
the initial architecture.

## Cross-cutting decisions

| Concern | Direction and trade-off |
| --- | --- |
| Startup | Typed environment validation; lazy bounded pool; separate migrate command; live does not imply ready or schema-compatible |
| Frontend | Svelte/Vite/TypeScript with static delivery; public indexable pages may use prerendering later, not private data SSR by default |
| HTTP | Same-origin versioned JSON product interfaces, fixed public errors, bounded pagination, explicit conflict behavior |
| Identity | Revocable server-side sessions; enrollment/recovery defaults must pass the identity decision gate |
| Authorization | Workspace/project operation matrix, checked on the server; database-enforced same-tenant relations |
| Jobs/events | Add durable PostgreSQL execution/outbox records when external effects arrive; no untracked fire-and-forget business jobs |
| Search | Permission-aware PostgreSQL indexing/querying first; separate search infrastructure only after measured need |
| Attachments | Later filesystem-backed storage outside web root behind authorized download interfaces; no raw public path serving |
| Observability | Structured redacted logs, current health/readiness; correlation and operational metrics added for demonstrated workflows |
| Upgrades | Forward migrations, version compatibility policy, restored-data drill before release; no automatic downgrade guarantee |
| Performance | Bounded queries/payloads and baseline-driven budgets; cache only with explicit invalidation and evidence |
| Testing | Domain/interface tests, real PostgreSQL, negative authorization and concurrency cases, then browser journeys |

Document bodies, attachments, identity secrets and integration credentials have
different exposure/retention needs. Avoid a universal generic resource table or
property engine that obscures ownership. See [module interfaces](module-boundaries.md),
[data flow](data-flow.md), and [database direction](../database/database-design.md).

Core ADRs: [monolith/PostgreSQL](../adr/0001-modular-monolith-and-postgresql.md),
[frontend/rendering](../adr/0002-frontend-and-rendering.md),
[sessions/tenancy](../adr/0003-sessions-and-tenancy.md).
