<script lang="ts">
  import { onMount, tick } from "svelte";
  import Overview from "./Overview.svelte";
  import Settings from "./Settings.svelte";
  import { withViewTransition } from "./lib/motion";
  import SignIn, { type User } from "./auth/SignIn.svelte";
  import WorkspaceChooser, {
    type Workspace,
  } from "./auth/WorkspaceChooser.svelte";
  import WorkspaceMembers from "./auth/WorkspaceMembers.svelte";
  import ProjectsView, { type Project } from "./projects/ProjectsView.svelte";
  import TasksView from "./projects/TasksView.svelte";
  import Dashboard from "./projects/Dashboard.svelte";
  import Icon from "./ui/Icon.svelte";
  import BrandMark from "./ui/BrandMark.svelte";
  import CommandPalette, { type Command } from "./ui/CommandPalette.svelte";
  import Toast from "./ui/Toast.svelte";
  import { accents, preferences } from "./lib/preferences.svelte";
  import { liveTimer } from "./lib/timer.svelte";
  import { identityHue, initials } from "./lib/identity";

  let user = $state<User | null>(null);
  let selectedWorkspace = $state<Workspace | null>(null);
  let selectedProject = $state<Project | null>(null);
  let openTaskId = $state<string | null>(null);
  let workspaceTab = $state<"dashboard" | "projects" | "members">("dashboard");
  let checkingAuth = $state(true);
  let settingsSection = $state("");
  // Skip the chooser on arrival; an explicit switch always shows it.
  let autoOpenWorkspace = $state(true);
  let activeTab = $state<"overview" | "workspaces" | "signin" | "settings">(
    "overview",
  );

  // Command palette
  let paletteOpen = $state(false);
  let paletteProjects = $state<Project[]>([]);
  let paletteLoading = $state(false);
  let paletteReturnFocus: HTMLElement | null = null;
  let toastMessage = $state<string | null>(null);

  function navigateToActiveTask() {
    if (!liveTimer.active) return;
    route(
      `task/${liveTimer.active.workspaceId}/${liveTimer.active.projectId}/${liveTimer.active.taskId}`,
    );
  }

  async function stopActiveTimer() {
    const token = getCookie("trestle-csrf");
    const res = await liveTimer.stopAndLog(token || "");
    if (res.ok) {
      toastMessage = `Logged ${res.spentHours}h to task`;
    } else {
      toastMessage = res.error || "Failed to log time";
    }
  }

  const isMac =
    typeof navigator !== "undefined" &&
    /Mac|iPhone|iPad/.test(navigator.platform || navigator.userAgent);

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

  const LAST_WS_KEY = "trestle.workspace";

  async function ensureWorkspaceSelected() {
    if (selectedWorkspace) return;
    try {
      const res = await fetch("/api/v1/workspaces", {
        headers: { Accept: "application/json" },
        cache: "no-store",
      });
      if (res.ok) {
        const data = await res.json();
        const workspaces: Workspace[] = data.workspaces || [];
        if (workspaces.length === 0) return;
        const lastId = (() => {
          try {
            return localStorage.getItem(LAST_WS_KEY);
          } catch {
            return null;
          }
        })();
        const match = workspaces.find((w) => w.id === lastId);
        selectedWorkspace = match || workspaces[0];
      }
    } catch {
      // safe fallback
    }
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
          void ensureWorkspaceSelected();
          const currentHash = window.location.hash.replace(/^#/, "");
          if (
            currentHash.startsWith("task/") ||
            currentHash.startsWith("workspaces/")
          ) {
            route(currentHash);
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
    selectedProject = null;
    activeTab = "signin";
  }

  // Every change of view runs through one place, so the old page can
  // cross-fade into the new one while the shell stays still.
  function go(update: () => void) {
    withViewTransition(async () => {
      update();
      await tick();
    });
  }

  function handleLoginSuccess(authenticatedUser: User) {
    go(() => {
      user = authenticatedUser;
      autoOpenWorkspace = true;
      activeTab = "workspaces";
      void ensureWorkspaceSelected();
      const currentHash = window.location.hash.replace(/^#/, "");
      if (
        currentHash.startsWith("task/") ||
        currentHash.startsWith("workspaces/")
      ) {
        route(currentHash);
      }
    });
  }

  function handleWorkspaceSelect(ws: Workspace) {
    go(() => {
      selectedWorkspace = ws;
      selectedProject = null;
      workspaceTab = "dashboard";
    });
  }

  function openWorkspaceView(tab: "dashboard" | "projects" | "members") {
    go(() => {
      activeTab = "workspaces";
      workspaceTab = tab;
      if (tab !== "members") selectedProject = null;
    });
  }

  function openProject(project: Project, taskId?: string) {
    go(() => {
      activeTab = "workspaces";
      workspaceTab = "projects";
      openTaskId = taskId ?? null;
      selectedProject = project;
    });
  }

  function closeProject() {
    go(() => (selectedProject = null));
  }

  function switchWorkspace() {
    go(() => {
      autoOpenWorkspace = false;
      selectedWorkspace = null;
      selectedProject = null;
      activeTab = "workspaces";
    });
  }

  // Light ↔ dark eases across the whole screen instead of flashing.
  function toggleScheme() {
    const root = document.documentElement;
    root.dataset.vt = "theme";
    const doc = document as Document & {
      startViewTransition?: (cb: () => void) => {
        finished: Promise<void>;
        ready: Promise<void>;
      };
    };
    if (
      !doc.startViewTransition ||
      matchMedia("(prefers-reduced-motion: reduce)").matches
    ) {
      preferences.toggleScheme();
      delete root.dataset.vt;
      return;
    }
    const vt = doc.startViewTransition(() => preferences.toggleScheme());
    vt.ready.catch(() => {});
    vt.finished.catch(() => {}).finally(() => delete root.dataset.vt);
  }

  // Settings sections (#appearance, #instance-status) are deep links into
  // the one Settings page: render it, then focus the section's heading.
  const SETTINGS_SECTIONS = [
    "appearance",
    "instance-status",
    "integration",
    "workspace",
    "profile",
  ];

  async function focusSection(id: string) {
    await tick();
    const target = document.getElementById(id);
    target?.focus({ preventScroll: true });
  }

  async function openTaskFromRoute(
    wsId: string,
    projId: string,
    taskId: string,
  ) {
    activeTab = "workspaces";
    workspaceTab = "projects";
    openTaskId = taskId;

    try {
      const res = await fetch("/api/v1/workspaces", {
        headers: { Accept: "application/json" },
        cache: "no-store",
      });
      if (res.ok) {
        const data = await res.json();
        const foundWs = (data.workspaces || []).find(
          (w: Workspace) => w.id === wsId,
        );
        if (foundWs) selectedWorkspace = foundWs;
      }
    } catch {
      // safe fallback
    }

    try {
      const res = await fetch(`/api/v1/workspaces/${wsId}/projects/${projId}`, {
        headers: { Accept: "application/json" },
        cache: "no-store",
      });
      if (res.ok) {
        const proj = await res.json();
        selectedProject = proj;
      }
    } catch {
      // safe fallback
    }
  }

  function route(hash: string) {
    const taskMatch =
      hash.match(
        /^(?:task|workspaces)\/([^/]+)\/projects\/([^/]+)\/tasks\/([^/]+)$/,
      ) || hash.match(/^task\/([^/]+)\/([^/]+)\/([^/]+)$/);
    if (taskMatch) {
      void openTaskFromRoute(taskMatch[1], taskMatch[2], taskMatch[3]);
      return;
    }

    if (hash === "settings") {
      settingsSection = "";
      activeTab = "settings";
      void focusSection("settings-heading");
    } else if (hash === "signin" || hash === "workspaces") {
      activeTab = hash;
    } else if (SETTINGS_SECTIONS.includes(hash)) {
      settingsSection = hash;
      activeTab = "settings";
      void focusSection(hash);
    } else {
      activeTab = "overview";
    }
  }

  function navigateHash(hash: string) {
    if (window.location.hash === `#${hash}`) route(hash);
    else window.location.hash = hash;
  }

  async function openPalette() {
    if (paletteOpen) return;
    paletteReturnFocus = document.activeElement as HTMLElement | null;
    paletteOpen = true;
    if (selectedWorkspace) {
      paletteLoading = true;
      try {
        const res = await fetch(
          `/api/v1/workspaces/${selectedWorkspace.id}/projects`,
          { headers: { Accept: "application/json" }, cache: "no-store" },
        );
        if (res.ok) {
          const data = await res.json();
          paletteProjects = data.projects || [];
        }
      } catch {
        // The palette still offers navigation without project results.
      } finally {
        paletteLoading = false;
      }
    } else {
      paletteProjects = [];
    }
  }

  function closePalette() {
    paletteOpen = false;
    const target = paletteReturnFocus;
    paletteReturnFocus = null;
    void tick().then(() => target?.focus?.());
  }

  const commands = $derived.by(() => {
    const list: Command[] = [];
    if (user && selectedWorkspace) {
      const ws = selectedWorkspace;
      for (const p of paletteProjects) {
        list.push({
          id: `project-${p.id}`,
          group: `Projects in ${ws.name}`,
          label: p.name,
          hint: p.key,
          icon: "folder",
          keywords: `${p.slug} ${p.key}`,
          run: () => openProject(p),
        });
      }
      list.push(
        {
          id: "go-dashboard",
          group: "Go to",
          label: "Dashboard",
          icon: "grid",
          run: () => openWorkspaceView("dashboard"),
        },
        {
          id: "go-projects",
          group: "Go to",
          label: "All projects",
          icon: "layers",
          run: () => openWorkspaceView("projects"),
        },
        {
          id: "go-members",
          group: "Go to",
          label: "Members",
          icon: "users",
          run: () => openWorkspaceView("members"),
        },
        {
          id: "switch-workspace",
          group: "Go to",
          label: "Switch workspace",
          icon: "chevron-up-down",
          run: switchWorkspace,
        },
      );
    } else if (user) {
      list.push({
        id: "go-workspaces",
        group: "Go to",
        label: "Workspaces",
        icon: "layers",
        run: () => void navigateHash("workspaces"),
      });
    } else {
      list.push({
        id: "go-signin",
        group: "Go to",
        label: "Sign in",
        icon: "log-in",
        run: () => void navigateHash("signin"),
      });
    }
    list.push(
      {
        id: "go-overview",
        group: "Go to",
        label: "Overview",
        icon: "home",
        run: () => void navigateHash("overview"),
      },
      {
        id: "go-appearance",
        group: "Go to",
        label: "Settings: Appearance",
        icon: "palette",
        keywords: "theme accent dark light instance status health readiness",
        run: () => void navigateHash("appearance"),
      },
      {
        id: "go-integration",
        group: "Go to",
        label: "Settings: Integration",
        icon: "settings",
        keywords: "project telegram slack discord notifications webhook",
        run: () => void navigateHash("integration"),
      },
      {
        id: "toggle-scheme",
        group: "Preferences",
        label: preferences.resolvedDark
          ? "Switch to light theme"
          : "Switch to dark theme",
        icon: preferences.resolvedDark ? "sun" : "moon",
        keywords: "theme mode appearance",
        run: () => toggleScheme(),
      },
      {
        id: "theme-system",
        group: "Preferences",
        label: "Follow system theme",
        icon: "monitor",
        keywords: "theme mode appearance",
        run: () => preferences.setTheme("system"),
      },
    );
    for (const a of accents) {
      list.push({
        id: `accent-${a.value}`,
        group: "Preferences",
        label: `Accent colour: ${a.label}`,
        icon: "palette",
        hint: preferences.accent === a.value ? "Current" : undefined,
        keywords: "accent colour color theme",
        run: () => preferences.setAccent(a.value),
      });
    }
    if (user) {
      list.push({
        id: "sign-out",
        group: "Account",
        label: "Sign out",
        icon: "log-out",
        run: () => void signOut(),
      });
    }
    return list;
  });

  function handleGlobalKeydown(e: KeyboardEvent) {
    const key = e.key.toLowerCase();
    if ((e.metaKey || e.ctrlKey) && key === "k") {
      e.preventDefault();
      if (paletteOpen) closePalette();
      else void openPalette();
      return;
    }
    if (key === "/" && !paletteOpen) {
      const el = e.target as HTMLElement | null;
      const typing =
        el &&
        (el.isContentEditable ||
          ["INPUT", "TEXTAREA", "SELECT"].includes(el.tagName));
      if (!typing && !document.querySelector('[aria-modal="true"]')) {
        e.preventDefault();
        void openPalette();
      }
    }
  }

  const navIsWorkspace = $derived(activeTab === "workspaces");

  // One lit pill slides to the current destination instead of blinking.
  let navEl = $state<HTMLElement | null>(null);
  let indicator = $state<HTMLElement | null>(null);

  function placeIndicator(animate = true) {
    if (!navEl || !indicator) return;
    const current = navEl.querySelector<HTMLElement>("a.selected");
    if (!current) {
      indicator.style.opacity = "0";
      return;
    }
    if (!animate) indicator.style.transition = "none";
    indicator.style.opacity = "1";
    indicator.style.transform = `translate(${current.offsetLeft}px, ${current.offsetTop}px)`;
    indicator.style.width = `${current.offsetWidth}px`;
    indicator.style.height = `${current.offsetHeight}px`;
    if (!animate) {
      void indicator.offsetWidth;
      indicator.style.transition = "";
    }
  }

  $effect(() => {
    void activeTab;
    void workspaceTab;
    void selectedWorkspace;
    void user;
    void tick().then(() => placeIndicator());
  });

  $effect(() => {
    if (!navEl) return;
    if (typeof ResizeObserver === "undefined") return;
    const ro = new ResizeObserver(() => placeIndicator(false));
    ro.observe(navEl);
    return () => ro.disconnect();
  });

  // The tab title names where you are, so tabs and history stay legible.
  $effect(() => {
    let place = "Overview";
    if (activeTab === "signin") place = "Sign in";
    else if (activeTab === "settings") place = "Settings";
    else if (activeTab === "workspaces" && user) {
      if (!selectedWorkspace) place = "Workspaces";
      else if (workspaceTab === "members")
        place = `Members · ${selectedWorkspace.name}`;
      else if (workspaceTab === "dashboard")
        place = `Dashboard · ${selectedWorkspace.name}`;
      else if (selectedProject) place = selectedProject.name;
      else place = `Projects · ${selectedWorkspace.name}`;
    }
    document.title = `${place} — Trestle`;
  });

  onMount(() => {
    preferences.init();
    liveTimer.init();

    if (getCookie("trestle-csrf")) {
      void checkSession();
    } else {
      checkingAuth = false;
    }

    const handleHash = () =>
      go(() => route(window.location.hash.replace(/^#/, "")));

    handleHash();
    window.addEventListener("hashchange", handleHash);
    return () => window.removeEventListener("hashchange", handleHash);
  });
</script>

<svelte:window onkeydown={handleGlobalKeydown} />

<a class="skip-link" href="#main">Skip to content</a>
<div class="shell">
  <aside class="sidebar">
    <div class="sidebar-brand">
      <BrandMark />
      <span class="wordmark">Trestle</span>
      <span class="build-label">Dev build</span>
    </div>

    {#if user && selectedWorkspace}
      <div class="ws-switcher">
        <span
          class="avatar ws-avatar"
          style:--avatar-hue={identityHue(selectedWorkspace.id)}
          aria-hidden="true">{initials(selectedWorkspace.name)}</span
        >
        <span class="ws-text">
          <span class="ws-name">{selectedWorkspace.name}</span>
          <span class="ws-role">{selectedWorkspace.role}</span>
        </span>
        <button
          type="button"
          class="btn-quiet btn-icon ws-switch-btn"
          onclick={switchWorkspace}
          aria-label="Switch workspace"
          title="Switch workspace"
        >
          <Icon name="chevron-up-down" size={16} />
        </button>
      </div>
    {/if}

    <nav aria-label="Application" bind:this={navEl}>
      <p class="nav-group">{user ? "Workspace" : "This instance"}</p>
      {#if user}
        {#if selectedWorkspace}
          <a
            href="#workspaces"
            class:selected={navIsWorkspace && workspaceTab === "dashboard"}
            aria-current={navIsWorkspace && workspaceTab === "dashboard"
              ? "page"
              : undefined}
            onclick={() => openWorkspaceView("dashboard")}
          >
            <Icon name="grid" size={18} />
            <span>Dashboard</span>
          </a>
          <a
            href="#workspaces"
            class:selected={navIsWorkspace && workspaceTab === "projects"}
            aria-current={navIsWorkspace && workspaceTab === "projects"
              ? "page"
              : undefined}
            onclick={() => openWorkspaceView("projects")}
          >
            <Icon name="folder" size={18} />
            <span>Projects</span>
          </a>
          <a
            href="#workspaces"
            class:selected={navIsWorkspace && workspaceTab === "members"}
            aria-current={navIsWorkspace && workspaceTab === "members"
              ? "page"
              : undefined}
            onclick={() => openWorkspaceView("members")}
          >
            <Icon name="users" size={18} />
            <span>Members</span>
          </a>
        {:else}
          <a
            href="#workspaces"
            class:selected={activeTab === "workspaces"}
            aria-current={activeTab === "workspaces" ? "page" : undefined}
            onclick={() => (activeTab = "workspaces")}
          >
            <Icon name="layers" size={18} />
            <span>Workspaces</span>
          </a>
        {/if}
      {:else}
        <a
          href="#signin"
          class:selected={activeTab === "signin"}
          aria-current={activeTab === "signin" ? "page" : undefined}
          onclick={() => (activeTab = "signin")}
        >
          <Icon name="log-in" size={18} />
          <span>Sign in</span>
        </a>
      {/if}

      {#if !user}
        <a
          href="#overview"
          class:selected={activeTab === "overview"}
          aria-current={activeTab === "overview" ? "page" : undefined}
        >
          <Icon name="home" size={18} />
          <span>Overview</span>
        </a>
      {/if}

      <p class="nav-group">{user ? "Account" : "Preferences"}</p>
      <a
        href="#settings"
        class:selected={activeTab === "settings"}
        aria-current={activeTab === "settings" ? "page" : undefined}
      >
        <Icon name="settings" size={18} />
        <span>Settings</span>
      </a>
      <span class="nav-indicator" aria-hidden="true" bind:this={indicator}
      ></span>
    </nav>

    <div class="sidebar-foot">
      {#if user}
        <div class="command-card">
          <span class="tile tile-lg command-tile" aria-hidden="true">
            <Icon name="command" size={20} />
          </span>
          <p class="command-title">Jump anywhere</p>
          <p class="command-text">
            Open a project, switch views or change the theme from the keyboard.
          </p>
          <button type="button" class="command-btn" onclick={openPalette}>
            Open search
            <span class="search-keys" aria-hidden="true">
              <kbd>{isMac ? "⌘" : "Ctrl"}</kbd><kbd>K</kbd>
            </span>
          </button>
        </div>
      {/if}
      <p class="foot-note">Self-hosted. Your data stays on this instance.</p>
    </div>
  </aside>

  <div class="canvas">
    <header class="topbar">
      <div class="topbar-brand">
        <BrandMark size={28} />
        <span class="wordmark">Trestle</span>
      </div>

      {#if user && selectedWorkspace && activeTab === "workspaces"}
        <div class="breadcrumbs" aria-label="Current location">
          <span class="crumb crumb-ws">{selectedWorkspace.name}</span>
          <Icon name="chevron-right" size={14} class="crumb-sep" />
          {#if workspaceTab === "members"}
            <span class="crumb crumb-current">Members</span>
          {:else if workspaceTab === "dashboard"}
            <span class="crumb crumb-current">Dashboard</span>
          {:else if selectedProject}
            <button
              type="button"
              class="crumb crumb-link"
              onclick={closeProject}
            >
              Projects
            </button>
            <Icon name="chevron-right" size={14} class="crumb-sep" />
            <span class="crumb crumb-current">{selectedProject.name}</span>
          {:else}
            <span class="crumb crumb-current">Projects</span>
          {/if}
        </div>
      {/if}

      <div class="topbar-actions">
        {#if liveTimer.active}
          <div class="live-timer-banner" aria-label="Active live timer">
            <span class="live-pulse-dot" aria-hidden="true"></span>
            <button
              type="button"
              class="live-timer-task"
              onclick={navigateToActiveTask}
              title="{liveTimer.active.taskHumanKey}: {liveTimer.active
                .taskTitle}"
            >
              <span class="live-timer-key">{liveTimer.active.taskHumanKey}</span
              >
              <span class="live-timer-time num"
                >{liveTimer.formattedDuration}</span
              >
            </button>
            <button
              type="button"
              class="live-timer-btn live-timer-stop"
              onclick={stopActiveTimer}
              title="Stop and log time"
              aria-label="Stop timer"
            >
              <Icon name="stop" size={12} />
              <span>Stop</span>
            </button>
            <button
              type="button"
              class="live-timer-btn live-timer-discard"
              onclick={() => liveTimer.discard()}
              title="Discard timer"
              aria-label="Discard timer"
            >
              <Icon name="close" size={12} />
            </button>
          </div>
        {/if}

        <button
          type="button"
          class="search-trigger"
          onclick={openPalette}
          aria-label="Search and commands"
          aria-keyshortcuts={isMac ? "Meta+K" : "Control+K"}
        >
          <Icon name="search" size={16} />
          <span class="search-text">Search or jump to…</span>
          <span class="search-keys" aria-hidden="true">
            <kbd>{isMac ? "⌘" : "Ctrl"}</kbd><kbd>K</kbd>
          </span>
        </button>

        <button
          type="button"
          class="btn-icon icon-chip"
          onclick={toggleScheme}
          aria-label={preferences.resolvedDark
            ? "Switch to light theme"
            : "Switch to dark theme"}
          title={preferences.resolvedDark
            ? "Switch to light theme"
            : "Switch to dark theme"}
        >
          <Icon name={preferences.resolvedDark ? "sun" : "moon"} size={18} />
        </button>

        {#if user}
          <div class="user-chip">
            <span
              class="avatar"
              style:--avatar-hue={identityHue(user.id)}
              aria-hidden="true">{initials(user.display_name)}</span
            >
            <span
              class="user-badge"
              aria-label="Signed in as {user.display_name}"
            >
              <span class="user-name">{user.display_name}</span>
              <span class="user-handle">@{user.username}</span>
            </span>
          </div>
          <button
            type="button"
            class="btn-quiet signout-button"
            onclick={signOut}
            aria-label="Sign out"
            title="Sign out"
          >
            <Icon name="log-out" size={17} />
            <span class="signout-text">Sign out</span>
          </button>
        {:else if !checkingAuth && activeTab !== "signin"}
          <a
            class="btn btn-primary signin-nav-link"
            href="#signin"
            onclick={() => (activeTab = "signin")}
          >
            Sign in
          </a>
        {/if}
      </div>
    </header>

    <main id="main" tabindex="-1" class:dense={activeTab === "workspaces"}>
      {#if activeTab === "signin" && !user}
        <SignIn onsuccess={handleLoginSuccess} />
      {:else if activeTab === "workspaces" && user}
        {#if !selectedWorkspace}
          <WorkspaceChooser
            onselect={handleWorkspaceSelect}
            autoselect={autoOpenWorkspace}
          />
        {:else if workspaceTab === "dashboard"}
          <Dashboard
            workspace={selectedWorkspace}
            {user}
            onopenproject={openProject}
            onopenprojects={() => openWorkspaceView("projects")}
          />
        {:else if workspaceTab === "projects"}
          {#if selectedProject}
            {#key selectedProject.id}
              <TasksView
                workspace={selectedWorkspace}
                project={selectedProject}
                currentUserId={user.id}
                {openTaskId}
                onback={closeProject}
                onprojectupdated={(updated) => (selectedProject = updated)}
              />
            {/key}
          {:else}
            <ProjectsView
              workspace={selectedWorkspace}
              onselectproject={(proj) => openProject(proj)}
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
      {:else if activeTab === "settings"}
        <Settings
          section={settingsSection}
          workspace={selectedWorkspace}
          {user}
          signedIn={!!user}
          initialProjectId={selectedProject?.id || ""}
          onselectsection={(sec) => (settingsSection = sec)}
          onselectworkspace={(ws) => (selectedWorkspace = ws)}
          onworkspaceupdated={(ws) => {
            selectedWorkspace = ws;
          }}
          onuserupdated={(u) => (user = u)}
        />
      {:else}
        <Overview
          {user}
          onopenworkspaces={() => (activeTab = "workspaces")}
          onsignin={() => (activeTab = "signin")}
        />
      {/if}
    </main>
  </div>
</div>

{#if paletteOpen}
  <CommandPalette {commands} loading={paletteLoading} onclose={closePalette} />
{/if}

{#if toastMessage}
  <Toast message={toastMessage} onclear={() => (toastMessage = null)} />
{/if}

<style>
  .shell {
    min-height: 100svh;
    display: grid;
    grid-template-columns: var(--nav-width) minmax(0, 1fr);
  }

  /* ── Sidebar ─────────────────────────────────────────────────── */
  .sidebar {
    position: sticky;
    top: 0;
    height: 100svh;
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    padding: var(--space-4) var(--space-3);
    padding-left: max(var(--space-3), env(safe-area-inset-left));
    background: var(--surface-nav);
    border-right: var(--border-width) solid var(--line);
    overflow-y: auto;
  }

  .sidebar-brand {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-1) var(--space-2) var(--space-3);
  }

  .wordmark {
    font-size: 1.25rem;
    font-weight: 560;
    letter-spacing: -0.035em;
    color: var(--text);
  }

  .build-label {
    margin-left: auto;
    padding: 0.125rem 0.5rem;
    border-radius: var(--radius-full);
    border: var(--border-width) solid var(--line-strong);
    font-size: 0.6875rem;
    font-weight: 600;
    color: var(--text-muted);
    white-space: nowrap;
  }

  .ws-switcher {
    background:
      var(--inset-fill) padding-box,
      var(--inset-edge) border-box;
    border: var(--border-width) solid transparent;
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-2) var(--space-2) var(--space-2) var(--space-3);
    margin-bottom: var(--space-2);
    border-radius: var(--radius);
    box-shadow: var(--shadow-card);
  }

  .ws-avatar {
    border-radius: var(--radius-sm);
  }

  .ws-text {
    display: grid;
    min-width: 0;
    flex: 1;
    line-height: 1.25;
  }

  .ws-name {
    font-weight: 620;
    color: var(--text);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .ws-role {
    font-size: var(--text-xs);
    color: var(--text-muted);
    text-transform: capitalize;
  }

  .ws-switch-btn {
    color: var(--text-muted);
  }

  nav {
    display: flex;
    flex-direction: column;
    gap: 2px;
  }

  .nav-group {
    padding: var(--space-4) var(--space-3) var(--space-2);
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--text-muted);
  }

  nav a {
    position: relative;
    display: flex;
    align-items: center;
    gap: var(--space-3);
    min-height: var(--control-height);
    padding: 0 var(--space-3);
    border: var(--border-width) solid transparent;
    border-radius: var(--radius);
    color: var(--text-muted);
    font-size: var(--text-ui);
    font-weight: 500;
    text-decoration: none;
    white-space: nowrap;
    transition:
      background-color var(--transition-fast),
      color var(--transition-fast);
  }

  nav a :global(.svg-icon) {
    color: var(--text-muted);
    transition: color var(--transition-fast);
  }

  nav a:hover {
    background: var(--hover);
    color: var(--text);
  }

  nav {
    position: relative;
  }

  nav a {
    z-index: 1;
  }

  nav a.selected {
    color: var(--text);
    font-weight: 580;
  }

  nav a.selected:hover {
    background: transparent;
  }

  /* The lit pill behind the current destination; it glides between items. */
  .nav-indicator {
    position: absolute;
    top: 0;
    left: 0;
    z-index: 0;
    opacity: 0;
    border-radius: var(--radius);
    border: var(--border-width) solid transparent;
    background:
      var(--inset-fill) padding-box,
      var(--inset-edge) border-box;
    box-shadow: 0 6px 16px -10px oklch(0 0 0 / 0.5);
    pointer-events: none;
    transition:
      transform 420ms var(--ease-spring),
      width 420ms var(--ease-spring),
      opacity var(--duration) var(--ease-out);
    will-change: transform;
  }

  /* A lit marker on the current destination. */
  .nav-indicator::after {
    content: "";
    position: absolute;
    right: -0.75rem;
    top: 50%;
    width: 3px;
    height: 1.25rem;
    border-radius: 3px 0 0 3px;
    transform: translateY(-50%);
    background: var(--accent);
    box-shadow: 0 0 12px 1px var(--glow-accent);
  }

  nav a.selected :global(.svg-icon) {
    color: var(--accent);
  }

  .sidebar-foot {
    margin-top: auto;
    display: grid;
    gap: var(--space-4);
    padding: var(--space-4) 0 var(--space-2);
  }

  .foot-note {
    padding: 0 var(--space-3);
    font-size: var(--text-xs);
    color: var(--text-muted);
    line-height: 1.45;
  }

  /* The glowing card: Trestle's command palette, one click away. */
  .command-card {
    position: relative;
    display: grid;
    justify-items: center;
    gap: var(--space-2);
    padding: var(--space-5) var(--space-4) var(--space-4);
    text-align: center;
    border-radius: var(--radius-lg);
    border: var(--border-width) solid transparent;
    background:
      radial-gradient(
          120% 90% at 50% 120%,
          light-dark(
            oklch(0.82 calc(var(--brand-chroma) * 0.9) var(--brand-hue) / 0.55),
            oklch(0.55 calc(var(--brand-chroma) * 1.3) var(--brand-hue) / 0.55)
          ),
          transparent 70%
        )
        padding-box,
      var(--card-fill) padding-box,
      linear-gradient(
          180deg,
          light-dark(oklch(0.9 0.03 var(--brand-hue)), oklch(1 0 0 / 0.14)),
          light-dark(
            oklch(0.78 calc(var(--brand-chroma) * 0.8) var(--brand-hue)),
            oklch(0.7 calc(var(--brand-chroma) * 0.9) var(--brand-hue) / 0.5)
          )
        )
        border-box;
    overflow: hidden;
  }

  .command-tile {
    margin-bottom: var(--space-1);
    border-radius: 50%;
  }

  .command-title {
    font-size: var(--text-section);
    font-weight: 540;
    color: var(--text);
    letter-spacing: -0.012em;
  }

  .command-text {
    font-size: var(--text-xs);
    line-height: 1.45;
    color: var(--text-secondary);
  }

  .command-btn {
    width: 100%;
    margin-top: var(--space-2);
    background:
      linear-gradient(
          light-dark(oklch(1 0 0 / 0.75), oklch(1 0 0 / 0.08)),
          light-dark(oklch(1 0 0 / 0.55), oklch(1 0 0 / 0.04))
        )
        padding-box,
      linear-gradient(
          light-dark(oklch(1 0 0), oklch(1 0 0 / 0.18)),
          transparent
        )
        border-box;
    font-size: var(--text-small);
  }

  /* ── Canvas ──────────────────────────────────────────────────── */
  .canvas {
    position: relative;
    min-width: 0;
    display: flex;
    flex-direction: column;
    isolation: isolate;
  }

  /* The ambient light: one soft source over the top of the work area. */
  .canvas::before {
    content: "";
    position: absolute;
    inset: 0 0 auto 0;
    height: 34rem;
    z-index: -1;
    pointer-events: none;
    background:
      linear-gradient(
        112deg,
        transparent 44%,
        var(--sheen) 56%,
        transparent 66%
      ),
      radial-gradient(
        46rem 22rem at 80% -8rem,
        var(--glow-accent),
        transparent 72%
      ),
      radial-gradient(
        30rem 16rem at 22% -10rem,
        light-dark(transparent, oklch(0.6 0.06 var(--brand-hue) / 0.1)),
        transparent 70%
      );
    -webkit-mask-image: linear-gradient(
      180deg,
      #000 0%,
      #000 35%,
      transparent 100%
    );
    mask-image: linear-gradient(180deg, #000 0%, #000 35%, transparent 100%);
  }

  .topbar {
    position: sticky;
    top: 0;
    z-index: var(--layer-sticky);
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: var(--space-3);
    min-height: var(--topbar-height);
    padding: var(--space-2) var(--gutter);
    padding-right: max(var(--gutter), env(safe-area-inset-right));
    background: light-dark(
      oklch(0.972 var(--tint) var(--brand-hue) / 0.6),
      oklch(0.118 var(--tint) var(--brand-hue) / 0.45)
    );
    backdrop-filter: blur(14px) saturate(1.3);
    -webkit-backdrop-filter: blur(14px) saturate(1.3);
    border-bottom: var(--border-width) solid var(--line);
  }

  .topbar-brand {
    display: none;
    align-items: center;
    gap: var(--space-2);
  }

  .breadcrumbs {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    min-width: 0;
    font-size: var(--text-small);
    color: var(--text-muted);
  }

  .breadcrumbs :global(.crumb-sep) {
    color: var(--text-muted);
    opacity: 0.7;
  }

  .crumb {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .crumb-current {
    color: var(--text);
    font-weight: 600;
  }

  .crumb-link {
    min-height: 2rem;
    padding: 0 var(--space-2);
    margin: 0 calc(var(--space-2) * -1);
    border: 0;
    background: none;
    box-shadow: none;
    color: var(--text-secondary);
    font-size: inherit;
    font-weight: 540;
  }

  .crumb-link:hover:not(:disabled) {
    background: var(--hover);
    color: var(--text);
  }

  .topbar-actions {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    justify-content: flex-end;
    gap: var(--space-2);
    margin-left: auto;
    min-width: 0;
  }

  .search-trigger {
    justify-content: flex-start;
    width: clamp(12rem, 26vw, 22rem);
    padding: 0 var(--space-2) 0 var(--space-3);
    border-radius: var(--radius);
    color: var(--text-muted);
    font-weight: 460;
  }

  .search-trigger:hover:not(:disabled) {
    color: var(--text-secondary);
  }

  .icon-chip {
    border-radius: var(--radius);
    color: var(--text-secondary);
  }

  .search-text {
    flex: 1;
    text-align: left;
    white-space: nowrap;
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .search-keys {
    display: inline-flex;
    gap: 3px;
  }

  .user-chip {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding-left: var(--space-2);
    margin-left: var(--space-1);
    border-left: var(--border-width) solid var(--line);
  }

  .user-badge {
    display: grid;
    line-height: 1.2;
  }

  .user-name {
    font-weight: 600;
    font-size: var(--text-small);
    color: var(--text);
    white-space: nowrap;
  }

  .user-handle {
    font-size: var(--text-xs);
    color: var(--text-muted);
  }

  .signout-button:hover:not(:disabled) {
    color: var(--danger);
  }

  main {
    flex: 1;
    width: 100%;
    max-width: var(--content-width);
    margin: 0 auto;
    padding: var(--space-8) var(--gutter) var(--space-12);
    padding-left: max(var(--gutter), env(safe-area-inset-left));
    padding-right: max(var(--gutter), env(safe-area-inset-right));
    overflow-wrap: anywhere;
  }

  /* ── Tablet: tighter search, drop the handle ─────────────────── */
  @media (max-width: 72rem) {
    .user-badge {
      display: none;
    }
    .search-trigger {
      width: auto;
    }
    .search-text {
      display: none;
    }
  }

  /* ── Phone and narrow tablet: nav becomes a scrolling strip ──── */
  @media (max-width: 60rem) {
    .shell {
      grid-template-columns: minmax(0, 1fr);
      grid-template-rows: auto auto 1fr auto;
    }

    .sidebar {
      position: sticky;
      top: 0;
      z-index: var(--layer-sticky);
      height: auto;
      flex-direction: row;
      align-items: center;
      gap: var(--space-2);
      padding: var(--space-2) var(--gutter);
      overflow-x: auto;
      overflow-y: hidden;
      overscroll-behavior-x: contain;
      scrollbar-width: none;
      border-right: 0;
      border-bottom: var(--border-width) solid var(--line);
      order: 2;
    }

    .sidebar::-webkit-scrollbar {
      display: none;
    }

    .sidebar-brand,
    .sidebar-foot,
    .nav-group {
      display: none;
    }

    .ws-switcher {
      flex-shrink: 0;
      margin: 0;
      padding: 0 0 0 var(--space-2);
      gap: var(--space-2);
      box-shadow: none;
      background: transparent;
    }

    .ws-switcher .avatar {
      --avatar-size: 1.75rem;
    }

    .ws-role {
      display: none;
    }

    .ws-name {
      max-width: 9rem;
    }

    nav {
      flex-direction: row;
      gap: var(--space-1);
      flex-shrink: 0;
    }

    nav a {
      padding: 0 var(--space-3);
      border-radius: var(--radius-full);
    }

    .nav-indicator {
      border-radius: var(--radius-full);
    }

    .nav-indicator::after {
      display: none;
    }

    .canvas {
      display: contents;
    }

    .canvas::before {
      position: fixed;
    }

    .topbar {
      grid-row: auto;
      order: 1;
      position: relative;
      z-index: calc(var(--layer-sticky) + 1);
      padding-top: max(var(--space-2), env(safe-area-inset-top));
    }

    .topbar-brand {
      display: flex;
    }

    .breadcrumbs {
      order: 3;
      flex-basis: 100%;
    }

    main {
      order: 3;
      padding-top: var(--space-6);
    }
  }

  @media (max-width: 40rem) {
    .signout-text,
    .search-keys,
    .topbar-brand .wordmark {
      display: none;
    }
    .signout-button {
      width: var(--control-height);
      padding: 0;
    }
    .user-chip {
      border-left: 0;
      margin-left: 0;
      padding-left: 0;
    }
    main {
      padding-top: var(--space-5);
    }
  }
</style>
