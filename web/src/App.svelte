<script lang="ts">
  import Readiness from "./Readiness.svelte";
  import Appearance from "./Appearance.svelte";
</script>

<a class="skip-link" href="#main">Skip to content</a>
<div class="shell">
  <header class="masthead">
    <span class="wordmark">Trestle<span aria-hidden="true">.</span></span>
    <span class="build-label">Development build</span>
  </header>
  <nav aria-label="Application">
    <p class="eyebrow">This instance</p>
    <a href="#overview">Overview</a>
    <a href="#appearance">Appearance</a>
    <a href="#instance-status">Instance status</a>
  </nav>
  <main id="main" tabindex="-1">
    <section aria-labelledby="overview">
      <p class="eyebrow">Engineering foundation</p>
      <h1 id="overview" tabindex="-1">Trestle</h1>
      <p class="lead">One place for projects, time and connected knowledge.</p>
      <p>
        This build contains the application foundation. Accounts, projects and
        tasks are not available yet.
      </p>
    </section>
    <Appearance />
    <Readiness />
  </main>
  <footer>
    <span>A self-hosted workspace, under development.</span>
    <a href="/assets/third-party-licenses.txt">Frontend third-party licenses</a>
  </footer>
</div>
