# Performance strategy

Status: measurement plan, no product latency or memory claim.

## Deployment profiles to baseline

- Small Linux instance: document CPU model/cores, RAM limit, storage and network;
  measure app and PostgreSQL separately and together.
- Team instance: same declared hardware with two-workspace fixtures and concurrent
  readers/writers, including permission changes and background work when present.
- Browser clients: desktop and physical Android/iOS where available, plus
  documented phone/tablet viewport and slow-network/CPU emulation.
- Termux server: separate feasibility experiment, never inferred from mobile web.

Select concrete hardware and record limits before approving numeric budgets.

## Repeatable measurements

Record commit, build profile/toolchains, DB/container versions, data generator
seed, rows per domain, payload sizes, concurrency, arrival rate, duration, warmup,
cache state and sample count. Measure cold start separately from warm requests.
Include p50/p95/p99, errors/timeouts, CPU, RSS/container memory, database query
latency/count, frontend transferred/compressed bytes, script execution, first
useful render and input/save/reconciliation timing. Separate UI feedback from
durable server acknowledgment.

Start with health/connectivity baselines, then project lists/task details, board
loads/moves, table edits, search, timer operations, reports and job execution as
they become real. Use at least small/medium/large fixture sizes selected and
published with each benchmark; never compare different workloads as equal.

## Design constraints before numerical budgets

Bound SQL and response sizes; keyset pagination for growing lists; inspect query
plans and N+1 behavior. Code-split editors/graphs, avoid duplicate requests and
polling, preserve interaction responsiveness with explicit errors/conflicts.
Do not cache permissions without a tested revocation/invalidation policy.

Initial acceptance is reproducible measurement, not an invented speed target.
Approve scenario-specific budgets after the baseline, store results with the
benchmark harness, and reject regressions against a declared tolerance/sample
policy. Performance changes require correctness and authorization tests too.
