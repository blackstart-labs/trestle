# Module interfaces and ownership

Status: logical ownership plan. Current code is `config`, `database`, `http`,
embedded `assets` and the binary; do not scaffold empty domains.

| Future module | Owns | Exposes to callers | Must not do |
| --- | --- | --- | --- |
| Identity | Users, credentials, sessions, account state | Authenticate actor, establish/revoke session | Grant project access merely because login succeeded |
| Workspaces | Membership/roles, workspace lifecycle | Check named workspace operations, mutate membership transactionally | Let frontend role labels authorize a mutation |
| Projects | Project settings, members and access | Authorized project operations and scoped identity | Read arbitrary task internals to duplicate their rules |
| Tasks/workflows | Tasks, statuses, transitions, ordering, history | Create/edit/transition/move with version expectations | Separate board and table authoritative models |
| Planning | Milestones/cycles, eligibility and progress | Authorized planning operations and rollups | Infer completion from display strings |
| Time tracking | Entries, timer lifecycle, time policies | Start/stop/edit/submission interfaces with invariants | Trust browser elapsed time or float money |
| Reporting | Defined, permission-aware queries | Bounded totals with explicit formula/date scope | Load all tenant records to aggregate in process |
| Knowledge | Markdown, revisions, links, attachments | Authorized document/link/export operations | Expose raw storage paths or unsanitized HTML |
| Integrations | Scoped credentials, external references/delivery | Validate inputs, schedule and inspect deliveries | Log secrets or send unbounded outbound requests |
| Automation | Rules, triggers and execution state | Evaluate allowed events/actions with actor checks | Run arbitrary user code or bypass domain operations |
| Search/notifications | Derived indexes and delivery preferences | Permission-filtered results and user notifications | Treat an index or cached membership as an access grant |

HTTP translates typed requests/errors. Application interfaces own authorization,
validation, domain calls and transaction scope. Persistence implementation stays
near its owning module; business writes cross that module's interface, not its
tables. Optimized read queries may join domains under documented ownership and
permission rules without introducing generic repository abstractions.

External effects eventually cross a durable outbox/execution seam. A rule invokes
the same authorized task transition interface as an interactive user. Avoid
circular module calls; coordinate cross-domain operations explicitly and persist
their ordering/atomicity requirements. Introduce adapter traits only where
behavior really varies, not for every SQL query.
