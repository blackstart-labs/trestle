<script lang="ts">
  let { size = 32 }: { size?: number } = $props();
</script>

<!-- A trestle: one deck carried by splayed legs and a braced span. -->
<span
  class="brand-mark"
  style:width="{size}px"
  style:height="{size}px"
  aria-hidden="true"
>
  <svg
    width={size * 0.75}
    height={size * 0.75}
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="1.75"
    stroke-linecap="round"
    stroke-linejoin="round"
  >
    <path d="M3 5h18M7 5 3 20M17 5l4 15M5 13h14M7 5l12 15M17 5 5 20" />
  </svg>
</span>

<style>
  .brand-mark {
    display: inline-grid;
    place-items: center;
    flex-shrink: 0;
    width: 2rem;
    height: 2rem;
    border-radius: 0.625rem;
    color: var(--button-text);
    background: var(--button-bg);
    box-shadow:
      inset 0 1px 0 oklch(1 0 0 / 0.25),
      0 4px 14px -4px var(--accent-glow);
  }
</style>
