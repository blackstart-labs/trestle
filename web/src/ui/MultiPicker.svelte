<script lang="ts" module>
  export interface PickerItem {
    value: string;
    label: string;
    /** A secondary line, e.g. a username. */
    sub?: string;
    color?: string;
    /** Stable hue for a person's avatar. */
    hue?: number;
    initials?: string;
    avatarUrl?: string | null;
  }
</script>

<script lang="ts">
  import { tick } from "svelte";
  import Icon from "./Icon.svelte";
  import { floating, uid } from "../lib/floating";
  import { pop, shrink } from "../lib/motion";

  // A searchable checklist in a popover: every toggle applies at once, so
  // assigning someone or tagging a task is one click, not pick-then-confirm.
  let {
    items,
    selected,
    ontoggle,
    oncreate,
    label,
    searchLabel = "Search",
    createLabel = (q: string) => `Create “${q}”`,
    emptyText = "Nothing to show",
  }: {
    items: readonly PickerItem[];
    selected: readonly string[];
    ontoggle: (value: string, nowSelected: boolean) => void;
    /** When present, a typed name with no exact match can be created. */
    oncreate?: (name: string) => void;
    /** The trigger's accessible name, e.g. "Add assignee". */
    label: string;
    searchLabel?: string;
    createLabel?: (query: string) => string;
    emptyText?: string;
  } = $props();

  const listId = uid("picker");
  let open = $state(false);
  let query = $state("");
  let active = $state(0);
  let trigger = $state<HTMLButtonElement | null>(null);
  let input = $state<HTMLInputElement | null>(null);

  const results = $derived.by(() => {
    const q = query.trim().toLowerCase();
    return q
      ? items.filter((i) =>
          `${i.label} ${i.sub ?? ""}`.toLowerCase().includes(q),
        )
      : [...items];
  });
  const canCreate = $derived(
    !!oncreate &&
      query.trim().length > 0 &&
      !items.some((i) => i.label.toLowerCase() === query.trim().toLowerCase()),
  );
  const rowCount = $derived(results.length + (canCreate ? 1 : 0));

  async function openPicker() {
    query = "";
    active = 0;
    open = true;
    await tick();
    input?.focus();
  }

  function close(restore = true) {
    if (!open) return;
    open = false;
    if (restore) trigger?.focus();
  }

  function activate(index: number) {
    if (index < results.length) {
      const item = results[index];
      ontoggle(item.value, !selected.includes(item.value));
    } else if (canCreate) {
      oncreate?.(query.trim());
      query = "";
    }
  }

  function onKey(e: KeyboardEvent) {
    if (e.key === "ArrowDown") {
      e.preventDefault();
      if (rowCount) active = (active + 1) % rowCount;
    } else if (e.key === "ArrowUp") {
      e.preventDefault();
      if (rowCount) active = (active - 1 + rowCount) % rowCount;
    } else if (e.key === "Enter") {
      e.preventDefault();
      activate(active);
    } else if (e.key === "Escape") {
      e.preventDefault();
      e.stopPropagation();
      close();
    } else if (e.key === "Tab") {
      close(false);
    }
  }
</script>

<button
  bind:this={trigger}
  type="button"
  class="add-btn"
  class:open
  aria-label={label}
  title={label}
  aria-haspopup="listbox"
  aria-expanded={open}
  onclick={() => (open ? close() : openPicker())}
>
  <Icon name="plus" size={15} />
</button>

{#if open}
  <div
    class="picker"
    use:floating={{ anchor: trigger, onclose: () => close(false) }}
    in:pop
    out:shrink
  >
    <div class="search">
      <Icon name="search" size={15} />
      <input
        bind:this={input}
        bind:value={query}
        oninput={() => (active = 0)}
        onkeydown={onKey}
        type="text"
        role="combobox"
        aria-expanded="true"
        aria-controls={listId}
        aria-activedescendant={rowCount ? `${listId}-${active}` : undefined}
        aria-label={searchLabel}
        placeholder={searchLabel}
        autocomplete="off"
        spellcheck="false"
      />
    </div>
    <ul
      id={listId}
      class="list"
      role="listbox"
      aria-label={label}
      aria-multiselectable="true"
    >
      {#each results as item, i (item.value)}
        {@const on = selected.includes(item.value)}
        <li
          id="{listId}-{i}"
          class="row"
          class:active={i === active}
          role="option"
          aria-selected={on}
          onpointermove={() => (active = i)}
          onclick={() => activate(i)}
          onkeydown={() => {}}
        >
          <span class="check" class:on aria-hidden="true">
            {#if on}<Icon name="check" size={12} />{/if}
          </span>
          {#if item.initials !== undefined}
            <span
              class="avatar avatar-sm"
              style:--avatar-hue={item.hue}
              aria-hidden="true"
            >
              {#if item.avatarUrl}
                <img
                  src={item.avatarUrl}
                  alt=""
                  class="avatar-img"
                  onerror={(e) => {
                    (e.currentTarget as HTMLElement).style.display = "none";
                  }}
                />
              {/if}
              <span>{item.initials}</span>
            </span>
          {:else if item.color}
            <span class="dot" style:background={item.color} aria-hidden="true"
            ></span>
          {/if}
          <span class="row-text">
            <span class="row-label">{item.label}</span>
            {#if item.sub}<span class="row-sub">{item.sub}</span>{/if}
          </span>
        </li>
      {/each}
      {#if canCreate}
        <li
          id="{listId}-{results.length}"
          class="row create"
          class:active={active === results.length}
          role="option"
          aria-selected="false"
          onpointermove={() => (active = results.length)}
          onclick={() => activate(results.length)}
          onkeydown={() => {}}
        >
          <Icon name="plus" size={14} />
          <span class="row-label">{createLabel(query.trim())}</span>
        </li>
      {/if}
      {#if rowCount === 0}
        <li class="empty-row" role="presentation">{emptyText}</li>
      {/if}
    </ul>
  </div>
{/if}

<style>
  .add-btn {
    width: 2rem;
    min-height: 2rem;
    padding: 0;
    border-radius: var(--radius-full);
    border: var(--border-width) dashed var(--line-strong);
    background: transparent;
    box-shadow: none;
    color: var(--text-muted);
  }

  .add-btn:hover:not(:disabled),
  .add-btn.open {
    border-style: solid;
    background:
      linear-gradient(var(--hover), var(--hover)) padding-box,
      linear-gradient(var(--line-strong), var(--line-strong)) border-box;
    color: var(--text);
  }

  .picker {
    z-index: var(--layer-popover);
    display: flex;
    flex-direction: column;
    width: 16.5rem;
    border-radius: var(--radius);
    border: var(--border-width) solid transparent;
    background:
      var(--card-fill) padding-box,
      var(--card-edge) border-box;
    box-shadow: var(--shadow-float);
    overflow: hidden;
  }

  .search {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: 0 var(--space-3);
    border-bottom: var(--border-width) solid var(--line);
    color: var(--text-muted);
  }

  .search input {
    min-height: 2.5rem;
    padding: 0;
    border: 0;
    background: transparent;
    box-shadow: none;
    font-size: var(--text-small);
  }

  .search input:focus {
    background: transparent;
    box-shadow: none;
  }

  .list {
    list-style: none;
    margin: 0;
    padding: 4px;
    max-height: 16rem;
    overflow-y: auto;
    overscroll-behavior: contain;
  }

  .row {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    min-height: 2.25rem;
    padding: 0.25rem 0.5rem;
    border-radius: calc(var(--radius) - 4px);
    font-size: var(--text-small);
    color: var(--text-secondary);
    cursor: pointer;
    user-select: none;
  }

  .row.active {
    background: var(--selected);
    color: var(--text);
  }

  .check {
    display: grid;
    place-items: center;
    width: 1rem;
    height: 1rem;
    border-radius: 0.3rem;
    border: 1.5px solid var(--border);
    flex-shrink: 0;
    transition:
      background-color var(--transition-fast),
      border-color var(--transition-fast);
  }

  .check.on {
    background: var(--button-bg);
    border-color: var(--button-bg);
    color: var(--button-text);
  }

  .dot {
    width: 0.55rem;
    height: 0.55rem;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .row-text {
    display: grid;
    min-width: 0;
    line-height: 1.25;
  }

  .row-label {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .row-sub {
    font-size: var(--text-xs);
    color: var(--text-muted);
  }

  .create {
    color: var(--accent);
    font-weight: 560;
  }

  .empty-row {
    padding: var(--space-4) var(--space-3);
    text-align: center;
    font-size: var(--text-small);
    color: var(--text-muted);
  }

  @media (pointer: coarse) {
    .add-btn {
      width: var(--control-height);
      min-height: var(--control-height);
    }
    .row {
      min-height: var(--control-height);
    }
  }
</style>
