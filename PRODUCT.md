# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Independent developers, small product teams, consultants and small
organisations who plan projects, move tasks through their workflows, and will
record time and connect documentation in one self-hosted place. Personas include
a contributor working from a phone, who must be able to finish core workflows by
touch on narrow screens.

## Product Purpose

A cohesive self-hosted workspace: projects, canonical tasks, custom workflows,
collaboration on tasks, then time and knowledge. Success is usable, responsive
everyday work with modest operational complexity, not the number of modules or
dashboard statistics.

## Positioning

One task identity (`KEY-123`) across lists, boards, planning, time and links,
on infrastructure the team owns, with portable content and no proprietary
service required for core work.

## Operating Context

Implemented workflows: local sign-in, Workspace creation/settings/membership and roles,
ownership transfer, Projects with keys and image uploads, Tasks with labels,
assignees, subtasks, flat comments and activity, custom Statuses mapped to five
canonical categories, milestones with rollups/timeline, List and Board views,
JSON exports, and Project notifications to Telegram, Slack and Discord.

Workspace Boards are a native diagram and whiteboard editor: shape library and
templates, direct manipulation, connectors, formatting, grouping and alignment,
pan/zoom, JSON/SVG/PNG export, explicit version-checked saves, undo/redo and
reversible archive. They are not Lucidchart/Excalidraw embeds or file-compatible
replacements. See [Boards](docs/boards.md) for permissions, limits and test
instructions.

Basic time logging is implemented: task-scoped manual hours, notes, dates and a
browser-local live timer. Server-authoritative timers, pause/resume, timesheets,
Customers, work Activities, rates and billing are not implemented. Project-level
membership/private visibility and invitations are also not implemented.

Goals, cycles, modules, dependencies, saved views, full Task calendar/Gantt,
Documents, connected knowledge, collections, GitHub integration, professional
reports and automation remain planned. The source-backed inventory and proposed
delivery order are in [Phase 0](docs/phase-0-gap-analysis.md); no plan entry is a
claim of an available feature.

## Capabilities and Constraints

- Svelte 5 + Vite static frontend, served from the Go binary on the same origin.
- A strict CSP: `default-src 'none'`, self-only scripts, styles, fonts and connections, images limited to same-origin and blob URLs, and no inline style attributes. Dynamic values go through the CSSOM (`style:` directives). Raw HTML and external images in task Markdown are not rendered.
- No AI features exist. The UI must not imply AI, metrics or modules that do not exist.
- Dragging is never the only way to change anything; keyboard and touch are first-class.

## Brand Commitments

- Name: Trestle. The mark is a drawn trestle (deck on braced legs).
- The visual direction is a premium, dark-first workbench chosen by the maintainer (see [DESIGN.md](DESIGN.md)).
- The colour theme is not final. The palette must stay re-themeable from a single hue setting.

## Evidence on Hand

No customers, testimonials, benchmarks or usage metrics exist. Do not fabricate them.

## Product Principles

1. Show real state; never invent numbers or capabilities.
2. Every frequent job is one click or one keystroke away (command palette, inline status menus).
3. Phone users finish the same workflows as desktop users.
4. Colour carries meaning (state, category, priority) and is always paired with words.
5. Owned infrastructure first: the browser makes no third-party requests. Optional project integrations send notifications from the server to explicitly configured providers.

## Accessibility & Inclusion

Target WCAG 2.2 AA. Contrast is verified automatically for every token pair in
both themes, and axe runs in e2e. Reflow is checked from 320px with 200% text.
Targets are 44px, focus is visible, and reduced motion is respected.
