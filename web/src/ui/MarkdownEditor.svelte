<script lang="ts">
  import { tick } from "svelte";
  import { renderMarkdown } from "../lib/markdown";

  let {
    id,
    value = $bindable(""),
    label = "Description",
    placeholder = "Add context, acceptance criteria, or links…",
    disabled = false,
    readonly = false,
  }: {
    id: string;
    value?: string;
    label?: string;
    placeholder?: string;
    disabled?: boolean;
    readonly?: boolean;
  } = $props();

  let preview = $state(false);
  let textarea = $state<HTMLTextAreaElement>();
  const html = $derived(renderMarkdown(value));

  async function format(before: string, after = "", fallback = "text") {
    if (disabled || readonly || !textarea) return;
    const start = textarea.selectionStart;
    const end = textarea.selectionEnd;
    const selected = value.slice(start, end) || fallback;
    value =
      value.slice(0, start) + before + selected + after + value.slice(end);
    await tick();
    textarea.focus();
    textarea.setSelectionRange(
      start + before.length,
      start + before.length + selected.length,
    );
  }
</script>

<div class="markdown-editor">
  <div class="editor-head">
    <label for={readonly || preview ? undefined : id}>{label}</label>
    {#if !readonly}
      <div class="editor-modes" role="group" aria-label="{label} mode">
        <button
          type="button"
          class="btn-quiet"
          aria-pressed={!preview}
          onclick={() => (preview = false)}>Write</button
        >
        <button
          type="button"
          class="btn-quiet"
          aria-pressed={preview}
          onclick={() => (preview = true)}>Preview</button
        >
      </div>
    {/if}
  </div>
  {#if readonly || preview}
    <!-- svelte-ignore a11y_no_noninteractive_tabindex — keyboard access to overflowing code/tables -->
    <div
      class="markdown-preview"
      role="region"
      aria-label="{label} preview"
      tabindex="0"
    >
      {#if value.trim()}
        {@html html}
      {:else}
        <p class="note">
          {readonly
            ? "No description."
            : "Nothing to preview. Write a description first."}
        </p>
      {/if}
    </div>
  {:else}
    <div
      class="editor-tools"
      role="group"
      aria-label="Format {label.toLowerCase()}"
    >
      <button
        type="button"
        class="btn-quiet"
        {disabled}
        onclick={() => format("**", "**")}>Bold</button
      >
      <button
        type="button"
        class="btn-quiet"
        {disabled}
        onclick={() => format("*", "*")}>Italic</button
      >
      <button
        type="button"
        class="btn-quiet"
        {disabled}
        onclick={() => format("\n## ", "\n", "Heading")}>Heading</button
      >
      <button
        type="button"
        class="btn-quiet"
        {disabled}
        onclick={() => format("\n- ", "\n", "List item")}>List</button
      >
      <button
        type="button"
        class="btn-quiet"
        {disabled}
        onclick={() => format("[", "](https://example.com)", "Link text")}
        >Link</button
      >
      <button
        type="button"
        class="btn-quiet"
        {disabled}
        onclick={() => format("`", "`", "code")}>Code</button
      >
    </div>
    <textarea
      bind:this={textarea}
      {id}
      bind:value
      {disabled}
      {placeholder}
      rows="6"
      aria-describedby="{id}-hint"></textarea>
    <p id="{id}-hint" class="note">
      Markdown supported. Preview shows your unsaved text. Images are shown as
      text.
    </p>
  {/if}
</div>

<style>
  .markdown-editor {
    display: grid;
    gap: var(--space-2);
    min-width: 0;
  }
  .editor-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2);
    flex-wrap: wrap;
  }
  .editor-head label {
    font-size: var(--text-small);
    font-weight: 540;
    color: var(--text-secondary);
  }
  .editor-modes,
  .editor-tools {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-1);
  }
  .editor-tools {
    border-bottom: 1px solid var(--line);
    padding-bottom: var(--space-2);
  }
  .editor-modes button,
  .editor-tools button {
    padding-inline: var(--space-2);
    font-size: var(--text-small);
    min-height: 44px;
  }
  .editor-modes button[aria-pressed="true"] {
    background: var(--selected);
    color: var(--text);
  }
  textarea {
    width: 100%;
    min-height: 9rem;
    resize: vertical;
    line-height: 1.65;
  }
</style>
