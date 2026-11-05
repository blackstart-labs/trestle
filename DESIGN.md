# Design

Visual direction for Trestle. Token values: [docs/design/design-tokens.md](docs/design/design-tokens.md).
Components and behaviour: [docs/design/components.md](docs/design/components.md).

## World

A premium night-shift workbench. Near-black grounds tinted toward one brand
hue, stacked in four tone steps (ground → nav → panel → floating). One soft
ambient light falls over the top of the work area; resting panels carry a
one-pixel edge light rather than a drop shadow. The light theme mirrors the
same structure on cool off-white, so neither theme is an afterthought.

Signature: the **trestle**. The mark is a deck on braced legs. The overview
draws a bridge whose four piers are the product's parts (Plan, Work, Record,
Know, honestly marked Available or Planned) under one continuous deck: the
shared task identity. Inside a project, the **progress rail** is the same idea
applied to data. One bar spans every status category, and its legend is the
filter.

## Conventions kept and departures

The direction follows the conventions of premium dark project-management
tools.

Kept:
- the dark layered surfaces
- a soft lavender-type accent with dark text on the primary button
- the sidebar with grouped navigation and a workspace switcher
- the top bar with search, theme and account
- 16px-radius panels with tinted status pills
- kanban columns with an add slot
- the task table with priority dots

Deliberately different:
- Dashboard figures are all real: a seven-day due chart, a status mini-rail, a completion ring and a deadline timeline, computed from tasks. There are no invented sparklines, "team online" counts or AI cards.
- Search is a working command palette (projects, navigation, preferences), not a decorative field.
- Status changes happen inline on list rows and board cards, without opening anything. Drag is a shortcut, not the only path.
- Identity tints: every project and person gets a stable hue for its key tile or avatar, distinct from the brand accent.
- The accent is a theme setting. Five presets ship, and any hue works.
- Phone is designed rather than collapsed: a sticky navigation strip, two-line task rows, swipeable board columns and bottom-sheet dialogs.

## Material

Panels use a gradient fill inside a luminous 1px edge, and rows are inset
cards within them. Icons sit in lit squircle tiles, and the primary button is
lit with a soft glow. A diagonal sheen crosses the top of the canvas, and a
glowing sidebar card opens the command palette. Headings are set light
(480–540) and large, which is the main source of the calm, premium voice.
Desktop pointers get compact 38px controls on work screens; touch keeps 44px.

## Type

Mona Sans variable, self-hosted. Weights run 420–600 (headings 480–540); display only on
the overview. Keys use tabular figures, not monospace.

## Colour rules

- The accent marks the primary action, the current item's icon, focus, links and the ambient light. Never whole chrome surfaces.
- Status, priority and category hues are fixed and always come with words.
- Every text and control pair is verified at AA in both themes and every accent preset.
