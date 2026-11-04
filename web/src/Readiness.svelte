<script lang="ts">
  import { onMount } from "svelte";
  import Button from "./ui/Button.svelte";
  import Feedback from "./ui/Feedback.svelte";

  let checking = $state(false);
  let status = $state("Checking database connection…");
  let active: AbortController | undefined;
  let tone = $state<"neutral" | "success" | "warning" | "danger">("neutral");
  let lastChecked = $state<Date | null>(null);

  async function check() {
    if (checking) return;
    checking = true;
    tone = "neutral";
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
        tone = "warning";
      } else if (
        response.status === 200 &&
        response.headers.get("content-type")?.split(";")[0].trim() ===
          "application/json" &&
        (await response.json())?.status === "ok"
      ) {
        status = "Database connection ready.";
        tone = "success";
      } else {
        status = "Could not check readiness. Try again.";
        tone = "danger";
      }
    } catch {
      status = "Could not check readiness. Try again.";
      tone = "danger";
    } finally {
      clearTimeout(deadline);
      checking = false;
      active = undefined;
      lastChecked = new Date();
    }
  }

  onMount(() => {
    void check();
    return () => active?.abort();
  });
</script>

<section class="panel panel-pad status-panel" aria-labelledby="instance-status">
  <div class="status-head">
    <h2 id="instance-status" tabindex="-1">Instance status</h2>
    <Button onclick={check} disabled={checking}>
      {#snippet children()}Check again{/snippet}
    </Button>
  </div>

  <div class="probe" data-tone={tone}>
    <span class="probe-light" aria-hidden="true"></span>
    <div class="probe-text">
      <span class="probe-name">Database</span>
      <Feedback {tone} announcement="status">{status}</Feedback>
    </div>
  </div>

  <p class="note">
    Readiness checks connectivity only, not schema compatibility or product
    availability.
    {#if lastChecked}
      Last checked at
      <time datetime={lastChecked.toISOString()}
        >{lastChecked.toLocaleTimeString(undefined, {
          hour: "2-digit",
          minute: "2-digit",
          second: "2-digit",
        })}</time
      >.
    {/if}
  </p>
</section>

<style>
  .status-panel {
    display: flex;
    flex-direction: column;
    gap: var(--space-5);
  }

  .status-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    flex-wrap: wrap;
  }

  .status-head h2 {
    font-size: 1.25rem;
  }

  .probe {
    background:
      var(--inset-fill) padding-box,
      var(--inset-edge) border-box;
    border: var(--border-width) solid transparent;
    display: flex;
    align-items: flex-start;
    gap: var(--space-4);
    padding: var(--space-4);
    border-radius: var(--radius);
  }

  .probe-light {
    position: relative;
    width: 0.75rem;
    height: 0.75rem;
    margin-top: 0.35rem;
    border-radius: 50%;
    background: var(--text-muted);
    flex-shrink: 0;
  }

  .probe[data-tone="success"] .probe-light {
    background: var(--success);
  }
  .probe[data-tone="warning"] .probe-light {
    background: var(--warning);
  }
  .probe[data-tone="danger"] .probe-light {
    background: var(--danger);
  }

  .probe[data-tone="neutral"] .probe-light::after {
    content: "";
    position: absolute;
    inset: -4px;
    border-radius: 50%;
    border: 2px solid var(--text-muted);
    animation: probe-ping 1.2s var(--ease-out) infinite;
  }

  @keyframes probe-ping {
    from {
      opacity: 0.8;
      transform: scale(0.6);
    }
    to {
      opacity: 0;
      transform: scale(1.6);
    }
  }

  .probe-text {
    display: grid;
    gap: var(--space-2);
    min-width: 0;
    flex: 1;
  }

  .probe-name {
    font-weight: 620;
    color: var(--text);
  }

  .probe-text :global(.feedback) {
    padding: 0;
    background: none;
    font-size: var(--text-ui);
  }

  .probe-text :global(.feedback::before) {
    display: none;
  }
</style>
