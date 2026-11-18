<script lang="ts">
  import { onMount } from "svelte";
  import Button from "../ui/Button.svelte";
  import Feedback from "../ui/Feedback.svelte";
  import { api, message } from "../lib/api";
  let {
    workspace,
    onopen,
  }: { workspace: { id: string }; onopen: (id: string) => void } = $props();
  type Node = { id: string; title: string; tags: string[] };
  let nodes = $state<Node[]>([]),
    edges = $state<{ source: string; target: string }[]>([]),
    loading = $state(false),
    error = $state(""),
    query = $state(""),
    tag = $state(""),
    truncated = $state(false),
    zoom = $state(1),
    x = $state(0),
    y = $state(0);
  const points = $derived(
    nodes.map((n, i) => ({
      ...n,
      x: 500 + Math.cos((i * Math.PI * 2) / Math.max(1, nodes.length)) * 380,
      y: 500 + Math.sin((i * Math.PI * 2) / Math.max(1, nodes.length)) * 380,
    })),
  );
  const positions = $derived(new Map(points.map((n) => [n.id, n])));
  async function load() {
    loading = true;
    error = "";
    try {
      const d = await api(
        `/api/v1/workspaces/${workspace.id}/documents/graph?${new URLSearchParams({ q: query, tag })}`,
      );
      nodes = d.nodes;
      edges = d.edges;
      truncated = d.truncated;
    } catch (e) {
      error = message(e);
    } finally {
      loading = false;
    }
  }
  onMount(() => {
    void load();
  });
</script>

<section aria-label="Document knowledge graph">
  <form
    class="graph-controls"
    onsubmit={(e) => {
      e.preventDefault();
      void load();
    }}
  >
    <div class="field">
      <label for="graph-search">Find document titles</label><input
        id="graph-search"
        bind:value={query}
        maxlength="200"
      />
    </div>
    <div class="field">
      <label for="graph-tag">Tag</label><input
        id="graph-tag"
        bind:value={tag}
        maxlength="40"
      />
    </div>
    <Button type="submit" disabled={loading}>Filter graph</Button>
  </form>
  <p>
    Up to 200 recently updated active pages. Lines show links between visible
    pages; filtering can hide connections. Use the document list below for
    keyboard navigation.
  </p>
  {#if truncated}<Feedback
      >More pages match. Narrow the title or tag filter to see other documents.</Feedback
    >{/if}
  {#if error}<Feedback tone="danger">{error}</Feedback><Button onclick={load}
      >Retry graph</Button
    >{/if}
  {#if loading}<Feedback announcement="status"
      >Loading knowledge graph…</Feedback
    >{:else if nodes.length}
    <div class="graph-controls" role="group" aria-label="Graph navigation">
      <Button
        onclick={() => (zoom = Math.min(4, zoom * 1.25))}
        disabled={zoom >= 4}>Zoom in</Button
      ><Button
        onclick={() => (zoom = Math.max(0.5, zoom / 1.25))}
        disabled={zoom <= 0.5}>Zoom out</Button
      ><Button onclick={() => (x -= 100 / zoom)}>Pan left</Button><Button
        onclick={() => (x += 100 / zoom)}>Pan right</Button
      ><Button onclick={() => (y -= 100 / zoom)}>Pan up</Button><Button
        onclick={() => (y += 100 / zoom)}>Pan down</Button
      ><Button
        onclick={() => {
          zoom = 1;
          x = 0;
          y = 0;
        }}>Reset view</Button
      >
    </div>
    <div class="panel graph-frame">
      <svg
        viewBox={`${x + 500 - 500 / zoom} ${y + 500 - 500 / zoom} ${1000 / zoom} ${1000 / zoom}`}
        role="img"
        aria-label={`${nodes.length} documents with ${edges.length} visible links`}
      >
        {#each edges as edge}{@const a = positions.get(edge.source)}{@const b =
            positions.get(edge.target)}{#if a && b}<line
              x1={a.x}
              y1={a.y}
              x2={b.x}
              y2={b.y}
            />{/if}{/each}
        {#each points as node}<g
            ><circle cx={node.x} cy={node.y} r="8" /><text
              x={node.x + 12}
              y={node.y + 4}>{node.title.slice(0, 20)}</text
            ></g
          >{/each}
      </svg>
    </div>
    <div class="graph-list">
      {#each nodes as node}<Button onclick={() => onopen(node.id)}
          >{node.title}</Button
        >{/each}
    </div>
  {:else if !error}<div class="empty">
      <h3>No pages in this graph</h3>
      <p>Create a document or clear your filters.</p>
    </div>{/if}
</section>

<style>
  .graph-controls,
  .graph-list {
    display: flex;
    flex-wrap: wrap;
    align-items: end;
    gap: var(--space-3);
    margin-block: var(--space-4);
  }
  .graph-controls .field {
    flex: 1 1 14rem;
    min-width: 0;
  }
  .graph-frame {
    overflow: hidden;
    aspect-ratio: 1.6;
    min-height: 20rem;
  }
  .graph-frame svg {
    width: 100%;
    height: 100%;
    display: block;
  }
  .graph-frame line {
    stroke: var(--border);
    stroke-width: 1;
  }
  .graph-frame circle {
    fill: var(--accent);
    stroke: var(--surface);
    stroke-width: 2;
  }
  .graph-frame text {
    fill: var(--text);
    font-family: var(--font-body);
    font-size: var(--text-xs);
  }
  .graph-list :global(button) {
    white-space: normal;
    overflow-wrap: anywhere;
    max-width: 100%;
  }
</style>
