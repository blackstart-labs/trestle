# Documents

Open **Documents** from Workspace navigation or the command palette. Create a blank page, Meeting notes, or Project brief. Edit Markdown with the existing Write/Preview editor and choose **Save document**. Failed saves keep your draft. Export Markdown before discarding a draft you want to retain.

## Available workflows

- Create/edit pages with up to 100,000 Unicode content characters, a 160-character title and 20 unique tags (40 characters each).
- Search titles and content across the Workspace; filter by a tag or archive status. Lists are paginated in 50-item pages, with no content payload in listing results.
- Create child pages, choose a parent through search, browse children and open the parent. Parent cycles and foreign-Workspace parents are rejected. Archiving a page leaves children available; choose another active parent before editing a child of an archived parent.
- Link to a saved document using **Copy link**, then paste `[[UUID|label]]` into Markdown. The Linked documents section opens those links; Backlinks lists active incoming pages. UUID identities survive renaming. Labels do not automatically rename. Cross-Workspace/missing IDs do not create resolved backlinks; opening them returns not found. References inside code blocks also count as links in this initial version.
- Every save creates an immutable revision. **Revision history → Load revision → Save document** restores content as a new version; history is retained.
- Owners/admins archive and restore pages. Guests read/export, while members can create/edit active pages. There are no per-page ACLs or public sharing.
- Import `.md`, `.markdown` or `.txt` (400 KB maximum; content limit also applies). Export the exact editor Markdown, including unsaved text. Markdown export does not encode tags, hierarchy or history. Workspace JSON export includes current active and archived document data; database backups preserve revisions and audit history.
- Leaving the page or reloading warns about unsaved work. Saving in two tabs uses version preconditions: the second stale writer receives 409 and retains its draft. Creation keys prevent duplicate pages when retrying an identical creation after losing a response.

## Knowledge graph

Choose **Knowledge graph** to view up to 200 recently updated active pages. Title and tag filters restrict both nodes and connections. Zoom/pan/reset controls adjust the SVG viewport; the button list opens documents with keyboard access. A truncation notice appears when more pages match. This is a bounded circular layout, not a full-workspace force simulation.

## API

All routes are under `/api/v1/workspaces/{workspace_id}/documents` and require an authenticated Workspace membership. Mutations require the existing CSRF token.

| Method / suffix | Contract |
|---|---|
| GET `/` | `q` (up to 200 characters), `tag`, `archived=true/false`, optional `parent_id`, `offset` (0–100000). Returns `documents`, `has_more`, `offset`, `limit`. |
| GET `/graph` | `q` title substring and `tag`; returns `nodes`, `edges`, `truncated` for at most 200 pages. |
| POST `/` | `title`, `content`, `tags`, nullable `parent_id`, unique UUID `creation_key`. Returns 201. |
| GET `/{id}` | Full saved page. |
| PUT `/{id}` | Full editable fields plus current `version` and `archived`; returns the incremented version. |
| GET `/{id}/versions` | Paginated revision summaries, `offset`. |
| GET `/{id}/versions/{version}` | Full immutable snapshot; submit its content through PUT using the current page version to restore. |
| GET `/{id}/backlinks` | Paginated active incoming pages, `offset`. |
| DELETE `/{id}` | Permanently delete document. Sub-notes are automatically and safely re-parented to the deleted document's parent. Emits `document.deleted` audit event. |

SQL migration `202611160017_documents.sql` is forward-only. Composite foreign keys enforce Workspace scope for parents and links. A Workspace row lock serializes hierarchy changes; optimistic versions reject lost updates. Mutation, history, link indexing and audit events commit atomically. Search uses bounded literal substring matching, not a full-text ranking engine; benchmark before large deployments. Revision retention is currently unlimited; operators must plan storage and backups.

## Explicit limits

This is a Markdown knowledge foundation, inspired by Trilium's 3-pane hierarchical organization and bi-directional linking. Permanent deletion is supported via `DELETE /{id}` with safe sub-tree re-parenting and confirmation dialog. The larger master brief remains a roadmap.

## Verification

The release handoff lists checks actually run. `web/e2e-real/documents.spec.ts` covers real sign-in, persistence, nesting/backlinks, conflict handling, navigation guards, restoration, archive/export, and both themes at 390/768/1440px with axe and a 320px enlarged-text reflow check. It requires a disposable PostgreSQL database named `trestle_e2e_*`. Browser tests must run on a host that permits Chromium's local sockets. Use `TRESTLE_BROWSER_EXECUTABLE` only when selecting an installed compatible Chromium; otherwise Playwright uses its bundled browser.
