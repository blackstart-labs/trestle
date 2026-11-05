# Copilot instructions

Follow [AGENTS.md](../AGENTS.md) for every change. It is the project's
contract for AI-assisted work. Essentials for frontend code in `web/`:

- Use semantic tokens from `web/src/tokens.css` and never raw colours. Use the
  global classes from `web/src/app.css` (`.panel`, `.inset`, `.tile`,
  `.page-head`, `.empty`, `.pill`, `.status-chip`, `.priority`).
- Use the shared components in `web/src/ui/`:
  - `Button` (with `variant`)
  - `Select`, never a native `<select>`
  - `DatePicker`, never `type="date"`
  - `MultiPicker`, `Modal`, `Toast` (success) and `Feedback` (errors)
  - `Icon`; never an icon package
- No `style="…"` attributes (CSP). Use `style:prop={value}`. No CDN or other
  third-party assets.
- Motion only from `web/src/lib/motion.ts` and the `--ease-*` tokens. It is
  critically damped, exits mirror entrances, and only transform and opacity
  animate.
- Never invent metrics, AI features or "coming soon" controls. Numbers come
  from the API.
- WCAG 2.2 AA: label every control, 44px targets, full keyboard support, AA
  contrast in light and dark.
- Before finishing, run in `web/`: `npm run lint:design`, `npm run check`,
  `npm test`, `npm run format:check`, `npm run build`.
