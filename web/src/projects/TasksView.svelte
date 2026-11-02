<script lang="ts">
  import { onMount } from "svelte";
  import Button from "../ui/Button.svelte";
  import InputField from "../ui/InputField.svelte";
  import Feedback from "../ui/Feedback.svelte";
  import type { Project } from "./ProjectsView.svelte";

  export interface Task {
    id: string;
    workspace_id: string;
    project_id: string;
    project_key: string;
    task_number: number;
    human_key: string;
    title: string;
    description: string;
    status: "backlog" | "unstarted" | "started" | "completed" | "cancelled";
    priority: "none" | "low" | "medium" | "high" | "urgent";
    due_date: string | null;
    created_by: string | null;
    created_at: string;
    updated_at: string;
  }

  let {
    workspace,
    project,
    onback,
  }: {
    workspace: { id: string; name: string; slug: string; role: string };
    project: Project;
    onback?: () => void;
  } = $props();

  let tasks = $state<Task[]>([]);
  let total = $state(0);
  let limit = $state(20);
  let offset = $state(0);
  let statusFilter = $state<string>("all");
  let loading = $state(true);
  let errorMessage = $state<string | null>(null);
  let successMessage = $state<string | null>(null);

  // New task modal
  let showCreate = $state(false);
  let createTitle = $state("");
  let createDesc = $state("");
  let createStatus = $state<string>("unstarted");
  let createPriority = $state<string>("none");
  let createDueDate = $state("");
  let creating = $state(false);
  let createError = $state<string | null>(null);

  const canDelete = $derived(
    workspace.role === "owner" || workspace.role === "admin",
  );

  const statusCategories = [
    { value: "all", label: "All tasks" },
    { value: "backlog", label: "Backlog" },
    { value: "unstarted", label: "Unstarted" },
    { value: "started", label: "In progress" },
    { value: "completed", label: "Completed" },
    { value: "cancelled", label: "Cancelled" },
  ] as const;

  const priorities = [
    { value: "none", label: "None" },
    { value: "low", label: "Low" },
    { value: "medium", label: "Medium" },
    { value: "high", label: "High" },
    { value: "urgent", label: "Urgent" },
  ] as const;

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

  export async function loadTasks() {
    loading = true;
    errorMessage = null;

    let url = `/api/v1/workspaces/${workspace.id}/projects/${project.id}/tasks?limit=${limit}&offset=${offset}`;
    if (statusFilter !== "all") {
      url += `&status=${encodeURIComponent(statusFilter)}`;
    }

    try {
      const res = await fetch(url, {
        headers: { Accept: "application/json" },
        cache: "no-store",
      });

      if (res.status === 200) {
        const data = await res.json();
        tasks = data.tasks || [];
        total = data.total || 0;
      } else if (res.status === 404) {
        errorMessage = "Project or workspace not found.";
      } else {
        errorMessage = "Could not load tasks. Please try again.";
      }
    } catch {
      errorMessage = "Could not load tasks. Check your network connection.";
    } finally {
      loading = false;
    }
  }

  function handleFilterChange(newStatus: string) {
    statusFilter = newStatus;
    offset = 0;
    void loadTasks();
  }

  function handlePrevPage() {
    if (offset > 0) {
      offset = Math.max(0, offset - limit);
      void loadTasks();
    }
  }

  function handleNextPage() {
    if (offset + limit < total) {
      offset += limit;
      void loadTasks();
    }
  }

  async function handleCreateTask(e: Event) {
    e.preventDefault();
    createError = null;

    const trimmedTitle = createTitle.trim();
    if (!trimmedTitle) {
      createError = "Task title is required.";
      return;
    }

    const csrf = getCsrfToken();
    if (!csrf) {
      createError = "Security token missing. Please refresh and try again.";
      return;
    }

    creating = true;
    try {
      const payload: Record<string, unknown> = {
        title: trimmedTitle,
        description: createDesc.trim(),
        status: createStatus,
        priority: createPriority,
      };
      if (createDueDate.trim()) {
        payload.due_date = createDueDate.trim();
      }

      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/projects/${project.id}/tasks`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "X-CSRF-Token": csrf,
            Accept: "application/json",
          },
          body: JSON.stringify(payload),
        },
      );

      if (res.status === 201) {
        const newTask: Task = await res.json();
        successMessage = `Task ${newTask.human_key} created.`;
        showCreate = false;
        createTitle = "";
        createDesc = "";
        createStatus = "unstarted";
        createPriority = "none";
        createDueDate = "";
        offset = 0;
        void loadTasks();
      } else {
        const data = await res.json().catch(() => null);
        createError = data?.error?.message || "Failed to create task.";
      }
    } catch {
      createError = "Network error. Please try again.";
    } finally {
      creating = false;
    }
  }

  async function handleStatusChange(task: Task, nextStatus: string) {
    const csrf = getCsrfToken();
    if (!csrf) return;

    try {
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/projects/${project.id}/tasks/${task.id}`,
        {
          method: "PATCH",
          headers: {
            "Content-Type": "application/json",
            "X-CSRF-Token": csrf,
            Accept: "application/json",
          },
          body: JSON.stringify({ status: nextStatus }),
        },
      );

      if (res.status === 200) {
        const updated: Task = await res.json();
        tasks = tasks.map((t) => (t.id === task.id ? updated : t));
      } else {
        errorMessage = "Could not update task status.";
      }
    } catch {
      errorMessage = "Network error while updating status.";
    }
  }

  async function handleDeleteTask(task: Task) {
    if (
      !confirm(`Permanently delete task ${task.human_key}: "${task.title}"?`)
    ) {
      return;
    }

    const csrf = getCsrfToken();
    if (!csrf) return;

    try {
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/projects/${project.id}/tasks/${task.id}`,
        {
          method: "DELETE",
          headers: {
            "X-CSRF-Token": csrf,
            Accept: "application/json",
          },
        },
      );

      if (res.status === 204) {
        tasks = tasks.filter((t) => t.id !== task.id);
        total = Math.max(0, total - 1);
        successMessage = `Task ${task.human_key} deleted.`;
      } else {
        errorMessage = "Could not delete task.";
      }
    } catch {
      errorMessage = "Network error while deleting task.";
    }
  }

  onMount(() => {
    void loadTasks();
  });
</script>

<section class="tasks-section" aria-labelledby="tasks-heading">
  <div class="breadcrumb-bar">
    <button type="button" class="back-btn" onclick={onback}>
      ← Back to projects
    </button>
  </div>

  <div class="tasks-header">
    <div class="title-group">
      <div class="project-tag">
        <span class="project-key">{project.key}</span>
        <span class="project-title">{project.name}</span>
      </div>
      <h2 id="tasks-heading" class="visually-hidden">
        Tasks for {project.name}
      </h2>
    </div>
    <Button onclick={() => (showCreate = true)}>
      {#snippet children()}New task{/snippet}
    </Button>
  </div>

  {#if successMessage}
    <Feedback tone="success">{successMessage}</Feedback>
  {/if}

  {#if errorMessage}
    <Feedback tone="danger">{errorMessage}</Feedback>
  {/if}

  <!-- Filters row -->
  <div class="filters-bar" role="toolbar" aria-label="Task status filters">
    <span class="filter-label">Filter:</span>
    <div class="filter-pills">
      {#each statusCategories as cat (cat.value)}
        <button
          type="button"
          class="pill-btn"
          class:active={statusFilter === cat.value}
          onclick={() => handleFilterChange(cat.value)}
        >
          {cat.label}
        </button>
      {/each}
    </div>
  </div>

  {#if showCreate}
    <div class="modal-backdrop">
      <div
        class="modal-card"
        role="dialog"
        aria-modal="true"
        aria-labelledby="create-task-heading"
      >
        <h3 id="create-task-heading">New task in {project.key}</h3>

        {#if createError}
          <Feedback tone="danger">{createError}</Feedback>
        {/if}

        <form onsubmit={handleCreateTask} class="task-form">
          <InputField
            id="task-title"
            label="Title"
            bind:value={createTitle}
            required
            hint="Describe the actionable work item"
          />

          <div class="field">
            <label for="task-desc">Description</label>
            <textarea
              id="task-desc"
              bind:value={createDesc}
              rows="3"
              placeholder="Additional details, acceptance criteria, or context"
            ></textarea>
          </div>

          <div class="row-3">
            <div class="field">
              <label for="task-status">Status</label>
              <select id="task-status" bind:value={createStatus}>
                <option value="unstarted">Unstarted</option>
                <option value="backlog">Backlog</option>
                <option value="started">In progress</option>
                <option value="completed">Completed</option>
                <option value="cancelled">Cancelled</option>
              </select>
            </div>

            <div class="field">
              <label for="task-priority">Priority</label>
              <select id="task-priority" bind:value={createPriority}>
                {#each priorities as p (p.value)}
                  <option value={p.value}>{p.label}</option>
                {/each}
              </select>
            </div>

            <InputField
              id="task-due-date"
              label="Due date"
              type="date"
              bind:value={createDueDate}
            />
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
                  ? "Saving…"
                  : "Save task"}{/snippet}
            </Button>
          </div>
        </form>
      </div>
    </div>
  {/if}

  <!-- Task list -->
  {#if loading}
    <Feedback tone="neutral" announcement="status">Loading tasks…</Feedback>
  {:else if tasks.length === 0}
    <div class="empty-state">
      <h3>No tasks match your selection</h3>
      <p class="note">
        {statusFilter === "all"
          ? "This project does not have any tasks yet. Create one to begin."
          : `No tasks found with status "${statusFilter}".`}
      </p>
      <Button onclick={() => (showCreate = true)}>
        {#snippet children()}Create a task{/snippet}
      </Button>
    </div>
  {:else}
    <div class="task-list" role="list">
      {#each tasks as task (task.id)}
        <div class="task-row" role="listitem">
          <div class="task-leading">
            <span class="human-key" aria-label="Task key {task.human_key}">
              {task.human_key}
            </span>
          </div>

          <div class="task-body">
            <span class="task-title">{task.title}</span>
            {#if task.description}
              <p class="task-snippet">{task.description}</p>
            {/if}
            <div class="task-meta">
              {#if task.due_date}
                <span class="due-date" title="Due date">
                  📅 {task.due_date}
                </span>
              {/if}
              {#if task.priority !== "none"}
                <span class="priority-tag" data-priority={task.priority}>
                  {task.priority}
                </span>
              {/if}
            </div>
          </div>

          <div class="task-controls">
            <select
              class="status-select"
              aria-label="Status for {task.human_key}"
              value={task.status}
              onchange={(e) =>
                handleStatusChange(task, (e.target as HTMLSelectElement).value)}
            >
              <option value="backlog">Backlog</option>
              <option value="unstarted">Unstarted</option>
              <option value="started">In progress</option>
              <option value="completed">Completed</option>
              <option value="cancelled">Cancelled</option>
            </select>

            {#if canDelete}
              <button
                type="button"
                class="delete-btn"
                onclick={() => handleDeleteTask(task)}
                title="Delete task"
                aria-label="Delete {task.human_key}"
              >
                Delete
              </button>
            {/if}
          </div>
        </div>
      {/each}
    </div>

    <!-- Pagination footer -->
    <div class="pagination-bar">
      <span class="page-indicator">
        Showing {offset + 1}–{Math.min(offset + limit, total)} of {total}
        {total === 1 ? "task" : "tasks"}
      </span>
      <div class="pagination-buttons">
        <Button onclick={handlePrevPage} disabled={offset === 0}>
          {#snippet children()}Previous{/snippet}
        </Button>
        <Button onclick={handleNextPage} disabled={offset + limit >= total}>
          {#snippet children()}Next{/snippet}
        </Button>
      </div>
    </div>
  {/if}
</section>

<style>
  .visually-hidden {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    white-space: nowrap;
    border: 0;
  }
  .tasks-section {
    margin-top: var(--space-6);
  }
  .breadcrumb-bar {
    margin-bottom: var(--space-3);
  }
  .back-btn {
    background: transparent;
    border: none;
    color: var(--accent);
    font-size: var(--text-small);
    font-weight: 650;
    cursor: pointer;
    padding: 0;
    text-decoration: underline;
  }
  .tasks-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: var(--space-4);
    gap: var(--space-4);
  }
  .project-tag {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }
  .project-key {
    font-family: var(--font-mono);
    font-weight: 700;
    font-size: var(--text-lead);
    background: var(--selected);
    color: var(--accent);
    padding: 2px 8px;
    border-radius: var(--radius);
  }
  .project-title {
    font-size: var(--text-lead);
    font-weight: 650;
  }
  .filters-bar {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    margin-bottom: var(--space-4);
    flex-wrap: wrap;
  }
  .filter-label {
    font-size: var(--text-small);
    color: var(--text-secondary);
    font-weight: 600;
  }
  .filter-pills {
    display: flex;
    gap: var(--space-2);
    flex-wrap: wrap;
  }
  .pill-btn {
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius);
    padding: var(--space-1) var(--space-3);
    font-size: var(--text-small);
    cursor: pointer;
    color: var(--text);
  }
  .pill-btn.active {
    background: var(--button-bg);
    color: var(--button-text);
    border-color: var(--button-bg);
    font-weight: 650;
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
    max-width: 36rem;
    width: 100%;
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.2);
  }
  .task-form {
    display: grid;
    gap: var(--space-3);
    margin-top: var(--space-4);
  }
  .row-3 {
    display: grid;
    grid-template-columns: 1fr 1fr 1fr;
    gap: var(--space-3);
  }
  @media (max-width: 600px) {
    .row-3 {
      grid-template-columns: 1fr;
    }
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
  .task-list {
    display: grid;
    gap: var(--space-2);
  }
  .task-row {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    background: var(--surface-raised);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius);
    padding: var(--space-3) var(--space-4);
  }
  .human-key {
    font-family: var(--font-mono);
    font-weight: 700;
    font-size: var(--text-small);
    color: var(--accent);
    min-width: 5.5rem;
    display: inline-block;
  }
  .task-body {
    flex: 1;
    min-width: 0;
  }
  .task-title {
    font-weight: 600;
    font-size: var(--text-body);
    display: block;
    word-break: break-word;
  }
  .task-snippet {
    margin: var(--space-1) 0 0 0;
    font-size: var(--text-small);
    color: var(--text-secondary);
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }
  .task-meta {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    margin-top: var(--space-1);
    font-size: var(--text-small);
  }
  .priority-tag {
    text-transform: capitalize;
    font-size: 0.75rem;
    font-weight: 600;
    padding: 1px 6px;
    border-radius: var(--radius);
    border: var(--border-width) solid var(--border);
  }
  .priority-tag[data-priority="urgent"] {
    background: var(--danger);
    color: #ffffff;
    border-color: var(--danger);
  }
  .priority-tag[data-priority="high"] {
    background: var(--warning);
    color: #000000;
  }
  .priority-tag[data-priority="medium"] {
    background: var(--selected);
    color: var(--text);
  }
  .priority-tag[data-priority="low"] {
    background: var(--surface);
    color: var(--text-muted);
  }
  .task-controls {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }
  .status-select {
    padding: var(--space-1) var(--space-2);
    font-size: var(--text-small);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
  }
  .delete-btn {
    background: transparent;
    border: none;
    color: var(--text-muted);
    font-size: var(--text-small);
    cursor: pointer;
    padding: var(--space-1);
  }
  .delete-btn:hover {
    color: var(--danger);
  }
  .pagination-bar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-top: var(--space-4);
    padding: var(--space-3) 0;
    flex-wrap: wrap;
    gap: var(--space-3);
  }
  .page-indicator {
    font-size: var(--text-small);
    color: var(--text-secondary);
  }
  .pagination-buttons {
    display: flex;
    gap: var(--space-2);
  }
</style>
