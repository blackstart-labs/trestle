<script lang="ts">
  import { onMount } from "svelte";
  import Button from "../ui/Button.svelte";
  import InputField from "../ui/InputField.svelte";
  import Feedback from "../ui/Feedback.svelte";

  export interface Project {
    id: string;
    workspace_id: string;
    name: string;
    slug: string;
    key: string;
    description: string;
    task_counter: number;
    archived: boolean;
    created_at: string;
    updated_at: string;
  }

  let {
    workspace,
    onselectproject,
  }: {
    workspace: { id: string; name: string; slug: string; role: string };
    onselectproject?: (project: Project) => void;
  } = $props();

  let projects = $state<Project[]>([]);
  let loading = $state(true);
  let errorMessage = $state<string | null>(null);
  let successMessage = $state<string | null>(null);

  // New project modal state
  let showCreate = $state(false);
  let createName = $state("");
  let createSlug = $state("");
  let createKey = $state("");
  let createDescription = $state("");
  let slugManual = $state(false);
  let keyManual = $state(false);
  let creating = $state(false);
  let createError = $state<string | null>(null);

  const canManageProjects = $derived(
    workspace.role === "owner" || workspace.role === "admin",
  );

  function getCsrfToken(): string | null {
    if (typeof document === "undefined") return null;
    for (const part of document.cookie.split(";")) {
      const trimmed = part.trim();
      if (trimmed.startsWith("trestle-csrf=")) {
        return trimmed.slice("trestle-csrf=".length);
      }
    }
    return null;
  }

  $effect(() => {
    if (!slugManual && createName) {
      createSlug = createName
        .toLowerCase()
        .replace(/[^a-z0-9]+/g, "-")
        .replace(/^-+|-+$/g, "")
        .slice(0, 48);
    }
  });

  $effect(() => {
    if (!keyManual && createName) {
      const words = createName.trim().split(/\s+/).filter(Boolean);
      if (words.length >= 2) {
        createKey = words
          .slice(0, 4)
          .map((w) => w[0]?.toUpperCase() || "")
          .join("")
          .slice(0, 8);
      } else if (words.length === 1 && words[0]) {
        createKey = words[0].slice(0, 4).toUpperCase();
      }
    }
  });

  export async function loadProjects() {
    loading = true;
    errorMessage = null;
    try {
      const res = await fetch(`/api/v1/workspaces/${workspace.id}/projects`, {
        headers: { Accept: "application/json" },
        cache: "no-store",
      });
      if (res.status === 200) {
        const data = await res.json();
        projects = data.projects || [];
      } else if (res.status === 404) {
        errorMessage = "Workspace not found or access denied.";
      } else {
        errorMessage = "Could not load projects. Please try again.";
      }
    } catch {
      errorMessage = "Could not load projects. Please check your connection.";
    } finally {
      loading = false;
    }
  }

  async function handleCreateProject(e: Event) {
    e.preventDefault();
    createError = null;

    const trimmedName = createName.trim();
    const trimmedSlug = createSlug.trim().toLowerCase();
    const trimmedKey = createKey.trim().toUpperCase();

    if (!trimmedName) {
      createError = "Project name is required.";
      return;
    }
    if (trimmedSlug.length < 2 || trimmedSlug.length > 48) {
      createError =
        "Project slug must be 2-48 lowercase letters, numbers, or hyphens.";
      return;
    }
    if (trimmedKey.length < 2 || trimmedKey.length > 8) {
      createError = "Project key must be 2-8 uppercase letters or numbers.";
      return;
    }

    const csrf = getCsrfToken();
    if (!csrf) {
      createError = "Security token missing. Please refresh and try again.";
      return;
    }

    creating = true;
    try {
      const res = await fetch(`/api/v1/workspaces/${workspace.id}/projects`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-CSRF-Token": csrf,
          Accept: "application/json",
        },
        body: JSON.stringify({
          name: trimmedName,
          slug: trimmedSlug,
          key: trimmedKey,
          description: createDescription.trim(),
        }),
      });

      if (res.status === 201) {
        const newProj: Project = await res.json();
        projects = [...projects, newProj];
        successMessage = `Project "${newProj.name}" created successfully.`;
        showCreate = false;
        createName = "";
        createSlug = "";
        createKey = "";
        createDescription = "";
        slugManual = false;
        keyManual = false;
      } else if (res.status === 409) {
        createError =
          "A project with this slug or key already exists in this workspace.";
      } else if (res.status === 403) {
        createError =
          "Only workspace owners and administrators can create projects.";
      } else {
        const data = await res.json().catch(() => null);
        createError =
          data?.error?.message ||
          "Could not create project. Please check input values.";
      }
    } catch {
      createError = "Network error. Please try again.";
    } finally {
      creating = false;
    }
  }

  async function handleArchiveProject(project: Project) {
    if (
      !confirm(`Are you sure you want to archive project "${project.name}"?`)
    ) {
      return;
    }

    const csrf = getCsrfToken();
    if (!csrf) {
      errorMessage = "Security token missing. Please refresh and try again.";
      return;
    }

    try {
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/projects/${project.id}/archive`,
        {
          method: "POST",
          headers: {
            "X-CSRF-Token": csrf,
            Accept: "application/json",
          },
        },
      );

      if (res.status === 200) {
        projects = projects.filter((p) => p.id !== project.id);
        successMessage = `Project "${project.name}" archived.`;
      } else {
        errorMessage = "Failed to archive project.";
      }
    } catch {
      errorMessage = "Network error while archiving project.";
    }
  }

  onMount(() => {
    void loadProjects();
  });
</script>

<section class="projects-section" aria-labelledby="projects-heading">
  <div class="header-row">
    <div>
      <h2 id="projects-heading" tabindex="-1">Projects</h2>
      <p class="section-lead">
        Organize work into distinct projects with isolated tasks and workflows.
      </p>
    </div>
    {#if canManageProjects}
      <Button onclick={() => (showCreate = true)}>
        {#snippet children()}New project{/snippet}
      </Button>
    {/if}
  </div>

  {#if successMessage}
    <Feedback tone="success">
      {successMessage}
    </Feedback>
  {/if}

  {#if errorMessage}
    <Feedback tone="danger">
      {errorMessage}
    </Feedback>
    <Button onclick={loadProjects}>
      {#snippet children()}Retry{/snippet}
    </Button>
  {/if}

  {#if showCreate}
    <div class="modal-backdrop">
      <div
        class="modal-card"
        role="dialog"
        aria-modal="true"
        aria-labelledby="create-project-heading"
      >
        <h3 id="create-project-heading">Create new project</h3>
        <p class="note">
          Each project has a short key (like ENG or TREST) used as a prefix for
          all task keys.
        </p>

        {#if createError}
          <Feedback tone="danger">{createError}</Feedback>
        {/if}

        <form onsubmit={handleCreateProject} class="create-form">
          <InputField
            id="proj-name"
            label="Project name"
            bind:value={createName}
            required
            hint="A clear name (e.g. Core Engine, Web Application)"
          />

          <div class="row-2">
            <InputField
              id="proj-slug"
              label="Slug"
              bind:value={createSlug}
              required
              hint="URL identifier (lowercase and hyphens)"
            />
            <InputField
              id="proj-key"
              label="Key"
              bind:value={createKey}
              required
              hint="Task prefix, 2–8 uppercase chars (e.g. CORE)"
            />
          </div>

          <div class="field">
            <label for="proj-desc">Description (optional)</label>
            <textarea
              id="proj-desc"
              bind:value={createDescription}
              rows="3"
              placeholder="What is this project's mission or scope?"></textarea>
          </div>

          <div class="modal-actions">
            <Button
              type="button"
              onclick={() => (showCreate = false)}
              disabled={creating}
            >
              {#snippet children()}Cancel{/snippet}
            </Button>
            <Button type="submit" disabled={creating}>
              {#snippet children()}{creating
                  ? "Creating…"
                  : "Create project"}{/snippet}
            </Button>
          </div>
        </form>
      </div>
    </div>
  {/if}

  {#if loading}
    <Feedback tone="neutral" announcement="status">Loading projects…</Feedback>
  {:else if projects.length === 0}
    <div class="empty-state">
      <h3>No projects yet</h3>
      <p class="note">
        Projects group related tasks, track milestones, and organize your work.
      </p>
      {#if canManageProjects}
        <Button onclick={() => (showCreate = true)}>
          {#snippet children()}Create your first project{/snippet}
        </Button>
      {/if}
    </div>
  {:else}
    <div class="projects-grid" role="list">
      {#each projects as proj (proj.id)}
        <div class="project-card" role="listitem">
          <div class="project-header">
            <span class="project-key" aria-label="Key {proj.key}"
              >{proj.key}</span
            >
            <span class="project-counter"
              >{proj.task_counter}
              {proj.task_counter === 1 ? "task" : "tasks"}</span
            >
          </div>
          <h3 class="project-name">{proj.name}</h3>
          <p class="project-slug">/{proj.slug}</p>
          {#if proj.description}
            <p class="project-desc">{proj.description}</p>
          {/if}
          <div class="project-actions">
            <Button onclick={() => onselectproject?.(proj)}>
              {#snippet children()}Open tasks{/snippet}
            </Button>
            {#if canManageProjects}
              <button
                type="button"
                class="archive-btn"
                onclick={() => handleArchiveProject(proj)}
                title="Archive project"
              >
                Archive
              </button>
            {/if}
          </div>
        </div>
      {/each}
    </div>
  {/if}
</section>

<style>
  .projects-section {
    margin-top: var(--space-6);
  }
  .header-row {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    gap: var(--space-4);
    margin-bottom: var(--space-4);
    flex-wrap: wrap;
  }
  .section-lead {
    color: var(--text-secondary);
    margin-top: var(--space-1);
    margin-bottom: 0;
  }
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.45);
    display: flex;
    align-items: center;
    justify-content: center;
    padding: var(--space-4);
    z-index: 100;
  }
  .modal-card {
    background: var(--surface-raised);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius);
    padding: var(--space-6);
    max-width: 32rem;
    width: 100%;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.2);
  }
  .create-form {
    display: grid;
    gap: var(--space-3);
    margin-top: var(--space-4);
  }
  .row-2 {
    display: grid;
    grid-template-columns: 2fr 1fr;
    gap: var(--space-3);
  }
  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--space-3);
    margin-top: var(--space-4);
  }
  .empty-state {
    padding: var(--space-8);
    text-align: center;
    background: var(--surface-raised);
    border: var(--border-width) dashed var(--border);
    border-radius: var(--radius);
    display: grid;
    gap: var(--space-3);
    justify-items: center;
  }
  .projects-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(18rem, 1fr));
    gap: var(--space-4);
    margin-top: var(--space-4);
  }
  .project-card {
    background: var(--surface-raised);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius);
    padding: var(--space-4);
    display: flex;
    flex-direction: column;
  }
  .project-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: var(--space-2);
  }
  .project-key {
    font-family: var(--font-mono);
    font-weight: 700;
    font-size: var(--text-small);
    color: var(--accent);
    background: var(--selected);
    padding: 2px 6px;
    border-radius: var(--radius);
    letter-spacing: 0.05em;
  }
  .project-counter {
    font-size: var(--text-small);
    color: var(--text-secondary);
  }
  .project-name {
    margin: 0 0 var(--space-1) 0;
    font-size: var(--text-lead);
  }
  .project-slug {
    font-family: var(--font-mono);
    font-size: var(--text-small);
    color: var(--text-secondary);
    margin: 0 0 var(--space-2) 0;
  }
  .project-desc {
    font-size: var(--text-small);
    color: var(--text-secondary);
    margin: 0 0 var(--space-4) 0;
    flex: 1;
  }
  .project-actions {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-top: auto;
    padding-top: var(--space-3);
    border-top: var(--border-width) solid var(--surface-nav);
  }
  .archive-btn {
    background: transparent;
    border: none;
    color: var(--text-muted);
    font-size: var(--text-small);
    cursor: pointer;
    padding: var(--space-1) var(--space-2);
  }
  .archive-btn:hover {
    color: var(--danger);
  }
</style>
