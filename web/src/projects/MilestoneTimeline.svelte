<script lang="ts">
  import { onMount } from "svelte";
  import type { Milestone } from "./MilestoneModal.svelte";
  import Button from "../ui/Button.svelte";
  import Icon from "../ui/Icon.svelte";
  import Select, { type SelectOption } from "../ui/Select.svelte";
  import { formatDueDate, isOverdue } from "../lib/identity";

  let {
    milestones,
    oneditmilestone,
    onfiltertasks,
    oncreatemilestone,
  }: {
    milestones: Milestone[];
    oneditmilestone: (m: Milestone) => void;
    onfiltertasks: (milestoneId: string) => void;
    oncreatemilestone: () => void;
  } = $props();

  let timeframe = $state<"month" | "quarter" | "year">("month");
  let scrollContainer = $state<HTMLDivElement | null>(null);

  const ROW_HEIGHT = 58;
  const BAR_HEIGHT = 28;
  const BAR_TOP_OFFSET = (ROW_HEIGHT - BAR_HEIGHT) / 2;

  const timeframeOptions: SelectOption[] = [
    { value: "month", label: "Timeframe: Month" },
    { value: "quarter", label: "Timeframe: Quarter" },
    { value: "year", label: "Timeframe: Year" },
  ];

  function getPercent(m: Milestone): number {
    if (!m.total_tasks || m.total_tasks <= 0) return 0;
    return Math.round((m.completed_tasks / m.total_tasks) * 100);
  }

  function parseDate(val?: string | null): Date | null {
    if (!val) return null;
    const [y, m, d] = val.split("-").map(Number);
    if (!y || !m || !d) return null;
    return new Date(Date.UTC(y, m - 1, d));
  }

  // Sort milestones chronologically
  const sortedMilestones = $derived.by(() => {
    return [...milestones].sort((a, b) => {
      const aStart = a.start_date || a.due_date || a.end_date || "9999-12-31";
      const bStart = b.start_date || b.due_date || b.end_date || "9999-12-31";
      if (aStart !== bStart) return aStart.localeCompare(bStart);
      const aEnd = a.due_date || a.end_date || a.start_date || "9999-12-31";
      const bEnd = b.due_date || b.end_date || b.start_date || "9999-12-31";
      return aEnd.localeCompare(bEnd);
    });
  });

  // Determine timeline boundary
  interface TimelineRange {
    startYear: number;
    startMonth: number;
    endYear: number;
    endMonth: number;
    startTimestamp: number;
    endTimestamp: number;
    totalMs: number;
    columns: Array<{
      id: string;
      label: string;
      year: number;
      showYear: boolean;
      startTimestamp: number;
      endTimestamp: number;
    }>;
  }

  const timelineRange = $derived.by<TimelineRange>(() => {
    const now = new Date();
    const todayYear = now.getUTCFullYear();
    const todayMonth = now.getUTCMonth();

    let minYear = todayYear;
    let minMonth = todayMonth;
    let maxYear = todayYear;
    let maxMonth = todayMonth;
    let hasDates = false;

    for (const m of sortedMilestones) {
      const s = parseDate(m.start_date);
      const e = parseDate(m.due_date || m.end_date);
      if (s) {
        hasDates = true;
        if (
          s.getUTCFullYear() < minYear ||
          (s.getUTCFullYear() === minYear && s.getUTCMonth() < minMonth)
        ) {
          minYear = s.getUTCFullYear();
          minMonth = s.getUTCMonth();
        }
        if (
          s.getUTCFullYear() > maxYear ||
          (s.getUTCFullYear() === maxYear && s.getUTCMonth() > maxMonth)
        ) {
          maxYear = s.getUTCFullYear();
          maxMonth = s.getUTCMonth();
        }
      }
      if (e) {
        hasDates = true;
        if (
          e.getUTCFullYear() < minYear ||
          (e.getUTCFullYear() === minYear && e.getUTCMonth() < minMonth)
        ) {
          minYear = e.getUTCFullYear();
          minMonth = e.getUTCMonth();
        }
        if (
          e.getUTCFullYear() > maxYear ||
          (e.getUTCFullYear() === maxYear && e.getUTCMonth() > maxMonth)
        ) {
          maxYear = e.getUTCFullYear();
          maxMonth = e.getUTCMonth();
        }
      }
    }

    // Pad months around the data
    let sYear = minYear;
    let sMonth = minMonth - 1;
    if (sMonth < 0) {
      sMonth = 11;
      sYear -= 1;
    }

    let eYear = maxYear;
    let eMonth = maxMonth + 2;
    while (eMonth > 11) {
      eMonth -= 12;
      eYear += 1;
    }

    // Ensure at least 14 months for Month view
    const monthDiff = (eYear - sYear) * 12 + (eMonth - sMonth) + 1;
    if (monthDiff < 14) {
      const needed = 14 - monthDiff;
      eMonth += needed;
      while (eMonth > 11) {
        eMonth -= 12;
        eYear += 1;
      }
    }

    const startTimestamp = Date.UTC(sYear, sMonth, 1);
    const endNextMonth = eMonth === 11 ? 0 : eMonth + 1;
    const endNextYear = eMonth === 11 ? eYear + 1 : eYear;
    const endTimestamp = Date.UTC(endNextYear, endNextMonth, 1);
    const totalMs = Math.max(1, endTimestamp - startTimestamp);

    const monthNames = [
      "January",
      "February",
      "March",
      "April",
      "May",
      "June",
      "July",
      "August",
      "September",
      "October",
      "November",
      "December",
    ];

    const columns: TimelineRange["columns"] = [];
    let curY = sYear;
    let curM = sMonth;
    let prevYear: number | null = null;

    while (curY < eYear || (curY === eYear && curM <= eMonth)) {
      const colStart = Date.UTC(curY, curM, 1);
      const nextM = curM === 11 ? 0 : curM + 1;
      const nextY = curM === 11 ? curY + 1 : curY;
      const colEnd = Date.UTC(nextY, nextM, 1);

      let label = monthNames[curM];
      if (timeframe === "quarter") {
        label = `Q${Math.floor(curM / 3) + 1}`;
      } else if (timeframe === "year") {
        label = String(curY);
      }

      columns.push({
        id: `${curY}-${curM}`,
        label,
        year: curY,
        showYear: prevYear !== curY,
        startTimestamp: colStart,
        endTimestamp: colEnd,
      });

      prevYear = curY;
      curM += 1;
      if (curM > 11) {
        curM = 0;
        curY += 1;
      }
    }

    return {
      startYear: sYear,
      startMonth: sMonth,
      endYear: eYear,
      endMonth: eMonth,
      startTimestamp,
      endTimestamp,
      totalMs,
      columns,
    };
  });

  const columnWidth = $derived(
    timeframe === "quarter" ? 200 : timeframe === "year" ? 240 : 160,
  );

  const totalWidth = $derived(timelineRange.columns.length * columnWidth);

  // Milestone layout coordinates
  interface MilestonePosition {
    milestone: Milestone;
    rowIndex: number;
    left: number;
    width: number;
    right: number;
    percent: number;
    isOverdue: boolean;
  }

  const milestonePositions = $derived.by<MilestonePosition[]>(() => {
    const range = timelineRange;
    const now = Date.now();

    return sortedMilestones.map((m, idx) => {
      let sDate = parseDate(m.start_date);
      let eDate = parseDate(m.due_date || m.end_date);

      if (!sDate && eDate) {
        sDate = new Date(eDate.getTime() - 28 * 86400000);
      } else if (sDate && !eDate) {
        eDate = new Date(sDate.getTime() + 28 * 86400000);
      } else if (!sDate && !eDate) {
        // Unscheduled: position at current time
        sDate = new Date(now);
        eDate = new Date(now + 28 * 86400000);
      }

      const sTime = sDate ? sDate.getTime() : now;
      let eTime = eDate ? eDate.getTime() : sTime + 14 * 86400000;
      if (eTime <= sTime) {
        eTime = sTime + 14 * 86400000;
      }

      const rawLeft =
        ((sTime - range.startTimestamp) / range.totalMs) * totalWidth;
      const rawWidth = Math.max(
        68,
        ((eTime - sTime) / range.totalMs) * totalWidth,
      );

      const left = Math.max(0, rawLeft);
      const width = Math.min(totalWidth - left, rawWidth);
      const right = left + width;
      const percent = getPercent(m);
      const overdue =
        m.status === "open" && isOverdue(m.due_date || m.end_date || null);

      return {
        milestone: m,
        rowIndex: idx,
        left,
        width,
        right,
        percent,
        isOverdue: overdue,
      };
    });
  });

  // Calculate SVG connector curves between sequential milestones
  interface ConnectorCurve {
    path: string;
    arrowX: number;
    arrowY: number;
  }

  const connectorCurves = $derived.by<ConnectorCurve[]>(() => {
    const curves: ConnectorCurve[] = [];
    const positions = milestonePositions;

    for (let i = 0; i < positions.length - 1; i++) {
      const from = positions[i];
      const to = positions[i + 1];

      const x1 = from.right;
      const y1 = from.rowIndex * ROW_HEIGHT + ROW_HEIGHT / 2;
      const x2 = to.left;
      const y2 = to.rowIndex * ROW_HEIGHT + ROW_HEIGHT / 2;

      // Draw smooth S-curve connecting end of milestone i to start of milestone i+1
      let path = "";
      if (x2 >= x1) {
        const dx = Math.max(20, Math.min(60, (x2 - x1) / 2));
        path = `M ${x1} ${y1} C ${x1 + dx} ${y1}, ${x2 - dx} ${y2}, ${x2} ${y2}`;
      } else {
        // Waterfall loop-back curve
        const loop = 24;
        const midY = (y1 + y2) / 2;
        path = `M ${x1} ${y1} C ${x1 + loop} ${y1}, ${x1 + loop} ${midY}, ${(x1 + x2) / 2} ${midY} C ${x2 - loop} ${midY}, ${x2 - loop} ${y2}, ${x2} ${y2}`;
      }

      curves.push({
        path,
        arrowX: x2,
        arrowY: y2,
      });
    }

    return curves;
  });

  // Today marker
  const todayX = $derived.by(() => {
    const range = timelineRange;
    const now = Date.now();
    if (now < range.startTimestamp || now > range.endTimestamp) return null;
    return ((now - range.startTimestamp) / range.totalMs) * totalWidth;
  });

  function scrollToToday() {
    if (!scrollContainer || todayX === null) return;
    const target = todayX - scrollContainer.clientWidth / 3;
    scrollContainer.scrollTo({
      left: Math.max(0, target),
      behavior: "smooth",
    });
  }

  function scrollBy(delta: number) {
    if (!scrollContainer) return;
    scrollContainer.scrollBy({
      left: delta,
      behavior: "smooth",
    });
  }

  onMount(() => {
    // Center timeline on today or active milestones
    if (scrollContainer && todayX !== null) {
      const target = todayX - scrollContainer.clientWidth / 3;
      scrollContainer.scrollLeft = Math.max(0, target);
    }
  });
</script>

<div class="timeline-wrapper panel">
  <!-- Timeline Toolbar -->
  <header class="timeline-toolbar">
    <div class="toolbar-left">
      <div class="nav-arrows">
        <Button
          variant="quiet"
          label="Scroll timeline left"
          onclick={() => scrollBy(-columnWidth * 2)}
        >
          {#snippet children()}
            <Icon name="arrow-left" size={15} />
          {/snippet}
        </Button>
        {#if todayX !== null}
          <Button variant="secondary" onclick={scrollToToday}>
            {#snippet children()}Today{/snippet}
          </Button>
        {/if}
        <Button
          variant="quiet"
          label="Scroll timeline right"
          onclick={() => scrollBy(columnWidth * 2)}
        >
          {#snippet children()}
            <Icon name="arrow-right" size={15} />
          {/snippet}
        </Button>
      </div>

      <span class="active-count num">
        {milestones.length}
        {milestones.length === 1 ? "milestone" : "milestones"}
      </span>
    </div>

    <div class="toolbar-right">
      <div class="timeframe-select">
        <Select
          variant="chip"
          label="Timeframe"
          value={timeframe}
          options={timeframeOptions}
          onchange={(v) => (timeframe = v as "month" | "quarter" | "year")}
        />
      </div>
    </div>
  </header>

  <!-- Scrollable Gantt Chart Container -->
  <div
    class="timeline-scroll"
    bind:this={scrollContainer}
    role="region"
    aria-label="Milestone Gantt chart timeline"
  >
    <div class="timeline-canvas" style:width={`${totalWidth}px`}>
      <!-- Months Header Row -->
      <div class="timeline-header" aria-hidden="true">
        {#each timelineRange.columns as col (col.id)}
          <div
            class="header-column"
            style:width={`${columnWidth}px`}
            class:is-new-year={col.showYear}
          >
            {#if col.showYear}
              <span class="year-label">{col.year}</span>
            {/if}
            <span class="month-label">{col.label}</span>
          </div>
        {/each}
      </div>

      <!-- Vertical Grid Lines -->
      <div class="grid-columns-bg" aria-hidden="true">
        {#each timelineRange.columns as col (col.id)}
          <div
            class="grid-column-line"
            style:width={`${columnWidth}px`}
            class:is-new-year={col.showYear}
          ></div>
        {/each}
      </div>

      <!-- Today Line Indicator -->
      {#if todayX !== null}
        <div
          class="today-marker"
          style:left={`${todayX}px`}
          aria-label="Today indicator"
        >
          <span class="today-badge">Today</span>
        </div>
      {/if}

      <!-- SVG Connector Curves (Waterfalls) -->
      <svg
        class="connector-canvas"
        width={totalWidth}
        height={milestonePositions.length * ROW_HEIGHT}
        aria-hidden="true"
      >
        <defs>
          <marker
            id="timeline-arrow"
            viewBox="0 0 8 8"
            refX="6"
            refY="4"
            markerWidth="6"
            markerHeight="6"
            orient="auto"
          >
            <path
              d="M 0 1 L 6 4 L 0 7 z"
              fill="var(--text-muted)"
              opacity="0.75"
            />
          </marker>
        </defs>

        {#each connectorCurves as curve, i (i)}
          <path
            d={curve.path}
            class="connector-path"
            marker-end="url(#timeline-arrow)"
          />
        {/each}
      </svg>

      <!-- Milestone Rows -->
      <div class="timeline-rows" role="list">
        {#each milestonePositions as pos (pos.milestone.id)}
          {@const m = pos.milestone}
          <div
            class="timeline-row"
            role="listitem"
            style:height={`${ROW_HEIGHT}px`}
            style:--milestone-color={m.color || "#6366f1"}
          >
            <!-- Horizontal Gantt Bar -->
            <button
              type="button"
              class="gantt-bar"
              style:left={`${pos.left}px`}
              style:width={`${pos.width}px`}
              style:height={`${BAR_HEIGHT}px`}
              style:top={`${BAR_TOP_OFFSET}px`}
              onclick={() => oneditmilestone(m)}
              aria-label={`Milestone ${m.title}, ${pos.percent}% done. Click to edit.`}
              title={`${m.title}: ${pos.percent}% completed`}
            >
              <!-- Progress fill inside bar -->
              {#if pos.percent > 0}
                <div
                  class="gantt-bar-progress"
                  style:width={`${pos.percent}%`}
                ></div>
              {/if}
            </button>

            <!-- Metadata info pill directly next to the bar -->
            <div
              class="gantt-info-pill"
              style:left={`${pos.right + 12}px`}
              style:top={`${BAR_TOP_OFFSET}px`}
            >
              <span
                class="status-dot"
                style:background={m.color || "var(--accent)"}
                aria-hidden="true"
              ></span>

              <span class="info-label">
                <span class="info-prefix">Milestone:</span>
                <button
                  type="button"
                  class="milestone-name-btn"
                  onclick={() => oneditmilestone(m)}
                >
                  {m.title}
                </button>
                <span class="percent-tag num">({pos.percent}%) Done</span>
              </span>

              {#if m.description}
                <span class="desc-snippet" title={m.description}
                  >{m.description}</span
                >
              {/if}

              {#if m.start_date || m.due_date || m.end_date}
                <span class="date-range num">
                  {#if m.start_date && (m.due_date || m.end_date)}
                    {formatDueDate(m.start_date)} – {formatDueDate(
                      m.due_date || m.end_date || "",
                    )}
                  {:else if m.start_date}
                    From {formatDueDate(m.start_date)}
                  {:else}
                    Due {formatDueDate(m.due_date || m.end_date || "")}
                  {/if}
                </span>
              {/if}

              {#if pos.isOverdue}
                <span class="overdue-chip">Overdue</span>
              {/if}

              <div class="row-hover-actions">
                <Button
                  variant="quiet"
                  label={`View tasks for milestone ${m.title}`}
                  onclick={() => onfiltertasks(m.id)}
                >
                  {#snippet children()}
                    <Icon name="list" size={13} />
                    <span class="action-btn-text">Tasks ({m.total_tasks})</span>
                  {/snippet}
                </Button>
                <Button
                  variant="quiet"
                  label={`Edit milestone ${m.title}`}
                  onclick={() => oneditmilestone(m)}
                >
                  {#snippet children()}
                    <Icon name="pencil" size={13} />
                  {/snippet}
                </Button>
              </div>
            </div>
          </div>
        {/each}
      </div>
    </div>
  </div>
</div>

<style>
  .timeline-wrapper {
    display: flex;
    flex-direction: column;
    width: 100%;
    border-radius: var(--radius-lg);
    border: var(--border-width) solid var(--border);
    background: var(--surface-card);
    overflow: hidden;
  }

  /* ── Toolbar ─────────────────────────────────────────────────── */
  .timeline-toolbar {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: var(--space-3) var(--space-4);
    border-bottom: var(--border-width) solid var(--line);
    background: var(--surface-sunken);
    gap: var(--space-4);
    flex-wrap: wrap;
  }

  .toolbar-left,
  .toolbar-right {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  .nav-arrows {
    display: flex;
    align-items: center;
    gap: var(--space-1);
  }

  .active-count {
    font-size: var(--text-small);
    color: var(--text-muted);
  }

  /* ── Scroll Area ──────────────────────────────────────────────── */
  .timeline-scroll {
    position: relative;
    width: 100%;
    max-width: 100%;
    overflow-x: auto;
    overflow-y: hidden;
    overscroll-behavior-x: contain;
    background: var(--surface);
    scrollbar-gutter: stable;
  }

  .timeline-scroll:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: -2px;
  }

  .timeline-canvas {
    position: relative;
    min-height: 24rem;
  }

  /* ── Header Columns ───────────────────────────────────────────── */
  .timeline-header {
    display: flex;
    height: 3.25rem;
    border-bottom: var(--border-width) solid var(--line-strong);
    background: var(--surface-sunken);
    position: sticky;
    top: 0;
    z-index: 5;
  }

  .header-column {
    display: flex;
    flex-direction: column;
    justify-content: flex-end;
    padding: var(--space-1) var(--space-3) var(--space-2);
    border-right: var(--border-width) solid var(--line);
    flex-shrink: 0;
    user-select: none;
  }

  .header-column.is-new-year {
    border-left: 2px solid var(--line-strong);
  }

  .year-label {
    font-size: var(--text-xs);
    font-weight: 700;
    color: var(--text);
    letter-spacing: 0.02em;
    line-height: 1.2;
  }

  .month-label {
    font-size: var(--text-small);
    font-weight: 560;
    color: var(--text-secondary);
    line-height: 1.3;
  }

  /* ── Grid Columns Background ──────────────────────────────────── */
  .grid-columns-bg {
    position: absolute;
    inset: 3.25rem 0 0 0;
    display: flex;
    pointer-events: none;
    z-index: 1;
  }

  .grid-column-line {
    border-right: var(--border-width) solid var(--line);
    height: 100%;
    flex-shrink: 0;
  }

  .grid-column-line.is-new-year {
    border-left: 2px solid var(--line-strong);
  }

  /* ── Today Marker Line ────────────────────────────────────────── */
  .today-marker {
    position: absolute;
    top: 0;
    bottom: 0;
    width: 2px;
    background: var(--accent);
    z-index: 6;
    pointer-events: none;
    box-shadow: 0 0 10px 1px var(--glow-accent);
  }

  .today-badge {
    position: sticky;
    top: 0.5rem;
    display: inline-block;
    padding: 0.15rem 0.45rem;
    border-radius: var(--radius-xs);
    background: var(--accent);
    color: var(--button-text);
    font-size: 0.6875rem;
    font-weight: 700;
    text-transform: uppercase;
    letter-spacing: 0.04em;
    transform: translateX(-50%);
    box-shadow: var(--shadow-sm);
  }

  /* ── SVG Connectors (Waterfalls) ──────────────────────────────── */
  .connector-canvas {
    position: absolute;
    top: 3.25rem;
    left: 0;
    pointer-events: none;
    z-index: 2;
  }

  .connector-path {
    fill: none;
    stroke: var(--line-strong);
    stroke-width: 1.75;
    stroke-dasharray: 4 3;
    opacity: 0.7;
  }

  /* ── Rows ─────────────────────────────────────────────────────── */
  .timeline-rows {
    position: relative;
    z-index: 3;
    padding-top: 3.25rem;
  }

  .timeline-row {
    position: relative;
    width: 100%;
    border-bottom: var(--border-width) solid var(--line);
    transition: background-color var(--transition-fast);
  }

  .timeline-row:hover {
    background-color: var(--hover);
  }

  /* ── Gantt Bar ────────────────────────────────────────────────── */
  .gantt-bar {
    position: absolute;
    border-radius: var(--radius-sm);
    background: var(--milestone-color, var(--accent));
    border: var(--border-width) solid oklch(1 0 0 / 0.2);
    box-shadow:
      inset 0 1px 0 oklch(1 0 0 / 0.3),
      0 3px 10px -2px var(--milestone-color, var(--accent));
    cursor: pointer;
    overflow: hidden;
    display: flex;
    align-items: center;
    padding: 0 var(--space-2);
    transition:
      transform var(--transition-fast),
      filter var(--transition-fast);
    z-index: 4;
  }

  .gantt-bar:hover {
    transform: translateY(-1px) scaleY(1.04);
    filter: brightness(1.1);
  }

  .gantt-bar:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: 2px;
  }

  .gantt-bar-progress {
    position: absolute;
    inset: 0 auto 0 0;
    background: oklch(0 0 0 / 0.22);
    pointer-events: none;
  }

  /* ── Adjacent Info Pill ───────────────────────────────────────── */
  .gantt-info-pill {
    position: absolute;
    display: flex;
    align-items: center;
    gap: var(--space-2);
    white-space: nowrap;
    z-index: 3;
    pointer-events: auto;
  }

  .status-dot {
    width: 0.625rem;
    height: 0.625rem;
    border-radius: 50%;
    flex-shrink: 0;
  }

  .info-label {
    display: flex;
    align-items: center;
    gap: 0.35rem;
    font-size: var(--text-small);
  }

  .info-prefix {
    color: var(--text-muted);
    font-size: var(--text-xs);
  }

  .milestone-name-btn {
    border: none;
    background: transparent;
    padding: 0;
    color: var(--text);
    font-weight: 600;
    cursor: pointer;
    text-align: left;
  }

  .milestone-name-btn:hover {
    color: var(--accent);
    text-decoration: underline;
  }

  .percent-tag {
    font-size: var(--text-xs);
    font-weight: 500;
    color: var(--text-secondary);
  }

  .desc-snippet {
    font-size: var(--text-xs);
    color: var(--text-muted);
    max-width: 18rem;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .date-range {
    font-size: var(--text-xs);
    color: var(--text-muted);
    padding: 0.1rem 0.4rem;
    border-radius: var(--radius-xs);
    background: var(--surface-sunken);
    border: var(--border-width) solid var(--line);
  }

  .overdue-chip {
    padding: 0.1rem 0.35rem;
    border-radius: var(--radius-xs);
    background: var(--danger-soft);
    color: var(--danger);
    font-size: 0.6875rem;
    font-weight: 600;
    text-transform: uppercase;
    letter-spacing: 0.03em;
  }

  .row-hover-actions {
    display: flex;
    align-items: center;
    gap: var(--space-1);
    opacity: 0;
    transition: opacity var(--transition-fast);
  }

  .timeline-row:hover .row-hover-actions,
  .timeline-row:focus-within .row-hover-actions {
    opacity: 1;
  }

  .action-btn-text {
    font-size: var(--text-xs);
  }

  @media (max-width: 48rem) {
    .timeline-toolbar {
      flex-direction: column;
      align-items: stretch;
    }
    .toolbar-left,
    .toolbar-right {
      justify-content: space-between;
      width: 100%;
    }
  }
</style>
