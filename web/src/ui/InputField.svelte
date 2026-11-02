<script lang="ts">
  import type { HTMLInputAttributes } from "svelte/elements";

  let {
    id,
    label,
    type = "text",
    value = $bindable(""),
    autocomplete,
    required = false,
    hint,
    disabled = false,
  }: {
    id: string;
    label: string;
    type?: string;
    value?: string;
    autocomplete?: HTMLInputAttributes["autocomplete"];
    required?: boolean;
    hint?: string;
    disabled?: boolean;
  } = $props();
</script>

<div class="field">
  <label for={id}>{label}</label>
  {#if hint}
    <p class="note" id="{id}-hint">{hint}</p>
  {/if}
  <input
    {id}
    {type}
    bind:value
    {autocomplete}
    {required}
    {disabled}
    aria-describedby={hint ? `${id}-hint` : undefined}
  />
</div>

<style>
  input {
    width: 100%;
    min-height: var(--control-height);
    padding: var(--space-2) var(--space-3);
    border-radius: var(--radius);
    border: var(--border-width) solid var(--border);
    background: var(--surface-raised);
    color: var(--text);
    font-size: 0.9375rem;
    font-family: inherit;
    transition:
      border-color var(--transition-fast),
      box-shadow var(--transition-fast);
  }

  input:focus {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
    border-color: var(--accent);
  }

  input:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
</style>
