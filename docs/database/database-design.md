# Database strategy

Status: PostgreSQL-only foundation. The only current table is SQLx migration
metadata from the no-product-schema baseline migration. No identity, project,
task, time or document tables exist.

## Rules for future migrations

- Specify the feature and invariants first; introduce only its required schema.
- Prefer normalized relationships and constrained typed values. JSONB is for
  genuinely dynamic preferences/properties, not tenant ownership or status rules.
- Use stable IDs (UUID is the proposed product identifier direction; exact
  generation/version is a feature decision), plus scoped human-readable keys
  where useful. An unguessable ID is not authorization.
- Put workspace/project consistency into composite unique/foreign keys where
  relationships need it, not independent foreign keys that permit mismatches.
- Use `TIMESTAMPTZ` for instants, `DATE` for calendar-only values, IANA timezone
  preferences for interpretation; never infer DST behavior from a fixed offset.
- Store raw time instants/duration separately from rounded billable values.
  Money uses exact decimals or validated minor units with explicit currency.
- Define transactions, conflict/version checks and lock ordering per mutation.
- Use appropriate check constraints, scoped uniqueness and partial indexes for
  invariants such as running timers once their scope is approved.
- Parameterize values and allowlist identifiers/sort keys. Do not construct SQL
  from arbitrary user field names or accept unbounded pagination.

## Index and query policy

Start with real access paths: workspace/project scope plus stable pagination,
task status/order, membership lookup and user/time interval filters. Add compound
or partial indexes based on queries and `EXPLAIN (ANALYZE, BUFFERS)` on seeded
data; indexes cost write/storage work. PostgreSQL full-text search is the first
search candidate, with permission filters applied to indexed matches.

## Lifecycle and operations

Default product deletion direction is archive/recover, not cascade deletion of
time/billing/history. Each feature must define purge eligibility, authorization,
retention and backup implications before destructive operations are implemented.
There is no global retention policy or legal compliance claim yet.

Migrations and upgrade behavior are authoritative in
[operations](../operations.md#migrations-and-upgrades); backup/restore procedures
are [there too](../operations.md#backup-and-restore-procedure). The baseline
restore was tested only with metadata, not future user records or attachments.
SQLite parity and zero-downtime schema upgrades are not established.
