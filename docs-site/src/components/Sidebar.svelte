<script lang="ts">
  import { SECTIONS, CATEGORIES, type DocSection } from "../content";

  interface Props {
    activeId: string;
    onselect: (id: string) => void;
    mobileOpen: boolean;
    oncloseMobile: () => void;
  }

  let { activeId, onselect, mobileOpen, oncloseMobile }: Props = $props();

  function handleSelect(id: string) {
    onselect(id);
    oncloseMobile();
  }
</script>

<aside class="sidebar {mobileOpen ? 'mobile-open' : ''}">
  <nav class="nav-content">
    <div class="category-group">
      <button
        class="nav-link home-showcase-btn {activeId === 'home' ? 'active' : ''}"
        onclick={() => handleSelect('home')}
      >
        <span class="link-label">✨ Home Showcase</span>
      </button>
    </div>

    {#each CATEGORIES as category}
      <div class="category-group">
        <h3 class="category-title">{category}</h3>
        <ul class="nav-list">
          {#each SECTIONS.filter(s => s.category === category && s.id !== 'home') as item}
            <li>
              <button
                class="nav-link {activeId === item.id ? 'active' : ''}"
                onclick={() => handleSelect(item.id)}
              >
                <span class="link-label">{item.title}</span>
                {#if item.badge}
                  <span class="link-badge">{item.badge}</span>
                {/if}
              </button>
            </li>
          {/each}
        </ul>
      </div>
    {/each}
  </nav>
</aside>

{#if mobileOpen}
  <div class="mobile-backdrop" onclick={oncloseMobile} role="presentation"></div>
{/if}

<style>
  .sidebar {
    width: 280px;
    height: calc(100vh - 56px);
    position: sticky;
    top: 56px;
    background: var(--surface-nav);
    border-right: 1px solid var(--line);
    overflow-y: auto;
    padding: 1.25rem 0.85rem;
    flex-shrink: 0;
  }
  @media (max-width: 860px) {
    .sidebar {
      position: fixed;
      top: 56px;
      left: 0;
      bottom: 0;
      z-index: 200;
      transform: translateX(-100%);
      transition: transform 0.25s cubic-bezier(0.16, 1, 0.3, 1);
      box-shadow: 10px 0 30px rgba(0,0,0,0.5);
    }
    .sidebar.mobile-open {
      transform: translateX(0);
    }
  }
  .mobile-backdrop {
    position: fixed;
    inset: 56px 0 0 0;
    background: rgba(0, 0, 0, 0.5);
    z-index: 190;
  }
  .category-group {
    margin-bottom: 1.5rem;
  }
  .category-title {
    font-size: 0.72rem;
    text-transform: uppercase;
    letter-spacing: 0.06em;
    font-weight: 700;
    color: var(--text-muted);
    padding: 0 0.65rem 0.45rem 0.65rem;
  }
  .nav-list {
    list-style: none;
  }
  .nav-link {
    width: 100%;
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0.45rem 0.65rem;
    border-radius: var(--radius-md);
    border: none;
    background: transparent;
    color: var(--text-secondary);
    font-size: 0.88rem;
    font-weight: 500;
    text-align: left;
    cursor: pointer;
    transition: all 0.15s ease;
  }
  .nav-link:hover {
    background: var(--hover);
    color: var(--text);
  }
  .nav-link.active {
    background: var(--accent-soft);
    color: var(--accent);
    font-weight: 600;
  }
  .link-badge {
    font-size: 0.68rem;
    font-weight: 600;
    padding: 0.1rem 0.35rem;
    border-radius: var(--radius-xs);
    background: var(--surface-sunken);
    color: var(--text-muted);
    border: 1px solid var(--line);
  }
  .nav-link.active .link-badge {
    background: var(--accent);
    color: oklch(0.12 0.02 var(--brand-hue));
    border-color: transparent;
  }
</style>

