# Component and shell contracts

Shared primitives live in `web/src/ui/`, shared helpers in `web/src/lib/`. Test
them through real application interactions, not private internals.

| Component | Contract |
| --- | --- |
| `Button` | Native button with `variant` primary, secondary, quiet, danger or danger-quiet; 44px target; optional `label` for icon-only use |
| `InputField` | Visible label linked by ID, hint below the control via `aria-describedby` |
| `Select` | The only dropdown, with variants field, chip and ghost. Options take a colour dot or icon, an optional hint and an optional description. It is a listbox popover with arrows, Home/End, typeahead and Esc (which closes only the menu). A visible `<label for=id>` or the `label` prop names it. |
| `DatePicker` | The only date input. A calendar popover with Today, Tomorrow, Next week and Clear, arrow-key grid navigation, today and overdue states, and the value as `YYYY-MM-DD` or `""`. |
| `MultiPicker` | A searchable checklist popover for assignees and labels. Each toggle applies immediately, and `oncreate` adds inline creation. |
| `Feedback` | Escaped text with neutral, success, warning or danger tone, always in words; the caller chooses status, alert or polite live text |
| `Modal` | `role=dialog`, labelled title, optional description, focus moves to the first field, Tab stays inside, Escape and scrim close it, focus returns to the opener. Bottom sheet below 40rem. |
| `CommandPalette` | Combobox and listbox with `aria-activedescendant`. Arrow keys, Home/End, Enter and Escape; opened with Ctrl/⌘ K, `/` or the search button. Offers only real destinations, the workspace's projects and preferences. |
| `Toast` | A transient success confirmation (polite live region) that floats bottom-right and leaves after 4.5s. Hover or focus pauses it, and it can be dismissed. Errors never use it: they stay inline until resolved. |
| `Icon` | One outline family (24px grid, 1.75 stroke), always `aria-hidden`; meaning comes from the adjacent label |
| `BrandMark`, `TrestleArt` | The trestle mark, and the overview linework (decorative, `aria-hidden`) |

## Shell

- A skip link targets the focusable `main`.
- The sidebar holds the brand, a workspace switcher and the labelled "Application" navigation: Dashboard, Projects and Members, or Workspaces, or Sign in; then Overview, Appearance and Instance status.
- The top bar holds breadcrumbs, search, a quick light/dark toggle and the account.
- Section links focus their headings, even when opened from another view.
- Shortcuts never fire while typing in a field or when a dialog is open. Ctrl/⌘ K or `/` opens search, and `C` creates a task in a project. Hints show only where a keyboard exists.
- On arrival Trestle opens the last-used workspace (`trestle.workspace`, browser-local), or the only one. "Switch workspace" always shows the chooser.
- The tab title names the current place, for example "Platform Core — Trestle".

## Work surfaces

- **Settings → Integration** selects an active Project in the current Workspace.
  Workspace owners/admins can save, replace, test, and disconnect Telegram, Slack,
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

- **Projects**: one card per project. It shows the key tile, the name and slug, live progress (open count, then done of total excluding cancelled, from bounded count queries), and when it was last updated. The key-sequence counter is shown only when live counts are unavailable. Archive asks for confirmation and explains that archiving keeps the project. A live filter appears above three projects.
- **Tasks** start with the **progress rail**. It shows project-wide totals per status category, read with bounded `limit=1` count queries, and its legend doubles as the category filter. The rail is a summary; the list stays authoritative.
- **Completion** means completed ÷ (all − cancelled) on every surface: the rail, the project cards and the dashboard.
- **List**: inset-card rows with key, title, description, labels, priority (dot and word), due date with an *Overdue* marker for open tasks, assignees, and a status menu styled as a chip in the status colour. Delete appears on row hover or focus for mouse users and stays visible on touch. Pagination appears only when there is more than one page.
- **Board**: one column per project status in workflow order, falling back to the five categories. Every card has a status menu, so dragging is an optional pointer shortcut and never required. The board loads up to 100 tasks and says so when more exist.
- **Task drawer**: an editable title (Enter saves) and description with an explicit save, an "Unsaved changes" marker, Discard, and a prompt before closing over unsaved edits. Everything else saves on change. The header holds copy key and delete (for owners and admins). It also has subtasks with progress, discussion (Ctrl/⌘ Enter posts), and an attribute sidebar for status, priority, due date, assignees, labels and activity. It is a full-screen sheet below 52rem.
- **Status manager**: create, rename, recolour and reorder (with buttons, not drag), and delete with a required remap target.
- **Members**: a table that becomes two-line rows on phones, plus the add-member form with role explanations and owner-only transfer behind password reconfirmation.

Empty, loading (skeletons plus a status message), filtered-empty, error and retry
states exist on every list. Destructive actions confirm and name what changes.
