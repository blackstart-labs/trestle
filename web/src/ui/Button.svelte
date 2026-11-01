<script lang="ts">
  import type { Snippet } from "svelte";
  let {
    children,
    disabled = false,
    onclick,
    type = "button",
  }: {
    children: Snippet;
    disabled?: boolean;
    onclick?: () => void;
    type?: "button" | "submit";
  } = $props();
</script>

<button {type} {disabled} {onclick}>{@render children()}</button>
