<script lang="ts">
  import TrestleArt from "./ui/TrestleArt.svelte";
  import Icon from "./ui/Icon.svelte";
  import type { User } from "./auth/SignIn.svelte";

  let {
    user,
    onopenworkspaces,
    onsignin,
  }: {
    user: User | null;
    onopenworkspaces: () => void;
    onsignin: () => void;
  } = $props();

  // The four piers of the trestle: what ships today and what is planned.
  const piers = [
    { name: "Plan", detail: "Projects, tasks and workflows", ready: true },
    { name: "Work", detail: "Boards, assignees and discussion", ready: true },
    { name: "Record", detail: "Timers and timesheets", ready: false },
    { name: "Know", detail: "Linked documents", ready: false },
  ];
</script>

<section class="hero" aria-labelledby="overview">
  <div class="hero-copy">
    <h1 id="overview" tabindex="-1">Trestle</h1>
    <p class="hero-lead">
      One place for projects, time and connected knowledge.
    </p>
    <p class="hero-note">
      {#if user}
        You are signed in as <strong>{user.display_name}</strong>. Open a
        workspace to pick up where you left off.
      {:else}
        Identity and workspace management are enabled. Sign in to access your
        workspaces.
      {/if}
    </p>
    <div class="hero-actions">
      {#if user}
        <a
          class="btn btn-primary"
          href="#workspaces"
          onclick={onopenworkspaces}
        >
          Open workspaces
          <Icon name="arrow-right" size={16} />
        </a>
      {:else}
        <a class="btn btn-primary" href="#signin" onclick={onsignin}>
          Sign in to Trestle
          <Icon name="arrow-right" size={16} />
        </a>
      {/if}
    </div>
  </div>

  <figure class="hero-figure">
    <TrestleArt />
    <figcaption class="piers">
      {#each piers as pier (pier.name)}
        <span class="pier" class:pending={!pier.ready}>
          <span class="pier-name">{pier.name}</span>
          <span class="pier-detail">{pier.detail}</span>
          <span class="pier-state">{pier.ready ? "Available" : "Planned"}</span>
        </span>
      {/each}
    </figcaption>
  </figure>
</section>

<style>
  .hero {
    background:
      var(--card-fill) padding-box,
      var(--card-edge) border-box;
    border: var(--border-width) solid transparent;
    position: relative;
    display: grid;
    grid-template-columns: minmax(0, 0.9fr) minmax(0, 1.1fr);
    gap: var(--space-8);
    align-items: center;
    padding: clamp(1.5rem, 1rem + 2.5vw, 3rem);
    margin-bottom: var(--space-6);
    border-radius: var(--radius-xl);
    background:
      radial-gradient(
        60% 90% at 100% 0%,
        light-dark(
          oklch(0.94 0.04 var(--brand-hue) / 0.7),
          oklch(0.3 0.08 var(--brand-hue) / 0.35)
        ),
        transparent 70%
      ),
      var(--surface-raised);
    box-shadow: var(--shadow-card);
    overflow: hidden;
  }

  h1 {
    font-size: var(--text-display);
    font-weight: 520;
    letter-spacing: var(--tracking-display);
    line-height: 0.95;
  }

  .hero-lead {
    margin-top: var(--space-4);
    max-width: 26ch;
    font-size: clamp(1.125rem, 1rem + 0.6vw, 1.5rem);
    line-height: 1.3;
    color: var(--text);
    font-weight: 420;
    letter-spacing: -0.018em;
  }

  .hero-note {
    margin-top: var(--space-4);
    max-width: 44ch;
  }

  .hero-actions {
    display: flex;
    gap: var(--space-3);
    flex-wrap: wrap;
    margin-top: var(--space-6);
  }

  .hero-figure {
    margin: 0;
    display: grid;
    gap: var(--space-4);
  }

  .piers {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: var(--space-2);
  }

  .pier {
    background:
      var(--inset-fill) padding-box,
      var(--inset-edge) border-box;
    border: var(--border-width) solid transparent;
    display: grid;
    gap: 0.125rem;
    align-content: start;
    padding: var(--space-3);
    border-radius: var(--radius);
  }

  .pier-name {
    font-weight: 620;
    color: var(--text);
  }

  .pier-detail {
    font-size: var(--text-xs);
    color: var(--text-muted);
    line-height: 1.35;
  }

  .pier-state {
    margin-top: var(--space-2);
    font-size: var(--text-xs);
    font-weight: 600;
    color: var(--success);
  }

  .pier.pending {
    background: transparent;
    border: var(--border-width) dashed var(--line-strong);
  }

  /* Light from the upper right, and a diagonal sheen across the hero. */
  .hero::before {
    content: "";
    position: absolute;
    inset: 0;
    pointer-events: none;
    background:
      linear-gradient(
        115deg,
        transparent 40%,
        var(--sheen) 55%,
        transparent 68%
      ),
      radial-gradient(60% 90% at 100% 0%, var(--glow-accent), transparent 70%);
    opacity: 0.9;
  }

  .hero > * {
    position: relative;
  }

  .pier.pending .pier-state {
    color: var(--text-muted);
  }

  @media (max-width: 64rem) {
    .hero {
      grid-template-columns: minmax(0, 1fr);
    }
  }

  @media (max-width: 40rem) {
    .piers {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }
</style>
