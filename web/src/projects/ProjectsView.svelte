<script lang="ts">
  import { onMount } from "svelte";
  import Button from "../ui/Button.svelte";
  import InputField from "../ui/InputField.svelte";
  import Feedback from "../ui/Feedback.svelte";
  import Icon from "../ui/Icon.svelte";
  import Modal from "../ui/Modal.svelte";
  import Toast from "../ui/Toast.svelte";
  import EditProjectModal from "./EditProjectModal.svelte";
  import { identityHue, relativeTime } from "../lib/identity";

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
  let query = $state("");
  let editingProject = $state<Project | null>(null);

  // Live progress per project, read with bounded limit=1 count queries.
  let stats = $state<
    Record<string, { total: number; done: number; cancelled: number }>
  >({});

  async function count(projectId: string, status?: string) {
    const res = await fetch(
      `/api/v1/workspaces/${workspace.id}/projects/${projectId}/tasks?limit=1&offset=0${status ? `&status=${status}` : ""}`,
      { headers: { Accept: "application/json" }, cache: "no-store" },
    );
    if (!res.ok) throw new Error("count unavailable");
    const data = await res.json();
    if (typeof data.total !== "number") throw new Error("count unavailable");
    return data.total as number;
  }

  async function loadStats(list: Project[]) {
    const entries = await Promise.all(
      list.slice(0, 24).map(async (p) => {
        try {
          const [total, done, cancelled] = await Promise.all([
            count(p.id),
            count(p.id, "completed"),
            count(p.id, "cancelled"),
          ]);
          return [p.id, { total, done, cancelled }] as const;
        } catch {
          return null;
        }
      }),
    );
    stats = Object.fromEntries(
      entries.filter((e): e is NonNullable<typeof e> => e !== null),
    );
  }

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

  const visibleProjects = $derived.by(() => {
    const q = query.trim().toLowerCase();
    if (!q) return projects;
    return projects.filter((p) =>
      `${p.name} ${p.key} ${p.slug} ${p.description}`.toLowerCase().includes(q),
    );
  });

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

  function closeCreate() {
    if (creating) return;
    showCreate = false;
    createError = null;
  }

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
        void loadStats(projects);
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
      !confirm(
        `Archive "${project.name}"? It will be removed from the projects list. Archived projects are kept, not deleted.`,
      )
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

<section class="projects" aria-labelledby="projects-heading">
  <div class="page-head">
    <div>
      <h2 id="projects-heading" tabindex="-1">Projects</h2>
      <p>
        Each project has its own key, tasks and workflow in {workspace.name}.
      </p>
    </div>
    {#if canManageProjects}
      <div class="page-actions">
        <Button variant="primary" onclick={() => (showCreate = true)}>
          {#snippet children()}
            <Icon name="plus" size={16} />
            New project
          {/snippet}
        </Button>
      </div>
    {/if}
  </div>

  <div class="stack messages">
    {#if errorMessage}
      <Feedback tone="danger">{errorMessage}</Feedback>
      <div>
        <Button onclick={loadProjects}>
          {#snippet children()}Retry{/snippet}
        </Button>
      </div>
    {/if}
  </div>

  {#if showCreate}
    <Modal
      title="Create new project"
      description="The key prefixes every task in the project, like CORE-12."
      onclose={closeCreate}
    >
      {#if createError}
        <Feedback tone="danger">{createError}</Feedback>
      {/if}

      <form onsubmit={handleCreateProject} class="form-grid">
        <InputField
          id="proj-name"
          label="Project name"
          bind:value={createName}
          required
          placeholder="Core Engine"
        />

        <div class="form-row">
          <div class="field">
            <label for="proj-slug">Slug</label>
            <input
              id="proj-slug"
              bind:value={createSlug}
              oninput={() => (slugManual = true)}
              required
              aria-describedby="proj-slug-hint"
            />
            <p class="note" id="proj-slug-hint">
              Lowercase letters and hyphens
            </p>
          </div>
          <div class="field">
            <label for="proj-key">Key</label>
            <input
              id="proj-key"
              class="key-input"
              bind:value={createKey}
              oninput={() => (keyManual = true)}
              required
              maxlength="8"
              aria-describedby="proj-key-hint"
            />
            <p class="note" id="proj-key-hint">2–8 letters or numbers</p>
          </div>
        </div>

        {#if createKey.trim()}
          <p class="key-preview" aria-live="polite">
            First task will be
            <span class="key-sample">{createKey.trim().toUpperCase()}-1</span>
          </p>
        {/if}

        <div class="field">
          <label for="proj-desc">Description (optional)</label>
          <textarea
            id="proj-desc"
            bind:value={createDescription}
            rows="3"
            placeholder="What is this project's mission or scope?"></textarea>
        </div>

        <div class="dialog-actions">
          <Button
            type="button"
            variant="quiet"
            onclick={closeCreate}
            disabled={creating}
          >
            {#snippet children()}Cancel{/snippet}
          </Button>
          <Button type="submit" variant="primary" disabled={creating}>
            {#snippet children()}{creating
                ? "Creating…"
                : "Create project"}{/snippet}
          </Button>
        </div>
      </form>
    </Modal>
  {/if}

  {#if loading}
    <div class="projects-grid" aria-hidden="true">
      {#each [0, 1, 2] as i (i)}
        <div class="project-card skeleton"></div>
      {/each}
    </div>
    <Feedback tone="neutral" announcement="status">Loading projects…</Feedback>
  {:else if projects.length === 0}
    <div class="empty">
      <span class="empty-mark"><Icon name="folder" size={22} /></span>
      <h3>No projects yet</h3>
      <p>
        Projects group related tasks under one key and one workflow, so everyone
        reads status the same way.
      </p>
      {#if canManageProjects}
        <Button variant="primary" onclick={() => (showCreate = true)}>
          {#snippet children()}Create your first project{/snippet}
        </Button>
      {:else}
        <p>Ask a workspace owner or admin to create one.</p>
      {/if}
    </div>
  {:else}
    {#if projects.length > 3}
      <div class="toolbar">
        <label class="search-field">
          <Icon name="search" size={16} />
          <span class="visually-hidden">Filter projects</span>
          <input
            type="search"
            bind:value={query}
            placeholder="Filter by name or key"
          />
        </label>
        <span class="note num">
          {visibleProjects.length} of {projects.length} projects
        </span>
      </div>
    {/if}

    {#if visibleProjects.length === 0}
      <div class="empty">
        <h3>No projects match “{query.trim()}”</h3>
        <Button variant="quiet" onclick={() => (query = "")}>
          {#snippet children()}Clear filter{/snippet}
        </Button>
      </div>
    {:else}
      <div class="projects-grid" role="list">
        {#each visibleProjects as proj (proj.id)}
          <article
            class="project-card"
            role="listitem"
            style:--project-hue={identityHue(proj.key)}
          >
            <div class="card-top">
              <span class="project-key" aria-label="Key {proj.key}"
                >{proj.key}</span
              >
              {#if !stats[proj.id]}
                <span class="project-counter num"
                  >{proj.task_counter}
                  {proj.task_counter === 1 ? "task" : "tasks"} created</span
                >
              {/if}
            </div>
            <h3 class="project-name">{proj.name}</h3>
            <p class="project-slug">/{proj.slug}</p>
            {#if proj.description}
              <p class="project-desc">{proj.description}</p>
            {:else}
              <p class="project-desc empty-desc">No description yet.</p>
            {/if}
            {#if stats[proj.id]?.total === 0}
              <p class="progress-empty">No tasks yet.</p>
            {:else if stats[proj.id]}
              {@const st = stats[proj.id]}
              {@const counted = st.total - st.cancelled}
              {@const pct =
                counted > 0 ? Math.round((st.done / counted) * 100) : 0}
              <div class="progress">
                <div class="progress-line">
                  <span class="num"
                    ><strong>{counted - st.done}</strong> open</span
                  >
                  <span class="num">{st.done} of {counted} done</span>
                </div>
                <span
                  class="progress-bar"
                  role="progressbar"
                  aria-label="{proj.name} completion"
                  aria-valuenow={pct}
                  aria-valuemin={0}
                  aria-valuemax={100}
                >
                  <span style:width="{pct}%"></span>
                </span>
              </div>
            {/if}
            <div class="card-foot">
              <span class="updated">
                <Icon name="clock" size={14} />
                Updated {relativeTime(proj.updated_at)}
              </span>
              <div class="card-actions">
                {#if canManageProjects}
                  <Button
                    variant="quiet"
                    label="Edit project {proj.name}"
                    onclick={() => (editingProject = proj)}
                  >
                    {#snippet children()}<Icon
                        name="pencil"
                        size={17}
                      />Edit{/snippet}
                  </Button>
                  <button
                    type="button"
                    class="btn-quiet btn-icon"
                    onclick={() => handleArchiveProject(proj)}
                    aria-label="Archive {proj.name}"
                    title="Archive project"
                  >
                    <Icon name="archive" size={17} />
                  </button>
                {/if}
                <Button onclick={() => onselectproject?.(proj)}>
                  {#snippet children()}
                    Open tasks
                    <Icon name="arrow-right" size={15} />
                  {/snippet}
                </Button>
              </div>
            </div>
          </article>
        {/each}
      </div>
    {/if}
  {/if}
</section>

<Toast message={successMessage} onclear={() => (successMessage = null)} />

{#if editingProject}
  <EditProjectModal
    project={editingProject}
    {workspace}
    onclose={() => (editingProject = null)}
    onupdated={(updated) => {
      projects = projects
        .map((project) => (project.id === updated.id ? updated : project))
        .sort((a, b) => a.name.localeCompare(b.name));
      editingProject = null;
      successMessage = "Project updated.";
    }}
  />
{/if}

<style>
  .messages:empty {
    display: none;
  }

  .messages {
    margin-bottom: var(--space-5);
  }

  .toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-4);
    margin-bottom: var(--space-4);
    flex-wrap: wrap;
  }

  .search-field {
    position: relative;
    display: flex;
    align-items: center;
    width: min(100%, 22rem);
    color: var(--text-muted);
  }

  .search-field :global(.svg-icon) {
    position: absolute;
    left: var(--space-3);
    pointer-events: none;
  }

  .search-field input {
    padding-left: 2.25rem;
  }

  .projects-grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(min(100%, 20rem), 1fr));
    gap: var(--space-4);
  }

  .project-card {
    background:
      var(--card-fill) padding-box,
      var(--card-edge) border-box;
    border: var(--border-width) solid transparent;
    --project-hue: var(--brand-hue);
    position: relative;
    display: flex;
    flex-direction: column;
    min-height: 13rem;
    padding: var(--space-5);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
    overflow: hidden;
    transition:
      border-color var(--transition-fast),
      box-shadow var(--duration) var(--ease-out),
      transform var(--duration-slow) var(--ease-spring);
  }

  /* Each project carries a faint light in its own hue. */
  .project-card::before {
    content: "";
    position: absolute;
    inset: 0;
    pointer-events: none;
    background: radial-gradient(
      90% 60% at 100% 0%,
      light-dark(
        oklch(0.93 0.04 var(--project-hue) / 0.55),
        oklch(0.32 0.07 var(--project-hue) / 0.28)
      ),
      transparent 70%
    );
  }

  .project-card > * {
    position: relative;
  }

  .project-card:hover {
    transform: translateY(-3px);
    border-color: var(--line-strong);
    box-shadow:
      var(--shadow-card),
      0 18px 40px -28px
        light-dark(oklch(0.4 0.1 var(--project-hue) / 0.45), oklch(0 0 0 / 0.8));
  }

  .project-card.skeleton {
    background: var(--surface-sunken);
    box-shadow: none;
    animation: pulse 1.4s var(--ease-in-out) infinite alternate;
  }

  .project-card.skeleton::before {
    display: none;
  }

  @keyframes pulse {
    to {
      opacity: 0.5;
    }
  }

  .card-top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    margin-bottom: var(--space-4);
  }

  .project-key {
    display: inline-flex;
    align-items: center;
    min-height: 1.75rem;
    padding: 0 0.625rem;
    border-radius: var(--radius-xs);
    font-size: var(--text-xs);
    font-weight: 680;
    letter-spacing: 0.06em;
    font-variant-numeric: tabular-nums;
    background: light-dark(
      oklch(0.92 0.05 var(--project-hue)),
      oklch(0.33 0.08 var(--project-hue))
    );
    color: light-dark(
      oklch(0.35 0.1 var(--project-hue)),
      oklch(0.92 0.06 var(--project-hue))
    );
  }

  .project-counter {
    font-size: var(--text-xs);
    font-weight: 560;
    color: var(--text-muted);
  }

  .project-name {
    font-size: 1.1875rem;
    font-weight: 620;
    letter-spacing: -0.018em;
  }

  .project-slug {
    margin-top: 0.125rem;
    font-size: var(--text-small);
    color: var(--text-muted);
  }

  .project-desc {
    margin-top: var(--space-3);
    font-size: var(--text-small);
    line-height: 1.5;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  .empty-desc {
    color: var(--text-muted);
    font-style: italic;
  }

  .progress-empty {
    margin-top: var(--space-4);
    font-size: var(--text-xs);
    color: var(--text-muted);
  }

  .progress {
    display: grid;
    gap: var(--space-2);
    margin-top: var(--space-4);
  }

  .progress-line {
    display: flex;
    justify-content: space-between;
    font-size: var(--text-xs);
    color: var(--text-muted);
  }

  .progress-line strong {
    color: var(--text);
    font-weight: 600;
  }

  .progress-bar {
    display: block;
    height: 0.375rem;
    border-radius: var(--radius-full);
    background: var(--surface-sunken);
    overflow: hidden;
  }

  .progress-bar span {
    display: block;
    height: 100%;
    border-radius: inherit;
    background: linear-gradient(
      90deg,
      light-dark(
        oklch(0.75 0.08 var(--project-hue)),
        oklch(0.5 0.1 var(--project-hue))
      ),
      light-dark(
        oklch(0.55 0.13 var(--project-hue)),
        oklch(0.78 0.12 var(--project-hue))
      )
    );
    transition: width var(--duration-slow) var(--ease-out);
  }

  .card-foot {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    margin-top: auto;
    padding-top: var(--space-4);
    flex-wrap: wrap;
  }

  .updated {
    display: inline-flex;
    align-items: center;
    gap: 0.375rem;
    font-size: var(--text-xs);
    color: var(--text-muted);
  }

  .card-actions {
    display: flex;
    align-items: center;
    gap: var(--space-1);
    margin-left: auto;
    flex-wrap: wrap;
  }

  .card-actions :global(.btn-icon:hover) {
    color: var(--danger);
  }

  .key-input {
    text-transform: uppercase;
    letter-spacing: 0.05em;
    font-weight: 600;
  }

  .key-preview {
    margin-top: calc(var(--space-2) * -1);
    font-size: var(--text-small);
    color: var(--text-muted);
  }

  .key-sample {
    margin-left: 0.25rem;
    padding: 0.125rem 0.5rem;
    border-radius: var(--radius-xs);
    background: var(--accent-soft);
    color: var(--accent);
    font-weight: 650;
    font-variant-numeric: tabular-nums;
  }
</style>
