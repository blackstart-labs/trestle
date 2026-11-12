# Component and shell contracts

Shared primitives live in `web/src/ui/`, shared helpers in `web/src/lib/`. Test
them through real application interactions, not private internals.

| Component | Contract |
| --- | --- |
| `Button` | Native button with `variant` primary, secondary, quiet, danger or danger-quiet; 44px target; optional `label` for icon-only use |
| `InputField` | Visible label linked by ID, hint below the control via `aria-describedby` |
| `MarkdownEditor` | Task descriptions in creation and the drawer. Write/Preview modes, selection formatting for bold, italic, headings, lists, links and code; preview uses the unsaved draft and never saves. Read-only users see rendered content. Raw HTML is escaped, output sanitized, and images become alt text with no requests. Wide code/tables scroll inside `.markdown-preview`. Markdown source (including significant whitespace) is stored unchanged. |
| `Select` | The only dropdown, with variants field, chip and ghost. Options take a colour dot or icon, an optional hint and an optional description. It is a listbox popover with arrows, Home/End, typeahead and Esc (which closes only the menu). A visible `<label for=id>` or the `label` prop names it. |
| `DatePicker` | The only date input. A calendar popover with Today, Tomorrow, Next week and Clear, arrow-key grid navigation, today and overdue states, and the value as `YYYY-MM-DD` or `""`. The trigger keeps its icon, date text and chevron on one line at the shared control height, including when no date is selected; placeholder styling must not use the global `.empty` surface class. |
| `MultiPicker` | A searchable checklist popover for assignees and labels. Each toggle applies immediately, and `oncreate` adds inline creation. `disabled` prevents opening while saving. |
| `Feedback` | Escaped text with neutral, success, warning or danger tone, always in words; the caller chooses status, alert or polite live text |
| `Modal` | `role=dialog`, labelled title, optional description, focus moves to the first field, Tab stays inside, Escape and scrim close it, focus returns to the opener. Bottom sheet below 40rem. |
| `CommandPalette` | Combobox and listbox with `aria-activedescendant`. Arrow keys, Home/End, Enter and Escape; opened with Ctrl/⌘ K, `/` or the search button. Offers only real destinations, the workspace's projects and preferences. |
| `Toast` | A transient success confirmation (polite live region) that floats bottom-right and leaves after 4.5s. Hover or focus pauses it, and it can be dismissed. Errors never use it: they stay inline until resolved. |
| `Icon` | One outline family (24px grid, 1.75 stroke), always `aria-hidden`; meaning comes from the adjacent label |
| `Avatar` | Person or workspace avatar in sizes `sm`, `md` and `lg`. Renders custom image when `avatarUrl` is present, falling back gracefully to initials with deterministic `identityHue` if missing or failing to load. Always `aria-hidden` when decorative beside text. |
| `BrandMark`, `TrestleArt` | The trestle mark, and the overview linework (decorative, `aria-hidden`) |

## Shell

- A skip link targets the focusable `main`.
- The sidebar holds the brand, a workspace switcher and the labelled "Application" navigation: Dashboard, Projects, Members and Boards, or Workspaces, or Sign in; then Overview, Appearance and Instance status.
- The top bar holds breadcrumbs, search, a quick light/dark toggle and the account.
- The top bar is opaque in both themes. The trestle mark and same-origin SVG favicon share the deck-and-braces geometry.
- A running timer shows the task key, “Tracking”, elapsed time and “Stop & log”, with 44px controls. Discard and switching tasks confirm the loss of unlogged time. Failed saves stay inline; saving disables duplicate logs across all timer surfaces.
- Section links focus their headings, even when opened from another view.
- Shortcuts never fire while typing in a field or when a dialog is open. Ctrl/⌘ K or `/` opens search, and `C` creates a task in a project. Hints show only where a keyboard exists.
- On arrival Trestle opens the last-used workspace (`trestle.workspace`, browser-local), or the only one. "Switch workspace" always shows the chooser.
- The chooser offers “New workspace” even when empty. Any signed-in user can create a Workspace and becomes its owner. Creation retains failed drafts, prevents duplicate submission and guards closing. Settings → Workspace lets owners/admins edit its name, slug and icon; other roles get an explanation. Changed settings guard navigation and reload.
- Task links (`#task/{workspace}/{project}/{task}`, also the expanded Workspace/Project/Task form) wait for sign-in and resolve the requested Workspace before rendering. Startup selection must not replace the linked Project. Failed links offer retry.
- The tab title names the current place, for example "Platform Core — Trestle".

## Work surfaces

- **Boards**: Workspace diagrams and freehand whiteboards use a full-window
  canvas-first editor with a collapsible shape library, formatting toolbar,
  contained canvas stage and an optional properties panel. Semantic ink colors
  and Mona Sans are unchanged. `boards/BoardCanvas.svelte` renders validated
  object scenes, not user-provided SVG; `boards/BoardEditor.svelte` owns direct
  manipulation and `boards/scene.ts` owns geometry, validation and export. The
  properties panel supplies object selection, coordinate/size editing,
  connection selection and coordinate-based stroke entry, so dragging is
  optional. Keyboard shortcuts never fire in fields, dialogs or menus. Explicit
  saves retain failed/conflicting drafts; archive is reversible and
  manager-only. Guests can inspect/export but cannot edit. New content is empty,
  never seeded with example business data. See [Boards](../boards.md) for limits.
  Shell `go()` and sign-out dispatch the cancelable
  `trestle:before-navigate` window event; the active board cancels navigation
  during saves or when the user declines to discard a dirty draft. Page reloads
  use `beforeunload`. Palette opening alone does not discard content.

- **Settings → Integration** selects an active Project in the current Workspace.
  Workspace owners/admins can save, replace, test, and disconnect Telegram (including optional Topic ID for forum supergroups), Slack,
  and Discord connections. It uses Select, Modal, Feedback, and Toast; credentials
  are password fields, never returned after saving. Dirty forms guard closing,
  failures preserve inputs, and disconnect explicitly confirms credential removal.
  Loading, retry, no-Project, signed-out, and permission-limited states are explicit.
  Settings section links and the command palette open `#integration` and focus its
  heading. Provider rows wrap actions on phones using the same tokens in both themes.
  Telegram, Slack, and Discord use their outline brand marks from `Icon`, with
  visible names. The adapted Tabler path license ships at `/assets/icon-licenses.txt`.
  Integration is a dedicated Settings section. A Project context rail describes
  event scope and shared metadata beside three separate provider panels; it moves
  above the providers below 52rem of available content width, including with enlarged
  text. The active Settings section is marked in words
  and with an underline. Appearance and Instance status retain their shared view.

- **Dashboard** is the workspace home, and all of it is computed from real tasks, reading at most 12 projects × 100 tasks and saying so when it samples:
  - stat cards for active projects, open tasks (with a status mini-rail), due this week (a seven-day bar chart and the overdue count) and completion (a ring)
  - **My work**: your open assigned tasks, the most urgent first
  - **Deadlines**: a 14-day timeline of real due dates
  - project rows with progress

  Selecting a task opens its project with that task's drawer already open.

- **Projects**: one card per project. It shows the key tile or uploaded project icon, the name and slug, live progress (open count, then done of total excluding cancelled, from bounded count queries), and when it was last updated. The key-sequence counter is shown only when live counts are unavailable. Both project creation and editing support uploading a project icon (PNG, JPEG, WebP, GIF, or SVG up to 5 MB) stored directly with the project and served over same-origin endpoints. Owners/admins have an Edit action on cards and Edit project in the task header. Both open `EditProjectModal`: prefilled name, slug, description, and project icon with preview and remove option, a read-only project key, explicit Save project, unsaved-change confirmation, saving/disabled state and persistent errors with drafts retained. Successful updates reconcile the card or the open project/header/breadcrumbs from the server response. Archive asks for confirmation and explains that archiving keeps the project. A live filter appears above three projects.
- **Tasks** start with the **progress rail**. It shows project-wide totals per status category, read with bounded `limit=1` count queries, and its legend doubles as the category filter. The rail is a summary; the list stays authoritative.
- The task header shows the Project name aligned with its icon, without repeating its description or slug. Descriptions remain available on Project cards and in the editor. Task descriptions accept up to 15,000 Unicode characters on creation and editing.
- New task drafts include an assignee MultiPicker and a milestone Select, even when no milestones exist. Selected people stay visible; metadata failures offer retry. Assignees and milestone are saved atomically with the Task. List and Board show API-provided direct subtask counts, independent of pagination and filtering, refreshed after the drawer closes.
- **Completion** means completed ÷ (all − cancelled) on every surface: the rail, the project cards and the dashboard.
- **List**: inset-card rows with key, title, description, labels, priority (dot and word), due date with an *Overdue* marker for open tasks, assignees, and a status menu styled as a chip in the status colour. Delete appears on row hover or focus for mouse users and stays visible on touch. Pagination appears only when there is more than one page.
- **Board**: one column per project status in workflow order, falling back to the five categories. Every card has a status menu, so dragging is an optional pointer shortcut and never required. The board loads up to 100 tasks and says so when more exist.
- **Task drawer**: an editable title (Enter saves) and description with an explicit save, an "Unsaved changes" marker, Discard, and a prompt before closing over unsaved edits. Everything else saves on change. The header holds copy key and delete (for owners and admins). It also has subtasks with progress, discussion (Ctrl/⌘ Enter posts), and an attribute sidebar for status, priority, due date, assignees, labels and activity. It is a full-screen sheet below 52rem.
- **Status manager**: create, rename, recolour and reorder (with buttons, not drag), and delete with a required remap target. Up/down moves save immediately, update positional values for the modal, Board and status pickers, and disable repeated moves while saving. A failed save restores the previous order and shows an inline error so the move can be retried.
- **Milestones**: planning targets grouping tasks toward an outcome and date. Accessible from the project header action "Milestones", opening `MilestoneModal`. Lists project milestones with live status badge ("Open", "Closed"), target due date, task counts ("X of Y completed (Z%)") and a progress rail. Members can create, edit (title, description, target date, open/closed status), and delete milestones (with plain-word confirmation explaining tasks will become unassigned rather than deleted). In task views, tasks assigned to a milestone display a `.milestone-badge` chip with a flag icon. The toolbar offers a milestone filter dropdown, the task creation modal provides an optional milestone selector, and the task drawer allows assigning or clearing milestones immediately.
- **Members**: a table that becomes two-line rows on phones, plus the add-member form with role explanations and owner-only transfer behind password reconfirmation.

Empty, loading (skeletons plus a status message), filtered-empty, error and retry
states exist on every list. Destructive actions confirm and name what changes.

Task property saves PATCH only the selected property. Title/description drafts
survive property, assignee and label refreshes; details save only changed text.
While saving, repeat writes are disabled. A failed write keeps the draft and the
last saved property value with a persistent inline error. Activity records status
and priority only when their persisted values change, including moves between
two Statuses in the same Status category; identical writes do not add entries.
The task list responds to available container width, including enlarged text,
so its columns become stacked rows before fixed columns overflow the page.
Segmented controls and narrow task action rows wrap when enlarged text needs another line; project header actions use whole, readable buttons rather than squeezing labels into vertical fragments.
Preview links open in a new tab with opener/referrer isolation, preserving drafts.
