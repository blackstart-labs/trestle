<script lang="ts">
  // Four bents (the piers) carry one continuous deck. Each pier stands for
  // a part of the product (plan, work, record, know); the deck is the shared
  // task identity running across all of them. Pure geometry.
  const bents = [70, 210, 350, 490];
  const deckY = 92;
  const chordY = 120;
  const groundY = 270;

  // Truss web under the deck: alternating diagonals between the chords.
  const web = Array.from({ length: 16 }, (_, i) => {
    const x0 = 14 + i * 33.25;
    const x1 = x0 + 33.25;
    return i % 2 === 0
      ? `M${x0} ${chordY} L${x1} ${deckY}`
      : `M${x0} ${deckY} L${x1} ${chordY}`;
  }).join(" ");
</script>

<svg
  class="trestle-art"
  viewBox="0 0 560 290"
  fill="none"
  aria-hidden="true"
  focusable="false"
>
  <path class="ground" d="M0 {groundY}h560" />

  <g class="bents">
    {#each bents as x (x)}
      <!-- Splayed legs with two girts and cross-bracing between them. -->
      <path
        d="M{x - 18} {chordY} L{x - 40} {groundY} M{x + 18} {chordY} L{x +
          40} {groundY}"
      />
      <path d="M{x - 25} 170h50M{x - 33} 222h66" />
      <path
        d="M{x - 18} {chordY} L{x + 25} 170 M{x + 18} {chordY} L{x -
          25} 170 M{x - 25} 170 L{x + 33} 222 M{x + 25} 170 L{x - 33} 222"
      />
    {/each}
  </g>

  <g class="truss">
    <path d="M14 {chordY}h532" />
    <path d={web} />
  </g>

  <path class="deck" d="M14 {deckY}h532" />

  <g class="nodes">
    {#each bents as x, i (x)}
      <circle
        cx={x}
        cy={deckY}
        r="5.5"
        style:animation-delay="{700 + i * 160}ms"
      />
    {/each}
  </g>
</svg>

<style>
  .trestle-art {
    width: 100%;
    height: auto;
    overflow: visible;
  }

  .ground {
    stroke: var(--line-strong);
    stroke-width: 1.5;
    stroke-dasharray: 2 8;
    stroke-linecap: round;
  }

  .bents path {
    stroke: var(--border);
    stroke-width: 1.5;
    stroke-linecap: round;
    opacity: 0.55;
  }

  .truss path {
    stroke: var(--border);
    stroke-width: 1.25;
    stroke-linecap: round;
    opacity: 0.7;
  }

  .deck {
    stroke: var(--accent);
    stroke-width: 3.5;
    stroke-linecap: round;
    stroke-dasharray: 540;
    stroke-dashoffset: 540;
    animation: lay-deck 1.2s var(--ease-out) 150ms forwards;
    filter: drop-shadow(0 0 10px var(--accent-glow));
  }

  .nodes circle {
    fill: var(--surface-raised);
    stroke: var(--accent);
    stroke-width: 2.5;
    opacity: 0;
    transform-box: fill-box;
    transform-origin: center;
    animation: node-in 420ms var(--ease-out) forwards;
  }

  @keyframes lay-deck {
    to {
      stroke-dashoffset: 0;
    }
  }

  @keyframes node-in {
    from {
      opacity: 0;
      transform: scale(0.4);
    }
    to {
      opacity: 1;
      transform: scale(1);
    }
  }

  @media (prefers-reduced-motion: reduce) {
    .deck {
      stroke-dashoffset: 0;
      animation: none;
    }
    .nodes circle {
      opacity: 1;
      animation: none;
    }
  }
</style>
