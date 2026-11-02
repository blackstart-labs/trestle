<script lang="ts">
  import { onMount } from "svelte";
  import Button from "../ui/Button.svelte";
  import InputField from "../ui/InputField.svelte";
  import Feedback from "../ui/Feedback.svelte";
  import Icon from "../ui/Icon.svelte";
  import TaskDetailDrawer from "./TaskDetailDrawer.svelte";
  import type { Project } from "./ProjectsView.svelte";
  import type {
    Label,
    TaskAssignee,
    WorkspaceMember,
  } from "./TaskDetailDrawer.svelte";

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
    parent_id?: string | null;
    labels?: Label[];
    assignees?: TaskAssignee[];
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
  let labelFilter = $state<string>("");
  let assigneeFilter = $state<string>("");
  let loading = $state(true);
  let errorMessage = $state<string | null>(null);
  let successMessage = $state<string | null>(null);

  // Available metadata for filters
  let projectLabels = $state<Label[]>([]);
  let workspaceMembers = $state<WorkspaceMember[]>([]);

  // Task detail drawer
  let selectedTask = $state<Task | null>(null);

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

  const hasActiveFilters = $derived(
    statusFilter !== "all" || labelFilter !== "" || assigneeFilter !== "",
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
    if (labelFilter) {
      url += `&label_id=${encodeURIComponent(labelFilter)}`;
    }
    if (assigneeFilter) {
      url += `&assignee_id=${encodeURIComponent(assigneeFilter)}`;
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

  export async function loadLabelsAndMembers() {
    try {
      const [labelsRes, membersRes] = await Promise.all([
        fetch(
          `/api/v1/workspaces/${workspace.id}/projects/${project.id}/labels`,
        ),
        fetch(`/api/v1/workspaces/${workspace.id}/members`),
      ]);
      if (labelsRes.ok) {
        const data = await labelsRes.json();
        projectLabels = data.labels || [];
      }
      if (membersRes.ok) {
        const data = await membersRes.json();
        workspaceMembers = data.members || [];
      }
    } catch {
      // safe fallback
    }
  }

  function handleStatusFilter(newStatus: string) {
    statusFilter = newStatus;
    offset = 0;
    void loadTasks();
  }

  function handleLabelFilter(e: Event) {
    labelFilter = (e.target as HTMLSelectElement).value;
    offset = 0;
    void loadTasks();
  }

  function handleAssigneeFilter(e: Event) {
    assigneeFilter = (e.target as HTMLSelectElement).value;
    offset = 0;
    void loadTasks();
  }

  function handleResetFilters() {
    statusFilter = "all";
    labelFilter = "";
    assigneeFilter = "";
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
        if (selectedTask?.id === task.id) {
          selectedTask = updated;
        }
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
        if (selectedTask?.id === task.id) {
          selectedTask = null;
        }
      } else {
        errorMessage = "Could not delete task.";
      }
    } catch {
      errorMessage = "Network error while deleting task.";
    }
  }

  function getInitials(name: string): string {
    return name
      .split(" ")
      .map((n) => n[0])
      .slice(0, 2)
      .join("")
      .toUpperCase();
  }

  onMount(() => {
    void loadTasks();
    void loadLabelsAndMembers();
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

  <!-- Multi-dimensional Filter Toolbar -->
  <div class="filters-toolbar" role="toolbar" aria-label="Task filters">
    <div class="filter-pills" role="radiogroup" aria-label="Filter by status">
      {#each statusCategories as cat (cat.value)}
        <button
          type="button"
          class="pill-btn"
          class:active={statusFilter === cat.value}
          onclick={() => handleStatusFilter(cat.value)}
        >
          {cat.label}
        </button>
      {/each}
    </div>

    <div class="filter-dropdowns">
      <!-- Label Filter Dropdown -->
      {#if projectLabels.length > 0}
        <select
          class="filter-select"
          value={labelFilter}
          onchange={handleLabelFilter}
          aria-label="Filter tasks by label"
        >
          <option value="">All labels</option>
          {#each projectLabels as l (l.id)}
            <option value={l.id}>{l.name}</option>
          {/each}
        </select>
      {/if}

      <!-- Assignee Filter Dropdown -->
      {#if workspaceMembers.length > 0}
        <select
          class="filter-select"
          value={assigneeFilter}
          onchange={handleAssigneeFilter}
          aria-label="Filter tasks by assignee"
        >
          <option value="">All assignees</option>
          {#each workspaceMembers as m (m.user_id)}
            <option value={m.user_id}>{m.display_name}</option>
          {/each}
        </select>
      {/if}

      {#if hasActiveFilters}
        <button
          type="button"
          class="clear-filters-btn"
          onclick={handleResetFilters}
        >
          Reset filters
        </button>
      {/if}
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
        {hasActiveFilters
          ? "No tasks found matching the active filter criteria. Clear your filters or create a new task."
          : "This project does not have any tasks yet. Create one to begin."}
      </p>
      {#if hasActiveFilters}
        <Button onclick={handleResetFilters}>
          {#snippet children()}Reset filters{/snippet}
        </Button>
      {:else}
        <Button onclick={() => (showCreate = true)}>
          {#snippet children()}Create a task{/snippet}
        </Button>
      {/if}
    </div>
  {:else}
    <div class="task-list" role="list">
      {#each tasks as task (task.id)}
        <div class="task-row" role="listitem">
          <div class="task-leading">
            <button
              type="button"
              class="key-btn"
              onclick={() => (selectedTask = task)}
              title="Open {task.human_key} details"
              aria-label="Open task {task.human_key}"
            >
              {task.human_key}
            </button>
          </div>

          <!-- Clickable Task Body opens detail drawer -->
          <div
            class="task-body"
            onclick={() => (selectedTask = task)}
            role="button"
            tabindex="0"
            onkeydown={(e) => {
              if (e.key === "Enter" || e.key === " ") selectedTask = task;
            }}
            aria-label="View task {task.human_key}: {task.title}"
          >
            <span class="task-title">{task.title}</span>
            {#if task.description}
              <p class="task-snippet">{task.description}</p>
            {/if}
            <div class="task-meta">
              {#if task.due_date}
                <span class="due-date" title="Due date">
                  <Icon name="calendar" size={13} />
                  <span>{task.due_date}</span>
                </span>
              {/if}
              {#if task.priority !== "none"}
                <span class="priority-tag" data-priority={task.priority}>
                  {task.priority}
                </span>
              {/if}

              <!-- Labels -->
              {#if task.labels && task.labels.length > 0}
                <div class="row-labels" aria-label="Task labels">
                  {#each task.labels as l (l.id)}
                    <span
                      class="row-label-badge"
                      style="--badge-color: {l.color};"
                    >
                      {l.name}
                    </span>
                  {/each}
                </div>
              {/if}

              <!-- Assignees -->
              {#if task.assignees && task.assignees.length > 0}
                <div class="row-assignees" aria-label="Task assignees">
                  {#each task.assignees as a (a.user_id)}
                    <span
                      class="row-avatar"
                      title="Assigned to {a.display_name}"
                    >
                      {getInitials(a.display_name)}
                    </span>
                  {/each}
                </div>
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

<!-- Task Detail Drawer -->
{#if selectedTask}
  <TaskDetailDrawer
    task={selectedTask}
    {workspace}
    {project}
    onclose={() => (selectedTask = null)}
    ontaskupdated={(updated) => {
      tasks = tasks.map((t) => (t.id === updated.id ? updated : t));
      selectedTask = updated;
    }}
  />
{/if}

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
  }
  .title-group {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }
  .project-tag {
    display: flex;
    align-items: baseline;
    gap: var(--space-2);
  }
  .project-key {
    font-family: var(--font-mono);
    font-weight: 700;
    color: var(--accent);
    background: var(--surface-raised);
    border: var(--border-width) solid var(--border);
    padding: var(--space-1) var(--space-2);
    border-radius: var(--radius);
    font-size: var(--text-small);
  }
  .project-title {
    font-size: var(--text-lead);
    font-weight: 700;
    color: var(--text);
  }

  /* Filters Toolbar */
  .filters-toolbar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: var(--space-3);
    margin-bottom: var(--space-4);
    flex-wrap: wrap;
    background: var(--surface-raised);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius);
    padding: var(--space-2) var(--space-3);
  }

  .filter-pills {
    display: flex;
    gap: var(--space-1);
    flex-wrap: wrap;
  }

  .pill-btn {
    background: transparent;
    border: var(--border-width) solid transparent;
    border-radius: var(--radius);
    padding: var(--space-1) var(--space-2);
    font-size: var(--text-small);
    color: var(--text-secondary);
    cursor: pointer;
    font-weight: 600;
    transition: all 120ms ease;
  }

  .pill-btn:hover {
    background-color: var(--surface);
    color: var(--text);
  }

  .pill-btn.active {
    background-color: var(--selected);
    color: var(--accent);
    border-color: var(--border);
    font-weight: 700;
  }

  .pill-btn:focus-visible,
  .filter-select:focus-visible,
  .clear-filters-btn:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
  }

  .filter-dropdowns {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .filter-select {
    background: var(--surface);
    color: var(--text);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius);
    padding: var(--space-1) var(--space-2);
    font-size: var(--text-small);
  }

  .clear-filters-btn {
    background: transparent;
    border: none;
    color: var(--accent);
    font-size: var(--text-small);
    font-weight: 650;
    cursor: pointer;
    padding: var(--space-1);
    text-decoration: underline;
  }

  /* Task Rows */
  .task-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    margin-bottom: var(--space-6);
  }

  .task-row {
    background: var(--surface-raised);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius);
    padding: var(--space-3);
    display: grid;
    grid-template-columns: auto 1fr auto;
    align-items: center;
    gap: var(--space-3);
    transition:
      border-color 120ms ease,
      box-shadow 120ms ease;
  }

  .task-row:hover {
    border-color: var(--accent);
    box-shadow: 0 2px 6px rgba(0, 0, 0, 0.05);
  }

  .task-leading {
    display: flex;
    align-items: center;
  }

  .key-btn {
    font-family: var(--font-mono);
    font-size: var(--text-small);
    font-weight: 700;
    color: var(--accent);
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    padding: var(--space-1) var(--space-2);
    border-radius: var(--radius);
    cursor: pointer;
  }

  .key-btn:hover {
    text-decoration: underline;
  }

  .key-btn:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
  }

  .task-body {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    cursor: pointer;
    text-align: left;
    outline: none;
  }

  .task-body:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
    border-radius: var(--radius);
  }

  .task-title {
    font-size: var(--text-body);
    font-weight: 650;
    color: var(--text);
  }

  .task-snippet {
    font-size: var(--text-small);
    color: var(--text-muted);
    margin: 0;
    display: -webkit-box;
    -webkit-line-clamp: 2;
    line-clamp: 2;
    -webkit-box-orient: vertical;
    overflow: hidden;
  }

  .task-meta {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    margin-top: var(--space-1);
    flex-wrap: wrap;
  }

  .due-date {
    font-size: var(--text-small);
    color: var(--text-secondary);
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
  }

  .priority-tag {
    font-size: 0.75rem;
    font-weight: 700;
    padding: 0.125rem 0.375rem;
    border-radius: var(--radius);
    text-transform: capitalize;
  }

  .priority-tag[data-priority="urgent"],
  .priority-tag[data-priority="high"] {
    background-color: var(--danger);
    color: var(--surface-raised);
  }

  .priority-tag[data-priority="medium"] {
    background-color: var(--warning);
    color: var(--text);
  }

  .priority-tag[data-priority="low"],
  .priority-tag[data-priority="none"] {
    background-color: var(--surface);
    color: var(--text-secondary);
    border: var(--border-width) solid var(--border);
  }

  .row-labels {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
  }

  .row-label-badge {
    font-size: 0.6875rem;
    font-weight: 650;
    color: var(--text);
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    border-left: 3px solid var(--badge-color);
    padding: 0.0625rem 0.375rem;
    border-radius: var(--badge-radius);
  }

  .row-assignees {
    display: inline-flex;
    align-items: center;
    gap: 0.125rem;
  }

  .row-avatar {
    width: 1.35rem;
    height: 1.35rem;
    border-radius: 9999px;
    background: var(--selected);
    color: var(--accent);
    display: inline-flex;
    align-items: center;
    justify-content: center;
    font-size: 0.5625rem;
    font-weight: 700;
    border: var(--border-width) solid var(--border);
  }

  .task-controls {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .status-select {
    background: var(--surface);
    color: var(--text);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius);
    padding: var(--space-1) var(--space-2);
    font-size: var(--text-small);
    font-weight: 600;
  }

  .status-select:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
  }

  .delete-btn {
    background: transparent;
    border: var(--border-width) solid transparent;
    color: var(--danger);
    font-size: var(--text-small);
    padding: var(--space-1) var(--space-2);
    border-radius: var(--radius);
    cursor: pointer;
    font-weight: 600;
  }

  .delete-btn:hover {
    background: var(--surface);
    border-color: var(--danger);
  }

  .delete-btn:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
  }

  /* Empty state */
  .empty-state {
    text-align: center;
    padding: var(--space-12) var(--space-4);
    background: var(--surface-raised);
    border: var(--border-width) dashed var(--border);
    border-radius: var(--radius);
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-2);
  }

  .empty-state h3 {
    margin: 0;
    font-size: var(--text-lead);
    color: var(--text);
  }

  .empty-state .note {
    color: var(--text-muted);
    font-size: var(--text-small);
    margin-bottom: var(--space-3);
    max-width: 28rem;
  }

  /* Pagination */
  .pagination-bar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-top: var(--space-4);
  }

  .page-indicator {
    font-size: var(--text-small);
    color: var(--text-muted);
  }

  .pagination-buttons {
    display: flex;
    gap: var(--space-2);
  }

  /* Modal */
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: var(--drawer-scrim);
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
    box-shadow: 0 4px 16px rgba(0, 0, 0, 0.2);
  }

  .modal-card h3 {
    margin: 0 0 var(--space-4);
    font-size: var(--text-lead);
    color: var(--text);
  }

  .task-form {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .field label {
    font-size: var(--text-small);
    font-weight: 650;
    color: var(--text);
  }

  .field textarea,
  .field select {
    background: var(--surface);
    color: var(--text);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius);
    padding: var(--space-2) var(--space-3);
    font-size: var(--text-body);
    font-family: inherit;
  }

  .field textarea:focus-visible,
  .field select:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
  }

  .row-3 {
    display: grid;
    grid-template-columns: 1fr 1fr 1fr;
    gap: var(--space-2);
  }

  @media (max-width: 600px) {
    .row-3 {
      grid-template-columns: 1fr;
    }
  }

  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--space-2);
    margin-top: var(--space-2);
  }
</style>
