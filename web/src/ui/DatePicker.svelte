<script lang="ts">
  import { tick } from "svelte";
  import Icon from "./Icon.svelte";
  import { floating, uid } from "../lib/floating";
  import { pop, shrink } from "../lib/motion";

  let {
    id,
    value,
    onchange,
    label,
    placeholder = "No date",
    variant = "field",
    disabled = false,
  }: {
    id?: string;
    /** YYYY-MM-DD, or "" for none. */
    value: string;
    onchange: (value: string) => void;
    label?: string;
    placeholder?: string;
    variant?: "field" | "chip";
    disabled?: boolean;
  } = $props();

  const dialogId = uid("datepicker");
  let open = $state(false);
  let trigger = $state<HTMLButtonElement | null>(null);
  let grid = $state<HTMLElement | null>(null);
  let focusDate = $state(new Date());
  let viewYear = $state(new Date().getFullYear());
  let viewMonth = $state(new Date().getMonth());

  const pad = (n: number) => String(n).padStart(2, "0");
  const toIso = (d: Date) =>
    `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
  function parse(v: string): Date | null {
    const [y, m, d] = v.split("-").map(Number);
    return y && m && d ? new Date(y, m - 1, d) : null;
  }
  const sameDay = (a: Date, b: Date) =>
    a.getFullYear() === b.getFullYear() &&
    a.getMonth() === b.getMonth() &&
    a.getDate() === b.getDate();

  const today = new Date();
  const selected = $derived(parse(value));
  const overdue = $derived(
    selected !== null &&
      selected <
        new Date(today.getFullYear(), today.getMonth(), today.getDate()),
  );

  const display = $derived.by(() => {
    if (!selected) return placeholder;
    const opts: Intl.DateTimeFormatOptions = { month: "short", day: "numeric" };
    if (selected.getFullYear() !== today.getFullYear()) opts.year = "numeric";
    if (sameDay(selected, today)) return "Today";
    const tomorrow = new Date(today);
    tomorrow.setDate(today.getDate() + 1);
    if (sameDay(selected, tomorrow)) return "Tomorrow";
    return selected.toLocaleDateString(undefined, opts);
  });

  const monthLabel = $derived(
    new Date(viewYear, viewMonth, 1).toLocaleDateString(undefined, {
      month: "long",
      year: "numeric",
    }),
  );

  // Week starts on Monday; six rows keep the panel from jumping.
  const weekdays = $derived(
    Array.from({ length: 7 }, (_, i) =>
      new Date(2024, 0, 1 + i).toLocaleDateString(undefined, {
        weekday: "narrow",
      }),
    ),
  );
  const days = $derived.by(() => {
    const first = new Date(viewYear, viewMonth, 1);
    const offset = (first.getDay() + 6) % 7;
    return Array.from({ length: 42 }, (_, i) => {
      const date = new Date(viewYear, viewMonth, 1 - offset + i);
      return { date, inMonth: date.getMonth() === viewMonth };
    });
  });

  async function openPicker() {
    if (disabled) return;
    const start = selected ?? today;
    focusDate = new Date(start);
    viewYear = start.getFullYear();
    viewMonth = start.getMonth();
    open = true;
    await focusGrid();
  }

  function close(restore = true) {
    if (!open) return;
    open = false;
    if (restore) trigger?.focus();
  }

  function pick(date: Date | null) {
    const next = date ? toIso(date) : "";
    close();
    if (next !== value) onchange(next);
  }

  function quick(days: number) {
    const d = new Date(today.getFullYear(), today.getMonth(), today.getDate());
    d.setDate(d.getDate() + days);
    pick(d);
  }

  function shiftMonth(delta: number) {
    const d = new Date(viewYear, viewMonth + delta, 1);
    viewYear = d.getFullYear();
    viewMonth = d.getMonth();
  }

  async function focusGrid() {
    await tick();
    grid?.querySelector<HTMLElement>('[tabindex="0"]')?.focus();
  }

  async function moveFocus(next: Date) {
    focusDate = next;
    if (next.getMonth() !== viewMonth || next.getFullYear() !== viewYear) {
      viewYear = next.getFullYear();
      viewMonth = next.getMonth();
    }
    await focusGrid();
  }

  function onGridKey(e: KeyboardEvent) {
    const d = new Date(focusDate);
    const moves: Record<string, () => void> = {
      ArrowLeft: () => d.setDate(d.getDate() - 1),
      ArrowRight: () => d.setDate(d.getDate() + 1),
      ArrowUp: () => d.setDate(d.getDate() - 7),
      ArrowDown: () => d.setDate(d.getDate() + 7),
      Home: () => d.setDate(d.getDate() - ((d.getDay() + 6) % 7)),
      End: () => d.setDate(d.getDate() + (6 - ((d.getDay() + 6) % 7))),
      PageUp: () => d.setMonth(d.getMonth() - 1),
      PageDown: () => d.setMonth(d.getMonth() + 1),
    };
    if (moves[e.key]) {
      e.preventDefault();
      moves[e.key]();
      void moveFocus(d);
    }
  }

  function onPanelKey(e: KeyboardEvent) {
    if (e.key === "Escape") {
      e.preventDefault();
      e.stopPropagation();
      close();
    }
  }
</script>

<button
  bind:this={trigger}
  {id}
  type="button"
  class="date date-{variant}"
  class:open
  class:empty={!selected}
  class:overdue
  aria-haspopup="dialog"
  aria-expanded={open}
  aria-controls={open ? dialogId : undefined}
  aria-label={label
    ? `${label}: ${selected ? selected.toLocaleDateString() : placeholder}`
    : undefined}
  {disabled}
  onclick={() => (open ? close() : openPicker())}
>
  <Icon name="calendar" size={15} class="date-icon" />
  <span class="date-text">{display}</span>
  {#if variant === "field"}
    <Icon name="chevron-down" size={15} class="date-chevron" />
  {/if}
</button>

{#if open}
  <div
    id={dialogId}
    class="picker"
    role="dialog"
    aria-label={label ?? "Choose a date"}
    tabindex="-1"
    use:floating={{ anchor: trigger, onclose: () => close(false) }}
    onkeydown={onPanelKey}
    in:pop
    out:shrink
  >
    <div class="quick">
      <button type="button" onclick={() => quick(0)}>Today</button>
      <button type="button" onclick={() => quick(1)}>Tomorrow</button>
      <button type="button" onclick={() => quick(7)}>Next week</button>
    </div>

    <div class="month">
      <button
        type="button"
        class="nav-btn"
        onclick={() => shiftMonth(-1)}
        aria-label="Previous month"
      >
        <Icon name="chevron-right" size={16} class="flip" />
      </button>
      <span class="month-label" aria-live="polite">{monthLabel}</span>
      <button
        type="button"
        class="nav-btn"
        onclick={() => shiftMonth(1)}
        aria-label="Next month"
      >
        <Icon name="chevron-right" size={16} />
      </button>
    </div>

    <div class="weekdays" aria-hidden="true">
      {#each weekdays as w, i (i)}<span>{w}</span>{/each}
    </div>

    <!-- Arrow keys move between days; the buttons inside are the controls. -->
    <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
    <div
      class="days"
      bind:this={grid}
      role="group"
      aria-label={monthLabel}
      onkeydown={onGridKey}
    >
      {#each days as { date, inMonth } (date.getTime())}
        {@const isSel = selected !== null && sameDay(date, selected)}
        {@const isToday = sameDay(date, today)}
        <button
          type="button"
          class="day"
          class:outside={!inMonth}
          class:today={isToday}
          class:selected={isSel}
          tabindex={sameDay(date, focusDate) ? 0 : -1}
          aria-pressed={isSel}
          aria-current={isToday ? "date" : undefined}
          aria-label={date.toLocaleDateString(undefined, {
            weekday: "long",
            month: "long",
            day: "numeric",
            year: "numeric",
          })}
          onclick={() => pick(date)}
          onfocus={() => (focusDate = date)}
        >
          {date.getDate()}
        </button>
      {/each}
    </div>

    {#if selected}
      <button type="button" class="clear" onclick={() => pick(null)}>
        <Icon name="close" size={14} />
        Clear date
      </button>
    {/if}
  </div>
{/if}

<style>
  .date {
    justify-content: flex-start;
    gap: var(--space-2);
    width: 100%;
    padding: 0 var(--space-2) 0 var(--space-3);
    text-align: left;
    font-weight: 500;
    box-shadow: none;
    font-variant-numeric: tabular-nums;
  }

  .date-text {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .date :global(.date-icon),
  .date :global(.date-chevron) {
    color: var(--text-muted);
    flex-shrink: 0;
  }

  .date.empty .date-text {
    color: var(--text-muted);
  }

  .date.overdue,
  .date.overdue :global(.date-icon) {
    color: var(--danger);
  }

  .date-field {
    background: var(--surface-sunken);
    border: var(--border-width) solid var(--border);
  }

  .date-field:hover:not(:disabled) {
    background: var(--surface-sunken);
    border-color: var(--text-muted);
  }

  .date-field.open {
    border-color: var(--focus);
    box-shadow: 0 0 0 3px var(--accent-soft);
  }

  .date-chip {
    width: auto;
    min-height: 2.125rem;
    padding: 0 0.75rem;
    border-radius: var(--radius-full);
    background: var(--surface-sunken);
    border: var(--border-width) solid transparent;
    font-size: var(--text-small);
  }

  .date-chip:hover:not(:disabled),
  .date-chip.open {
    background:
      linear-gradient(var(--hover), var(--hover)) padding-box,
      linear-gradient(var(--line-strong), var(--line-strong)) border-box;
  }

  /* ── Panel ───────────────────────────────────────────────────── */
  .picker {
    z-index: var(--layer-popover);
    width: 18.5rem;
    padding: var(--space-3);
    border-radius: var(--radius-lg);
    border: var(--border-width) solid transparent;
    background:
      var(--card-fill) padding-box,
      var(--card-edge) border-box;
    box-shadow: var(--shadow-float);
    outline: none;
    overflow-y: auto;
  }

  .quick {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 4px;
    padding-bottom: var(--space-3);
    margin-bottom: var(--space-2);
    border-bottom: var(--border-width) solid var(--line);
  }

  .quick button {
    min-height: 2rem;
    padding: 0 var(--space-2);
    font-size: var(--text-xs);
    border-radius: var(--radius-sm);
  }

  .month {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-bottom: var(--space-2);
  }

  .month-label {
    font-size: var(--text-small);
    font-weight: 580;
    color: var(--text);
  }

  .nav-btn {
    width: 2rem;
    min-height: 2rem;
    padding: 0;
    border: 0;
    background: transparent;
    box-shadow: none;
    color: var(--text-muted);
  }

  .nav-btn:hover:not(:disabled) {
    background: var(--hover);
    color: var(--text);
  }

  .nav-btn :global(.flip) {
    transform: rotate(180deg);
  }

  .weekdays,
  .days {
    display: grid;
    grid-template-columns: repeat(7, 1fr);
    gap: 2px;
  }

  .weekdays span {
    display: grid;
    place-items: center;
    height: 1.75rem;
    font-size: 0.6875rem;
    font-weight: 600;
    color: var(--text-muted);
  }

  .day {
    min-height: 2.25rem;
    padding: 0;
    border: 0;
    border-radius: var(--radius-sm);
    background: transparent;
    box-shadow: none;
    font-size: var(--text-small);
    font-weight: 480;
    color: var(--text);
    font-variant-numeric: tabular-nums;
  }

  .day:hover:not(:disabled) {
    background: var(--hover);
  }

  .day.outside {
    color: var(--text-muted);
    opacity: 0.55;
  }

  .day.today {
    color: var(--accent);
    font-weight: 680;
    box-shadow: inset 0 0 0 1px var(--accent-soft);
  }

  .day.selected,
  .day.selected:hover:not(:disabled) {
    background: var(--primary-fill);
    color: var(--button-text);
    font-weight: 650;
    box-shadow:
      inset 0 1px 0 oklch(1 0 0 / 0.3),
      0 4px 12px -4px var(--glow-accent);
  }

  .clear {
    width: 100%;
    margin-top: var(--space-2);
    min-height: 2.25rem;
    font-size: var(--text-small);
    color: var(--text-secondary);
    background: transparent;
    border-color: transparent;
    box-shadow: none;
  }

  .clear:hover:not(:disabled) {
    color: var(--danger);
    background: var(--danger-soft);
  }
</style>
