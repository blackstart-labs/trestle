<script lang="ts">
  let { size = 32 }: { size?: number } = $props();
</script>

<!-- A trestle: one deck carried by splayed legs and a braced span. -->
<span class="brand-mark" aria-hidden="true">
  <svg
    width={size * 0.62}
    height={size * 0.62}
    viewBox="0 0 24 24"
    fill="none"
    stroke="currentColor"
    stroke-width="2.25"
    stroke-linecap="round"
    stroke-linejoin="round"
  >
    <path d="M2.5 6h19" />
    <path d="M6.5 6 4 20M17.5 6 20 20" />
    <path d="M6.5 6 12 13l5.5-7" />
    <path d="M5.2 14.5h13.6" />
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
