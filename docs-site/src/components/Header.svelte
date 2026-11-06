<script lang="ts">
  interface Props {
    theme: "light" | "dark";
    ontoggleTheme: () => void;
    onopenSearch: () => void;
    ontoggleMobileMenu: () => void;
  }

  let { theme, ontoggleTheme, onopenSearch, ontoggleMobileMenu }: Props = $props();
</script>

<header class="top-nav">
  <div class="nav-left">
    <button class="mobile-menu-btn" onclick={ontoggleMobileMenu} aria-label="Toggle menu">
      <svg viewBox="0 0 24 24" width="20" height="20" fill="none" stroke="currentColor" stroke-width="2"><line x1="3" y1="12" x2="21" y2="12"/><line x1="3" y1="6" x2="21" y2="6"/><line x1="3" y1="18" x2="21" y2="18"/></svg>
    </button>

    <div class="brand">
      <div class="brand-icon">
        <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2.2">
          <path d="M2 6h20M4 6l4 14M20 6l-4 14M6 13h12" />
        </svg>
      </div>
      <span class="brand-name">Trestle</span>
      <span class="badge docs-badge">Docs</span>
    </div>
  </div>

  <div class="nav-center">
    <button class="search-trigger-btn" onclick={onopenSearch}>
      <svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><line x1="21" y1="21" x2="16.65" y2="16.65"/></svg>
      <span>Quick search docs...</span>
      <kbd>⌘K</kbd>
    </button>
  </div>

  <div class="nav-right">
    <button class="icon-btn theme-toggle" onclick={ontoggleTheme} aria-label="Toggle theme">
      {#if theme === "dark"}
        <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2"><circle cx="12" cy="12" r="5"/><line x1="12" y1="1" x2="12" y2="3"/><line x1="12" y1="21" x2="12" y2="23"/><line x1="4.22" y1="4.22" x2="5.64" y2="5.64"/><line x1="18.36" y1="18.36" x2="19.78" y2="19.78"/><line x1="1" y1="12" x2="3" y2="12"/><line x1="21" y1="12" x2="23" y2="12"/><line x1="4.22" y1="19.78" x2="5.64" y2="18.36"/><line x1="18.36" y1="5.64" x2="19.78" y2="4.22"/></svg>
      {:else}
        <svg viewBox="0 0 24 24" width="18" height="18" fill="none" stroke="currentColor" stroke-width="2"><path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z"/></svg>
      {/if}
    </button>

    <a href="https://github.com/blackstart-labs/trestle" target="_blank" rel="noopener noreferrer" class="icon-btn github-link" aria-label="GitHub Repository">
      <svg viewBox="0 0 24 24" width="18" height="18" fill="currentColor">
        <path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0 0 24 12c0-6.63-5.37-12-12-12z"/>
      </svg>
    </a>
  </div>
</header>

<style>
  .top-nav {
    height: 56px;
    border-bottom: 1px solid var(--line);
    background: var(--surface-nav);
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 0 1.5rem;
    position: sticky;
    top: 0;
    z-index: 100;
  }
  .nav-left {
    display: flex;
    align-items: center;
    gap: 0.85rem;
  }
  .mobile-menu-btn {
    display: none;
    background: transparent;
    border: none;
    color: var(--text);
    cursor: pointer;
    padding: 0.35rem;
  }
  @media (max-width: 860px) {
    .mobile-menu-btn {
      display: block;
    }
  }
  .brand {
    display: flex;
    align-items: center;
    gap: 0.65rem;
  }
  .brand-icon {
    width: 28px;
    height: 28px;
    border-radius: var(--radius-sm);
    background: var(--accent);
    color: oklch(0.12 0.02 var(--brand-hue));
    display: flex;
    align-items: center;
    justify-content: center;
  }
  .brand-name {
    font-weight: 700;
    font-size: 1.15rem;
    letter-spacing: -0.02em;
    color: var(--text);
  }
  .docs-badge {
    font-size: 0.7rem;
    padding: 0.1rem 0.45rem;
  }
  .nav-center {
    flex: 1;
    max-width: 420px;
    margin: 0 1.5rem;
  }
  @media (max-width: 680px) {
    .nav-center {
      display: none;
    }
  }
  .search-trigger-btn {
    width: 100%;
    display: flex;
    align-items: center;
    gap: 0.65rem;
    background: var(--surface-sunken);
    border: 1px solid var(--line);
    border-radius: var(--radius-md);
    padding: 0.45rem 0.75rem;
    color: var(--text-muted);
    font-size: 0.84rem;
    cursor: pointer;
    transition: all 0.15s ease;
  }
  .search-trigger-btn:hover {
    border-color: var(--border);
    color: var(--text-secondary);
  }
  .search-trigger-btn kbd {
    margin-left: auto;
    font-size: 0.7rem;
    padding: 0.1rem 0.35rem;
    border: 1px solid var(--line);
    border-radius: var(--radius-xs);
    background: var(--surface-raised);
  }
  .nav-right {
    display: flex;
    align-items: center;
    gap: 0.5rem;
  }
  .icon-btn {
    display: flex;
    align-items: center;
    justify-content: center;
    width: 34px;
    height: 34px;
    border-radius: var(--radius-md);
    background: transparent;
    border: 1px solid transparent;
    color: var(--text-secondary);
    cursor: pointer;
    transition: all 0.15s ease;
  }
  .icon-btn:hover {
    background: var(--hover);
    color: var(--text);
    border-color: var(--line);
  }
</style>

