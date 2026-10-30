# Data flow

## Implemented foundation

Environment → typed config → bounded lazy pool → router → health response.
Readiness performs `SELECT 1` with a deadline. Liveness does no database work.
`trestle migrate` separately runs embedded migrations; serving does not do so.
Request diagnostics contain matched route/method/status/duration, not raw URL,
headers, payload or database error text.

## Planned protected mutation

1. Browser sends a typed same-origin request with session cookie, CSRF protection
   and resource version/idempotency input where applicable.
2. Transport validates shape and bounded size; identity resolves an active actor.
3. Application interface opens its transaction, resolves tenant/project ownership,
   checks the operation and validates domain invariants.
4. Mutation and history commit atomically; conflicts return a stable typed error,
   never silently overwrite a newer record.
5. Return the canonical updated resource/version so views reconcile consistently.
6. Where external effects exist, a durable event/outbox record commits with the
   mutation; a bounded executor delivers afterward and records attempts.

Membership revocation and mutations must serialize or use an equivalent verified
version/locking policy so a grant revoked first cannot authorize a later commit.
The identity/feature implementation must test this race, not just read access
before beginning an unrelated transaction.

## Planned reads and derived data

Authorize before fetching the resource projection. Apply allowlisted filters,
stable ordering and bounded pagination in SQL. Aggregates, search, document links
and exports use the same visibility rules. Never expose a forbidden title or
count through an otherwise permission-aware list.

No domain event, job executor, session or protected mutation exists yet.
