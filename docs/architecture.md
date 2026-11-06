# Backend foundation

Status: initial architecture approved; product domains are not implemented.

## Decisions

- Start with one Go module and one HTTP process, not microservices
  ([ADR-0005](adr/0005-go-backend.md) replaced the original Rust backend).
- Use the standard library `net/http` (ServeMux), `encoding/json`, pgx with
  PostgreSQL only, and structured `log/slog` JSON logging.
- The Go version is set by the `go` line in `go.mod`; commit `go.sum`.
- Keep migrations explicit (`trestle migrate`), separate from HTTP startup.
  This permits a future migration role distinct from the runtime database role.
- Serve the Svelte/Vite/TypeScript foundation from the same origin. `go:embed`
  in `main.go` embeds the production output; Node/npm are build/test tools only.
- Future tenancy is workspace-based. Future browser authentication uses
  revocable server-side sessions; no authentication library is selected yet.

## Current module responsibilities

| Module | Interface and responsibility |
| --- | --- |
| `internal/config` | Parse known environment settings, validate bounds, keep credentials out of diagnostics. |
| `internal/database` | Construct a bounded lazy pgx pool and execute embedded migrations (`migrations/`) explicitly, recorded in `_sqlx_migrations`. |
| `internal/identity` | Users, passwords (Argon2id), sessions, workspaces and memberships. |
| `internal/projects` | Projects, tasks, statuses and task collaboration. |
| `internal/httpapi` | Construct the `net/http` router and middleware; translate health checks, API calls and errors into stable JSON responses; mount embedded assets on exact GET/HEAD routes with no disk reads or SPA fallback. |
| `main.go` | Embed `web/dist`, initialize diagnostics, choose serve/migrate/operator subcommand, bind, handle shutdown, close the pool. |

Configuration parsing, HTTP responses, and migration execution are the agreed
test seams. Tests use these public interfaces, including real PostgreSQL for
migrations and successful readiness checks.
M01b additionally tests visible browser readiness interactions and compiled asset
delivery through HTTP and Chromium; see its feature specification.

## Future module design

Introduce identity, workspaces, projects, tasks/workflows, time tracking, and
documents as vertical slices. Application interfaces own authorization,
validation, transactions, and domain operations. HTTP handlers and future UI
modules do not own business rules. Persistence stays near its owning module.
Do not introduce generic repository traits or a package per domain speculatively.

Workspace is the tenant, project belongs to a workspace, and task is the
canonical work item. Configurable status names map to stable categories.
Use composite constraints where necessary to keep task/project/workspace and
workflow relationships consistent. Billing customers are not tenant identities.

## Resource model

The HTTP process does not require a database connection to start. Readiness
performs one bounded `SELECT 1`; liveness performs no database work. There is no
background polling, caching, worker, broker, or automatic schema modification.
Pool minimum is zero; maximum and acquisition/readiness timeout are bounded.
Statements and lock waits have server-side timeouts. HTTP request diagnostics
omit raw URLs, headers, bodies, and database errors.

There are no measured latency, memory, throughput, or cross-product performance
claims. Set budgets after measuring representative workloads, including total
PostgreSQL and application resources. SQLite and Android/Termux are unsupported
until explicitly tested; Android cannot guarantee background service uptime.

## Dependency review policy

Direct Go dependencies are pgx (MIT), google/uuid (BSD-3-Clause),
golang.org/x/crypto and golang.org/x/term (BSD-3-Clause); every module in the
shipped binary is MIT or BSD-3-Clause.
`govulncheck ./...` reports known vulnerabilities in reachable code of the
resolved module graph; `go mod verify` checks downloaded modules against
`go.sum`. The project's existing `LICENSE` is manually preserved.
This is not a security guarantee or a substitute for reviewing behavior and
future advisories.

Review lockfile changes deliberately. Update Go, dependencies, pinned GitHub
Action SHAs and container digests in focused changes, run all documented checks,
and review upstream release notes and advisories. Never automatically publish.
No license exceptions or advisory ignores are permitted without a documented
reason and approval. Preserve the existing project
license; do not infer a new SPDX grant from the generic license's example notices.
