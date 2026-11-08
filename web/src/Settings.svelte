<script lang="ts">
  import Appearance from "./Appearance.svelte";
  import Readiness from "./Readiness.svelte";
  import Integrations from "./Integrations.svelte";
  import type { Workspace } from "./auth/WorkspaceChooser.svelte";
  let {
    workspace = null,
    signedIn = false,
    initialProjectId = "",
    section = "",
    onselectworkspace,
  }: {
    workspace?: Workspace | null;
    signedIn?: boolean;
    initialProjectId?: string;
    section?: string;
    onselectworkspace?: (ws: Workspace) => void;
  } = $props();
</script>

<section class="settings" aria-labelledby="settings-heading">
  <div class="page-head">
    <div>
      <h1 id="settings-heading" tabindex="-1">Settings</h1>
      <p>Appearance, instance health, and project integrations.</p>
    </div>
  </div>

  <nav class="page-actions settings-sections" aria-label="Settings sections">
    <a
      href="#appearance"
      aria-current={section !== "integration" ? "page" : undefined}
      >Appearance</a
    >
    <a
      href="#integration"
      aria-current={section === "integration" ? "page" : undefined}
      >Integration</a
    >
  </nav>

  {#if section === "integration"}
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

  @media (max-width: 64rem) {
    .settings-grid {
      grid-template-columns: minmax(0, 1fr);
    }
  }
</style>
