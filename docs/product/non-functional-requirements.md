# Non-functional requirements

Status: policy and acceptance gates, not a report of implemented safeguards.

| ID | Requirement | Verification gate |
| --- | --- | --- |
| NFR-SEC | Every protected read/write/aggregate/export is server-authorized; private data stays private | Two-workspace negative tests and membership-revocation races for every domain |
| NFR-DATA | Critical integrity is database-enforced and multi-step mutations are atomic | Constraint, rollback, migration and concurrency tests on PostgreSQL |
| NFR-PERF | Everyday work is responsive with bounded requests and queries | Baseline methodology plus approved p50/p95/p99/resource budgets, not guessed targets |
| NFR-OPS | One core app plus PostgreSQL; repeatable deploy/upgrade/recovery | Direct/container smoke tests and restored-data verification |
| NFR-A11Y | Target WCAG 2.2 AA for shipped workflows | Semantic controls, automated checks, manual keyboard/focus/screen-reader/touch review |
| NFR-MOBILE | Core actions work without dragging or a desktop-width table | Phone portrait/landscape and tablet end-to-end journeys |
| NFR-PORT | Markdown and exports remain understandable outside Trestle | Import/export round-trips, attachment/link metadata documentation |
| NFR-OBS | Useful operational diagnostics without secret/content leakage | Redaction tests, failure drills and bounded diagnostic data |
| NFR-COMPAT | Support is explicit per server/build/client platform | Published tested matrix; no inferred SQLite/native/Termux/offline support |
| NFR-MAINT | Changes have focused specs, migrations, tests and evidence | Definition-of-done review; versioned contracts and release checklist |

The current enforced pool/acquisition/statement/shutdown bounds are documented
in the root README. These are safeguards, not throughput or latency guarantees.
No zero-downtime, availability SLA, recovery objective or cross-product
performance claim is established yet. Approve numeric service objectives only
after deployment profiles and workloads are measured.
