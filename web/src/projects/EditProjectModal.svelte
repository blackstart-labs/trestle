<script lang="ts">
  import Button from "../ui/Button.svelte";
  import Feedback from "../ui/Feedback.svelte";
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
  let saving = $state(false);
  let error = $state<string | null>(null);
  const canManage = $derived(
    workspace.role === "owner" || workspace.role === "admin",
  );
  const dirty = $derived(
    name.trim() !== project.name ||
      slug.trim().toLowerCase() !== project.slug ||
      description !== project.description,
  );

  function requestClose() {
    if (saving) return;
    if (dirty && !confirm("Discard your unsaved changes to this project?"))
      return;
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
      <label for="edit-project-description">Description</label>
      <textarea
        id="edit-project-description"
        bind:value={description}
        rows="4"
        maxlength="2000"
        disabled={saving || !canManage}></textarea>
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
</style>
