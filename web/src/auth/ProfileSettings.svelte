<script lang="ts">
  import Button from "../ui/Button.svelte";
  import Feedback from "../ui/Feedback.svelte";
  import Icon from "../ui/Icon.svelte";
  import Toast from "../ui/Toast.svelte";
  import type { User } from "./SignIn.svelte";
  import { identityHue, initials } from "../lib/identity";

  let {
    user,
    onuserupdated,
  }: {
    user: User;
    onuserupdated?: (user: User) => void;
  } = $props();

  // svelte-ignore state_referenced_locally
  let displayName = $state(user.display_name);
  let avatarVersion = $state(Date.now());
  let avatarPreview = $state<string | null>(null);
  let savingName = $state(false);
  let uploadingAvatar = $state(false);
  let deletingAvatar = $state(false);
  let error = $state<string | null>(null);
  let toastMessage = $state<string | null>(null);

  const dirtyName = $derived(displayName.trim() !== user.display_name);

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

  async function handleAvatarFile(e: Event) {
    const input = e.target as HTMLInputElement;
    const file = input.files?.[0];
    if (!file) return;
    if (file.size > 5 * 1024 * 1024) {
      error = "Image must be 5 MB or smaller.";
      return;
    }

    uploadingAvatar = true;
    error = null;
    const csrf = getCsrfToken();
    const formData = new FormData();
    formData.append("file", file);

    try {
      const res = await fetch("/api/v1/me/avatar", {
        method: "PUT",
        headers: csrf ? { "X-CSRF-Token": csrf } : {},
        body: formData,
      });

      if (!res.ok) {
        const data = await res.json().catch(() => null);
        error = data?.error?.message || "Failed to update profile picture.";
        return;
      }

      avatarVersion = Date.now();
      avatarPreview = null;
      toastMessage = "Profile picture updated.";
      onuserupdated?.({
        ...user,
        avatar_url: `/api/v1/users/${user.id}/avatar?v=${avatarVersion}`,
      });
    } catch {
      error = "Network error while uploading profile picture.";
    } finally {
      uploadingAvatar = false;
      input.value = "";
    }
  }

  async function handleRemoveAvatar() {
    if (!confirm("Remove your profile picture?")) return;
    deletingAvatar = true;
    error = null;
    const csrf = getCsrfToken();

    try {
      const res = await fetch("/api/v1/me/avatar", {
        method: "DELETE",
        headers: csrf ? { "X-CSRF-Token": csrf } : {},
      });

      if (!res.ok && res.status !== 204) {
        error = "Failed to remove profile picture.";
        return;
      }

      avatarVersion = Date.now();
      avatarPreview = null;
      toastMessage = "Profile picture removed.";
      onuserupdated?.({ ...user, avatar_url: null });
    } catch {
      error = "Network error while removing profile picture.";
    } finally {
      deletingAvatar = false;
    }
  }

  async function handleSaveName(e: SubmitEvent) {
    e.preventDefault();
    const trimmed = displayName.trim();
    if (!trimmed || trimmed.length > 120) {
      error = "Display name must be between 1 and 120 characters.";
      return;
    }
    savingName = true;
    error = null;
    const csrf = getCsrfToken();

    try {
      const res = await fetch("/api/v1/me", {
        method: "PATCH",
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
          ...(csrf ? { "X-CSRF-Token": csrf } : {}),
        },
        body: JSON.stringify({ display_name: trimmed }),
      });

      if (!res.ok) {
        const data = await res.json().catch(() => null);
        error = data?.error?.message || "Failed to save name.";
        return;
      }

      const updated = await res.json();
      toastMessage = "Display name saved.";
      onuserupdated?.({ ...user, display_name: updated.display_name });
    } catch {
      error = "Network error while saving name.";
    } finally {
      savingName = false;
    }
  }
</script>

<section class="panel panel-pad" aria-labelledby="profile-heading">
  <div class="pref-head">
    <h2 id="profile-heading" tabindex="-1">Profile</h2>
    <p>Your personal account details and avatar.</p>
  </div>

  {#if error}
    <Feedback tone="danger">{error}</Feedback>
  {/if}

  {#if toastMessage}
    <Toast message={toastMessage} onclear={() => (toastMessage = null)} />
  {/if}

  <div class="field">
    <span class="field-label">Profile picture</span>
    <div class="avatar-uploader">
      <div class="avatar-display" style:--avatar-hue={identityHue(user.id)}>
        <img
          src={`/api/v1/users/${user.id}/avatar?v=${avatarVersion}`}
          alt={user.display_name}
          class="avatar-img"
          onerror={(e) => {
            (e.currentTarget as HTMLElement).style.display = "none";
            const fallback = (e.currentTarget as HTMLElement)
              .nextElementSibling;
            if (fallback) (fallback as HTMLElement).style.display = "flex";
          }}
        />
        <span class="avatar-fallback" aria-hidden="true">
          {initials(user.display_name)}
        </span>
      </div>

      <div class="uploader-controls">
        <label class="btn btn-secondary file-picker-label">
          <Icon name="upload" size={16} />
          <span>{uploadingAvatar ? "Uploading…" : "Change picture"}</span>
          <input
            type="file"
            accept="image/png,image/jpeg,image/webp,image/gif,image/svg+xml"
            onchange={handleAvatarFile}
            disabled={uploadingAvatar || deletingAvatar}
            class="visually-hidden"
            aria-label="Upload profile picture"
          />
        </label>
        <Button
          variant="quiet"
          disabled={uploadingAvatar || deletingAvatar}
          onclick={handleRemoveAvatar}
        >
          {#snippet children()}{deletingAvatar
              ? "Removing…"
              : "Remove picture"}{/snippet}
        </Button>
      </div>
    </div>
    <p class="note">PNG, JPEG, WebP, GIF or SVG up to 5 MB.</p>
  </div>

  <form onsubmit={handleSaveName} class="profile-form">
    <div class="field">
      <label for="profile-username">Email / Username</label>
      <input
        id="profile-username"
        value={user.username}
        readonly
        aria-describedby="profile-username-hint"
      />
      <p class="note" id="profile-username-hint">
        Contact workspace administrator to change login credentials.
      </p>
    </div>

    <div class="field">
      <label for="profile-name">Display name</label>
      <input
        id="profile-name"
        bind:value={displayName}
        maxlength="120"
        required
        aria-describedby="profile-name-hint"
      />
      <p class="note" id="profile-name-hint">
        Visible across all projects, comments and task assignments.
      </p>
    </div>

    <div class="form-actions">
      <Button
        type="submit"
        variant="primary"
        disabled={savingName || !dirtyName}
      >
        {#snippet children()}{savingName ? "Saving…" : "Save changes"}{/snippet}
      </Button>
    </div>
  </form>
</section>

<style>
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
    border-radius: var(--radius-full);
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
  .profile-form {
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
</style>
