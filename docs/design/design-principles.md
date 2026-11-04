# Design system direction

Status: implemented across the shell, auth, projects, tasks (list and board),
task drawer, status manager and members views. [Tokens](design-tokens.md) and
[component contracts](components.md) describe the system.

## Identity and hierarchy

Trestle is a calm, precise workbench that feels premium after dark. Layered
near-black surfaces tinted toward one brand hue, a single soft ambient light
over the work area, and an edge light on resting panels carry the atmosphere.
Hierarchy comes from type weight, a quiet sidebar and tone steps between
surfaces, not from colour on chrome. Colour is reserved for state, data and the
one primary action.

The product's own mark is a trestle: one deck carried by braced legs. The same
idea appears as the overview linework (four piers, one continuous deck) and as
the progress rail, where one bar spans every status category of a project.

Do not add sparkline KPIs, invented metrics, AI or assistant surfaces, or
"coming soon" controls for capabilities that do not exist. Planned modules may
be named only as planned, as the overview's piers are.

Mona Sans is self-hosted (OFL-1.1) with Latin, Latin Extended and
Vietnamese subsets; check glyph coverage before claiming other scripts.
Body text stays readable at 200% zoom.

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

High-frequency task and table actions have no motion beyond colour feedback.
Occasional state changes (dialogs, drawers, palette, rail resizing) use brief
transform/opacity with an exponential ease-out; the overview deck draws once.
Never delay durable feedback or make dragging the only control. Reduced motion
removes all travel and keeps the content.

## Public versus private pages

Public indexable pages, if introduced, need semantic content, metadata, canonical
URLs and deliberate sitemap/robots behavior. Private tasks, documents and users
never enter public metadata, generated static pages or public indexes. No public
sharing or marketing website is implemented in this milestone.
