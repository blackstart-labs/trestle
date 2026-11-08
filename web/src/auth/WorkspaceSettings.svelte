<script lang="ts">
  import Button from "../ui/Button.svelte";
  import Feedback from "../ui/Feedback.svelte";
  import Icon from "../ui/Icon.svelte";
  import Toast from "../ui/Toast.svelte";
  import type { Workspace } from "./WorkspaceChooser.svelte";
  import { identityHue, initials } from "../lib/identity";

  let {
    workspace,
    onworkspaceupdated,
  }: {
    workspace: Workspace;
    onworkspaceupdated?: (ws: Workspace) => void;
  } = $props();

  // svelte-ignore state_referenced_locally
  let name = $state(workspace.name);
  // svelte-ignore state_referenced_locally
  let slug = $state(workspace.slug);
  let imageVersion = $state(Date.now());
  let saving = $state(false);
  let uploadingImage = $state(false);
  let deletingImage = $state(false);
  let error = $state<string | null>(null);
  let toastMessage = $state<string | null>(null);

  const canManage = $derived(
    workspace.role === "owner" || workspace.role === "admin",
  );

  const dirty = $derived(
    name.trim() !== workspace.name ||
      slug.trim().toLowerCase() !== workspace.slug,
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

  async function handleImageFile(e: Event) {
    if (!canManage) return;
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    if (file.size > 5 * 1024 * 1024) {
      error = "Image must be 5 MB or smaller.";
      return;
    }

    uploadingImage = true;
    error = null;
    const csrf = getCsrfToken();
    const formData = new FormData();
    formData.append("file", file);

    try {
      const res = await fetch(`/api/v1/workspaces/${workspace.id}/image`, {
        method: "PUT",
        headers: csrf ? { "X-CSRF-Token": csrf } : {},
        body: formData,
      });

      if (!res.ok) {
        const data = await res.json().catch(() => null);
        error = data?.error?.message || "Failed to update workspace icon.";
        return;
      }

      imageVersion = Date.now();
      toastMessage = "Workspace icon updated.";
      onworkspaceupdated?.({
        ...workspace,
        avatar_url: `/api/v1/workspaces/${workspace.id}/image?v=${imageVersion}`,
      });
    } catch {
      error = "Network error while uploading workspace icon.";
    } finally {
      uploadingImage = false;
      input.value = "";
    }
  }

  async function handleRemoveImage() {
    if (!canManage) return;
    if (!confirm("Remove the workspace icon?")) return;
    deletingImage = true;
    error = null;
    const csrf = getCsrfToken();

    try {
      const res = await fetch(`/api/v1/workspaces/${workspace.id}/image`, {
        method: "DELETE",
        headers: csrf ? { "X-CSRF-Token": csrf } : {},
      });

      if (!res.ok && res.status !== 204) {
        error = "Failed to remove workspace icon.";
        return;
      }

      imageVersion = Date.now();
      toastMessage = "Workspace icon removed.";
      onworkspaceupdated?.({ ...workspace, avatar_url: null });
    } catch {
      error = "Network error while removing workspace icon.";
    } finally {
      deletingImage = false;
    }
  }

  async function handleSaveDetails(e: SubmitEvent) {
    e.preventDefault();
    if (!canManage) return;
    const trimmedName = name.trim();
    const trimmedSlug = slug.trim().toLowerCase();

    if (!trimmedName || trimmedName.length > 120) {
      error = "Workspace name must be between 1 and 120 characters.";
      return;
    }
    if (
      trimmedSlug.length < 2 ||
      trimmedSlug.length > 48 ||
      !/^[a-z0-9][a-z0-9-]*[a-z0-9]$/.test(trimmedSlug)
    ) {
      error =
        "Workspace slug must be 2-48 lowercase alphanumeric characters or hyphens.";
      return;
    }

    saving = true;
    error = null;
    const csrf = getCsrfToken();

    try {
      const res = await fetch(`/api/v1/workspaces/${workspace.id}`, {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
          ...(csrf ? { "X-CSRF-Token": csrf } : {}),
        },
        body: JSON.stringify({ name: trimmedName, slug: trimmedSlug }),
      });

      if (!res.ok) {
        const data = await res.json().catch(() => null);
        error = data?.error?.message || "Failed to save workspace.";
        return;
      }

      const updated = await res.json();
      toastMessage = "Workspace details saved.";
      onworkspaceupdated?.({
        ...workspace,
        name: updated.name,
        slug: updated.slug,
      });
    } catch {
      error = "Network error while saving workspace.";
    } finally {
      saving = false;
    }
  }
</script>

<div class="workspace-settings-container">
  <section class="panel panel-pad" aria-labelledby="ws-heading">
    <div class="pref-head">
      <h2 id="ws-heading" tabindex="-1">Workspace settings</h2>
      <p>Name, URL slug and branding for {workspace.name}.</p>
    </div>

    {#if error}
      <Feedback tone="danger">{error}</Feedback>
    {/if}

    {#if toastMessage}
      <Toast message={toastMessage} onclear={() => (toastMessage = null)} />
    {/if}

    <div class="field">
      <span class="field-label">Workspace icon</span>
      <div class="avatar-uploader">
        <div
          class="avatar-display"
          style:--avatar-hue={identityHue(workspace.id)}
        >
          <img
            src={`/api/v1/workspaces/${workspace.id}/image?v=${imageVersion}`}
            alt={workspace.name}
            class="avatar-img"
            onerror={(e) => {
              (e.currentTarget as HTMLElement).style.display = "none";
              const fallback = (e.currentTarget as HTMLElement)
                .nextElementSibling;
              if (fallback) (fallback as HTMLElement).style.display = "flex";
            }}
          />
          <span class="avatar-fallback" aria-hidden="true">
            {initials(workspace.name)}
          </span>
        </div>

        {#if canManage}
          <div class="uploader-controls">
            <label class="btn btn-secondary file-picker-label">
              <Icon name="upload" size={16} />
              <span>{uploadingImage ? "Uploading…" : "Change icon"}</span>
              <input
                type="file"
                accept="image/png,image/jpeg,image/webp,image/gif,image/svg+xml"
                onchange={handleImageFile}
                disabled={uploadingImage || deletingImage}
                class="visually-hidden"
                aria-label="Upload workspace icon"
              />
            </label>
            <Button
              variant="quiet"
              disabled={uploadingImage || deletingImage}
              onclick={handleRemoveImage}
            >
              {#snippet children()}{deletingImage
                  ? "Removing…"
                  : "Remove icon"}{/snippet}
            </Button>
          </div>
        {/if}
      </div>
      <p class="note">PNG, JPEG, WebP, GIF or SVG up to 5 MB.</p>
    </div>

    <form onsubmit={handleSaveDetails} class="ws-form">
      <div class="field">
        <label for="ws-name">Workspace name</label>
        <input
          id="ws-name"
          bind:value={name}
          maxlength="120"
          required
          disabled={!canManage || saving}
        />
      </div>

      <div class="field">
        <label for="ws-slug">Workspace slug</label>
        <input
          id="ws-slug"
          bind:value={slug}
          maxlength="48"
          required
          disabled={!canManage || saving}
          aria-describedby="ws-slug-hint"
        />
        <p class="note" id="ws-slug-hint">
          Used in links and data export filenames.
        </p>
      </div>

      {#if canManage}
        <div class="form-actions">
          <Button type="submit" variant="primary" disabled={saving || !dirty}>
            {#snippet children()}{saving ? "Saving…" : "Save changes"}{/snippet}
          </Button>
        </div>
      {/if}
    </form>
  </section>

  <section
    class="panel panel-pad export-section"
    aria-labelledby="export-heading"
  >
    <div class="pref-head">
      <h2 id="export-heading" tabindex="-1">Data export & backup</h2>
      <p>Download a complete snapshot of all workspace data as JSON.</p>
    </div>

    <p class="export-description">
      Includes all projects, task statuses, milestones, tasks, subtasks,
      assignments, and logged time entries. You can use this snapshot for
      off-site backups or migration.
    </p>

    <div class="export-actions">
      <a
        href={`/api/v1/workspaces/${workspace.id}/export`}
        download={`${workspace.slug}-export.json`}
        class="btn btn-secondary"
      >
        <Icon name="upload" size={16} />
        <span>Export all workspace data (JSON)</span>
      </a>
    </div>
  </section>
</div>

<style>
  .workspace-settings-container {
    display: flex;
    flex-direction: column;
    gap: var(--space-5);
  }
  .pref-head {
    margin-bottom: var(--space-5);
  }
  .pref-head h2 {
    font-size: var(--text-lg);
    font-weight: 520;
    margin: 0 0 var(--space-1) 0;
  }
  .pref-head p {
    margin: 0;
    color: var(--text-secondary);
    font-size: var(--text-sm);
  }
  .avatar-uploader {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    margin-top: var(--space-2);
  }
  .avatar-display {
    width: 4rem;
    height: 4rem;
    border-radius: var(--radius-md);
    background: oklch(0.85 0.08 var(--avatar-hue, 220));
    color: oklch(0.25 0.08 var(--avatar-hue, 220));
    display: flex;
    align-items: center;
    justify-content: center;
    font-weight: 600;
    font-size: var(--text-lg);
    overflow: hidden;
    flex-shrink: 0;
    border: var(--border-width) solid var(--line);
  }
  .avatar-img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }
  .avatar-fallback {
    display: none;
    width: 100%;
    height: 100%;
    align-items: center;
    justify-content: center;
  }
  .uploader-controls {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-2);
  }
  .file-picker-label {
    cursor: pointer;
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
  }
  .visually-hidden {
    position: absolute;
    width: 1px;
    height: 1px;
    padding: 0;
    margin: -1px;
    overflow: hidden;
    clip: rect(0, 0, 0, 0);
    border: 0;
  }
  .ws-form {
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
    margin-top: var(--space-4);
  }
  .form-actions {
    display: flex;
    justify-content: flex-end;
    margin-top: var(--space-2);
  }
  .export-section {
    border-top: var(--border-width) solid var(--line);
  }
  .export-description {
    font-size: var(--text-sm);
    color: var(--text-secondary);
    line-height: var(--line-height-base);
    margin: 0 0 var(--space-4) 0;
  }
  .export-actions {
    display: flex;
    align-items: center;
  }
</style>
