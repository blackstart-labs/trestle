# Domain relationships

Status: logical model for incremental specifications, not a pre-created schema.
Definitions belong in [CONTEXT](../../CONTEXT.md); do not redefine them here.

```text
User ──< WorkspaceMembership >── Workspace ──< Project
User ──< ProjectMembership >────────────────────┘
Project ──< Status
Project ──< Task ──< Comment / History
Task ──< TaskAssignee >── User
Task ──< TaskLabel >── Label (project scoped)
Task ── optional parent Task (same project, acyclic)
Task ── optional Milestone / Cycle (same project)
User + Project ──< TimeEntry ── optional Task
Workspace ──< Document ── optional Project links
Document ──< DocumentLink >── Document
Workspace ──< Integration ──< ExternalReference
Workspace ──< AutomationRule ──< Execution
```

## Invariants to carry into feature specifications

- A project has one workspace; its tasks/statuses/labels/planning relations
  cannot point into another project. Subtasks cannot form cycles.
- Milestones are planning entities, not special task rows. Start with an
  optional milestone association; decide multi-milestone semantics before
  expanding it. Cycles and team membership are not implemented yet.
- Task IDs remain stable across views. Human project task keys supplement IDs;
  allocation must be unique and transactional, with gaps permitted. Per-project
  counters serialize same-project allocations; do not call them lock-free.
- A user's presence in an instance is not workspace membership. Guests, teams,
  customer relationships and assignee lists are not implicit access grants.
- A time entry belongs to one user/project and optionally a same-project task.
  A timesheet groups entries; it is not another authoritative duration source.
- An estimate, measured duration and billable/rounded duration are distinct.
  Historical applied rates must not change when a future default rate changes.
- Document identity survives rename/move. Human titles or filesystem paths alone
  cannot safely resolve links when titles collide or users lose access.
- Structured collection records are not tasks by default. Unification requires
  an explicit future domain decision, not a generic JSON table.
- Search results, backlinks, reports, events and notifications inherit the
  resource's access policy. Derived data is not an authorization escape hatch.

Concrete feature schemas must resolve cardinalities, lifecycle and retention
before migration. Do not introduce every association in this diagram at once.
