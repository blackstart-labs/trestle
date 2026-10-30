# Frontend foundation (M01b)

Status: implemented; see [executed evidence](../verification-frontend.md).
Requirements PR-07/08/10; depends on M01a
and ADR-0002. M02's navigation/design system and all product domains are excluded.

## 1. Purpose
Establish reproducible browser builds and one-origin delivery without a production
Node service, fabricated workspace data or pretend product functionality.
## 2. Users/use cases
Developers build/test the browser client; operators open the foundation page and
check current database readiness.
## 3. Functional requirements
FF-01: Svelte/Vite/strict TypeScript with locked dependencies and pinned Node/npm.
FF-02: Rust embeds the production HTML/CSS/JS at build time and serves known paths.
FF-03: a visible bounded readiness check distinguishes checking, ready, unavailable,
and network/protocol failure; retry is explicit, with no polling.
## 4. Stories
As an operator I can see what exists and check readiness without exposing secrets.
As a developer I can verify the page through DOM interactions and real HTTP.
## 5. Acceptance criteria
AC-FF-01: clean `npm ci`, format/type checks, tests and production build succeed.
AC-FF-02: `/` serves HTML and referenced assets load with correct MIME types;
HEAD is bodyless, unsupported methods are 405; unknown/API/traversal paths are
JSON 404, never index HTML. No runtime disk reads or SPA fallback.
AC-FF-03: all delivery has no-store/nosniff and a restrictive production CSP;
no request URL/header/content is added to diagnostics.
AC-FF-04: successful readiness, 503, malformed responses, network failure and retry
are tested visibly. One request at a time, 5-second browser deadline, no raw errors.
AC-FF-05: browser smoke loads production assets from Rust, exercises retry, and
checks keyboard operation and a narrow viewport. Hosted CI remains unverified
until run remotely.
## 6. Screens/flow
One honest foundation page: title, capability boundary, readiness status and check
button. Check on mount; user can check again. No accounts/projects navigation yet.
## 7. States
Checking disables duplicate submission; ready states only database connectivity,
not schema compatibility; unavailable reflects 503; network/protocol failure uses
fixed safe text. No arbitrary server response is rendered.
## 8. Domain rules
No new domain entities. Browser readiness uses the existing public endpoint.
## 9. Database
No tables, migrations, writes or additional DB privileges.
## 10. HTTP
GET/HEAD `/` and build-manifest-known `/assets/*`; existing health/error contracts
unchanged. Root is noindex; no public indexing/sharing introduced.
## 11. Security
No sessions/CORS/credentials added. CSP restricts scripts/styles/connect to self,
forbids framing, objects and base changes; all content is public diagnostics only.
Allowlisted embedded assets prevent filesystem/path traversal exposure. Development
Vite is loopback-only and proxies only health; it is not a deployment server.
## 12. Validation
Accept readiness only for a 200 JSON object with `status: "ok"`; handle 503 separately.
Other status/body types fail safely. Unknown paths remain fixed JSON errors.
## 13. Concurrency
Disable checks during a request; abort on timeout/component teardown. No polling,
optimistic mutation, persistence or ordering.
## 14. Performance
Record emitted asset sizes, not speculative budgets. No fonts/icons/animation/UI
libraries. Embed assets once; no filesystem access on requests. Deployment size
and browser timings are observations, not a throughput guarantee.
## 15. Tests
User approved seams: Rust root/asset HTTP responses; Svelte visible check/error/
retry interactions; production browser smoke served by Rust. Existing health tests
remain. HTTP tests cover delivery, HEAD/methods, missing paths and headers; DOM
tests replace only boundary fetch. Real-browser smoke verifies compiled delivery.
## 16. Accessibility/responsiveness
Semantic main/heading/button, visible focus, status live region, system fonts,
readable wrapping and no horizontal page overflow on phones. No motion. Full
design-system/WCAG certification is not claimed by this milestone.
## 17. Integration
Rust build requires `web/dist` produced by `npm run build`; generated files are
ignored. Docker/CI build web before Rust. A missing build fails with instructions,
never silently ships an empty page. Native backend changes use the same sequence.
## 18. Compatibility
Root changes from JSON 404 to HTML; health and unknown-route behavior remain.
No future private route fallback or asset compatibility window is promised.
## 19. Operations
Node/npm build-only; document clean builds, tests, dev proxy and containers. Dist
is embedded; changing frontend requires rebuilding the binary. Preserve license
and record dependency notices; no push/release is authorized implicitly.
## 20. Done
Meet AC-FF-01–05, update CI/container/docs/matrix/tracker, run relevant Rust checks
and complete suite against disposable PostgreSQL, review, commit locally. Disclose
hosted CI, platform and accessibility/performance evidence limits.
