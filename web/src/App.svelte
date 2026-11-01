<script lang="ts">
  import { onMount } from "svelte";
  import Readiness from "./Readiness.svelte";
  import Appearance from "./Appearance.svelte";
  import SignIn, { type User } from "./auth/SignIn.svelte";
  import WorkspaceChooser, {
    type Workspace,
  } from "./auth/WorkspaceChooser.svelte";
  import Button from "./ui/Button.svelte";

  let user = $state<User | null>(null);
  let selectedWorkspace = $state<Workspace | null>(null);
  let checkingAuth = $state(true);
  let activeTab = $state<
    "overview" | "workspaces" | "signin" | "appearance" | "instance-status"
  >("overview");

  function getCookie(name: string): string | null {
    if (typeof document === "undefined") return null;
    for (const part of document.cookie.split(";")) {
      const trimmed = part.trim();
      if (trimmed.startsWith(`${name}=`)) {
        return trimmed.slice(name.length + 1);
      }
    }
    return null;
  }

  async function checkSession() {
    try {
      const response = await fetch("/api/v1/me", {
        headers: { Accept: "application/json" },
        cache: "no-store",
      });
      if (response.status === 200) {
        const data = await response.json();
        if (data && typeof data.username === "string") {
          user = data as User;
          if (activeTab === "signin") {
            activeTab = "workspaces";
          }
        } else {
          user = null;
        }
      } else {
        user = null;
      }
    } catch {
      user = null;
    } finally {
      checkingAuth = false;
    }
  }

  async function signOut() {
    const csrfToken = getCookie("trestle-csrf");
    try {
      await fetch("/api/v1/session", {
        method: "DELETE",
        headers: csrfToken ? { "X-CSRF-Token": csrfToken } : {},
      });
    } catch {
      // Ignore network errors on logout
    }
    user = null;
    selectedWorkspace = null;
    activeTab = "signin";
  }

  function handleLoginSuccess(authenticatedUser: User) {
    user = authenticatedUser;
    activeTab = "workspaces";
  }

  function handleWorkspaceSelect(ws: Workspace) {
    selectedWorkspace = ws;
  }

  onMount(() => {
    if (getCookie("trestle-csrf")) {
      void checkSession();
    } else {
      checkingAuth = false;
    }

    const handleHash = () => {
      const hash = window.location.hash.replace(/^#/, "");
      if (hash === "signin" || hash === "workspaces") {
        activeTab = hash;
      } else {
        activeTab = "overview";
      }
    };

    handleHash();
    window.addEventListener("hashchange", handleHash);
    return () => window.removeEventListener("hashchange", handleHash);
  });
</script>

<a class="skip-link" href="#main">Skip to content</a>
<div class="shell">
  <header class="masthead">
    <div class="brand">
      <span class="wordmark">Trestle<span aria-hidden="true">.</span></span>
      <span class="build-label">Development build</span>
    </div>
    <div class="user-area">
      {#if user}
        <span class="user-badge" aria-label="Signed in as {user.display_name}">
          <span class="user-name">{user.display_name}</span>
          <span class="user-handle">@{user.username}</span>
        </span>
        <button type="button" class="signout-button" onclick={signOut}
          >Sign out</button
        >
      {:else if !checkingAuth}
        <a
          class="signin-nav-link"
          href="#signin"
          onclick={() => (activeTab = "signin")}>Sign in</a
        >
      {/if}
    </div>
  </header>

  <nav aria-label="Application">
    <p class="eyebrow">{user ? "Workspace" : "This instance"}</p>
    {#if user}
      <a
        href="#workspaces"
        class:selected={activeTab === "workspaces"}
        onclick={() => (activeTab = "workspaces")}>Workspaces</a
      >
    {:else}
      <a
        href="#signin"
        class:selected={activeTab === "signin"}
        onclick={() => (activeTab = "signin")}>Sign in</a
      >
    {/if}
    <a
      href="#overview"
      class:selected={activeTab === "overview"}
      onclick={() => (activeTab = "overview")}>Overview</a
    >
    <a href="#appearance" onclick={() => (activeTab = "overview")}>Appearance</a
    >
    <a href="#instance-status" onclick={() => (activeTab = "overview")}
      >Instance status</a
    >
  </nav>

  <main id="main" tabindex="-1">
    {#if activeTab === "signin" && !user}
      <SignIn onsuccess={handleLoginSuccess} />
    {:else if activeTab === "workspaces" && user}
      <WorkspaceChooser onselect={handleWorkspaceSelect} />
      {#if selectedWorkspace}
        <section class="active-workspace-card" aria-label="Selected workspace">
          <p class="eyebrow">Active selection</p>
          <h3>{selectedWorkspace.name}</h3>
          <p class="note">
            Slug: /{selectedWorkspace.slug} · Role: {selectedWorkspace.role}
          </p>
        </section>
      {/if}
    {:else}
      <section aria-labelledby="overview">
        <p class="eyebrow">Engineering foundation</p>
        <h1 id="overview" tabindex="-1">Trestle</h1>
        <p class="lead">
          One place for projects, time and connected knowledge.
        </p>
        <p>
          {#if user}
            You are signed in as <strong>{user.display_name}</strong>. Access
            your workspaces from the navigation.
          {:else}
            Identity and workspace management are enabled. Sign in to access
            your workspaces.
          {/if}
        </p>
      </section>
      <Appearance />
      <Readiness />
    {/if}
  </main>

  <footer>
    <span>A self-hosted workspace, under development.</span>
    <a href="/assets/third-party-licenses.txt">Frontend third-party licenses</a>
  </footer>
</div>

<style>
  .brand {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    flex-wrap: wrap;
  }
  .user-area {
    display: flex;
    align-items: center;
    gap: var(--space-4);
  }
  .user-badge {
    display: flex;
    flex-direction: column;
    text-align: right;
  }
  .user-name {
    font-weight: 650;
    font-size: var(--text-small);
  }
  .user-handle {
    font-size: var(--text-small);
    color: var(--text-secondary);
    font-family: var(--font-mono);
  }
  .signout-button {
    margin-bottom: 0;
    min-height: calc(var(--control-height) - 0.5rem);
    padding: var(--space-1) var(--space-3);
    font-size: var(--text-small);
  }
  .signin-nav-link {
    font-weight: 650;
    text-decoration: none;
  }
  nav a.selected {
    background: var(--selected);
    font-weight: 650;
  }
  .active-workspace-card {
    margin-top: var(--space-6);
    padding: var(--space-4);
    background: var(--surface-raised);
    border: var(--border-width) solid var(--accent);
    border-radius: var(--radius);
  }
</style>
