<script lang="ts">
  import { onMount } from "svelte";
  import Button from "../ui/Button.svelte";
  import Select from "../ui/Select.svelte";
  import Feedback from "../ui/Feedback.svelte";
  import Toast from "../ui/Toast.svelte";
  import Icon from "../ui/Icon.svelte";
  import Modal from "../ui/Modal.svelte";
  import { relativeTime } from "../lib/identity";
  import BoardEditor from "./BoardEditor.svelte";
  import WhiteboardEditor from "./WhiteboardEditor.svelte";
  import {
    validateScene,
    convertDiagramXML,
    convertWhiteboardJSON,
    type Board,
    type Scene,
  } from "./scene";

  let {
    workspace,
    mode = "all",
    oneditingchange = () => {},
  }: {
    workspace: { id: string; name: string; role: string };
    mode?: "diagram" | "whiteboard" | "all";
    oneditingchange?: (editing: boolean) => void;
  } = $props();
  const canEdit = $derived(workspace.role !== "guest");
  const canArchive = $derived(["owner", "admin"].includes(workspace.role));
  const base = $derived(`/api/v1/workspaces/${workspace.id}/boards`);
  let list = $state<Board[]>([]);
  let loading = $state(true);
  let error = $state("");
  let success = $state<string | null>(null);
  let archived = $state(false);
  let offset = $state(0);
  let more = $state(false);
  let filter = $state("");
  let viewMode = $state<"grid" | "list">("grid");
  let current = $state<Board | null>(null);
  let creationKey = "";
  let title = $state("");
  let scene = $state<Scene>({ version: 1, elements: [] });
  let baseline = $state("");
  let saving = $state(false);
  let opening = $state(false);
  const dirty = $derived(
    !!current && JSON.stringify({ title, scene }) !== baseline,
  );
  const visible = $derived(
    list.filter((b) => b.title.toLowerCase().includes(filter.toLowerCase())),
  );
  $effect(() => oneditingchange(!!current));

  async function request(path: string, method = "GET", body?: unknown) {
    const csrf =
      document.cookie
        .split(";")
        .map((p) => p.trim())
        .find((p) => p.startsWith("trestle-csrf="))
        ?.slice(13) || "";
    let res: Response;
    try {
      res = await fetch(path, {
        method,
        cache: "no-store",
        headers: {
          Accept: "application/json",
          ...(body
            ? { "Content-Type": "application/json", "X-CSRF-Token": csrf }
            : {}),
        },
        ...(body ? { body: JSON.stringify(body) } : {}),
      });
    } catch {
      throw new Error(
        "Could not reach the server. Your draft is unchanged. Check your connection and try again.",
      );
    }
    const data = await res.json().catch(() => null);
    if (!res.ok || !data)
      throw new Error(
        data?.error?.message ||
          "Could not load or save this board. Check your connection and access, then try again.",
      );
    return data;
  }
  async function load() {
    loading = true;
    error = "";
    try {
      const kindParam = mode !== "all" ? `&kind=${mode}` : "";
      const data = await request(
        `${base}?archived=${archived}&offset=${offset}${kindParam}`,
      );
      list = data.boards;
      more = data.has_more;
    } catch (e) {
      error =
        e instanceof Error ? e.message : "Could not load boards. Try again.";
    } finally {
      loading = false;
    }
  }
  function setBoard(board: Board) {
    creationKey = crypto.randomUUID();
    current = board;
    title = board.title;
    scene = structuredClone(board.scene || { version: 1, elements: [] });
    baseline = board.id ? JSON.stringify({ title, scene }) : "";
    error = "";
  }
  function create(kind?: Board["kind"]) {
    const k = kind || (mode === "whiteboard" ? "whiteboard" : "diagram");
    setBoard({
      id: "",
      workspace_id: workspace.id,
      title: k === "diagram" ? "Untitled diagram" : "Untitled whiteboard",
      kind: k,
      version: 0,
      archived: false,
      updated_at: "",
    });
  }
  async function open(board: Board) {
    opening = true;
    error = "";
    try {
      setBoard(await request(`${base}/${board.id}`));
    } catch (e) {
      error =
        e instanceof Error ? e.message : "Could not open board. Try again.";
    } finally {
      opening = false;
    }
  }
  function canLeave() {
    if (saving) return false;
    if (dirty && canEdit) {
      void save(false);
    }
    return true;
  }
  function back() {
    if (!canLeave()) return;
    current = null;
    error = "";
    void load();
  }
  async function save(archive = current?.archived || false) {
    if (!current || saving) return;
    saving = true;
    error = "";
    try {
      const data = await request(
        current.id ? `${base}/${current.id}` : base,
        current.id ? "PUT" : "POST",
        {
          title,
          kind: current.kind,
          scene,
          version: current.version,
          archived: archive,
          creation_key: creationKey,
        },
      );
      current = data;
      title = data.title;
      scene = data.scene;
      baseline = JSON.stringify({ title, scene });
      success = archive ? "Board archived." : "Board saved.";
    } catch (e) {
      error =
        e instanceof Error
          ? e.message
          : "Could not save board. Your draft is unchanged. Try again.";
    } finally {
      saving = false;
    }
  }
  async function importJSON(event: Event) {
    const input = event.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    try {
      if (file.size > 2000000)
        throw new Error("Choose a file no larger than 2 MB.");
      const text = await file.text();
      let importedKind: Board["kind"] =
        mode === "whiteboard" ? "whiteboard" : "diagram";
      let importedTitle =
        file.name.replace(/\.[^.]+$/, "") ||
        (mode === "whiteboard" ? "Imported whiteboard" : "Imported diagram");
      let importedScene: Scene = { version: 1, elements: [] };

      if (
        text.trim().startsWith("<") ||
        file.name.endsWith(".xml") ||
        file.name.endsWith(".drawio")
      ) {
        const elements = convertDiagramXML(text);
        if (!elements.length)
          throw new Error("Could not find diagram shapes in this XML file.");
        importedKind = "diagram";
        importedScene = { version: 1, elements };
      } else {
        const data = JSON.parse(text);
        if (
          data.type === "whiteboard" ||
          data.type === "excalidraw" ||
          (Array.isArray(data.elements) &&
            data.elements.some((e: any) => e.type))
        ) {
          const elements = convertWhiteboardJSON(data.elements || []);
          if (!elements.length)
            throw new Error(
              "Could not find whiteboard elements in this JSON file.",
            );
          importedKind = "whiteboard";
          importedScene = { version: 1, elements };
        } else if (
          data.format === "trestle-board" &&
          data.version === 1 &&
          validateScene(data.scene)
        ) {
          importedKind = data.kind || importedKind;
          importedTitle = data.title || importedTitle;
          importedScene = data.scene;
        } else {
          throw new Error(
            "Choose a valid diagram XML or whiteboard JSON file.",
          );
        }
      }

      setBoard({
        id: "",
        workspace_id: workspace.id,
        title: importedTitle.slice(0, 120),
        kind: importedKind,
        scene: importedScene,
        version: 0,
        archived: false,
        updated_at: "",
      });
    } catch (e) {
      error = e instanceof Error ? e.message : "Could not import board.";
    } finally {
      input.value = "";
    }
  }
  onMount(() => {
    void load();
    const unload = (e: BeforeUnloadEvent) => {
      if (dirty || saving) {
        e.preventDefault();
        e.returnValue = "";
      }
    };
    const navigate = (e: Event) => {
      if (!canLeave()) e.preventDefault();
      else current = null;
    };
    window.addEventListener("beforeunload", unload);
    window.addEventListener("trestle:before-navigate", navigate);
    return () => {
      window.removeEventListener("beforeunload", unload);
      window.removeEventListener("trestle:before-navigate", navigate);
      oneditingchange(false);
    };
  });

  let boardToDelete = $state<Board | null>(null);
  let deleting = $state(false);

  async function confirmDeleteBoard() {
    if (!boardToDelete || deleting) return;
    deleting = true;
    error = "";
    try {
      const csrf =
        document.cookie
          .split(";")
          .map((p) => p.trim())
          .find((p) => p.startsWith("trestle-csrf="))
          ?.slice(13) || "";
      const res = await fetch(`${base}/${boardToDelete.id}`, {
        method: "DELETE",
        headers: { Accept: "application/json", "X-CSRF-Token": csrf },
      });
      if (!res.ok) {
        const data = await res.json().catch(() => null);
        throw new Error(data?.error?.message || "Could not delete board.");
      }
      if (current?.id === boardToDelete.id) {
        current = null;
        oneditingchange(false);
      }
      success = `${boardToDelete.kind === "diagram" ? "Diagram" : "Whiteboard"} deleted.`;
      boardToDelete = null;
      void load();
    } catch (e) {
      error = e instanceof Error ? e.message : "Could not delete board.";
    } finally {
      deleting = false;
    }
  }
</script>

{#if current}
  {#if current.kind === "whiteboard"}
    <WhiteboardEditor
      board={current}
      workspaceName={workspace.name}
      bind:title
      bind:scene
      {dirty}
      {saving}
      {error}
      {canEdit}
      {canArchive}
      onback={back}
      onsave={save}
      onreload={() => {
        if (current && canLeave()) void open(current);
      }}
      ondelete={() => {
        if (current) {
          boardToDelete = current;
          void confirmDeleteBoard();
        }
      }}
    />
  {:else}
    <BoardEditor
      board={current}
      workspaceName={workspace.name}
      bind:title
      bind:scene
      {dirty}
      {saving}
      {error}
      {canEdit}
      {canArchive}
      onback={back}
      onsave={save}
      onreload={() => {
        if (current && canLeave()) void open(current);
      }}
      ondelete={() => {
        if (current) {
          boardToDelete = current;
          void confirmDeleteBoard();
        }
      }}
    />
  {/if}
{:else}
  <div class="page-head">
    <div>
      <h2>
        {mode === "diagram"
          ? "Diagrams"
          : mode === "whiteboard"
            ? "Whiteboards"
            : "Boards"}
      </h2>
      <p>
        {mode === "diagram"
          ? `Design system architectures, flowcharts, and diagrams in ${workspace.name}.`
          : mode === "whiteboard"
            ? `Brainstorm, sketch, and collaborate on infinite digital whiteboards in ${workspace.name}.`
            : `Map a process or sketch an idea in ${workspace.name}.`}
      </p>
    </div>
    {#if canEdit}
      <div class="page-actions">
        {#if mode === "diagram"}
          <Button
            variant="primary"
            disabled={opening}
            onclick={() => create("diagram")}>New diagram</Button
          >
          <label class="btn import-button"
            >Import diagram<input
              type="file"
              accept=".json,.xml,application/json,text/xml"
              disabled={opening}
              onchange={importJSON}
            /></label
          >
        {:else if mode === "whiteboard"}
          <Button
            variant="primary"
            disabled={opening}
            onclick={() => create("whiteboard")}>New whiteboard</Button
          >
          <label class="btn import-button"
            >Import whiteboard<input
              type="file"
              accept=".json,application/json"
              disabled={opening}
              onchange={importJSON}
            /></label
          >
        {:else}
          <Button
            variant="primary"
            disabled={opening}
            onclick={() => create("diagram")}>New diagram</Button
          ><Button disabled={opening} onclick={() => create("whiteboard")}
            >New whiteboard</Button
          ><label class="btn import-button"
            >Import board<input
              type="file"
              accept=".json,.xml,application/json,text/xml"
              disabled={opening}
              onchange={importJSON}
            /></label
          >
        {/if}
      </div>
    {/if}
  </div>
  {#if !canEdit}<Feedback
      >Guests can view and export Workspace boards. Ask a Workspace owner to
      change your role to edit.</Feedback
    >{/if}
  <div class="board-toolbar">
    <div class="segmented" role="group" aria-label="Layout mode">
      <button
        type="button"
        aria-pressed={viewMode === "grid"}
        onclick={() => (viewMode = "grid")}
      >
        <Icon name="grid" size={16} />
        <span>Grid</span>
      </button>
      <button
        type="button"
        aria-pressed={viewMode === "list"}
        onclick={() => (viewMode = "list")}
      >
        <Icon name="list" size={16} />
        <span>List</span>
      </button>
    </div>
    <Select
      label="Board archive filter"
      variant="ghost"
      value={String(archived)}
      options={[
        {
          value: "false",
          label:
            mode === "diagram"
              ? "Active diagrams"
              : mode === "whiteboard"
                ? "Active whiteboards"
                : "Active boards",
        },
        {
          value: "true",
          label:
            mode === "diagram"
              ? "Archived diagrams"
              : mode === "whiteboard"
                ? "Archived whiteboards"
                : "Archived boards",
        },
      ]}
      onchange={(v) => {
        archived = v === "true";
        offset = 0;
        void load();
      }}
      disabled={loading || opening}
    />
    <div class="field">
      <label for="board-filter">Filter this page</label><input
        id="board-filter"
        bind:value={filter}
        type="search"
      />
    </div>
  </div>
  {#if error}<Feedback tone="danger">{error}</Feedback><Button onclick={load}
      >Retry boards</Button
    >{/if}
  {#if loading}<Feedback announcement="status">Loading boards…</Feedback>
    <div class="panel skeleton-board" aria-hidden="true"></div>
  {:else if !error && !visible.length}<div class="empty">
      <span class="empty-mark tile"><Icon name="layers" /></span>
      <h3>
        {filter
          ? mode === "diagram"
            ? "No matching diagrams"
            : mode === "whiteboard"
              ? "No matching whiteboards"
              : "No matching boards"
          : archived
            ? mode === "diagram"
              ? "No archived diagrams"
              : mode === "whiteboard"
                ? "No archived whiteboards"
                : "No archived boards"
            : mode === "diagram"
              ? "No diagrams yet"
              : mode === "whiteboard"
                ? "No whiteboards yet"
                : "No boards yet"}
      </h3>
      <p>
        {filter
          ? "Try another title or clear the filter."
          : mode === "diagram"
            ? "Create a diagram for an architecture or process."
            : mode === "whiteboard"
              ? "Create a whiteboard to sketch an idea."
              : "Create a diagram for a process or a whiteboard for a sketch."}
      </p>
      {#if filter}<Button onclick={() => (filter = "")}>Clear filter</Button
        >{:else if canEdit && !archived}<Button
          onclick={() =>
            create(mode === "whiteboard" ? "whiteboard" : "diagram")}
          >{mode === "whiteboard"
            ? "Create whiteboard"
            : "Create diagram"}</Button
        >{/if}
    </div>
  {:else if !error}
    {#if viewMode === "grid"}
      <div class="board-grid">
        {#each visible as b (b.id)}
          <article class="card board-card grid-card">
            <div class="card-header">
              <span class="tile">
                <Icon name={b.kind === "diagram" ? "workflow" : "pencil"} />
              </span>
              <div class="card-meta">
                <span class="board-kind-badge"
                  >{b.kind === "diagram" ? "Diagram" : "Whiteboard"}</span
                >
                <span
                  class="board-time"
                  title={b.updated_at
                    ? new Date(b.updated_at).toLocaleString()
                    : ""}
                >
                  {b.updated_at
                    ? `Modified ${relativeTime(b.updated_at)}`
                    : "Recently"}
                </span>
              </div>
            </div>
            <h3 class="card-title">{b.title}</h3>
            <div class="card-footer">
              <div class="card-actions">
                <Button disabled={opening} onclick={() => open(b)}>Open</Button>
                {#if canArchive}
                  <Button
                    variant="danger-quiet"
                    class="btn-icon"
                    disabled={opening}
                    label={`Delete ${b.title}`}
                    onclick={() => (boardToDelete = b)}
                  >
                    <Icon name="trash" size={16} />
                  </Button>
                {/if}
              </div>
            </div>
          </article>
        {/each}
      </div>
    {:else}
      <div class="board-list">
        {#each visible as b (b.id)}
          <article class="card board-card list-row">
            <span class="tile">
              <Icon name={b.kind === "diagram" ? "workflow" : "pencil"} />
            </span>
            <div class="row-main">
              <h3 class="row-title">{b.title}</h3>
              <div class="row-sub">
                <span class="board-kind-badge"
                  >{b.kind === "diagram" ? "Diagram" : "Whiteboard"}</span
                >
                <span
                  class="board-time"
                  title={b.updated_at
                    ? new Date(b.updated_at).toLocaleString()
                    : ""}
                >
                  {b.updated_at
                    ? `Modified ${relativeTime(b.updated_at)}`
                    : "Recently"}
                </span>
              </div>
            </div>
            <div class="card-actions">
              <Button disabled={opening} onclick={() => open(b)}>Open</Button>
              {#if canArchive}
                <Button
                  variant="danger-quiet"
                  class="btn-icon"
                  disabled={opening}
                  label={`Delete ${b.title}`}
                  onclick={() => (boardToDelete = b)}
                >
                  <Icon name="trash" size={16} />
                </Button>
              {/if}
            </div>
          </article>
        {/each}
      </div>
    {/if}
  {/if}
  {#if offset || more}<div class="board-toolbar">
      <Button
        disabled={loading || offset === 0}
        onclick={() => {
          offset -= 50;
          void load();
        }}>Previous page</Button
      >
      <p>Showing up to 50 boards per page.</p>
      <Button
        disabled={loading || !more}
        onclick={() => {
          offset += 50;
          void load();
        }}>Next page</Button
      >
    </div>{/if}
{/if}
{#if boardToDelete}
  <Modal
    title={`Delete ${boardToDelete.kind === "diagram" ? "diagram" : "whiteboard"}`}
    description={`Are you sure you want to permanently delete “${boardToDelete.title}”? This action cannot be undone.`}
    onclose={() => (boardToDelete = null)}
  >
    <div class="dialog-actions">
      <Button
        variant="quiet"
        onclick={() => (boardToDelete = null)}
        disabled={deleting}>Cancel</Button
      >
      <Button variant="danger" onclick={confirmDeleteBoard} disabled={deleting}>
        {deleting ? "Deleting…" : "Delete board"}
      </Button>
    </div>
  </Modal>
{/if}
{#if success}<Toast message={success} onclear={() => (success = null)} />{/if}

<style>
  .board-toolbar {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-3);
    margin-block: var(--space-4);
  }
  .board-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
    gap: var(--space-4);
  }
  .board-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }
  .board-card {
    padding: var(--space-4);
    display: flex;
    position: relative;
    border-radius: var(--radius-md);
  }
  .grid-card {
    flex-direction: column;
    gap: var(--space-3);
    min-height: 140px;
  }
  .grid-card .card-header {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }
  .grid-card .card-meta {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .grid-card .card-title {
    font-size: var(--text-base);
    font-weight: 580;
    margin: 0;
    line-height: var(--line-snug);
    overflow-wrap: anywhere;
    flex: 1 1 auto;
  }
  .grid-card .card-footer {
    display: flex;
    align-items: center;
    justify-content: flex-end;
    margin-top: auto;
    padding-top: var(--space-2);
  }
  .list-row {
    flex-direction: row;
    align-items: center;
    gap: var(--space-4);
    padding: var(--space-3) var(--space-4);
  }
  .list-row .row-main {
    flex: 1 1 auto;
    min-width: 0;
    display: flex;
    flex-direction: column;
    gap: 2px;
  }
  .list-row .row-title {
    font-size: var(--text-base);
    font-weight: 580;
    margin: 0;
    overflow-wrap: anywhere;
  }
  .list-row .row-sub {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }
  .board-kind-badge {
    display: inline-block;
    font-size: var(--text-xs);
    font-weight: 540;
    color: var(--text-secondary);
    background: var(--surface-sunken);
    padding: 1px 6px;
    border-radius: var(--radius-sm);
    border: var(--border-width) solid var(--line);
    width: fit-content;
  }
  .board-time {
    font-size: var(--text-xs);
    color: var(--text-muted);
  }
  .card-actions {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    margin-left: auto;
  }
  .skeleton-board {
    min-height: 10rem;
    background: var(--surface-sunken);
  }
  .import-button {
    position: relative;
    overflow: hidden;
    cursor: pointer;
  }
  .import-button input {
    position: absolute;
    inset: 0;
    width: 100%;
    opacity: 0;
    cursor: pointer;
  }
  .import-button:focus-within {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
  }
</style>
