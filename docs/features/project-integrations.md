# Project integrations

In **Settings → Integration**, select an active Project in the current Workspace.
Workspace owners and admins can connect one destination per provider per Project:

- Telegram: a BotFather bot token and numeric chat ID or public @channel username.
  Add the bot to the chat and grant posting permissions; start the bot before
  using a private conversation. Channel bots need permission to post messages.
- Slack: an incoming webhook URL from a Slack app with Incoming Webhooks enabled.
  Standard Slack and GovSlack webhook hosts are supported.
- Discord: the incoming webhook URL from channel settings → Integrations → Webhooks.

Integration has a dedicated Settings view at `#integration`. The Project context
rail shows the event scope and shared metadata beside separate provider panels;
it moves above the providers when available space or enlarged text needs it.
Provider marks always appear with their names, and connection state comes from
saved configuration and the latest delivery attempt.

Save stores credentials without sending a message. **Test** explicitly sends a
test notification using the saved connection. **Replace** requires credentials
again; existing secrets cannot be retrieved. **Disconnect** confirms removal of
credentials and stops future notifications; previously posted messages remain.
Each Project has independent configuration. Members cannot read or manage it.
Archived Projects retain credentials but do not send notifications; credentials
can still be removed through the API.

## Events and delivery

Task and Subtask creation, task property updates through PATCH, and task deletion
send notifications after the database mutation succeeds. Creation and updates
include Project name/key, task key/title, and Status category. Deletion includes
Project name/key and task key. Comments, labels, assignees, and bulk Status remaps
do not generate notifications. Messages use plain text and disable Discord mentions
and Slack link/mention parsing.

Delivery is best effort: no durable queue, replay, or automatic retries. Up to
three provider requests run concurrently within a shared three-second deadline;
recording results allows up to one further second. A failed send never rolls back
a task mutation. The settings panel shows the latest attempt and a sanitized
failure message; reopen it to refresh after task changes. Test requests within
five seconds of the previous attempt are rejected. This is a convenience cooldown,
not an instance-wide abuse limiter; apply request rate limits at the reverse proxy.
Providers may accept a message before a timeout, so a failed attempt does not prove
that no message was posted.

## API and persistence

All routes require a signed-in Workspace owner/admin and the normal CSRF check:

| Method | Route under `/api/v1/workspaces/{workspace_id}/projects/{project_id}` |
| --- | --- |
| GET | `/integrations` |
| PUT | `/integrations/{telegram\|slack\|discord}` with `{ "secret": "…", "chat_id": "…" }` (chat ID only for Telegram) |
| DELETE | `/integrations/{provider}` |
| POST | `/integrations/{provider}/test` |

Responses omit secrets. A composite foreign key prevents cross-Workspace project
configuration. Apply the forward migration with `trestle migrate` before starting
the new binary. No new environment variables or dependencies are required.

Credentials are stored as plaintext in PostgreSQL, accessible to database operators
and backups. Protect database access, disks, backups, and HTTPS to this instance;
rotate credentials at the provider if they are exposed. They never enter browser
storage, API responses, application request logs, or delivery error text. Outbound
requests use only validated provider HTTPS endpoints; proxies and redirects are
disabled, and the actual dialed IP must be public. The browser CSP is unchanged.

Provider contracts: [Telegram sendMessage](https://core.telegram.org/bots/api#sendmessage),
[Slack incoming webhooks](https://docs.slack.dev/messaging/sending-messages-using-incoming-webhooks/),
[Discord execute webhook](https://docs.discord.com/developers/resources/webhook#execute-webhook).
