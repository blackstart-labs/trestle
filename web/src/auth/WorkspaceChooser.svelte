<script lang="ts">
  import { onMount } from "svelte";
  import Button from "../ui/Button.svelte";
  import Feedback from "../ui/Feedback.svelte";

  export interface Workspace {
    id: string;
    name: string;
    slug: string;
    role: string;
  }

  let {
    onselect,
  }: {
    onselect?: (workspace: Workspace) => void;
  } = $props();

  let workspaces = $state<Workspace[]>([]);
  let loading = $state(true);
  let errorMessage = $state<string | null>(null);

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

<section aria-labelledby="workspaces-heading">
  <p class="eyebrow">Workspaces</p>
  <h2 id="workspaces-heading" tabindex="-1">Choose a workspace</h2>
  <p class="lead">
    Select a workspace to open your projects, tasks and boards.
  </p>

  {#if loading}
    <Feedback tone="neutral" announcement="status">
      Loading your workspaces…
    </Feedback>
  {:else if errorMessage}
    <Feedback tone="danger" announcement="alert">
      {errorMessage}
    </Feedback>
    <Button onclick={loadWorkspaces}>
      {#snippet children()}Try again{/snippet}
    </Button>
  {:else if workspaces.length === 0}
    <Feedback tone="warning">
      You are not currently a member of any workspaces. Contact your
      administrator to be added to a workspace.
    </Feedback>
  {:else}
    <ul class="workspace-list" role="list">
      {#each workspaces as ws (ws.id)}
        <li class="workspace-card">
          <div class="workspace-info">
            <span class="workspace-name">{ws.name}</span>
            <span class="workspace-slug">/{ws.slug}</span>
          </div>
          <div class="workspace-actions">
            <span class="role-badge" data-role={ws.role}>{ws.role}</span>
            <Button onclick={() => onselect?.(ws)}>
              {#snippet children()}Open{/snippet}
            </Button>
          </div>
        </li>
      {/each}
    </ul>
  {/if}
</section>

<style>
  .workspace-list {
    list-style: none;
    padding: 0;
    margin: var(--space-4) 0;
    display: grid;
    gap: var(--space-3);
    max-width: 36rem;
  }
  .workspace-card {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-4);
    background: var(--surface-raised);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius);
    gap: var(--space-4);
  }
  .workspace-info {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }
  .workspace-name {
    font-weight: 650;
    font-size: var(--text-lead);
  }
  .workspace-slug {
    font-family: var(--font-mono);
    font-size: var(--text-small);
    color: var(--text-secondary);
  }
  .workspace-actions {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }
  .role-badge {
    font-size: var(--text-small);
    font-family: var(--font-mono);
    padding: var(--space-1) var(--space-2);
    border-radius: var(--radius);
    border: var(--border-width) solid var(--border);
    background: var(--surface);
    text-transform: capitalize;
  }
</style>
