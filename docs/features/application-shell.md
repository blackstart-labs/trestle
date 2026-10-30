# Application shell and design primitives (M02)

Status: implemented scoped shell; [executed evidence](../verification-shell.md).
Depends on M01b; PR-07/08/10. This does not certify all future product UI.

## 1. Purpose
Give future product workflows a calm, consistent, keyboard/touch-usable frame,
without implying identity, projects or data are implemented.
## 2. Users and use cases
Operators check the instance; users choose a readable appearance; developers
reuse controls and feedback rather than creating divergent visual patterns.
## 3. Functional requirements
SH-01: semantic header/navigation/main/footer and a keyboard-visible skip link.
SH-02: real in-page Overview, Appearance and Instance status navigation, adapting
to phone/desktop without hover, a modal drawer or nonexistent product links.
SH-03: system/light/dark preference; system follows live OS changes. Valid local
override survives reload; invalid/unavailable storage safely falls back. No server
profile, cross-tab synchronization or account preference is implied.
SH-04: semantic tokens and reusable button, select-field and feedback primitives,
used in the real appearance and readiness flows, not a component demo gallery.
## 4. Stories
As a keyboard user I skip navigation and reach a section with clear focus. As a
phone user I can read and operate all current actions without page overflow.
As a user I can choose a theme, including returning to OS preference.
## 5. Acceptance criteria
AC-SH-01: landmarks, skip/focus behavior and actual anchor targets pass browser
checks; no fake feature links/counters. Existing readiness retry/error tests pass.
AC-SH-02: theme selector has visible label/hint; valid persistence, reload,
system changes, corrupt storage and blocked read/write paths pass UI/browser tests.
Failure to persist displays a fixed, accessible message while the chosen theme
still works for the current page. Never store credentials/content.
AC-SH-03: light/dark text contrast >=4.5:1; large text and focus/control boundaries
>=3:1 for documented pairs. Automated accessibility scan has no violations on the
current page in both themes; this is not WCAG certification.
AC-SH-04: 320/360/390px phones, landscape, tablet and desktop reflow, 200% text and
keyboard operation pass. Interactive targets at least 44px; no essential motion.
AC-SH-05: reproducible frontend checks/build, existing Rust/delivery regressions,
container build/runtime and documentation/review/commit complete.
## 6. Screens and flows
Single public shell, with honest development status. Anchor navigation does not
fetch new pages or authorize anything. Appearance change applies immediately;
instance section uses the existing bounded readiness request/retry.
## 7. States
Retain checking/ready/unavailable/network-failure feedback. Appearance has system,
explicit override and storage-failure states; success confirms saved preference.
No data-loading skeleton, signed-in state or forbidden product screen is invented.
## 8. Entities and rules
Only local presentation preference. Task/workspace domain model unchanged.
## 9. Database
No schema, migrations, queries or writes added.
## 10. HTTP
No new routes. Continue embedded same-origin HTML/CSS/JS and existing health/errors.
No SPA fallback or public content index. Foundation HTML title remains compatible.
## 11. Security
Keep restrictive CSP, external styles/scripts, no inline style/theme bootstrap or
new remote assets. Store only allowlisted theme value at `trestle.theme` in browser
localStorage. Reads/writes may throw; do not log errors or echo stored contents.
## 12. Validation
Only `system`, `light`, `dark`; all other persisted values use system. Blocked
storage must not prevent mounting/readiness. Use native select semantics.
## 13. Concurrency
Theme changes are synchronous; readiness behavior/bounds are unchanged. System
theme uses CSS media query, no polling/listener cleanup burden. Cross-tab edits
are intentionally reflected on reload, not synchronized live.
## 14. Performance
No new production dependency, font download, icon library, animation or router.
Record build bytes; do not claim performance budgets from bundle sizes.
## 15. Tests
User approved extended DOM/browser seams (shell/focus, theme persistence/reflow,
form/feedback), plus existing HTTP/readiness regressions. Test UI not private
helpers. Browser tests load compiled assets from Rust. Accessibility tooling is
build/test-only. Review baseline approved: `243a53e`.
## 16. Accessibility and responsive behavior
Visible labels, descriptive hints/error association, native controls, focus rings,
text feedback independent of color. System fonts, semantic typography, responsive
landmarks, safe-area padding and wrapping. No hover-only or drag-only action. No
motion; honor forced colors with browser-native semantics. Document manual and
automated evidence/limits, especially lack of physical-device/screen-reader audit.
## 17. Integration
Use controls in appearance/readiness; build still embeds output through existing
Rust seam. Future protected domains must replace/extend this shell deliberately,
not assume anchors or local preferences establish authorization.
## 18. Migration/compatibility
No persistence migration. Unknown theme values degrade to system; local preference
belongs to origin/browser, not identity. Existing health/error contracts unchanged.
## 19. Operations
Same npm-build then Cargo sequence. Rebuild binary/image for UI changes. Theme
storage is optional; browsers denying it remain usable. No backend settings added.
## 20. Done
Meet AC-SH-01–05, document tokens/component contracts and evidence, update tracker/
matrix, review standards/spec separately and commit locally. Identity decisions
remain the next material gate, not implicitly resolved by UI implementation.
