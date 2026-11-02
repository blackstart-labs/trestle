<script lang="ts">
  import { onMount } from "svelte";
  import Readiness from "./Readiness.svelte";
  import Appearance from "./Appearance.svelte";
  import SignIn, { type User } from "./auth/SignIn.svelte";
  import WorkspaceChooser, {
    type Workspace,
  } from "./auth/WorkspaceChooser.svelte";
  import WorkspaceMembers from "./auth/WorkspaceMembers.svelte";
  import ProjectsView, { type Project } from "./projects/ProjectsView.svelte";
  import TasksView from "./projects/TasksView.svelte";
  import Button from "./ui/Button.svelte";

  let user = $state<User | null>(null);
  let selectedWorkspace = $state<Workspace | null>(null);
  let selectedProject = $state<Project | null>(null);
  let workspaceTab = $state<"projects" | "members">("projects");
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
          if (activeTab === "signin" || activeTab === "overview") {
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
    selectedProject = null;
    workspaceTab = "projects";
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
      <div class="brand-icon" aria-hidden="true">T</div>
      <span class="wordmark">Trestle<span aria-hidden="true">.</span></span>
      <span class="build-label">Development build</span>
    </div>

    {#if user && selectedWorkspace}
      <div class="breadcrumbs-bar" aria-label="Current location">
        <span class="breadcrumb-item ws-name">{selectedWorkspace.name}</span>
        <span class="breadcrumb-separator" aria-hidden="true">/</span>
        {#if selectedProject}
          <button
            type="button"
            class="breadcrumb-link"
            onclick={() => (selectedProject = null)}
          >
            Projects
          </button>
          <span class="breadcrumb-separator" aria-hidden="true">/</span>
          <span class="breadcrumb-item project-crumb"
            >{selectedProject.name}</span
          >
        {:else}
          <span class="breadcrumb-item current-view">
            {workspaceTab === "projects" ? "Projects" : "Members"}
          </span>
        {/if}
      </div>
    {/if}

    <div class="user-area">
      {#if user}
        <div class="user-profile">
          <div class="user-avatar-circle" aria-hidden="true">
            {user.display_name.slice(0, 1).toUpperCase()}
          </div>
          <span
            class="user-badge"
            aria-label="Signed in as {user.display_name}"
          >
            <span class="user-name">{user.display_name}</span>
            <span class="user-handle">@{user.username}</span>
          </span>
        </div>
        <button type="button" class="signout-button" onclick={signOut}>
          Sign out
        </button>
      {:else if !checkingAuth}
        <a
          class="signin-nav-link"
          href="#signin"
          onclick={() => (activeTab = "signin")}
        >
          Sign in
        </a>
      {/if}
    </div>
  </header>

  <nav aria-label="Application">
    <p class="eyebrow nav-section-title">
      {user ? "Workspace" : "This instance"}
    </p>
    {#if user}
      <a
        href="#workspaces"
        class:selected={activeTab === "workspaces"}
        onclick={() => (activeTab = "workspaces")}
      >
        Workspaces
      </a>
    {:else}
      <a
        href="#signin"
        class:selected={activeTab === "signin"}
        onclick={() => (activeTab = "signin")}
      >
        Sign in
      </a>
    {/if}

    <a
      href="#overview"
      class:selected={activeTab === "overview"}
      onclick={() => (activeTab = "overview")}
    >
      Overview
    </a>

    <p class="eyebrow nav-section-title system-divider">Preferences</p>

    <a href="#appearance" onclick={() => (activeTab = "overview")}>
      Appearance
    </a>

    <a href="#instance-status" onclick={() => (activeTab = "overview")}>
      Instance status
    </a>
  </nav>

  <main id="main" tabindex="-1">
    {#if activeTab === "signin" && !user}
      <SignIn onsuccess={handleLoginSuccess} />
    {:else if activeTab === "workspaces" && user}
      {#if !selectedWorkspace}
        <WorkspaceChooser onselect={handleWorkspaceSelect} />
      {:else}
        <section class="active-workspace-card" aria-label="Selected workspace">
          <div class="workspace-header-layout">
            <div class="ws-title-group">
              <div class="ws-avatar-sm" aria-hidden="true">
                {selectedWorkspace.name.slice(0, 1).toUpperCase()}
              </div>
              <div>
                <p class="eyebrow">Active workspace</p>
                <h3>{selectedWorkspace.name}</h3>
                <p class="note">
                  Slug: /{selectedWorkspace.slug} · Role: {selectedWorkspace.role}
                </p>
              </div>
            </div>

            <div class="workspace-header-actions">
              <div
                class="workspace-subnav"
                role="tablist"
                aria-label="Workspace views"
              >
                <button
                  type="button"
                  role="tab"
                  aria-selected={workspaceTab === "projects"}
                  class="subnav-btn"
                  class:active={workspaceTab === "projects"}
                  onclick={() => (workspaceTab = "projects")}
                >
                  Projects & Tasks
                </button>
                <button
                  type="button"
                  role="tab"
                  aria-selected={workspaceTab === "members"}
                  class="subnav-btn"
                  class:active={workspaceTab === "members"}
                  onclick={() => (workspaceTab = "members")}
                >
                  Members
                </button>
              </div>
              <button
                type="button"
                class="switch-ws-btn"
                onclick={() => (selectedWorkspace = null)}
              >
                Switch workspace
              </button>
            </div>
          </div>
        </section>

        {#if workspaceTab === "projects"}
          {#if selectedProject}
            <TasksView
              workspace={selectedWorkspace}
              project={selectedProject}
              onback={() => (selectedProject = null)}
            />
          {:else}
            <ProjectsView
              workspace={selectedWorkspace}
              onselectproject={(proj) => (selectedProject = proj)}
            />
          {/if}
        {:else if workspaceTab === "members"}
          <WorkspaceMembers
            workspace={selectedWorkspace}
            currentUserId={user.id}
            onleave={() => {
              selectedWorkspace = null;
              selectedProject = null;
              void checkSession();
            }}
          />
        {/if}
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
    gap: var(--space-3);
  }

  .breadcrumbs-bar {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    font-size: 0.875rem;
    color: var(--text-secondary);
  }

  .breadcrumb-item {
    font-weight: 600;
    color: var(--text);
  }

  .breadcrumb-link {
    background: transparent;
    border: none;
    color: var(--accent);
    padding: 0;
    min-height: auto;
    font-size: inherit;
    font-weight: 600;
    cursor: pointer;
    text-decoration: underline;
  }

  .breadcrumb-separator {
    color: var(--text-muted);
  }

  .user-area {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  .user-profile {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }

  .user-avatar-circle {
    width: 2rem;
    height: 2rem;
    border-radius: var(--radius-full);
    background: linear-gradient(135deg, #4f46e5 0%, #7c3aed 100%);
    color: #ffffff;
    font-weight: 700;
    font-size: 0.8125rem;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .user-badge {
    display: flex;
    flex-direction: column;
    text-align: right;
  }

  .user-name {
    font-weight: 650;
    font-size: 0.8125rem;
    color: var(--text);
    line-height: 1.2;
  }

  .user-handle {
    font-size: 0.75rem;
    color: var(--text-muted);
    font-family: var(--font-mono);
  }

  .signout-button {
    margin-bottom: 0;
    min-height: 2.75rem;
    padding: var(--space-1) var(--space-3);
    font-size: var(--text-small);
    border-radius: var(--radius);
    background: var(--surface);
    color: var(--text);
    border: var(--border-width) solid var(--border);
  }

  .signout-button:hover {
    background: var(--selected);
    color: var(--danger);
  }

  .signin-nav-link {
    font-weight: 650;
    text-decoration: none;
    padding: var(--space-2) var(--space-4);
    min-height: 2.75rem;
    border-radius: var(--radius);
    background: var(--button-bg);
    color: var(--button-text);
    display: inline-flex;
    align-items: center;
  }

  .system-divider {
    margin-top: var(--space-4);
    padding-top: var(--space-2);
    border-top: var(--border-width) solid var(--border);
  }

  .active-workspace-card {
    margin-bottom: var(--space-6);
    padding: var(--space-4) var(--space-5);
    background: var(--surface-raised);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius-lg);
    box-shadow: var(--shadow-card);
  }

  .workspace-header-layout {
    display: flex;
    justify-content: space-between;
    align-items: center;
    gap: var(--space-4);
    flex-wrap: wrap;
  }

  .ws-title-group {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  .ws-avatar-sm {
    width: 2.5rem;
    height: 2.5rem;
    border-radius: var(--radius-md);
    background: linear-gradient(135deg, #4f46e5 0%, #7c3aed 100%);
    color: #ffffff;
    font-weight: 700;
    font-size: 1.125rem;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .ws-title-group h3 {
    margin: 0 0 0.125rem 0;
    font-size: 1.25rem;
    color: var(--text);
  }

  .workspace-header-actions {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    flex-wrap: wrap;
  }

  .workspace-subnav {
    display: flex;
    gap: var(--space-2);
  }

  .subnav-btn {
    background: var(--surface);
    border: var(--border-width) solid var(--border);
    border-radius: var(--radius);
    padding: var(--space-2) var(--space-3);
    font-size: 0.875rem;
    font-weight: 600;
    cursor: pointer;
    color: var(--text-secondary);
    min-height: 2.75rem;
  }

  .subnav-btn:hover {
    color: var(--text);
    background: var(--selected);
  }

  .subnav-btn.active {
    background: var(--button-bg);
    color: var(--button-text);
    border-color: var(--button-bg);
  }

  .switch-ws-btn {
    background: transparent;
    border: var(--border-width) solid var(--border);
    color: var(--text-secondary);
    font-size: 0.8125rem;
    font-weight: 600;
    border-radius: var(--radius);
    padding: var(--space-2) var(--space-3);
    min-height: 2.75rem;
    cursor: pointer;
  }

  .switch-ws-btn:hover {
    background: var(--selected);
    color: var(--text);
  }
</style>
