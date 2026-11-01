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
