<script lang="ts">
  interface Props {
    code: string;
    lang?: string;
  }

  let { code, lang = "bash" }: Props = $props();
  let copied = $state(false);

  function copy() {
    navigator.clipboard.writeText(code.trim());
    copied = true;
    setTimeout(() => {
      copied = false;
    }, 2000);
  }
</script>

<div class="code-wrapper">
  <div class="code-header">
    <span class="lang-tag">{lang}</span>
    <button class="copy-btn" onclick={copy} aria-label="Copy code">
      {#if copied}
        <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2.5"><polyline points="20 6 9 17 4 12"/></svg>
        <span>Copied!</span>
      {:else}
        <svg viewBox="0 0 24 24" width="14" height="14" fill="none" stroke="currentColor" stroke-width="2"><rect x="9" y="9" width="13" height="13" rx="2" ry="2"/><path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1"/></svg>
        <span>Copy</span>
      {/if}
    </button>
  </div>
  <pre><code>{code.trim()}</code></pre>
</div>

<style>
  .code-wrapper {
    position: relative;
    background: var(--surface-sunken);
    border: 1px solid var(--line);
    border-radius: var(--radius-md);
    margin: 1rem 0 1.5rem 0;
    overflow: hidden;
  }
  .code-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.4rem 0.85rem;
    background: var(--surface-nav);
    border-bottom: 1px solid var(--line);
    font-size: 0.78rem;
  }
  .lang-tag {
    font-family: var(--font-mono);
    text-transform: uppercase;
    font-size: 0.72rem;
    color: var(--text-muted);
    font-weight: 600;
  }
  .copy-btn {
    display: flex;
    align-items: center;
    gap: 0.35rem;
    background: transparent;
    border: 1px solid var(--line);
    color: var(--text-secondary);
    padding: 0.2rem 0.5rem;
    border-radius: var(--radius-xs);
    cursor: pointer;
    font-size: 0.75rem;
    transition: all 0.15s ease;
  }
  .copy-btn:hover {
    background: var(--hover);
    color: var(--text);
  }
  pre {
    padding: 0.9rem 1.1rem;
    overflow-x: auto;
    margin: 0;
    font-family: var(--font-mono);
    font-size: 0.88rem;
    line-height: 1.5;
    color: var(--text);
  }
  code {
    background: transparent;
    padding: 0;
    border: none;
    font-family: inherit;
  }
</style>

