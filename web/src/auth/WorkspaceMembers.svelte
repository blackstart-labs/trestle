<script lang="ts">
  import { onMount } from "svelte";
  import Button from "../ui/Button.svelte";
  import InputField from "../ui/InputField.svelte";
  import Feedback from "../ui/Feedback.svelte";
  import Icon from "../ui/Icon.svelte";
  import Modal from "../ui/Modal.svelte";
  import Toast from "../ui/Toast.svelte";
  import Select from "../ui/Select.svelte";
  import Avatar from "../ui/Avatar.svelte";
  import { identityHue, initials } from "../lib/identity";

  export interface Member {
    user_id: string;
    username: string;
    display_name: string;
    role: "owner" | "admin" | "member" | "guest";
    avatar_url?: string | null;
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

  const canAdd = $derived(
    workspace.role === "owner" || workspace.role === "admin",
  );

  const roleOptions = [
    {
      value: "admin",
      label: "Admin",
      description: "Manages projects, workflows and members.",
    },
    {
      value: "member",
      label: "Member",
      description: "Works on projects and their tasks.",
    },
    {
      value: "guest",
      label: "Guest",
      description: "Sees only what is explicitly shared.",
    },
  ];

  const roleHelp: Record<"admin" | "member" | "guest", string> = {
    admin: "Manages projects, workflows and members.",
    member: "Works on projects and their tasks.",
    guest: "Sees only what is explicitly shared.",
  };

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

<section class="members" aria-labelledby="members-heading">
  <div class="page-head">
    <div>
      <h2 id="members-heading" tabindex="-1">Members of {workspace.name}</h2>
      <p>Manage workspace access, roles and ownership.</p>
    </div>
    {#if workspace.role === "owner" && members.length > 1}
      <div class="page-actions">
        <button type="button" onclick={() => (showTransferModal = true)}>
          <Icon name="user" size={16} />
          Transfer ownership
        </button>
      </div>
    {/if}
  </div>

  <div class="stack messages">
    {#if errorMessage}
      <Feedback tone="danger">{errorMessage}</Feedback>
    {/if}
  </div>

  <Toast message={successMessage} onclear={() => (successMessage = null)} />

  <div class="members-layout" class:with-form={canAdd}>
    <div class="panel table-panel">
      <div class="panel-head">
        <h3 class="panel-title">People</h3>
        {#if !loading}
          <span class="note num"
            >{members.length}
            {members.length === 1 ? "member" : "members"}</span
          >
        {/if}
      </div>

      {#if loading}
        <p class="note loading-note">Loading members…</p>
      {:else}
        <table class="members-table" aria-label="Workspace members">
          <thead>
            <tr>
              <th scope="col">Member</th>
              <th scope="col">Role</th>
              <th scope="col"><span class="visually-hidden">Actions</span></th>
            </tr>
          </thead>
          <tbody>
            {#each members as member (member.user_id)}
              <tr>
                <td>
                  <div class="member-info">
                    <Avatar
                      id={member.user_id}
                      name={member.display_name}
                      avatarUrl={member.avatar_url}
                    />
                    <span class="member-text">
                      <span class="member-name">
                        <span>{member.display_name}</span>
                        {#if member.user_id === currentUserId}
                          <span class="you">You</span>
                        {/if}
                      </span>
                      <span class="member-handle">@{member.username}</span>
                    </span>
                  </div>
                </td>
                <td>
                  {#if (workspace.role === "owner" && member.role !== "owner") || (workspace.role === "admin" && member.role !== "owner")}
                    <span class="role-select">
                      <Select
                        variant="chip"
                        label="Change role for @{member.username}"
                        value={member.role}
                        options={roleOptions}
                        onchange={(v) => handleRoleChange(member, v)}
                      />
                    </span>
                  {:else}
                    <span class="pill role-badge" data-role={member.role}
                      >{member.role}</span
                    >
                  {/if}
                </td>
                <td class="actions-cell">
                  {#if member.user_id === currentUserId}
                    {#if member.role !== "owner"}
                      <button
                        type="button"
                        class="btn-danger-quiet"
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
                      class="btn-danger-quiet"
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
      {/if}
    </div>

    {#if canAdd}
      <div class="panel panel-pad add-member">
        <h3>Add a member</h3>
        <p class="note">Add an existing user account to this workspace.</p>
        <form onsubmit={handleAddMember} class="form-grid">
          <InputField
            id="add-username"
            label="Username"
            hint="User must already be provisioned on this instance."
            bind:value={addUsername}
            required
          />
          <div class="field">
            <label for="add-role">Role</label>
            <Select
              id="add-role"
              value={addRole}
              options={roleOptions}
              onchange={(v) => (addRole = v as typeof addRole)}
            />
            <p class="note">{roleHelp[addRole]}</p>
          </div>
          <Button
            type="submit"
            variant="primary"
            disabled={adding || !addUsername.trim()}
          >
            {adding ? "Adding…" : "Add to workspace"}
          </Button>
        </form>
      </div>
    {/if}
  </div>

  {#if showTransferModal}
    <Modal
      title="Transfer workspace ownership"
      onclose={() => {
        if (transferring) return;
        showTransferModal = false;
        transferPassword = "";
        transferError = null;
      }}
    >
      <p class="note transfer-note">
        Transfer ownership of <strong>{workspace.name}</strong> to another
        member. You will be assigned the <strong>Admin</strong> role. This requires
        fresh password confirmation.
      </p>

      {#if transferError}
        <Feedback tone="danger">{transferError}</Feedback>
      {/if}

      <form onsubmit={handleTransferOwnership} class="form-grid">
        <div class="field">
          <label for="transfer-target">New owner</label>
          <Select
            id="transfer-target"
            value={transferTargetId}
            placeholder="Select a member"
            options={members
              .filter((m) => m.user_id !== currentUserId)
              .map((m) => ({
                value: m.user_id,
                label: m.display_name,
                hint: `@${m.username} · ${m.role}`,
              }))}
            onchange={(v) => (transferTargetId = v)}
          />
        </div>

        <InputField
          id="transfer-password"
          label="Your password confirmation"
          type="password"
          hint="Re-authenticate to confirm ownership transfer."
          bind:value={transferPassword}
          required
        />

        <div class="dialog-actions">
          <button
            type="button"
            class="btn-quiet"
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
            variant="primary"
            disabled={transferring || !transferTargetId || !transferPassword}
          >
            {transferring ? "Transferring…" : "Confirm transfer"}
          </Button>
        </div>
      </form>
    </Modal>
  {/if}
</section>

<style>
  .messages {
    margin-bottom: var(--space-4);
  }

  .messages:empty {
    display: none;
  }

  .members-layout {
    display: grid;
    gap: var(--space-6);
    align-items: start;
  }

  .members-layout.with-form {
    grid-template-columns: minmax(0, 1fr) minmax(18rem, 22rem);
  }

  .table-panel {
    overflow: hidden;
  }

  .loading-note {
    padding: var(--space-5);
  }

  .members-table {
    width: 100%;
    border-collapse: collapse;
  }

  .members-table th {
    padding: var(--space-2) var(--space-5);
    text-align: left;
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--text-muted);
    background: var(--surface-sunken);
    border-bottom: var(--border-width) solid var(--line);
  }

  .members-table td {
    padding: var(--space-3) var(--space-5);
    border-bottom: var(--border-width) solid var(--line);
    vertical-align: middle;
  }

  .members-table tr:last-child td {
    border-bottom: 0;
  }

  .members-table tbody tr {
    transition: background-color var(--transition-fast);
  }

  .members-table tbody tr:hover {
    background: var(--hover);
  }

  .member-info {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    min-width: 0;
  }

  .member-text {
    display: grid;
    min-width: 0;
    line-height: 1.3;
  }

  .member-name {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    font-weight: 580;
    color: var(--text);
  }

  .you {
    padding: 0.0625rem 0.4rem;
    border-radius: var(--radius-xs);
    background: var(--accent-soft);
    color: var(--accent);
    font-size: 0.6875rem;
    font-weight: 650;
  }

  .member-handle {
    font-size: var(--text-small);
    color: var(--text-muted);
  }

  .role-select {
    display: block;
    width: 8.5rem;
  }

  .role-badge {
    text-transform: capitalize;
  }

  .role-badge[data-role="owner"] {
    background: var(--accent-soft);
    color: var(--accent);
  }

  .actions-cell {
    text-align: right;
    white-space: nowrap;
  }

  .add-member {
    display: grid;
    gap: var(--space-2);
    position: sticky;
    top: calc(var(--topbar-height) + var(--space-4));
  }

  .add-member h3 {
    font-size: var(--text-section);
  }

  .add-member form {
    margin-top: var(--space-3);
  }

  .transfer-note {
    font-size: var(--text-small);
    color: var(--text-secondary);
  }

  @media (max-width: 64rem) {
    .members-layout.with-form {
      grid-template-columns: minmax(0, 1fr);
    }

    .add-member {
      position: static;
    }
  }

  /* Phone: each member becomes a two-line card row. */
  @media (max-width: 40rem) {
    .members-table thead {
      display: none;
    }

    .members-table,
    .members-table tbody,
    .members-table tr,
    .members-table td {
      display: block;
    }

    .members-table tr {
      display: grid;
      grid-template-columns: minmax(0, 1fr) auto;
      gap: var(--space-2) var(--space-3);
      padding: var(--space-3) var(--space-4);
      border-bottom: var(--border-width) solid var(--line);
    }

    .members-table tr:last-child {
      border-bottom: 0;
    }

    .members-table td {
      padding: 0;
      border: 0;
    }

    .members-table td:first-child {
      grid-column: 1 / -1;
    }

    .actions-cell {
      align-self: center;
    }
  }
</style>
