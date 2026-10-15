# Product threat model

Status: planned product threat analysis. Implemented foundation safeguards and
limits are authoritative in the [current threat model](../threat-model.md).
No item below implies its product control is implemented.

## Assets and trust

User credentials/sessions, workspace content, private project relationships,
time/billing history, documents/attachments, integration secrets, execution
records and backups. Browsers, uploaded content, destination URLs, webhook
payloads, automation configuration and third-party responses are untrusted.
Operators control host/storage and are outside application-level confidentiality
guarantees. Workspace administrators' content access is explicit in the
[permission matrix](permissions-matrix.md).

| Threat | Planned controls | Required evidence before feature release |
| --- | --- | --- |
| Session theft / credential stuffing | Opaque revocable cookies, rotation/expiry, Argon2id when applicable, bounded hash concurrency, abuse limits | Fixation/revocation/expiry/enumeration/limit tests |
| Broken access control / tenant leakage | Named operations, scoped queries, composite integrity, no frontend-only grants | Two-workspace/role/guessed-ID and revoked-membership races |
| Injection | Typed bounded input, parameterized values, allowlisted query dimensions | Malicious/filter/sort/oversized-input tests |
| XSS / Markdown | Sanitized rendering and safe URL schemes, content policy, no raw HTML trust | Stored/reflected payload and export/render tests |
| CSRF / hostile origins | Verified CSRF strategy and credential origin policy | Cross-origin write and cookie-policy tests |
| SSRF / hostile integrations | Destination restrictions, address resolution/rebinding controls, egress policy, bounded timeouts/retries | Local/private/metadata address and redirect/rebinding tests |
| Malicious uploads / path traversal | Bounded type/size, random storage keys outside web root, authorized download, metadata limits | Traversal, oversize, active-content, unauthorized fetch tests |
| Secret / backup disclosure | Redacted logs/projections, protected keys/backups, limited credential scopes | Response/log/artifact review and protected restore drill |
| Automation abuse | Allowed triggers/actions, actor authorization recheck, depth/dedup/concurrency/retry limits | Loops, duplicate events, revoked actor, runaway schedule tests |
| Concurrent authorization change | Transaction ordering/locks or equivalent verified version policy | Revoke-versus-write/export/job execution tests |
| Supply chain / insecure deployment | Lockfiles/digests, advisory/license review, non-root, HTTPS/DB roles and safe proxy trust | CI evidence, dependency review and deployment validation |
| Operational loss / resource exhaustion | Database constraints, bounded work, backups, graceful interruption and explicit recovery | Concurrency, outage, restore, timeout and load tests |

Incoming webhooks require signature verification where supported and replay/
idempotency rules even when signatures are valid. Outbound destinations require
SSRF protection independently of incoming signature checks. Sensitive audit
history needs restricted readers and a retention policy, not unrestricted logs.

Update this analysis with feature-specific data flows and residual risks. Do
not choose an authentication library, Markdown renderer or queue and infer the
controls are therefore complete. Before first exposure beyond trusted local
development, verify actual implemented controls and the release checklist.
