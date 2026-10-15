---
status: accepted
---
# One application and PostgreSQL first

Trestle needs transactions and consistent authorization across projects, time
and knowledge without requiring a small operator to run a service fleet. Use
one Rust application crate with logical domain modules and PostgreSQL through
SQLx; add crates or services only for demonstrated ownership/build/runtime needs.

Compared with microservices this reduces deployment and distributed-consistency
cost, at the price of a shared process and database failure domain. Compared
with immediate PostgreSQL/SQLite parity, one target permits stronger tested
constraints and a smaller migration/query surface. SQLite remains a separate
future decision, not a compatibility claim. This direction was approved after
the repository audit and is implemented only for the foundation so far.
