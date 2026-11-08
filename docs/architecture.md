# Architecture & System Design

Trestle is a self-hosted workspace for projects, tasks, time, and knowledge:
a single Go (`net/http`) backend serving an embedded Svelte 5 + Vite static frontend from one origin.

## Core Architectural Decisions

- **Modular Monolith**: One Go module and one HTTP process, eliminating microservice deployment complexity while preserving clean module boundaries ([ADR-0005](adr/0005-go-backend.md)).
- **Standard Library Foundation**: Built on Go's `net/http` (`ServeMux`), `encoding/json`, `log/slog` structured logging, and PostgreSQL via `pgx/v5` connection pooling.
- **Single-Origin Deployment**: Svelte 5 production assets are embedded into the Go binary at compile-time via `go:embed`. Node.js and npm are build-time tools only; the deployed production server is a single static Go binary.
- **Explicit Migrations**: Database schema migrations (`trestle migrate`) run forward explicitly before HTTP serving starts. Serving HTTP never applies schema migrations automatically.
- **Tenant Isolation**: Workspace is the primary tenant boundary. All data lookups, mutations, aggregates, and searches are strictly scoped by workspace and project IDs.
- **Strict Content Security Policy (CSP)**: `default-src 'none'; script-src 'self'; style-src 'self'; font-src 'self'; img-src 'self' blob:; frame-ancestors 'none'; base-uri 'none'; form-action 'self'`. No runtime CDNs, external fonts, remote scripts, or inline `style="..."` attributes.

---

## System Architecture

```text
┌────────────────────────────────────────────────────────┐
│                   Web Browser / Client                 │
│              (Svelte 5 + Vite, Mona Sans)              │
└───────────────────────────┬────────────────────────────┘
                            │ HTTPS / Same-Origin HTTP
┌───────────────────────────▼────────────────────────────┐
│                    Trestle HTTP Server                 │
│                                                        │
│  ┌──────────────────────────────────────────────────┐  │
│  │ internal/httpapi (Router, Session, CSRF, Health) │  │
│  └────────┬───────────────────────┬─────────────────┘  │
│           │                       │                    │
│  ┌────────▼──────────┐ ┌──────────▼─────────────────┐  │
│  │ internal/identity │ │    internal/projects       │  │
│  │ (Auth, Sessions,  │ │ (Projects, Tasks, Statuses,│  │
│  │  Workspaces)      │ │  Milestones, Integrations) │  │
│  └────────┬──────────┘ └──────────┬─────────────────┘  │
│           │                       │                    │
│  ┌────────▼───────────────────────▼─────────────────┐  │
│  │  internal/database (pgxpool, Migrations, Store)  │  │
│  └────────────────────────┬─────────────────────────┘  │
└───────────────────────────┼────────────────────────────┘
                            │ TCP / UNIX Socket
┌───────────────────────────▼────────────────────────────┐
│                  PostgreSQL Database                   │
│          (_sqlx_migrations, Workspaces, Tasks)         │
└────────────────────────────────────────────────────────┘
```

---

## Module Responsibilities

| Module | Responsibility |
| --- | --- |
| `main.go` | Entry point, CLI subcommand dispatch (`serve`, `migrate`, `bootstrap`), embedded asset mounting, graceful shutdown with request draining. |
| `internal/config` | Strict environment variable parsing, bounds validation, credential redaction in logs. |
| `internal/database` | Bounded lazy `pgxpool.Pool` construction, transaction helpers, and forward migration runner using PostgreSQL advisory locking and SHA-384 checksums. |
| `internal/identity` | Argon2id password hashing, constant-time verification, session lifecycle, workspaces, memberships, and operator recovery. |
| `internal/projects` | Project management, task scheduling, custom statuses, category rollups, milestones, and webhook event dispatch. |
| `internal/integrations` | Outbound event delivery (Telegram, Slack, Discord) with SSRF safeguards, non-routable IP blocking, and write-only credentials. |
| `internal/httpapi` | HTTP router (`ServeMux`), middleware (sessions, CSRF, logging, recoverer), JSON error envelopes, and static asset serving. |
| `web/` | Svelte 5 frontend with runes (`$state`, `$derived`, `$props`), Mona Sans typography, semantic design tokens, and WCAG 2.2 AA accessibility. |

---

## Domain Model & Invariants

```text
User ──< WorkspaceMembership >── Workspace ──< Project
                                                 │
                        ┌────────────────────────┼────────────────────────┐
                        │                        │                        │
                     Status                  Milestone                   Task
                        │                                                 │
                        └────────────────── Task ─────────────────────────┘
                                             │
                        ┌────────────────────┼────────────────────┐
                        │                    │                    │
                    Assignee               Label               Comment
```

### Invariants:
1. **Tenant Containment**: A project belongs to exactly one workspace. Tasks, statuses, labels, and milestones cannot cross project or workspace boundaries.
2. **Deterministic Task Keys**: Each task has an internal UUID and a human-readable identifier (e.g. `CORE-12`) scoped to the project key and allocated sequentially within a transaction.
3. **Status Category Mapping**: Custom project statuses map to five canonical categories: `backlog`, `unstarted`, `started`, `completed`, `cancelled`. Completion rates always compute as `completed ÷ (all − cancelled)`.
4. **Milestone Targets**: Milestones group tasks toward outcomes with start and due dates, color branding, and progress rollup calculations.

---

## Resource & Performance Strategy

- **Zero-Start Dependency**: The HTTP server starts without requiring a reachable database; `/health/live` responds immediately, while `/health/ready` evaluates database readiness.
- **Bounded Pools**: The `pgxpool` configuration enforces explicit connection limits (`MinConns = 0`, bounded `MaxConns`, `ConnectTimeout`).
- **Server-Side Deadlines**: Queries run with `statement_timeout=5000` and `lock_timeout=5000` to prevent stalled locks from exhausting connection capacity.
- **Critically Damped Motion**: Frontend transitions run through `web/src/lib/motion.ts` using spring physics (`--ease-spring`) without overshooting or layout cascading.
