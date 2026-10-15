# Release checklist

Status: release gate; no production release has met this checklist.

- [ ] Feature matrix accurately states shipped scope and unresolved limitations.
- [ ] Requirement-to-test traceability and manual workflow review complete.
- [ ] Formatting, lint/type checks, all required tests and production builds pass.
- [ ] Server authorization/session/tenant tests pass for every protected domain.
- [ ] Security threat model, dependency advisories/licenses and private reporting
      channel reviewed; complete artifact notices/corresponding source available.
- [ ] Critical concurrency, retry, transaction and migration invariants verified.
- [ ] Frontend keyboard/screen-reader/touch/zoom/mobile acceptance completed.
- [ ] Reproducible performance results meet approved scenario budgets; resource
      limits and query plans recorded, with no unsupported speed claims.
- [ ] Fresh install, upgrade and restored-record invariants checked on supported
      server/deployment profiles; attachments included when present.
- [ ] Production HTTPS, trusted proxy/cookies, DB TLS/roles, ingress limits,
      secret handling and audit/diagnostic controls verified.
- [ ] Backup protection, retention, recovery objectives and restore drill recorded.
- [ ] API/event/schema compatibility and rollback limitations documented.
- [ ] Supported server/build/client platforms distinguished; no inferred native,
      SQLite, offline or Android background-uptime promise.
- [ ] Operations/contribution/security docs and configuration examples match code.
- [ ] Focused commits reviewed; hosted CI results inspected, not inferred locally.
- [ ] Maintainer explicitly authorizes publishing/release/deployment actions.

For an internal development milestone, mark nonapplicable release requirements
as not yet satisfied and do not call it production-ready. See the existing
[foundation verification](../verification.md) for its narrower executed evidence.
