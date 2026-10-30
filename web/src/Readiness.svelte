<script lang="ts">
  import { onMount } from "svelte";
  import Button from "./ui/Button.svelte";
  import Feedback from "./ui/Feedback.svelte";

  let checking = $state(false);
  let status = $state("Checking database connection…");
  let active: AbortController | undefined;
  let tone = $state<"neutral" | "success" | "warning" | "danger">("neutral");

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
    }
  }

  onMount(() => {
    void check();
    return () => active?.abort();
  });
</script>

<section aria-labelledby="instance-status">
  <p class="eyebrow">Operations</p>
  <h2 id="instance-status" tabindex="-1">Instance status</h2>
  <Feedback {tone} announcement="status">{status}</Feedback>
  <Button onclick={check} disabled={checking}>Check again</Button>
  <p class="note">
    Readiness checks connectivity only, not schema compatibility or product
    availability.
  </p>
</section>
