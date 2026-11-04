<script lang="ts">
  import { onMount } from "svelte";
  import Feedback from "../ui/Feedback.svelte";
  import Icon from "../ui/Icon.svelte";
  import Select from "../ui/Select.svelte";
  import { dialogIn, dialogOut, fadeIn, sheet } from "../lib/motion";
  import { flip } from "svelte/animate";

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

  let dialogElement = $state<HTMLElement | null>(null);

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

  onMount(() => {
    const opener = document.activeElement as HTMLElement | null;
    dialogElement?.focus();
    return () => {
      if (opener && document.contains(opener)) opener.focus();
    };
  });
</script>

<div
  class="scrim"
  role="presentation"
  in:fadeIn|global={{ duration: 200 }}
  out:fadeIn|global={{ duration: 160 }}
  onclick={(e) => {
    if (e.target === e.currentTarget) onclose();
  }}
>
  <div
    class="dialog dialog-wide status-dialog"
    role="dialog"
    aria-modal="true"
    aria-labelledby="status-manager-title"
    tabindex="-1"
    bind:this={dialogElement}
    in:dialogIn|global
    out:dialogOut|global
    onkeydown={(e) => {
      if (e.key === "Escape" && !statusToDelete) {
        e.stopPropagation();
        onclose();
      }
    }}
  >
    <div class="dialog-head">
      <div>
        <h2 id="status-manager-title">Workflow statuses & transitions</h2>
        <p>
          Configure custom project statuses while preserving canonical category
          completion semantics for {project.name}.
        </p>
      </div>
      <button
        type="button"
        class="btn-quiet btn-icon"
        onclick={onclose}
        aria-label="Close status manager"
      >
        <Icon name="close" size={18} />
      </button>
    </div>

    <div class="dialog-body">
      {#if error}
        <Feedback tone="danger">{error}</Feedback>
      {/if}

      {#if success}
        <Feedback tone="success">{success}</Feedback>
      {/if}

      <div class="section-toolbar">
        <span class="count-label num"
          >{localStatuses.length}
          {localStatuses.length === 1 ? "status" : "statuses"} configured</span
        >
        {#if canManage && !showCreateForm}
          <button
            type="button"
            onclick={() => {
              showCreateForm = true;
              error = null;
              success = null;
            }}
          >
            <Icon name="plus" size={16} />
            Add custom status
          </button>
        {/if}
      </div>

      {#if showCreateForm}
        <form
          class="create-card"
          onsubmit={handleCreateStatus}
          aria-label="Add custom status form"
        >
          <div class="create-head">
            <span class="create-title">New project status</span>
            <button
              type="button"
              class="btn-quiet btn-icon"
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

          <div class="form-row">
            <div class="field">
              <label for="new-status-name">Status name</label>
              <input
                id="new-status-name"
                type="text"
                bind:value={newName}
                placeholder="e.g. In Review, QA, Blocked"
                maxlength="50"
                required
              />
            </div>

            <div class="field">
              <label for="new-status-category">Canonical category</label>
              <Select
                id="new-status-category"
                value={newCategory}
                options={categoryOptions.map((o) => ({
                  value: o.value,
                  label: o.label,
                  description: o.description,
                  color: `var(--cat-${o.value})`,
                }))}
                onchange={(v) => (newCategory = v as TaskStatus["category"])}
              />
              <p class="note">
                {categoryOptions.find((c) => c.value === newCategory)
                  ?.description}
              </p>
            </div>
          </div>

          <div class="field">
            <span class="field-label">Status color</span>
            <div class="preset-palette">
              {#each presetColors as p}
                <button
                  type="button"
                  class="color-chip"
                  class:selected={newColor === p.value}
                  style:--chip-color={p.value}
                  onclick={() => (newColor = p.value)}
                  title={p.label}
                  aria-label={p.label}
                  aria-pressed={newColor === p.value}
                ></button>
              {/each}
              <label class="custom-color">
                <input
                  type="color"
                  bind:value={newColor}
                  aria-label="Choose custom status color"
                />
                <span class="hex-text">{newColor}</span>
              </label>
            </div>
          </div>

          <div class="dialog-actions">
            <button
              type="button"
              class="btn-quiet"
              onclick={() => {
                showCreateForm = false;
                createError = null;
              }}
            >
              Cancel
            </button>
            <button type="submit" class="btn-primary" disabled={creating}>
              {creating ? "Saving…" : "Save status"}
            </button>
          </div>
        </form>
      {/if}

      <div class="status-list" role="list">
        {#each localStatuses as st, idx (st.id)}
          <div
            class="status-row"
            class:is-editing={editingStatusId === st.id}
            role="listitem"
            animate:flip={{ duration: 320, easing: sheet }}
          >
            {#if editingStatusId === st.id}
              <div class="inline-edit">
                <input
                  type="text"
                  bind:value={editName}
                  maxlength="50"
                  aria-label="Edit status name"
                />
                <label class="custom-color">
                  <input
                    type="color"
                    bind:value={editColor}
                    aria-label="Edit status color"
                  />
                  <span class="hex-text">{editColor}</span>
                </label>
                <div class="edit-actions">
                  <button type="button" class="btn-quiet" onclick={cancelEdit}>
                    Cancel
                  </button>
                  <button
                    type="button"
                    class="btn-primary"
                    disabled={savingEdit}
                    onclick={() => handleSaveEdit(st.id)}
                  >
                    {savingEdit ? "Saving…" : "Save"}
                  </button>
                </div>
              </div>
            {:else}
              <div class="row-lead">
                <span
                  class="swatch"
                  style:--chip-color={st.color}
                  aria-hidden="true"
                ></span>
                <span class="status-name">{st.name}</span>
                <span
                  class="pill category-pill"
                  style:--cat-color="var(--cat-{st.category})"
                >
                  <span class="dot" aria-hidden="true"></span>
                  {getCategoryLabel(st.category)}
                </span>
              </div>

              {#if canManage}
                <div class="row-controls">
                  <div class="reorder" role="group" aria-label="Reorder">
                    <button
                      type="button"
                      class="btn-quiet btn-icon"
                      disabled={idx === 0}
                      onclick={() => handleMove(idx, "up")}
                      aria-label="Move {st.name} up"
                    >
                      <Icon name="arrow-up" size={16} />
                    </button>
                    <button
                      type="button"
                      class="btn-quiet btn-icon"
                      disabled={idx === localStatuses.length - 1}
                      onclick={() => handleMove(idx, "down")}
                      aria-label="Move {st.name} down"
                    >
                      <Icon name="arrow-down" size={16} />
                    </button>
                  </div>
                  <button
                    type="button"
                    class="btn-quiet btn-icon"
                    onclick={() => startEdit(st)}
                    aria-label="Edit {st.name}"
                  >
                    <Icon name="pencil" size={16} />
                  </button>
                  <button
                    type="button"
                    class="btn-quiet btn-icon delete-btn"
                    disabled={localStatuses.length <= 1}
                    onclick={() => promptDelete(st)}
                    aria-label="Delete {st.name}"
                  >
                    <Icon name="trash" size={16} />
                  </button>
                </div>
              {/if}
            {/if}
          </div>
        {/each}
      </div>

      <div class="dialog-actions">
        <button type="button" onclick={onclose}>Done</button>
      </div>
    </div>
  </div>
</div>

{#if statusToDelete}
  <div
    class="scrim confirm-scrim"
    role="presentation"
    onclick={(e) => {
      if (e.target === e.currentTarget) cancelDelete();
    }}
  >
    <div
      class="dialog confirm-dialog"
      role="dialog"
      aria-modal="true"
      aria-labelledby="delete-dialog-title"
      tabindex="-1"
      onkeydown={(e) => {
        if (e.key === "Escape") {
          e.stopPropagation();
          cancelDelete();
        }
      }}
    >
      <div class="dialog-head">
        <div class="confirm-head">
          <span class="warning-badge" aria-hidden="true">
            <Icon name="alert-triangle" size={20} />
          </span>
          <div>
            <h3 id="delete-dialog-title">
              Delete "{statusToDelete.name}" status
            </h3>
            <p>
              Tasks currently assigned to this status must be safely remapped so
              they are never left in an orphaned state.
            </p>
          </div>
        </div>
      </div>

      <div class="dialog-body">
        {#if deleteError}
          <Feedback tone="danger">{deleteError}</Feedback>
        {/if}

        <div class="field">
          <label for="remap-target-select">Reassign active tasks to:</label>
          <Select
            id="remap-target-select"
            value={remapTargetId}
            options={localStatuses
              .filter((s) => s.id !== statusToDelete?.id)
              .map((t) => ({
                value: t.id,
                label: t.name,
                hint: getCategoryLabel(t.category),
                color: t.color,
              }))}
            onchange={(v) => (remapTargetId = v)}
          />
        </div>

        <div class="dialog-actions">
          <button
            type="button"
            class="btn-quiet"
            disabled={deleting}
            onclick={cancelDelete}
          >
            Cancel
          </button>
          <button
            type="button"
            class="btn-danger"
            disabled={deleting}
            onclick={handleConfirmDelete}
          >
            {deleting ? "Reassigning…" : "Reassign & Delete"}
          </button>
        </div>
      </div>
    </div>
  </div>
{/if}

<style>
  .status-dialog:focus,
  .confirm-dialog:focus {
    outline: none;
  }

  .section-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    flex-wrap: wrap;
  }

  .count-label {
    font-size: var(--text-small);
    font-weight: 560;
    color: var(--text-muted);
  }

  .create-card {
    background:
      var(--inset-fill) padding-box,
      var(--inset-edge) border-box;
    border: var(--border-width) solid transparent;
    display: grid;
    gap: var(--space-4);
    padding: var(--space-4);
    border-radius: var(--radius-lg);
  }

  .create-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .create-title {
    font-weight: 620;
    color: var(--text);
  }

  .preset-palette {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-2);
  }

  .color-chip {
    width: 2.25rem;
    min-height: 2.25rem;
    padding: 0;
    border-radius: 50%;
    border: 3px solid var(--surface-sunken);
    background: var(--chip-color);
    box-shadow: 0 0 0 1px var(--line-strong);
  }

  .color-chip:hover:not(:disabled) {
    background: var(--chip-color);
    border-color: var(--surface-sunken);
  }

  .color-chip.selected {
    box-shadow: 0 0 0 2px var(--text);
  }

  .custom-color {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    min-height: var(--control-height);
    padding: 0 var(--space-3) 0 var(--space-1);
    border-radius: var(--radius-full);
    border: var(--border-width) solid var(--line-strong);
    background: var(--surface-raised);
    cursor: pointer;
  }

  .custom-color input {
    width: 2rem;
    height: 2rem;
    padding: 0;
    border: 0;
    border-radius: 50%;
    background: none;
    cursor: pointer;
  }

  .hex-text {
    font-size: var(--text-xs);
    font-variant-numeric: tabular-nums;
    color: var(--text-muted);
    text-transform: uppercase;
  }

  .status-list {
    display: grid;
    border: var(--border-width) solid var(--line);
    border-radius: var(--radius-lg);
    overflow: hidden;
  }

  .status-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    min-height: 3.5rem;
    padding: var(--space-1) var(--space-2) var(--space-1) var(--space-4);
    border-bottom: var(--border-width) solid var(--line);
    background: var(--surface-raised);
    flex-wrap: wrap;
  }

  .status-row:last-child {
    border-bottom: 0;
  }

  .status-row.is-editing {
    background: var(--surface-sunken);
    padding: var(--space-3) var(--space-4);
  }

  .row-lead {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    min-width: 0;
    flex-wrap: wrap;
  }

  .swatch {
    width: 0.875rem;
    height: 0.875rem;
    border-radius: 0.25rem;
    background: var(--chip-color);
    flex-shrink: 0;
  }

  .status-name {
    font-weight: 580;
    color: var(--text);
  }

  .category-pill {
    color: var(--cat-color);
  }

  .row-controls,
  .reorder {
    display: flex;
    align-items: center;
    gap: 2px;
  }

  .reorder {
    padding-right: var(--space-1);
    margin-right: var(--space-1);
    border-right: var(--border-width) solid var(--line);
  }

  .delete-btn:hover:not(:disabled) {
    color: var(--danger);
    background: var(--danger-soft);
  }

  .inline-edit {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    flex-wrap: wrap;
    width: 100%;
  }

  .inline-edit > input {
    flex: 1 1 12rem;
    width: auto;
  }

  .edit-actions {
    display: flex;
    gap: var(--space-2);
    margin-left: auto;
  }

  .confirm-scrim {
    z-index: calc(var(--layer-overlay) + 10);
  }

  .confirm-head {
    display: flex;
    gap: var(--space-4);
  }

  .confirm-head h3 {
    font-size: 1.125rem;
  }

  .warning-badge {
    display: grid;
    place-items: center;
    width: 2.75rem;
    height: 2.75rem;
    border-radius: var(--radius);
    background: var(--danger-soft);
    color: var(--danger);
    flex-shrink: 0;
  }
</style>
