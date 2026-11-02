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

<section aria-labelledby="workspaces-heading" class="workspaces-container">
  <div class="workspaces-header">
    <p class="eyebrow">Workspaces</p>
    <h2 id="workspaces-heading" tabindex="-1">Choose a workspace</h2>
    <p class="lead">
      Select an active workspace to access your projects, tasks, and teams.
    </p>
  </div>

  {#if loading}
    <div class="feedback-wrapper">
      <Feedback tone="neutral" announcement="status">
        Loading your workspaces…
      </Feedback>
    </div>
  {:else if errorMessage}
    <div class="feedback-wrapper">
      <Feedback tone="danger" announcement="alert">
        {errorMessage}
      </Feedback>
      <div class="retry-action">
        <Button onclick={loadWorkspaces}>
          {#snippet children()}Try again{/snippet}
        </Button>
      </div>
    </div>
  {:else if workspaces.length === 0}
    <div class="feedback-wrapper">
      <Feedback tone="warning">
        You are not currently a member of any workspaces. Contact your
        administrator to be added to a workspace.
      </Feedback>
    </div>
  {:else}
    <ul class="workspace-grid" role="list">
      {#each workspaces as ws (ws.id)}
        <li class="workspace-card">
          <div class="card-lead">
            <div class="ws-avatar" aria-hidden="true">
              {ws.name.slice(0, 1).toUpperCase()}
            </div>
            <div class="workspace-info">
              <span class="workspace-name">{ws.name}</span>
              <span class="workspace-slug">/{ws.slug}</span>
            </div>
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
  .workspaces-container {
    max-width: 48rem;
    margin: 0 auto;
    padding: var(--space-4) 0;
  }

  .workspaces-header {
    margin-bottom: var(--space-6);
  }

  .feedback-wrapper {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .retry-action {
    margin-top: var(--space-2);
  }

  .workspace-grid {
    list-style: none;
    padding: 0;
    margin: 0;
    display: grid;
    gap: var(--space-3);
  }

  .workspace-card {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-4) var(--space-5);
    background: var(--surface-raised);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
    transition: all var(--transition-fast);
    gap: var(--space-4);
  }

  .workspace-card:hover {
    border-color: var(--accent);
    transform: translateY(-1px);
    box-shadow: var(--shadow-card);
  }

  .card-lead {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    min-width: 0;
  }

  .ws-avatar {
    width: 2.75rem;
    height: 2.75rem;
    border-radius: var(--radius-md);
    background: linear-gradient(135deg, #4f46e5 0%, #7c3aed 100%);
    color: #ffffff;
    font-weight: 700;
    font-size: 1.125rem;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
    box-shadow: 0 4px 10px -2px rgba(79, 70, 229, 0.3);
  }

  .workspace-info {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
    min-width: 0;
  }

  .workspace-name {
    font-weight: 700;
    font-size: 1.125rem;
    color: var(--text);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .workspace-slug {
    font-family: var(--font-mono);
    font-size: 0.8125rem;
    color: var(--text-muted);
  }

  .workspace-actions {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    flex-shrink: 0;
  }

  .role-badge {
    font-size: 0.75rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    padding: 0.2rem 0.5rem;
    border-radius: var(--radius-full);
    background: var(--selected);
    color: var(--text-secondary);
    border: var(--border-width) solid var(--border);
  }

  .role-badge[data-role="owner"] {
    background: rgba(79, 70, 229, 0.1);
    color: var(--accent);
    border-color: var(--accent);
  }

  .role-badge[data-role="admin"] {
    background: rgba(14, 165, 233, 0.1);
    color: var(--focus);
  }
</style>
