---
status: accepted
---
# Local identity with operator-controlled provisioning and recovery

On 2026-10-15 the maintainer approved CLI bootstrap/provisioning/recovery,
username/password login, bounded revocable server-side sessions and explicit
ownership/reauthentication rules. Authoritative values and trust boundaries are
in [identity policy](../security/identity-policy.md). These are approved
requirements, not implemented authentication capabilities.

This avoids requiring SMTP or an external identity service for initial
self-hosting. It deliberately trusts the host operator to provision/recover
accounts. Recovery revokes sessions; application workspace roles do not grant
access to the host/CLI. Users cannot self-register or request email resets.

Use local usernames instead of implying an email address has been verified.
Invitations, external identity, MFA and recovery codes need separate specs.
Recovery cannot make content confidential from an operator controlling storage.

Login limits initially use the TCP peer, never untrusted forwarded headers.
This resists header spoofing but shares a bucket behind proxies/NAT and does not
stop distributed guessing. Document/test those limits; do not advertise complete
abuse protection. Trusted-proxy support requires a deliberate follow-up design.
