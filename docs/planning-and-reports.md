# Planning, reports and paused timers

## Planning

Workspace → Planning exposes Goals, Objectives, Key results, Cycles and Modules.
Create a goal, open it and create an objective; create key results from its
objective. A key result stores a positive target and nonnegative actual value.
The measured rollup is the equal-weight average of descendant key results,
each capped at 100%. Archived/cancelled descendants are excluded. No key results
means no measurement, rather than a fabricated zero. Task counts are separate
and reflect canonical task statuses; cancelled tasks do not count as completed.

Set owners, status, optional project, dates and Markdown descriptions. Cycles
and modules require an active project. Save before linking tasks. Assigning a
task to a cycle/module replaces its previous assignment of that kind; the UI
explains this action. Links to goals/objectives/key results may coexist.
Remove linked tasks before changing a plan's project. Projects, owners and
parents must belong to the current Workspace. An archived parent must be
restored or replaced before editing its child.

Guests may read. Members may create/edit/link. Owners/admins additionally
archive/restore. Archived plans keep their links and cannot be edited until
restored. Concurrent edits return 409 and retain the draft. Search/list and
relation pages return 50 records with `has_more` and offsets.

Routes below require the existing session and CSRF token on mutations:

| Method | Workspace-relative route | Response/purpose |
|---|---|---|
| GET | `/planning?kind=goal&q=launch&archived=false&offset=0` | `items`, `has_more`, `offset` |
| POST | `/planning` | Create using UUID `creation_key`; returns item |
| GET | `/planning/{id}` | `item`, nullable `measurement_progress` |
| PUT | `/planning/{id}` | Update editable fields with current `version` |
| GET | `/planning/{id}/children?offset=0` | Active immediate children |
| GET | `/planning/{id}/tasks?offset=0` | Linked canonical tasks |
| PUT | `/planning/{id}/tasks/{task_id}` | Associate/move; idempotent |
| DELETE | `/planning/{id}/tasks/{task_id}` | Remove association |

The editable body contains `kind`, `title`, `description`, `status`, nullable
`parent_id`, `project_id`, `owner_id`, `start_date`, `due_date`, numeric `target`
and `actual`, and `archived`. Target defaults to 100 in the UI and must be
positive even for non-key-result items; both numeric values are at most 1e9.
Titles are at most 160 characters; descriptions at most 15,000.

## Time reports

Workspace → Time reports provides date, project and member filters, recorded
hours, total entry count, paginated entries and CSV. Week starts Monday;
Today/This week/This month use the browser's local calendar. The underlying
recorded dates retain the existing time-log semantics. Active/paused timers
are not recorded hours until stopped. Edits in filters require Apply; paging
and export use the applied query.

`GET /api/v1/workspaces/{id}/time-report` accepts `from` and `to` as YYYY-MM-DD,
optional `project_id`, `user_id`, `offset`, and `format=csv`. Maximum range is
367 inclusive dates. JSON includes `entries`, `total_hours`, `total_entries`,
`has_more`, `offset`. Pages contain at most 100 entries; CSV exports at most
10,000. Narrow filters if export exceeds this limit. Owners/admins see the
Workspace; members are always scoped to themselves, regardless of `user_id`.
Guests cannot access reports. CSV formula prefixes are escaped.

## Pause/resume

`POST /api/v1/timer/pause` accepts `timer_id` and boolean `paused`. Explicit
state makes retries idempotent. Only the owning user with a non-guest Workspace
membership may pause/resume. The timer stays `active` for the existing unique
index, so starting another task follows the existing replacement workflow.
The response includes `timer` and `server_now`; timer data gains nullable
`paused_at` and cumulative `paused_seconds`. A paused header readout freezes.
Stop subtracts pause duration and logs once; the existing minimum 0.01 hours,
240-hour bound and start-date timezone behavior remain in place.

These additions build on existing schema and domain boundaries. See
`IMPLEMENTATION_STATUS.md` for the explicit untested final-revision status.
