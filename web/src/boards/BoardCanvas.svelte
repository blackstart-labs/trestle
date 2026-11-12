<script lang="ts">
  import {
    arrowEnds,
    bounds,
    connectorPath,
    wrapLabel,
    type Scene,
    type Point,
  } from "./scene";
  let {
    scene,
    selected = "",
    selectedIds = [],
    svg = $bindable(),
    viewBox = "0 0 1600 1000",
    scale = 1,
    grid = true,
    connecting = false,
    marquee = null,
    wire = null,
    preview = null,
    onpointerdown,
    onpointermove,
    onpointerup,
    onkeydown,
    ondblclick,
    onwheel,
  }: {
    scene: Scene;
    selected?: string;
    selectedIds?: string[];
    svg?: SVGSVGElement;
    viewBox?: string;
    scale?: number;
    grid?: boolean;
    connecting?: boolean;
    marquee?: { x: number; y: number; width: number; height: number } | null;
    wire?: { from: Point; to: Point } | null;
    preview?: { points: Point[]; color: string; width: number } | null;
    onpointerdown?: (e: PointerEvent) => void;
    onpointermove?: (e: PointerEvent) => void;
    onpointerup?: (e: PointerEvent) => void;
    onkeydown?: (e: KeyboardEvent) => void;
    ondblclick?: (e: MouseEvent) => void;
    onwheel?: (e: WheelEvent) => void;
  } = $props();
  const extent = $derived(bounds(scene.elements));
  const paperWidth = $derived(
    Math.min(
      4000,
      Math.max(1600, Math.ceil((extent.x + extent.width + 80) / 100) * 100),
    ),
  );
  const paperHeight = $derived(
    Math.min(
      4000,
      Math.max(1000, Math.ceil((extent.y + extent.height + 80) / 100) * 100),
    ),
  );
  const selection = $derived(
    selectedIds.length ? selectedIds : selected ? [selected] : [],
  );
</script>

<!-- svelte-ignore a11y_no_noninteractive_tabindex, a11y_no_noninteractive_element_interactions (A keyboard-operated visual editor; the object inspector provides standard-control parity.) -->
<svg
  bind:this={svg}
  {viewBox}
  preserveAspectRatio="none"
  role="application"
  aria-label="Board canvas"
  aria-describedby="board-canvas-help"
  tabindex="0"
  {onpointerdown}
  {onpointermove}
  {onpointerup}
  onpointercancel={onpointerup}
  {onkeydown}
  {ondblclick}
  {onwheel}
>
  <title>Board canvas</title>
  <defs>
    <marker
      id="board-arrow"
      viewBox="0 0 10 10"
      refX="9"
      refY="5"
      markerWidth="7"
      markerHeight="7"
      orient="auto-start-reverse"
      ><path d="M1 1L9 5L1 9" class="arrow-tip" /></marker
    >
    <pattern
      id="board-grid"
      width="20"
      height="20"
      patternUnits="userSpaceOnUse"
      ><path d="M20 0H0V20" class="grid-line" /></pattern
    >
  </defs>
  <rect x="-10000" y="-10000" width="24000" height="24000" class="surround" />
  <rect
    width={paperWidth}
    height={paperHeight}
    class="paper"
    data-board-paper
  />
  {#if grid}<rect
      width={paperWidth}
      height={paperHeight}
      fill="url(#board-grid)"
      class="grid"
    />{/if}
  {#each scene.elements as e (e.id)}
    {@const font = e.fontSize || 18}
    <g
      data-object={e.id}
      data-ink={e.color}
      data-fill={e.fill || "default"}
      class:connector={e.kind === "arrow"}
      class:shape={e.kind !== "arrow" && e.kind !== "stroke"}
      class:stroke={e.kind === "stroke"}
      style:--object-line-width={e.strokeWidth || 2}
      style:--object-font-size={`${font}px`}
      style:--object-weight={e.bold ? 600 : 450}
      style:--object-dash={e.dashed ? "8 5" : "none"}
    >
      {#if e.kind === "arrow"}
        {@const ends = arrowEnds(e, scene)}
        <path class="connector-hit" d={connectorPath(e, scene)} />
        <path
          d={connectorPath(e, scene)}
          marker-end={e.heads === "none" ? undefined : "url(#board-arrow)"}
          marker-start={e.heads === "both" ? "url(#board-arrow)" : undefined}
        />
        {#if e.text}<text
            x={(ends.from.x + ends.to.x) / 2}
            y={(ends.from.y + ends.to.y) / 2 - 14}
            class="edge-label">{e.text}</text
          >{/if}
      {:else if e.kind === "stroke"}<polyline
          points={e.points?.map((p) => `${p.x},${p.y}`).join(" ")}
        />
      {:else}
        {#if e.kind === "rectangle" || e.kind === "frame" || e.kind === "swimlane" || e.kind === "note" || e.kind === "terminator"}
          <rect
            x={e.x}
            y={e.y}
            width={e.width}
            height={e.height}
            rx={e.kind === "terminator"
              ? e.height / 2
              : e.kind === "note"
                ? 2
                : 6}
          />
          {#if e.kind === "swimlane"}<path
              d={`M${e.x},${e.y + e.height / 3}h${e.width} M${e.x},${e.y + (e.height * 2) / 3}h${e.width} M${e.x + 44},${e.y}v${e.height}`}
            />{/if}
          {#if e.kind === "note"}<path
              d={`M${e.x + e.width - 22},${e.y}v22h22`}
            />{/if}
        {:else if e.kind === "ellipse"}<ellipse
            cx={e.x + e.width / 2}
            cy={e.y + e.height / 2}
            rx={e.width / 2}
            ry={e.height / 2}
          />
        {:else if e.kind === "diamond"}<polygon
            points={`${e.x + e.width / 2},${e.y} ${e.x + e.width},${e.y + e.height / 2} ${e.x + e.width / 2},${e.y + e.height} ${e.x},${e.y + e.height / 2}`}
          />
        {:else if e.kind === "parallelogram"}<polygon
            points={`${e.x + e.width * 0.15},${e.y} ${e.x + e.width},${e.y} ${e.x + e.width * 0.85},${e.y + e.height} ${e.x},${e.y + e.height}`}
          />
        {:else if e.kind === "document"}<path
            d={`M${e.x},${e.y}h${e.width}v${e.height * 0.85}q${-e.width / 4},${-e.height * 0.25} ${-e.width / 2},0t${-e.width / 2},0Z`}
          />
        {:else if e.kind === "database"}<path
            d={`M${e.x},${e.y + 16}a${e.width / 2},16 0 0 1 ${e.width},0v${e.height - 32}a${e.width / 2},16 0 0 1 ${-e.width},0Z M${e.x},${e.y + 16}a${e.width / 2},16 0 0 0 ${e.width},0`}
          />
        {:else if e.kind === "text"}<rect
            class="text-hit"
            x={e.x}
            y={e.y}
            width={e.width}
            height={e.height}
          />{/if}
        {@const labelWidth = e.kind === "diamond" ? e.width * 0.65 : e.width}
        {@const lines = wrapLabel(e.text, labelWidth, font)}
        <text
          x={e.align === "left"
            ? e.x + (e.kind === "swimlane" ? 56 : 12)
            : e.align === "right"
              ? e.x + e.width - 12
              : e.x + e.width / 2}
          y={e.kind === "frame" || e.kind === "swimlane"
            ? e.y + font + 8
            : e.y + e.height / 2 - (lines.length - 1) * font * 0.65}
          text-anchor={e.align === "left"
            ? "start"
            : e.align === "right"
              ? "end"
              : "middle"}
        >
          {#each lines as line, i}<tspan
              x={e.align === "left"
                ? e.x + (e.kind === "swimlane" ? 56 : 12)
                : e.align === "right"
                  ? e.x + e.width - 12
                  : e.x + e.width / 2}
              dy={i ? font * 1.3 : 0}>{line}</tspan
            >{/each}
        </text>
      {/if}
    </g>
  {/each}
  {#if preview && preview.points.length > 1}<g
      data-editor-only
      class="stroke"
      data-ink={preview.color}
      style:--object-line-width={preview.width}
      ><polyline
        points={preview.points.map((p) => `${p.x},${p.y}`).join(" ")}
      /></g
    >{/if}
  <g data-editor-only class="editor-overlay">
    {#each scene.elements.filter((e) => selection.includes(e.id)) as e (e.id)}
      {@const box = bounds([e])}
      {#if e.kind === "arrow"}<path
          class="selected-edge"
          d={connectorPath(e, scene)}
        />
      {:else}
        <rect
          class="selection"
          x={box.x - 4 / scale}
          y={box.y - 4 / scale}
          width={box.width + 8 / scale}
          height={box.height + 8 / scale}
        />
        {#if selection.length === 1 && !e.locked && e.kind !== "stroke" && !connecting}
          {#each ["nw", "ne", "se", "sw"] as handle}{@const x = handle.includes(
              "w",
            )
              ? e.x
              : e.x + e.width}{@const y = handle.includes("n")
              ? e.y
              : e.y + e.height}<rect
              data-handle={handle}
              data-target={e.id}
              class="handle-hit"
              x={x - 22 / scale}
              y={y - 22 / scale}
              width={44 / scale}
              height={44 / scale}
            /><rect
              class="handle"
              x={x - 4 / scale}
              y={y - 4 / scale}
              width={8 / scale}
              height={8 / scale}
            />{/each}
        {/if}
      {/if}
    {/each}
    {#if connecting}{#each scene.elements.filter((e) => e.kind !== "arrow" && e.kind !== "stroke") as e (e.id)}{#each ["top", "right", "bottom", "left"] as port}{@const x =
            port === "left"
              ? e.x
              : port === "right"
                ? e.x + e.width
                : e.x + e.width / 2}{@const y =
            port === "top"
              ? e.y
              : port === "bottom"
                ? e.y + e.height
                : e.y + e.height / 2}<circle
            data-port={port}
            data-target={e.id}
            class="port-hit"
            cx={x}
            cy={y}
            r={22 / scale}
          /><circle
            class="port"
            cx={x}
            cy={y}
            r={5 / scale}
          />{/each}{/each}{/if}
    {#if marquee}<rect
        class="marquee"
        x={marquee.x}
        y={marquee.y}
        width={marquee.width}
        height={marquee.height}
      />{/if}
    {#if wire}<path
        class="wire"
        d={`M${wire.from.x},${wire.from.y}L${wire.to.x},${wire.to.y}`}
      />{/if}
  </g>
</svg>

<style>
  svg {
    display: block;
    width: 100%;
    height: 100%;
    touch-action: none;
    user-select: none;
  }
  .surround {
    fill: var(--surface-sunken);
  }
  .paper {
    fill: var(--surface-raised);
    stroke: var(--line-strong);
    stroke-width: 1;
  }
  .grid {
    pointer-events: none;
  }
  .grid-line {
    fill: none;
    stroke: var(--line);
    stroke-width: 0.6;
  }
  [data-object],
  .stroke {
    --ink: var(--text);
    --fill: var(--surface-raised);
    color: var(--ink);
  }
  [data-ink="accent"] {
    --ink: var(--accent);
  }
  [data-ink="info"] {
    --ink: var(--info);
  }
  [data-ink="success"] {
    --ink: var(--success);
  }
  [data-ink="warning"] {
    --ink: var(--warning);
  }
  [data-ink="danger"] {
    --ink: var(--danger);
  }
  [data-fill="none"] {
    --fill: none;
  }
  [data-fill="ink"] {
    --fill: var(--selected);
  }
  [data-fill="accent"] {
    --fill: var(--accent-soft);
  }
  [data-fill="info"] {
    --fill: var(--info-soft);
  }
  [data-fill="success"] {
    --fill: var(--success-soft);
  }
  [data-fill="warning"] {
    --fill: var(--warning-soft);
  }
  [data-fill="danger"] {
    --fill: var(--danger-soft);
  }
  .shape > :is(rect, ellipse, polygon, path) {
    fill: var(--fill);
    stroke: var(--ink);
    stroke-width: var(--object-line-width);
    stroke-dasharray: var(--object-dash);
  }
  .connector > path,
  .stroke > polyline {
    fill: none;
    stroke: var(--ink);
    stroke-width: var(--object-line-width);
    stroke-dasharray: var(--object-dash);
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  .connector > .connector-hit {
    stroke: transparent;
    stroke-width: 18;
    pointer-events: stroke;
  }
  .shape > .text-hit {
    fill: transparent;
    stroke: none;
  }
  text {
    fill: var(--text);
    stroke: none;
    font: var(--object-font-size) var(--font-body);
    font-weight: var(--object-weight);
    dominant-baseline: middle;
    pointer-events: none;
  }
  .edge-label {
    text-anchor: middle;
    paint-order: stroke;
    stroke: var(--surface-raised);
    stroke-width: 6;
    stroke-linejoin: round;
  }
  .arrow-tip {
    fill: none;
    stroke: context-stroke;
    stroke-width: 1.5;
    stroke-linecap: round;
    stroke-linejoin: round;
  }
  .selection {
    fill: none;
    stroke: var(--accent);
    stroke-width: 1.5;
    stroke-dasharray: 5 3;
    vector-effect: non-scaling-stroke;
    pointer-events: none;
  }
  .handle {
    fill: var(--surface-float);
    stroke: var(--accent);
    stroke-width: 1.5;
    vector-effect: non-scaling-stroke;
    pointer-events: none;
  }
  .handle-hit,
  .port-hit {
    fill: transparent;
    stroke: none;
    cursor: crosshair;
  }
  .port {
    fill: var(--accent);
    stroke: var(--surface-raised);
    stroke-width: 1.5;
    vector-effect: non-scaling-stroke;
    pointer-events: none;
  }
  .selected-edge {
    fill: none;
    stroke: var(--accent);
    stroke-width: 3;
    stroke-dasharray: 5 3;
    vector-effect: non-scaling-stroke;
    pointer-events: none;
  }
  .marquee {
    fill: var(--accent-soft);
    fill-opacity: 0.45;
    stroke: var(--accent);
    vector-effect: non-scaling-stroke;
    pointer-events: none;
  }
  .wire {
    fill: none;
    stroke: var(--accent);
    stroke-width: 2;
    stroke-dasharray: 5 3;
    vector-effect: non-scaling-stroke;
    pointer-events: none;
  }
</style>
