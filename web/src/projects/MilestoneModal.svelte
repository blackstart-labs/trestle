<script lang="ts">
  import { onMount } from "svelte";
  import Modal from "../ui/Modal.svelte";
  import Button from "../ui/Button.svelte";
  import InputField from "../ui/InputField.svelte";
  import DatePicker from "../ui/DatePicker.svelte";
  import Select from "../ui/Select.svelte";
  import Feedback from "../ui/Feedback.svelte";
  import Icon from "../ui/Icon.svelte";
  import { formatDueDate } from "../lib/identity";

  export interface Milestone {
    id: string;
    workspace_id: string;
    project_id: string;
    title: string;
    description: string;
    start_date?: string | null;
    due_date: string | null;
    end_date?: string | null;
    color: string;
    status: "open" | "closed";
    total_tasks: number;
    completed_tasks: number;
    open_tasks: number;
    created_at: string;
    updated_at: string;
  }

  const presetColors = [
    { label: "Indigo", value: "#6366f1" },
    { label: "Steel Blue", value: "#1e4d79" },
    { label: "Cyan", value: "#0891b2" },
    { label: "Forest Green", value: "#26633b" },
    { label: "Sage", value: "#315b3b" },
    { label: "Amber Gold", value: "#775300" },
    { label: "Crimson", value: "#9f302b" },
    { label: "Purple", value: "#7c3aed" },
    { label: "Slate", value: "#4c544c" },
  ];

  let {
    workspace,
    project,
    initialMilestoneId = null,
    onclose,
    onmilestonesupdated,
  }: {
    workspace: { id: string; name: string; role: string };
    project: { id: string; name: string; key: string };
    initialMilestoneId?: string | null;
    onclose: () => void;
    onmilestonesupdated?: () => void;
  } = $props();

  let milestones = $state<Milestone[]>([]);
  let loading = $state(true);
  let error = $state<string | null>(null);

  // Form state
  let isEditing = $state(false);
  let editingId = $state<string | null>(null);
  let formTitle = $state("");
  let formDescription = $state("");
  let formStartDate = $state("");
  let formDueDate = $state("");
  let formColor = $state("#6366f1");
  let formStatus = $state<"open" | "closed">("open");
  let saving = $state(false);
  let formError = $state<string | null>(null);

  // Delete state
  let deletingMilestone = $state<Milestone | null>(null);
  let deleting = $state(false);

  function getCsrfToken(): string | null {
    if (typeof document === "undefined") return null;
    for (const part of document.cookie.split(";")) {
      const [k, v] = part.trim().split("=");
      if (k === "trestle-csrf") return decodeURIComponent(v ?? "");
    }
    return null;
  }

  async function loadMilestones() {
    loading = true;
    error = null;
    try {
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/projects/${project.id}/milestones`,
        { credentials: "same-origin" },
      );
      if (!res.ok) {
        error = "Failed to load project milestones.";
        return;
      }
      const data = await res.json();
      milestones = data.milestones || [];
      if (initialMilestoneId) {
        const found = milestones.find((m) => m.id === initialMilestoneId);
        if (found) {
          startEdit(found);
        }
      }
    } catch {
      error = "Network error loading milestones.";
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    void loadMilestones();
  });

  function startCreate() {
    isEditing = true;
    editingId = null;
    formTitle = "";
    formDescription = "";
    formStartDate = "";
    formDueDate = "";
    formColor = "#6366f1";
    formStatus = "open";
    formError = null;
  }

  function startEdit(m: Milestone) {
    isEditing = true;
    editingId = m.id;
    formTitle = m.title;
    formDescription = m.description;
    formStartDate = m.start_date || "";
    formDueDate = m.due_date || m.end_date || "";
    formColor = m.color || "#6366f1";
    formStatus = m.status;
    formError = null;
  }

  function cancelForm() {
    isEditing = false;
    editingId = null;
    formError = null;
  }

  async function saveMilestone(e: SubmitEvent) {
    e.preventDefault();
    const title = formTitle.trim();
    if (!title) {
      formError = "Enter a milestone title.";
      return;
    }
    if (title.length > 255) {
      formError = "Title must be 255 characters or fewer.";
      return;
    }
    const csrf = getCsrfToken();
    if (!csrf) {
      formError = "Security token missing. Reload and try again.";
      return;
    }

    saving = true;
    formError = null;
    try {
      const isNew = editingId === null;
      const url = isNew
        ? `/api/v1/workspaces/${workspace.id}/projects/${project.id}/milestones`
        : `/api/v1/workspaces/${workspace.id}/projects/${project.id}/milestones/${editingId}`;

      const payload: Record<string, any> = {
        title,
        description: formDescription,
        start_date: formStartDate ? formStartDate : null,
        end_date: formDueDate ? formDueDate : null,
        due_date: formDueDate ? formDueDate : null,
        color: formColor || "#6366f1",
      };
      if (!isNew) {
        payload.status = formStatus;
      }

      const res = await fetch(url, {
        method: isNew ? "POST" : "PATCH",
        headers: {
          "Content-Type": "application/json",
          "X-CSRF-Token": csrf,
          Accept: "application/json",
        },
        body: JSON.stringify(payload),
      });

      if (!res.ok) {
        const body = await res.json().catch(() => null);
        formError = body?.error?.message || "Failed to save milestone.";
        return;
      }

      cancelForm();
      await loadMilestones();
      onmilestonesupdated?.();
    } catch {
      formError = "Network error saving milestone.";
    } finally {
      saving = false;
    }
  }

  async function confirmDelete() {
    if (!deletingMilestone) return;
    const csrf = getCsrfToken();
    if (!csrf) {
      error = "Security token missing. Reload and try again.";
      return;
    }

    deleting = true;
    try {
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/projects/${project.id}/milestones/${deletingMilestone.id}`,
        {
          method: "DELETE",
          headers: {
            "X-CSRF-Token": csrf,
          },
        },
      );
      if (!res.ok) {
        error = "Failed to delete milestone.";
        return;
      }
      deletingMilestone = null;
      await loadMilestones();
      onmilestonesupdated?.();
    } catch {
      error = "Network error deleting milestone.";
    } finally {
      deleting = false;
    }
  }

  function getProgressPercent(m: Milestone): number {
    if (m.total_tasks <= 0) return 0;
    return Math.round((m.completed_tasks / m.total_tasks) * 100);
  }
</script>

<Modal
  title={`Milestones — ${project.name}`}
  description="Planning targets grouping tasks toward an outcome and date."
  wide={true}
  {onclose}
>
  {#snippet children()}
    <div class="milestones-container">
      {#if error}
        <div class="banner">
          <Feedback tone="danger">{error}</Feedback>
        </div>
      {/if}

      {#if deletingMilestone}
        <div class="confirm-box panel">
          <h3>Delete milestone "{deletingMilestone.title}"?</h3>
          <p class="confirm-desc">
            Tasks assigned to this milestone will not be deleted; they will
            become unassigned from this milestone. This action cannot be undone.
          </p>
          <div class="confirm-actions">
            <Button
              variant="quiet"
              disabled={deleting}
              onclick={() => (deletingMilestone = null)}
            >
              {#snippet children()}Cancel{/snippet}
            </Button>
            <Button
              variant="danger"
              disabled={deleting}
              onclick={() => void confirmDelete()}
            >
              {#snippet children()}
                {deleting ? "Deleting…" : "Delete milestone"}
              {/snippet}
            </Button>
          </div>
        </div>
      {:else if isEditing}
        <form
          class="milestone-form panel"
          onsubmit={(e) => void saveMilestone(e)}
        >
          <div class="form-head">
            <h3>{editingId ? "Edit milestone" : "New milestone"}</h3>
          </div>

          {#if formError}
            <Feedback tone="danger">{formError}</Feedback>
          {/if}

          <div class="field">
            <InputField
              id="milestone-title"
              label="Title"
              bind:value={formTitle}
              placeholder="e.g. Beta Launch, Q4 Core Upgrades"
              disabled={saving}
              required
            />
          </div>

          <div class="field">
            <label for="milestone-desc">Description</label>
            <textarea
              id="milestone-desc"
              class="textarea"
              bind:value={formDescription}
              placeholder="Target outcomes and scope of this milestone"
              disabled={saving}
              rows={3}></textarea>
          </div>

          <div class="field">
            <span class="field-label">Color</span>
            <div class="preset-palette">
              {#each presetColors as p}
                <button
                  type="button"
                  class="color-chip"
                  class:selected={formColor.toLowerCase() ===
                    p.value.toLowerCase()}
                  style:--chip-color={p.value}
                  onclick={() => (formColor = p.value)}
                  title={p.label}
                  aria-label={p.label}
                  aria-pressed={formColor.toLowerCase() ===
                    p.value.toLowerCase()}
                ></button>
              {/each}
              <label class="custom-color">
                <input
                  type="color"
                  bind:value={formColor}
                  aria-label="Choose custom milestone color"
                />
                <span class="hex-text">{formColor}</span>
              </label>
            </div>
          </div>

          <div class="form-row">
            <div class="field">
              <label for="milestone-start-date">Start date</label>
              <DatePicker
                id="milestone-start-date"
                label="Start date"
                placeholder="No start date"
                value={formStartDate}
                disabled={saving}
                onchange={(v) => (formStartDate = v || "")}
              />
            </div>

            <div class="field">
              <label for="milestone-due-date">End date</label>
              <DatePicker
                id="milestone-due-date"
                label="End date"
                placeholder="No end date"
                value={formDueDate}
                disabled={saving}
                onchange={(v) => (formDueDate = v || "")}
              />
            </div>

            {#if editingId}
              <div class="field">
                <label for="milestone-status">Status</label>
                <Select
                  id="milestone-status"
                  variant="field"
                  label="Status"
                  value={formStatus}
                  options={[
                    { value: "open", label: "Open" },
                    { value: "closed", label: "Closed" },
                  ]}
                  disabled={saving}
                  onchange={(v) => (formStatus = v as "open" | "closed")}
                />
              </div>
            {/if}
          </div>

          <div class="form-actions">
            <Button variant="quiet" disabled={saving} onclick={cancelForm}>
              {#snippet children()}Cancel{/snippet}
            </Button>
            <Button variant="primary" type="submit" disabled={saving}>
              {#snippet children()}
                {saving
                  ? "Saving…"
                  : editingId
                    ? "Save milestone"
                    : "Create milestone"}
              {/snippet}
            </Button>
          </div>
        </form>
      {:else}
        <div class="toolbar">
          <p class="milestone-count">
            {milestones.length === 1
              ? "1 milestone"
              : `${milestones.length} milestones`}
          </p>
          <Button variant="primary" onclick={startCreate}>
            {#snippet children()}
              <Icon name="plus" size={14} />New milestone
            {/snippet}
          </Button>
        </div>

        {#if loading}
          <div class="loading-state">
            <Feedback tone="neutral" announcement="status">
              Loading milestones…
            </Feedback>
          </div>
        {:else if milestones.length === 0}
          <div class="empty">
            <div class="empty-mark">
              <Icon name="flag" size={24} />
            </div>
            <h4>No milestones yet</h4>
            <p>
              Group tasks toward key targets, releases, or calendar dates for {project.name}.
            </p>
            <Button variant="primary" onclick={startCreate}>
              {#snippet children()}
                <Icon name="plus" size={14} />Create milestone
              {/snippet}
            </Button>
          </div>
        {:else}
          <div class="milestones-list" role="list">
            {#each milestones as m (m.id)}
              <div
                class="milestone-card panel"
                role="listitem"
                data-status={m.status}
                style:--milestone-color={m.color || "var(--accent)"}
              >
                <div class="card-head">
                  <div class="title-group">
                    <span
                      class="color-dot"
                      style:background={m.color || "var(--accent)"}
                      aria-hidden="true"
                    ></span>
                    <span
                      class="status-chip"
                      class:chip-open={m.status === "open"}
                      class:chip-closed={m.status === "closed"}
                    >
                      {m.status === "open" ? "Open" : "Closed"}
                    </span>
                    <h4 class="milestone-title">{m.title}</h4>
                  </div>
                  <div class="card-actions">
                    <Button
                      variant="quiet"
                      label={`Edit milestone ${m.title}`}
                      onclick={() => startEdit(m)}
                    >
                      {#snippet children()}
                        <Icon name="pencil" size={14} />Edit
                      {/snippet}
                    </Button>
                    <Button
                      variant="danger-quiet"
                      label={`Delete milestone ${m.title}`}
                      onclick={() => (deletingMilestone = m)}
                    >
                      {#snippet children()}
                        <Icon name="trash" size={14} />
                      {/snippet}
                    </Button>
                  </div>
                </div>

                {#if m.description}
                  <p class="milestone-description">{m.description}</p>
                {/if}

                <div class="meta-row">
                  <div class="due-pill">
                    <Icon name="calendar" size={14} />
                    <span>
                      {#if m.start_date && (m.due_date || m.end_date)}
                        {formatDueDate(m.start_date)} – {formatDueDate(
                          m.due_date || m.end_date || "",
                        )}
                      {:else if m.start_date}
                        From {formatDueDate(m.start_date)}
                      {:else if m.due_date || m.end_date}
                        Target: {formatDueDate(m.due_date || m.end_date || "")}
                      {:else}
                        No target dates
                      {/if}
                    </span>
                  </div>

                  <div class="task-stats">
                    {#if m.total_tasks > 0}
                      <span>
                        {m.completed_tasks} of {m.total_tasks} completed ({getProgressPercent(
                          m,
                        )}%)
                      </span>
                    {:else}
                      <span class="muted">0 tasks assigned</span>
                    {/if}
                  </div>
                </div>

                {#if m.total_tasks > 0}
                  <div
                    class="progress-rail"
                    role="progressbar"
                    aria-label={`Milestone progress: ${getProgressPercent(m)}%`}
                    aria-valuenow={getProgressPercent(m)}
                    aria-valuemin={0}
                    aria-valuemax={100}
                  >
                    <div
                      class="progress-fill"
                      style:inline-size={`${getProgressPercent(m)}%`}
                      style:background={m.color || "var(--accent)"}
                    ></div>
                  </div>
                {/if}
              </div>
            {/each}
          </div>
        {/if}
      {/if}
    </div>
  {/snippet}
</Modal>

<style>
  .milestones-container {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
    min-inline-size: min(100%, 36rem);
  }

  .banner {
    margin-block-end: var(--space-2);
  }

  .toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding-block-end: var(--space-2);
    border-block-end: 1px solid var(--line);
  }

  .milestone-count {
    margin: 0;
    font-size: var(--text-sm);
    color: var(--text-secondary);
  }

  .milestones-list {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .milestone-card {
    padding: var(--space-4);
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    border: 1px solid var(--border);
    border-radius: var(--radius-md);
    background: var(--surface-card);
  }

  .card-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
  }

  .title-group {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    flex-wrap: wrap;
  }

  .color-dot {
    inline-size: 0.65rem;
    block-size: 0.65rem;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .milestone-title {
    margin: 0;
    font-size: var(--text-base);
    font-weight: 600;
    color: var(--text);
  }

  .card-actions {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .status-chip {
    display: inline-flex;
    align-items: center;
    padding: 0.125rem 0.5rem;
    font-size: var(--text-xs);
    font-weight: 500;
    border-radius: var(--radius-full);
    border: 1px solid var(--border);
  }

  .chip-open {
    color: var(--cat-unstarted);
    border-color: var(--cat-unstarted);
    background: var(--surface-panel);
  }

  .chip-closed {
    color: var(--cat-completed);
    border-color: var(--cat-completed);
    background: var(--surface-panel);
  }

  .milestone-description {
    margin: 0;
    font-size: var(--text-sm);
    color: var(--text-secondary);
    line-height: 1.45;
  }

  .meta-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    flex-wrap: wrap;
    font-size: var(--text-xs);
    color: var(--text-secondary);
  }

  .due-pill {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .task-stats {
    font-weight: 500;
  }

  .muted {
    color: var(--text-muted);
    font-weight: 400;
  }

  .progress-rail {
    inline-size: 100%;
    block-size: 6px;
    background: var(--surface-panel);
    border-radius: var(--radius-full);
    overflow: hidden;
    border: 1px solid var(--line);
  }

  .progress-fill {
    block-size: 100%;
    background: var(--accent);
    border-radius: var(--radius-full);
    transition: inline-size var(--duration-normal) var(--ease-out);
  }

  /* Form */
  .milestone-form {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
    padding: var(--space-4);
  }

  .form-head h3 {
    margin: 0;
    font-size: var(--text-base);
    font-weight: 600;
    color: var(--text);
  }

  .field {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }

  .field label {
    font-size: var(--text-xs);
    font-weight: 500;
    color: var(--text-secondary);
  }

  .field-label {
    font-size: var(--text-xs);
    font-weight: 500;
    color: var(--text-secondary);
  }

  .preset-palette {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-2);
  }

  .color-chip {
    inline-size: 2rem;
    min-inline-size: 2rem;
    block-size: 2rem;
    min-block-size: 2rem;
    padding: 0;
    border-radius: 50%;
    border: 3px solid var(--surface-sunken);
    background: var(--chip-color);
    box-shadow: 0 0 0 1px var(--line-strong);
    cursor: pointer;
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
    min-block-size: var(--control-height);
    padding: 0 var(--space-3) 0 var(--space-1);
    border-radius: var(--radius-full);
    border: var(--border-width) solid var(--line-strong);
    background: var(--surface-raised);
    cursor: pointer;
  }

  .custom-color input {
    inline-size: 1.75rem;
    block-size: 1.75rem;
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

  .textarea {
    inline-size: 100%;
    font-size: var(--text-sm);
    color: var(--text);
    background: var(--surface-panel);
    border: 1px solid var(--border);
    border-radius: var(--radius-sm);
    padding: var(--space-2) var(--space-3);
    resize: vertical;
    box-sizing: border-box;
  }

  .textarea:focus {
    outline: none;
    border-color: var(--accent);
    box-shadow: 0 0 0 1px var(--accent);
  }

  .form-row {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(12rem, 1fr));
    gap: var(--space-3);
  }

  .form-actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--space-2);
    margin-block-start: var(--space-2);
  }

  /* Confirm dialog */
  .confirm-box {
    padding: var(--space-4);
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }

  .confirm-box h3 {
    margin: 0;
    font-size: var(--text-base);
    font-weight: 600;
    color: var(--text);
  }

  .confirm-desc {
    margin: 0;
    font-size: var(--text-sm);
    color: var(--text-secondary);
    line-height: 1.45;
  }

  .confirm-actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--space-2);
    margin-block-start: var(--space-2);
  }

  /* Empty state */
  .empty {
    padding: var(--space-8) var(--space-4);
    display: flex;
    flex-direction: column;
    align-items: center;
    text-align: center;
    gap: var(--space-3);
  }

  .empty-mark {
    inline-size: 48px;
    block-size: 48px;
    border-radius: var(--radius-md);
    background: var(--surface-card);
    border: 1px solid var(--border);
    display: flex;
    align-items: center;
    justify-content: center;
    color: var(--accent);
  }

  .empty h4 {
    margin: 0;
    font-size: var(--text-base);
    font-weight: 600;
    color: var(--text);
  }

  .empty p {
    margin: 0;
    font-size: var(--text-sm);
    color: var(--text-secondary);
    max-inline-size: 24rem;
  }

  .loading-state {
    padding: var(--space-6);
    display: flex;
    justify-content: center;
  }
</style>
