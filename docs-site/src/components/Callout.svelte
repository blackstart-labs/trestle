<script lang="ts">
  import type { Snippet } from "svelte";

  interface Props {
    type?: "tip" | "note" | "warning" | "danger";
    title?: string;
    children?: Snippet;
  }

  let { type = "note", title, children }: Props = $props();

  const labels = {
    tip: "TIP",
    note: "NOTE",
    warning: "IMPORTANT",
    danger: "CAUTION"
  };
</script>

<div class="callout callout-{type}">
  <div class="callout-header">
    <span class="callout-label">{title || labels[type]}</span>
  </div>
  <div class="callout-body">
    {#if children}
      {@render children()}
    {/if}
  </div>
</div>

<style>
  .callout {
    border-left: 4px solid var(--accent);
    background: var(--surface-raised);
    border-radius: 0 var(--radius-md) var(--radius-md) 0;
    padding: 0.85rem 1.1rem;
    margin: 1.25rem 0;
    border-top: 1px solid var(--line);
    border-right: 1px solid var(--line);
    border-bottom: 1px solid var(--line);
  }
  .callout-tip {
    border-left-color: var(--success);
    background: var(--success-soft);
  }
  .callout-note {
    border-left-color: var(--info);
    background: var(--info-soft);
  }
  .callout-warning {
    border-left-color: var(--warning);
    background: var(--warning-soft);
  }
  .callout-danger {
    border-left-color: var(--danger);
    background: var(--danger-soft);
  }
  .callout-header {
    font-size: 0.78rem;
    font-weight: 700;
    letter-spacing: 0.04em;
    margin-bottom: 0.35rem;
    color: var(--text);
  }
  .callout-body {
    font-size: 0.92rem;
    color: var(--text-secondary);
  }
</style>

