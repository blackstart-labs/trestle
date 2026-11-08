# Operations & Deployment Guide

This guide covers building, configuring, deploying, and maintaining Trestle in production and local environments.

---

## 1. Quickstart (Local Development)

### Prerequisites:
- Go 1.24+
- Node.js 20+ & npm 10+
- PostgreSQL 15+ (running locally or in a container)

```bash
# 1. Build frontend static assets into web/dist
npm ci --prefix web --ignore-scripts
npm run build --prefix web

# 2. Build the single static Go binary (with embedded frontend)
CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o trestle .

# 3. Export database connection
export DATABASE_URL="postgres://trestle:secret@127.0.0.1:5432/trestle?sslmode=disable"

# 4. Apply schema migrations
./trestle migrate

# 5. Bootstrap initial admin account (if first time)
./trestle bootstrap

# 6. Start the server
./trestle
```

Access the application in your browser at `http://127.0.0.1:3000`.

---

## 2. Environment Variables

| Variable | Default | Description |
| --- | --- | --- |
| `DATABASE_URL` | *(Required)* | PostgreSQL connection URL, e.g. `postgres://user:pass@db:5432/trestle?sslmode=verify-full`. |
| `TRESTLE_BIND_ADDR` | `127.0.0.1:3000` | Interface and port to listen on. In containers, set to `0.0.0.0:3000`. |
| `TRESTLE_ORIGIN` | `http://127.0.0.1:3000` | Canonical public origin for SameSite cookie handling and CSRF validation. |
| `SESSION_TTL_HOURS` | `336` (14 days) | Absolute session duration before requiring re-authentication. |
| `LOG_LEVEL` | `info` | Structured logging verbosity (`debug`, `info`, `warn`, `error`). |
| `TRESTLE_BOOTSTRAP_USERNAME` | `admin` | Optional non-interactive initial admin username. |
| `TRESTLE_BOOTSTRAP_PASSWORD` | *(None)* | Optional initial admin password for automated provisioning. |

---

## 3. CLI Subcommands

The compiled `trestle` binary provides operational commands:

```bash
# Start the HTTP server (default if no subcommand given)
trestle

# Run pending forward database migrations
trestle migrate

# Create initial administrator account (prompts for password securely)
trestle bootstrap

# Create an additional user
trestle adduser <username>

# Reset a user's password (immediately revokes all active sessions)
trestle resetpassword <username>

# Disable / enable user access
trestle disableuser <username>
trestle enableuser <username>
```

---

## 4. Container Deployment (Docker / Podman)

### 4.1 Multi-Stage Container Build
The root `Dockerfile` compiles the Svelte frontend and the Go backend into a minimal non-root image:

```bash
docker build -t trestle:latest .
```

### 4.2 Running with PostgreSQL

Create a shared network and start PostgreSQL:
```bash
docker network create trestle-net

docker run -d --name trestle-db \
  --network trestle-net \
  -e POSTGRES_DB=trestle \
  -e POSTGRES_USER=trestle \
  -e POSTGRES_PASSWORD=supersecret \
  -v trestle_pgdata:/var/lib/postgresql/data \
  postgres:16-alpine
```

Run schema migrations:
```bash
docker run --rm \
  --network trestle-net \
  -e DATABASE_URL="postgres://trestle:supersecret@trestle-db:5432/trestle?sslmode=disable" \
  trestle:latest migrate
```

Bootstrap initial admin account:
```bash
docker run --rm -it \
  --network trestle-net \
  -e DATABASE_URL="postgres://trestle:supersecret@trestle-db:5432/trestle?sslmode=disable" \
  -e TRESTLE_BOOTSTRAP_USERNAME="admin" \
  -e TRESTLE_BOOTSTRAP_PASSWORD="ChangeMePromptly123!" \
  trestle:latest bootstrap
```

Start the application container:
```bash
docker run -d --name trestle-app \
  --network trestle-net \
  -p 127.0.0.1:3000:3000 \
  -e DATABASE_URL="postgres://trestle:supersecret@trestle-db:5432/trestle?sslmode=disable" \
  -e TRESTLE_BIND_ADDR="0.0.0.0:3000" \
  -e TRESTLE_ORIGIN="https://trestle.example.com" \
  trestle:latest
```

---

## 5. Health Probes & Monitoring

Trestle exposes standard health probe endpoints for Kubernetes, Docker Compose, or Nomad:

- **Liveness Probe**: `GET /health/live`
  - Returns `200 OK` immediately if the HTTP process is running.
  - Does not query PostgreSQL.
- **Readiness Probe**: `GET /health/ready`
  - Pings PostgreSQL and evaluates connection pool health.
  - Returns `200 OK` with `{"status":"ready"}` when healthy.
  - Returns `503 Service Unavailable` with `{"status":"not_ready"}` during database outages or startup delays.

### Graceful Shutdown
Upon receiving `SIGINT` or `SIGTERM`, the HTTP server stops accepting new connections and permits a 10-second drain period for in-flight requests before terminating. Ensure orchestrator stop timeouts are configured for at least 15 seconds.

---

## 6. Backup & Recovery

Trestle's state is stored in PostgreSQL. Standard `pg_dump` creates consistent snapshots:

### Backup:
```bash
pg_dump --format=custom --file=trestle-$(date +%F).dump "$DATABASE_URL"
```

### Restore:
```bash
# 1. Restore to a staging or check database
createdb trestle_restore_test
pg_restore --exit-on-error --no-owner --dbname=trestle_restore_test trestle-*.dump

# 2. Verify with trestle migrate and readiness probe
DATABASE_URL="postgres://.../trestle_restore_test" ./trestle migrate
```

---

## 7. Troubleshooting

| Symptom | Cause & Solution |
| --- | --- |
| `database migration failed or timed out` | Check PostgreSQL connectivity and credentials. If checksum mismatch occurs, ensure migration files were not modified after application. |
| `connection refused on port 3000` | Ensure `TRESTLE_BIND_ADDR` is set to `0.0.0.0:3000` when running inside containers. |
| `CSRF token missing or mismatch` | Verify `TRESTLE_ORIGIN` matches the reverse proxy's public HTTPS URL, and the client passes `X-CSRF-Token` header. |
| `401 Unauthorized after container restart` | Verify session storage in PostgreSQL was persisted across container restarts (check volume mount). |
