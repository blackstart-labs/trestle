# Trestle

An open-source, self-hosted productivity platform under development.

**Implemented:** a Rust HTTP backend foundation, PostgreSQL connectivity,
explicit migrations, JSON health checks, structured diagnostics, and automated
tests. There is no frontend, account system, workspace, project, task, timer,
document editor, or integration yet. This is not a production-ready release.

## Requirements

- Rust toolchain from `rust-toolchain.toml` (Rustup installs it on first use).
- PostgreSQL 18 (the tested database major).
- Docker with Compose, Podman with a Compose provider, or a local PostgreSQL
  installation. OpenSSL is used below only to generate a local password.

## Start locally

From the repository root, in one shell:

```bash
export POSTGRES_PASSWORD="$(openssl rand -hex 24)"
export DATABASE_URL="postgres://trestle:${POSTGRES_PASSWORD}@127.0.0.1:5432/trestle"
docker compose up -d --wait db
cargo run --locked -- migrate
cargo run --locked
```

Stop with Ctrl-C. Use `docker compose stop db` to stop PostgreSQL without
deleting its volume. The generated password must be saved securely if reusing
that volume: changing `POSTGRES_PASSWORD` does not change an initialized
database's password. Do not delete volumes to resolve an authentication error.

On Podman versions with the `compose` subcommand, replace `docker compose` with
`podman compose`; a Compose provider must be installed. Older Podman versions
may require `podman-compose` directly. See [operations](docs/operations.md) for
a provider-free Podman command and a direct PostgreSQL path.

The app does not load `.env` automatically. `.env.example` documents settings
without working credentials; supply actual values through the environment.

## HTTP interface

```bash
curl --fail http://127.0.0.1:3000/health/live
curl --fail http://127.0.0.1:3000/health/ready
```

| Request | Response |
| --- | --- |
| `GET /health/live` | `200 {"status":"ok"}`; no database access |
| `GET /health/ready` | `200 {"status":"ok"}` when `SELECT 1` succeeds |
| Unavailable or timed-out database | `503 {"error":{"code":"unavailable","message":"Service unavailable"}}` |
| Unknown route | `404 {"error":{"code":"not_found","message":"Not found"}}` |
| Unsupported method on a known route | `405 {"error":{"code":"method_not_allowed","message":"Method not allowed"}}` |

Responses are JSON with `Cache-Control: no-store` and
`X-Content-Type-Options: nosniff`. `HEAD` follows normal HTTP semantics (no body).
Readiness checks connectivity, **not** schema compatibility or migration status.
Run migrations as a separate deployment step. No product endpoints exist.

## Configuration

| Setting | Default | Validation |
| --- | --- | --- |
| `DATABASE_URL` | Required | PostgreSQL URL parsed by SQLx; never printed |
| `TRESTLE_BIND_ADDR` | `127.0.0.1:3000` | IP socket address; port must be nonzero |
| `TRESTLE_MAX_DB_CONNECTIONS` | `5` | Integer, 1–32 |
| `TRESTLE_DB_TIMEOUT_MS` | `2000` | Integer, 100–10000; pool acquisition and full readiness check |

Pool minimum is zero. Statements and lock waits are limited to five seconds at
the database. The migration command has a 60-second overall limit, so future
long-running migrations will require an explicit operational design change.
Shutdown drains requests for up to ten seconds and then closes the pool.
Diagnostics are JSON at info level, without raw request URLs or database errors.

## Checks

```bash
cargo fmt --all -- --check
cargo clippy --locked --all-targets -- -D warnings
cargo test --locked --test config
cargo test --locked --test http
# Requires DATABASE_URL pointing to a disposable development/test server:
cargo test --locked
cargo build --locked --release
cargo install cargo-deny --version 0.20.2 --locked
cargo deny check
docker build --tag trestle:local .
```

The database tests require a role with `CREATEDB` permission and create isolated
SQLx test databases. **Never run them against production.** Missing database
configuration fails database tests rather than silently skipping them. No test
coverage percentage or performance baseline is claimed.

CI runs the same checks with PostgreSQL, plus a container build. It does not
publish images, deploy, or create releases.

## Documentation

- [Product documentation, feature status and roadmap](docs/README.md)
- [Architecture and dependency policy](docs/architecture.md)
- [Initial threat model](docs/threat-model.md)
- [Deployment, migrations, backups and troubleshooting](docs/operations.md)
- [Executed checks and limitations](docs/verification.md)
- [Contributing](CONTRIBUTING.md)
- [Security reporting](SECURITY.md)

## Direction

Next: finish the product-definition documentation, then the frontend foundation
and accessible application shell. Resolve identity/bootstrap/recovery decisions
before implementing sessions and isolated workspaces, followed by one authorized
project/task workflow. Boards, time tracking, Markdown knowledge management,
integrations and automation follow incrementally. These are plans, not current
capabilities; see the implementation tracker for dependencies.

## License

See [LICENSE](LICENSE) for the existing GNU Affero General Public License v3.
Modified network deployments must satisfy its corresponding-source obligations.
