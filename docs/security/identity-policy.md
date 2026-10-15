# Approved initial identity policy

Status: **approved requirements; not implemented**. Maintainer approved all three
decision groups on 2026-10-15. This file owns policy values; the
[feature spec](../features/identity-and-workspaces.md) owns acceptance, and the
[permissions matrix](permissions-matrix.md) owns scoped operation grants.

## Provisioning and recovery authority

- One-time bootstrap is an explicit operator CLI command, not startup seeding,
  public HTTP setup or a default login. Password entry is hidden at a terminal;
  never accept passwords in command arguments, echo them or log them.
- Bootstrap is atomic/concurrency-safe; repeated attempts refuse without
  modifying existing accounts or elevating their roles.
- Additional identities are provisioned through the operator CLI until
  invitations are separately specified. An instance identity alone grants no
  membership in an existing workspace.
- No public registration, SMTP dependency, email verification or self-service
  reset in the first identity slice.
- Authorized host/CLI access is the recovery authority. Credential recovery
  revokes **all** existing sessions atomically and is audited without passwords
  or hashes. Workspace/browser roles do not authorize operator CLI access.
- The operator controls host/database credentials and can impersonate a user
  through recovery; tenant privacy does not protect against that operator.
- Document terminal/container commands and failure/rollback behavior with the
  implementation. Automated secret input is not implicitly approved by permitting
  an interactive hidden prompt.

## Login identifiers and passwords

- Local usernames, **not email addresses**: 3–64 ASCII characters drawn from
  letters, digits, dot, underscore and hyphen. Accept uppercase input and
  canonicalize to ASCII lowercase; enforce uniqueness on that representation.
  Do not add unstated leading-character restrictions or silently strip input.
- Local passwords use Argon2id with unique cryptographic salts. No trimming,
  normalization or silent truncation. Define explicit bounded password/body
  limits in the executable contract and support password managers/paste.
- Argon2id parameters are **not yet selected or benchmarked**. Measure memory/
  latency on a documented host profile before sign-off. Bound concurrent hashing
  and waiting work; do not block Tokio with unbounded CPU work.
- Unknown users and wrong passwords have the same public failure. Include
  comparable dummy verification for unknown users; a shared `401` alone is not
  evidence of timing indistinguishability.
- Email login/verification, external identity and MFA are separate future work.
  Initial identity needs no email-normalization or claimed verified-contact model.

## Browser sessions and transport

| Policy | Approved requirement |
| --- | --- |
| Authority | PostgreSQL-backed opaque session; no browser JWT/localStorage credential |
| Idle lifetime | 12 hours since last accepted authenticated activity |
| Absolute lifetime | 7 days from creation; activity cannot extend it |
| Login | Fresh cryptographically random token, never adopt a supplied identifier |
| Revocation | Logout revokes current session; recovery/account disabling revoke all |
| Production cookie | Secure, HttpOnly, SameSite=Lax, deliberately scoped host/path |
| Development exception | Explicit HTTP mode only with loopback public origin and listener |
| CSRF | Protect cookie-authenticated writes and login; SameSite alone is insufficient |

Expiry is server-authoritative, not a browser timestamp. Store a digest of the
reusable authentication token, never the raw credential. Keep credentials out of
URLs, JSON projections, diagnostics, localStorage and error messages. Separately
specify CSRF material delivery/validation; never expose the session credential as
a CSRF token. No permissive credentialed CORS.

Public origin/HTTPS configuration must be explicit; never infer trust or cookie
policy from arbitrary Host/Forwarded headers. Development exceptions fail closed
on non-loopback configuration. Session refresh/revocation and protected mutations
need a documented transactional policy and controlled-clock race tests. Membership
changes take effect server-side, not via stale roles embedded at login. Already
accepted requests are not magically cancelled by another tab's logout; order
commits deliberately and reject the next request after revocation.

## Ownership and privileged operations

- Transfer requires the current owner's password reauthentication within five
  minutes, with freshness established/checked by the server.
- Workspace admins cannot grant themselves ownership or transfer it. Ownership
  changes are named protected operations, not an arbitrary role update field.
- Preserve an active owner during transfer/demotion/removal/account disabling,
  including concurrent operations. Audit security changes transactionally.
- Define target eligibility/acceptance and exact transfer contracts in the
  implementation slice without weakening these approved boundaries.
- Credential recovery is not an undocumented general role-editing interface.
  Any additional break-glass ownership operation needs explicit design/approval.

## Login abuse limits

| Scope | Failure allowance | Window |
| --- | --- | --- |
| TCP source IP + canonical username | 5 | 15 minutes |
| TCP source IP across usernames | 30 | 15 minutes |

No permanent account lockout. A correct password does not bypass an active
limit. Atomic accounting/reservations must prevent concurrent overshoot and keep
retained bucket state bounded. Never retain raw passwords, tokens or bodies.
Use generic `429` with bounded retry indication, without revealing account
existence. Specify successful-attempt accounting, crash/retry and cleanup behavior
in the executable contract before implementation.

Use the actual TCP peer by default, not X-Forwarded-For/Forwarded. Proxy/NAT users
may share a bucket. Document this rather than promising per-end-user enforcement
through an untrusted header. These limits do not stop distributed attacks and do
not replace ingress connection/request-size/time controls. No Redis is required.

## Remaining engineering work

The material policy gate is closed. Still required: concrete CLI/HTTP/config and
input limits; hashing measurements; schema/constraints/lock ordering; CSRF/session
handling; bounded limiter; audit retention/exposure; UI states, tests and evidence.
Agree public test seams before test-first work. Do not mark identity implemented
or create speculative project/time/document tables while doing this.
