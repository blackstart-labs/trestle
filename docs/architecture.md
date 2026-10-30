# Backend foundation

Status: initial architecture approved; product domains are not implemented.

## Decisions

- Start with one Rust crate and one HTTP process, not microservices.
- Use Axum/Tokio, Serde, SQLx with PostgreSQL only, and structured tracing.
- Pin Rust in `rust-toolchain.toml`; commit `Cargo.lock`. SQLx 0.9 requires
  Rust 1.94 or newer; the selected toolchain is 1.99.0.
- Keep migrations explicit (`trestle migrate`), separate from HTTP startup.
  This permits a future migration role distinct from the runtime database role.
- Serve the Svelte/Vite/TypeScript foundation from the same origin. `build.rs`
  embeds the production output; Node/npm are build/test tools only.
- Future tenancy is workspace-based. Future browser authentication uses
  revocable server-side sessions; no authentication library is selected yet.

## Current module responsibilities

| Module | Interface and responsibility |
| --- | --- |
| `config` | Parse known environment settings, validate bounds, keep credentials out of diagnostics. |
| `database` | Construct a bounded lazy PostgreSQL pool and execute embedded migrations explicitly. |
| `http` | Construct the router; translate health checks and errors into stable JSON responses. |
| `assets` / `build.rs` | Embed reviewed frontend output and mount exact GET/HEAD routes; no disk reads or SPA fallback. |
| binary | Initialize diagnostics, choose serve/migrate mode, bind, handle shutdown, close the pool. |

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
Do not introduce generic repository traits or a crate per domain speculatively.

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

At initial selection, the crates.io metadata and SQLx release notes were checked
for current releases, minimum Rust versions, and declared licenses. The chosen
direct libraries use MIT or MIT/Apache-2.0 licenses. The transitive
`webpki-roots` certificate data uses CDLA-Permissive-2.0; its reviewed terms allow
redistribution provided the license text accompanies the data. Preserve that
notice in distribution artifacts (see `THIRD_PARTY_NOTICES.md`).
`cargo deny check` evaluates the resolved dependency graph, not just this summary.
The unpublished root package is excluded from license inference: its existing
`LICENSE` is manually preserved, rather than allowing a scanner to assign an
"or later" grant from a generic example. This is not a security guarantee or
a substitute for reviewing behavior and future advisories.

Review lockfile changes deliberately. Update Rust, dependencies, pinned GitHub
Action SHAs and container digests in focused changes, run all documented checks,
and review upstream release notes and advisories. Never automatically publish.
No license exceptions or advisory ignores are permitted without a documented
reason and approval. `license-file` preserves the existing project license;
do not infer a new SPDX grant from the generic license's example notices.
