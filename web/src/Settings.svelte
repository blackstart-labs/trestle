<script lang="ts">
  import Appearance from "./Appearance.svelte";
  import Readiness from "./Readiness.svelte";
</script>

<section class="settings" aria-labelledby="settings-heading">
  <div class="page-head">
    <div>
      <h1 id="settings-heading" tabindex="-1">Settings</h1>
      <p>How Trestle looks on this browser, and the health of this instance.</p>
    </div>
  </div>

  <div class="settings-grid">
    <Appearance />
    <Readiness />
  </div>
</section>

<style>
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
