<script lang="ts">
  import Feedback from "./ui/Feedback.svelte";
  import Icon, { type IconName } from "./ui/Icon.svelte";
  import { accents, preferences } from "./lib/preferences.svelte";

  const themeCards: {
    value: "system" | "light" | "dark";
    label: string;
    hint: string;
    icon: IconName;
  }[] = [
    {
      value: "system",
      label: "System",
      hint: "Follows your device",
      icon: "monitor",
    },
    { value: "light", label: "Light", hint: "Bright surfaces", icon: "sun" },
    { value: "dark", label: "Dark", hint: "Low-light surfaces", icon: "moon" },
  ];
</script>

<section class="panel panel-pad pref-panel" aria-labelledby="appearance">
  <div class="pref-head">
    <h2 id="appearance" tabindex="-1">Appearance</h2>
    <p>Applies immediately and is saved only on this browser.</p>
  </div>

  <fieldset class="group" aria-describedby="theme-feedback">
    <legend class="field-label">Theme</legend>
    <div class="themes">
      {#each themeCards as card (card.value)}
        <label class="theme-card" data-preview={card.value}>
          <input
            type="radio"
            name="theme"
            value={card.value}
            checked={preferences.theme === card.value}
            onchange={() => preferences.setTheme(card.value)}
          />
          <span class="preview" aria-hidden="true">
            <span class="pv-side"></span>
            <span class="pv-main">
              <span class="pv-bar"></span>
              <span class="pv-row"></span>
              <span class="pv-row short"></span>
            </span>
          </span>
          <span class="theme-meta">
            <Icon name={card.icon} size={15} />
            <span class="theme-name">{card.label}</span>
            <span class="theme-hint">{card.hint}</span>
          </span>
        </label>
      {/each}
    </div>
  </fieldset>

  <fieldset class="group">
    <legend class="field-label">Accent colour</legend>
    <div class="swatches">
      {#each accents as accent (accent.value)}
        <label class="swatch" style:--swatch-hue={accent.hue}>
          <input
            type="radio"
            name="accent"
            value={accent.value}
            checked={preferences.accent === accent.value}
            onchange={() => preferences.setAccent(accent.value)}
          />
          <span class="swatch-color" aria-hidden="true"></span>
          <span class="swatch-name">{accent.label}</span>
        </label>
      {/each}
    </div>
  </fieldset>

  <div id="theme-feedback">
    {#if preferences.storageError}
      <Feedback tone="warning" announcement="alert"
        >Theme storage is unavailable. Changes apply to this page but may not
        survive a reload.</Feedback
      >
    {:else if preferences.saved}
      <Feedback tone="success">Theme saved on this browser.</Feedback>
    {/if}
  </div>
</section>

<style>
  .pref-panel {
    display: grid;
    gap: var(--space-6);
    align-content: start;
  }

  .pref-head {
    display: grid;
    gap: var(--space-1);
  }

  .pref-head h2 {
    font-size: 1.25rem;
  }

  .pref-head p {
    font-size: var(--text-small);
    color: var(--text-muted);
  }

  .group {
    margin: 0;
    padding: 0;
    border: 0;
    min-width: 0;
  }

  .group legend {
    margin-bottom: var(--space-3);
  }

  /* ── Theme cards: a miniature of the interface in each scheme ── */
  .themes {
    display: grid;
    grid-template-columns: repeat(3, minmax(0, 1fr));
    gap: var(--space-3);
  }

  .theme-card {
    --pv-ground: oklch(0.97 0.01 var(--brand-hue));
    --pv-panel: oklch(1 0 0);
    --pv-line: oklch(0.88 0.02 var(--brand-hue));
    position: relative;
    display: grid;
    gap: var(--space-3);
    padding: var(--space-2);
    border-radius: var(--radius-lg);
    border: var(--border-width) solid transparent;
    background:
      var(--inset-fill) padding-box,
      var(--inset-edge) border-box;
    cursor: pointer;
    transition:
      transform var(--duration) var(--ease-spring),
      box-shadow var(--transition-fast);
  }

  .theme-card[data-preview="dark"] {
    --pv-ground: oklch(0.13 0.02 var(--brand-hue));
    --pv-panel: oklch(0.2 0.02 var(--brand-hue));
    --pv-line: oklch(0.3 0.02 var(--brand-hue));
  }

  .theme-card:hover {
    transform: translateY(-2px);
  }

  .theme-card input {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    margin: 0;
    opacity: 0;
    cursor: pointer;
  }

  .theme-card:has(input:checked) {
    background:
      var(--inset-fill) padding-box,
      linear-gradient(var(--accent), var(--accent)) border-box;
    box-shadow: 0 0 0 3px var(--accent-soft);
  }

  .theme-card:has(input:focus-visible) {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
  }

  .preview {
    display: flex;
    gap: 4px;
    height: 4.75rem;
    padding: 6px;
    border-radius: calc(var(--radius-lg) - 6px);
    background: var(--pv-ground);
    overflow: hidden;
  }

  /* System shows both halves of the choice. */
  .theme-card[data-preview="system"] .preview {
    background: linear-gradient(
      115deg,
      oklch(0.97 0.01 var(--brand-hue)) 50%,
      oklch(0.13 0.02 var(--brand-hue)) 50%
    );
  }

  .pv-side {
    width: 22%;
    border-radius: 4px;
    background: var(--pv-panel);
    opacity: 0.8;
  }

  .pv-main {
    flex: 1;
    display: grid;
    align-content: start;
    gap: 5px;
    padding: 5px;
    border-radius: 4px;
    background: var(--pv-panel);
  }

  .theme-card[data-preview="system"] .pv-side,
  .theme-card[data-preview="system"] .pv-main {
    background: oklch(0.6 0.02 var(--brand-hue) / 0.35);
  }

  .pv-bar {
    width: 55%;
    height: 6px;
    border-radius: 3px;
    background: var(--accent);
  }

  .pv-row {
    height: 5px;
    border-radius: 3px;
    background: var(--pv-line);
  }

  .pv-row.short {
    width: 70%;
  }

  .theme-meta {
    display: grid;
    grid-template-columns: auto 1fr;
    align-items: center;
    gap: 0 var(--space-2);
    padding: 0 var(--space-1) var(--space-1);
    color: var(--text-muted);
  }

  .theme-name {
    font-size: var(--text-small);
    font-weight: 580;
    color: var(--text);
  }

  .theme-hint {
    grid-column: 2;
    font-size: var(--text-xs);
    color: var(--text-muted);
  }

  .swatches {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2);
  }

  .swatch {
    position: relative;
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    min-height: var(--control-height);
    padding: 0 var(--space-3) 0 var(--space-2);
    border-radius: var(--radius-full);
    border: var(--border-width) solid var(--line-strong);
    background: var(--surface-sunken);
    cursor: pointer;
    font-size: var(--text-small);
    font-weight: 560;
    color: var(--text-secondary);
    transition:
      border-color var(--transition-fast),
      background-color var(--transition-fast),
      transform var(--duration) var(--ease-spring);
  }

  .swatch:hover {
    background: var(--hover);
    transform: translateY(-1px);
  }

  .swatch input {
    position: absolute;
    inset: 0;
    width: 100%;
    height: 100%;
    margin: 0;
    opacity: 0;
    cursor: pointer;
  }

  .swatch-color {
    width: 1.5rem;
    height: 1.5rem;
    border-radius: 50%;
    background: light-dark(
      oklch(0.5 0.15 var(--swatch-hue)),
      oklch(0.8 0.12 var(--swatch-hue))
    );
    box-shadow: inset 0 0 0 1px oklch(0 0 0 / 0.12);
    transition: transform var(--duration) var(--ease-spring);
  }

  .swatch:has(input:checked) {
    border-color: var(--accent);
    background: var(--accent-soft);
    color: var(--text);
  }

  .swatch:has(input:checked) .swatch-color {
    transform: scale(1.1);
  }

  .swatch:has(input:focus-visible) {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
  }

  @media (max-width: 40rem) {
    .themes {
      gap: var(--space-2);
    }
    .theme-hint {
      display: none;
    }
    .preview {
      height: 3.5rem;
    }
  }
</style>
