# Application shell verification (M02)

Date: 2026-10-15. Linux x86_64, Node 24.21.0/npm 11.19.0, Rust 1.99.0.
Scope and agreed seams: [application shell](features/application-shell.md).
Baseline: `243a53e4aba769c00992f9a23246580b272b59c4`.

## Frontend checks executed

- Clean locked `npm ci --ignore-scripts` succeeds. The only added dependency is
  test-only `@axe-core/playwright` 4.13.0 (MPL-2.0); no production library added.
- Prettier passes; Svelte/strict TypeScript reports zero errors/warnings.
- Fifteen DOM cases pass: all existing readiness cases plus preference
  apply/save, valid/corrupt restoration, blocked storage reads, failed writes and
  recovery without secret/error leakage. Tests exercise rendered application UI.
- Fifteen Chromium tests pass against compiled assets served by Rust: existing
  readiness/retry/delivery, skip-to-main focus, live system/explicit preference and
  reload, 56 token-pair contrast checks, Axe scans for both themes, and reflow
  at 320x640, 360x800, 390x844, 844x390, 768x1024 and 1440x900 with 200% root text.
  Touch emulation also navigates to instance status and retries real readiness.
- Text/link/status/button pairs reach 4.5:1; tested control/focus boundary pairs
  reach 3:1. Axe reports no tagged WCAG A/AA violations on the checked page states.
- `npm audit --audit-level=low`: zero reported vulnerabilities at execution time.
- Production build passes: JS 42.47 kB (16.71 kB gzip), CSS 5.65 kB (1.61 kB gzip),
  HTML 0.58 kB; bundled license text 2.32 kB. These are emitted bytes, not measured
  network/interaction latency or an approved performance budget. Svelte select/
  binding/feedback support increases the JS over M01b; no new runtime package.
- Actionlint and cargo-deny pass; existing Rust transitive duplicate warnings
  remain without advisory suppression. Browser dependency notices remain bundled.
- Desktop (1280px) and phone (390px) screenshots in both themes were visually
  inspected against real PostgreSQL readiness. This was not a screen-reader audit.

## Native/container verification

- `cargo fmt --all -- --check`, Clippy with warnings denied and locked release
  build pass.
- Full `cargo test --locked` passes against disposable PostgreSQL 18.6: 19 tests,
  none ignored. Existing configuration, HTTP/error/delivery, migration-concurrency
  and bounded pool-saturation checks were retained unchanged.
- `docker build --tag trestle:shell-check .` passes. Runtime passes with UID10001,
  read-only filesystem, dropped capabilities, no Node or `/app/web` directory.
  HTML/CSS/JS/CSP, readiness200 on private-network PostgreSQL, migrations twice,
  redacted logs and SIGTERM exit0 verified. Compiled-container mobile Chromium
  check verifies real-ready feedback, theme persistence/reload and touch retry.
  An initial browser launch failed because the shared browser executable was
  missing; installing the pinned Chromium into a session-private cache resolved
  the tool prerequisite and the complete smoke was rerun successfully. Playwright
  uses its Ubuntu22.04 fallback on this unsupported host OS.
- Local Markdown checks pass for 44 documents and 89 local links/fragments,
  including all20 feature sections; original license bytes unchanged.

## Standards review

No documented-standard violation or blocking finding. One optional duplicated
dark-palette smell was addressed with native CSS `light-dark()` pairs; explicit
and OS theme selectors only choose a scheme. All theme/contrast/browser tests
were rerun successfully after consolidation.

## Spec review

No material missing/incorrect behavior or scope creep found. Scheduled native,
container and evidence/tracker checks were subsequently completed as recorded above.

## Security and limitations

No database/schema/API/configuration changes. Same-origin embedding, CSP, no-store,
nosniff, fixed errors and bounded readiness remain. LocalStorage contains only an
allowlisted presentation preference; it is neither identity nor authorization.
Browser storage exceptions do not prevent the page from rendering or checking
readiness. No external fonts/icons or motion, no default credentials or user data.

Only Chromium has been executed; modern CSS `light-dark()` is required. Headless
viewport/text-enlargement and Axe checks do not substitute for physical-device,
real 200% browser zoom, virtual-keyboard, assistive-technology or full WCAG review.
No screen-reader/speech review, Firefox/WebKit, hosted Actions, Podman rerun,
production deployment or load benchmark is claimed. Later protected workflows,
dialog/table/editor patterns and density settings remain planned.
