---
status: accepted
---
# Go backend

Supersedes the Rust/SQLx choice in [ADR 0001](0001-modular-monolith-and-postgresql.md)
and the Rust asset server in [ADR 0002](0002-frontend-and-rendering.md); their
other decisions stand.

The backend was one Rust (axum, tokio, SQLx) crate. Rewrite it in Go as one
module and one process, using the standard library `net/http` ServeMux,
`log/slog` JSON diagnostics and `embed` for the compiled frontend, with pgx for
PostgreSQL, `golang.org/x/crypto/argon2` for Argon2id and
`golang.org/x/term` for operator password prompts. Packages live under
`internal/` (config, database, identity, projects, httpapi, testdb).

The HTTP API, cookies, CSP, JSON shapes, environment variables, subcommands and
database schema are unchanged. Migration history stays in `_sqlx_migrations`
with the same SHA-384 checksums, so existing databases migrate forward without
any manual step and applied migration files must still never change.
`CGO_ENABLED=0` produces a single static binary. Checks become `gofmt`,
`go vet`, `go test ./...` and `govulncheck`; database tests skip when
`DATABASE_URL` is unset, so CI must always set it. govulncheck does not review
licenses, so `THIRD_PARTY_NOTICES.md` is maintained by hand from `go.sum`.
