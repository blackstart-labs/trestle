# Testing strategy

Status: foundation tests executed; product test obligations below are planned.
Executed results belong in [verification](../verification.md), not in this policy.

## Test surfaces

Test public application/HTTP interfaces and domain rules, not private functions
or mocks of SQL behavior. Foundation seams were explicitly agreed: configuration,
health/error responses, and migrations on disposable PostgreSQL. Agree new
feature seams before test-first implementation. Use red→green vertical slices
where practical, then review/refactor without weakening failed assertions.

| Layer | Required evidence |
| --- | --- |
| Domain | Invariants, state transitions, validation, permission decisions, exact time/money calculations |
| PostgreSQL | Constraints, transactions/rollback, fresh/upgrade migrations, concurrent numbering/order/timers/ownership, real query behavior |
| HTTP | Typed contracts, limits, fixed errors, actor/session/CSRF behavior, role and tenant isolation |
| Frontend | Interaction/form states, keyboard/focus, optimistic failure/conflict recovery, type/build checks |
| End to end | Real authorized workflow with controlled fixtures, mobile/touch alternatives, denied/expired/network-failure paths |
| Security | Cross-workspace IDs, role escalation, revoked actors, malicious Markdown/uploads, SSRF/signatures when those features exist |
| Operations | Direct/container startup, migrate, outage recovery, signals, restore and upgrade |
| Performance | Seeded workloads, query plans, distributions/resources and declared regression policy |

Use deterministic seeds, controlled clocks, isolated SQLx databases and a
disposable `CREATEDB` role. Never run tests with production credentials. Missing
prerequisites must fail required suites explicitly, not silently skip coverage.
Property-based tests are useful for ordering, time intervals, graph cycles and
rounding once those rules exist; no property-test dependency is required now.

## Acceptance traceability

Feature requirement → acceptance criterion → named test/journey → executed
verification record → feature-matrix status. Record denied and failure behavior
alongside success. A foundation health test is not an authorization test; a
configured CI file is not a hosted pass; a screenshot is not an end-to-end test.

Run relevant single suites during development, formatting/type checks regularly,
and the full suite before milestone completion. Explain environmental limits and
flakiness; repair the cause rather than deleting/skipping tests to make CI green.

Documentation-only changes require local-link/structure/status consistency and
diff review. They do not create new product-test evidence. Use maintained
documentation tooling only when it adds value; no framework is needed for a
small Markdown tree.
