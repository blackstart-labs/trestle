<script lang="ts">
  import { onMount } from "svelte";
  import Button from "../ui/Button.svelte";
  import Select from "../ui/Select.svelte";
  import Feedback from "../ui/Feedback.svelte";
  import Toast from "../ui/Toast.svelte";
  import Icon from "../ui/Icon.svelte";
  import { relativeTime } from "../lib/identity";
  import BoardEditor from "./BoardEditor.svelte";
  import { validateScene, type Board, type Scene } from "./scene";

  let {
    workspace,
    oneditingchange = () => {},
  }: {
    workspace: { id: string; name: string; role: string };
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
      const data = await request(
        `${base}?archived=${archived}&offset=${offset}`,
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
  function create(kind: Board["kind"]) {
    setBoard({
      id: "",
      workspace_id: workspace.id,
      title: kind === "diagram" ? "Untitled diagram" : "Untitled whiteboard",
      kind,
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
    return (
      !saving &&
      (!dirty ||
        confirm(
          "Discard unsaved board changes? Export your draft first if you want to keep them.",
        ))
    );
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
      if (file.size > 1000000)
        throw new Error("Choose a board JSON file no larger than 1 MB.");
      const data = JSON.parse(await file.text());
      if (
        data.format !== "trestle-board" ||
        data.version !== 1 ||
        !["diagram", "whiteboard"].includes(data.kind) ||
        typeof data.title !== "string" ||
        !data.title.trim() ||
        [...data.title].length > 120 ||
        !validateScene(data.scene)
      )
        throw new Error(
          "Choose a valid Trestle board JSON export (version 1).",
        );
      setBoard({
        id: "",
        workspace_id: workspace.id,
        title: data.title,
        kind: data.kind,
        scene: data.scene,
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
</script>

{#if current}
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
  />
{:else}
  <div class="page-head">
    <div>
      <h2>Boards</h2>
      <p>Map a process or sketch an idea in {workspace.name}.</p>
    </div>
    {#if canEdit}<div class="page-actions">
        <Button
          variant="primary"
          disabled={opening}
          onclick={() => create("diagram")}>New diagram</Button
        ><Button disabled={opening} onclick={() => create("whiteboard")}
          >New whiteboard</Button
        ><label class="btn import-button"
          >Import board<input
            type="file"
            accept="application/json,.json"
            disabled={opening}
            onchange={importJSON}
          /></label
        >
      </div>{/if}
  </div>
  {#if !canEdit}<Feedback
      >Guests can view and export Workspace boards. Ask a Workspace owner to
      change your role to edit.</Feedback
    >{/if}
  <div class="board-toolbar">
    <Select
      label="Board archive filter"
      variant="ghost"
      value={String(archived)}
      options={[
        { value: "false", label: "Active boards" },
        { value: "true", label: "Archived boards" },
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
          ? "No matching boards"
          : archived
            ? "No archived boards"
            : "No boards yet"}
      </h3>
      <p>
        {filter
          ? "Try another title or clear the filter."
          : "Create a diagram for a process or a whiteboard for a sketch."}
      </p>
      {#if filter}<Button onclick={() => (filter = "")}>Clear filter</Button
        >{:else if canEdit && !archived}<Button
          onclick={() => create("diagram")}>Create diagram</Button
        >{/if}
    </div>
  {:else if !error}<div class="board-list">
      {#each visible as b (b.id)}<article class="card board-card">
          <span class="tile"
            ><Icon name={b.kind === "diagram" ? "workflow" : "pencil"} /></span
          >
          <div>
            <h3>{b.title}</h3>
            <p>
              {b.kind === "diagram" ? "Diagram" : "Whiteboard"} · Updated {relativeTime(
                b.updated_at,
              )}
            </p>
          </div>
          <Button disabled={opening} onclick={() => open(b)}>Open board</Button>
        </article>{/each}
    </div>{/if}
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
{#if success}<Toast message={success} onclear={() => (success = null)} />{/if}

<style>
  .board-toolbar {
    display: flex;
    flex-wrap: wrap;
    align-items: end;
    gap: var(--space-3);
    margin-block: var(--space-4);
  }
  .board-list {
    display: grid;
    gap: var(--space-3);
  }
  .board-card {
    padding: var(--space-4);
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-4);
  }
  .board-card > div {
    flex: 1 1 12rem;
    min-width: 0;
    overflow-wrap: anywhere;
  }
  .board-card p {
    font-size: var(--text-small);
    margin-top: var(--space-2);
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
