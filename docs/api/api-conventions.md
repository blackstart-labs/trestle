# HTTP interface conventions

Status: health endpoints implemented; product conventions below are planned.

- Product JSON interfaces will live under `/api/v1`; health remains
  `/health/live` and `/health/ready`. No `/api/v1` product routes exist yet.
- Use resource identifiers in paths and typed JSON bodies. List operations
  accept allowlisted filter/sort keys, bounded limits and opaque continuation
  cursors; each feature specifies the exact stable ordering and tie-breaker.
- Authenticated writes use revocable sessions and CSRF protection. No permissive
  cross-origin credential policy is assumed.
- Validation errors name fields without echoing secret values. Use the shared
  error envelope and stable codes; never return database/internal errors.
- Concurrency-sensitive edits require an expected version; stale edits return
  `409` with a safe reconciliation contract. Timer starts and integration-driven
  creates require scoped idempotency rules before implementation.
- Timestamps are RFC3339 instants with timezone offsets (normally UTC); calendar
  dates are `YYYY-MM-DD`. Money/time units must be explicit, not locale strings.
- Define unknown-property behavior, maximum lengths/body size and response
  fields per endpoint. Do not invent shared permissive DTOs for all domains.
- Authorization-denied resources must not leak existence through title/count
  projections; feature specifications choose consistent `403`/`404` behavior.
- Publish maintained endpoint contracts before browser/integration callers rely
  on them. Evaluate generated OpenAPI when product endpoints exist, not as a
  mandatory empty dependency now.

Contract versioning is not just a URL prefix. Before external clients ship,
define additive-change rules, deprecation notice periods, event schema versions,
and migration/support policy. No backward-compatibility window is promised yet.
