# Design tokens and typography

Exact values are authoritative in [`web/src/tokens.css`](../../web/src/tokens.css).
Global element styles and shared component classes live in
[`app.css`](../../web/src/app.css). Components reference semantic roles only,
never raw colours, so a palette change never touches component code.

## Re-theming

The whole palette is derived in OKLCH from two knobs at the top of `tokens.css`:

| Knob | Default | Effect |
| --- | --- | --- |
| `--brand-hue` | `284` (iris) | Hue of the accent, focus ring, primary button, selection and ambient light. Neutrals lean slightly toward it. |
| `--brand-chroma` | `0.13` | Saturation of the accent family. |
| `--tint` | `0.014` | How far grounds, surfaces and lines lean toward the brand hue. `0` gives true greys. |

To change the default, edit those values. To offer another choice, add a
`:root[data-accent="name"]` block that sets the two knobs and add the same hue
to `accents` in [`preferences.svelte.ts`](../../web/src/lib/preferences.svelte.ts).
Shipped presets: iris (284), sky (246), tide (192), moss (142), ember (52).
Status, priority and category colours use fixed hues, so their meaning never
moves with the brand.

After any palette change, run the contrast check in `e2e/shell.spec.ts`. It
resolves each token through a canvas, so it measures what is actually displayed
regardless of colour space.

## Semantic roles

| Tokens | Contract |
| --- | --- |
| `surface`, `surface-nav`, `surface-sunken`, `surface-raised`, `surface-float` | Page ground, sidebar, wells and inputs, panels and cards, dialogs and menus (lightest to heaviest elevation) |
| `hover`, `selected` | Pointer hover and the selected or current item |
| `line`, `line-strong` | Panel and row separators. Decorative, not a boundary signal. |
| `border` | Control boundaries; at least 3:1 against every surface |
| `text`, `text-secondary`, `text-muted` | Primary, supporting and de-emphasised text; all at least 4.5:1 on every surface |
| `accent`, `accent-strong`, `accent-soft`, `accent-glow`, `focus` | Links and current state, hover, tinted fills, ambient light, keyboard focus ring |
| `button-bg`, `button-bg-hover`, `button-text` | The primary action pair |
| `success`, `warning`, `danger`, `info` (+ `-soft`) | State text and its tinted fill, always paired with words |
| `cat-backlog` … `cat-cancelled` | The five status categories (progress rail, board fallback columns, category pills) |
| `shadow-card`, `shadow-float`, `shadow-modal` | Resting edge light; floating menus; dialogs and drawers |

Light and dark pairs are written once with `light-dark()`; `color-scheme`
selects the side. System is the default and follows the device live. Explicit
light or dark is stored in `trestle.theme`, the accent in `trestle.accent`.
Both are allowlisted on read and are browser-local, not server profile data.

## Material

Premium depth comes from material, not from heavier shadows:

| Token | Use |
| --- | --- |
| `card-fill`, `card-edge` | Panels, cards, dialogs, board columns. The fill falls from light at the top left to shade at the bottom right, drawn inside a 1px gradient edge that is brightest at the top (`background: fill padding-box, edge border-box; border: 1px solid transparent`). |
| `inset-fill`, `inset-edge` | Cards inside panels: task rows, dashboard rows, secondary buttons, the selected nav item |
| `tile-fill`, `tile-ink` | `.tile`: a lit squircle holding one icon |
| `primary-fill`, `glow-accent` | The lit primary button and the light that pools under it and in stat cards |
| `sheen` | The diagonal light across the top of the canvas, the overview hero and the `.rim` edge |

All of these derive from the theme knobs, so they re-colour with the accent.
Text is never placed on a fill that differs from `surface-raised` by more than
the contrast-tested range.

## Type

Mona Sans (variable, OFL-1.1) is self-hosted from `@fontsource-variable/mona-sans`,
so it loads under the `font-src 'self'` policy with no third-party request. Task
and project keys use the text face with tabular figures; monospace is kept for
code only.

| Token | Use |
| --- | --- |
| `text-display` | The overview title only |
| `text-title` | Page titles (22–28px) |
| `text-section`, `text-lead` | Panel and section headings, lead sentences |
| `text-ui` (15px) | Default interface text |
| `text-small`, `text-xs` | Supporting text, labels, meta (never below 12px) |

Headings track at -0.022em, the display at -0.035em; nothing is tighter than
-0.04em.

## Shape, space and motion

- Spacing is a 4px scale (`space-1` … `space-16`); layout gutters use `--gutter`.
- Radii: `radius-sm` 8px for controls, `radius-lg` 16px for panels, `radius-xl` 20px for dialogs; pills only on small chips.
- Controls are 44px tall (`--control-height`) at every size. Dense rows keep 44px hit areas.
- Motion follows Apple's fluid-interface rules, implemented in `web/src/lib/motion.ts`:
  - **Critically damped by default.** `--ease-spring` and `settle()` are a damping-ratio-1 spring with no overshoot. `--ease-bounce` is reserved for the subtask checkmark landing.
  - **Exits retrace entrances and run faster.** The drawer enters from the right on the iOS sheet curve (`--ease-sheet`, 460ms) and leaves right on an accelerate curve (280ms). Dialogs materialise in place (320ms in, 160ms out). Toasts rise from and sink into the bottom edge.
  - **Popovers grow from their trigger.** The positioner sets `transform-origin` to the trigger's point.
  - **Page changes are View Transitions.** Only `<main>` cross-fades, and the shell stays still. The page is named only while it transitions, because a permanent name would trap overlays under the top bar. Theme switches cross-fade the whole screen.
  - **No cascades.** Data that loads appears in one short fade, and board cards travel between columns (flip plus crossfade).
  - **Hover washes animate.** They use a registered `@property --veil` colour, because gradients cannot interpolate. Press feedback is on pointer-down (80ms) with a soft release.
  - **Only transform and opacity move.** Reduced motion turns every transition into a short cross-fade, and reduced transparency removes backdrop blur.

## Responsive policy

- Above 60rem: a fixed sidebar and a sticky translucent top bar.
- At 60rem and below: navigation becomes a sticky, horizontally scrolling strip under the top bar, so every destination stays one tap away with no hidden menu.
- At 40rem and below: dialogs become bottom sheets, task rows become two-line cards, and boards swipe column by column.

The page never scrolls sideways; wide content (boards) scrolls inside its own
frame. Reflow is tested at 320–1440px with 200% text.
