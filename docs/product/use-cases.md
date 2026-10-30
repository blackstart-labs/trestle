# Primary workflows

Status: planned except health, migration and operational checks.

| ID | Workflow | Success condition | Prerequisites |
| --- | --- | --- | --- |
| UC-01 | Operator initializes and diagnoses an instance | Build/run/migrate and distinguish live from ready | Backend foundation (implemented) |
| UC-02 | Owner signs in and manages workspace access | Revocable session; membership changes take effect on the next request | Identity, workspace permissions |
| UC-03 | Member creates and completes project work | Same task identity and semantic completion across views | UC-02, projects, tasks/workflows |
| UC-04 | Two members edit or move the same task | No silent lost updates; losing edit receives a conflict and reconciliation path | UC-03, versioned mutations |
| UC-05 | Member tracks work and reconciles a timesheet | Retry-safe timer operations and explainable period totals | UC-03, time entries, timezone rules |
| UC-06 | Member writes and links a project note | Safe Markdown; readable exported content; stable links after rename | UC-02/03, knowledge |
| UC-07 | Maintainer links a pull request and triggers a notification | Scoped credentials, authorized events, visible delivery failure | Stable tasks/events, integration delivery |
| UC-08 | Operator restores an installation | Restored records and grants verified before cutover | Backups and feature-specific recovery tests |

For UC-03, a mobile user can select a status through a menu instead of dragging.
For UC-04, hidden/filtered board records retain their order and authorization.
For UC-05, elapsed time comes from persisted instants, never a browser clock.
For UC-06, revoked access must also revoke search/backlink visibility.

The backend-only health checks are not a completed implementation of UC-02–08.
