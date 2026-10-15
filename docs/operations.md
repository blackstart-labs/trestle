# Development and operations

This milestone is a development foundation, not a production release.

## Direct PostgreSQL path

Use PostgreSQL 18 with a dedicated local database and login role. Create these
with your installation's administration tools, then export a PostgreSQL
`DATABASE_URL` for that role. Never put a real credential in a committed file.

From the repository root:

```bash
cargo build --locked --release
./target/release/trestle migrate
./target/release/trestle
```

The application defaults to `127.0.0.1:3000` and does not require a database
connection to start serving liveness. Readiness reports database outages.
Supply `TRESTLE_BIND_ADDR` explicitly to expose another interface. Run behind a
TLS reverse proxy with appropriate connection, timeout, and rate limits before
any non-local exposure. The app itself serves plaintext HTTP/1.

The runtime pool adds `statement_timeout=5000` and `lock_timeout=5000` (milliseconds)
and disables statement logging. For remote PostgreSQL use `sslmode=verify-full`
and a trusted server certificate; SQLx supports `sslrootcert` in the URL.

## Podman without Compose

Generate and retain a local password, then:

```bash
export POSTGRES_PASSWORD="$(openssl rand -hex 24)"
podman volume create trestle-postgres-data
podman run --detach --name trestle-db \
  --env POSTGRES_DB=trestle --env POSTGRES_USER=trestle \
  --env POSTGRES_PASSWORD --publish 127.0.0.1:5432:5432 \
  --volume trestle-postgres-data:/var/lib/postgresql \
  postgres:18-bookworm@sha256:afc7e2d441324c0388fa80c3d24f733b4194a4eb7f47dd8ee2b08eb1a24a647c
export DATABASE_URL="postgres://trestle:${POSTGRES_PASSWORD}@127.0.0.1:5432/trestle"
podman exec trestle-db pg_isready -U trestle -d trestle
```

Wait until PostgreSQL is accepting connections before migrating. Use
`podman stop trestle-db` to stop it and `podman start trestle-db` to resume.
Do not remove volumes to troubleshoot credentials. The development database role
is a superuser inside this local database container; it is not suitable for
production. Rootless Podman support depends on the host's namespace/network
configuration and must be checked on the deployment host.

## Application container

Build with `docker build --tag trestle:local .` (or `podman build`). The image
contains a non-root application process, license, and certificate-data notice.
It does not contain source secrets or a PostgreSQL server. Container networking
is distinct from the host: **do not** use a host-loopback database URL inside
the app container. Use a private network and the database container's DNS name,
or a database hostname reachable from the container.

Given a reachable database URL, pass it without placing its value on the command
line:

```bash
docker run --rm --env DATABASE_URL trestle:local migrate
docker run --rm --name trestle-app --env DATABASE_URL \
  --publish 127.0.0.1:3000:3000 trestle:local
```

Add the appropriate `--network` for your database. The image binds all interfaces
*inside* the container; the example publishes only host loopback. Configure
external health probes against `/health/live` and `/health/ready`; no shell,
curl, or container healthcheck is required by the runtime image. Containers
receive SIGTERM, with a ten-second request drain deadline. Set an orchestrator
stop timeout longer than that deadline (for example, 15 seconds).

No container, release, or registry publishing is automated. The notice inventory
must be completed before distributing release artifacts. Modified network
deployments must offer the corresponding source required by AGPL section 13.

## Migrations and upgrades

- `trestle migrate` runs embedded forward SQLx migrations and exits nonzero on
  failure. It never prints the connection URL or raw database errors.
- Baseline `202610150001_foundation.sql` executes `SELECT 1`; SQLx creates its
  `_sqlx_migrations` metadata table. There are no product tables.
- Serving HTTP does **not** apply or validate migrations. Readiness checks
  connectivity only. Deployments must migrate successfully before routing users.
- Migrations use SQLx checksums, transactional execution by default, and its
  PostgreSQL migration lock. Do not change applied migration files or bypass
  checksum errors. Add a new forward migration for schema changes.
- A migration command is bounded to 60 seconds overall; statements and lock
  waits to five seconds. Do not assume these bounds suit future data migrations.
- Use a dedicated migration role and a least-privilege runtime role in
  production. Never grant the runtime role `CREATEDB` merely to run tests.
- Before future upgrades, take a verified backup, test migrations on a restored
  copy, document compatibility, run the migration command once, then verify
  readiness and the affected workflows. No automatic downgrade exists.

For SQLx tests, use a disposable test server with a `CREATEDB` role. Each
`#[sqlx::test]` receives a separate database; failed tests may leave databases
for inspection. Do not run the suite with production credentials.

## Backup and restore procedure

There is no product data yet. This is a starting procedure, not a claim that a
future application's full backup requirements have been verified.

For a direct deployment, configure libpq credentials securely (for example a
mode-0600 passfile), then use matching PostgreSQL client tools. Database names
below are operator-selected placeholders:

```bash
pg_dump --format=custom --file=trestle.dump trestle
createdb trestle_restore_check
pg_restore --exit-on-error --no-owner --dbname=trestle_restore_check trestle.dump
```

Never restore over a live database as a verification step. Restrict backup file
permissions, encrypt backups where appropriate, store copies off-host, and
define retention and recovery objectives. Back up role/grant definitions and
operator configuration separately without exposing secrets. Future attachments
will require coordinated storage backups too.

Point a separate application instance at the restored database, verify migration
execution/readiness and future record invariants, and record the restore result.
Take a final consistent backup before an actual cutover; stop writes as needed.
Binary rollback is safe only when schema compatibility has been established.
Never automatically delete the original database or volume.

## Troubleshooting

| Symptom | Check |
| --- | --- |
| Missing/invalid configuration | Export known settings; the app does not load `.env`. Errors deliberately omit values. |
| Bind failure | Another process may own the port; verify the IP/port and permissions. |
| Liveness succeeds, readiness fails | Database DNS/routing, credentials, TLS certificates, pool saturation, role permissions. |
| Migration failure | Verify database/server logs privately, permissions, migration history/checksums and timeout limits. Never bypass a checksum mismatch. |
| Container cannot reach database | Container localhost is not host localhost; check private network and DNS name. |
| Password changed but login fails | Container initialization settings do not reset an existing volume's password. Recover through authorized database administration. |
| Dependency checker cannot fetch advisories | Restore network access and rerun; do not report a successful security check. |

Do not paste raw database logs, URLs, credentials, or backups into public reports.
