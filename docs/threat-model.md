# Initial threat model

This is a development foundation, not an authenticated productivity application.

## Assets and trust

- PostgreSQL credentials, database contents, migration history, and operator logs.
- The operator controls process environment, binary, database role, and network.
- HTTP callers and network inputs are untrusted, including request URLs.
- SQLx, PostgreSQL, Rust dependencies, CI actions, and container bases are supply
  chain inputs; lockfiles, digests, and advisory checks reduce but do not remove risk.

## Current safeguards

- Default HTTP bind is loopback. Development PostgreSQL is bound to loopback.
- Health endpoints expose only fixed status/error objects, never database details.
- Configuration diagnostics redact database options; parser errors name settings,
  not their values. SQL statement logging is disabled for application pools.
- Request diagnostics record method, matched route, status, and duration, not
  raw paths/query strings, headers, cookies, credentials, or bodies.
- Pool size, acquisition/readiness duration, SQL statement/lock duration, and
  shutdown drain duration are bounded. Health requests do not parse bodies.
- The serving process never runs migrations automatically.
- Container process runs without root; no credentials are baked into the image.
- CI has read-only repository permissions and does not publish or deploy.

## Remaining risks and required next work

- There is no authentication, authorization, session handling, CSRF protection,
  or tenant data isolation yet. Do not add product endpoints before these are
  designed, implemented, and tested.
- Health endpoints are unauthenticated. Repeated readiness calls still consume
  database resources; restrict them at the reverse proxy and tune probe frequency.
- HTTP TLS, connection-level limits, slow-client protection and ingress rate
  limits are operator responsibilities for this milestone. The built-in server
  is not an Internet-facing hardened edge proxy.
- PostgreSQL TLS is supported by the driver; local examples use local plaintext
  connections. For remote databases require certificate verification
  (`sslmode=verify-full`) and trusted certificates. Do not assume TLS is enforced
  by the default URL behavior.
- Environment variables are visible to privileged host/container operators.
  Use an appropriate secret manager; never paste credentials into issues or logs.
- Migration credentials can modify schema. Use a separate migration role in
  production and a restricted runtime role. Do not use the local development
  superuser configuration for production.
- No backup/restore or zero-downtime upgrade guarantee exists. Follow and verify
  the operational procedure before trusting data to a deployment.
- There is no file upload, Markdown rendering, webhook, API token, or integration
  credential handling. Add explicit threat-model updates when these arrive.
- Password authentication will need Argon2id, bounded hashing concurrency,
  abuse controls, bootstrap/recovery rules, and revocable sessions.
- Workspace isolation must cover reads, writes, membership changes, aggregates,
  search and exports, with database constraints and negative cross-tenant tests.

## Source availability

Preserve the AGPL license. Network deployments of modified versions must provide
the corresponding source as required by section 13. Release artifacts must be
traceable to their source, including build and installation instructions.
