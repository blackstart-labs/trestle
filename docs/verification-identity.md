# Identity and workspaces verification (M03a.1)

Date: 2026-11-01. Linux x86_64, Node 24.22.0/npm 11.19.1, Rust 1.99.0, PostgreSQL 18-bookworm.
Scope and agreed seams: [Identity and workspaces](features/identity-and-workspaces.md), [Identity policy](security/identity-policy.md), [ADR-0004](adr/0004-identity-and-workspace-isolation.md).

## Summary of deliverables

- **Database migration**: `migrations/202611010002_identity.sql` adds `users`, `sessions`, `workspaces`, `workspace_memberships`, and `security_audit_events` tables with strict constraints, indexes, and cascades.
- **Identity domain modules (`src/identity/`)**:
  - `usernames.rs`: Normalization (case-folding to lowercase), regex and length bounds (3-64 ASCII alphanumeric, `.`, `_`, `-`).
  - `passwords.rs`: Argon2id hashing and constant-time verification with OWASP-aligned defaults (`m=65536`, `t=2`, `p=1`). Includes `dummy_verify` to equalise timing on non-existent usernames.
  - `tokens.rs`: Cryptographically secure 32-byte tokens (`OsRng`) encoded as unpadded URL-safe base64 (43 characters). Database stores SHA-256 hex digests.
  - `limits.rs`: In-memory thread-safe rate limiter (`DashMap`) tracking IP+user and IP-wide failed attempts with automatic eviction.
  - `store.rs`: Explicit parameterized SQL queries for atomic bootstrap, user CRUD, session create/resolve/revoke, scoped workspace queries, and security audit event logging.
- **HTTP handlers & middleware (`src/http/`)**:
  - `POST /api/v1/session`: Login endpoint with timing-safe validation, rate limiting, and issuance of `trestle-session` (HttpOnly) and `trestle-csrf` (accessible to JS) cookies.
  - `DELETE /api/v1/session`: Logout endpoint with CSRF verification and session revocation.
  - `GET /api/v1/me`: Returns safe user projection for the authenticated session.
  - `GET /api/v1/workspaces`: Returns only workspaces the authenticated user belongs to.
  - `middleware.rs`: Constant-time CSRF token verification middleware and session resolution middleware injecting `AuthenticatedSession` into request extensions.
- **CLI commands (`src/main.rs`)**:
  - `trestle bootstrap`: Atomic initial instance setup creating first user, workspace, and owner membership with terminal echo suppression (via safe `rpassword`).
- **Frontend Svelte components (`web/src/`)**:
  - `web/src/auth/SignIn.svelte`: Accessible keyboard-first login form with username/password inputs, rate-limit and error feedback, and credential zeroing on submit.
  - `web/src/auth/WorkspaceChooser.svelte`: Authorized workspace list displaying workspace names, slugs, and role badges.
  - `web/src/App.svelte`: Application shell updated with authentication state, user handle badge, sign-in/sign-out navigation, and workspace selection.
  - `web/src/ui/InputField.svelte`: Reusable accessible form input field.

## Test execution results

### 1. Rust unit & integration test suite (`cargo test`)

Executed against real PostgreSQL 18 container (`trestle-db-1`):
- **42 library unit tests**:
  - Username normalization and rejection of invalid characters, at-signs, spaces, and length boundaries.
  - Password minimum and maximum bounds, Argon2id format, correct/wrong verification, and distinct salting.
  - Session/CSRF token generation, unpadded base64url length (43 chars), deterministic SHA-256 hex digests (64 chars), and decoding validation.
  - Rate limiter bucket isolation, threshold triggers, and expiry eviction.
  - Constant-time string equality comparisons for CSRF validation.
- **5 identity integration tests (`tests/identity.rs`)**:
  - `bootstrap_creates_user_workspace_and_owner_membership_atomically`: verifies atomic creation and conflict rejection on second bootstrap attempt.
  - `session_creation_resolution_and_revocation`: verifies session creation, token resolution, and unresolvability after revocation.
  - `workspace_list_is_scoped_to_user_memberships`: verifies users only see workspaces where they have active memberships.
  - `http_login_success_and_logout_flow`: verifies unauthenticated 401, login 200 + cookies, authenticated `/api/v1/me` and `/api/v1/workspaces`, CSRF-protected logout, and session revocation.
  - `http_login_rate_limiting`: verifies 5 failures return 401 and 6th failure returns 429 Too Many Requests with `Retry-After` header.
- **19 existing test cases** in `tests/assets.rs`, `tests/config.rs`, `tests/database.rs`, and `tests/http.rs` retained and passing.
- **Total**: 66 passed, 0 failed, 0 ignored.

### 2. Frontend unit tests (`vitest`)

- **22 component tests** across 3 test files:
  - `src/App.test.ts`: 15 tests covering shell reflow, theme persistence, storage failure recovery, and readiness checks.
  - `src/auth/SignIn.test.ts`: 4 tests covering accessible labels, submit handling, 401 error alert + password zeroing, and 429 rate limit feedback.
  - `src/auth/WorkspaceChooser.test.ts`: 3 tests covering loading state, authorized workspace rendering, empty state, and retry on failure.
- **Total**: 22 passed, 0 failed.

### 3. Playwright browser E2E tests (`playwright test`)

- **19 Chromium E2E tests** across 3 spec files:
  - `e2e/auth.spec.ts`: 4 tests verifying navigation to sign-in, keyboard-first focus flow (Tab key navigation), error alert on invalid credentials, and mobile phone viewport reflow without horizontal scroll.
  - `e2e/foundation.spec.ts`: 2 tests verifying Rust server delivery and phone viewport layout.
  - `e2e/shell.spec.ts`: 13 tests verifying skip-link focus, theme transitions, 56 token-pair contrast thresholds, Axe WCAG A/AA audits, and reflow across 6 viewport sizes (320px to 1440px).
- **Total**: 19 passed, 0 failed.

### 4. Code quality & linting

- `cargo fmt --check`: Clean formatting across all Rust files.
- `cargo clippy --all-targets -- -D warnings`: Zero warnings, zero errors.
- `npm run check --prefix web`: `svelte-check` reports 0 errors and 0 warnings with strict TypeScript.
- `npm run format:check --prefix web`: Prettier formatting clean across all frontend files.
- `unsafe_code = "forbid"`: Fully preserved in `Cargo.toml`. Zero `unsafe` blocks in application code.

## Verification verdict

Milestone M03a.1 exit criteria are fully satisfied. All automated tests pass against live PostgreSQL, Playwright browser audits report zero accessibility or contrast violations, and the application shell handles authentication and workspace discovery.
