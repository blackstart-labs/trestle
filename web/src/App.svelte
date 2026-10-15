<script lang="ts">
  import { onMount } from "svelte";

  let checking = $state(false);
  let status = $state("Checking database connection…");
  let active: AbortController | undefined;

  async function check() {
    if (checking) return;
    checking = true;
    status = "Checking database connection…";
    const controller = new AbortController();
    active = controller;
    const deadline = setTimeout(() => controller.abort(), 5000);
    try {
      const response = await fetch("/health/ready", {
        signal: controller.signal,
        cache: "no-store",
      });
      if (response.status === 503) {
        status = "Database connection unavailable.";
      } else if (
        response.status === 200 &&
        response.headers.get("content-type")?.split(";")[0].trim() ===
          "application/json" &&
        (await response.json())?.status === "ok"
      ) {
        status = "Database connection ready.";
      } else {
        status = "Could not check readiness. Try again.";
      }
    } catch {
      status = "Could not check readiness. Try again.";
    } finally {
      clearTimeout(deadline);
      checking = false;
      active = undefined;
    }
  }

  onMount(() => {
    void check();
    return () => active?.abort();
  });
</script>

<main>
  <p class="eyebrow">Engineering foundation</p>
  <h1>Trestle</h1>
  <p>A self-hosted workspace for projects, time and connected knowledge.</p>
  <p>
    This build contains the application foundation. Accounts, projects and tasks
    are not available yet.
  </p>
  <section aria-labelledby="connection-title">
    <h2 id="connection-title">Database connection</h2>
    <p role="status" aria-live="polite">{status}</p>
    <button onclick={check} disabled={checking}>Check again</button>
    <p class="note">
      Readiness checks connectivity only, not schema compatibility or product
      availability.
    </p>
  </section>
  <p class="note">
    <a href="/assets/third-party-licenses.txt">Frontend third-party licenses</a>
  </p>
</main>

<style>
  :global(html) {
    font-family: system-ui, sans-serif;
    background: #f7f7f4;
    color: #242824;
  }
  :global(body) {
    margin: 0;
  }
  :global(*) {
    box-sizing: border-box;
  }
  main {
    max-width: 48rem;
    margin: 0 auto;
    padding: clamp(1.25rem, 6vw, 4rem);
    line-height: 1.6;
    overflow-wrap: anywhere;
  }
  h1 {
    font-size: clamp(2.5rem, 8vw, 4rem);
    line-height: 1.1;
    margin: 0.5rem 0 1.5rem;
  }
  h2 {
    font-size: 1.25rem;
  }
  .eyebrow {
    text-transform: uppercase;
    font-size: 0.8rem;
    letter-spacing: 0.08em;
  }
  section {
    border-top: 1px solid #b7bdb7;
    margin-top: 2rem;
    padding-top: 1rem;
  }
  .note {
    font-size: 0.9rem;
    color: #4c544c;
  }
  button {
    font: inherit;
    padding: 0.6rem 1rem;
    min-height: 44px;
    background: #242824;
    color: #fff;
    border: 1px solid #242824;
    border-radius: 3px;
    cursor: pointer;
  }
  button:disabled {
    opacity: 0.6;
    cursor: wait;
  }
  button:focus-visible {
    outline: 3px solid #486c4b;
    outline-offset: 4px;
  }
</style>
