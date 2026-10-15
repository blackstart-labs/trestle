# Deployment and compatibility

Status: core app + PostgreSQL foundation only. Exact current commands and
configuration belong in [operations](../operations.md), not duplicated here.

One app process serves HTTP and future static assets. PostgreSQL holds durable
state; future filesystem attachments need their own protected persistent volume.
A reverse proxy provides HTTPS/ingress safeguards for non-local exposure. No
frontend server, cache or worker fleet is required. Migrations are a separate
explicit step using appropriate credentials, before user traffic reaches a new
schema-dependent binary.

## Evidence matrix

| Target | Status | What is / is not established |
| --- | --- | --- |
| Linux x86_64 direct binary | Foundation verified | Build, health/migrate, signals; no product workflows |
| Docker Linux image | Foundation verified | Build, non-root app + PostgreSQL/private-network smoke |
| Rootless Podman 3.4.4 | Partial verification | Imported same app image, health/errors/signals; no separate build/full DB deployment |
| PostgreSQL 18.6 | Foundation verified | Real tests and metadata-only restore; not future domain scale |
| macOS / Windows server or source build | Unverified | Do not claim support merely because Rust is portable |
| Linux ARM64 | Unverified | No build/runtime evidence yet |
| Android/Termux server | Evaluation planned | No toolchain/PostgreSQL/storage/process-continuity evidence |
| Desktop/tablet/mobile browsers | Partial | Foundation smoke tested in Chromium/320px viewport; no product-browser support matrix yet |
| Native desktop/mobile app | Not an initial target | Responsive web does not imply native packages or offline sync |

Configure process/container resource limits and DB pool budgets together; they
are not fair performance comparisons unless both services are measured. Network
binding inside a container differs from publishing host ports. Backups must
eventually coordinate database, attachment storage and operator secret/config
recovery. Never remove a volume merely to fix credentials.

Production upgrades require tested schema compatibility, protected backups,
restored-record invariants and documented rollback limits. No automatic release,
zero-downtime or availability SLA is established.
