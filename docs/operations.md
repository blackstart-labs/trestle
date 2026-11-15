# Operations & Deployment Guide

This guide covers building, configuring, deploying, and maintaining Trestle in production and local environments.

---

## 1. Quickstart (Local Development)

### Prerequisites:
- Go 1.27.4 (the version pinned in `go.mod`)
- Node.js 24.21+ and npm 11.19+ (pinned in `.node-version` and `web/package.json`)
- PostgreSQL 18 (running locally or in a container)

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
| `TRESTLE_DEV_MODE` | `false` | Relaxes the Secure cookie flag for loopback HTTP only. It is rejected for any non-loopback bind. |
| `SESSION_TTL_HOURS` | `336` (14 days) | Absolute session lifetime; the idle limit remains 12 hours. |
| `TRESTLE_MAX_DB_CONNECTIONS` | `5` | Connection pool size, 1–32. |
| `TRESTLE_DB_TIMEOUT_MS` | `2000` | Connection and readiness timeout, 100–10000 ms. |
| `TRESTLE_ARGON2_M_COST` / `_T_COST` / `_PARALLELISM` | `65536` / `2` / `1` | Password hashing cost; raise only with a measured capacity plan. |
| `TRESTLE_BOOTSTRAP_USERNAME` | `admin` | Optional non-interactive initial admin username. |
| `TRESTLE_BOOTSTRAP_PASSWORD` | *(None)* | Optional initial admin password for automated provisioning. The local runner generates a random one instead of shipping a fixed default. |

CSRF is validated with the `trestle-csrf` cookie and the `X-CSRF-Token` header;
there is no `TRESTLE_ORIGIN` setting. Cookies are `SameSite=Lax`, so a public
deployment must be served over HTTPS for browsers to send them.

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
The root `Dockerfile` compiles the Svelte frontend and the Go backend into a minimal non-root image. The pinned base images are multi-arch, so the same file builds on amd64 and arm64:

```bash
# Native build on the current machine
docker build -t trestle:latest .

# Explicit multi-arch build (needs buildx and QEMU on the builder)
docker buildx build --platform linux/amd64,linux/arm64 -t your-registry/trestle:latest --push .
```

For a local arm64 image on an amd64 machine:
```bash
docker buildx build --platform linux/arm64 -t trestle:arm64 --load .
```

### 4.2 VPS / production stack

`compose.prod.yaml` runs PostgreSQL, applies migrations, then starts the app.
The database is not published and Trestle binds to loopback so a reverse proxy
can terminate TLS.

```bash
cp .env.production.example .env
# Replace the password in POSTGRES_PASSWORD and DATABASE_URL.
docker compose -f compose.prod.yaml up -d --build
docker compose -f compose.prod.yaml run --rm app bootstrap
```

Put Caddy, nginx or Traefik in front of `127.0.0.1:3000` and give it a public
HTTPS hostname. Cookies are Secure outside `TRESTLE_DEV_MODE`, so plain HTTP on
a public address will sign in but the browser will not send the session back.
Example Caddyfile:

```caddyfile
trestle.example.com {
    reverse_proxy 127.0.0.1:3000
}
```

Update the stack with `docker compose -f compose.prod.yaml up -d --build`.
`run.sh` remains the local development runner; it binds to loopback by default
and refuses to rewrite an explicit bind address or dev-mode choice.

### 4.3 Plain container run

If you do not use Compose, the image still works with three commands. Create a
database, run `migrate`, bootstrap, then serve. The container needs
`TRESTLE_BIND_ADDR=0.0.0.0:3000`, a `DATABASE_URL` that reaches PostgreSQL, and
`TRESTLE_DEV_MODE=false` behind HTTPS.

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
| `CSRF token missing or mismatch` | The browser must send the `trestle-csrf` cookie and mirror it in `X-CSRF-Token`. Clear stale cookies, then confirm the proxy forwards both the cookie and that header. |
| Sign-in succeeds but the next request is unauthenticated | Cookies are `Secure` unless `TRESTLE_DEV_MODE=true` is set on a loopback bind. Serve the public URL over HTTPS. |
| `exec format error` on a Raspberry Pi or ARM VPS | The image was built for the wrong architecture. Build natively on the host or with `docker buildx build --platform linux/arm64`. |
| `401 Unauthorized after container restart` | Verify session storage in PostgreSQL was persisted across container restarts (check volume mount). |
