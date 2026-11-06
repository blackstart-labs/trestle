<script lang="ts">
  import { SECTIONS, type DocSection } from "../content";

  interface Props {
    open: boolean;
    onselect: (id: string) => void;
    onclose: () => void;
  }

  let { open = $bindable(), onselect, onclose }: Props = $props();
  let query = $state("");
  let selectedIndex = $state(0);

  let filtered = $derived(
    query.trim() === ""
      ? SECTIONS
      : SECTIONS.filter(s =>
          s.title.toLowerCase().includes(query.toLowerCase()) ||
          s.category.toLowerCase().includes(query.toLowerCase()) ||
          s.summary.toLowerCase().includes(query.toLowerCase())
        )
  );

  function handleKeydown(e: KeyboardEvent) {
    if (!open) return;
    if (e.key === "Escape") {
      onclose();
    } else if (e.key === "ArrowDown") {
      e.preventDefault();
      selectedIndex = (selectedIndex + 1) % Math.max(1, filtered.length);
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      selectedIndex = (selectedIndex - 1 + filtered.length) % Math.max(1, filtered.length);
    } else if (e.key === "Enter" && filtered[selectedIndex]) {
      e.preventDefault();
      select(filtered[selectedIndex].id);
    }
  }

  function select(id: string) {
    onselect(id);
    onclose();
  }
</script>

<svelte:window onkeydown={handleKeydown} />

{#if open}
  <div class="backdrop" onclick={onclose} role="presentation">
    <!-- svelte-ignore a11y_click_events_have_key_events -->
    <div class="dialog" onclick={e => e.stopPropagation()} role="dialog" tabindex="-1" aria-modal="true" aria-label="Search documentation">
      <div class="search-input-wrapper">
        <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
        <input
          type="text"
          placeholder="Search documentation, guides, API..."
          bind:value={query}
        />
        <kbd class="esc-kbd">ESC</kbd>
      </div>

      <div class="results-list">
        {#if filtered.length === 0}
          <div class="no-results">No documentation matched "{query}"</div>
        {:else}
          {#each filtered as section, i}
            <button
              class="result-item {i === selectedIndex ? 'selected' : ''}"
              onclick={() => select(section.id)}
            >
              <div class="result-head">
                <span class="result-title">{section.title}</span>
                <span class="result-cat">{section.category}</span>
              </div>
              <p class="result-summary">{section.summary}</p>
            </button>
          {/each}
        {/if}
      </div>
    </div>
  </div>
{/if}

<style>
  .backdrop {
    position: fixed;
    inset: 0;
    background: rgba(0, 0, 0, 0.65);
    backdrop-filter: blur(4px);
    display: flex;
    justify-content: center;
    align-items: flex-start;
    padding-top: 10vh;
    z-index: 1000;
  }
  .dialog {
    width: 90%;
    max-width: 580px;
    background: var(--surface-raised);
    border: 1px solid var(--border);
    border-radius: var(--radius-lg);
    box-shadow: 0 20px 40px rgba(0,0,0,0.4);
    overflow: hidden;
  }
  .search-input-wrapper {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 0.85rem 1.1rem;
    border-bottom: 1px solid var(--line);
    color: var(--text-muted);
  }
  .search-input-wrapper input {
    flex: 1;
    background: transparent;
    border: none;
    color: var(--text);
    font-size: 1.05rem;
    outline: none;
    font-family: inherit;
  }
  .esc-kbd {
    font-size: 0.72rem;
    padding: 0.15rem 0.4rem;
    border: 1px solid var(--line);
    border-radius: var(--radius-xs);
    background: var(--surface-sunken);
    color: var(--text-muted);
  }
  .results-list {
    max-height: 380px;
    overflow-y: auto;
    padding: 0.5rem;
  }
  .result-item {
    display: block;
    width: 100%;
    text-align: left;
    padding: 0.75rem 0.85rem;
    border-radius: var(--radius-md);
    border: none;
    background: transparent;
    cursor: pointer;
    transition: background 0.15s ease;
  }
  .result-item:hover, .result-item.selected {
    background: var(--hover);
  }
  .result-head {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.25rem;
  }
  .result-title {
    font-weight: 600;
    color: var(--text);
    font-size: 0.95rem;
  }
  .result-cat {
    font-size: 0.75rem;
    font-weight: 500;
    color: var(--accent);
    background: var(--accent-soft);
    padding: 0.1rem 0.45rem;
    border-radius: 999px;
  }
  .result-summary {
    font-size: 0.84rem;
    color: var(--text-secondary);
    margin: 0;
  }
  .no-results {
    padding: 2rem;
    text-align: center;
    color: var(--text-muted);
    font-size: 0.92rem;
  }
</style>
