<script lang="ts">
  import Header from "./components/Header.svelte";
  import Sidebar from "./components/Sidebar.svelte";
  import SearchModal from "./components/SearchModal.svelte";
  import DocContent from "./pages/DocContent.svelte";
  import Home from "./pages/Home.svelte";
  import { SECTIONS } from "./content";

  let activeSection = $state("home");
  let theme: "light" | "dark" = $state("dark");
  let searchOpen = $state(false);
  let mobileOpen = $state(false);

  // Sync hash with activeSection
  $effect(() => {
    const handleHash = () => {
      const hash = window.location.hash.replace(/^#/, "");
      if (hash && (hash === "home" || SECTIONS.some(s => s.id === hash))) {
        activeSection = hash;
      } else if (!hash) {
        activeSection = "home";
      }
    };
    handleHash();
    window.addEventListener("hashchange", handleHash);
    return () => window.removeEventListener("hashchange", handleHash);
  });

  function selectSection(id: string) {
    activeSection = id;
    window.location.hash = id;
    window.scrollTo({ top: 0, behavior: "smooth" });
  }

  function toggleTheme() {
    theme = theme === "dark" ? "light" : "dark";
    document.documentElement.setAttribute("data-theme", theme);
  }

  // Keyboard shortcut Ctrl+K / Cmd+K to open search
  function handleKeydown(e: KeyboardEvent) {
    if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === "k") {
      e.preventDefault();
      searchOpen = !searchOpen;
    }
  }

  let currentIndex = $derived(SECTIONS.findIndex(s => s.id === activeSection));
  let prevSection = $derived(currentIndex > 1 ? SECTIONS[currentIndex - 1] : null);
  let nextSection = $derived(currentIndex >= 1 && currentIndex < SECTIONS.length - 1 ? SECTIONS[currentIndex + 1] : (currentIndex === 0 ? SECTIONS[1] : null));
</script>

<svelte:window onkeydown={handleKeydown} />

<div class="docs-app" data-theme={theme}>
  {#if activeSection === "home"}
    <Home
      onnavigate={selectSection}
      {theme}
      ontoggleTheme={toggleTheme}
      onopenSearch={() => (searchOpen = true)}
    />
  {:else}
    <Header
      {theme}
      ontoggleTheme={toggleTheme}
      onopenSearch={() => (searchOpen = true)}
      ontoggleMobileMenu={() => (mobileOpen = !mobileOpen)}
      onnavigateHome={() => selectSection("home")}
    />

    <div class="layout-container">
      <Sidebar
        activeId={activeSection}
        onselect={selectSection}
        {mobileOpen}
        oncloseMobile={() => (mobileOpen = false)}
      />

      <main class="content-pane">
        <div class="content-inner">
          <DocContent sectionId={activeSection} />

          <div class="doc-pagination">
            {#if prevSection}
              <button class="page-nav prev" onclick={() => selectSection(prevSection.id)}>
                <span class="nav-sub">Previous</span>
                <span class="nav-title">← {prevSection.title}</span>
              </button>
            {:else}
              <button class="page-nav prev" onclick={() => selectSection("home")}>
                <span class="nav-sub">Previous</span>
                <span class="nav-title">← Home Showcase</span>
              </button>
            {/if}

            {#if nextSection}
              <button class="page-nav next" onclick={() => selectSection(nextSection.id)}>
                <span class="nav-sub">Next</span>
                <span class="nav-title">{nextSection.title} →</span>
              </button>
            {/if}
          </div>
        </div>
      </main>
    </div>
  {/if}

  <SearchModal
    bind:open={searchOpen}
    onselect={selectSection}
    onclose={() => (searchOpen = false)}
  />
</div>

<style>
  .docs-app {
    min-height: 100vh;
    display: flex;
    flex-direction: column;
    background: var(--ground);
    color: var(--text);
  }
  .layout-container {
    display: flex;
    flex: 1;
    max-width: 1440px;
    width: 100%;
    margin: 0 auto;
  }
  .content-pane {
    flex: 1;
    min-width: 0;
    padding: 2.5rem 3rem 4rem 3rem;
  }
  @media (max-width: 768px) {
    .content-pane {
      padding: 1.5rem 1.25rem 3rem 1.25rem;
    }
  }
  .content-inner {
    max-width: 820px;
    margin: 0 auto;
  }
  .doc-pagination {
    display: flex;
    justify-content: space-between;
    gap: 1rem;
    margin-top: 3.5rem;
    padding-top: 1.5rem;
    border-top: 1px solid var(--line);
  }
  .page-nav {
    display: flex;
    flex-direction: column;
    background: var(--surface-sunken);
    border: 1px solid var(--line);
    border-radius: var(--radius-md);
    padding: 0.85rem 1.25rem;
    cursor: pointer;
    text-align: left;
    transition: all 0.15s ease;
  }
  .page-nav:hover {
    border-color: var(--accent);
    background: var(--hover);
  }
  .page-nav.next {
    text-align: right;
    margin-left: auto;
  }
  .nav-sub {
    font-size: 0.72rem;
    font-weight: 600;
    text-transform: uppercase;
    color: var(--text-muted);
  }
  .nav-title {
    font-size: 0.95rem;
    font-weight: 600;
    color: var(--text);
    margin-top: 0.15rem;
  }
</style>

