# Foundation verification

Date: 2026-10-15. Environment: Linux x86_64, Rust 1.99.0, Docker 29.8.2,
Docker Compose 2.36.0, PostgreSQL 18.6 in the pinned PostgreSQL image.
These are functional checks, not performance benchmarks or security guarantees.

## Executed successfully

- `cargo fmt --all -- --check`.
- `cargo clippy --locked --all-targets -- -D warnings`.
- `cargo test --locked`: 15 integration tests, none ignored. Configuration (7),
  HTTP (5), PostgreSQL/migrations (3).
- `cargo build --locked --release`.
- `cargo deny check` with cargo-deny 0.20.2: advisories, dependency licenses,
  sources and bans pass. Transitive duplicate-version warnings remain, including
  SQLx's different cryptographic dependency versions. No advisory suppressions.
  The unpublished root package's existing license is reviewed manually rather
  than inferred by the scanner; see the architecture notes.
- Actionlint 1.7.7 validation of `.github/workflows/ci.yml`.
- Docker Compose configuration validation with a generated local password.
- Final multi-stage `docker build --tag trestle:foundation-check .`.
- Docker runtime: UID/GID `10001:10001`, explicit migrations twice, healthy
  readiness against PostgreSQL over a private container network, SIGTERM exit 0.
- The same final image imported into rootless Podman 3.4.4: non-root runtime,
  liveness 200 and readiness 503 with an unavailable database, SIGTERM exit 0,
  credentials absent from application logs. A separate Podman build and a full
  PostgreSQL deployment under Podman were not tested.
- Native binary: explicit migration command twice; liveness/readiness, HEAD,
  unsupported method and unknown route responses; JSON diagnostics omit raw
  paths/query strings and credentials.
- Paused disposable PostgreSQL: liveness stays healthy, readiness changes
  `200 → 503 → 200` across pause/unpause without restarting the app.
- SIGTERM exits successfully; release binary SIGINT exits successfully, including
  liveness when PostgreSQL is unavailable.
- PostgreSQL custom-format dump and restore into a separate disposable database;
  migration/checksum validation succeeds afterward. Only migration metadata
  exists, so this does not validate recovery of future product data.

## Review

- Standards review: one documented diagnostic mismatch and one naming suggestion;
  both addressed. Non-Unicode configuration errors now identify the setting
  without exposing its bytes, with a regression test.
- Spec review: no missing requirements, scope creep, or incorrect behavior found
  in static review against `docs/foundation-milestone.md`.

## Limits

- Hosted GitHub Actions has not been run; no branch was pushed or release
  published. Local workflow linting is not proof of a hosted CI pass.
- No frontend, authentication, authorization or tenant isolation exists yet.
- No dependency audit can guarantee absence of vulnerabilities.
- No production deployment, remote PostgreSQL TLS interoperability, cross-platform
  support, load tests, timing budgets, accessibility or coverage percentage is
  established by this foundation.
- Podman 3.4.4 on this host lacks the `podman compose` subcommand. Documentation
  includes a provider-free path and notes version/provider requirements.
