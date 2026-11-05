# AGENTS.md — rules for AI agents and developers

Read this whole file before changing anything in `web/`. It is the contract for
every new screen, feature and component. Human contributors follow it too.
Where this file and older docs disagree, this file wins; fix the older doc.

Trestle is a self-hosted workspace for projects, tasks, time and knowledge:
a Rust (axum) backend serving a Svelte 5 + Vite static frontend from one
origin. Product terms live in [CONTEXT.md](CONTEXT.md). Use them exactly:
Workspace, Project, Task, Status, Status category, and so on.

## 1. Read before you build

| Read | For |
| --- | --- |
| [DESIGN.md](DESIGN.md) | The visual world, what to keep, what never to add |
| [PRODUCT.md](PRODUCT.md) | Users, principles, what exists and what does not |
| [docs/design/design-tokens.md](docs/design/design-tokens.md) | Every token, the motion rules, responsive policy |
| [docs/design/components.md](docs/design/components.md) | Every shared component and screen contract |
| [docs/design/design-principles.md](docs/design/design-principles.md) | Interaction contracts (forms, dialogs, boards, search) |
| `web/src/tokens.css`, `web/src/app.css` | The actual values and global classes |
| An existing screen like yours | `projects/TasksView.svelte`, `projects/Dashboard.svelte`, `auth/WorkspaceMembers.svelte` |

Build the new thing so that it looks like it was always part of those screens.

## 2. The non-negotiables

These are checked by `npm run lint:design` (CI fails on a violation),
`svelte-check`, the unit tests and the Playwright suite (axe, contrast, reflow).

1. **No raw colours in components.** Use semantic tokens:
   `--text`, `--text-secondary`, `--text-muted`, `--accent`, `--surface-*`,
   `--line`, `--border`, `--success`, `--warning`, `--danger`, `--info`
   (and their `-soft` fills), `--cat-*` for status categories. Derived
   tints must reference a variable, like
   `oklch(0.92 0.05 var(--project-hue))`. Pure white or black with alpha is
   allowed for glints and shadows. Need a new colour role? Add a token to
   `tokens.css` with light and dark values and document it.
2. **No native pickers.** Never use `<select>` or `<input type="date">`. Use
   `ui/Select.svelte` (variants `field`, `chip`, `ghost`) and
   `ui/DatePicker.svelte`. For multi-choice, like people or labels, use
   `ui/MultiPicker.svelte`.
3. **No inline `style="…"` attributes.** The CSP is `style-src 'self'`. Pass
   dynamic values with Svelte directives: `style:--status-color={c}`.
4. **No third-party runtime requests.** No CDN fonts, scripts or images. Install
   through npm and bundle. No `data:` images (the CSP forbids them); draw with CSS
   or inline SVG.
5. **Motion comes only from `web/src/lib/motion.ts`** and the easing tokens
   (`--ease-spring`, `--ease-out`, `--ease-sheet`, `--ease-exit`,
   `--ease-bounce`). Never import `fade`, `fly`, `scale` or `slide` from
   `svelte/transition`. Never write `cubic-bezier()` or `transition: all`.
6. **Layers come from tokens** (`--layer-sticky`, `--layer-overlay`,
   `--layer-palette`, `--layer-popover`, `--layer-skip`), not raw `z-index`.
7. **Icons come from `ui/Icon.svelte` only.** It is one outline family, 1.75
   stroke. Add a new icon as a path there; never install an icon package.
8. **Type is Mona Sans through `--font-body`.** Use the size tokens
   (`--text-xs` … `--text-display`). Headings are light (480–540); emphasis
   comes from weight and size, never colour.
9. **Never invent data.** No placeholder metrics, sparklines, "team online"
   counts, AI or assistant surfaces, or "coming soon" buttons. Every number
   comes from the API. When a figure is sampled or bounded, say so on screen.
   Completion is always completed ÷ (all − cancelled).
10. **Accessibility is WCAG 2.2 AA:**
    - every control has an accessible name (`label` or `aria-label` on icon-only buttons)
    - 44px touch targets
    - visible focus
    - keyboard parity for everything; dragging is never the only way
    - AA contrast in light and dark and in every accent preset
    - no horizontal page scroll from 320px wide with 200% text

A deliberate exception needs `design-allow: <rule> <reason>` on the line.
Reviewers question every one.

## 3. Use the system's building blocks

| Need | Use |
| --- | --- |
| Button | `ui/Button.svelte` with `variant` = `primary` (one per area), `secondary`, `quiet`, `danger`, `danger-quiet`. A bare `<button>` gets the secondary style. Icon-only: add classes `btn-quiet btn-icon` plus an `aria-label`. |
| Text field | `ui/InputField.svelte`, or `<input>` inside a `.field` with a `<label for>` |
| Dropdown | `ui/Select.svelte`. A form field uses `variant="field"` with `<label for=id>`; an inline status or priority uses `variant="chip"`; a toolbar filter uses `variant="ghost"` |
| Date | `ui/DatePicker.svelte` |
| People or labels | `ui/MultiPicker.svelte`. Toggles apply immediately; there is no confirm step. |
| Dialog | `ui/Modal.svelte`. It handles focus trap, Escape, focus return and the phone sheet. |
| Side panel | Follow `projects/TaskDetailDrawer.svelte`: `drawerIn`/`drawerOut`, focus trap, and an unsaved-changes guard |
| Success message | `ui/Toast.svelte`, which is transient |
| Error or warning | `ui/Feedback.svelte` inline with `tone="danger"`. It stays until fixed and is never a toast. |
| Loading | Skeleton blocks shaped like the content, plus one `Feedback announcement="status"` |
| Empty state | `.empty` with an `.empty-mark` tile, a heading, one sentence and the next permitted action |
| Surfaces | `.panel` or `.card` (gradient fill and luminous edge), `.inset` for cards inside a panel, `.tile` for an icon squircle. Never invent new shadow or border recipes. |
| Status, priority, labels | `.status-chip`, `.priority` (`data-priority`), `.label-badge`, always with words, never colour alone |
| People | `.avatar` with `style:--avatar-hue={identityHue(id)}` and `initials(name)` from `lib/identity.ts` |
| Dates and recency | `formatDueDate`, `isOverdue`, `relativeTime` from `lib/identity.ts` |
| Page header | `.page-head` with a noun title (`h2`), one supporting sentence, and `.page-actions` on the right |
| Navigation and search | Add destinations to the sidebar `nav` and the command palette list in `App.svelte`; change views through `go()` so they use View Transitions |

If none of these fits, **extend the system instead of one-off styling**: add a
token, a class in `app.css`, or a component in `web/src/ui/`; document it in
`docs/design/`; then use it.

## 4. Behaviour every screen must have

- **States:** loading, empty, filtered-empty, error with retry, success, disabled, and permission-limited (hide or explain actions the role cannot take).
- **Destructive actions** confirm in words that name what happens and whether it can be undone. Archive comes before delete.
- **Forms** preserve input on error, prevent double submit (disable while saving), name errors and their fix, and guard unsaved edits on close.
- **Saving:** inline property changes save immediately. Long text saves explicitly, with a visible unsaved state.
- **Keyboard:** Esc closes only the top layer; shortcuts never fire while typing or when a dialog is open; show `kbd` hints only where a keyboard exists.
- **Responsive:**
  - above 60rem: sidebar layout
  - at or below 60rem: a scrolling nav strip
  - at or below 40rem: dialogs become bottom sheets and tables become two-line rows
  - wide content (boards) scrolls inside its own frame
- **Motion:**
  - Default to the critically damped spring; nothing overshoots unless it was thrown.
  - Exits retrace entrances and run faster.
  - Popovers grow from their trigger.
  - Data appears in one short fade, never a cascade.
  - Animate only `transform` and `opacity`. Hover washes use the `--veil` property.
  - Reduced motion becomes a cross-fade.
- **Copy:** sentence case, plain words, the product's own nouns. A button names its action ("Save task", not "Submit"). Errors say what happened and what to do.

## 5. Definition of done

Run from `web/`. Every check must pass before you call the work finished.

```bash
npm run lint:design   # design-system guard
npm run check         # svelte-check: 0 errors, 0 warnings
npm test              # unit tests (add tests for new behaviour)
npm run format:check  # prettier
npm run build         # production build
npm run test:e2e      # needs the Rust toolchain; axe, contrast, reflow, keyboard
```

Then look at the result, in **both themes**, at **390, 768 and 1440px**:

- [ ] It uses only tokens and existing components, or extends the system and updates `docs/design/`.
- [ ] Every state in §4 exists and was seen.
- [ ] Keyboard-only and touch-only paths work.
- [ ] No number on screen is invented, and repeated figures agree.
- [ ] Docs are updated for any new component, token or behaviour.

Report honestly which checks ran, which did not, and why.

## 6. Repository rules

Follow [CONTRIBUTING.md](CONTRIBUTING.md):

- Small, specific changes.
- Forward-only migrations.
- No new dependencies without a reason.
- Commit the lockfile, never `node_modules` or `dist`.
- Never push, publish or commit unless asked.
- Do not change the CSP to make a feature work. Change the feature.
