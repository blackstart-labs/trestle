<script lang="ts">
  import { onMount } from "svelte";
  import Button from "../ui/Button.svelte";
  import InputField from "../ui/InputField.svelte";
  import Feedback from "../ui/Feedback.svelte";
  import Icon from "../ui/Icon.svelte";
  import type { Task } from "./TasksView.svelte";
  import type { Project } from "./ProjectsView.svelte";

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
    currentUserId,
    onclose,
    ontaskupdated,
  }: {
    task: Task;
    workspace: { id: string; name: string; slug: string; role: string };
    project: Project;
    currentUserId?: string;
    onclose: () => void;
    ontaskupdated?: (task: Task) => void;
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

  function showNotice(msg: string, tone: "success" | "danger" | "neutral") {
    feedbackMessage = msg;
    feedbackTone = tone;
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
        editPriority = updated.priority;
        editDueDate = updated.due_date || "";
        ontaskupdated?.(updated);
      }
    } catch {
      // ignore
    }
  }

  async function handleSaveDetails(e: Event) {
    e.preventDefault();
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
        status: editStatus,
        priority: editPriority,
        due_date: editDueDate.trim() ? editDueDate.trim() : null,
      };

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

  async function handleAttachLabel() {
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

  async function handleCreateAndAttachLabel(e: Event) {
    e.preventDefault();
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

  async function handleAssignUser() {
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

  function handleKeydown(e: KeyboardEvent) {
    if (e.key === "Escape") {
      onclose();
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
    void loadSubtasks();
    void loadComments();
    void loadActivities();
    void loadProjectLabels();
    void loadWorkspaceMembers();

    // Trap focus inside drawer if possible
    drawerElement?.focus();
  });
</script>

<svelte:window onkeydown={handleKeydown} />

<!-- Backdrop -->
<div
  class="drawer-backdrop"
  onclick={(e) => {
    if (e.target === e.currentTarget) onclose();
  }}
  role="presentation"
>
  <div
    class="drawer-panel"
    bind:this={drawerElement}
    role="dialog"
    aria-modal="true"
    aria-labelledby="drawer-heading"
    tabindex="-1"
  >
    <!-- Top Bar -->
    <header class="drawer-header">
      <div class="header-left">
        <span class="human-key">{currentTask.human_key}</span>
        <span class="project-title">{project.name}</span>
      </div>
      <button
        type="button"
        class="close-btn"
        onclick={onclose}
        aria-label="Close task details"
      >
        <Icon name="close" size={18} />
      </button>
    </header>

    {#if feedbackMessage}
      <div class="drawer-feedback">
        <Feedback tone={feedbackTone}>{feedbackMessage}</Feedback>
      </div>
    {/if}

    <div class="drawer-body">
      <!-- Left Column: Primary Content -->
      <div class="drawer-main">
        <form onsubmit={handleSaveDetails} class="main-form">
          <div class="field">
            <label for="detail-title" class="field-label">Title</label>
            <input
              id="detail-title"
              class="title-input"
              bind:value={editTitle}
              required
            />
          </div>

          <div class="field">
            <label for="detail-desc" class="field-label">Description</label>
            <textarea
              id="detail-desc"
              class="desc-textarea"
              bind:value={editDescription}
              rows="4"
              placeholder="Add details, acceptance criteria, or context…"
            ></textarea>
          </div>

          <div class="action-row">
            <Button type="submit" disabled={savingDetails}>
              {#snippet children()}{savingDetails
                  ? "Saving…"
                  : "Save details"}{/snippet}
            </Button>
          </div>
        </form>

        <!-- Subtasks Section -->
        <section class="section-card" aria-labelledby="subtasks-heading">
          <div class="section-header">
            <div class="section-title-wrap">
              <h3 id="subtasks-heading" class="section-heading">Subtasks</h3>
              {#if subtasks.length > 0}
                <span class="progress-pill">
                  {completedSubtasks}/{subtasks.length} ({subtasksPercent}%)
                </span>
              {/if}
            </div>
          </div>

          {#if subtasks.length > 0}
            <div
              class="progress-bar-bg"
              role="progressbar"
              aria-valuenow={subtasksPercent}
              aria-valuemin={0}
              aria-valuemax={100}
            >
              <div
                class="progress-bar-fill"
                style="width: {subtasksPercent}%;"
              ></div>
            </div>

            <ul class="subtask-list" role="list">
              {#each subtasks as sub (sub.id)}
                <li
                  class="subtask-row"
                  class:completed={sub.status === "completed"}
                >
                  <label class="checkbox-wrap">
                    <input
                      type="checkbox"
                      checked={sub.status === "completed"}
                      onchange={() => handleToggleSubtaskStatus(sub)}
                      aria-label="Mark {sub.human_key} as {sub.status ===
                      'completed'
                        ? 'unstarted'
                        : 'completed'}"
                    />
                    <span class="subtask-key">{sub.human_key}</span>
                  </label>
                  <span class="subtask-title">{sub.title}</span>
                  {#if sub.priority !== "none"}
                    <span class="priority-tag" data-priority={sub.priority}>
                      {sub.priority}
                    </span>
                  {/if}
                </li>
              {/each}
            </ul>
          {:else if !loadingSubtasks}
            <p class="empty-note">
              No subtasks yet. Break down this task into smaller steps below.
            </p>
          {/if}

          {#if canManage}
            <form onsubmit={handleCreateSubtask} class="add-subtask-form">
              <input
                type="text"
                class="inline-input"
                bind:value={newSubtaskTitle}
                placeholder="New subtask title…"
                aria-label="New subtask title"
              />
              <Button type="submit" disabled={creatingSubtask}>
                {#snippet children()}{creatingSubtask
                    ? "Adding…"
                    : "Add subtask"}{/snippet}
              </Button>
            </form>
          {/if}
        </section>

        <!-- Comments & Discussion Section -->
        <section class="section-card" aria-labelledby="comments-heading">
          <h3 id="comments-heading" class="section-heading">Discussion</h3>

          <div class="comments-stream">
            {#if comments.length === 0 && !loadingComments}
              <p class="empty-note">
                No comments yet. Start the conversation below.
              </p>
            {:else}
              {#each comments as comment (comment.id)}
                <div class="comment-card">
                  <div class="comment-header">
                    <div class="comment-author-badge">
                      <span class="avatar-circle">
                        {getInitials(comment.author_name)}
                      </span>
                      <span class="author-name">{comment.author_name}</span>
                      <span class="author-handle"
                        >@{comment.author_username}</span
                      >
                    </div>
                    <div class="comment-meta">
                      <time
                        class="comment-time"
                        datetime={comment.created_at}
                        title={comment.created_at}
                      >
                        {new Date(comment.created_at).toLocaleDateString(
                          undefined,
                          {
                            month: "short",
                            day: "numeric",
                            hour: "2-digit",
                            minute: "2-digit",
                          },
                        )}
                      </time>
                      {#if canDeleteComment(comment)}
                        <button
                          type="button"
                          class="delete-comment-btn"
                          onclick={() => handleDeleteComment(comment.id)}
                          title="Delete comment"
                          aria-label="Delete comment by {comment.author_name}"
                        >
                          <Icon name="trash" size={14} />
                        </button>
                      {/if}
                    </div>
                  </div>
                  <p class="comment-body">{comment.content}</p>
                </div>
              {/each}
            {/if}
          </div>

          <form onsubmit={handlePostComment} class="post-comment-form">
            <textarea
              class="comment-textarea"
              bind:value={newCommentText}
              rows="3"
              placeholder="Leave a comment, update, or question…"
              aria-label="Write a comment"></textarea>
            <div class="action-row">
              <Button
                type="submit"
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

      <!-- Right Column: Metadata Sidebar -->
      <aside class="drawer-sidebar" aria-label="Task attributes">
        <!-- Status & Priority -->
        <div class="meta-group">
          <label for="detail-status" class="meta-label">Status</label>
          <select
            id="detail-status"
            class="meta-select"
            bind:value={editStatus}
            onchange={handleSaveDetails}
          >
            {#each statusCategories as s (s.value)}
              <option value={s.value}>{s.label}</option>
            {/each}
          </select>
        </div>

        <div class="meta-group">
          <label for="detail-priority" class="meta-label">Priority</label>
          <select
            id="detail-priority"
            class="meta-select"
            bind:value={editPriority}
            onchange={handleSaveDetails}
          >
            {#each priorities as p (p.value)}
              <option value={p.value}>{p.label}</option>
            {/each}
          </select>
        </div>

        <div class="meta-group">
          <label for="detail-due-date" class="meta-label">Due Date</label>
          <input
            type="date"
            id="detail-due-date"
            class="meta-input"
            bind:value={editDueDate}
            onchange={handleSaveDetails}
          />
        </div>

        <hr class="meta-divider" />

        <!-- Assignees Section -->
        <div class="meta-group" aria-labelledby="assignees-heading">
          <div class="meta-header-row">
            <span id="assignees-heading" class="meta-label">Assignees</span>
          </div>

          {#if currentTask.assignees && currentTask.assignees.length > 0}
            <div class="assignee-list">
              {#each currentTask.assignees as a (a.user_id)}
                <div class="assignee-chip">
                  <span class="avatar-small">{getInitials(a.display_name)}</span
                  >
                  <span class="assignee-name">{a.display_name}</span>
                  {#if canManage}
                    <button
                      type="button"
                      class="chip-remove-btn"
                      onclick={() => handleUnassignUser(a.user_id)}
                      title="Remove {a.display_name}"
                      aria-label="Remove {a.display_name}"
                    >
                      <Icon name="close" size={12} />
                    </button>
                  {/if}
                </div>
              {/each}
            </div>
          {:else}
            <p class="meta-empty">Unassigned</p>
          {/if}

          {#if canManage && availableMembers.length > 0}
            <div class="picker-row">
              <select
                class="meta-select"
                bind:value={selectedMemberId}
                aria-label="Assign a workspace member"
              >
                <option value="">Assign member…</option>
                {#each availableMembers as m (m.user_id)}
                  <option value={m.user_id}
                    >{m.display_name} (@{m.username})</option
                  >
                {/each}
              </select>
              <Button
                onclick={handleAssignUser}
                disabled={!selectedMemberId}
                type="button"
              >
                {#snippet children()}Assign{/snippet}
              </Button>
            </div>
          {/if}
        </div>

        <hr class="meta-divider" />

        <!-- Labels Section -->
        <div class="meta-group" aria-labelledby="labels-heading">
          <div class="meta-header-row">
            <span id="labels-heading" class="meta-label">Labels</span>
          </div>

          {#if currentTask.labels && currentTask.labels.length > 0}
            <div class="label-badges">
              {#each currentTask.labels as l (l.id)}
                <span
                  class="label-badge"
                  style="--badge-color: {l.color};"
                  title="Label: {l.name}"
                >
                  <span class="badge-dot"></span>
                  <span class="badge-text">{l.name}</span>
                  {#if canManage}
                    <button
                      type="button"
                      class="chip-remove-btn"
                      onclick={() => handleDetachLabel(l.id)}
                      title="Detach {l.name}"
                      aria-label="Detach {l.name}"
                    >
                      <Icon name="close" size={12} />
                    </button>
                  {/if}
                </span>
              {/each}
            </div>
          {:else}
            <p class="meta-empty">No labels attached</p>
          {/if}

          {#if canManage}
            {#if availableLabels.length > 0}
              <div class="picker-row">
                <select
                  class="meta-select"
                  bind:value={selectedLabelId}
                  aria-label="Select label to attach"
                >
                  <option value="">Attach label…</option>
                  {#each availableLabels as pl (pl.id)}
                    <option value={pl.id}>{pl.name}</option>
                  {/each}
                </select>
                <Button
                  onclick={handleAttachLabel}
                  disabled={!selectedLabelId}
                  type="button"
                >
                  {#snippet children()}Attach{/snippet}
                </Button>
              </div>
            {/if}

            <div class="create-label-action">
              {#if !showCreateLabel}
                <button
                  type="button"
                  class="text-link-btn"
                  onclick={() => (showCreateLabel = true)}
                >
                  <Icon name="plus" size={14} /> New project label
                </button>
              {:else}
                <form
                  onsubmit={handleCreateAndAttachLabel}
                  class="mini-label-form"
                >
                  <input
                    type="text"
                    class="meta-input"
                    bind:value={newLabelName}
                    placeholder="Label name"
                    required
                    aria-label="New label name"
                  />
                  <div
                    class="color-palette"
                    role="radiogroup"
                    aria-label="Color"
                  >
                    {#each presetColors as col (col)}
                      <button
                        type="button"
                        class="color-dot"
                        class:selected={newLabelColor === col}
                        style="background-color: {col};"
                        onclick={() => (newLabelColor = col)}
                        aria-label="Color {col}"
                      ></button>
                    {/each}
                  </div>
                  <div class="mini-form-actions">
                    <Button
                      type="button"
                      onclick={() => (showCreateLabel = false)}
                      disabled={creatingLabel}
                    >
                      {#snippet children()}Cancel{/snippet}
                    </Button>
                    <Button type="submit" disabled={creatingLabel}>
                      {#snippet children()}{creatingLabel
                          ? "Saving…"
                          : "Save"}{/snippet}
                    </Button>
                  </div>
                </form>
              {/if}
            </div>
          {/if}
        </div>

        <hr class="meta-divider" />

        <!-- Activity Timeline -->
        <div class="meta-group" aria-labelledby="activity-heading">
          <span id="activity-heading" class="meta-label">Activity History</span>
          {#if activities.length === 0 && !loadingActivities}
            <p class="meta-empty">No activity recorded</p>
          {:else}
            <ol class="timeline" role="list">
              {#each activities as act (act.id)}
                <li class="timeline-item">
                  <div class="timeline-marker"></div>
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
                    <time class="timeline-date" datetime={act.created_at}>
                      {new Date(act.created_at).toLocaleDateString(undefined, {
                        month: "short",
                        day: "numeric",
                        hour: "2-digit",
                        minute: "2-digit",
                      })}
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

<style>
  .drawer-backdrop {
    position: fixed;
    inset: 0;
    background-color: var(--drawer-scrim);
    display: flex;
    justify-content: flex-end;
    z-index: 100;
    backdrop-filter: blur(2px);
    animation: fadeIn 150ms ease-out;
  }

  .drawer-panel {
    background-color: var(--surface);
    color: var(--text);
    width: var(--drawer-width);
    height: 100%;
    display: flex;
    flex-direction: column;
    box-shadow: -4px 0 24px rgba(0, 0, 0, 0.25);
    outline: none;
    animation: slideIn 200ms cubic-bezier(0.16, 1, 0.3, 1);
    overflow: hidden;
  }

  @keyframes fadeIn {
    from {
      opacity: 0;
    }
    to {
      opacity: 1;
    }
  }

  @keyframes slideIn {
    from {
      transform: translateX(100%);
    }
    to {
      transform: translateX(0);
    }
  }

  .drawer-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: var(--space-4) var(--space-6);
    border-bottom: var(--border-width) solid var(--border);
    background-color: var(--surface-nav);
    flex-shrink: 0;
  }

  .header-left {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  .human-key {
    font-family: var(--font-mono);
    font-weight: 700;
    color: var(--accent);
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    padding: var(--space-1) var(--space-2);
    border-radius: var(--radius);
    font-size: var(--text-small);
  }

  .project-title {
    font-size: var(--text-small);
    color: var(--text-muted);
    font-weight: 600;
  }

  .close-btn {
    background: transparent;
    border: var(--border-width) solid transparent;
    color: var(--text);
    padding: var(--space-2);
    border-radius: var(--radius);
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: background-color 120ms ease;
  }

  .close-btn:hover {
    background-color: var(--surface-raised);
    border-color: var(--border);
  }

  .close-btn:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
  }

  .drawer-feedback {
    padding: var(--space-2) var(--space-6);
  }

  .drawer-body {
    flex: 1;
    display: grid;
    grid-template-columns: 1fr 18rem;
    overflow-y: auto;
  }

  @media (max-width: 768px) {
    .drawer-body {
      grid-template-columns: 1fr;
    }
  }

  .drawer-main {
    padding: var(--space-6);
    border-right: var(--border-width) solid var(--border);
    display: flex;
    flex-direction: column;
    gap: var(--space-6);
  }

  .main-form {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .field-label {
    font-size: var(--text-small);
    font-weight: 650;
    color: var(--text);
  }

  .title-input {
    font-size: var(--text-lead);
    font-weight: 700;
    color: var(--text);
    background: var(--surface-raised);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius);
    padding: var(--space-2) var(--space-3);
  }

  .title-input:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
  }

  .desc-textarea {
    font-family: var(--font-body);
    font-size: var(--text-body);
    line-height: var(--leading);
    color: var(--text);
    background: var(--surface-raised);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius);
    padding: var(--space-3);
    resize: vertical;
  }

  .desc-textarea:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
  }

  .action-row {
    display: flex;
    justify-content: flex-end;
  }

  /* Section cards */
  .section-card {
    background: var(--surface-raised);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius);
    padding: var(--space-4);
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .section-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .section-title-wrap {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  .section-heading {
    margin: 0;
    font-size: var(--text-body);
    font-weight: 700;
    color: var(--text);
  }

  .progress-pill {
    font-size: var(--text-small);
    color: var(--text-muted);
    font-weight: 600;
  }

  .progress-bar-bg {
    width: 100%;
    height: 6px;
    background: var(--surface);
    border-radius: 9999px;
    overflow: hidden;
  }

  .progress-bar-fill {
    height: 100%;
    background: var(--accent);
    transition: width 200ms ease;
  }

  .subtask-list {
    list-style: none;
    padding: 0;
    margin: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .subtask-row {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-2);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius);
    background: var(--surface);
  }

  .subtask-row.completed {
    opacity: 0.65;
  }

  .subtask-row.completed .subtask-title {
    text-decoration: line-through;
  }

  .checkbox-wrap {
    display: flex;
    align-items: center;
    gap: var(--space-1);
    cursor: pointer;
  }

  .subtask-key {
    font-family: var(--font-mono);
    font-size: var(--text-small);
    font-weight: 650;
    color: var(--text-muted);
  }

  .subtask-title {
    flex: 1;
    font-size: var(--text-body);
    color: var(--text);
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
    background-color: var(--surface-nav);
    color: var(--text-secondary);
  }

  .add-subtask-form {
    display: flex;
    gap: var(--space-2);
    margin-top: var(--space-2);
  }

  .inline-input {
    flex: 1;
    background: var(--surface);
    color: var(--text);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius);
    padding: var(--space-2) var(--space-3);
    font-size: var(--text-body);
  }

  .inline-input:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
  }

  .empty-note {
    font-size: var(--text-small);
    color: var(--text-muted);
    font-style: italic;
    margin: var(--space-2) 0;
  }

  /* Comments */
  .comments-stream {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .comment-card {
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius);
    padding: var(--space-3);
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .comment-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .comment-author-badge {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .avatar-circle {
    width: var(--avatar-size);
    height: var(--avatar-size);
    border-radius: 9999px;
    background: var(--selected);
    color: var(--accent);
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 700;
    font-size: 0.75rem;
    border: var(--border-width) solid var(--border);
  }

  .author-name {
    font-weight: 700;
    font-size: var(--text-small);
    color: var(--text);
  }

  .author-handle {
    font-size: var(--text-small);
    color: var(--text-muted);
  }

  .comment-meta {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .comment-time {
    font-size: 0.75rem;
    color: var(--text-muted);
  }

  .delete-comment-btn {
    background: transparent;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    padding: var(--space-1);
    display: flex;
    align-items: center;
  }

  .delete-comment-btn:hover {
    color: var(--danger);
  }

  .comment-body {
    margin: 0;
    font-size: var(--text-body);
    line-height: var(--leading);
    color: var(--text);
    white-space: pre-wrap;
  }

  .post-comment-form {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    margin-top: var(--space-2);
  }

  .comment-textarea {
    font-family: var(--font-body);
    font-size: var(--text-body);
    color: var(--text);
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius);
    padding: var(--space-2) var(--space-3);
    resize: vertical;
  }

  .comment-textarea:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
  }

  /* Sidebar */
  .drawer-sidebar {
    padding: var(--space-6);
    background-color: var(--surface-nav);
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
    overflow-y: auto;
  }

  .meta-group {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .meta-header-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
  }

  .meta-label {
    font-size: var(--text-small);
    font-weight: 700;
    color: var(--text-secondary);
    text-transform: uppercase;
    letter-spacing: 0.04em;
  }

  .meta-select,
  .meta-input {
    background: var(--surface-raised);
    color: var(--text);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius);
    padding: var(--space-2) var(--space-3);
    font-size: var(--text-small);
  }

  .meta-select:focus-visible,
  .meta-input:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
  }

  .meta-divider {
    border: 0;
    border-top: var(--border-width) solid var(--border);
    margin: var(--space-1) 0;
  }

  .meta-empty {
    font-size: var(--text-small);
    color: var(--text-muted);
    font-style: italic;
    margin: 0;
  }

  /* Assignees chips */
  .assignee-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .assignee-chip {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    background: var(--surface-raised);
    border: var(--border-width) solid var(--border);
    padding: var(--space-1) var(--space-2);
    border-radius: var(--radius);
  }

  .avatar-small {
    width: 1.25rem;
    height: 1.25rem;
    border-radius: 9999px;
    background: var(--selected);
    color: var(--accent);
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.625rem;
    font-weight: 700;
  }

  .assignee-name {
    flex: 1;
    font-size: var(--text-small);
    font-weight: 600;
  }

  .chip-remove-btn {
    background: transparent;
    border: none;
    color: var(--text-muted);
    cursor: pointer;
    padding: var(--space-1);
    display: flex;
    align-items: center;
  }

  .chip-remove-btn:hover {
    color: var(--danger);
  }

  .picker-row {
    display: flex;
    gap: var(--space-2);
    margin-top: var(--space-1);
  }

  .picker-row .meta-select {
    flex: 1;
  }

  /* Labels */
  .label-badges {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-1);
  }

  .label-badge {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
    background: var(--surface-raised);
    border: var(--border-width) solid var(--border);
    border-left: 3px solid var(--badge-color);
    padding: 0.125rem 0.5rem;
    border-radius: var(--badge-radius);
    font-size: var(--text-small);
    font-weight: 600;
    color: var(--text);
  }

  .badge-dot {
    width: 6px;
    height: 6px;
    border-radius: 9999px;
    background-color: var(--badge-color);
  }

  .text-link-btn {
    background: transparent;
    border: none;
    color: var(--accent);
    font-size: var(--text-small);
    font-weight: 650;
    cursor: pointer;
    padding: 0;
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
  }

  .text-link-btn:hover {
    text-decoration: underline;
  }

  .mini-label-form {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    background: var(--surface-raised);
    border: var(--border-width) solid var(--border);
    padding: var(--space-2);
    border-radius: var(--radius);
  }

  .color-palette {
    display: flex;
    gap: var(--space-1);
  }

  .color-dot {
    width: 1.25rem;
    height: 1.25rem;
    border-radius: 9999px;
    border: 2px solid transparent;
    cursor: pointer;
    padding: 0;
  }

  .color-dot.selected {
    border-color: var(--text);
    transform: scale(1.15);
  }

  .mini-form-actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--space-2);
  }

  /* Timeline */
  .timeline {
    list-style: none;
    padding: 0;
    margin: 0;
    position: relative;
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .timeline-item {
    position: relative;
    padding-left: var(--space-4);
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
  }

  .timeline-marker {
    position: absolute;
    left: 0;
    top: 0.375rem;
    width: 6px;
    height: 6px;
    border-radius: 9999px;
    background-color: var(--accent);
  }

  .timeline-item::before {
    content: "";
    position: absolute;
    left: 2px;
    top: 0.75rem;
    bottom: -0.75rem;
    width: 2px;
    background-color: var(--timeline-line);
  }

  .timeline-item:last-child::before {
    display: none;
  }

  .timeline-action {
    font-size: 0.8125rem;
    color: var(--text);
    line-height: 1.35;
  }

  .timeline-date {
    font-size: 0.6875rem;
    color: var(--text-muted);
  }
</style>
