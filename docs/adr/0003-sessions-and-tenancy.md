---
status: accepted
---
# Workspace tenancy and revocable browser sessions

A user may belong to multiple workspaces. Treat workspace as the tenant and
projects as its subordinate scope; customer billing identity is separate. Use
opaque, revocable server-side browser sessions rather than long-lived browser
JWTs whose revocation would need another mechanism anyway.

This costs a server-side lookup but supports logout, account disabling and
membership changes with explicit semantics. Do not store browser credentials in
localStorage. Future scoped integration tokens are a different credential type,
not a reason to weaken browser sessions. Membership checks remain server-side
for every protected interface; database constraints prevent cross-tenant links.

Bootstrap, enrollment/recovery channels, session expiry and ownership transfer
remain decision gates in the identity feature specification. This ADR records
the already-approved direction, not a completed authentication implementation.
