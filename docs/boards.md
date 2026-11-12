# Boards: diagrams and whiteboards

Boards are native Workspace content. Open **Boards** in the sidebar or search
for Boards, diagram or whiteboard in the command palette. No third-party service,
new runtime dependency or relaxed CSP is required.

## Available workflows

- Create a Diagram or Whiteboard and give it a title. Editing uses the full
  window: a collapsible shape library on the left, a formatting toolbar, a
  canvas-first stage, and optional context properties on the right.
- The shape library groups Standard, Flowchart and Container symbols: process,
  decision, start/end, document, database, input/output, ellipse, text, sticky
  note, frame and swimlane. Click to place a shape at the viewport centre or
  drag it onto the page. Search, Layers and drag-to-place are optional paths.
- Select one or more objects on the canvas. Drag to move; Shift-click or drag a
  marquee to extend the selection; drag a corner handle to resize. Arrow keys
  move by 1 unit (Shift: 10). Double-click a shape to edit its label inline.
- Format the selection from the toolbar: semantic fill and line colour, line
  width, dash, font size, bold, text alignment and connector routing. The
  properties panel adds exact position/size, object label, arrowheads and
  accessible object selection. Every canvas action has a standard-control path.
- Connect shapes with the connector tool: choose a source, then a destination,
  or drag between the connection points. Arrows follow their shapes and support
  elbow, straight and curved routing plus end/both/none arrowheads. Removing a
  shape also removes its connections, with confirmation and undo.
- Arrange selected objects: align left/centre/right/top/middle/bottom,
  distribute horizontally/vertically, bring to front/send to back, group and
  ungroup, lock/unlock, copy/paste and duplicate. Locked objects cannot be moved,
  resized, edited or erased until unlocked.
- Draw freehand with mouse, pen or touch using the pen tool. Alternatively,
  enter space-separated `x,y` pairs in **Stroke points**, then choose **Add
  stroke from points**.
- Templates offer an editable basic flowchart or three-lane process. A template
  replaces the current drawing only after confirmation and is undoable.
- Undo/redo the last 50 scene changes in the current editing session. Canvas
  Ctrl/Command Z undoes; Shift Ctrl/Command Z redoes. Title changes are ordinary
  text-field edits, not part of scene undo history.
- Pan by dragging with the hand tool, the middle button, Alt-drag, the wheel or
  Shift-wheel. Ctrl/Command-wheel zooms around the pointer. The footer offers
  zoom presets, Fit drawing, Grid and Snap. Wide drawings stay inside their own
  stage on narrow screens, not the whole page.
- Save explicitly with the Save button or Ctrl/Command S. Failed saves retain
  the draft. Navigating away, signing out, closing the editor or reloading warns
  about unsaved edits. Reloading a saved board intentionally discards the draft
  after confirmation; export it first.
- Export the complete editable draft as versioned Trestle JSON, and the whole
  drawing as SVG or PNG in its current theme colors. Import creates an unsaved
  copy in the current Workspace, never overwrites an existing board.
- Owners/admins can archive and restore a saved board from the Export menu.
  Archive is reversible; permanent deletion is intentionally not offered. Save
  or discard changes before archiving. Archived boards remain readable and
  exportable.

All Workspace roles can read active and archived boards. Owners, admins and
members can create/edit; guests are read-only. Boards are Workspace-wide, not
private or restricted to a Project. No public sharing, Project/Task attachment,
multiplayer cursors, live merge, image embeds, automatic layout, Lucidchart
import or `.excalidraw` compatibility is implemented. Diagram and Whiteboard use
the same object editor; the type describes the board's purpose and its default
ink and fill.

## Persistence and limits

Forward migration `202611110014_boards.sql` creates `boards`. Apply migrations
before serving this build. Existing tables and content are unchanged. Mutation
and security-audit insertion share one transaction. Scene content is not copied
into audit metadata. Workspace JSON exports include complete active **and**
archived board scenes in the same repeatable-read snapshot as other content.
PostgreSQL backups remain the complete-instance recovery path; Workspace JSON
exports are not complete restorable instance backups.

- Scene format version 1; up to 500 objects and 20,000 total stroke points.
- Coordinates 0–4000 on each axis; shapes fit within 4000 × 4000, minimum 20 × 20.
- Labels up to 200 Unicode characters; title up to 120. Text is plain text,
  never interpreted as HTML/SVG or loaded as a URL.
- Pointer strokes collect at most 2,000 points per gesture. Longer strokes can
  be split into gestures; coordinate imports share the total point budget.
- Import files must be valid Trestle board exports smaller than or equal to 1 MB.
- Lists return at most 50 board summaries per page, without scene payloads. The
  title filter applies to the current page and is labeled accordingly.
- Undo history is browser-session-only and holds 50 steps. Unsaved work is not a
  durable backup; save or export before closing. There is no automatic
  multi-author merging, and a stale version returns a conflict instead of
  overwriting the other editor.
- SVG/PNG exports include only board content, not selection handles, ports,
  marquee or the text editor overlay. Exported colors follow the active theme;
  switch theme before exporting if a particular palette is required.

## API

All routes require a session; writes also require CSRF protection. Cross-Workspace
access returns a concealed 404. Guest writes return 403.

| Method | Workspace-scoped path | Contract |
| --- | --- | --- |
| GET | `/api/v1/workspaces/{ws}/boards?archived=false&offset=0` | `{boards, has_more, limit:50, offset}`; offset 0–100000 |
| POST | `/api/v1/workspaces/{ws}/boards` | `{title, kind, scene, creation_key}`; kind `diagram` or `whiteboard`, UUID creation key |
| GET | `/api/v1/workspaces/{ws}/boards/{id}` | Full board including scene, archived flag and version |
| PUT | `/api/v1/workspaces/{ws}/boards/{id}` | `{title, kind, scene, version, archived}`; kind immutable, version required |

Creation retries with the same key, actor and unchanged initial payload return
the same board, without repeating its audit event. Reusing a creation key for a
different payload conflicts. Updates lock/check the version and return 409 for a
stale save; neither editor's work is silently overwritten. Export a conflicted
draft before reloading the saved version. A lost update response can therefore
produce a conflict on retry, not a second mutation. Archived content must be
restored before editing, and only managers may change its archived flag.

Scene elements have `id`, `kind`, `x`, `y`, `width`, `height`, `text`, `color`,
plus optional `fill`, `strokeWidth`, `dashed`, `fontSize`, `bold`, `align`,
`group` and `locked`. Arrows have `from`/`to` shape IDs and optional `route`,
`fromPort`, `toPort` and `heads`; strokes have `points: [{x,y}]`. Supported kinds
are `rectangle`, `ellipse`, `diamond`, `text`, `stroke`, `arrow`, `terminator`,
`document`, `database`, `parallelogram`, `note`, `frame` and `swimlane`.
Supported colors and fills are `ink`, `accent`, `success`, `warning`, `danger`
and `info` (fill also accepts `none`), resolved through theme tokens. Line widths
are 1–8, font sizes 12–72, group IDs at most 64 bytes. Duplicate IDs,
dangling/self arrows, unsupported types/colors, nonfinite/out-of-range
coordinates and oversized scenes are rejected server-side.

## Verification

The regular fixture-backed Playwright suite covers diagrams, pointer/coordinate
drawing, JSON round trips, failed/conflicting saves, guest UI, navigation guards,
axe WCAG checks, both themes and responsive reflow. Backend tests exercise real
PostgreSQL isolation, CSRF/roles, version races, archive/restore, exports,
creation retries and audit-failure rollback.

For a real browser → Go API → PostgreSQL journey, first build the frontend and
create a **disposable** database named `trestle_e2e_*`. From `web/`:

```bash
npm run build
TRESTLE_E2E_DATABASE_URL='postgres://test_user:test_password@127.0.0.1:5432/trestle_e2e_boards?sslmode=disable' npm run test:e2e:real
```

This separate configuration refuses other database names, applies migrations,
bootstraps test-only `boardtester` credentials and serves on loopback port 4318.
It does not drop or reset the supplied database. Use a fresh database with no
existing users on its first run, and remove your disposable database afterwards.
Never point either database test suite at production infrastructure.
