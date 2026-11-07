<script lang="ts">
  import { onMount } from "svelte";
  import Button from "../ui/Button.svelte";
  import InputField from "../ui/InputField.svelte";
  import MarkdownEditor from "../ui/MarkdownEditor.svelte";
  import Feedback from "../ui/Feedback.svelte";
  import Icon from "../ui/Icon.svelte";
  import Modal from "../ui/Modal.svelte";
  import Toast from "../ui/Toast.svelte";
  import Select, { type SelectOption } from "../ui/Select.svelte";
  import DatePicker from "../ui/DatePicker.svelte";
  import { flip } from "svelte/animate";
  import { crossfade } from "svelte/transition";
  import { appear, fadeIn, reducedMotion, sheet } from "../lib/motion";
  import TaskDetailDrawer from "./TaskDetailDrawer.svelte";
  import StatusManagerModal, {
    type TaskStatus,
  } from "./StatusManagerModal.svelte";
  import type { Project } from "./ProjectsView.svelte";
  import type {
    Label,
    TaskAssignee,
    WorkspaceMember,
  } from "./TaskDetailDrawer.svelte";
  import {
    formatDueDate,
    identityHue,
    initials,
    isOverdue,
  } from "../lib/identity";

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
    status_id?: string | null;
    status_name?: string | null;
    status_color?: string | null;
    priority: "none" | "low" | "medium" | "high" | "urgent";
    due_date: string | null;
    created_by: string | null;
    parent_id?: string | null;
    labels?: Label[];
    assignees?: TaskAssignee[];
    created_at: string;
    updated_at: string;
  }

  type Category = Task["status"];

  let {
    workspace,
    project,
    currentUserId,
    openTaskId,
    onback,
  }: {
    workspace: { id: string; name: string; slug: string; role: string };
    project: Project;
    currentUserId?: string;
    /** A task to open in the drawer on arrival (from the dashboard). */
    openTaskId?: string | null;
    onback?: () => void;
  } = $props();

  const LIST_LIMIT = 20;
  const BOARD_LIMIT = 100;
  const VIEW_KEY = "trestle.taskView";

  let view = $state<"list" | "board">("list");
  let tasks = $state<Task[]>([]);
  let total = $state(0);
  let limit = $state(LIST_LIMIT);
  let offset = $state(0);
  let statusFilter = $state<string>("all");
  let statusIdFilter = $state<string>("");
  let labelFilter = $state<string>("");
  let assigneeFilter = $state<string>("");
  let loading = $state(true);
  let errorMessage = $state<string | null>(null);
  let successMessage = $state<string | null>(null);

  // Project-wide totals per status category, for the progress rail.
  let categoryTotals = $state<Record<Category, number> | null>(null);

  // Available metadata for filters
  let projectStatuses = $state<TaskStatus[]>([]);
  let projectLabels = $state<Label[]>([]);
  let workspaceMembers = $state<WorkspaceMember[]>([]);

  // Task detail drawer
  let selectedTask = $state<Task | null>(null);

  // Workflow status manager modal
  let showStatusManager = $state(false);

  // New task modal
  let showCreate = $state(false);
  let createTitle = $state("");
  let createDesc = $state("");
  let createStatus = $state<string>("unstarted");
  let createStatusId = $state<string>("");
  let createPriority = $state<string>("none");
  let createDueDate = $state("");
  let creating = $state(false);
  let createError = $state<string | null>(null);

  // Board drag state (pointer enhancement; the status menu is the primary path)
  let draggingId = $state<string | null>(null);
  let dropTarget = $state<string | null>(null);

  // Board horizontal scroll hints.
  let boardEl = $state<HTMLElement | null>(null);
  let moreRight = $state(false);
  let moreLeft = $state(false);

  function updateBoardEdges() {
    if (!boardEl) return;
    const { scrollLeft, scrollWidth, clientWidth } = boardEl;
    moreLeft = scrollLeft > 4;
    moreRight = scrollLeft + clientWidth < scrollWidth - 4;
  }

  $effect(() => {
    if (!boardEl) return;
    void columns;
    updateBoardEdges();
    if (typeof ResizeObserver === "undefined") return;
    const ro = new ResizeObserver(updateBoardEdges);
    ro.observe(boardEl);
    return () => ro.disconnect();
  });

  // "C" creates a task, as in other trackers, unless the person is typing.
  function handleShortcut(e: KeyboardEvent) {
    if (e.key !== "c" || e.metaKey || e.ctrlKey || e.altKey) return;
    const el = e.target as HTMLElement | null;
    if (
      el &&
      (el.isContentEditable ||
        ["INPUT", "TEXTAREA", "SELECT"].includes(el.tagName))
    )
      return;
    if (document.querySelector('[aria-modal="true"]')) return;
    e.preventDefault();
    openCreate();
  }

  const canDelete = $derived(
    workspace.role === "owner" || workspace.role === "admin",
  );

  const hasActiveFilters = $derived(
    statusFilter !== "all" ||
      statusIdFilter !== "" ||
      labelFilter !== "" ||
      assigneeFilter !== "",
  );

  const statusCategories = [
    { value: "all", label: "All tasks" },
    { value: "backlog", label: "Backlog" },
    { value: "unstarted", label: "Unstarted" },
    { value: "started", label: "In progress" },
    { value: "completed", label: "Completed" },
    { value: "cancelled", label: "Cancelled" },
  ] as const;

  const categoryOrder: Category[] = [
    "backlog",
    "unstarted",
    "started",
    "completed",
    "cancelled",
  ];

  const priorities = [
    { value: "none", label: "None" },
    { value: "low", label: "Low" },
    { value: "medium", label: "Medium" },
    { value: "high", label: "High" },
    { value: "urgent", label: "Urgent" },
  ] as const;

  const statusOptions = $derived<SelectOption[]>(
    projectStatuses.length > 0
      ? [...projectStatuses]
          .sort((a, b) => a.position - b.position)
          .map((s) => ({ value: s.id, label: s.name, color: s.color }))
      : categoryOrder.map((c) => ({
          value: c,
          label: statusCategories.find((s) => s.value === c)?.label ?? c,
          color: `var(--cat-${c})`,
        })),
  );

  const priorityOptions: SelectOption[] = [
    { value: "none", label: "No priority", color: "var(--line-strong)" },
    { value: "low", label: "Low", color: "var(--success)" },
    { value: "medium", label: "Medium", color: "var(--warning)" },
    { value: "high", label: "High", color: "var(--danger)" },
    {
      value: "urgent",
      label: "Urgent",
      icon: "alert-triangle",
      color: "var(--danger)",
    },
  ];

  // Board cards travel between columns instead of blinking out and in.
  const [send, receive] = crossfade({
    duration: (d) =>
      reducedMotion() ? 0 : Math.min(520, 260 + Math.sqrt(d) * 6),
    easing: sheet,
    fallback: (node) => fadeIn(node, { duration: 160 }),
  });

  const railTotal = $derived(
    categoryTotals
      ? categoryOrder.reduce((sum, c) => sum + categoryTotals![c], 0)
      : 0,
  );

  // Cancelled work is not work left to do, so it is outside the share.
  const railCounted = $derived(
    categoryTotals ? railTotal - categoryTotals.cancelled : 0,
  );
  const doneShare = $derived(
    categoryTotals && railCounted > 0
      ? Math.round((categoryTotals.completed / railCounted) * 100)
      : 0,
  );

  // Board columns: the project's own statuses in workflow order, or the
  // canonical categories when a project has none configured.
  const columns = $derived.by(() => {
    if (projectStatuses.length > 0) {
      return [...projectStatuses]
        .sort((a, b) => a.position - b.position)
        .map((s) => ({
          id: s.id,
          name: s.name,
          color: s.color,
          category: s.category,
          tasks: tasks.filter((t) => t.status_id === s.id),
        }));
    }
    return categoryOrder.map((c) => ({
      id: c,
      name: statusCategories.find((s) => s.value === c)?.label ?? c,
      color: `var(--cat-${c})`,
      category: c,
      tasks: tasks.filter((t) => t.status === c),
    }));
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

  export async function loadTasks() {
    loading = true;
    errorMessage = null;

    let url = `/api/v1/workspaces/${workspace.id}/projects/${project.id}/tasks?limit=${limit}&offset=${offset}`;
    if (statusIdFilter) {
      url += `&status_id=${encodeURIComponent(statusIdFilter)}`;
    } else if (statusFilter !== "all") {
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

  /** Bounded count queries (limit=1) read each category's total. */
  export async function loadCategoryTotals() {
    try {
      const base = `/api/v1/workspaces/${workspace.id}/projects/${project.id}/tasks?limit=1&offset=0`;
      const counts = await Promise.all(
        categoryOrder.map(async (c) => {
          const res = await fetch(`${base}&status=${c}`, {
            headers: { Accept: "application/json" },
            cache: "no-store",
          });
          if (!res.ok) throw new Error("count unavailable");
          const data = await res.json();
          return typeof data.total === "number" ? data.total : 0;
        }),
      );
      const next = {} as Record<Category, number>;
      categoryOrder.forEach((c, i) => (next[c] = counts[i]));
      categoryTotals = next;
    } catch {
      // The rail is a summary; the task list remains authoritative.
      categoryTotals = null;
    }
  }

  export async function loadProjectStatuses() {
    try {
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/projects/${project.id}/statuses`,
      );
      if (res.ok) {
        const data = await res.json();
        projectStatuses = data.statuses || [];
        if (projectStatuses.length > 0 && !createStatusId) {
          const defaultStatus =
            projectStatuses.find((s) => s.category === "unstarted") ||
            projectStatuses[0];
          createStatusId = defaultStatus.id;
        }
      }
    } catch {
      // safe fallback
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

  function refreshAll() {
    void loadTasks();
    void loadCategoryTotals();
  }

  function setView(next: "list" | "board") {
    if (view === next) return;
    view = next;
    limit = next === "board" ? BOARD_LIMIT : LIST_LIMIT;
    offset = 0;
    try {
      localStorage.setItem(VIEW_KEY, next);
    } catch {
      // A per-browser convenience only.
    }
    void loadTasks();
  }

  function handleStatusFilter(newStatus: string) {
    statusFilter = newStatus;
    statusIdFilter = "";
    offset = 0;
    void loadTasks();
  }

  function handleResetFilters() {
    statusFilter = "all";
    statusIdFilter = "";
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

  function openCreate(statusId?: string) {
    if (statusId) {
      if (projectStatuses.some((s) => s.id === statusId)) {
        createStatusId = statusId;
      } else {
        createStatus = statusId;
      }
    }
    createError = null;
    showCreate = true;
  }

  function closeCreate() {
    if (creating) return;
    if (
      (createTitle.trim() || createDesc) &&
      !confirm(
        "Close this task draft? Your unsaved text will be kept until you leave this Project.",
      )
    )
      return;
    showCreate = false;
    createError = null;
  }

  async function handleCreateTask(e: Event) {
    e.preventDefault();
    if (creating) return;
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
      let resolvedCategory = createStatus;
      if (createStatusId) {
        const matched = projectStatuses.find((s) => s.id === createStatusId);
        if (matched) resolvedCategory = matched.category;
      }

      const payload: Record<string, unknown> = {
        title: trimmedTitle,
        description: createDesc,
        status: resolvedCategory,
        priority: createPriority,
      };
      if (createStatusId) {
        payload.status_id = createStatusId;
      }
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
        refreshAll();
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

  async function handleStatusChange(task: Task, nextStatusOrId: string) {
    const csrf = getCsrfToken();
    if (!csrf) return;

    try {
      const isStatusId = projectStatuses.some((s) => s.id === nextStatusOrId);
      const payload = isStatusId
        ? { status_id: nextStatusOrId }
        : { status: nextStatusOrId };

      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/projects/${project.id}/tasks/${task.id}`,
        {
          method: "PATCH",
          headers: {
            "Content-Type": "application/json",
            "X-CSRF-Token": csrf,
            Accept: "application/json",
          },
          body: JSON.stringify(payload),
        },
      );

      if (res.status === 200) {
        const updated: Task = await res.json();
        tasks = tasks.map((t) => (t.id === task.id ? updated : t));
        if (selectedTask?.id === task.id) {
          selectedTask = updated;
        }
        void loadCategoryTotals();
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
        void loadCategoryTotals();
      } else {
        errorMessage = "Could not delete task.";
      }
    } catch {
      errorMessage = "Network error while deleting task.";
    }
  }

  function isOpen(task: Task) {
    return task.status !== "completed" && task.status !== "cancelled";
  }

  function handleDrop(columnId: string) {
    const task = tasks.find((t) => t.id === draggingId);
    draggingId = null;
    dropTarget = null;
    if (!task) return;
    const current =
      projectStatuses.length > 0 ? task.status_id : (task.status as string);
    if (current !== columnId) void handleStatusChange(task, columnId);
  }

  onMount(() => {
    try {
      if (localStorage.getItem(VIEW_KEY) === "board") {
        view = "board";
        limit = BOARD_LIMIT;
      }
    } catch {
      // Default to the list.
    }
    void loadTasks();
    void loadLabelsAndMembers();
    void loadProjectStatuses();
    void loadCategoryTotals();
    if (openTaskId) void openTaskById(openTaskId);
  });

  async function openTaskById(id: string) {
    try {
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/projects/${project.id}/tasks/${id}`,
        { headers: { Accept: "application/json" }, cache: "no-store" },
      );
      if (res.ok) selectedTask = await res.json();
    } catch {
      // The list is still available; the drawer simply does not open.
    }
  }
</script>

{#snippet statusMenu(task: Task)}
  <Select
    variant="chip"
    label="Status for {task.human_key}"
    value={projectStatuses.length > 0
      ? task.status_id || task.status
      : task.status}
    options={statusOptions}
    align="end"
    onchange={(v) => handleStatusChange(task, v)}
  />
{/snippet}

{#snippet assigneeStack(task: Task)}
  {#if task.assignees && task.assignees.length > 0}
    <span class="avatar-stack" aria-label="Task assignees">
      {#each task.assignees.slice(0, 3) as a (a.user_id)}
        <span
          class="avatar avatar-sm"
          style:--avatar-hue={identityHue(a.user_id)}
          title="Assigned to {a.display_name}">{initials(a.display_name)}</span
        >
      {/each}
      {#if task.assignees.length > 3}
        <span class="avatar avatar-sm more-avatars"
          >+{task.assignees.length - 3}</span
        >
      {/if}
    </span>
  {/if}
{/snippet}

{#snippet dueDate(task: Task)}
  {#if task.due_date}
    {@const late = isOpen(task) && isOverdue(task.due_date)}
    <span class="due" class:late title="Due {task.due_date}">
      <Icon name="calendar" size={13} />
      <span>{formatDueDate(task.due_date)}</span>
      {#if late}<span class="late-word">Overdue</span>{/if}
    </span>
  {/if}
{/snippet}

<section class="tasks" aria-labelledby="tasks-heading">
  <div class="page-head">
    <div class="project-title">
      <button
        type="button"
        class="btn-quiet btn-icon back-btn"
        onclick={onback}
        aria-label="Back to projects"
        title="Back to projects"
      >
        <Icon name="arrow-left" size={18} />
      </button>
      <span
        class="project-key"
        style:--project-hue={identityHue(project.key)}
        aria-hidden="true">{project.key}</span
      >
      <div class="title-text">
        <h2 id="tasks-heading" tabindex="-1">
          <span class="visually-hidden">Tasks for </span>{project.name}
        </h2>
        <p>
          {#if project.description}{project.description}{:else}/{project.slug}{/if}
        </p>
      </div>
    </div>
    <div class="page-actions">
      <Button
        onclick={() => (showStatusManager = true)}
        label="Workflow statuses"
      >
        {#snippet children()}
          <Icon name="workflow" size={16} />
          <span class="label-long">Workflow statuses</span>
          <span class="label-short" aria-hidden="true">Statuses</span>
        {/snippet}
      </Button>
      <Button variant="primary" onclick={() => openCreate()}>
        {#snippet children()}
          <Icon name="plus" size={16} />
          New task
          <kbd class="btn-kbd" aria-hidden="true">C</kbd>
        {/snippet}
      </Button>
    </div>
  </div>

  {#if categoryTotals && railTotal > 0}
    <div class="rail panel" role="group" aria-label="Filter by status">
      <div class="rail-head">
        <span class="rail-summary">
          <strong class="num">{doneShare}%</strong> complete
          <span class="note num"
            >· {categoryTotals.completed} of {railCounted} tasks done</span
          >
        </span>
      </div>
      <div class="rail-track" aria-hidden="true">
        {#each categoryOrder as c (c)}
          {#if categoryTotals[c] > 0}
            <span
              class="rail-span"
              class:dim={statusFilter !== "all" && statusFilter !== c}
              style:--span-color="var(--cat-{c})"
              style:flex-grow={categoryTotals[c]}
            ></span>
          {/if}
        {/each}
      </div>
      <div class="rail-legend">
        {#each statusCategories as cat (cat.value)}
          <button
            type="button"
            class="legend-btn"
            aria-pressed={statusFilter === cat.value && !statusIdFilter}
            onclick={() => handleStatusFilter(cat.value)}
          >
            {#if cat.value !== "all"}
              <span
                class="dot"
                style:background="var(--cat-{cat.value})"
                aria-hidden="true"
              ></span>
            {/if}
            <span>{cat.label}</span>
            <span class="legend-count num"
              >{cat.value === "all"
                ? railTotal
                : categoryTotals[cat.value as Category]}</span
            >
          </button>
        {/each}
      </div>
    </div>
  {/if}

  <div class="toolbar" role="toolbar" aria-label="Task filters">
    <div class="segmented" role="group" aria-label="Task layout">
      <button
        type="button"
        aria-pressed={view === "list"}
        onclick={() => setView("list")}
      >
        <Icon name="list" size={16} />
        List
      </button>
      <button
        type="button"
        aria-pressed={view === "board"}
        onclick={() => setView("board")}
      >
        <Icon name="board" size={16} />
        Board
      </button>
    </div>

    {#if !categoryTotals || railTotal === 0}
      <div class="filter-pills" role="group" aria-label="Filter by status">
        {#each statusCategories as cat (cat.value)}
          <button
            type="button"
            class="legend-btn"
            aria-pressed={statusFilter === cat.value && !statusIdFilter}
            onclick={() => handleStatusFilter(cat.value)}
          >
            {cat.label}
          </button>
        {/each}
      </div>
    {/if}

    <div class="filter-dropdowns">
      {#if projectStatuses.length > 0}
        <Select
          variant="ghost"
          label="Filter tasks by custom status"
          value={statusIdFilter}
          options={[
            { value: "", label: "All statuses", icon: "layers" },
            ...statusOptions,
          ]}
          onchange={(v) => {
            statusIdFilter = v;
            statusFilter = "all";
            offset = 0;
            void loadTasks();
          }}
        />
      {/if}

      {#if projectLabels.length > 0}
        <Select
          variant="ghost"
          label="Filter tasks by label"
          value={labelFilter}
          options={[
            { value: "", label: "All labels", icon: "tag" },
            ...projectLabels.map((l) => ({
              value: l.id,
              label: l.name,
              color: l.color,
            })),
          ]}
          onchange={(v) => {
            labelFilter = v;
            offset = 0;
            void loadTasks();
          }}
        />
      {/if}

      {#if workspaceMembers.length > 0}
        <Select
          variant="ghost"
          label="Filter tasks by assignee"
          value={assigneeFilter}
          align="end"
          options={[
            { value: "", label: "All assignees", icon: "users" },
            ...workspaceMembers.map((m) => ({
              value: m.user_id,
              label: m.display_name,
              hint: `@${m.username}`,
            })),
          ]}
          onchange={(v) => {
            assigneeFilter = v;
            offset = 0;
            void loadTasks();
          }}
        />
      {/if}

      {#if hasActiveFilters}
        <button type="button" class="btn-link" onclick={handleResetFilters}>
          Reset filters
        </button>
      {/if}
    </div>
  </div>

  <div class="stack messages">
    {#if errorMessage}
      <Feedback tone="danger">{errorMessage}</Feedback>
    {/if}
  </div>

  {#if showCreate}
    <Modal title="New task in {project.key}" onclose={closeCreate}>
      {#if createError}
        <Feedback tone="danger">{createError}</Feedback>
      {/if}

      <form onsubmit={handleCreateTask} class="form-grid">
        <InputField
          id="task-title"
          label="Title"
          bind:value={createTitle}
          required
          placeholder="Describe the actionable work item"
        />

        <MarkdownEditor
          id="task-desc"
          bind:value={createDesc}
          disabled={creating}
        />

        <div class="form-row three">
          <div class="field">
            <label for="task-status">Status</label>
            <Select
              id="task-status"
              value={projectStatuses.length > 0 ? createStatusId : createStatus}
              options={statusOptions}
              onchange={(v) => {
                if (projectStatuses.length > 0) createStatusId = v;
                else createStatus = v;
              }}
            />
          </div>

          <div class="field">
            <label for="task-priority">Priority</label>
            <Select
              id="task-priority"
              value={createPriority}
              options={priorityOptions}
              onchange={(v) => (createPriority = v)}
            />
          </div>

          <div class="field">
            <label for="task-due-date">Due date</label>
            <DatePicker
              id="task-due-date"
              value={createDueDate}
              placeholder="No date"
              onchange={(v) => (createDueDate = v)}
            />
          </div>
        </div>

        <div class="dialog-actions">
          <Button variant="quiet" onclick={closeCreate} disabled={creating}>
            {#snippet children()}Cancel{/snippet}
          </Button>
          <Button type="submit" variant="primary" disabled={creating}>
            {#snippet children()}{creating ? "Saving…" : "Save task"}{/snippet}
          </Button>
        </div>
      </form>
    </Modal>
  {/if}

  {#if loading && tasks.length === 0}
    <div class="skeleton-list" aria-hidden="true">
      {#each [0, 1, 2, 3] as i (i)}
        <div class="skeleton-row"></div>
      {/each}
    </div>
    <Feedback tone="neutral" announcement="status">Loading tasks…</Feedback>
  {:else if tasks.length === 0}
    <div class="empty">
      <span class="empty-mark"><Icon name="list" size={22} /></span>
      <h3>No tasks match your selection</h3>
      <p>
        {hasActiveFilters
          ? "No tasks found matching the active filter criteria. Clear your filters or create a new task."
          : "This project does not have any tasks yet. Create one to begin."}
      </p>
      {#if hasActiveFilters}
        <Button onclick={handleResetFilters}>
          {#snippet children()}Reset filters{/snippet}
        </Button>
      {:else}
        <Button variant="primary" onclick={() => openCreate()}>
          {#snippet children()}Create a task{/snippet}
        </Button>
      {/if}
    </div>
  {:else if view === "list"}
    <div class="task-table panel" class:refreshing={loading}>
      <div class="task-head" aria-hidden="true">
        <span>Task</span>
        <span>Priority</span>
        <span>Due</span>
        <span>People</span>
        <span>Status</span>
      </div>
      <div class="task-list" role="list">
        {#each tasks as task (task.id)}
          <div class="task-row" class:done={!isOpen(task)} role="listitem">
            <button
              type="button"
              class="task-open"
              onclick={() => (selectedTask = task)}
              aria-label="View task {task.human_key}: {task.title}"
            >
              <span class="task-line">
                <span class="key">{task.human_key}</span>
                <span class="task-title">{task.title}</span>
              </span>
              {#if task.description}
                <span class="task-snippet">{task.description}</span>
              {/if}
              {#if task.labels && task.labels.length > 0}
                <span class="row-labels" aria-label="Task labels">
                  {#each task.labels as l (l.id)}
                    <span class="label-badge" style:--badge-color={l.color}>
                      <span class="dot" aria-hidden="true"></span>{l.name}
                    </span>
                  {/each}
                </span>
              {/if}
            </button>

            <span class="cell cell-priority">
              {#if task.priority !== "none"}
                <span class="priority" data-priority={task.priority}
                  >{task.priority}</span
                >
              {:else}
                <span class="none" aria-hidden="true">—</span>
              {/if}
            </span>

            <span class="cell cell-due">
              {#if task.due_date}
                {@render dueDate(task)}
              {:else}
                <span class="none" aria-hidden="true">—</span>
              {/if}
            </span>

            <span class="cell cell-people">
              {@render assigneeStack(task)}
            </span>

            <span class="cell cell-controls">
              {@render statusMenu(task)}
              {#if canDelete}
                <button
                  type="button"
                  class="btn-quiet btn-icon delete-btn"
                  onclick={() => handleDeleteTask(task)}
                  title="Delete task"
                  aria-label="Delete {task.human_key}"
                >
                  <Icon name="trash" size={16} />
                </button>
              {/if}
            </span>
          </div>
        {/each}
      </div>
    </div>

    <div class="pagination-bar">
      <span class="page-indicator num">
        Showing {offset + 1}–{Math.min(offset + limit, total)} of {total}
        {total === 1 ? "task" : "tasks"}
      </span>
      {#if total > limit}
        <div class="pagination-buttons">
          <Button onclick={handlePrevPage} disabled={offset === 0}>
            {#snippet children()}Previous{/snippet}
          </Button>
          <Button onclick={handleNextPage} disabled={offset + limit >= total}>
            {#snippet children()}Next{/snippet}
          </Button>
        </div>
      {/if}
    </div>
  {:else}
    {#if total > tasks.length}
      <p class="board-note">
        Showing the first {tasks.length} of {total} tasks. Use filters to narrow the
        board.
      </p>
    {/if}
    <div
      class="board"
      class:refreshing={loading}
      class:more-right={moreRight}
      class:more-left={moreLeft}
      bind:this={boardEl}
      onscroll={updateBoardEdges}
    >
      {#each columns as col (col.id)}
        <section
          class="board-col"
          class:drop={dropTarget === col.id}
          aria-label="{col.name}, {col.tasks.length} tasks"
          ondragover={(e) => {
            if (!draggingId) return;
            e.preventDefault();
            dropTarget = col.id;
          }}
          ondragleave={(e) => {
            if (e.currentTarget === e.target) dropTarget = null;
          }}
          ondrop={(e) => {
            e.preventDefault();
            handleDrop(col.id);
          }}
        >
          <header class="col-head">
            <span class="dot" style:background={col.color} aria-hidden="true"
            ></span>
            <h3 class="col-name">{col.name}</h3>
            <span class="col-count num">{col.tasks.length}</span>
            <button
              type="button"
              class="btn-quiet btn-icon col-add"
              onclick={() => openCreate(col.id)}
              aria-label="Add task to {col.name}"
              title="Add task to {col.name}"
            >
              <Icon name="plus" size={16} />
            </button>
          </header>
          <div class="col-body">
            {#each col.tasks as task (task.id)}
              <article
                animate:flip={{
                  duration: reducedMotion() ? 0 : 360,
                  easing: sheet,
                }}
                in:receive={{ key: task.id }}
                out:send={{ key: task.id }}
                class="card"
                class:dragging={draggingId === task.id}
                draggable="true"
                ondragstart={(e) => {
                  draggingId = task.id;
                  e.dataTransfer?.setData("text/plain", task.id);
                  if (e.dataTransfer) e.dataTransfer.effectAllowed = "move";
                }}
                ondragend={() => {
                  draggingId = null;
                  dropTarget = null;
                }}
              >
                <button
                  type="button"
                  class="card-open"
                  onclick={() => (selectedTask = task)}
                  aria-label="View task {task.human_key}: {task.title}"
                >
                  <span class="card-top">
                    <span class="key">{task.human_key}</span>
                    {#if task.priority !== "none"}
                      <span class="priority" data-priority={task.priority}
                        >{task.priority}</span
                      >
                    {/if}
                  </span>
                  <span class="card-title">{task.title}</span>
                  {#if task.labels && task.labels.length > 0}
                    <span class="row-labels">
                      {#each task.labels as l (l.id)}
                        <span class="label-badge" style:--badge-color={l.color}>
                          <span class="dot" aria-hidden="true"></span>{l.name}
                        </span>
                      {/each}
                    </span>
                  {/if}
                </button>
                <div class="card-foot">
                  {@render dueDate(task)}
                  {@render assigneeStack(task)}
                  {@render statusMenu(task)}
                </div>
              </article>
            {:else}
              <p class="col-empty">No tasks</p>
            {/each}
          </div>
        </section>
      {/each}
    </div>
  {/if}
</section>

<svelte:window onkeydown={handleShortcut} />

<Toast message={successMessage} onclear={() => (successMessage = null)} />

{#if selectedTask}
  <TaskDetailDrawer
    task={selectedTask}
    {workspace}
    {project}
    {currentUserId}
    statuses={projectStatuses}
    ondelete={canDelete ? handleDeleteTask : undefined}
    siblings={tasks}
    onnavigate={(t) => (selectedTask = t)}
    onclose={() => (selectedTask = null)}
    ontaskupdated={(updated) => {
      tasks = tasks.map((t) => (t.id === updated.id ? updated : t));
      selectedTask = updated;
      void loadCategoryTotals();
    }}
  />
{/if}

{#if showStatusManager}
  <StatusManagerModal
    {workspace}
    {project}
    statuses={projectStatuses}
    onclose={() => (showStatusManager = false)}
    onstatusesupdated={(updated) => {
      projectStatuses = updated;
      refreshAll();
    }}
  />
{/if}

<style>
  .tasks {
    container-type: inline-size;
  }

  /* ── Heading ─────────────────────────────────────────────────── */
  :global(.btn-kbd) {
    min-width: 1.25rem;
    height: 1.25rem;
    margin-left: var(--space-1);
    border-color: oklch(0 0 0 / 0.12);
    background: oklch(0 0 0 / 0.1);
    color: inherit;
    opacity: 0.75;
  }

  .label-short {
    display: none;
  }

  /* Shortcut hints only where there is a keyboard to press them. */
  @media (hover: none), (max-width: 40rem) {
    :global(.btn-kbd) {
      display: none;
    }
  }

  @media (max-width: 40rem) {
    .label-long {
      display: none;
    }
    .label-short {
      display: inline;
    }
  }

  .project-title {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    min-width: 0;
  }

  .back-btn {
    margin-left: calc(var(--space-2) * -1);
  }

  .project-key {
    --project-hue: var(--brand-hue);
    display: inline-grid;
    place-items: center;
    min-width: 3rem;
    height: 3rem;
    padding: 0 var(--space-2);
    border-radius: var(--radius);
    font-size: var(--text-xs);
    font-weight: 700;
    letter-spacing: 0.06em;
    background: light-dark(
      oklch(0.92 0.05 var(--project-hue)),
      oklch(0.33 0.08 var(--project-hue))
    );
    color: light-dark(
      oklch(0.35 0.1 var(--project-hue)),
      oklch(0.92 0.06 var(--project-hue))
    );
    flex-shrink: 0;
  }

  .title-text {
    min-width: 0;
  }

  .title-text p {
    margin-top: 0.125rem;
    font-size: var(--text-small);
    color: var(--text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    max-width: 52ch;
  }

  /* ── Progress rail ───────────────────────────────────────────── */
  .rail {
    position: relative;
    display: grid;
    gap: var(--space-3);
    padding: var(--space-4) var(--space-5);
    margin-bottom: var(--space-4);
    overflow: hidden;
    isolation: isolate;
  }

  .rail::before {
    content: "";
    position: absolute;
    inset: 0;
    z-index: -1;
    background: radial-gradient(
      50% 120% at 100% 0%,
      var(--glow-accent),
      transparent 70%
    );
  }

  .rail-summary {
    font-size: var(--text-small);
    color: var(--text-secondary);
  }

  .rail-summary strong {
    font-size: 1.125rem;
    font-weight: 650;
    letter-spacing: -0.01em;
  }

  .rail-track {
    display: flex;
    gap: 3px;
    height: 0.5rem;
    border-radius: var(--radius-full);
    overflow: hidden;
    background: var(--surface-sunken);
  }

  .rail-span {
    flex-basis: 0;
    min-width: 0.5rem;
    background: var(--span-color);
    box-shadow: 0 0 10px -2px var(--span-color);
    transition:
      opacity var(--transition-normal),
      flex-grow var(--duration-slow) var(--ease-out);
  }

  .rail-span.dim {
    opacity: 0.25;
  }

  .rail-legend,
  .filter-pills {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-1);
  }

  .legend-btn {
    min-height: 2.25rem;
    padding: 0 var(--space-3);
    border-radius: var(--radius-full);
    border-color: transparent;
    background: transparent;
    box-shadow: none;
    color: var(--text-secondary);
    font-size: var(--text-small);
    font-weight: 540;
  }

  .legend-btn:hover:not(:disabled) {
    border-color: transparent;
  }

  .legend-btn[aria-pressed="true"] {
    background: var(--selected);
    color: var(--text);
    border-color: var(--line-strong);
  }

  .legend-count {
    color: var(--text-muted);
    font-weight: 600;
  }

  /* ── Toolbar ─────────────────────────────────────────────────── */
  .toolbar {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    flex-wrap: wrap;
    margin-bottom: var(--space-4);
  }

  .filter-dropdowns {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    flex-wrap: wrap;
    margin-left: auto;
  }

  .messages {
    margin-bottom: var(--space-4);
  }

  .form-row.three {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }

  @media (max-width: 40rem) {
    .form-row.three {
      grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
    }
  }

  .messages:empty {
    display: none;
  }

  /* ── List ────────────────────────────────────────────────────── */
  .task-table {
    padding: var(--space-2);
    transition: opacity var(--transition-normal);
  }

  .task-list {
    display: grid;
    gap: 6px;
  }

  .refreshing {
    opacity: 0.6;
  }

  .task-head,
  .task-row {
    display: grid;
    grid-template-columns: minmax(0, 1fr) 6.5rem 7.5rem 5.5rem 13.5rem;
    align-items: center;
    gap: var(--space-4);
    padding: 0 var(--space-4) 0 var(--space-2);
  }

  .task-head {
    min-height: 2.5rem;
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--text-muted);
  }

  .task-head span:first-child {
    padding-left: var(--space-3);
  }

  /* Rows are inset cards inside the panel. */
  .task-row {
    min-height: 4rem;
    border-radius: var(--radius);
    border: var(--border-width) solid transparent;
    background:
      linear-gradient(var(--veil), var(--veil)) padding-box,
      var(--inset-fill) padding-box,
      var(--inset-edge) border-box;
    transition: --veil 180ms var(--ease-out);
  }

  .task-row:hover {
    --veil: var(--hover-veil);
  }

  .task-open {
    display: grid;
    justify-content: stretch;
    gap: 0.25rem;
    justify-items: start;
    min-width: 0;
    min-height: 3.5rem;
    margin: var(--space-1) 0;
    padding: var(--space-2) var(--space-3);
    border: 0;
    background: transparent;
    box-shadow: none;
    text-align: left;
    white-space: normal;
    font-weight: 400;
  }

  .task-open:hover:not(:disabled) {
    background: transparent;
  }

  .task-line {
    display: flex;
    align-items: baseline;
    gap: var(--space-3);
    min-width: 0;
    max-width: 100%;
  }

  .task-line .key {
    min-width: 4.25rem;
  }

  .task-title {
    font-weight: 580;
    color: var(--text);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .task-row.done .task-title {
    color: var(--text-muted);
    text-decoration: line-through;
    text-decoration-color: var(--line-strong);
  }

  .task-snippet {
    max-width: 100%;
    padding-left: calc(4.25rem + var(--space-3));
    font-size: var(--text-small);
    color: var(--text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .row-labels {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-1);
  }

  .task-open .row-labels {
    padding-left: calc(4.25rem + var(--space-3));
  }

  .cell {
    display: flex;
    align-items: center;
    min-width: 0;
  }

  .none {
    color: var(--text-muted);
  }

  .due {
    display: inline-flex;
    align-items: center;
    gap: 0.35rem;
    font-size: var(--text-small);
    color: var(--text-secondary);
    white-space: nowrap;
    font-variant-numeric: tabular-nums;
  }

  .due.late {
    color: var(--danger);
    font-weight: 560;
  }

  .late-word {
    font-size: var(--text-xs);
  }

  .more-avatars {
    background: var(--selected);
    color: var(--text-secondary);
  }

  .cell-controls {
    gap: var(--space-1);
    justify-content: flex-end;
  }

  .delete-btn {
    color: var(--text-muted);
  }

  @media (hover: hover) and (pointer: fine) {
    .task-row .delete-btn {
      opacity: 0;
      transition: opacity var(--transition-fast);
    }

    .task-row:hover .delete-btn,
    .task-row:focus-within .delete-btn {
      opacity: 1;
    }
  }

  .delete-btn:hover:not(:disabled) {
    color: var(--danger);
    background: var(--danger-soft);
  }

  .pagination-bar {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: var(--space-3);
    flex-wrap: wrap;
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

  .skeleton-list {
    display: grid;
    gap: 2px;
    margin-bottom: var(--space-4);
    border-radius: var(--radius-lg);
    overflow: hidden;
  }

  .skeleton-row {
    height: 4rem;
    background: var(--surface-sunken);
    animation: pulse 1.4s var(--ease-in-out) infinite alternate;
  }

  @keyframes pulse {
    to {
      opacity: 0.5;
    }
  }

  /* ── Board ───────────────────────────────────────────────────── */
  .board-note {
    margin-bottom: var(--space-3);
    font-size: var(--text-small);
    color: var(--text-muted);
  }

  .board {
    display: grid;
    grid-auto-flow: column;
    grid-auto-columns: minmax(16rem, 1fr);
    gap: var(--space-3);
    overflow-x: auto;
    padding-bottom: var(--space-3);
    scroll-snap-type: x proximity;
    overscroll-behavior-x: contain;
    transition: opacity var(--transition-normal);
  }

  .board.more-right {
    -webkit-mask-image: linear-gradient(
      90deg,
      #000 calc(100% - 4rem),
      transparent
    );
    mask-image: linear-gradient(90deg, #000 calc(100% - 4rem), transparent);
  }

  .board.more-left {
    -webkit-mask-image: linear-gradient(90deg, transparent, #000 4rem);
    mask-image: linear-gradient(90deg, transparent, #000 4rem);
  }

  .board.more-left.more-right {
    -webkit-mask-image: linear-gradient(
      90deg,
      transparent,
      #000 4rem,
      #000 calc(100% - 4rem),
      transparent
    );
    mask-image: linear-gradient(
      90deg,
      transparent,
      #000 4rem,
      #000 calc(100% - 4rem),
      transparent
    );
  }

  .board-col {
    display: flex;
    flex-direction: column;
    min-height: 16rem;
    max-height: calc(100svh - 18rem);
    border-radius: var(--radius-lg);
    background:
      var(--card-fill) padding-box,
      var(--card-edge) border-box;
    border: var(--border-width) solid transparent;
    scroll-snap-align: start;
    transition:
      border-color var(--transition-fast),
      background-color var(--transition-fast);
  }

  .board-col.drop {
    background:
      linear-gradient(var(--accent-soft), var(--accent-soft)) padding-box,
      linear-gradient(var(--accent), var(--accent)) border-box;
  }

  .col-head {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-2) var(--space-2) var(--space-2) var(--space-4);
    border-bottom: var(--border-width) solid var(--line);
  }

  .col-name {
    font-size: var(--text-small);
    font-weight: 620;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .col-count {
    min-width: 1.5rem;
    padding: 0 0.4rem;
    border-radius: var(--radius-full);
    background: var(--selected);
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--text-muted);
    text-align: center;
  }

  .col-add {
    margin-left: auto;
    color: var(--text-muted);
  }

  .col-body {
    display: grid;
    align-content: start;
    gap: var(--space-2);
    padding: var(--space-2);
    overflow-y: auto;
    flex: 1;
  }

  .col-empty {
    padding: var(--space-6) var(--space-3);
    text-align: center;
    font-size: var(--text-small);
    color: var(--text-muted);
    border: var(--border-width) dashed var(--line-strong);
    border-radius: var(--radius);
  }

  .card {
    background:
      var(--inset-fill) padding-box,
      var(--inset-edge) border-box;
    border: var(--border-width) solid transparent;
    display: grid;
    border-radius: var(--radius);
    box-shadow: var(--shadow-card);
    cursor: grab;
    transition:
      border-color var(--transition-fast),
      transform var(--transition-fast),
      opacity var(--transition-fast);
  }

  .card:hover {
    border-color: var(--line-strong);
  }

  .card.dragging {
    opacity: 0.45;
    transform: rotate(-1.5deg);
  }

  .card-open {
    display: grid;
    justify-content: stretch;
    gap: var(--space-2);
    justify-items: start;
    min-height: 0;
    padding: var(--space-3) var(--space-3) var(--space-2);
    border: 0;
    border-radius: var(--radius) var(--radius) 0 0;
    background: transparent;
    box-shadow: none;
    text-align: left;
    white-space: normal;
    font-weight: 400;
  }

  .card-open:hover:not(:disabled) {
    background: transparent;
  }

  .card-top {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2);
    width: 100%;
  }

  .card-title {
    font-weight: 580;
    color: var(--text);
    line-height: 1.35;
  }

  .card-foot {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: var(--space-2);
    padding: 0 var(--space-3) var(--space-3);
  }

  .card-foot :global(.select) {
    margin-left: auto;
    max-width: 9.5rem;
  }

  /* ── Responsive list ─────────────────────────────────────────── */
  @container (max-width: 60rem) {
    .task-head {
      display: none;
    }

    .task-row {
      grid-template-areas:
        "open open open open"
        "priority due people controls";
      grid-template-columns: auto auto 1fr auto;
      gap: 0 var(--space-4);
      padding: 0 var(--space-3) var(--space-3) var(--space-1);
    }

    .task-open {
      grid-area: open;
      margin-bottom: 0;
    }

    .cell-priority {
      grid-area: priority;
      padding-left: var(--space-3);
    }
    .cell-due {
      grid-area: due;
    }
    .cell-people {
      grid-area: people;
    }
    .cell-controls {
      grid-area: controls;
    }

    .none {
      display: none;
    }
  }

  @container (max-width: 40rem) {
    .page-head {
      align-items: stretch;
    }

    .page-actions {
      width: 100%;
    }

    .page-actions :global(.btn) {
      flex: 1;
    }

    .task-row {
      grid-template-areas:
        "open open open"
        "priority due people"
        "controls controls controls";
      grid-template-columns: auto auto 1fr;
      row-gap: var(--space-2);
    }

    .cell-controls {
      justify-content: space-between;
      padding-left: var(--space-3);
    }

    .task-snippet,
    .task-open .row-labels {
      padding-left: 0;
    }

    .task-line {
      flex-direction: column;
      gap: 0.125rem;
    }

    .task-title {
      white-space: normal;
    }

    .filter-dropdowns {
      margin-left: 0;
      width: 100%;
    }

    .filter-dropdowns :global(.select) {
      flex: 1 1 9rem;
    }

    .board {
      grid-auto-columns: minmax(16rem, 86%);
    }

    .board-col {
      max-height: none;
    }
  }
</style>
