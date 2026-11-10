<script lang="ts">
  import { onMount } from "svelte";
  import Button from "../ui/Button.svelte";
  import Feedback from "../ui/Feedback.svelte";
  import Icon from "../ui/Icon.svelte";
  import Modal from "../ui/Modal.svelte";
  import Avatar from "../ui/Avatar.svelte";

  export interface Workspace {
    id: string;
    name: string;
    slug: string;
    role: string;
    avatar_url?: string | null;
  }

  let {
    onselect,
    autoselect = false,
  }: {
    onselect?: (workspace: Workspace) => void;
    /** Open the remembered (or only) workspace without asking. */
    autoselect?: boolean;
  } = $props();

  const LAST_KEY = "trestle.workspace";

  function remembered(): string | null {
    try {
      return localStorage.getItem(LAST_KEY);
    } catch {
      return null;
    }
  }

  function choose(ws: Workspace) {
    try {
      localStorage.setItem(LAST_KEY, ws.id);
    } catch {
      // Remembering is a convenience only.
    }
    onselect?.(ws);
  }

  let workspaces = $state<Workspace[]>([]);
  let loading = $state(true);
  let errorMessage = $state<string | null>(null);
  let showCreate = $state(false);
  let name = $state("");
  let slug = $state("");
  let saving = $state(false);
  let createError = $state("");

  function closeCreate() {
    if (saving) return;
    if ((name || slug) && !confirm("Discard this unsaved workspace?")) return;
    showCreate = false;
  }

  async function createWorkspace(e: SubmitEvent) {
    e.preventDefault();
    if (saving) return;
    saving = true;
    createError = "";
    try {
      const csrf = document.cookie
        .split("; ")
        .find((c) => c.startsWith("trestle-csrf="))
        ?.slice("trestle-csrf=".length);
      const res = await fetch("/api/v1/workspaces", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-CSRF-Token": csrf || "",
        },
        body: JSON.stringify({
          name: name.trim(),
          slug: slug.trim().toLowerCase(),
        }),
      });
      const data = await res.json();
      if (!res.ok) {
        createError =
          data.error?.message || "Could not create workspace. Try again.";
        return;
      }
      showCreate = false;
      workspaces = [...workspaces, data];
      name = "";
      slug = "";
      choose(data);
    } catch {
      createError =
        "Could not create workspace. Check your connection and try again.";
    } finally {
      saving = false;
    }
  }

  async function loadWorkspaces() {
    loading = true;
    errorMessage = null;

    try {
      const response = await fetch("/api/v1/workspaces", {
        headers: { Accept: "application/json" },
      });

      if (response.status === 200) {
        const data = await response.json();
        workspaces = data.workspaces || [];
        if (autoselect) {
          const last = workspaces.find((w) => w.id === remembered());
          const only = workspaces.length === 1 ? workspaces[0] : undefined;
          const target = last ?? only;
          if (target) {
            choose(target);
            return;
          }
        }
      } else if (response.status === 401) {
        errorMessage = "Your session has expired. Please sign in again.";
      } else {
        errorMessage = "Could not load workspaces. Please try again.";
      }
    } catch {
      errorMessage = "Could not load workspaces. Please check your connection.";
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    void loadWorkspaces();
  });
</script>

<section aria-labelledby="workspaces-heading" class="workspaces">
  <div class="page-head">
    <div>
      <h2 id="workspaces-heading" tabindex="-1">Choose a workspace</h2>
      <p>Each workspace keeps its own members, projects and tasks.</p>
    </div>
    <div class="page-actions">
      <Button variant="primary" onclick={() => (showCreate = true)}>
        {#snippet children()}<Icon name="plus" size={16} />New workspace{/snippet}
      </Button>
    </div>
  </div>

  {#if loading}
    <div class="ws-grid" aria-hidden="true">
      {#each [0, 1, 2] as i (i)}
        <div class="ws-card skeleton"></div>
      {/each}
    </div>
    <Feedback tone="neutral" announcement="status">
      Loading your workspaces…
    </Feedback>
  {:else if errorMessage}
    <div class="stack">
      <Feedback tone="danger" announcement="alert">
        {errorMessage}
      </Feedback>
      <div>
        <Button onclick={loadWorkspaces}>
          {#snippet children()}Try again{/snippet}
        </Button>
      </div>
    </div>
  {:else if workspaces.length === 0}
    <div class="empty">
      <span class="empty-mark"><Icon name="layers" size={22} /></span>
      <h3>No workspaces yet</h3>
      <p>Create a workspace, or ask an owner to add you to an existing one.</p>
    </div>
  {:else}
    <ul class="ws-grid" role="list">
      {#each workspaces as ws (ws.id)}
        <li>
          <button type="button" class="ws-card" onclick={() => choose(ws)}>
            <Avatar
              id={ws.id}
              name={ws.name}
              avatarUrl={ws.avatar_url}
              size="lg"
              class="ws-tile"
            />
            <span class="ws-info">
              <span class="ws-name">{ws.name}</span>
              <span class="ws-slug">/{ws.slug}</span>
            </span>
            <span class="pill role-pill" data-role={ws.role}>{ws.role}</span>
            <span class="ws-open">
              Open
              <Icon name="arrow-right" size={16} />
            </span>
          </button>
        </li>
      {/each}
    </ul>
  {/if}
</section>

{#if showCreate}
  <Modal
    title="Create workspace"
    description="You will be its owner. Invite members after creating it."
    onclose={closeCreate}
  >
    {#if createError}<Feedback tone="danger" announcement="alert"
        >{createError}</Feedback
      >{/if}
    <form class="form-grid" onsubmit={createWorkspace}>
      <div class="field">
        <label for="workspace-name">Workspace name</label>
        <input
          id="workspace-name"
          bind:value={name}
          required
          maxlength="120"
          disabled={saving}
        />
      </div>
      <div class="field">
        <label for="workspace-slug">Workspace slug</label>
        <input
          id="workspace-slug"
          bind:value={slug}
          required
          minlength="2"
          maxlength="48"
          pattern="[a-z0-9][a-z0-9\-]*[a-z0-9]"
          disabled={saving}
          aria-describedby="workspace-slug-hint"
        />
        <p class="note" id="workspace-slug-hint">
          2–48 lowercase letters, numbers or hyphens.
        </p>
      </div>
      <div class="dialog-actions">
        <Button variant="quiet" onclick={closeCreate} disabled={saving}
          >{#snippet children()}Cancel{/snippet}</Button
        >
        <Button type="submit" variant="primary" disabled={saving}
          >{#snippet children()}{saving
              ? "Creating…"
              : "Create workspace"}{/snippet}</Button
        >
      </div>
    </form>
  </Modal>
{/if}

<style>
  .ws-grid {
    list-style: none;
    margin: 0 0 var(--space-4);
    padding: 0;
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(min(100%, 19rem), 1fr));
    gap: var(--space-4);
  }

  .ws-card {
    background:
      var(--card-fill) padding-box,
      var(--card-edge) border-box;
    border: var(--border-width) solid transparent;
    justify-content: stretch;
    width: 100%;
    min-height: 8.5rem;
    display: grid;
    grid-template-columns: auto minmax(0, 1fr) auto;
    grid-template-rows: auto 1fr auto;
    align-items: start;
    gap: var(--space-2) var(--space-3);
    padding: var(--space-5);
    border-radius: var(--radius-lg);
    text-align: left;
    white-space: normal;
    font-weight: 400;
  }

  .ws-card {
    transition:
      transform var(--duration-slow) var(--ease-spring),
      box-shadow var(--duration) var(--ease-out);
  }

  .ws-card:hover:not(:disabled) {
    transform: translateY(-3px);
    background: var(--surface-raised);
    border-color: var(--accent);
    box-shadow:
      var(--shadow-card),
      0 16px 40px -24px var(--accent-glow);
  }

  .ws-card.skeleton {
    background: var(--surface-sunken);
    box-shadow: none;
    animation: pulse 1.4s var(--ease-in-out) infinite alternate;
  }

  @keyframes pulse {
    to {
      opacity: 0.5;
    }
  }

  :global(.ws-tile) {
    border-radius: var(--radius);
    grid-row: span 2;
  }

  .ws-info {
    display: grid;
    gap: 0.125rem;
    min-width: 0;
  }

  .ws-name {
    font-size: var(--text-section);
    font-weight: 620;
    color: var(--text);
    letter-spacing: -0.012em;
  }

  .ws-slug {
    font-size: var(--text-small);
    color: var(--text-muted);
  }

  .role-pill {
    text-transform: capitalize;
  }

  .role-pill[data-role="owner"] {
    background: var(--accent-soft);
    color: var(--accent);
  }

  .ws-open {
    grid-column: 1 / -1;
    align-self: end;
    display: inline-flex;
    align-items: center;
    justify-content: flex-end;
    gap: var(--space-2);
    padding-top: var(--space-3);
    border-top: var(--border-width) solid var(--line);
    font-size: var(--text-small);
    font-weight: 600;
    color: var(--accent);
  }

  .ws-card:hover .ws-open :global(.svg-icon) {
    transform: translateX(3px);
  }

  .ws-open :global(.svg-icon) {
    transition: transform var(--transition-fast);
  }

  .empty :global(.feedback) {
    max-width: 30rem;
    text-align: left;
  }
</style>
