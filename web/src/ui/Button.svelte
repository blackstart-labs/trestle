<script lang="ts">
  import type { Snippet } from "svelte";
  let {
    children,
    disabled = false,
    onclick,
  }: {
    children: Snippet;
    disabled?: boolean;
    onclick: () => void;
  } = $props();
</script>

<button type="button" {disabled} {onclick}>{@render children()}</button>
