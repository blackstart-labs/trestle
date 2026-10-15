# Design system direction

Status: specification direction only; no frontend, tokens, themes or components
have been implemented. M01b/M02 must turn this into verified interfaces.

## Identity and hierarchy

Trestle is a calm, precise workbench for planned work, measured work and knowledge.
Use readable typography, aligned data, restrained borders and clear selected/
focus states. Prefer one purposeful workspace over repeated rounded cards,
gradients, decorative charts, glass effects or fake metrics. No logo asset or
font license is claimed to exist.

Start with system UI fonts for privacy, local rendering, language fallback and
zero font-download cost; use a system monospace face for task keys/code. Check
actual glyph coverage and numerals with target languages before adopting a
custom self-hosted font and its license. Keep body text readable at 200% zoom.

## Token contract for M02

Define semantic surface/background/elevated, text/secondary/muted, border,
accent/selected/focus, and success/warning/danger tokens. A status color conveys
classification, not authorization or the only indication of meaning. Add spacing,
type scale/line height, control size, radius and layering tokens rather than
scattered literals. Set exact values with contrast/visual review in M02.

Light/dark themes use the same semantics, respect system preference and allow a
persisted override. Density changes row spacing, not minimum touch targets or
text contrast. A theme/token document becomes authoritative when implemented;
do not infer WCAG conformance from choosing a palette.

## Shared interaction contracts

| Pattern | Required behavior |
| --- | --- |
| Forms | Visible labels, field-associated errors, preserve unsaved input, prevent accidental duplicate submission, explicit save state |
| Dialog/menu | Semantic accessible role, initial/return focus, Escape behavior, keyboard navigation, no invisible focus trap |
| Tables | Header semantics, discoverable sorting, stable row identity, bounded data, keyboard access to edit controls |
| Boards | Pointer drag is optional; keyboard/touch move menu, persistent ordering, conflict/retry feedback |
| Search | Keyboard-open/close and result navigation, bounded authorized results, empty/loading/error distinction |
| Feedback | Loading does not impersonate data; empty explains next permitted action; saving/success/error/denied are distinct |
| Destructive actions | Explain affected data and recovery; confirm consequential operations; archive before purge where specified |

Use optimistic updates only with canonical server reconciliation and a clear
conflict/failure path. Avoid simultaneous modal/drawer state that strands focus.

## Responsive and accessible acceptance

Test 320/360/390px phone widths, tablet and wide desktop, portrait/landscape,
virtual keyboards, safe-area insets and long translated text. Navigation adapts;
core task/time/document actions do not require desktop hover. Tables deliberately
choose columns or offer a readable alternate/detail view; horizontal scrolling is
contained rather than applied to the whole screen.

Target WCAG 2.2 AA: semantic HTML, contrast checks, visible focus, 200% zoom/reflow,
keyboard and screen-reader journeys, reduced motion, and minimum target spacing/
size requirements. Prefer 44px touch hit areas while retaining efficient desktop
density. Automated checks supplement, not replace, manual interaction review.

High-frequency task/table actions should normally have no motion. Where motion
clarifies an occasional state change, prefer brief transform/opacity changes,
with ease-out for entering/exiting UI and ease-in-out for repositioning. Never
delay durable feedback or make dragging the only control. Reduced-motion mode
removes nonessential movement. No decorative animation library is justified.

## Public versus private pages

Public indexable pages, if introduced, need semantic content, metadata, canonical
URLs and deliberate sitemap/robots behavior. Private tasks, documents and users
never enter public metadata, generated static pages or public indexes. No public
sharing or marketing website is implemented in this milestone.
