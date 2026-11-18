<script lang="ts">
  import { onMount } from "svelte";
  import Button from "../ui/Button.svelte";
  import Select from "../ui/Select.svelte";
  import DatePicker from "../ui/DatePicker.svelte";
  import Feedback from "../ui/Feedback.svelte";
  import Toast from "../ui/Toast.svelte";
  import MarkdownEditor from "../ui/MarkdownEditor.svelte";
  import { api, message } from "../lib/api";
  type Item = {
    id: string;
    kind: string;
    title: string;
    description: string;
    status: string;
    parent_id: string | null;
    project_id: string | null;
    owner_id: string | null;
    start_date: string | null;
    due_date: string | null;
    target: number;
    actual: number;
    archived: boolean;
    version: number;
    total_tasks: number;
    completed_tasks: number;
    cancelled_tasks: number;
  };
  type Task = {
    id: string;
    project_id: string;
    human_key: string;
    title: string;
    status: string;
  };
  let { workspace }: { workspace: { id: string; name: string; role: string } } =
    $props();
  const base = $derived(`/api/v1/workspaces/${workspace.id}/planning`);
  const canEdit = $derived(workspace.role !== "guest");
  const manager = $derived(["owner", "admin"].includes(workspace.role));
  const kinds = [
    { value: "goal", label: "Goals" },
    { value: "objective", label: "Objectives" },
    { value: "key_result", label: "Key results" },
    { value: "cycle", label: "Cycles" },
    { value: "module", label: "Modules" },
  ];
  const statuses = [
    { value: "planned", label: "Planned" },
    { value: "active", label: "Active" },
    { value: "at_risk", label: "At risk" },
    { value: "blocked", label: "Blocked" },
    { value: "completed", label: "Completed" },
    { value: "cancelled", label: "Cancelled" },
  ];
  let kind = $state("goal"),
    query = $state(""),
    archived = $state(false),
    offset = $state(0),
    more = $state(false),
    items = $state<Item[]>([]),
    current = $state<Item | null>(null),
    draft = $state<Item | null>(null),
    baseline = $state(""),
    creationKey = "";
  let loading = $state(false),
    busy = $state(false),
    error = $state(""),
    success = $state<string | null>(null),
    projects = $state<{ id: string; name: string }[]>([]),
    members = $state<{ user_id: string; display_name: string }[]>([]),
    parents = $state<Item[]>([]),
    parentQuery = $state("");
  let tasks = $state<Task[]>([]),
    taskOffset = $state(0),
    taskMore = $state(false),
    taskQuery = $state(""),
    matches = $state<Task[]>([]),
    children = $state<Item[]>([]),
    childOffset = $state(0),
    childMore = $state(false),
    measurement = $state<number | null>(null);
  const dirty = $derived(!!draft && JSON.stringify(draft) !== baseline);
  const editable = $derived(canEdit && !current?.archived);
  function canLeave() {
    return !busy && (!dirty || confirm("Discard unsaved planning changes?"));
  }
  async function load() {
    loading = true;
    error = "";
    try {
      const data = await api(
        `${base}?${new URLSearchParams({ kind, q: query, archived: String(archived), offset: String(offset) })}`,
      );
      items = data.items;
      more = data.has_more;
    } catch (e) {
      error = message(e);
    } finally {
      loading = false;
    }
  }
  async function metadata() {
    try {
      const [p, m] = await Promise.all([
        api(`/api/v1/workspaces/${workspace.id}/projects`),
        api(`/api/v1/workspaces/${workspace.id}/members`),
      ]);
      projects = p.projects;
      members = m.members;
    } catch (e) {
      error = message(e);
    }
  }
  function setItem(item: Item) {
    current = item;
    draft = structuredClone(item);
    baseline = JSON.stringify(draft);
    creationKey = crypto.randomUUID();
    tasks = [];
    children = [];
    parents = [];
    matches = [];
    measurement = null;
    taskOffset = 0;
    childOffset = 0;
    taskQuery = "";
    parentQuery = "";
  }
  async function relations() {
    if (!current?.id) return;
    const [t, c, d] = await Promise.all([
      api(`${base}/${current.id}/tasks?offset=${taskOffset}`),
      api(`${base}/${current.id}/children?offset=${childOffset}`),
      api(`${base}/${current.id}`),
    ]);
    tasks = t.items;
    taskMore = t.has_more;
    children = c.items;
    childMore = c.has_more;
    measurement = d.measurement_progress;
    current = d.item;
  }
  async function open(id: string) {
    if (!canLeave()) return;
    busy = true;
    error = "";
    try {
      const data = await api(`${base}/${id}`);
      setItem(data.item);
      await relations();
    } catch (e) {
      error = message(e);
    } finally {
      busy = false;
    }
  }
  function create(type = kind, parent: string | null = null) {
    if (!canLeave()) return;
    setItem({
      id: "",
      kind: type,
      title: "",
      description: "",
      status: "planned",
      parent_id: parent,
      project_id: current?.project_id || null,
      owner_id: null,
      start_date: null,
      due_date: null,
      target: 100,
      actual: 0,
      archived: false,
      version: 0,
      total_tasks: 0,
      completed_tasks: 0,
      cancelled_tasks: 0,
    });
    baseline = "";
    error = "";
  }
  async function save(archive = draft?.archived || false) {
    if (!draft || busy) return;
    busy = true;
    error = "";
    try {
      const data = await api(
        draft.id ? `${base}/${draft.id}` : base,
        draft.id ? "PUT" : "POST",
        { ...draft, archived: archive, creation_key: creationKey },
      );
      setItem(data);
      await relations();
      success = archive ? "Plan archived." : "Plan saved.";
    } catch (e) {
      error = message(e);
    } finally {
      busy = false;
    }
  }
  async function findParents() {
    if (!draft || busy) return;
    busy = true;
    error = "";
    try {
      const data = await api(
        `${base}?kind=${draft.kind === "objective" ? "goal" : "objective"}&q=${encodeURIComponent(parentQuery)}`,
      );
      parents = data.items;
    } catch (e) {
      error = message(e);
    } finally {
      busy = false;
    }
  }
  async function findTasks() {
    if (busy) return;
    busy = true;
    error = "";
    try {
      const data = await api(
        `/api/v1/workspaces/${workspace.id}/tasks?query=${encodeURIComponent(taskQuery)}`,
      );
      matches = (data.tasks || []).filter(
        (t: Task) =>
          !current?.project_id || t.project_id === current.project_id,
      );
    } catch (e) {
      error = message(e);
    } finally {
      busy = false;
    }
  }
  async function link(task: Task, remove = false) {
    if (!current || busy || dirty) return;
    if (
      !remove &&
      (current.kind === "cycle" || current.kind === "module") &&
      !confirm(
        `Assign ${task.human_key} here? This replaces its current ${current.kind} assignment.`,
      )
    )
      return;
    busy = true;
    error = "";
    try {
      await api(
        `${base}/${current.id}/tasks/${task.id}`,
        remove ? "DELETE" : "PUT",
        {},
      );
      await relations();
      success = remove ? "Task removed from plan." : "Task linked to plan.";
    } catch (e) {
      error = message(e);
    } finally {
      busy = false;
    }
  }
  async function paginateRelation(type: "tasks" | "children", delta: number) {
    if (busy) return;
    busy = true;
    error = "";
    if (type === "tasks") taskOffset += delta;
    else childOffset += delta;
    try {
      await relations();
    } catch (e) {
      if (type === "tasks") taskOffset -= delta;
      else childOffset -= delta;
      error = message(e);
    } finally {
      busy = false;
    }
  }
  onMount(() => {
    void load();
    void metadata();
    const guard = (e: Event) => {
      if (!canLeave()) e.preventDefault();
    };
    const unload = (e: BeforeUnloadEvent) => {
      if (dirty || busy) {
        e.preventDefault();
        e.returnValue = "";
      }
    };
    window.addEventListener("trestle:before-navigate", guard);
    window.addEventListener("beforeunload", unload);
    return () => {
      window.removeEventListener("trestle:before-navigate", guard);
      window.removeEventListener("beforeunload", unload);
    };
  });
</script>

<div class="page-head">
  <div>
    <h2>Planning</h2>
    <p>
      Connect goals, measurable results and project delivery in {workspace.name}.
    </p>
  </div>
  <div class="page-actions">
    {#if current}<Button
        disabled={busy}
        onclick={() => {
          if (canLeave()) {
            current = null;
            draft = null;
            void load();
          }
        }}>All plans</Button
      >{#if editable}<Button
          variant="primary"
          disabled={busy || !dirty}
          onclick={() => save()}>Save plan</Button
        >{/if}{:else if canEdit}<Button
        variant="primary"
        disabled={loading}
        onclick={() => create()}>Create plan</Button
      >{/if}
  </div>
</div>
{#if error}<Feedback tone="danger" announcement="alert">{error}</Feedback
  ><Button
    disabled={busy}
    onclick={() => {
      void metadata();
      if (!current) void load();
    }}>Retry loading</Button
  >{/if}
{#if !canEdit}<Feedback
    >Guests can read planning items. Ask a Workspace owner for editing access.</Feedback
  >{/if}
{#if draft && current}
  <div class="planning-layout">
    <section class="panel plan-form">
      <h3>{kinds.find((k) => k.value === draft?.kind)?.label}</h3>
      {#if dirty}<p>Unsaved changes</p>{/if}{#if current.archived}<p>
          Archived
        </p>{/if}
      <div class="field">
        <label for="plan-title">Title</label><input
          id="plan-title"
          bind:value={draft.title}
          maxlength="160"
          disabled={busy || !editable}
        />
      </div>
      <MarkdownEditor
        id="plan-description"
        label="Description"
        bind:value={draft.description}
        disabled={busy}
        readonly={!editable}
      />
      <div class="form-grid">
        <Select
          label="Plan status"
          value={draft.status}
          options={statuses}
          disabled={busy || !editable}
          onchange={(v) => {
            if (draft) draft.status = v;
          }}
        />
        <Select
          label="Project"
          value={draft.project_id || ""}
          options={[
            { value: "", label: "No project" },
            ...projects.map((p) => ({ value: p.id, label: p.name })),
          ]}
          disabled={busy || !editable}
          onchange={(v) => {
            if (draft) draft.project_id = v || null;
          }}
        />
        <Select
          label="Owner"
          value={draft.owner_id || ""}
          options={[
            { value: "", label: "Unassigned" },
            ...members.map((m) => ({
              value: m.user_id,
              label: m.display_name,
            })),
          ]}
          disabled={busy || !editable}
          onchange={(v) => {
            if (draft) draft.owner_id = v || null;
          }}
        />
        <DatePicker
          label="Start date"
          value={draft.start_date || ""}
          disabled={busy || !editable}
          onchange={(v) => {
            if (draft) draft.start_date = v || null;
          }}
        /><DatePicker
          label="Due date"
          value={draft.due_date || ""}
          disabled={busy || !editable}
          onchange={(v) => {
            if (draft) draft.due_date = v || null;
          }}
        />
      </div>
      {#if draft.kind === "key_result"}<div class="form-grid">
          <div class="field">
            <label for="plan-target">Target value</label><input
              id="plan-target"
              type="number"
              min="0.01"
              max="1000000000"
              step="any"
              bind:value={draft.target}
              disabled={busy || !editable}
            />
          </div>
          <div class="field">
            <label for="plan-actual">Actual value</label><input
              id="plan-actual"
              type="number"
              min="0"
              max="1000000000"
              step="any"
              bind:value={draft.actual}
              disabled={busy || !editable}
            />
          </div>
        </div>{/if}
      {#if draft.kind === "objective" || draft.kind === "key_result"}<div
          class="field"
        >
          <label for="plan-parent-query"
            >Find a parent {draft.kind === "objective"
              ? "goal"
              : "objective"}</label
          ><input
            id="plan-parent-query"
            bind:value={parentQuery}
            disabled={busy || !editable}
          />
        </div>
        <Button disabled={busy || !editable} onclick={findParents}
          >Find parent plans</Button
        ><Select
          label="Parent plan"
          value={draft.parent_id || ""}
          disabled={busy || !editable}
          options={[
            { value: "", label: "Choose a parent" },
            ...(draft.parent_id &&
            !parents.some((p) => p.id === draft?.parent_id)
              ? [{ value: draft.parent_id, label: "Current parent" }]
              : []),
            ...parents.map((p) => ({ value: p.id, label: p.title })),
          ]}
          onchange={(v) => {
            if (draft) draft.parent_id = v || null;
          }}
        />{/if}
    </section>
    <aside class="panel plan-summary">
      <h3>Delivery progress</h3>
      <p>
        {current.completed_tasks} completed / {Math.max(
          0,
          current.total_tasks - current.cancelled_tasks,
        )} eligible tasks
      </p>
      <progress
        aria-label="Task completion"
        max={Math.max(1, current.total_tasks - current.cancelled_tasks)}
        value={current.completed_tasks}
      ></progress>
      <p>Cancelled tasks are excluded.</p>
      {#if measurement !== null}<h3>Key result progress</h3>
        <p>{measurement.toFixed(1)}%</p>
        <progress
          aria-label="Key result completion"
          max="100"
          value={measurement}
        ></progress>
        <p>
          Average of active descendant key results, capped at 100% each.
        </p>{/if}
      {#if current.id}<Button disabled={busy} onclick={() => open(current!.id)}
          >Reload saved plan</Button
        >{#if current.parent_id}<Button
            disabled={busy}
            onclick={() => open(current!.parent_id!)}>Open parent plan</Button
          >{/if}{#if editable && (current.kind === "goal" || current.kind === "objective")}<Button
            disabled={busy}
            onclick={() =>
              create(
                current?.kind === "goal" ? "objective" : "key_result",
                current?.id,
              )}
            >Create {current.kind === "goal"
              ? "objective"
              : "key result"}</Button
          >{/if}{#if manager}<Button
            variant="danger-quiet"
            disabled={busy || dirty}
            onclick={() => {
              if (
                confirm(
                  current?.archived
                    ? "Restore this plan?"
                    : "Archive this plan? It can be restored; linked tasks and child plans are retained.",
                )
              )
                void save(!current?.archived);
            }}>{current.archived ? "Restore plan" : "Archive plan"}</Button
          >{/if}{/if}
    </aside>
  </div>
  {#if current.id}<section class="panel plan-form">
      <h3>Linked tasks</h3>
      {#if dirty}<Feedback
          >Save or reload your draft before changing task assignments.</Feedback
        >{/if}{#if editable}<form
          class="plan-toolbar"
          onsubmit={(e) => {
            e.preventDefault();
            void findTasks();
          }}
        >
          <div class="field">
            <label for="plan-task-query">Find a task by title or key</label
            ><input
              id="plan-task-query"
              bind:value={taskQuery}
              maxlength="100"
            />
          </div>
          <Button type="submit" disabled={busy || dirty}>Find tasks</Button>
        </form>
        <p>
          Search is bounded. Refine your query to find another task; only the
          selected Project is eligible.
        </p>
        {#each matches as task}<div class="plan-row">
            <span>{task.human_key} · {task.title}</span><Button
              disabled={busy || dirty}
              onclick={() => link(task)}>Assign task</Button
            >
          </div>{/each}{/if}
      {#each tasks as task}<div class="plan-row">
          <a href={`#task/${workspace.id}/${task.project_id}/${task.id}`}
            >{task.human_key} · {task.title}</a
          ><span>{task.status}</span>{#if editable}<Button
              disabled={busy || dirty}
              onclick={() => link(task, true)}>Remove assignment</Button
            >{/if}
        </div>{:else}<p>No linked tasks on this page.</p>{/each}
      <div class="plan-toolbar">
        <Button
          disabled={busy || taskOffset === 0}
          onclick={() => paginateRelation("tasks", -50)}>Previous tasks</Button
        ><Button
          disabled={busy || !taskMore}
          onclick={() => paginateRelation("tasks", 50)}>Next tasks</Button
        >
      </div>
    </section>
    {#if current.kind === "goal" || current.kind === "objective"}<section
        class="panel plan-form"
      >
        <h3>{current.kind === "goal" ? "Objectives" : "Key results"}</h3>
        {#each children as child}<div class="plan-row">
            <span
              >{child.title} · {child.status}{child.kind === "key_result"
                ? ` · ${child.actual} / ${child.target}`
                : ""}</span
            ><Button disabled={busy} onclick={() => open(child.id)}
              >Open plan</Button
            >
          </div>{:else}<p>No child plans on this page.</p>{/each}
        <div class="plan-toolbar">
          <Button
            disabled={busy || childOffset === 0}
            onclick={() => paginateRelation("children", -50)}
            >Previous children</Button
          ><Button
            disabled={busy || !childMore}
            onclick={() => paginateRelation("children", 50)}
            >Next children</Button
          >
        </div>
      </section>{/if}{/if}
{:else}
  <form
    class="plan-toolbar"
    onsubmit={(e) => {
      e.preventDefault();
      offset = 0;
      void load();
    }}
  >
    <Select
      label="Planning area"
      value={kind}
      options={kinds}
      disabled={loading}
      onchange={(v) => {
        kind = v;
        offset = 0;
        void load();
      }}
    /><Select
      label="Plan archive filter"
      value={String(archived)}
      options={[
        { value: "false", label: "Active plans" },
        { value: "true", label: "Archived plans" },
      ]}
      disabled={loading}
      onchange={(v) => {
        archived = v === "true";
        offset = 0;
        void load();
      }}
    />
    <div class="field">
      <label for="plan-search">Search plans</label><input
        id="plan-search"
        bind:value={query}
        maxlength="200"
      />
    </div>
    <Button type="submit" disabled={loading}>Search</Button>
  </form>
  {#if loading}<Feedback announcement="status">Loading plans…</Feedback>
    <div
      class="panel plan-skeleton"
      aria-hidden="true"
    ></div>{:else if !error}{#each items as item}<article
        class="card plan-card"
      >
        <div>
          <h3>{item.title}</h3>
          <p>
            {item.status.replaceAll("_", " ")} · {item.start_date ||
              "No start date"} → {item.due_date || "No deadline"}
          </p>
          <p>
            {item.kind === "key_result"
              ? `${item.actual} / ${item.target}`
              : `${item.completed_tasks} / ${item.total_tasks - item.cancelled_tasks} tasks complete`}
          </p>
        </div>
        <Button disabled={busy} onclick={() => open(item.id)}>Open plan</Button>
      </article>{:else}<div class="empty">
        <h3>No plans found</h3>
        <p>Create a plan or change your search.</p>
        {#if canEdit}<Button onclick={() => create()}>Create plan</Button>{/if}
      </div>{/each}{/if}
  {#if more || offset}<div class="plan-toolbar">
      <Button
        disabled={loading || offset === 0}
        onclick={() => {
          offset -= 50;
          void load();
        }}>Previous page</Button
      >
      <p>Page {offset / 50 + 1}</p>
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
  .planning-layout {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(14rem, 20rem);
    gap: var(--space-4);
  }
  .plan-form,
  .plan-summary {
    padding: var(--space-5);
    min-width: 0;
    margin-bottom: var(--space-4);
    display: grid;
    gap: var(--space-4);
    align-content: start;
  }
  .plan-summary {
    align-self: start;
  }
  .plan-summary progress {
    width: 100%;
    accent-color: var(--accent);
  }
  .form-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(min(100%, 14rem), 1fr));
    gap: var(--space-4);
  }
  .plan-toolbar,
  .plan-row,
  .plan-card {
    display: flex;
    gap: var(--space-3);
    align-items: center;
    flex-wrap: wrap;
    margin-block: var(--space-4);
  }
  .plan-toolbar {
    align-items: end;
  }
  .plan-card {
    padding: var(--space-4);
  }
  .plan-card > div,
  .plan-row > span:first-child,
  .plan-row > a {
    flex: 1 1 12rem;
    min-width: 0;
    overflow-wrap: anywhere;
  }
  .plan-skeleton {
    min-height: 10rem;
    background: var(--surface-sunken);
  }
  @media (max-width: 65rem) {
    .planning-layout {
      grid-template-columns: minmax(0, 1fr);
    }
  }
</style>
