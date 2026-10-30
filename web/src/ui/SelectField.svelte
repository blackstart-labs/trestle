<script lang="ts">
  let {
    id,
    label,
    hint,
    value,
    options,
    onchange,
    feedbackId,
  }: {
    id: string;
    label: string;
    hint: string;
    value: string;
    options: readonly { value: string; label: string }[];
    onchange: (value: string) => void;
    feedbackId?: string;
  } = $props();
</script>

<div class="field">
  <label for={id}>{label}</label>
  <p id={`${id}-hint`} class="note">{hint}</p>
  <select
    {id}
    {value}
    aria-describedby={`${id}-hint${feedbackId ? ` ${feedbackId}` : ""}`}
    onchange={(event) => onchange(event.currentTarget.value)}
  >
    {#each options as option (option.value)}
      <option value={option.value}>{option.label}</option>
    {/each}
  </select>
</div>
