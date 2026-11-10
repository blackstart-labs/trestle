# Threat Model & Security Policy

This document defines Trestle's security architecture, trust boundaries, access control policies, and operational safeguards.

---

## 1. Trust Boundaries & Actors

```text
┌────────────────────────────────────────────────────────┐
│                   Untrusted Network                    │
│           (Callers, Web Browsers, Webhooks)            │
└───────────────────────────┬────────────────────────────┘
                            │ HTTPS / HTTP
┌───────────────────────────▼────────────────────────────┐
│                    Reverse Proxy                       │  Boundary: TLS termination,
│             (Nginx, Caddy, Cloudflare)                 │  rate-limiting, DDoS defense
└───────────────────────────┬────────────────────────────┘
                            │ Loopback / Private Network
┌───────────────────────────▼────────────────────────────┐
│                 Trestle Application                    │  Boundary: Auth, CSRF,
│                  (Go net/http Server)                  │  Tenant isolation, RBAC
└─────────────┬───────────────────────────┬──────────────┘
              │                           │
              │ SQL                       │ Outbound Webhooks (SSRF Guard)
┌─────────────▼───────────────┐ ┌─────────▼──────────────┐
│     PostgreSQL Database     │ │ External Services      │
│     (Workspaces, Tasks)     │ │ (Telegram, Discord...) │
└─────────────────────────────┘ └────────────────────────┘
```

### Actor Categories:
- **Host / Cluster Operator**: Controls binary, environment variables, PostgreSQL credentials, and filesystem. Has root authority over local data.
- **Workspace Owner**: Has full administrative authority over a workspace, including billing, member management, and ownership transfer.
- **Workspace Admin**: Can manage projects, members, and project integrations, but cannot remove/demote the owner or transfer ownership.
- **Workspace Member / Contributor**: Can view and create tasks, participate in discussions, update assigned work, and view projects in their workspace.
- **Workspace Guest / Viewer**: Read-only participant with explicit grants to specific projects.
- **Untrusted Caller**: Any unauthenticated network client submitting HTTP requests to the public listener.

---

## 2. Authentication & Identity Policy

### Credential Handling
- **Usernames**: 3–64 ASCII alphanumeric characters plus `.`, `_`, and `-`. Stored and matched case-insensitively using lowercase canonicalization.
- **Argon2id Hashing**: Passwords are never stored in plaintext. They are salted with unique cryptographic randomness and hashed using Argon2id (`time=1`, `memory=64MB`, `threads=4`).
- **Timing Leak Prevention**: Login verification uses constant-time comparison. When an unknown username is supplied, a dummy verification pass is executed to ensure equivalent response latency.

### Sessions
- **Opaque Session Tokens**: Cryptographically secure 256-bit random tokens stored as SHA-256 digests in PostgreSQL (`sessions` table).
- **Cookie Flags**: `HttpOnly`, `SameSite=Lax`, `Path=/`, and `Secure` (in TLS production environments).
- **Session Lifespan**:
  - Idle timeout: 12 hours of inactivity.
  - Absolute timeout: Configured via `SESSION_TTL_HOURS` (default 14 days).
- **Revocation**:
  - Logout revokes the current session immediately.
  - Password change or operator reset revokes all active sessions for that account.

### CSRF Protection
- State-changing operations (`POST`, `PUT`, `PATCH`, `DELETE`) require an `X-CSRF-Token` HTTP header matching the value stored in the `trestle-csrf` cookie.
- Validated via constant-time string comparison (`subtle.ConstantTimeCompare`) before routing to handlers.

---

## 3. Permissions & RBAC Matrix

| Operation | Workspace Owner | Workspace Admin | Workspace Member | Workspace Guest |
| --- | :---: | :---: | :---: | :---: |
| Transfer workspace ownership | **Yes** (Owner only) | No | No | No |
| Delete workspace | **Yes** (Owner only) | No | No | No |
| Add / remove workspace members | **Yes** | **Yes** (cannot demote owner) | No | No |
| Create / archive projects | **Yes** | **Yes** | No | No |
| Configure project integrations | **Yes** | **Yes** | No | No |
| Manage custom statuses & milestones | **Yes** | **Yes** | **Yes** | No |
| Create / edit tasks | **Yes** | **Yes** | **Yes** | No |
| View tasks & project contents | **Yes** | **Yes** | **Yes** | Granted projects only |
| Add comments | **Yes** | **Yes** | **Yes** | Granted projects only |
| Delete own comments | **Yes** | **Yes** | **Yes** | **Yes** |
| Delete any comment | **Yes** | **Yes** | No | No |
| Revoke own sessions | **Yes** | **Yes** | **Yes** | **Yes** |

---

## 4. Tenant Isolation Invariants

1. **Workspace Scoping**: Every database lookup, insertion, update, and deletion is constrained by `workspace_id`.
2. **Concealed 404s**: Probing IDs from another tenant returns `404 Not Found`, identical to non-existent IDs. `403 Forbidden` is only used when the resource's existence within the user's workspace is known but the action exceeds role permissions.
3. **Task Key Uniqueness**: Sequential task numbering (`project.task_counter`) is updated atomically in transactions, scoped strictly to the parent project.

---

## 5. Webhook Security & SSRF Protection

Trestle allows projects to dispatch notifications to Discord, Slack, and Telegram:
- **SSRF Mitigation**: Webhook URLs are strictly validated:
  - Scheme must be `https://`.
  - Non-routable IP addresses (RFC 1918 private ranges, loopback `127.0.0.0/8`, link-local `169.254.0.0/16`, IPv6 loopback `::1`) are blocked during DNS resolution.
  - Custom ports, userinfo, and query parameter overrides are rejected.
  - Redirects and HTTP proxies are disabled on outbound webhook clients.
- **Write-Only Storage**: Webhook tokens and secrets are write-only through the API; they are never reflected back in API responses.
- **Notification contents**: Enabling a Project integration sends its project/milestone name, task title, status, priority, assignee display names and a description excerpt to that provider. Legacy HTML descriptions are converted to text; description excerpts are limited to 1,000 Unicode characters and assignee names to 200. Longer content is explicitly marked as truncated. Telegram HTML and Slack control characters are escaped, and Discord mentions are disabled. Comments and credentials are not included. Enable integrations only for destinations allowed to receive this Project's content.

---

## 6. Content Security Policy (CSP) & XSS Defenses

Trestle enforces a strict Content Security Policy header on all HTML responses:
```http
Content-Security-Policy: default-src 'none'; script-src 'self'; style-src 'self'; font-src 'self'; img-src 'self' blob:; frame-ancestors 'none'; base-uri 'none'; form-action 'self'
```

- **No Remote CDNs**: All scripts, styles, and fonts (Mona Sans) are self-hosted and bundled.
- **No Inline Styles**: The CSP explicitly forbids `style="..."` attributes. Dynamic properties are passed through Svelte CSS variables (`style:--status-color`).
- **Markdown Sanitization**: User-submitted markdown (descriptions, comments) is rendered with an allowlist-based HTML sanitizer. Raw `<script>`, `<iframe>`, and dangerous attributes (`onclick`, `javascript:`) are stripped.
