# Trestle

An open-source, self-hosted productivity platform unifying projects, time tracking, knowledge management, and developer workflows in one lightweight system.

**Implemented:** a Rust HTTP backend foundation, PostgreSQL connectivity,
explicit migrations, JSON health checks, structured diagnostics, Argon2id identity
and sessions, workspace tenancy, and automated tests, plus a responsive Svelte
application shell with light/dark/system appearance, keyboard navigation, real
readiness check, sign-in, and workspace selector embedded in the single Rust binary.

## Quick start (one-command setup)

Run the entire application with one command from GitHub across **Linux, macOS, and Windows** (automatically detects free ports, configures PostgreSQL, builds the frontend, migrates, bootstraps, and launches Trestle):

#### Linux / macOS / WSL:
```bash
git clone https://github.com/maruf-pfc/trestle.git && cd trestle && ./run.sh
```

#### Windows (PowerShell):
```powershell
git clone https://github.com/maruf-pfc/trestle.git; cd trestle; .\run.ps1
```

#### Windows (Command Prompt):
```cmd
git clone https://github.com/maruf-pfc/trestle.git && cd trestle && run.cmd
```

Or if you already have the repository cloned:
- **Linux / macOS:** `./run.sh`
- **Windows:** `.\run.ps1` or `run.cmd`

### What the runner handles automatically:

1. **Port conflict resolution:** If port `5432` (PostgreSQL) or `3000` (web application) is already in use by another database or application on your system, `run.sh` automatically finds and binds to the next available free ports without colliding.
2. **Environment & credentials:** Generates a secure `.env` file with a dedicated database password and saves your configured ports.
3. **Database startup:** Launches PostgreSQL 18 via Docker Compose (or Podman Compose) and waits for connection readiness.
4. **Frontend build:** Checks and builds production assets into `web/dist` embedded into the Rust executable.
5. **Database migrations:** Automatically applies all schema migrations.
6. **Admin bootstrap:** Sets up the initial operator account (`admin` / `adminpassword123`). Pass `./run.sh --interactive` if you prefer to prompt for custom credentials.
7. **Application launch:** Starts the HTTP server and displays the local URL (`http://127.0.0.1:3000` or auto-selected port).

To stop background services:

```bash
./run.sh stop
```

## Requirements

- Rust toolchain from `rust-toolchain.toml` (Rustup installs it on first use).
- Node 24.22.0 (`.node-version`) and npm 11.19.1, for building/testing only.
- PostgreSQL 18 (the tested database major).
- Docker with Compose, Podman with a Compose provider, or a local PostgreSQL
  installation.

## Manual development start

If you prefer to run each step manually in separate commands:

```bash
npm ci --prefix web --ignore-scripts
npm run build --prefix web
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

Open <http://127.0.0.1:3000/>. Cargo requires `web/dist`: build the frontend
before any Cargo build/test/run, and rebuild the binary after frontend changes.
The runtime needs no Node, npm, source tree or asset directory. For local UI
development, run the backend and `npm run dev --prefix web`; Vite binds loopback
and proxies `/health` to port 3000. Do not expose Vite as a production server.

The shell's Overview, Appearance and Instance status links are real in-page
sections, not product routes. Theme preference is saved only on this browser;
blocked storage shows a warning without breaking the page. No accounts or
workspaces are simulated. See the [shell specification](docs/features/application-shell.md)
and [design tokens](docs/design/design-tokens.md).

## HTTP interface

```bash
curl --fail http://127.0.0.1:3000/health/live
curl --fail http://127.0.0.1:3000/health/ready
```

| Request | Response |
| --- | --- |
| `GET /` | Foundation HTML with external compiled JS/CSS |
| `GET /assets/<known file>` | Embedded JS/CSS or frontend license text; no filesystem lookup |
| `GET /health/live` | `200 {"status":"ok"}`; no database access |
| `GET /health/ready` | `200 {"status":"ok"}` when `SELECT 1` succeeds |
| Unavailable or timed-out database | `503 {"error":{"code":"unavailable","message":"Service unavailable"}}` |
| Unknown route | `404 {"error":{"code":"not_found","message":"Not found"}}` |
| Unsupported method on a known route | `405 {"error":{"code":"method_not_allowed","message":"Method not allowed"}}` |

Health/errors are JSON. All responses have `Cache-Control: no-store` and
`X-Content-Type-Options: nosniff`. `HEAD` follows normal HTTP semantics (no body).
Readiness checks connectivity, **not** schema compatibility or migration status.
Run migrations as a separate deployment step. No product endpoints exist.
Production responses also carry a restrictive same-origin CSP. Unknown paths
never fall back to HTML; no client-side product routing is implemented yet.

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
npm ci --prefix web --ignore-scripts
npm run format:check --prefix web
npm run check --prefix web
npm test --prefix web
npm audit --prefix web --audit-level=low
npm run build --prefix web
npm exec --prefix web -- playwright install chromium
npm run test:e2e --prefix web
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

Next: implement the first bootstrap/sign-in/session/workspace-access slice under
the [approved identity policy](docs/security/identity-policy.md) (M03a.1), then
multi-user membership/recovery/ownership and one authorized
project/task workflow. Boards, time tracking, Markdown knowledge management,
integrations and automation follow incrementally. These are plans, not current
capabilities; see the implementation tracker for dependencies.

## License

See [LICENSE](LICENSE) for the existing GNU Affero General Public License v3.
Modified network deployments must satisfy its corresponding-source obligations.
