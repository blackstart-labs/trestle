# Approved milestone: runnable backend foundation

Approved in the implementation conversation following the read-only audit of
commit `d0f26824065d72dad43e37209eb7f3c2fbf49e0c`.

## Goal and scope

A fresh checkout can start and verify a minimal Rust application connected to
PostgreSQL, with documented behavior and automated checks.

Backend foundation only. No authentication, product CRUD, frontend dashboard,
SQLite, integrations, or artificial product tables.

## Acceptance criteria

1. Reproducible build: documented Rust toolchain, committed application lockfile,
   exact setup/build/test/run commands, dependency compatibility/advisory/license
   review.
2. Typed configuration with safe validation. No embedded credentials or database
   credentials in public responses or logs.
3. Tested HTTP behavior: process liveness, bounded database readiness, safe
   unavailable responses, consistent fallback and internal-error responses.
4. Bounded PostgreSQL connection pool; explicit versioned migrations exercised
   on disposable PostgreSQL, including repeat execution; no speculative schema.
5. Structured request diagnostics, graceful shutdown, explicit development bind,
   documented production exposure requirements, Docker/Podman development
   instructions and a direct-run path.
6. Automated formatting, Clippy, tests, build, PostgreSQL integration and
   dependency/license checks; minimal CI permissions; no publishing/deployment
   credentials. Tests cover readiness failure and invalid configuration.
7. README distinguishes implemented capabilities from plans; architecture and
   initial threat-model notes; preserve license, history, remotes, local ignored
   reference reports, and existing `.gitignore` rule.

Completion requires checks actually executed successfully, with environment
limitations explicitly reported. A configured CI workflow is not evidence that
a hosted CI run passed.

## Approved test seams

- Configuration parsing, including secret-safe errors.
- HTTP health, readiness, and error responses.
- Migration execution on disposable PostgreSQL, including repeat execution.

Test observable behavior through public interfaces rather than private
implementation details.

## Approved architecture direction

Rust modular monolith, PostgreSQL-only initial persistence, later
Svelte/Vite/TypeScript frontend, workspace-based tenancy, and revocable
server-side browser sessions. Only the foundation is implemented here.
