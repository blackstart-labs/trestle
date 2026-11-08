<script lang="ts">
  import Button from "../ui/Button.svelte";
  import Feedback from "../ui/Feedback.svelte";
  import Icon from "../ui/Icon.svelte";
  import Modal from "../ui/Modal.svelte";
  import type { Project } from "./ProjectsView.svelte";

  let {
    project,
    workspace,
    onclose,
    onupdated,
  }: {
    project: Project;
    workspace: { id: string; role: string };
    onclose: () => void;
    onupdated: (project: Project) => void;
  } = $props();

  // The dialog edits one Project; the caller closes it after a successful save.
  // svelte-ignore state_referenced_locally
  let name = $state(project.name);
  // svelte-ignore state_referenced_locally
  let slug = $state(project.slug);
  // svelte-ignore state_referenced_locally
  let description = $state(project.description);
  let newImage = $state("");
  // svelte-ignore state_referenced_locally
  let newImagePreview = $state(project.image_url || "");
  let removeImage = $state(false);
  let saving = $state(false);
  let error = $state<string | null>(null);
  const canManage = $derived(
    workspace.role === "owner" || workspace.role === "admin",
  );
  const dirty = $derived(
    name.trim() !== project.name ||
      slug.trim().toLowerCase() !== project.slug ||
      description !== project.description ||
      newImage !== "" ||
      (removeImage && Boolean(project.image_url)),
  );

  function handleImageFile(e: Event) {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    if (file.size > 5 * 1024 * 1024) {
      error = "Image must be 5 MB or smaller.";
      return;
    }
    removeImage = false;
    if (newImagePreview && newImagePreview.startsWith("blob:")) {
      URL.revokeObjectURL(newImagePreview);
    }
    const blobUrl = URL.createObjectURL(file);
    newImagePreview = blobUrl;
    const reader = new FileReader();
    reader.onload = () => {
      newImage = reader.result as string;
    };
    reader.readAsDataURL(file);
  }

  function handleRemoveImage() {
    newImage = "";
    if (newImagePreview && newImagePreview.startsWith("blob:")) {
      URL.revokeObjectURL(newImagePreview);
    }
    newImagePreview = "";
    removeImage = true;
  }

  function requestClose() {
    if (saving) return;
    if (dirty && !confirm("Discard your unsaved changes to this project?"))
      return;
    if (newImagePreview && newImagePreview.startsWith("blob:")) {
      URL.revokeObjectURL(newImagePreview);
    }
    onclose();
  }

  async function save(event: SubmitEvent) {
    event.preventDefault();
    if (saving || !dirty || !canManage) return;
    error = null;
    const trimmedName = name.trim();
    const normalizedSlug = slug.trim().toLowerCase();
    const bytes = (value: string) => new TextEncoder().encode(value).length;
    if (!trimmedName || bytes(trimmedName) > 120) {
      error = "Enter a project name of up to 120 characters.";
      return;
    }
    if (
      normalizedSlug.length < 2 ||
      normalizedSlug.length > 48 ||
      !/^[a-z0-9][a-z0-9-]*[a-z0-9]$/.test(normalizedSlug)
    ) {
      error =
        "Use 2–48 lowercase letters, numbers, or hyphens for the slug. Start and end with a letter or number.";
      return;
    }
    if (bytes(description) > 2000) {
      error = "The project description is too long. Shorten it and try again.";
      return;
    }
    const csrf = document.cookie
      .split(";")
      .map((part) => part.trim())
      .find((part) => part.startsWith("trestle-csrf="))
      ?.slice("trestle-csrf=".length);
    if (!csrf) {
      error = "Security token missing. Reload and try again.";
      return;
    }
    const payload: Record<string, string> = {};
    if (trimmedName !== project.name) payload.name = trimmedName;
    if (normalizedSlug !== project.slug) payload.slug = normalizedSlug;
    if (description !== project.description) payload.description = description;
    if (removeImage && project.image_url) {
      payload.image = "";
    } else if (newImage) {
      payload.image = newImage;
    }
    saving = true;
    try {
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/projects/${project.id}`,
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
        if (newImagePreview && newImagePreview.startsWith("blob:")) {
          URL.revokeObjectURL(newImagePreview);
        }
        onupdated(await res.json());
      } else {
        const data = await res.json().catch(() => null);
        error =
          data?.error?.message ||
          (res.status === 409
            ? "A project already uses this slug. Choose a different slug and save again."
            : "Could not update the project. Try again.");
      }
    } catch {
      error =
        "Could not update the project. Check your connection and try again.";
    } finally {
      saving = false;
    }
  }
</script>

<Modal
  title="Edit project"
  description="Update the details for {project.name}."
  onclose={requestClose}
>
  {#if error}<Feedback tone="danger" announcement="alert">{error}</Feedback
    >{/if}
  {#if !canManage}<Feedback tone="danger"
      >Only workspace owners and administrators can edit projects.</Feedback
    >{/if}
  <form class="form-grid" onsubmit={save}>
    <div class="field">
      <label for="edit-project-name">Project name</label>
      <input
        id="edit-project-name"
        bind:value={name}
        required
        maxlength="120"
        disabled={saving || !canManage}
      />
    </div>
    <div class="form-row">
      <div class="field">
        <label for="edit-project-slug">Slug</label>
        <input
          id="edit-project-slug"
          bind:value={slug}
          required
          minlength="2"
          maxlength="48"
          disabled={saving || !canManage}
          aria-describedby="edit-project-slug-hint"
        />
        <p class="note" id="edit-project-slug-hint">
          Unique within this workspace. Use lowercase letters, numbers, and
          hyphens.
        </p>
      </div>
      <div class="field">
        <label for="edit-project-key">Project key</label>
        <input
          id="edit-project-key"
          value={project.key}
          readonly
          aria-describedby="edit-project-key-hint"
        />
        <p class="note" id="edit-project-key-hint">
          Fixed so existing task keys stay the same.
        </p>
      </div>
    </div>
    <div class="field">
      <label for="edit-project-image">Project icon (optional)</label>
      <div class="image-uploader">
        <div class="avatar-preview">
          {#if newImagePreview}
            <img
              src={newImagePreview}
              alt="Project icon preview"
              class="preview-img"
            />
          {:else}
            <span class="preview-empty" aria-hidden="true">
              <Icon name="image" size={20} />
            </span>
          {/if}
        </div>
        <div class="uploader-actions">
          <label class="btn btn-secondary file-picker-label">
            <Icon name="upload" size={16} />
            <span>{newImagePreview ? "Change image" : "Upload image"}</span>
            <input
              id="edit-project-image"
              type="file"
              accept="image/png,image/jpeg,image/webp,image/gif,image/svg+xml"
              onchange={handleImageFile}
              disabled={saving || !canManage}
              class="visually-hidden"
            />
          </label>
          {#if newImagePreview}
            <button
              type="button"
              class="btn-quiet btn-icon"
              onclick={handleRemoveImage}
              disabled={saving || !canManage}
              aria-label="Remove image"
              title="Remove image"
            >
              <Icon name="trash" size={16} />
            </button>
          {/if}
        </div>
      </div>
      <p class="note">PNG, JPEG, WebP, GIF or SVG up to 5 MB.</p>
    </div>
    <div class="field">
      <label for="edit-project-description">Description</label>
      <textarea
        id="edit-project-description"
        bind:value={description}
        rows="4"
        maxlength="2000"
        disabled={saving || !canManage}></textarea>
    </div>

    <div class="project-export-box">
      <div class="export-meta">
        <span class="export-title">Project Data Export</span>
        <span class="export-note"
          >Download tasks, milestones, statuses, and time logs.</span
        >
      </div>
      <a
        href={`/api/v1/workspaces/${workspace.id}/projects/${project.id}/export`}
        download={`${project.key}-export.json`}
        class="btn btn-secondary btn-export"
      >
        <Icon name="upload" size={16} />
        <span>Export JSON</span>
      </a>
    </div>

    {#if dirty}<p class="note" aria-live="polite">Unsaved changes</p>{/if}
    <div class="dialog-actions">
      <Button variant="quiet" disabled={saving} onclick={requestClose}
        >{#snippet children()}Cancel{/snippet}</Button
      >
      <Button
        type="submit"
        variant="primary"
        disabled={saving || !dirty || !canManage}
        >{#snippet children()}{saving
            ? "Saving…"
            : "Save project"}{/snippet}</Button
      >
    </div>
  </form>
</Modal>

<style>
  .form-row {
    align-items: start;
  }

  .image-uploader {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  .avatar-preview {
    width: 3rem;
    height: 3rem;
    border-radius: var(--radius);
    border: var(--border-width) solid var(--line);
    background: var(--surface-sunken);
    display: grid;
    place-items: center;
    overflow: hidden;
    flex-shrink: 0;
  }

  .preview-img {
    width: 100%;
    height: 100%;
    object-fit: cover;
  }

  .preview-empty {
    color: var(--text-muted);
    display: grid;
    place-items: center;
  }

  .uploader-actions {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .project-export-box {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-4);
    padding: var(--space-3);
    border-radius: var(--radius);
    background: var(--surface-muted);
    border: var(--border-width) solid var(--line);
    margin-top: var(--space-2);
  }

  .export-meta {
    display: flex;
    flex-direction: column;
    gap: var(--space-0-5);
  }

  .export-title {
    font-size: var(--text-sm);
    font-weight: 500;
    color: var(--text);
  }

  .export-note {
    font-size: var(--text-xs);
    color: var(--text-secondary);
  }

  .btn-export {
    text-decoration: none;
    flex-shrink: 0;
  }

  .file-picker-label {
    cursor: pointer;
  }
</style>
