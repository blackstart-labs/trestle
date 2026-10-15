# Identity and workspaces

Status: **planned specification; not implemented**. Linked requirements: PR-01,
PR-07, PR-08; UC-02; NFR-SEC and NFR-DATA. Bootstrap/enrollment/recovery and exact
session policy remain material decisions to resolve before implementation.

## 1. Purpose

Establish a trusted actor and isolated workspaces so later project, time and
knowledge operations cannot rely on browser-only authorization or guessed IDs.

## 2. Users and primary use cases

Instance operator initializes access; owner manages membership; contributor
switches between their workspaces; user revokes a lost-device session. Operator
and workspace-owner responsibilities are distinct even when one person fills both.

## 3. Functional requirements

- ID-01: establish and revoke an opaque server-side browser session.
- ID-02: associate users with workspaces through scoped memberships.
- ID-03: enforce the [permission matrix](../security/permissions-matrix.md).
- ID-04: account disable/logout/revocation invalidate access without waiting for
  a long-lived browser token to expire.
- ID-05: protect ownership and concurrent membership changes; audit security events.
- ID-06: store timezone/locale preferences without changing historical instants.

Initial M03a should implement the smallest approved bootstrap/login/logout and
workspace-membership slice. M03b adds invitation/recovery lifecycle only after
its channel and abuse controls are specified. No public registration is implied.

## 4. User stories

- As an owner, I can grant/revoke membership without granting instance control.
- As a member of two workspaces, switching context never exposes another
  workspace through guessed URLs or stale lists.
- As a user, I can sign out/revoke sessions and see actionable non-secret errors.
- As an operator, I can initialize/recover access without embedded credentials.

## 5. Acceptance criteria

- AC-ID-01: successful login rotates the session identifier and issues a secure
  cookie; failed login is generic and rate-limited with no account enumeration.
- AC-ID-02: logout/revocation/disable makes the next protected request fail.
- AC-ID-03: a workspace-A-only user cannot read/write workspace-B resources,
  memberships, lists or derived counts by supplying B's identifiers.
- AC-ID-04: a role change cannot race a later unauthorized mutation into a commit;
  two last-owner removals cannot leave a workspace ownerless.
- AC-ID-05: CSRF attempts and credentialed requests from unauthorized origins fail.
- AC-ID-06: keyboard and phone users can sign in, choose an authorized workspace,
  manage permitted membership actions and recover from network failure.
- AC-ID-07: session and password/reset/invite secrets are absent from responses,
  persisted logs, browser storage and test snapshots except intended one-time
  delivery/cookie channels.

These criteria require tests; none currently passes against a product identity
implementation because no such implementation exists.

## 6. Screens and flow

Approved-channel bootstrap → sign-in → authorized workspace chooser → workspace
settings/members; profile/timezone/session settings from the user menu. A return
destination must be a validated same-origin relative path, not an open redirect.
Uninitialized instances show safe setup guidance, never a reusable default login.

## 7. UI states

Loading: bounded submit state; empty: no workspace explains the next permitted
action; error: preserve safe input but never passwords; success: canonical session/
membership state; denied: explain lack of action rights without leaking hidden
membership; expired: preserve non-secret draft context and require reauthentication.
Do not replace a real failed request with a successful-looking shell.

## 8. Entities and business rules

User, session, workspace, membership and security audit event. Sessions identify
users, not a trusted workspace role. Membership is unique per user/workspace.
At least one active owner remains. Workspace/project ownership and customer
billing identity are distinct. Teams/assignees are not implicit grants.

## 9. Proposed persistence contract

Feature migrations will introduce `users`, `sessions`, `workspaces`,
`workspace_memberships` and appropriately restricted audit records. These tables
do not exist yet. Users need stable IDs and a carefully defined normalized login
identity; password credentials use Argon2id if local passwords are selected.
Sessions store a digest of a high-entropy opaque token, expiry/revocation and
actor association; never store raw reusable tokens. Membership needs foreign
keys and unique `(workspace_id, user_id)` plus constrained role values.

Index session-token digest and expiry, normalized login identity, user/workspace
membership lookup. Owner preservation is a transaction/concurrency invariant,
not a simple row CHECK. Define email normalization (do not blindly rewrite local
parts), maximum lengths and account reactivation rules before migration.

## 10. Proposed HTTP contracts

All following product endpoints are **proposed and absent**:

| Interface | Input / successful output | Error requirements |
| --- | --- | --- |
| `POST /api/v1/session` | Bounded login identity/password → safe current-user projection plus Set-Cookie | Generic 401; 422 invalid shape; 429 abuse limit |
| `DELETE /api/v1/session` | CSRF-protected current-session logout → 204 | Idempotent policy, no raw token echo |
| `GET /api/v1/me` | Active session → id, display name, locale/timezone | 401 expired/revoked |
| `GET /api/v1/workspaces` | Active actor → bounded memberships/workspace summaries | No hidden workspace enumeration |
| `GET /api/v1/workspaces/{id}/members` | Authorized admin → bounded safe member projections | Concealed 404 or authorized-action 403 |
| Membership mutation interface | Target user/role plus expected version → updated safe membership | 409 stale/last-owner conflict; authorized actor only |

Membership method/path and creation/invitation/recovery bodies await the decision
gate. Use [shared conventions](../api/api-conventions.md) and
[error contract](../api/error-contract.md); do not expose password hashes,
session digests, private contact fields or reset/invite tokens in projections.

## 11. Authentication and authorization

Opaque browser cookie with Secure in HTTPS deployment, HttpOnly, deliberate
SameSite and path/domain policy; no localStorage credentials. Resolve current
session and membership server-side, with CSRF defenses for cookie-authenticated
writes. Local HTTP development is an explicit exception, not a production default.
Reauthentication requirements for owner/security changes must be approved.

## 12. Validation and edge cases

Handle duplicate/Unicode login identities, very long passwords/inputs, invalid
locale/timezone, disabled users, stale tabs, concurrent login/logout, redirects,
multiple workspaces, inactive final owners and repeated invitation/reset use.
Rate-limit login/recovery by approved account/network policy without logging
raw credentials or creating an easy account-lockout denial of service.

## 13. Concurrency and transactions

Serialize ownership and role changes with related protected writes using a
documented lock/version strategy. Audit membership mutation atomically. Rotation,
revocation and invitation/reset token consumption require retry-safe atomicity.
Test simultaneous owner removal, duplicate membership and revoke-versus-write.
Do not split the permission check from a later unrelated mutation transaction.

## 14. Performance and budgets

Bound bodies, password-hashing concurrency, active sessions where policy requires,
pool usage and member pagination. Choose Argon2id parameters from measured host
profiles and current guidance; do not invent a sub-millisecond login budget.
Measure login/membership p50/p95/p99, rate-limit behavior and resource use using
the [performance plan](../architecture/performance-strategy.md). Numeric budgets
and expiry/retention values must be recorded before implementation sign-off.

## 15. Tests

Unit: normalization/validation/expiry/role decisions. PostgreSQL integration:
membership uniqueness, owner races, session rotation/revocation and audit
atomicity. HTTP/security: login enumeration, CSRF/origin, token disclosure,
cross-workspace IDs, revoked/disabled actors and limits. Browser journeys:
bootstrap-approved flow, sign-in/out, workspace switch, role changes, network
errors and narrow-screen keyboard/touch behavior. Controlled clocks for expiry.

## 16. Accessibility and responsive behavior

Visible labels/autocomplete semantics, readable errors associated with fields,
focus management on submit/expiry, password-manager compatibility and keyboard
operation. Members tables adapt to mobile detail views; actions are not hover-only.
Follow the [design direction](../design/design-principles.md) and manually review
screen-reader, zoom/reflow and touch flows; automated checks are insufficient.

## 17. Integration points

All product domains consume a trusted actor and named scoped operations. Search,
reports, jobs and integrations must respect revocation, not cache stale role
claims. Future API tokens are separate scoped credentials, not browser sessions.

## 18. Migrations and compatibility

Add forward migrations without editing the foundation baseline. Define bootstrap
as a one-time guarded operation, not repeated startup seeding. Test fresh install
and upgrades with two workspaces. Session format or hashing-policy changes require
an explicit expiry/rehash/migration path and downgrade limitations.

## 19. Operations and troubleshooting

No default credentials. Document approved bootstrap/recovery channels and operator
verification before enabling enrollment. Record authentication/audit event codes,
not passwords, cookies or document content. Diagnose clock skew, cookie Secure/
proxy settings, revoked access and database limits with safe guidance. Protected
backups include credentials/session metadata and need encryption/access controls.

## 20. Definition of done and material decision gate

Close and record decisions for (a) bootstrap authority/channel, (b) registration
versus invite-only enrollment and invitation delivery, (c) account/owner recovery,
(d) session absolute/idle lifetimes and rotation, (e) owner transfer and privileged
reauthentication, (f) login normalization and rate-limit policy. These materially
affect the security model and require review before code, not assumptions hidden
in an authentication library.

Then implement the scoped M03a/M03b criteria, migrations, safe HTTP contracts,
server authorization, UI states, security/concurrency/browser tests and measured
resource bounds. Run quality checks, review, update matrix/docs and commit the
verified slice when authorized. A specification with open decisions is not a
completed identity feature or permission guarantee.
