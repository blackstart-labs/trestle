<script lang="ts">
  import { onMount } from "svelte";
  import Button from "../ui/Button.svelte";
  import InputField from "../ui/InputField.svelte";
  import Feedback from "../ui/Feedback.svelte";

  export interface Member {
    user_id: string;
    username: string;
    display_name: string;
    role: "owner" | "admin" | "member" | "guest";
  }

  let {
    workspace,
    currentUserId,
    onleave,
  }: {
    workspace: { id: string; name: string; slug: string; role: string };
    currentUserId: string;
    onleave?: () => void;
  } = $props();

  let members = $state<Member[]>([]);
  let loading = $state(true);
  let errorMessage = $state<string | null>(null);
  let successMessage = $state<string | null>(null);

  // Add member form state
  let addUsername = $state("");
  let addRole = $state<"admin" | "member" | "guest">("member");
  let adding = $state(false);

  // Transfer ownership modal state
  let showTransferModal = $state(false);
  let transferTargetId = $state("");
  let transferPassword = $state("");
  let transferring = $state(false);
  let transferError = $state<string | null>(null);

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

  async function loadMembers() {
    loading = true;
    errorMessage = null;
    try {
      const res = await fetch(`/api/v1/workspaces/${workspace.id}/members`, {
        headers: { Accept: "application/json" },
        cache: "no-store",
      });
      if (res.status === 200) {
        const data = await res.json();
        members = data.members || [];
      } else {
        errorMessage = "Failed to load workspace members.";
      }
    } catch {
      errorMessage = "Network error loading members.";
    } finally {
      loading = false;
    }
  }

  async function handleAddMember(e: SubmitEvent) {
    e.preventDefault();
    if (!addUsername.trim()) return;

    adding = true;
    errorMessage = null;
    successMessage = null;
    const csrf = getCsrfToken();

    try {
      const res = await fetch(`/api/v1/workspaces/${workspace.id}/members`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Accept: "application/json",
          ...(csrf ? { "X-CSRF-Token": csrf } : {}),
        },
        body: JSON.stringify({
          username: addUsername.trim(),
          role: addRole,
        }),
      });

      if (res.status === 201) {
        const newMember = await res.json();
        members = [...members, newMember];
        addUsername = "";
        successMessage = `Added @${newMember.username} as ${newMember.role}.`;
      } else if (res.status === 409) {
        errorMessage = "User is already a member of this workspace.";
      } else if (res.status === 404) {
        errorMessage =
          "User not found. Ensure the user is already provisioned.";
      } else if (res.status === 400) {
        errorMessage = "Invalid username or role.";
      } else if (res.status === 403) {
        errorMessage = "You do not have permission to add members.";
      } else {
        errorMessage = "Failed to add member.";
      }
    } catch {
      errorMessage = "Network error adding member.";
    } finally {
      adding = false;
    }
  }

  async function handleRoleChange(member: Member, newRole: string) {
    errorMessage = null;
    successMessage = null;
    const csrf = getCsrfToken();

    try {
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/members/${member.user_id}`,
        {
          method: "PATCH",
          headers: {
            "Content-Type": "application/json",
            Accept: "application/json",
            ...(csrf ? { "X-CSRF-Token": csrf } : {}),
          },
          body: JSON.stringify({ role: newRole }),
        },
      );

      if (res.status === 200) {
        const updated = await res.json();
        members = members.map((m) =>
          m.user_id === member.user_id ? updated : m,
        );
        successMessage = `Updated @${member.username}'s role to ${newRole}.`;
      } else if (res.status === 409) {
        errorMessage = "Cannot demote the last active owner.";
      } else if (res.status === 403) {
        errorMessage = "Permission denied changing member role.";
      } else {
        errorMessage = "Failed to update member role.";
      }
    } catch {
      errorMessage = "Network error updating member role.";
    }
  }

  async function handleRemoveMember(member: Member) {
    const isSelf = member.user_id === currentUserId;
    const confirmText = isSelf
      ? "Are you sure you want to leave this workspace?"
      : `Remove @${member.username} from this workspace?`;

    if (!window.confirm(confirmText)) return;

    errorMessage = null;
    successMessage = null;
    const csrf = getCsrfToken();

    try {
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/members/${member.user_id}`,
        {
          method: "DELETE",
          headers: csrf ? { "X-CSRF-Token": csrf } : {},
        },
      );

      if (res.status === 204) {
        members = members.filter((m) => m.user_id !== member.user_id);
        if (isSelf && onleave) {
          onleave();
        } else {
          successMessage = `Removed @${member.username} from workspace.`;
        }
      } else if (res.status === 409) {
        errorMessage = "Cannot remove the last active owner.";
      } else if (res.status === 403) {
        errorMessage = "Permission denied removing member.";
      } else {
        errorMessage = "Failed to remove member.";
      }
    } catch {
      errorMessage = "Network error removing member.";
    }
  }

  async function handleTransferOwnership(e: SubmitEvent) {
    e.preventDefault();
    if (!transferTargetId || !transferPassword) return;

    transferring = true;
    transferError = null;
    const csrf = getCsrfToken();

    try {
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/transfer-ownership`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            Accept: "application/json",
            ...(csrf ? { "X-CSRF-Token": csrf } : {}),
          },
          body: JSON.stringify({
            target_user_id: transferTargetId,
            password: transferPassword,
          }),
        },
      );

      if (res.status === 200) {
        showTransferModal = false;
        transferPassword = "";
        successMessage = "Workspace ownership transferred successfully.";
        workspace.role = "admin";
        await loadMembers();
      } else if (res.status === 401) {
        transferError = "Invalid password confirmation.";
      } else if (res.status === 403) {
        transferError = "Only the workspace owner can transfer ownership.";
      } else {
        transferError = "Failed to transfer ownership.";
      }
    } catch {
      transferError = "Network error during ownership transfer.";
    } finally {
      transferring = false;
    }
  }

  onMount(() => {
    void loadMembers();
  });
</script>

<section class="members-section" aria-labelledby="members-heading">
  <div class="header-row">
    <div>
      <h2 id="members-heading">Members of {workspace.name}</h2>
      <p class="note">Manage workspace access, roles and ownership.</p>
    </div>
    {#if workspace.role === "owner" && members.length > 1}
      <button
        type="button"
        class="transfer-btn"
        onclick={() => (showTransferModal = true)}
      >
        Transfer ownership
      </button>
    {/if}
  </div>

  {#if errorMessage}
    <Feedback tone="danger">{errorMessage}</Feedback>
  {/if}
  {#if successMessage}
    <Feedback tone="success">{successMessage}</Feedback>
  {/if}

  {#if loading}
    <p class="note">Loading members...</p>
  {:else}
    <div class="table-wrapper">
      <table class="members-table" aria-label="Workspace members">
        <thead>
          <tr>
            <th scope="col">Member</th>
            <th scope="col">Role</th>
            <th scope="col">Actions</th>
          </tr>
        </thead>
        <tbody>
          {#each members as member (member.user_id)}
            <tr>
              <td>
                <div class="member-info">
                  <span class="member-name">{member.display_name}</span>
                  <span class="member-handle">@{member.username}</span>
                </div>
              </td>
              <td>
                {#if (workspace.role === "owner" && member.role !== "owner") || (workspace.role === "admin" && member.role !== "owner")}
                  <select
                    aria-label="Change role for @{member.username}"
                    value={member.role}
                    onchange={(e) =>
                      handleRoleChange(member, e.currentTarget.value)}
                  >
                    <option value="admin">Admin</option>
                    <option value="member">Member</option>
                    <option value="guest">Guest</option>
                  </select>
                {:else}
                  <span class="role-badge role-{member.role}"
                    >{member.role}</span
                  >
                {/if}
              </td>
              <td>
                {#if member.user_id === currentUserId}
                  {#if member.role !== "owner"}
                    <button
                      type="button"
                      class="remove-btn"
                      onclick={() => handleRemoveMember(member)}
                    >
                      Leave
                    </button>
                  {:else}
                    <span class="note">Sole owner</span>
                  {/if}
                {:else if workspace.role === "owner" || (workspace.role === "admin" && member.role !== "owner")}
                  <button
                    type="button"
                    class="remove-btn"
                    onclick={() => handleRemoveMember(member)}
                  >
                    Remove
                  </button>
                {/if}
              </td>
            </tr>
          {/each}
        </tbody>
      </table>
    </div>
  {/if}

  {#if workspace.role === "owner" || workspace.role === "admin"}
    <div class="add-member-card">
      <h3>Add a member</h3>
      <p class="note">Add an existing user account to this workspace.</p>
      <form onsubmit={handleAddMember} class="add-member-form">
        <InputField
          id="add-username"
          label="Username"
          hint="User must already be provisioned on this instance."
          bind:value={addUsername}
          required
        />
        <div class="field">
          <label for="add-role">Role</label>
          <select id="add-role" bind:value={addRole}>
            <option value="admin">Admin</option>
            <option value="member">Member</option>
            <option value="guest">Guest</option>
          </select>
        </div>
        <Button type="submit" disabled={adding || !addUsername.trim()}>
          {adding ? "Adding..." : "Add to workspace"}
        </Button>
      </form>
    </div>
  {/if}

  {#if showTransferModal}
    <div
      class="modal-backdrop"
      role="dialog"
      aria-modal="true"
      aria-labelledby="transfer-title"
    >
      <div class="modal-card">
        <h3 id="transfer-title">Transfer workspace ownership</h3>
        <p class="note">
          Transfer ownership of <strong>{workspace.name}</strong> to another
          member. You will be assigned the <strong>Admin</strong> role. This requires
          fresh password confirmation.
        </p>

        {#if transferError}
          <Feedback tone="danger">{transferError}</Feedback>
        {/if}

        <form onsubmit={handleTransferOwnership}>
          <div class="field">
            <label for="transfer-target">New owner</label>
            <select id="transfer-target" bind:value={transferTargetId} required>
              <option value="" disabled selected>Select a member</option>
              {#each members.filter((m) => m.user_id !== currentUserId) as member (member.user_id)}
                <option value={member.user_id}>
                  {member.display_name} (@{member.username}) - {member.role}
                </option>
              {/each}
            </select>
          </div>

          <InputField
            id="transfer-password"
            label="Your password confirmation"
            type="password"
            hint="Re-authenticate to confirm ownership transfer."
            bind:value={transferPassword}
            required
          />

          <div class="modal-actions">
            <button
              type="button"
              class="cancel-btn"
              onclick={() => {
                showTransferModal = false;
                transferPassword = "";
                transferError = null;
              }}
            >
              Cancel
            </button>
            <Button
              type="submit"
              disabled={transferring || !transferTargetId || !transferPassword}
            >
              {transferring ? "Transferring..." : "Confirm transfer"}
            </Button>
          </div>
        </form>
      </div>
    </div>
  {/if}
</section>

<style>
  .members-section {
    margin-top: var(--space-6);
  }
  .header-row {
    display: flex;
    justify-content: space-between;
    align-items: center;
    flex-wrap: wrap;
    gap: var(--space-4);
    margin-bottom: var(--space-4);
  }
  .table-wrapper {
    overflow-x: auto;
    margin-bottom: var(--space-6);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
    border: var(--border-width) solid var(--border);
  }
  .members-table {
    width: 100%;
    border-collapse: collapse;
    background: var(--surface-raised);
  }
  .members-table th,
  .members-table td {
    padding: var(--space-3) var(--space-4);
    text-align: left;
    border-bottom: var(--border-width) solid var(--border);
  }
  .members-table tr:last-child td {
    border-bottom: none;
  }
  .members-table th {
    background: var(--surface);
    font-size: 0.75rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.05em;
    color: var(--text-muted);
  }
  .member-info {
    display: flex;
    flex-direction: column;
    gap: 0.125rem;
  }
  .member-name {
    font-weight: 650;
    color: var(--text);
  }
  .member-handle {
    font-size: 0.8125rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
  }
  .role-badge {
    display: inline-block;
    padding: 0.2rem 0.5rem;
    font-size: 0.75rem;
    border-radius: var(--radius-full);
    text-transform: capitalize;
    font-weight: 600;
  }
  .role-owner {
    background: rgba(79, 70, 229, 0.1);
    color: var(--accent);
    border: var(--border-width) solid var(--accent);
  }
  .role-admin {
    background: rgba(14, 165, 233, 0.1);
    color: var(--focus);
    border: var(--border-width) solid var(--border);
  }
  .role-member {
    background: var(--surface);
    color: var(--text-secondary);
    border: var(--border-width) solid var(--border);
  }
  .role-guest {
    background: var(--surface);
    color: var(--text-muted);
    border: var(--border-width) solid var(--border);
  }
  .remove-btn {
    color: var(--danger);
    background: transparent;
    border: var(--border-width) solid var(--danger);
    padding: var(--space-1) var(--space-3);
    font-size: var(--text-small);
    border-radius: var(--radius);
    cursor: pointer;
    min-height: 2.75rem;
  }
  .remove-btn:hover {
    background: rgba(239, 68, 68, 0.1);
  }
  .transfer-btn {
    background: transparent;
    border: var(--border-width) solid var(--accent);
    color: var(--accent);
    padding: var(--space-2) var(--space-3);
    font-size: var(--text-small);
    font-weight: 600;
    border-radius: var(--radius);
    cursor: pointer;
    min-height: 2.75rem;
  }
  .transfer-btn:hover {
    background: var(--selected);
  }
  .add-member-card {
    background: var(--surface-raised);
    padding: var(--space-6);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
    margin-top: var(--space-6);
  }
  .add-member-form {
    display: grid;
    gap: var(--space-3);
    margin-top: var(--space-3);
    max-width: 28rem;
  }
  .modal-backdrop {
    position: fixed;
    inset: 0;
    background: var(--drawer-scrim);
    backdrop-filter: blur(8px);
    display: flex;
    justify-content: center;
    align-items: center;
    z-index: 100;
    padding: var(--space-4);
  }
  .modal-card {
    background: var(--surface-raised);
    padding: var(--space-6);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius-lg);
    max-width: 32rem;
    width: 100%;
    box-shadow: var(--shadow-modal);
  }
  .modal-actions {
    display: flex;
    justify-content: flex-end;
    gap: var(--space-3);
    margin-top: var(--space-4);
  }
  .cancel-btn {
    background: transparent;
    border: var(--border-width) solid var(--border);
    padding: var(--space-2) var(--space-4);
    border-radius: var(--radius);
    cursor: pointer;
    min-height: 2.75rem;
  }
</style>
