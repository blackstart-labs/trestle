# Running timers and task time

Trestle records task time in two ways: a manual time log and a server-owned
running timer that becomes a time log when stopped. This document describes the
running timer. Manual logs are unchanged.

## What the server owns

- **One running timer per user.** A partial unique index enforces it in
  PostgreSQL, not in a browser tab.
- **The start time.** Elapsed time is computed from the server's clock, so a
  wrong device clock or a suspended laptop cannot change the recorded hours.
- **The stop.** Stopping computes hours, writes the time log and marks the timer
  stopped in one transaction. The timer keeps the resulting log id.

The browser only presents state and sends commands. It does not create time
logs from local elapsed time.

## Workflows

### Start

Starting a task creates a timer owned by the signed-in user. Starting the same
task again returns the existing timer instead of creating another.

Starting a different task while a timer runs returns `409 timer_active` with the
running timer. The editor asks before replacing it; replacement discards the
previous unlogged interval in the same transaction. Guest members cannot start
timers, and archived Projects refuse new timers until restored.

### Stop and log

Stopping sends the timer id and the current note. The server:

1. locks the timer row;
2. computes elapsed hours from `started_at`, rounded to two decimals with a
   minimum of `0.01`, and refuses an interval longer than 240 hours;
3. writes one time log dated by the timer's start in the user's timezone;
4. stores the log id on the timer and marks it stopped.

A retried stop — lost response, double click, another tab — returns the original
time log. It never creates a second log. If the server committed but the
response was lost, the editor keeps showing the timer until the retry succeeds,
then clears it.

A timer older than 240 hours is not logged automatically. The editor explains
that the interval is too long, and the user can log it manually and discard the
timer.

### Note

The note is saved 600 ms after typing stops and again with the final stop
request, so the last words are not lost when the connection drops. Notes are
plain text, up to 2,000 characters.

### Discard

Discard ends a running timer without writing a time log. Discarding twice is a
no-op. A timer that already produced a log cannot be discarded; the log must be
deleted separately if it was wrong.

### Other tabs and devices

The timer is read from the server when the session starts and when the tab gains
focus or becomes visible. Same-browser tabs also announce changes over a
`BroadcastChannel`, so starting or stopping in one tab updates the others.
Another device sees the change on its next load, focus or reload.

### A timer from the previous version

The earlier build stored a timer in `localStorage` only. That value is detected
but **not imported**: an untrusted local interval must not silently become
server time. Log that work manually if it was real, then dismiss the old value.
`LiveTimer.dismissLegacy()` removes it.

## API

All routes require a signed-in session. `POST` and `PATCH` also require the CSRF
header. Errors use the normal `{error: {code, message}}` shape.

| Method | Path | Body | Success |
| --- | --- | --- | --- |
| GET | `/api/v1/timer` | — | `{timer: Timer \| null, server_now}` |
| POST | `/api/v1/timer` | `{task_id, note?, replace_active?}` | `201 {timer, server_now}` |
| PATCH | `/api/v1/timer` | `{timer_id, note}` | `{timer}` |
| POST | `/api/v1/timer/stop` | `{timer_id, note?}` | `{time_log, spent_hours}` |
| POST | `/api/v1/timer/discard` | `{timer_id}` | `{status: "discarded"}` |

`Timer` is
`{id, workspace_id, project_id, task_id, task_key, task_title, note, started_at}`.

Relevant failures:

- `403` — guest member tries to start a timer.
- `404` — unknown task, unknown timer, or a task outside the caller's Workspace.
- `409 timer_active` — another timer is running; the response includes it.
- `409` — the Project is archived, the timer was discarded, or its time is
  already logged.
- `422` — missing fields, or the timer exceeded 240 hours.

## Limits and non-goals

- One active timer per user; there is no parallel timer or pause/resume.
- Hours are rounded to two decimals; the minimum is `0.01`.
- Notes are at most 2,000 characters.
- Timers belong to a user, not a Workspace membership. Removing someone from a
  Workspace does not delete an already-running timer; it can still be stopped
  and logged.
- A logged entry can be deleted by the person who logged it or by a Workspace
  owner/admin. The delete and its `time_log.deleted` activity record commit
  together, and other members do not see the control.
- Timesheets, approvals, work Activities, Customers, rates and billing are not
  implemented.

## Verification

From the repository root:

```bash
# Go API and PostgreSQL behaviour (needs DATABASE_URL)
go test ./internal/httpapi -run TestTimer -count=1
go test -race ./...

# Frontend unit tests
cd web && npm test

# Fixture-backed browser journeys, including the running timer banner
cd web && npm run test:e2e

# Real browser → API → PostgreSQL timer journey (needs a disposable
# trestle_e2e_* database; see docs/boards.md for the setup pattern)
TRESTLE_E2E_DATABASE_URL='postgres://…/trestle_e2e_timer?sslmode=disable' \
  npm run test:e2e:real
```

The Go tests cover start/replace, note updates, discard, idempotent stop,
concurrent stop, guest and non-member access, archived Projects and the logged
date in the user's timezone.
