<script lang="ts">
  import { onMount, untrack } from "svelte";
  import Button from "../ui/Button.svelte";
  import InputField from "../ui/InputField.svelte";
  import Feedback from "../ui/Feedback.svelte";
  import Icon from "../ui/Icon.svelte";
  import type { Task } from "./TasksView.svelte";
  import type { Project } from "./ProjectsView.svelte";
  import type { TaskStatus } from "./StatusManagerModal.svelte";
  import { identityHue } from "../lib/identity";
  import Toast from "../ui/Toast.svelte";
  import Select, { type SelectOption } from "../ui/Select.svelte";
  import DatePicker from "../ui/DatePicker.svelte";
  import MultiPicker from "../ui/MultiPicker.svelte";
  import { appear, drawerIn, drawerOut, fadeIn } from "../lib/motion";
  import { relativeTime } from "../lib/identity";

  export interface Label {
    id: string;
    workspace_id: string;
    project_id: string;
    name: string;
    color: string;
    created_at: string;
  }

  export interface TaskAssignee {
    task_id: string;
    user_id: string;
    username: string;
    display_name: string;
    avatar_url?: string | null;
    assigned_by?: string | null;
    created_at: string;
  }

  export interface TaskComment {
    id: string;
    workspace_id: string;
    task_id: string;
    user_id: string;
    author_name: string;
    author_username: string;
    content: string;
    created_at: string;
    updated_at: string;
  }

  export interface TaskActivity {
    id: string;
    workspace_id: string;
    task_id: string;
    actor_id: string | null;
    actor_name: string | null;
    action: string;
    details: Record<string, unknown>;
    created_at: string;
  }

  export interface WorkspaceMember {
    user_id: string;
    username: string;
    display_name: string;
    role: string;
  }

  let {
    task,
    workspace,
    project,
    statuses = [],
    currentUserId,
    onclose,
    ontaskupdated,
    ondelete,
    siblings = [],
    onnavigate,
  }: {
    task: Task;
    workspace: { id: string; name: string; slug: string; role: string };
    project: Project;
    statuses?: TaskStatus[];
    currentUserId?: string;
    onclose: () => void;
    ontaskupdated?: (task: Task) => void;
    /** Present when the viewer may delete tasks. */
    ondelete?: (task: Task) => void;
    /** The list the task was opened from, for previous/next. */
    siblings?: Task[];
    onnavigate?: (task: Task) => void;
  } = $props();

  // Local task state
  // svelte-ignore state_referenced_locally
  let currentTask = $state<Task>(task);
  // svelte-ignore state_referenced_locally
  let editTitle = $state(task.title);
  // svelte-ignore state_referenced_locally
  let editDescription = $state(task.description || "");
  // svelte-ignore state_referenced_locally
  let editStatus = $state(task.status);
  // svelte-ignore state_referenced_locally
  let editStatusId = $state(task.status_id || "");
  // svelte-ignore state_referenced_locally
  let editPriority = $state(task.priority);
  // svelte-ignore state_referenced_locally
  let editDueDate = $state(task.due_date || "");
  let savingDetails = $state(false);

  $effect(() => {
    if (task) {
      currentTask = task;
      editTitle = task.title;
      editDescription = task.description || "";
      editStatus = task.status;
      editStatusId = task.status_id || "";
      editPriority = task.priority;
      editDueDate = task.due_date || "";
    }
  });

  // Subtasks
  let subtasks = $state<Task[]>([]);
  let loadingSubtasks = $state(false);
  let newSubtaskTitle = $state("");
  let creatingSubtask = $state(false);

  // Comments
  let comments = $state<TaskComment[]>([]);
  let loadingComments = $state(false);
  let newCommentText = $state("");
  let postingComment = $state(false);

  // Activities
  let activities = $state<TaskActivity[]>([]);
  let loadingActivities = $state(false);

  // Labels & Assignees
  let projectLabels = $state<Label[]>([]);
  let workspaceMembers = $state<WorkspaceMember[]>([]);
  let selectedLabelId = $state("");
  let selectedMemberId = $state("");

  // New Label Modal/Form
  let showCreateLabel = $state(false);
  let newLabelName = $state("");
  let newLabelColor = $state("#315b3b");
  let creatingLabel = $state(false);

  // Feedback notifications
  let feedbackMessage = $state<string | null>(null);
  let feedbackTone = $state<"success" | "danger" | "neutral">("neutral");

  let drawerElement = $state<HTMLElement | null>(null);

  const canManage = $derived(
    workspace.role === "owner" ||
      workspace.role === "admin" ||
      workspace.role === "member",
  );

  const canDeleteComment = (comment: TaskComment) =>
    workspace.role === "owner" ||
    workspace.role === "admin" ||
    comment.user_id === currentUserId;

  const priorities = [
    { value: "none", label: "None" },
    { value: "low", label: "Low" },
    { value: "medium", label: "Medium" },
    { value: "high", label: "High" },
    { value: "urgent", label: "Urgent" },
  ] as const;

  const statusCategories = [
    { value: "backlog", label: "Backlog" },
    { value: "unstarted", label: "Unstarted" },
    { value: "started", label: "In progress" },
    { value: "completed", label: "Completed" },
    { value: "cancelled", label: "Cancelled" },
  ] as const;

  const presetColors = [
    "#315b3b",
    "#26633b",
    "#1e4d79",
    "#6b4c9a",
    "#9f302b",
    "#775300",
    "#4c544c",
  ];

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

  // Successes are transient toasts; problems stay inline until fixed.
  let toastMessage = $state<string | null>(null);

  function showNotice(msg: string, tone: "success" | "danger" | "neutral") {
    if (tone === "success") {
      toastMessage = msg;
      feedbackMessage = null;
    } else {
      feedbackMessage = msg;
      feedbackTone = tone;
    }
  }

  // Title and description save explicitly; everything else saves on change.
  const dirty = $derived(
    editTitle.trim() !== currentTask.title ||
      editDescription.trim() !== (currentTask.description || "").trim(),
  );

  function requestClose() {
    if (
      dirty &&
      !confirm("Discard your unsaved changes to the title and description?")
    ) {
      return;
    }
    onclose();
  }

  async function copyKey() {
    try {
      await navigator.clipboard.writeText(currentTask.human_key);
      showNotice(`Copied ${currentTask.human_key}.`, "success");
    } catch {
      showNotice("Could not copy to the clipboard.", "danger");
    }
  }

  export async function loadSubtasks() {
    loadingSubtasks = true;
    try {
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/projects/${project.id}/tasks/${currentTask.id}/subtasks`,
      );
      if (res.ok) {
        subtasks = await res.json();
      }
    } catch {
      // ignore network errors silently in background
    } finally {
      loadingSubtasks = false;
    }
  }

  export async function loadComments() {
    loadingComments = true;
    try {
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/projects/${project.id}/tasks/${currentTask.id}/comments`,
      );
      if (res.ok) {
        const data = await res.json();
        comments = data.comments || [];
      }
    } catch {
      // ignore
    } finally {
      loadingComments = false;
    }
  }

  export async function loadActivities() {
    loadingActivities = true;
    try {
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/projects/${project.id}/tasks/${currentTask.id}/activity`,
      );
      if (res.ok) {
        const data = await res.json();
        activities = data.activities || [];
      }
    } catch {
      // ignore
    } finally {
      loadingActivities = false;
    }
  }

  export async function loadProjectLabels() {
    try {
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/projects/${project.id}/labels`,
      );
      if (res.ok) {
        const data = await res.json();
        projectLabels = data.labels || [];
      }
    } catch {
      // ignore
    }
  }

  export async function loadWorkspaceMembers() {
    try {
      const res = await fetch(`/api/v1/workspaces/${workspace.id}/members`);
      if (res.ok) {
        const data = await res.json();
        workspaceMembers = data.members || [];
      }
    } catch {
      // ignore
    }
  }

  export async function refreshTask() {
    try {
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/projects/${project.id}/tasks/${currentTask.id}`,
      );
      if (res.ok) {
        const updated = await res.json();
        currentTask = updated;
        editTitle = updated.title;
        editDescription = updated.description || "";
        editStatus = updated.status;
        editStatusId = updated.status_id || "";
        editPriority = updated.priority;
        editDueDate = updated.due_date || "";
        ontaskupdated?.(updated);
      }
    } catch {
      // ignore
    }
  }

  async function handleSaveDetails(e?: Event) {
    e?.preventDefault();
    const trimmedTitle = editTitle.trim();
    if (!trimmedTitle) {
      showNotice("Task title is required.", "danger");
      return;
    }

    const csrf = getCsrfToken();
    if (!csrf) return;

    savingDetails = true;
    try {
      const payload: Record<string, unknown> = {
        title: trimmedTitle,
        description: editDescription.trim(),
        priority: editPriority,
        due_date: editDueDate.trim() ? editDueDate.trim() : null,
      };

      if (editStatusId && statuses.some((s) => s.id === editStatusId)) {
        payload.status_id = editStatusId;
      } else {
        payload.status = editStatus;
      }

      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/projects/${project.id}/tasks/${currentTask.id}`,
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

      if (res.ok) {
        const updated: Task = await res.json();
        currentTask = updated;
        ontaskupdated?.(updated);
        showNotice("Task details saved.", "success");
        void loadActivities();
      } else {
        const err = await res.json().catch(() => null);
        showNotice(
          err?.error?.message || "Failed to save task details.",
          "danger",
        );
      }
    } catch {
      showNotice("Network error. Please try again.", "danger");
    } finally {
      savingDetails = false;
    }
  }

  async function handleCreateSubtask(e: Event) {
    e.preventDefault();
    const title = newSubtaskTitle.trim();
    if (!title) return;

    const csrf = getCsrfToken();
    if (!csrf) return;

    creatingSubtask = true;
    try {
      const res = await fetch(
        `/api/v1/workspaces/{workspace.id}/projects/{project.id}/tasks/{currentTask.id}/subtasks`
          .replace("{workspace.id}", workspace.id)
          .replace("{project.id}", project.id)
          .replace("{currentTask.id}", currentTask.id),
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "X-CSRF-Token": csrf,
            Accept: "application/json",
          },
          body: JSON.stringify({
            title,
            status: "unstarted",
            priority: "none",
          }),
        },
      );

      if (res.ok) {
        newSubtaskTitle = "";
        void loadSubtasks();
        void loadActivities();
        showNotice("Subtask added.", "success");
      } else {
        showNotice("Failed to add subtask.", "danger");
      }
    } catch {
      showNotice("Network error adding subtask.", "danger");
    } finally {
      creatingSubtask = false;
    }
  }

  async function handleToggleSubtaskStatus(subtask: Task) {
    const csrf = getCsrfToken();
    if (!csrf) return;

    const nextStatus =
      subtask.status === "completed" ? "unstarted" : "completed";
    try {
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/projects/${project.id}/tasks/${subtask.id}`,
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
      if (res.ok) {
        subtasks = subtasks.map((s) =>
          s.id === subtask.id ? { ...s, status: nextStatus } : s,
        );
        void loadActivities();
      }
    } catch {
      showNotice("Could not toggle subtask status.", "danger");
    }
  }

  async function handlePostComment(e: Event) {
    e.preventDefault();
    const content = newCommentText.trim();
    if (!content) return;

    const csrf = getCsrfToken();
    if (!csrf) return;

    postingComment = true;
    try {
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/projects/${project.id}/tasks/${currentTask.id}/comments`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "X-CSRF-Token": csrf,
            Accept: "application/json",
          },
          body: JSON.stringify({ content }),
        },
      );

      if (res.ok) {
        newCommentText = "";
        void loadComments();
        void loadActivities();
        showNotice("Comment posted.", "success");
      } else {
        showNotice("Failed to post comment.", "danger");
      }
    } catch {
      showNotice("Network error posting comment.", "danger");
    } finally {
      postingComment = false;
    }
  }

  async function handleDeleteComment(commentId: string) {
    if (!confirm("Permanently delete this comment?")) return;
    const csrf = getCsrfToken();
    if (!csrf) return;

    try {
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/projects/${project.id}/tasks/${currentTask.id}/comments/${commentId}`,
        {
          method: "DELETE",
          headers: { "X-CSRF-Token": csrf },
        },
      );
      if (res.ok) {
        comments = comments.filter((c) => c.id !== commentId);
        showNotice("Comment deleted.", "success");
      }
    } catch {
      showNotice("Could not delete comment.", "danger");
    }
  }

  async function handleAttachLabel(labelId: string) {
    selectedLabelId = labelId;
    if (!selectedLabelId) return;
    const csrf = getCsrfToken();
    if (!csrf) return;

    try {
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/projects/${project.id}/tasks/${currentTask.id}/labels`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "X-CSRF-Token": csrf,
          },
          body: JSON.stringify({ label_id: selectedLabelId }),
        },
      );
      if (res.ok) {
        selectedLabelId = "";
        void refreshTask();
        void loadActivities();
      }
    } catch {
      showNotice("Could not attach label.", "danger");
    }
  }

  async function handleDetachLabel(labelId: string) {
    const csrf = getCsrfToken();
    if (!csrf) return;

    try {
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/projects/${project.id}/tasks/${currentTask.id}/labels/${labelId}`,
        {
          method: "DELETE",
          headers: { "X-CSRF-Token": csrf },
        },
      );
      if (res.ok) {
        void refreshTask();
        void loadActivities();
      }
    } catch {
      showNotice("Could not remove label.", "danger");
    }
  }

  async function handleCreateAndAttachLabel(e?: Event, typed?: string) {
    e?.preventDefault();
    if (typed) {
      newLabelName = typed;
      newLabelColor = presetColors[projectLabels.length % presetColors.length];
    }
    const name = newLabelName.trim();
    if (!name) return;

    const csrf = getCsrfToken();
    if (!csrf) return;

    creatingLabel = true;
    try {
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/projects/${project.id}/labels`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "X-CSRF-Token": csrf,
            Accept: "application/json",
          },
          body: JSON.stringify({ name, color: newLabelColor }),
        },
      );

      if (res.ok) {
        const newLabel: Label = await res.json();
        projectLabels = [...projectLabels, newLabel];
        newLabelName = "";
        showCreateLabel = false;
        // Attach immediately to this task
        await fetch(
          `/api/v1/workspaces/${workspace.id}/projects/${project.id}/tasks/${currentTask.id}/labels`,
          {
            method: "POST",
            headers: {
              "Content-Type": "application/json",
              "X-CSRF-Token": csrf,
            },
            body: JSON.stringify({ label_id: newLabel.id }),
          },
        );
        void refreshTask();
        void loadActivities();
      } else {
        const err = await res.json().catch(() => null);
        showNotice(err?.error?.message || "Failed to create label.", "danger");
      }
    } catch {
      showNotice("Network error creating label.", "danger");
    } finally {
      creatingLabel = false;
    }
  }

  async function handleAssignUser(userId: string) {
    selectedMemberId = userId;
    if (!selectedMemberId) return;
    const csrf = getCsrfToken();
    if (!csrf) return;

    try {
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/projects/${project.id}/tasks/${currentTask.id}/assignees`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "X-CSRF-Token": csrf,
          },
          body: JSON.stringify({ user_id: selectedMemberId }),
        },
      );
      if (res.ok) {
        selectedMemberId = "";
        void refreshTask();
        void loadActivities();
      }
    } catch {
      showNotice("Could not assign user.", "danger");
    }
  }

  async function handleUnassignUser(userId: string) {
    const csrf = getCsrfToken();
    if (!csrf) return;

    try {
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/projects/${project.id}/tasks/${currentTask.id}/assignees/${userId}`,
        {
          method: "DELETE",
          headers: { "X-CSRF-Token": csrf },
        },
      );
      if (res.ok) {
        void refreshTask();
        void loadActivities();
      }
    } catch {
      showNotice("Could not unassign user.", "danger");
    }
  }

  // Previous / next within the list the task was opened from.
  const position = $derived(siblings.findIndex((t) => t.id === currentTask.id));
  const prevTask = $derived(position > 0 ? siblings[position - 1] : null);
  const nextTask = $derived(
    position >= 0 && position < siblings.length - 1
      ? siblings[position + 1]
      : null,
  );

  function navigate(target: Task | null) {
    if (!target || !onnavigate) return;
    if (
      dirty &&
      !confirm("Discard your unsaved changes to the title and description?")
    ) {
      return;
    }
    onnavigate(target);
  }

  function handleKeydown(e: KeyboardEvent) {
    const overlay = document.querySelector(".scrim, .menu, .picker");
    if (e.key === "Escape" && !overlay) {
      requestClose();
      return;
    }
    const el = e.target as HTMLElement | null;
    const typing =
      el &&
      (el.isContentEditable ||
        ["INPUT", "TEXTAREA", "SELECT"].includes(el.tagName));
    if (typing || overlay || e.metaKey || e.ctrlKey || e.altKey) return;
    if (e.key === "k") navigate(prevTask);
    else if (e.key === "j") navigate(nextTask);
  }

  // Comments, subtasks and history follow the task being shown.
  $effect(() => {
    void currentTask.id;
    untrack(() => {
      void loadSubtasks();
      void loadComments();
      void loadActivities();
    });
  });

  const statusOptions = $derived<SelectOption[]>(
    statuses.length > 0
      ? statuses.map((s) => ({ value: s.id, label: s.name, color: s.color }))
      : statusCategories.map((c) => ({
          value: c.value,
          label: c.label,
          color: `var(--cat-${c.value})`,
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

  // Keep Tab inside the drawer while it is open.
  function trapFocus(e: KeyboardEvent) {
    if (e.key !== "Tab" || !drawerElement) return;
    const items = [
      ...drawerElement.querySelectorAll<HTMLElement>(
        'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])',
      ),
    ];
    if (items.length === 0) return;
    const first = items[0];
    const last = items[items.length - 1];
    if (e.shiftKey && document.activeElement === first) {
      e.preventDefault();
      last.focus();
    } else if (!e.shiftKey && document.activeElement === last) {
      e.preventDefault();
      first.focus();
    }
  }

  // Available labels not yet attached
  const availableLabels = $derived(
    projectLabels.filter(
      (pl) => !currentTask.labels?.some((tl) => tl.id === pl.id),
    ),
  );

  // Available members not yet assigned
  const availableMembers = $derived(
    workspaceMembers.filter(
      (m) => !currentTask.assignees?.some((a) => a.user_id === m.user_id),
    ),
  );

  // Subtask progress
  const completedSubtasks = $derived(
    subtasks.filter((s) => s.status === "completed").length,
  );
  const subtasksPercent = $derived(
    subtasks.length > 0
      ? Math.round((completedSubtasks / subtasks.length) * 100)
      : 0,
  );

  function getInitials(name?: string | null): string {
    if (!name) return "?";
    return (
      name
        .trim()
        .split(/\s+/)
        .filter(Boolean)
        .map((n) => n[0])
        .slice(0, 2)
        .join("")
        .toUpperCase() || "?"
    );
  }

  onMount(() => {
    void loadProjectLabels();
    void loadWorkspaceMembers();

    // Move focus into the drawer, and give it back to the opener on close.
    const opener = document.activeElement as HTMLElement | null;
    drawerElement?.focus();
    return () => {
      if (opener && document.contains(opener)) opener.focus();
    };
  });
</script>

<svelte:window onkeydown={handleKeydown} />

<div
  class="drawer-backdrop"
  onclick={(e) => {
    if (e.target === e.currentTarget) requestClose();
  }}
  role="presentation"
  in:fadeIn|global={{ duration: 300 }}
  out:fadeIn|global={{ duration: 260 }}
>
  <div
    class="drawer-panel"
    bind:this={drawerElement}
    role="dialog"
    aria-modal="true"
    aria-labelledby="drawer-heading"
    tabindex="-1"
    onkeydown={trapFocus}
    in:drawerIn|global
    out:drawerOut|global
  >
    <header class="drawer-header">
      <div class="header-left">
        {#if onnavigate && siblings.length > 1}
          <div class="stepper" role="group" aria-label="Task navigation">
            <button
              type="button"
              class="btn-quiet btn-icon step-btn"
              disabled={!prevTask}
              onclick={() => navigate(prevTask)}
              aria-label="Previous task"
              title="Previous task (K)"
            >
              <Icon name="arrow-up" size={16} />
            </button>
            <button
              type="button"
              class="btn-quiet btn-icon step-btn"
              disabled={!nextTask}
              onclick={() => navigate(nextTask)}
              aria-label="Next task"
              title="Next task (J)"
            >
              <Icon name="arrow-down" size={16} />
            </button>
            <span class="step-count num" aria-hidden="true"
              >{position + 1} / {siblings.length}</span
            >
          </div>
        {/if}
        <nav class="crumbs" aria-label="Task location">
          <span class="project-title">{project.name}</span>
          <Icon name="chevron-right" size={14} class="crumb-sep" />
          <h2 id="drawer-heading" class="drawer-title">
            <span class="key human-key">{currentTask.human_key}</span>
            <span class="visually-hidden">: {currentTask.title}</span>
          </h2>
        </nav>
      </div>
      <div class="header-actions">
        <button
          type="button"
          class="btn-quiet btn-icon"
          onclick={copyKey}
          aria-label="Copy task key {currentTask.human_key}"
          title="Copy task key"
        >
          <Icon name="copy" size={17} />
        </button>
        {#if ondelete}
          <button
            type="button"
            class="btn-quiet btn-icon header-delete"
            onclick={() => ondelete?.(currentTask)}
            aria-label="Delete task {currentTask.human_key}"
            title="Delete task"
          >
            <Icon name="trash" size={17} />
          </button>
        {/if}
        <span class="header-divider" aria-hidden="true"></span>
        <button
          type="button"
          class="btn-quiet btn-icon"
          onclick={requestClose}
          aria-label="Close task details"
          title="Close (Esc)"
        >
          <Icon name="close" size={18} />
        </button>
      </div>
    </header>

    {#if feedbackMessage}
      <div class="drawer-feedback">
        <Feedback tone={feedbackTone}>{feedbackMessage}</Feedback>
      </div>
    {/if}

    <div class="drawer-body">
      <div class="drawer-main">
        {#key currentTask.id}
          <form onsubmit={handleSaveDetails} class="doc" in:appear>
            <label for="detail-title" class="visually-hidden">Title</label>
            <textarea
              id="detail-title"
              class="title-input"
              bind:value={editTitle}
              required
              rows="1"
              onkeydown={(e) => {
                // A title is one line: Enter saves instead of breaking it.
                if (e.key === "Enter") {
                  e.preventDefault();
                  (e.currentTarget.form as HTMLFormElement).requestSubmit();
                }
              }}></textarea>

            <label for="detail-desc" class="visually-hidden">Description</label>
            <textarea
              id="detail-desc"
              class="desc-input"
              bind:value={editDescription}
              rows="3"
              placeholder="Add a description: context, acceptance criteria, links…"
            ></textarea>

            <div class="save-row">
              {#if dirty}
                <div class="save-bar" in:appear>
                  <span class="unsaved" aria-live="polite">Unsaved changes</span
                  >
                  <Button
                    variant="quiet"
                    onclick={() => {
                      editTitle = currentTask.title;
                      editDescription = currentTask.description || "";
                    }}
                  >
                    {#snippet children()}Discard{/snippet}
                  </Button>
                  <Button
                    type="submit"
                    variant="primary"
                    disabled={savingDetails}
                  >
                    {#snippet children()}{savingDetails
                        ? "Saving…"
                        : "Save details"}{/snippet}
                  </Button>
                </div>
              {:else}
                <span class="saved-note" in:fadeIn>
                  <Icon name="check" size={14} />
                  All changes saved
                </span>
              {/if}
            </div>
          </form>
        {/key}

        <section class="block" aria-labelledby="subtasks-heading">
          <div class="block-head">
            <h3 id="subtasks-heading">Subtasks</h3>
            {#if subtasks.length > 0}
              <span class="count-pill num">
                {completedSubtasks}/{subtasks.length} ({subtasksPercent}%)
              </span>
              <span
                class="mini-progress"
                role="progressbar"
                aria-label="Subtasks completed"
                aria-valuenow={subtasksPercent}
                aria-valuemin={0}
                aria-valuemax={100}
              >
                <span style:width="{subtasksPercent}%"></span>
              </span>
            {/if}
          </div>

          <div class="subtasks inset">
            {#if subtasks.length > 0}
              <ul class="subtask-list" role="list">
                {#each subtasks as sub, i (sub.id)}
                  <li
                    class="subtask-row"
                    class:completed={sub.status === "completed"}
                    in:appear
                  >
                    <label class="check">
                      <input
                        type="checkbox"
                        class="round-check"
                        checked={sub.status === "completed"}
                        onchange={() => handleToggleSubtaskStatus(sub)}
                        aria-label="Mark {sub.human_key} as {sub.status ===
                        'completed'
                          ? 'unstarted'
                          : 'completed'}"
                      />
                      <span class="key">{sub.human_key}</span>
                    </label>
                    <span class="subtask-title">{sub.title}</span>
                    {#if sub.priority !== "none"}
                      <span class="priority" data-priority={sub.priority}>
                        {sub.priority}
                      </span>
                    {/if}
                  </li>
                {/each}
              </ul>
            {:else if !loadingSubtasks}
              <p class="empty-note">
                Break this task into smaller steps. Each subtask gets its own
                key.
              </p>
            {/if}

            {#if canManage}
              <form onsubmit={handleCreateSubtask} class="add-row">
                <Icon name="plus" size={16} class="add-icon" />
                <input
                  type="text"
                  bind:value={newSubtaskTitle}
                  placeholder="New subtask title…"
                  aria-label="New subtask title"
                />
                <Button
                  type="submit"
                  variant="quiet"
                  disabled={creatingSubtask || !newSubtaskTitle.trim()}
                >
                  {#snippet children()}{creatingSubtask
                      ? "Adding…"
                      : "Add subtask"}{/snippet}
                </Button>
              </form>
            {/if}
          </div>
        </section>

        <section class="block" aria-labelledby="comments-heading">
          <div class="block-head">
            <h3 id="comments-heading">Discussion</h3>
            {#if comments.length > 0}
              <span class="count-pill num">{comments.length}</span>
            {/if}
          </div>

          <div class="comments">
            {#if comments.length === 0 && !loadingComments}
              <p class="empty-note">
                No comments yet. Start the conversation below.
              </p>
            {:else}
              {#each comments as comment, i (comment.id)}
                <article class="comment" in:appear>
                  <span
                    class="avatar"
                    style:--avatar-hue={identityHue(comment.user_id)}
                    aria-hidden="true">{getInitials(comment.author_name)}</span
                  >
                  <div class="comment-main">
                    <div class="comment-header">
                      <span class="author-name">{comment.author_name}</span>
                      <span class="author-handle"
                        >@{comment.author_username}</span
                      >
                      <time
                        class="comment-time"
                        datetime={comment.created_at}
                        title={new Date(comment.created_at).toLocaleString()}
                      >
                        {relativeTime(comment.created_at)}
                      </time>
                      {#if canDeleteComment(comment)}
                        <button
                          type="button"
                          class="btn-quiet btn-icon comment-delete"
                          onclick={() => handleDeleteComment(comment.id)}
                          title="Delete comment"
                          aria-label="Delete comment by {comment.author_name}"
                        >
                          <Icon name="trash" size={15} />
                        </button>
                      {/if}
                    </div>
                    <p class="comment-body inset">{comment.content}</p>
                  </div>
                </article>
              {/each}
            {/if}
          </div>

          <form onsubmit={handlePostComment} class="composer inset">
            <textarea
              bind:value={newCommentText}
              rows="2"
              placeholder="Leave a comment, update, or question…"
              aria-label="Write a comment"
              onkeydown={(e) => {
                if (e.key === "Enter" && (e.metaKey || e.ctrlKey)) {
                  e.preventDefault();
                  (e.currentTarget.form as HTMLFormElement).requestSubmit();
                }
              }}></textarea>
            <div class="composer-actions">
              <span class="note composer-hint">
                <kbd>Ctrl</kbd> <kbd>Enter</kbd> to post
              </span>
              <Button
                type="submit"
                variant="primary"
                disabled={postingComment || !newCommentText.trim()}
              >
                {#snippet children()}{postingComment
                    ? "Posting…"
                    : "Post comment"}{/snippet}
              </Button>
            </div>
          </form>
        </section>
      </div>

      <aside class="drawer-sidebar" aria-label="Task attributes">
        <h3 class="side-title">Properties</h3>
        <dl class="props">
          <div class="prop">
            <dt><label for="detail-status">Status</label></dt>
            <dd>
              <Select
                id="detail-status"
                variant="chip"
                value={statuses.length > 0 ? editStatusId : editStatus}
                options={statusOptions}
                onchange={(v) => {
                  if (statuses.length > 0) {
                    editStatusId = v;
                    const matched = statuses.find((s) => s.id === v);
                    if (matched) editStatus = matched.category;
                  } else {
                    editStatusId = "";
                    editStatus = v as Task["status"];
                  }
                  void handleSaveDetails();
                }}
              />
            </dd>
          </div>

          <div class="prop">
            <dt><label for="detail-priority">Priority</label></dt>
            <dd>
              <Select
                id="detail-priority"
                variant="chip"
                value={editPriority}
                options={priorityOptions}
                onchange={(v) => {
                  editPriority = v as Task["priority"];
                  void handleSaveDetails();
                }}
              />
            </dd>
          </div>

          <div class="prop">
            <dt><span id="due-label">Due date</span></dt>
            <dd>
              <DatePicker
                id="detail-due-date"
                variant="chip"
                label="Due date"
                placeholder="No due date"
                value={editDueDate}
                onchange={(v) => {
                  editDueDate = v;
                  void handleSaveDetails();
                }}
              />
            </dd>
          </div>

          <div class="prop prop-stack" aria-labelledby="assignees-heading">
            <dt><span id="assignees-heading">Assignees</span></dt>
            <dd class="chips">
              {#each currentTask.assignees ?? [] as a (a.user_id)}
                <span class="person-chip" in:appear>
                  <span
                    class="avatar avatar-sm"
                    style:--avatar-hue={identityHue(a.user_id)}
                    aria-hidden="true">{getInitials(a.display_name)}</span
                  >
                  <span class="person-name">{a.display_name}</span>
                  {#if canManage}
                    <button
                      type="button"
                      class="chip-remove"
                      onclick={() => handleUnassignUser(a.user_id)}
                      title="Remove {a.display_name}"
                      aria-label="Remove {a.display_name}"
                    >
                      <Icon name="close" size={12} />
                    </button>
                  {/if}
                </span>
              {:else}
                <span class="meta-empty">Unassigned</span>
              {/each}
              {#if canManage}
                <MultiPicker
                  label="Assign a workspace member"
                  searchLabel="Search members"
                  emptyText="No members match"
                  items={workspaceMembers.map((m) => ({
                    value: m.user_id,
                    label: m.display_name,
                    sub: `@${m.username}`,
                    hue: identityHue(m.user_id),
                    initials: getInitials(m.display_name),
                  }))}
                  selected={(currentTask.assignees ?? []).map((a) => a.user_id)}
                  ontoggle={(id, on) =>
                    on ? handleAssignUser(id) : handleUnassignUser(id)}
                />
              {/if}
            </dd>
          </div>

          <div class="prop prop-stack" aria-labelledby="labels-heading">
            <dt><span id="labels-heading">Labels</span></dt>
            <dd class="chips">
              {#each currentTask.labels ?? [] as l (l.id)}
                <span
                  class="label-badge label-chip"
                  style:--badge-color={l.color}
                  title="Label: {l.name}"
                  in:appear
                >
                  <span class="dot" aria-hidden="true"></span>
                  <span>{l.name}</span>
                  {#if canManage}
                    <button
                      type="button"
                      class="chip-remove"
                      onclick={() => handleDetachLabel(l.id)}
                      title="Detach {l.name}"
                      aria-label="Detach {l.name}"
                    >
                      <Icon name="close" size={12} />
                    </button>
                  {/if}
                </span>
              {:else}
                <span class="meta-empty">No labels</span>
              {/each}
              {#if canManage}
                <MultiPicker
                  label="Add a label"
                  searchLabel="Search or create a label"
                  emptyText="Type to create a label"
                  createLabel={(q) => `Create label “${q}”`}
                  items={projectLabels.map((l) => ({
                    value: l.id,
                    label: l.name,
                    color: l.color,
                  }))}
                  selected={(currentTask.labels ?? []).map((l) => l.id)}
                  ontoggle={(id, on) =>
                    on ? handleAttachLabel(id) : handleDetachLabel(id)}
                  oncreate={(name) =>
                    handleCreateAndAttachLabel(undefined, name)}
                />
              {/if}
            </dd>
          </div>
        </dl>

        <dl class="meta">
          <div>
            <dt>Created</dt>
            <dd title={new Date(currentTask.created_at).toLocaleString()}>
              {relativeTime(currentTask.created_at)}
            </dd>
          </div>
          <div>
            <dt>Updated</dt>
            <dd title={new Date(currentTask.updated_at).toLocaleString()}>
              {relativeTime(currentTask.updated_at)}
            </dd>
          </div>
        </dl>

        <div class="activity" aria-labelledby="activity-heading">
          <h3 id="activity-heading" class="side-title">Activity</h3>
          {#if activities.length === 0 && !loadingActivities}
            <p class="meta-empty">No activity recorded</p>
          {:else}
            <ol class="timeline" role="list">
              {#each activities as act, i (act.id)}
                <li class="timeline-item" in:appear>
                  <span class="timeline-marker" aria-hidden="true"></span>
                  <div class="timeline-content">
                    <span class="timeline-action">
                      <strong>{act.actor_name || "System"}</strong>
                      {#if act.action === "task.created"}
                        created this task
                      {:else if act.action === "task.status_changed"}
                        updated status to {act.details?.status || "new status"}
                      {:else if act.action === "task.priority_changed"}
                        set priority to {act.details?.priority ||
                          "new priority"}
                      {:else if act.action === "label.attached"}
                        attached label "{act.details?.name || ""}"
                      {:else if act.action === "label.detached"}
                        removed a label
                      {:else if act.action === "user.assigned"}
                        assigned {act.details?.display_name || "member"}
                      {:else if act.action === "user.unassigned"}
                        unassigned a member
                      {:else if act.action === "subtask.created"}
                        added subtask "{act.details?.title || ""}"
                      {:else if act.action === "comment.added"}
                        added a comment
                      {:else}
                        {act.action}
                      {/if}
                    </span>
                    <time
                      class="timeline-date"
                      datetime={act.created_at}
                      title={new Date(act.created_at).toLocaleString()}
                    >
                      {relativeTime(act.created_at)}
                    </time>
                  </div>
                </li>
              {/each}
            </ol>
          {/if}
        </div>
      </aside>
    </div>
  </div>
</div>

<Toast message={toastMessage} onclear={() => (toastMessage = null)} />

<style>
  .drawer-backdrop {
    position: fixed;
    inset: 0;
    z-index: var(--layer-overlay);
    display: flex;
    justify-content: flex-end;
    background: var(--drawer-scrim);
    backdrop-filter: blur(3px);
    -webkit-backdrop-filter: blur(3px);
  }

  .drawer-panel {
    width: var(--drawer-width);
    height: 100%;
    display: flex;
    flex-direction: column;
    background:
      var(--card-fill) padding-box,
      var(--card-edge) border-box;
    border: var(--border-width) solid transparent;
    border-right: 0;
    border-radius: var(--radius-xl) 0 0 var(--radius-xl);
    box-shadow: var(--shadow-modal);
    outline: none;
    overflow: hidden;
  }

  /* ── Header ──────────────────────────────────────────────────── */
  .drawer-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    min-height: 3.75rem;
    padding: var(--space-2) var(--space-3) var(--space-2) var(--space-4);
    padding-top: max(var(--space-2), env(safe-area-inset-top));
    border-bottom: var(--border-width) solid var(--line);
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    min-width: 0;
  }

  .stepper {
    display: flex;
    align-items: center;
    gap: 2px;
    padding-right: var(--space-3);
    border-right: var(--border-width) solid var(--line);
  }

  .step-btn {
    width: 2.25rem;
    min-height: 2.25rem;
  }

  .step-count {
    margin-left: var(--space-1);
    font-size: var(--text-xs);
    color: var(--text-muted);
  }

  .crumbs {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    min-width: 0;
  }

  .crumbs :global(.crumb-sep) {
    color: var(--text-muted);
    opacity: 0.7;
  }

  .project-title {
    font-size: var(--text-small);
    color: var(--text-muted);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .drawer-title {
    font-size: var(--text-small);
  }

  .human-key {
    font-size: var(--text-small);
    color: var(--text);
  }

  .header-actions {
    display: flex;
    align-items: center;
    gap: 2px;
  }

  .header-divider {
    width: 1px;
    height: 1.25rem;
    margin: 0 var(--space-1);
    background: var(--line-strong);
  }

  .header-delete:hover:not(:disabled) {
    color: var(--danger);
    background: var(--danger-soft);
  }

  .drawer-feedback {
    padding: var(--space-3) var(--space-6) 0;
  }

  /* ── Body ────────────────────────────────────────────────────── */
  .drawer-body {
    flex: 1;
    min-height: 0;
    display: grid;
    grid-template-columns: minmax(0, 1fr) 19rem;
    overflow: hidden;
  }

  .drawer-main {
    display: grid;
    align-content: start;
    gap: var(--space-8);
    padding: var(--space-6) var(--space-8) var(--space-10);
    overflow-y: auto;
    overscroll-behavior: contain;
  }

  /* The task reads like a document, not a form. */
  .doc {
    display: grid;
    gap: var(--space-2);
  }

  .title-input,
  .desc-input {
    width: calc(100% + var(--space-6));
    margin: 0 calc(var(--space-3) * -1);
    padding: var(--space-2) var(--space-3);
    border: var(--border-width) solid transparent;
    border-radius: var(--radius-sm);
    background: transparent;
    resize: none;
    field-sizing: content;
    transition:
      background-color var(--transition-fast),
      border-color var(--transition-fast);
  }

  .title-input {
    min-height: 3rem;
    font-size: 1.625rem;
    font-weight: 520;
    letter-spacing: -0.025em;
    line-height: 1.25;
    color: var(--text);
  }

  .desc-input {
    min-height: 5.5rem;
    font-size: var(--text-body);
    line-height: 1.65;
    color: var(--text-secondary);
  }

  .title-input:hover:not(:focus),
  .desc-input:hover:not(:focus) {
    background: var(--hover);
  }

  .title-input:focus,
  .desc-input:focus {
    background: var(--surface-sunken);
    border-color: var(--focus);
    box-shadow: 0 0 0 3px var(--accent-soft);
  }

  .save-row {
    display: flex;
    justify-content: flex-end;
    min-height: var(--control-height);
  }

  .save-bar {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    width: 100%;
    padding: var(--space-1) var(--space-1) var(--space-1) var(--space-4);
    border-radius: var(--radius);
    background: var(--warning-soft);
  }

  .unsaved {
    margin-right: auto;
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    font-size: var(--text-small);
    font-weight: 560;
    color: var(--warning);
  }

  .unsaved::before {
    content: "";
    width: 0.45rem;
    height: 0.45rem;
    border-radius: 50%;
    background: currentColor;
    animation: breathe 1.6s var(--ease-in-out) infinite alternate;
  }

  @keyframes breathe {
    to {
      opacity: 0.35;
    }
  }

  .saved-note {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    font-size: var(--text-xs);
    color: var(--text-muted);
  }

  .block {
    display: grid;
    gap: var(--space-3);
  }

  .block-head {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  .block-head h3 {
    font-size: var(--text-ui);
    font-weight: 560;
  }

  .count-pill {
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--text-muted);
    padding: 0.125rem 0.5rem;
    border-radius: var(--radius-full);
    background: var(--selected);
  }

  .mini-progress {
    flex: 1;
    max-width: 10rem;
    height: 0.3rem;
    margin-left: auto;
    border-radius: var(--radius-full);
    background: var(--surface-sunken);
    overflow: hidden;
  }

  .mini-progress span {
    display: block;
    height: 100%;
    border-radius: inherit;
    background: var(--cat-completed);
    transition: width var(--duration-slow) var(--ease-spring);
  }

  .subtasks {
    overflow: hidden;
  }

  .subtask-list {
    list-style: none;
    margin: 0;
    padding: 0;
  }

  .subtask-row {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    min-height: 2.75rem;
    padding: 0 var(--space-3);
    border-bottom: var(--border-width) solid var(--line);
  }

  .check {
    display: inline-flex;
    align-items: center;
    gap: var(--space-3);
    min-height: 2.75rem;
    cursor: pointer;
  }

  /* A round, animated check that matches the product, not the browser. */
  .round-check {
    appearance: none;
    -webkit-appearance: none;
    display: grid;
    place-items: center;
    width: 1.125rem;
    height: 1.125rem;
    margin: 0;
    border-radius: 50%;
    border: 1.5px solid var(--border);
    background: transparent;
    cursor: pointer;
    transition:
      background-color 200ms var(--ease-out),
      border-color 200ms var(--ease-out),
      transform 160ms var(--ease-out);
  }

  .round-check::after {
    content: "";
    width: 0.3rem;
    height: 0.55rem;
    margin-top: -2px;
    border: solid var(--button-text);
    border-width: 0 2px 2px 0;
    transform: rotate(45deg) scale(0);
    transition: transform 260ms var(--ease-bounce);
  }

  .round-check:checked {
    background: var(--cat-completed);
    border-color: var(--cat-completed);
  }

  .round-check:checked::after {
    transform: rotate(45deg) scale(1);
  }

  .round-check:active {
    transform: scale(0.88);
  }

  .subtask-title {
    flex: 1;
    min-width: 0;
    color: var(--text);
    transition: color var(--transition-fast);
  }

  .subtask-row.completed .subtask-title {
    color: var(--text-muted);
    text-decoration: line-through;
    text-decoration-color: var(--line-strong);
  }

  .add-row {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: 0 var(--space-1) 0 var(--space-3);
  }

  .add-row :global(.add-icon) {
    color: var(--text-muted);
  }

  .add-row input {
    flex: 1;
    min-width: 0;
    border-color: transparent;
    background: transparent;
    padding-left: var(--space-1);
  }

  .add-row input:focus {
    box-shadow: none;
    border-color: transparent;
    background: transparent;
  }

  .empty-note,
  .meta-empty {
    font-size: var(--text-small);
    color: var(--text-muted);
  }

  .subtasks .empty-note {
    padding: var(--space-4) var(--space-4) var(--space-2);
  }

  .comments {
    display: grid;
    gap: var(--space-4);
  }

  .comment {
    display: flex;
    gap: var(--space-3);
  }

  .comment-main {
    flex: 1;
    min-width: 0;
  }

  .comment-header {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    flex-wrap: wrap;
    min-height: 2rem;
  }

  .author-name {
    font-weight: 600;
    font-size: var(--text-small);
    color: var(--text);
  }

  .author-handle,
  .comment-time {
    font-size: var(--text-xs);
    color: var(--text-muted);
  }

  .comment-delete {
    margin-left: auto;
    width: 2rem;
    min-height: 2rem;
    color: var(--text-muted);
    opacity: 0;
    transition: opacity var(--transition-fast);
  }

  .comment:hover .comment-delete,
  .comment-delete:focus-visible {
    opacity: 1;
  }

  @media (hover: none) {
    .comment-delete {
      opacity: 1;
    }
  }

  .comment-delete:hover:not(:disabled) {
    color: var(--danger);
    background: var(--danger-soft);
  }

  .comment-body {
    margin-top: var(--space-1);
    padding: var(--space-3) var(--space-4);
    border-radius: 0.25rem var(--radius) var(--radius) var(--radius);
    color: var(--text);
    white-space: pre-wrap;
  }

  .composer {
    display: grid;
    gap: var(--space-2);
    padding: var(--space-2);
    transition: box-shadow var(--transition-fast);
  }

  .composer:focus-within {
    box-shadow: 0 0 0 3px var(--accent-soft);
  }

  .composer textarea {
    min-height: 3.5rem;
    border: 0;
    background: transparent;
    resize: none;
    field-sizing: content;
    max-height: 14rem;
  }

  .composer textarea:focus {
    box-shadow: none;
    background: transparent;
  }

  .composer-actions {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    padding: 0 0 0 var(--space-2);
  }

  .composer-hint {
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
  }

  /* ── Properties ──────────────────────────────────────────────── */
  .drawer-sidebar {
    display: grid;
    align-content: start;
    gap: var(--space-6);
    padding: var(--space-5) var(--space-5) var(--space-8);
    border-left: var(--border-width) solid var(--line);
    background: light-dark(
      oklch(0.975 calc(var(--tint) * 0.8) var(--brand-hue) / 0.6),
      oklch(0.1 0.01 var(--brand-hue) / 0.35)
    );
    overflow-y: auto;
    overscroll-behavior: contain;
  }

  .side-title {
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--text-muted);
    letter-spacing: 0;
  }

  .props,
  .meta {
    margin: 0;
    display: grid;
    gap: var(--space-1);
  }

  .props {
    margin-top: calc(var(--space-4) * -1);
  }

  .prop {
    display: grid;
    grid-template-columns: 5.5rem minmax(0, 1fr);
    align-items: center;
    gap: var(--space-3);
    min-height: 2.5rem;
  }

  .prop-stack {
    align-items: start;
    padding-top: var(--space-1);
  }

  .prop-stack dt {
    padding-top: 0.4rem;
  }

  .prop dt,
  .meta dt {
    font-size: var(--text-small);
    color: var(--text-muted);
  }

  .prop dt label {
    cursor: pointer;
  }

  .prop dd,
  .meta dd {
    margin: 0;
    min-width: 0;
  }

  .chips {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-1);
  }

  .chips .meta-empty {
    padding: 0 var(--space-1);
  }

  .person-chip {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    min-height: 2rem;
    padding: 0 0.25rem;
    border-radius: var(--radius-full);
    background:
      var(--inset-fill) padding-box,
      var(--inset-edge) border-box;
    border: var(--border-width) solid transparent;
    font-size: var(--text-small);
    color: var(--text);
  }

  .person-name {
    padding-right: 0.125rem;
  }

  .label-chip {
    min-height: 2rem;
    padding-right: 0.25rem;
  }

  .chip-remove {
    min-height: 1.5rem;
    width: 1.5rem;
    padding: 0;
    border: 0;
    border-radius: 50%;
    background: transparent;
    box-shadow: none;
    color: var(--text-muted);
  }

  .chip-remove:hover:not(:disabled) {
    background: var(--danger-soft);
    color: var(--danger);
  }

  .meta {
    padding-top: var(--space-4);
    border-top: var(--border-width) solid var(--line);
  }

  .meta div {
    display: flex;
    justify-content: space-between;
    gap: var(--space-3);
  }

  .meta dd {
    font-size: var(--text-small);
    color: var(--text-secondary);
  }

  .activity {
    display: grid;
    gap: var(--space-3);
    padding-top: var(--space-4);
    border-top: var(--border-width) solid var(--line);
  }

  .timeline {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
  }

  .timeline-item {
    position: relative;
    display: flex;
    gap: var(--space-3);
    padding-bottom: var(--space-4);
  }

  .timeline-item::before {
    content: "";
    position: absolute;
    left: 4px;
    top: 14px;
    bottom: 0;
    width: 1px;
    background: var(--line-strong);
  }

  .timeline-item:last-child::before {
    display: none;
  }

  .timeline-marker {
    position: relative;
    width: 9px;
    height: 9px;
    margin-top: 5px;
    border-radius: 50%;
    background: var(--surface-raised);
    border: 2px solid var(--accent);
    flex-shrink: 0;
  }

  .timeline-content {
    display: grid;
    gap: 0.125rem;
    min-width: 0;
  }

  .timeline-action {
    font-size: var(--text-small);
    color: var(--text-secondary);
    line-height: 1.45;
  }

  .timeline-date {
    font-size: var(--text-xs);
    color: var(--text-muted);
  }

  @media (max-width: 52rem) {
    .drawer-panel {
      width: 100%;
      border-radius: 0;
      border-left: 0;
    }

    .drawer-body {
      display: block;
      overflow-y: auto;
      overscroll-behavior: contain;
    }

    .drawer-main {
      overflow: visible;
      padding: var(--space-5) var(--space-4) var(--space-6);
    }

    .drawer-sidebar {
      overflow: visible;
      border-left: 0;
      border-top: var(--border-width) solid var(--line);
      padding: var(--space-5) var(--space-4);
      padding-bottom: max(var(--space-8), env(safe-area-inset-bottom));
    }

    .drawer-header {
      padding-left: var(--space-3);
    }

    .step-count,
    .project-title,
    .crumbs :global(.crumb-sep),
    .composer-hint {
      display: none;
    }

    .title-input {
      font-size: 1.3125rem;
    }
  }
</style>
