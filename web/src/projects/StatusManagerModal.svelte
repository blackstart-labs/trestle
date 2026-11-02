<script lang="ts">
  import Button from "../ui/Button.svelte";
  import Feedback from "../ui/Feedback.svelte";
  import Icon from "../ui/Icon.svelte";

  export interface TaskStatus {
    id: string;
    workspace_id: string;
    project_id: string;
    name: string;
    category: "backlog" | "unstarted" | "started" | "completed" | "cancelled";
    color: string;
    position: number;
    created_at: string;
    updated_at: string;
  }

  let {
    workspace,
    project,
    statuses = [],
    onclose,
    onstatusesupdated,
  }: {
    workspace: { id: string; name: string; slug: string; role: string };
    project: { id: string; name: string; key: string };
    statuses?: TaskStatus[];
    onclose: () => void;
    onstatusesupdated?: (statuses: TaskStatus[]) => void;
  } = $props();

  let localStatuses = $state<TaskStatus[]>([]);
  let loading = $state(false);
  let error = $state<string | null>(null);
  let success = $state<string | null>(null);

  // New status form state
  let showCreateForm = $state(false);
  let newName = $state("");
  let newCategory = $state<TaskStatus["category"]>("started");
  let newColor = $state("#315b3b");
  let creating = $state(false);
  let createError = $state<string | null>(null);

  // Edit status state
  let editingStatusId = $state<string | null>(null);
  let editName = $state("");
  let editColor = $state("");
  let savingEdit = $state(false);

  // Delete / Remap state
  let statusToDelete = $state<TaskStatus | null>(null);
  let remapTargetId = $state<string>("");
  let deleting = $state(false);
  let deleteError = $state<string | null>(null);

  const canManage = $derived(
    workspace.role === "owner" || workspace.role === "admin",
  );

  const presetColors = [
    { label: "Slate", value: "#4c544c" },
    { label: "Steel Blue", value: "#1e4d79" },
    { label: "Forest Green", value: "#26633b" },
    { label: "Sage", value: "#315b3b" },
    { label: "Amber Gold", value: "#775300" },
    { label: "Crimson", value: "#9f302b" },
    { label: "Indigo", value: "#6366f1" },
    { label: "Cyan", value: "#0891b2" },
    { label: "Purple", value: "#7c3aed" },
  ];

  const categoryOptions: {
    value: TaskStatus["category"];
    label: string;
    description: string;
  }[] = [
    {
      value: "backlog",
      label: "Backlog",
      description: "Ideas and work noted for future consideration",
    },
    {
      value: "unstarted",
      label: "Unstarted",
      description: "Triaged items ready to begin execution (e.g. To Do)",
    },
    {
      value: "started",
      label: "Started",
      description: "Active work in development, review, or QA",
    },
    {
      value: "completed",
      label: "Completed",
      description: "Work verified and shipped (terminal success state)",
    },
    {
      value: "cancelled",
      label: "Cancelled",
      description: "Dropped or duplicate items (terminal inactive state)",
    },
  ];

  // Initialize and keep localStatuses in sync
  $effect(() => {
    if (statuses && statuses.length > 0) {
      localStatuses = [...statuses].sort((a, b) => a.position - b.position);
    }
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

  export async function refreshStatuses() {
    loading = true;
    error = null;
    try {
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/projects/${project.id}/statuses`,
        { cache: "no-store" },
      );
      if (res.ok) {
        const data = await res.json();
        localStatuses = data.statuses || [];
        onstatusesupdated?.(localStatuses);
      } else {
        error = "Could not reload statuses.";
      }
    } catch {
      error = "Network error loading statuses.";
    } finally {
      loading = false;
    }
  }

  async function handleCreateStatus(e: Event) {
    e.preventDefault();
    createError = null;
    const trimmed = newName.trim();
    if (!trimmed) {
      createError = "Status name is required.";
      return;
    }

    const csrf = getCsrfToken();
    if (!csrf) {
      createError = "Security token missing. Please refresh.";
      return;
    }

    creating = true;
    try {
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/projects/${project.id}/statuses`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "x-csrf-token": csrf,
          },
          body: JSON.stringify({
            name: trimmed,
            category: newCategory,
            color: newColor,
          }),
        },
      );

      if (res.status === 201) {
        newName = "";
        newColor = "#315b3b";
        showCreateForm = false;
        success = "Status created successfully.";
        await refreshStatuses();
      } else if (res.status === 409) {
        createError = "A status with this name already exists in this project.";
      } else {
        const data = await res.json().catch(() => ({}));
        createError = data.error?.message || "Failed to create status.";
      }
    } catch {
      createError = "Network error creating status.";
    } finally {
      creating = false;
    }
  }

  function startEdit(status: TaskStatus) {
    editingStatusId = status.id;
    editName = status.name;
    editColor = status.color;
    error = null;
  }

  function cancelEdit() {
    editingStatusId = null;
    editName = "";
    editColor = "";
  }

  async function handleSaveEdit(statusId: string) {
    const trimmed = editName.trim();
    if (!trimmed) {
      error = "Status name cannot be empty.";
      return;
    }

    const csrf = getCsrfToken();
    if (!csrf) {
      error = "Security token missing. Please refresh.";
      return;
    }

    savingEdit = true;
    error = null;
    try {
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/projects/${project.id}/statuses/${statusId}`,
        {
          method: "PATCH",
          headers: {
            "Content-Type": "application/json",
            "x-csrf-token": csrf,
          },
          body: JSON.stringify({
            name: trimmed,
            color: editColor,
          }),
        },
      );

      if (res.ok) {
        editingStatusId = null;
        success = "Status updated.";
        await refreshStatuses();
      } else if (res.status === 409) {
        error = "Another status already uses this name.";
      } else {
        const data = await res.json().catch(() => ({}));
        error = data.error?.message || "Could not update status.";
      }
    } catch {
      error = "Network error updating status.";
    } finally {
      savingEdit = false;
    }
  }

  async function handleMove(index: number, direction: "up" | "down") {
    if (!canManage) return;
    const targetIndex = direction === "up" ? index - 1 : index + 1;
    if (targetIndex < 0 || targetIndex >= localStatuses.length) return;

    const newOrder = [...localStatuses];
    const [moved] = newOrder.splice(index, 1);
    newOrder.splice(targetIndex, 0, moved);

    // Optimistic local update
    localStatuses = newOrder;

    const csrf = getCsrfToken();
    if (!csrf) return;

    try {
      const statusIds = newOrder.map((s) => s.id);
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/projects/${project.id}/statuses/positions`,
        {
          method: "PUT",
          headers: {
            "Content-Type": "application/json",
            "x-csrf-token": csrf,
          },
          body: JSON.stringify({ status_ids: statusIds }),
        },
      );

      if (res.ok) {
        onstatusesupdated?.(localStatuses);
      } else {
        // Rollback
        await refreshStatuses();
      }
    } catch {
      await refreshStatuses();
    }
  }

  function promptDelete(status: TaskStatus) {
    if (localStatuses.length <= 1) {
      error = "Cannot delete the only status in the project.";
      return;
    }
    statusToDelete = status;
    deleteError = null;
    // Pre-select first valid alternative status
    const alt = localStatuses.find((s) => s.id !== status.id);
    remapTargetId = alt ? alt.id : "";
  }

  function cancelDelete() {
    statusToDelete = null;
    deleteError = null;
    remapTargetId = "";
  }

  async function handleConfirmDelete() {
    if (!statusToDelete) return;

    const csrf = getCsrfToken();
    if (!csrf) {
      deleteError = "Security token missing. Please refresh.";
      return;
    }

    deleting = true;
    deleteError = null;

    let url = `/api/v1/workspaces/${workspace.id}/projects/${project.id}/statuses/${statusToDelete.id}`;
    if (remapTargetId) {
      url += `?remap_to=${encodeURIComponent(remapTargetId)}`;
    }

    try {
      const res = await fetch(url, {
        method: "DELETE",
        headers: {
          "x-csrf-token": csrf,
        },
      });

      if (res.status === 204) {
        statusToDelete = null;
        success = "Status removed and tasks safely remapped.";
        await refreshStatuses();
      } else if (res.status === 409) {
        deleteError =
          "Cannot delete status: select a valid replacement status for existing tasks.";
      } else {
        const data = await res.json().catch(() => ({}));
        deleteError = data.error?.message || "Failed to delete status.";
      }
    } catch {
      deleteError = "Network error during status deletion.";
    } finally {
      deleting = false;
    }
  }

  function getCategoryLabel(cat: string): string {
    const found = categoryOptions.find((c) => c.value === cat);
    return found ? found.label : cat;
  }
</script>

<div
  class="modal-backdrop"
  role="presentation"
  onclick={(e) => {
    if (e.target === e.currentTarget) onclose();
  }}
>
  <div
    class="modal-content"
    role="dialog"
    aria-modal="true"
    aria-labelledby="status-manager-title"
  >
    <div class="modal-header">
      <div>
        <h2 id="status-manager-title" class="modal-title">
          Workflow statuses & transitions
        </h2>
        <p class="modal-subtitle">
          Configure custom project statuses while preserving canonical category
          completion semantics for {project.name}.
        </p>
      </div>
      <button
        type="button"
        class="icon-button close-btn"
        onclick={onclose}
        aria-label="Close status manager"
      >
        <Icon name="close" size={18} />
      </button>
    </div>

    {#if error}
      <div class="message-banner">
        <Feedback tone="danger">{error}</Feedback>
      </div>
    {/if}

    {#if success}
      <div class="message-banner">
        <Feedback tone="success">{success}</Feedback>
      </div>
    {/if}

    <!-- Statuses List -->
    <div class="status-list-section">
      <div class="section-toolbar">
        <span class="count-label"
          >{localStatuses.length}
          {localStatuses.length === 1 ? "status" : "statuses"} configured</span
        >
        {#if canManage && !showCreateForm}
          <button
            type="button"
            class="btn btn-secondary"
            onclick={() => {
              showCreateForm = true;
              error = null;
              success = null;
            }}
          >
            <Icon name="plus" size={14} class="btn-icon" />
            Add custom status
          </button>
        {/if}
      </div>

      <!-- Add New Status Drawer / Form -->
      {#if showCreateForm}
        <form
          class="create-status-card"
          onsubmit={handleCreateStatus}
          aria-label="Add custom status form"
        >
          <div class="card-header">
            <span class="card-title">New project status</span>
            <button
              type="button"
              class="icon-button"
              onclick={() => {
                showCreateForm = false;
                createError = null;
              }}
              aria-label="Cancel adding status"
            >
              <Icon name="close" size={16} />
            </button>
          </div>

          {#if createError}
            <Feedback tone="danger">{createError}</Feedback>
          {/if}

          <div class="form-grid">
            <div class="form-group">
              <label for="new-status-name" class="field-label"
                >Status name</label
              >
              <input
                id="new-status-name"
                type="text"
                class="text-input"
                bind:value={newName}
                placeholder="e.g. In Review, QA Verification, Blocked"
                maxlength="50"
                required
              />
            </div>

            <div class="form-group">
              <label for="new-status-category" class="field-label"
                >Canonical category</label
              >
              <select
                id="new-status-category"
                class="select-input"
                bind:value={newCategory}
              >
                {#each categoryOptions as opt}
                  <option value={opt.value}
                    >{opt.label} — {opt.description}</option
                  >
                {/each}
              </select>
            </div>
          </div>

          <div class="color-picker-group">
            <span class="field-label">Status color</span>
            <div class="preset-palette">
              {#each presetColors as p}
                <button
                  type="button"
                  class="color-chip"
                  class:selected={newColor === p.value}
                  style="background-color: {p.value};"
                  onclick={() => (newColor = p.value)}
                  title={p.label}
                  aria-label={p.label}
                ></button>
              {/each}
              <div class="custom-color-wrap">
                <input
                  type="color"
                  bind:value={newColor}
                  class="native-color-picker"
                  aria-label="Choose custom status color"
                />
                <span class="hex-text">{newColor}</span>
              </div>
            </div>
          </div>

          <div class="form-actions">
            <button
              type="button"
              class="btn btn-secondary"
              onclick={() => {
                showCreateForm = false;
                createError = null;
              }}
            >
              Cancel
            </button>
            <button type="submit" class="btn btn-primary" disabled={creating}>
              {creating ? "Saving…" : "Save status"}
            </button>
          </div>
        </form>
      {/if}

      <!-- Status Items Table / List -->
      <div class="statuses-container" role="list">
        {#each localStatuses as st, idx (st.id)}
          <div
            class="status-row"
            class:is-editing={editingStatusId === st.id}
            role="listitem"
          >
            {#if editingStatusId === st.id}
              <!-- Inline edit row -->
              <div class="inline-edit-form">
                <div class="edit-inputs">
                  <input
                    type="text"
                    class="text-input edit-name-input"
                    bind:value={editName}
                    maxlength="50"
                    aria-label="Edit status name"
                  />
                  <div class="color-swatch-picker">
                    <input
                      type="color"
                      bind:value={editColor}
                      class="native-color-picker"
                      aria-label="Edit status color"
                    />
                    <span class="hex-text">{editColor}</span>
                  </div>
                </div>
                <div class="edit-actions">
                  <button
                    type="button"
                    class="btn btn-secondary"
                    onclick={cancelEdit}
                  >
                    Cancel
                  </button>
                  <button
                    type="button"
                    class="btn btn-primary"
                    disabled={savingEdit}
                    onclick={() => handleSaveEdit(st.id)}
                  >
                    {savingEdit ? "Saving…" : "Save"}
                  </button>
                </div>
              </div>
            {:else}
              <!-- Normal row display -->
              <div class="row-lead">
                <span
                  class="status-color-indicator"
                  style="background-color: {st.color};"
                  aria-hidden="true"
                ></span>
                <span class="status-name">{st.name}</span>
                <span class="category-pill category-{st.category}">
                  {getCategoryLabel(st.category)}
                </span>
              </div>

              {#if canManage}
                <div class="row-controls">
                  <!-- Reorder buttons -->
                  <div class="reorder-group" role="group" aria-label="Reorder">
                    <button
                      type="button"
                      class="icon-button reorder-btn"
                      disabled={idx === 0}
                      onclick={() => handleMove(idx, "up")}
                      aria-label="Move {st.name} up"
                    >
                      <Icon name="arrow-up" size={14} />
                    </button>
                    <button
                      type="button"
                      class="icon-button reorder-btn"
                      disabled={idx === localStatuses.length - 1}
                      onclick={() => handleMove(idx, "down")}
                      aria-label="Move {st.name} down"
                    >
                      <Icon name="arrow-down" size={14} />
                    </button>
                  </div>

                  <!-- Edit button -->
                  <button
                    type="button"
                    class="icon-button edit-btn"
                    onclick={() => startEdit(st)}
                    aria-label="Edit {st.name}"
                  >
                    <Icon name="pencil" size={14} />
                  </button>

                  <!-- Delete button -->
                  <button
                    type="button"
                    class="icon-button delete-btn"
                    disabled={localStatuses.length <= 1}
                    onclick={() => promptDelete(st)}
                    aria-label="Delete {st.name}"
                  >
                    <Icon name="trash" size={14} />
                  </button>
                </div>
              {/if}
            {/if}
          </div>
        {/each}
      </div>
    </div>

    <!-- Safe Deletion and Remapping Dialog -->
    {#if statusToDelete}
      <div
        class="delete-confirm-overlay"
        role="presentation"
        onclick={(e) => {
          if (e.target === e.currentTarget) cancelDelete();
        }}
      >
        <div
          class="delete-confirm-box"
          role="dialog"
          aria-modal="true"
          aria-labelledby="delete-dialog-title"
        >
          <div class="confirm-header">
            <span class="warning-icon-badge">
              <Icon name="alert-triangle" size={20} />
            </span>
            <div>
              <h3 id="delete-dialog-title" class="confirm-title">
                Delete "{statusToDelete.name}" status
              </h3>
              <p class="confirm-desc">
                Tasks currently assigned to this status must be safely remapped
                so they are never left in an orphaned state.
              </p>
            </div>
          </div>

          {#if deleteError}
            <Feedback tone="danger">{deleteError}</Feedback>
          {/if}

          <div class="remap-select-group">
            <label for="remap-target-select" class="field-label">
              Reassign active tasks to:
            </label>
            <select
              id="remap-target-select"
              class="select-input"
              bind:value={remapTargetId}
            >
              {#each localStatuses.filter((s) => s.id !== statusToDelete?.id) as target}
                <option value={target.id}>
                  {target.name} ({getCategoryLabel(target.category)})
                </option>
              {/each}
            </select>
          </div>

          <div class="confirm-actions">
            <button
              type="button"
              class="btn btn-secondary"
              disabled={deleting}
              onclick={cancelDelete}
            >
              Cancel
            </button>
            <button
              type="button"
              class="btn btn-danger"
              disabled={deleting}
              onclick={handleConfirmDelete}
            >
              {deleting ? "Reassigning…" : "Reassign & Delete"}
            </button>
          </div>
        </div>
      </div>
    {/if}

    <div class="modal-footer">
      <button type="button" class="btn btn-secondary" onclick={onclose}>
        Done
      </button>
    </div>
  </div>
</div>

<style>
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: var(--drawer-scrim, rgba(0, 0, 0, 0.5));
    backdrop-filter: blur(2px);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1000;
    padding: var(--space-4);
  }

  .modal-content {
    background: var(--surface-raised);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius);
    width: 100%;
    max-width: 42rem;
    max-height: 90vh;
    overflow-y: auto;
    box-shadow: 0 10px 25px rgba(0, 0, 0, 0.2);
    display: flex;
    flex-direction: column;
  }

  .modal-header {
    padding: var(--space-4) var(--space-6);
    border-bottom: var(--border-width) solid var(--border);
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    gap: var(--space-4);
  }

  .modal-title {
    font-size: 1.125rem;
    font-weight: 600;
    color: var(--text);
    margin: 0 0 var(--space-1) 0;
  }

  .modal-subtitle {
    font-size: var(--text-small);
    color: var(--text-muted);
    margin: 0;
    line-height: 1.4;
  }

  .message-banner {
    padding: var(--space-2) var(--space-6);
  }

  .status-list-section {
    padding: var(--space-4) var(--space-6);
    flex: 1;
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }

  .section-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2);
  }

  .count-label {
    font-size: var(--text-small);
    color: var(--text-secondary);
    font-weight: 500;
  }

  :global(.btn-icon) {
    margin-right: var(--space-1);
    vertical-align: middle;
  }

  /* Buttons */
  .btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    padding: 0.375rem 0.75rem;
    font-size: var(--text-small);
    font-weight: 600;
    border-radius: var(--radius);
    border: var(--border-width) solid transparent;
    cursor: pointer;
    font-family: inherit;
    transition:
      background-color 0.15s ease,
      border-color 0.15s ease,
      color 0.15s ease;
  }

  .btn:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: 1px;
  }

  .btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-primary {
    background: var(--button-bg);
    color: var(--button-text);
  }

  .btn-primary:hover:not(:disabled) {
    opacity: 0.9;
  }

  .btn-secondary {
    background: var(--surface);
    border-color: var(--border);
    color: var(--text);
  }

  .btn-secondary:hover:not(:disabled) {
    background: var(--surface-nav);
  }

  .btn-danger {
    background: var(--danger);
    color: #ffffff;
  }

  .btn-danger:hover:not(:disabled) {
    opacity: 0.9;
  }

  /* Create form */
  .create-status-card {
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius);
    padding: var(--space-4);
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .card-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .card-title {
    font-size: var(--text-small);
    font-weight: 600;
    color: var(--text);
  }

  .form-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: var(--space-3);
  }

  @media (max-width: 600px) {
    .form-grid {
      grid-template-columns: 1fr;
    }
  }

  .form-group {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .field-label {
    font-size: var(--text-small);
    font-weight: 500;
    color: var(--text-secondary);
  }

  .text-input,
  .select-input {
    width: 100%;
    height: 2.25rem;
    padding: 0 var(--space-2);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius);
    background: var(--surface-raised);
    color: var(--text);
    font-family: inherit;
    font-size: var(--text-small);
  }

  .text-input:focus,
  .select-input:focus {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: 1px;
  }

  .color-picker-group {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .preset-palette {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: var(--space-2);
  }

  .color-chip {
    width: 1.5rem;
    height: 1.5rem;
    border-radius: var(--badge-radius);
    border: 2px solid transparent;
    cursor: pointer;
    transition:
      transform 0.15s ease,
      border-color 0.15s ease;
  }

  .color-chip:hover {
    transform: scale(1.15);
  }

  .color-chip.selected {
    border-color: var(--text);
    box-shadow: 0 0 0 2px var(--surface-raised);
  }

  .custom-color-wrap {
    display: flex;
    align-items: center;
    gap: var(--space-1);
    margin-left: var(--space-2);
  }

  .native-color-picker {
    width: 1.75rem;
    height: 1.75rem;
    border: none;
    padding: 0;
    background: transparent;
    cursor: pointer;
    border-radius: var(--radius);
  }

  .hex-text {
    font-size: 0.75rem;
    font-family: var(--font-mono);
    color: var(--text-muted);
  }

  .form-actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--space-2);
    margin-top: var(--space-1);
  }

  /* Status List */
  .statuses-container {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }

  .status-row {
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius);
    padding: var(--space-2) var(--space-3);
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    transition: background 0.15s ease;
  }

  .status-row:hover {
    background: var(--surface-nav);
  }

  .row-lead {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    flex: 1;
    min-width: 0;
  }

  .status-color-indicator {
    width: 0.75rem;
    height: 0.75rem;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .status-name {
    font-weight: 500;
    color: var(--text);
    font-size: var(--text-small);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .category-pill {
    font-size: 0.75rem;
    font-weight: 500;
    padding: 0.125rem 0.5rem;
    border-radius: var(--badge-radius);
    background: var(--surface-raised);
    border: var(--border-width) solid var(--border);
    color: var(--text-secondary);
    text-transform: capitalize;
  }

  .category-backlog {
    border-color: #4c544c;
    color: var(--text-secondary);
  }

  .category-unstarted {
    border-color: #1e4d79;
    color: #1e4d79;
  }

  .category-started {
    border-color: #775300;
    color: #775300;
  }

  .category-completed {
    border-color: #26633b;
    color: #26633b;
  }

  .category-cancelled {
    border-color: #9f302b;
    color: #9f302b;
  }

  .row-controls {
    display: flex;
    align-items: center;
    gap: var(--space-1);
  }

  .reorder-group {
    display: flex;
    gap: 2px;
  }

  .icon-button {
    background: transparent;
    border: none;
    color: var(--text-secondary);
    cursor: pointer;
    padding: var(--space-1);
    border-radius: var(--radius);
    display: flex;
    align-items: center;
    justify-content: center;
    transition:
      color 0.15s ease,
      background 0.15s ease;
  }

  .icon-button:hover:not(:disabled) {
    color: var(--text);
    background: var(--selected);
  }

  .icon-button:disabled {
    opacity: 0.35;
    cursor: not-allowed;
  }

  .delete-btn:hover:not(:disabled) {
    color: var(--danger);
  }

  /* Inline edit */
  .inline-edit-form {
    display: flex;
    align-items: center;
    justify-content: space-between;
    width: 100%;
    gap: var(--space-2);
  }

  .edit-inputs {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    flex: 1;
  }

  .edit-name-input {
    flex: 1;
  }

  .color-swatch-picker {
    display: flex;
    align-items: center;
    gap: var(--space-1);
  }

  .edit-actions {
    display: flex;
    align-items: center;
    gap: var(--space-1);
  }

  /* Delete Confirm Modal Overlay */
  .delete-confirm-overlay {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.6);
    display: flex;
    align-items: center;
    justify-content: center;
    z-index: 1100;
    padding: var(--space-4);
  }

  .delete-confirm-box {
    background: var(--surface-raised);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius);
    width: 100%;
    max-width: 28rem;
    padding: var(--space-4);
    box-shadow: 0 12px 30px rgba(0, 0, 0, 0.25);
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .confirm-header {
    display: flex;
    align-items: flex-start;
    gap: var(--space-3);
  }

  .warning-icon-badge {
    color: var(--warning);
    background: var(--surface);
    padding: var(--space-2);
    border-radius: 50%;
    border: var(--border-width) solid var(--border);
    flex-shrink: 0;
  }

  .confirm-title {
    font-size: 1rem;
    font-weight: 600;
    color: var(--text);
    margin: 0 0 var(--space-1) 0;
  }

  .confirm-desc {
    font-size: var(--text-small);
    color: var(--text-secondary);
    margin: 0;
    line-height: 1.4;
  }

  .remap-select-group {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .confirm-actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--space-2);
    margin-top: var(--space-2);
  }

  .modal-footer {
    padding: var(--space-3) var(--space-6);
    border-top: var(--border-width) solid var(--border);
    display: flex;
    justify-content: flex-end;
    background: var(--surface);
  }
</style>
