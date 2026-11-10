<script lang="ts">
  import { tick } from "svelte";
  import Appearance from "./Appearance.svelte";
  import Readiness from "./Readiness.svelte";
  import Integrations from "./Integrations.svelte";
  import ProfileSettings from "./auth/ProfileSettings.svelte";
  import WorkspaceSettings from "./auth/WorkspaceSettings.svelte";
  import type { Workspace } from "./auth/WorkspaceChooser.svelte";
  import type { User } from "./auth/SignIn.svelte";

  let {
    workspace = null,
    user = null,
    signedIn = false,
    initialProjectId = "",
    section = "",
    onselectsection,
    onselectworkspace,
    onworkspaceupdated,
    onuserupdated,
  }: {
    workspace?: Workspace | null;
    user?: User | null;
    signedIn?: boolean;
    initialProjectId?: string;
    section?: string;
    onselectsection?: (section: string) => void;
    onselectworkspace?: (ws: Workspace) => void;
    onworkspaceupdated?: (ws: Workspace) => void;
    onuserupdated?: (u: User) => void;
  } = $props();

  async function handleTabClick(e: MouseEvent, targetSection: string) {
    e.preventDefault();
    onselectsection?.(targetSection);
    try {
      history.replaceState(null, "", "#" + targetSection);
    } catch {}
    await tick();
    document
      .getElementById(
        targetSection === "workspace" ? "ws-heading" : targetSection,
      )
      ?.focus({ preventScroll: true });
  }
</script>

<section class="settings" aria-labelledby="settings-heading">
  <div class="page-head">
    <div>
      <h1 id="settings-heading" tabindex="-1">Settings</h1>
      <p>
        Workspace, profile, appearance, instance health, and project
        integrations.
      </p>
    </div>
  </div>

  <nav class="page-actions settings-sections" aria-label="Settings sections">
    {#if signedIn && user}
      <a
        href="#profile"
        aria-current={section === "profile" ? "page" : undefined}
        onclick={(e) => handleTabClick(e, "profile")}>Profile</a
      >
    {/if}
    {#if workspace}
      <a
        href="#workspace"
        aria-current={section === "workspace" ? "page" : undefined}
        onclick={(e) => handleTabClick(e, "workspace")}>Workspace</a
      >
    {/if}
    <a
      href="#appearance"
      aria-current={section === "appearance" ||
      (!section && !workspace && !user) ||
      (section !== "integration" &&
        section !== "workspace" &&
        section !== "profile")
        ? "page"
        : undefined}
      onclick={(e) => handleTabClick(e, "appearance")}>Appearance</a
    >
    <a
      href="#integration"
      aria-current={section === "integration" ? "page" : undefined}
      onclick={(e) => handleTabClick(e, "integration")}>Integration</a
    >
  </nav>

  {#if section === "profile" && user}
    <div class="settings-single">
      <ProfileSettings {user} {onuserupdated} />
    </div>
  {:else if section === "workspace" && workspace}
    <div class="settings-single">
      {#key workspace.id}<WorkspaceSettings
          {workspace}
          {onworkspaceupdated}
        />{/key}
    </div>
  {:else if section === "integration"}
    <Integrations
      {workspace}
      {signedIn}
      {initialProjectId}
      {onselectworkspace}
    />
  {:else}
    <div class="settings-grid">
      <Appearance />
      <Readiness />
    </div>
  {/if}
</section>

<style>
  .settings-sections {
    margin-bottom: var(--space-5);
    gap: var(--space-5);
    border-bottom: var(--border-width) solid var(--line);
  }
  .settings-sections a {
    display: inline-flex;
    align-items: center;
    min-height: var(--control-height);
    border-bottom: 2px solid transparent;
    text-decoration: none;
    color: var(--text-muted);
  }
  .settings-sections a[aria-current="page"] {
    border-bottom-color: var(--accent);
    color: var(--text);
  }
  .settings-grid {
    display: grid;
    grid-template-columns: minmax(0, 3fr) minmax(0, 2fr);
    gap: var(--space-5);
    align-items: start;
  }
  .settings-single {
    max-width: 44rem;
  }

  @media (max-width: 64rem) {
    .settings-grid {
      grid-template-columns: minmax(0, 1fr);
    }
  }
</style>
