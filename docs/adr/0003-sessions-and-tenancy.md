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

Bootstrap, enrollment/recovery, session expiry and ownership policy were resolved
by [ADR-0004](0004-local-identity-and-operator-recovery.md) and its linked policy.
Concrete implementation and security/concurrency evidence remain pending. These
ADRs record approved direction, not completed authentication functionality.
