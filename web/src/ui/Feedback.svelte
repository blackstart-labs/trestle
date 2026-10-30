<script lang="ts">
  import type { Snippet } from "svelte";
  let {
    children,
    tone = "neutral",
    announcement,
  }: {
    children: Snippet;
    tone?: "neutral" | "success" | "warning" | "danger";
    announcement?: "status" | "alert";
  } = $props();
</script>

<p
  class="feedback"
  data-tone={tone}
  role={announcement}
  aria-live={announcement ? undefined : "polite"}
>
  {@render children()}
</p>
