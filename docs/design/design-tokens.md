# Design tokens and typography

Status: M02 implemented for the public shell, not all future product screens.
Exact values are authoritative in [`web/src/tokens.css`](../../web/src/tokens.css).
Do not copy the palette into component styles. Global patterns live in
[`app.css`](../../web/src/app.css); components use semantic classes/tokens.

## Visual identity

A restrained workbench: warm neutral surfaces, green-ink links/focus, readable
data-oriented typography, borders rather than floating card stacks. `Trestle.`
is a text wordmark, not a newly licensed logo asset. No gradients, decorative
charts, downloaded fonts, icons or motion. Future iconography must have named
actions and reviewed licensing; no arbitrary icon library is selected now.

## Semantic roles

| Tokens | Contract |
| --- | --- |
| `surface`, `surface-raised`, `surface-nav` | Page, feedback/control and navigation backgrounds |
| `text`, `text-secondary`, `text-muted` | Primary/supporting/de-emphasized text, all readable rather than low-opacity |
| `accent`, `selected`, `focus` | Links, hover/focus navigation background, visible keyboard outline |
| `border` | Control/layout boundaries; not a signal of status alone |
| `success`, `warning`, `danger` | Text plus border emphasis, always accompanied by meaningful words |
| `button-bg`, `button-text` | Primary action pair, independent of status meanings |
| `space-*`, `control-height`, `radius`, `border-width` | Shared rhythm, 44px minimum targets at default size, restrained corners |
| `focus-width`, `focus-offset`, `layer-skip` | Visible offset ring and skip link above other content |
| `content-width`, `nav-width` | Readable main measure and navigation column, not unbounded full-screen text |

System UI fonts avoid privacy/network/licensing overhead; monospace is a system
fallback for small contextual labels. Body uses 1rem and 1.6 line height;
supporting text 0.875rem; section/lead 1.25rem; title fluid 2–3rem. Heading line
height is 1.2. No custom font files are distributed. Language coverage and future
internationalization require real-language review, not assumptions about glyphs.

## Themes and preference

System is the default and follows `prefers-color-scheme` live. Explicit light/dark
overrides apply through a root `data-theme` attribute; no inline CSS/script or
CSP exception. `trestle.theme` stores only an allowlisted value in localStorage.
Native CSS `light-dark()` pairs keep each palette definition in one place;
scheme selectors/media queries only choose the scheme, without JS OS listeners.
This targets modern browsers supporting `light-dark()`; only Chromium has been
executed here, not an established browser compatibility matrix.
Corrupt values use system; storage exceptions leave the page usable and show a
safe warning. Changing the select applies immediately and attempts persistence.
This is a browser/origin preference, not a server user profile. Cross-tab changes
apply on reload. Preference initialization happens at mount, not a claimed
flash-free pre-paint server render.

## Contrast and interaction

Browser tests measure 28 foreground/background pairs per theme using the WCAG
relative-luminance formula: text/link/muted/status/button >=4.5:1; focus/boundary
pairs >=3:1. Axe supplements those checks on both rendered themes. Disabled
controls retain native disabled semantics; status is never communicated by color
alone. No theme color choice alone proves WCAG conformance.

Main/heading anchor targets and all keyboard controls have focus indication.
No transitions/animations or reduced-motion exception is needed. Forced colors
are not disabled; native control/outline semantics remain available. Full
screen-reader/physical-device/high-contrast validation remains a release gate.

## Responsive and density policy

At 48rem and below, the sidebar becomes wrapping navigation above content. It is
not hidden behind a modal or hover interaction. Main content uses a minmax track,
bounded measure and wrapping; no body-wide horizontal scrolling. Safe-area padding
protects content on devices supplying insets. Viewport and 200% text-enlargement
checks are documented in the M02 evidence record.

There is no density toggle or data table yet. Future dense rows may reduce
spacing but must not shrink text contrast or accessible hit targets. Tokens are
the shared vocabulary, not an excuse to ship unused dialogs/tables/editors now.
