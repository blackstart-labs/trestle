# Product

<!-- impeccable:product-schema 1 -->

## Platform

web

## Users

Independent developers, small product teams, consultants and small
organisations who plan projects, move tasks through their workflows, and will
record time and connect documentation in one self-hosted place. Personas in
[docs/product/personas.md](docs/product/personas.md) are planning hypotheses,
not validated research. They include a contributor working from a phone, who
must be able to finish core workflows by touch on narrow screens.

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

Shipped today: local sign-in, workspace membership and roles, ownership
transfer, projects with keys, tasks with labels, assignees, subtasks, comments
and activity, custom statuses mapped to five canonical categories, and list and
board views. Planned and not yet built: milestones, cycles, time tracking,
documents, reports, automation. See [the roadmap](docs/product/roadmap.md).

## Capabilities and Constraints

- Svelte 5 + Vite static frontend, served from the Rust binary on the same origin.
- A strict CSP: `default-src 'none'`, self-only scripts, styles and fonts, and no images or inline styles. Any visual must be CSS or inline SVG; dynamic values go through the CSSOM (`style:` directives).
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
5. Owned infrastructure first: no third-party requests at runtime.

## Accessibility & Inclusion

Target WCAG 2.2 AA. Contrast is verified automatically for every token pair in
both themes, and axe runs in e2e. Reflow is checked from 320px with 200% text.
Targets are 44px, focus is visible, and reduced motion is respected.
