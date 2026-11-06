<script lang="ts">
  import { onMount } from "svelte";

  interface Props {
    onnavigate: (sectionId: string) => void;
    ontoggleTheme: () => void;
    theme: "light" | "dark";
    onopenSearch: () => void;
  }

  let { onnavigate, ontoggleTheme, theme, onopenSearch }: Props = $props();

  // Carousel slides matching product features & reference video
  const slides = [
    {
      title: "Focus on What Matters Today",
      desc: "Track your most important tasks as they are intelligently prioritized and scheduled.",
      tag: "TASK WORKFLOWS"
    },
    {
      title: "One Continuous Progress Rail",
      desc: "A unified deck spanning planning, working, recording, and knowing across all projects.",
      tag: "SHARED IDENTITY"
    },
    {
      title: "Precision Time Records",
      desc: "Accurate duration tracking and audit trails linked directly to real work items.",
      tag: "TIME TRACKING"
    },
    {
      title: "Single Static Monolith",
      desc: "High-performance Go net/http backend embedding reactive Svelte 5 with zero SaaS telemetry.",
      tag: "SELF-HOSTED"
    },
    {
      title: "Full Data Ownership",
      desc: "PostgreSQL 18 data storage, argon2id encryption, strict same-origin CSP, and AGPL v3 license.",
      tag: "PRIVACY FIRST"
    }
  ];

  let currentSlide = $state(0);

  function nextSlide() {
    currentSlide = (currentSlide + 1) % slides.length;
  }

  function prevSlide() {
    currentSlide = (currentSlide - 1 + slides.length) % slides.length;
  }

  // Ultra-smooth 60-120fps physics tilt via requestAnimationFrame
  let targetTiltX = 0;
  let targetTiltY = 0;
  let currentTiltX = $state(0);
  let currentTiltY = $state(0);
  let animFrameId: number | null = null;
  let isHovered = $state(false);

  function updateTilt() {
    // Critically damped spring interpolation
    currentTiltX += (targetTiltX - currentTiltX) * 0.08;
    currentTiltY += (targetTiltY - currentTiltY) * 0.08;

    if (
      Math.abs(targetTiltX - currentTiltX) > 0.01 ||
      Math.abs(targetTiltY - currentTiltY) > 0.01
    ) {
      animFrameId = requestAnimationFrame(updateTilt);
    } else {
      currentTiltX = targetTiltX;
      currentTiltY = targetTiltY;
      animFrameId = null;
    }
  }

  function handleMouseMove(e: MouseEvent) {
    const rect = (e.currentTarget as HTMLElement).getBoundingClientRect();
    const x = (e.clientX - rect.left) / rect.width - 0.5;
    const y = (e.clientY - rect.top) / rect.height - 0.5;
    targetTiltX = x * 16; // degrees
    targetTiltY = -y * 16;

    if (!animFrameId) {
      animFrameId = requestAnimationFrame(updateTilt);
    }
  }

  function handleMouseEnter() {
    isHovered = true;
  }

  function handleMouseLeave() {
    isHovered = false;
    targetTiltX = 0;
    targetTiltY = 0;
    if (!animFrameId) {
      animFrameId = requestAnimationFrame(updateTilt);
    }
  }

  onMount(() => {
    return () => {
      if (animFrameId) cancelAnimationFrame(animFrameId);
    };
  });
</script>

<div class="showcase-outer" data-theme={theme}>
  <!-- Ambient backdrop lighting matching project theme -->
  <div class="ambient-glow" aria-hidden="true"></div>

  <!-- Main Hero Stage Card matching the Dribbble Canvas -->
  <section
    class="hero-stage"
    onmousemove={handleMouseMove}
    onmouseenter={handleMouseEnter}
    onmouseleave={handleMouseLeave}
    aria-label="Trestle Product Showcase"
  >
    <!-- Top Navbar inside the stage -->
    <header class="stage-nav">
      <div
        class="nav-brand"
        role="button"
        tabindex="0"
        onclick={() => onnavigate("home")}
        onkeydown={(e) => e.key === "Enter" && onnavigate("home")}
      >
        <div class="brand-bolt" aria-hidden="true">
          <svg
            viewBox="0 0 24 24"
            width="17"
            height="17"
            fill="none"
            stroke="currentColor"
            stroke-width="2.5"
            stroke-linecap="round"
            stroke-linejoin="round"
          >
            <path d="M2.5 6h19" />
            <path d="M6.5 6 4 20M17.5 6 20 20" />
            <path d="M6.5 6 12 13l5.5-7" />
            <path d="M5.2 14.5h13.6" />
          </svg>
        </div>
        <span class="brand-title">TRESTLE</span>
      </div>

      <nav class="nav-links" aria-label="Main navigation">
        <button class="nav-link" onclick={() => onnavigate("overview")}>Product</button>
        <button class="nav-link" onclick={() => onnavigate("projects-tasks")}>Workflows</button>
        <button class="nav-link" onclick={() => onnavigate("overview")}>Architecture</button>
        <button class="nav-link" onclick={() => onnavigate("self-hosting")}>Self-Hosting</button>
        <button class="nav-link" onclick={() => onnavigate("overview")}>Docs</button>
      </nav>

      <div class="nav-actions">
        <button
          class="action-btn icon-search"
          onclick={onopenSearch}
          aria-label="Search documentation"
          title="Quick Search (⌘K)"
        >
          <svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2">
            <circle cx="11" cy="11" r="8" />
            <line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
        </button>

        <button
          class="action-btn icon-theme"
          onclick={ontoggleTheme}
          aria-label="Toggle light/dark theme"
          title="Toggle Theme"
        >
          {#if theme === "dark"}
            <svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2">
              <circle cx="12" cy="12" r="5" />
              <line x1="12" y1="1" x2="12" y2="3" />
              <line x1="12" y1="21" x2="12" y2="23" />
              <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
              <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
              <line x1="1" y1="12" x2="3" y2="12" />
              <line x1="21" y1="12" x2="23" y2="12" />
            </svg>
          {:else}
            <svg viewBox="0 0 24 24" width="15" height="15" fill="none" stroke="currentColor" stroke-width="2">
              <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
            </svg>
          {/if}
        </button>

        <button class="get-started-btn" onclick={() => onnavigate("quickstart")}>
          Get Started
        </button>
      </div>
    </header>

    <!-- Main Stage Workspace -->
    <div class="stage-workspace">
      <!-- Massive Editorial Headlines -->
      <h1 class="headline-left">FOCUS</h1>
      <h1 class="headline-right">YOUR TIME</h1>

      <!-- Centerpiece 3D Sculptural Emblem -->
      <div
        class="centerpiece-tilt"
        style:transform="perspective(1200px) rotateY({currentTiltX}deg) rotateX({currentTiltY}deg) {isHovered ? 'scale3d(1.025, 1.025, 1.025)' : 'scale3d(1, 1, 1)'}"
      >
        <!-- Pure GPU-composited Ground Shadow -->
        <div class="ground-shadow" aria-hidden="true"></div>

        <!-- Continuous Smooth Floating Motion Layer -->
        <div class="centerpiece-float">
          <div class="emblem-3d">
            <!-- 3D Velvet Extrusion & Depth (honey amber / warm velvet) -->
            <svg class="emblem-depth" viewBox="0 0 380 440" fill="none" aria-hidden="true">
              <defs>
                <linearGradient id="amberVelvet" x1="0%" y1="0%" x2="100%" y2="100%">
                  <stop offset="0%" stop-color="oklch(0.68 0.19 55)" />
                  <stop offset="50%" stop-color="oklch(0.58 0.18 45)" />
                  <stop offset="100%" stop-color="oklch(0.48 0.16 35)" />
                </linearGradient>
              </defs>
              <path
                d="M 190 28
                   C 230 28, 255 58, 240 98
                   L 205 180
                   L 285 180
                   C 325 180, 340 210, 315 245
                   L 165 400
                   C 135 435, 95 415, 115 370
                   L 155 275
                   L 85 275
                   C 45 275, 35 240, 65 195
                   L 165 50
                   C 175 35, 182 28, 190 28 Z"
                fill="url(#amberVelvet)"
              />
            </svg>

            <!-- Front Surface: Textured Ivory/Cream Plush Face -->
            <div class="emblem-front">
              <svg class="emblem-plush" viewBox="0 0 380 440" fill="none" aria-hidden="true">
                <defs>
                  <linearGradient id="ivoryPlush" x1="20%" y1="0%" x2="80%" y2="100%">
                    <stop offset="0%" stop-color="oklch(0.97 0.02 85)" />
                    <stop offset="40%" stop-color="oklch(0.94 0.025 80)" />
                    <stop offset="70%" stop-color="oklch(0.90 0.03 75)" />
                    <stop offset="100%" stop-color="oklch(0.86 0.035 70)" />
                  </linearGradient>
                  <radialGradient id="highlightSpot" cx="45%" cy="30%" r="55%">
                    <stop offset="0%" stop-color="oklch(1 0 0 / 0.8)" />
                    <stop offset="70%" stop-color="oklch(1 0 0 / 0)" />
                  </radialGradient>
                </defs>

                <!-- Front Facet -->
                <path
                  d="M 188 38
                     C 224 38, 246 64, 233 100
                     L 200 178
                     L 278 178
                     C 312 178, 324 204, 302 235
                     L 168 378
                     C 142 408, 108 392, 124 354
                     L 158 268
                     L 94 268
                     C 60 268, 52 238, 78 200
                     L 168 56
                     C 176 43, 182 38, 188 38 Z"
                  fill="url(#ivoryPlush)"
                />
                <!-- Soft Luminous Sheen -->
                <path
                  d="M 188 38
                     C 224 38, 246 64, 233 100
                     L 200 178
                     L 278 178
                     C 312 178, 324 204, 302 235
                     L 168 378
                     C 142 408, 108 392, 124 354
                     L 158 268
                     L 94 268
                     C 60 268, 52 238, 78 200
                     L 168 56
                     C 176 43, 182 38, 188 38 Z"
                  fill="url(#highlightSpot)"
                  opacity="0.9"
                />
              </svg>
            </div>
          </div>
        </div>
      </div>

      <!-- Floating Glass Widget 1: Top-Left "How It Works" -->
      <div
        class="glass-card widget-video"
        role="button"
        tabindex="0"
        onclick={() => onnavigate("quickstart")}
        onkeydown={(e) => e.key === "Enter" && onnavigate("quickstart")}
        title="Click to view Quickstart and Demo"
      >
        <div class="phone-thumbnail" aria-hidden="true">
          <div class="phone-screen">
            <div class="play-indicator">
              <svg viewBox="0 0 24 24" width="13" height="13" fill="currentColor">
                <polygon points="6 3 20 12 6 21 6 3" />
              </svg>
            </div>
          </div>
        </div>
        <div class="video-info">
          <span class="video-title">How It Works</span>
          <span class="video-sub">Watch the video to see how it works.</span>
        </div>
      </div>

      <!-- Floating Glass Widget 2: Mid-Left Task List Capsule -->
      <div class="glass-card widget-tasks-capsule" aria-label="Task preview list">
        <div class="capsule-row">
          <span class="task-title">App Design</span>
          <span class="priority-pill pill-high">High</span>
          <span class="status-name">In Progress</span>
          <span class="task-duration">4h</span>
        </div>
        <div class="capsule-divider"></div>
        <div class="capsule-row">
          <span class="task-title">Team Meeting</span>
          <span class="priority-pill pill-med">Medium</span>
          <span class="status-name">Scheduled</span>
          <span class="task-duration">30m</span>
        </div>
      </div>

      <!-- Floating Glass Widget 3: Mid-Right "Today's Tasks" Board Card -->
      <div class="glass-card widget-board">
        <div class="board-header">
          <div>
            <h3 class="board-title">Today's Tasks</h3>
            <span class="board-sub">To-do list for today</span>
          </div>
          <button
            class="board-arrow-btn"
            onclick={() => onnavigate("projects-tasks")}
            aria-label="Open Tasks view"
            title="View all tasks"
          >
            <svg
              viewBox="0 0 24 24"
              width="14"
              height="14"
              fill="none"
              stroke="currentColor"
              stroke-width="2.2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <line x1="7" y1="17" x2="17" y2="7" />
              <polyline points="7 7 17 7 17 17" />
            </svg>
          </button>
        </div>

        <div class="board-cards-grid">
          <!-- Sub-card A -->
          <div class="task-subcard">
            <div class="subcard-time">24.03 &nbsp;10:00 AM</div>
            <div class="subcard-title">Mobile App Design</div>
            <div class="subcard-estimate">4 Hours Per Task</div>
            <div class="subcard-footer">
              <span class="subcard-status">In Progress</span>
              <span class="status-indicator-dot dot-orange" aria-hidden="true"></span>
            </div>
          </div>

          <!-- Sub-card B -->
          <div class="task-subcard">
            <div class="subcard-time">24.03 &nbsp;02:00 PM</div>
            <div class="subcard-title">Dashboard Design</div>
            <div class="subcard-estimate">3 Hours Per Task</div>
            <div class="subcard-footer">
              <span class="subcard-status">Scheduled</span>
              <span class="status-indicator-dot dot-yellow" aria-hidden="true"></span>
            </div>
          </div>
        </div>
      </div>

      <!-- Bottom Narrative Copy & Interactive Carousel -->
      <div class="bottom-narrative">
        <div class="slide-badge">{slides[currentSlide].tag}</div>
        <h2 class="slide-heading">{slides[currentSlide].title}</h2>
        <p class="slide-description">{slides[currentSlide].desc}</p>

        <div class="carousel-nav" role="toolbar" aria-label="Feature Carousel Navigation">
          <button class="carousel-arrow" onclick={prevSlide} aria-label="Previous slide">
            <svg
              viewBox="0 0 24 24"
              width="15"
              height="15"
              fill="none"
              stroke="currentColor"
              stroke-width="2.2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <line x1="19" y1="12" x2="5" y2="12" />
              <polyline points="12 19 5 12 12 5" />
            </svg>
          </button>

          <div class="carousel-dots" role="tablist">
            {#each slides as slide, idx}
              <button
                class="carousel-dot"
                class:active={idx === currentSlide}
                onclick={() => (currentSlide = idx)}
                role="tab"
                aria-selected={idx === currentSlide}
                aria-label="Slide {idx + 1}"
              ></button>
            {/each}
          </div>

          <button class="carousel-arrow" onclick={nextSlide} aria-label="Next slide">
            <svg
              viewBox="0 0 24 24"
              width="15"
              height="15"
              fill="none"
              stroke="currentColor"
              stroke-width="2.2"
              stroke-linecap="round"
              stroke-linejoin="round"
            >
              <line x1="5" y1="12" x2="19" y2="12" />
              <polyline points="12 5 19 12 12 19" />
            </svg>
          </button>
        </div>
      </div>

      <!-- Bottom Right Tagline -->
      <div class="bottom-tagline" aria-hidden="true">
        SMART WIDGETS THAT ADAPT TO YOUR<br />WORKFLOW ON WEB AND MOBILE
      </div>
    </div>
  </section>

  <!-- Below-the-Stage Quick Documentation Directory -->
  <section class="quick-directory">
    <div class="directory-inner">
      <div class="directory-header">
        <span class="directory-pill">DOCUMENTATION & ARCHITECTURE</span>
        <h3 class="directory-title">Explore the Trestle Platform</h3>
        <p class="directory-desc">
          Single binary self-hosted workspace with zero SaaS telemetry. Everything you need to deploy, customize, and operate Trestle for yourself or your team.
        </p>
      </div>

      <div class="directory-grid">
        <div
          class="directory-card"
          role="button"
          tabindex="0"
          onclick={() => onnavigate("quickstart")}
          onkeydown={(e) => e.key === "Enter" && onnavigate("quickstart")}
        >
          <div class="card-icon-tile">🚀</div>
          <h4 class="card-title">1-Command Quickstart</h4>
          <p class="card-desc">Run <code>./run.sh</code> or <code>./run.ps1</code>. Auto-provisions Postgres, checks ports, builds Vite, and seeds admin.</p>
          <span class="card-action">View setup guide →</span>
        </div>

        <div
          class="directory-card"
          role="button"
          tabindex="0"
          onclick={() => onnavigate("self-hosting")}
          onkeydown={(e) => e.key === "Enter" && onnavigate("self-hosting")}
        >
          <div class="card-icon-tile">🐳</div>
          <h4 class="card-title">Docker Hub Deployment</h4>
          <p class="card-desc">Official <code>marufsarker/trestle:latest</code> image (~32MB). Ready for Docker Compose, systemd, or bare metal.</p>
          <span class="card-action">Deploy with Docker →</span>
        </div>

        <div
          class="directory-card"
          role="button"
          tabindex="0"
          onclick={() => onnavigate("projects-tasks")}
          onkeydown={(e) => e.key === "Enter" && onnavigate("projects-tasks")}
        >
          <div class="card-icon-tile">📋</div>
          <h4 class="card-title">Tasks & Workflows</h4>
          <p class="card-desc">Dual List and Kanban views, atomic human keys (<code>&lt;KEY&gt;-&lt;NUM&gt;</code>), multi-assignees, subtasks, and progress rails.</p>
          <span class="card-action">Explore workflows →</span>
        </div>

        <div
          class="directory-card"
          role="button"
          tabindex="0"
          onclick={() => onnavigate("custom-statuses")}
          onkeydown={(e) => e.key === "Enter" && onnavigate("custom-statuses")}
        >
          <div class="card-icon-tile">🎨</div>
          <h4 class="card-title">Custom Statuses</h4>
          <p class="card-desc">5 canonical categories (Backlog, Unstarted, Started, Completed, Cancelled) with customizable labels, colors, and delete remaps.</p>
          <span class="card-action">Configure statuses →</span>
        </div>

        <div
          class="directory-card"
          role="button"
          tabindex="0"
          onclick={() => onnavigate("api-reference")}
          onkeydown={(e) => e.key === "Enter" && onnavigate("api-reference")}
        >
          <div class="card-icon-tile">🔌</div>
          <h4 class="card-title">REST API & Monolith</h4>
          <p class="card-desc">Clean JSON endpoints for workspaces, projects, tasks, comments, and time entries with cookie-based auth.</p>
          <span class="card-action">Read API specs →</span>
        </div>

        <div
          class="directory-card"
          role="button"
          tabindex="0"
          onclick={() => onnavigate("design-system")}
          onkeydown={(e) => e.key === "Enter" && onnavigate("design-system")}
        >
          <div class="card-icon-tile">✨</div>
          <h4 class="card-title">WCAG 2.2 AA Design</h4>
          <p class="card-desc">Mona Sans typography, OKLCH semantic tokens, critically damped spring motion, and strict no-native-picker rules.</p>
          <span class="card-action">Review design system →</span>
        </div>
      </div>
    </div>
  </section>
</div>

<style>
  /* Outer Showcase Wrapper */
  .showcase-outer {
    position: relative;
    width: 100%;
    min-height: 100vh;
    padding: 1.5rem 1.5rem 4rem;
    display: flex;
    flex-direction: column;
    align-items: center;
    background: var(--ground);
    color: var(--text);
    overflow-x: hidden;
    box-sizing: border-box;
  }

  /* Ambient soft radial glow matching theme */
  .ambient-glow {
    position: absolute;
    top: 5%;
    left: 50%;
    transform: translate3d(-50%, 0, 0);
    width: 80vw;
    max-width: 1200px;
    height: 560px;
    border-radius: 50%;
    background: radial-gradient(
      ellipse at center,
      light-dark(oklch(0.92 0.05 75 / 0.55), oklch(0.24 0.08 284 / 0.4)),
      transparent 70%
    );
    pointer-events: none;
    z-index: 0;
  }

  /* Main Hero Stage Card matching the Dribbble Canvas */
  .hero-stage {
    position: relative;
    z-index: 1;
    width: 100%;
    max-width: 1360px;
    height: clamp(680px, 86vh, 840px);
    margin: 0 auto;
    border-radius: 28px;
    padding: 2rem 2.5rem 2rem;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    overflow: hidden;
    background: light-dark(
      oklch(0.905 0.032 75),
      oklch(0.165 calc(var(--tint) * 1.5) var(--brand-hue))
    );
    border: 1px solid light-dark(
      oklch(0.85 0.03 75),
      oklch(0.26 calc(var(--tint) * 1.5) var(--brand-hue))
    );
    box-shadow:
      0 20px 60px -15px light-dark(oklch(0.4 0.04 75 / 0.12), oklch(0 0 0 / 0.5)),
      inset 0 1px 0 light-dark(oklch(1 0 0 / 0.6), oklch(1 0 0 / 0.06));
    user-select: none;
    box-sizing: border-box;
  }

  /* Stage Nav Bar */
  .stage-nav {
    display: flex;
    align-items: center;
    justify-content: space-between;
    position: relative;
    z-index: 10;
    flex-shrink: 0;
  }

  .nav-brand {
    display: flex;
    align-items: center;
    gap: 0.65rem;
    cursor: pointer;
  }

  .brand-bolt {
    width: 2.1rem;
    height: 2.1rem;
    border-radius: 0.6rem;
    display: grid;
    place-items: center;
    background: light-dark(oklch(0.18 0.02 75), var(--accent));
    color: light-dark(oklch(0.98 0 0), oklch(0.15 0.02 var(--brand-hue)));
    box-shadow: 0 4px 12px -2px light-dark(oklch(0 0 0 / 0.2), var(--accent-glow));
    transition: transform 0.2s ease;
  }
  .nav-brand:hover .brand-bolt {
    transform: scale(1.05);
  }

  .brand-title {
    font-size: 1.2rem;
    font-weight: 850;
    letter-spacing: -0.03em;
    color: var(--text);
  }

  .nav-links {
    display: flex;
    align-items: center;
    gap: 2rem;
  }

  .nav-link {
    background: transparent;
    border: none;
    font-family: inherit;
    font-size: 0.92rem;
    font-weight: 520;
    color: light-dark(oklch(0.3 0.02 75), var(--text-secondary));
    cursor: pointer;
    transition: color 0.15s ease;
  }
  .nav-link:hover {
    color: var(--text);
  }

  .nav-actions {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .action-btn {
    width: 2.2rem;
    height: 2.2rem;
    border-radius: 50%;
    border: 1px solid light-dark(oklch(0.8 0.02 75), var(--line));
    background: light-dark(oklch(0.94 0.02 75 / 0.8), var(--surface-raised));
    color: var(--text);
    display: grid;
    place-items: center;
    cursor: pointer;
    transition: all 0.15s ease;
  }
  .action-btn:hover {
    background: var(--hover);
    transform: translate3d(0, -1px, 0);
  }

  .get-started-btn {
    padding: 0.52rem 1.3rem;
    border-radius: 9999px;
    font-family: inherit;
    font-size: 0.9rem;
    font-weight: 600;
    border: none;
    background: light-dark(oklch(0.15 0.02 75), var(--accent));
    color: light-dark(oklch(0.98 0 0), oklch(0.15 0.02 var(--brand-hue)));
    cursor: pointer;
    box-shadow: 0 4px 14px -3px light-dark(oklch(0 0 0 / 0.25), var(--accent-glow));
    transition: transform 0.15s ease, box-shadow 0.15s ease;
  }
  .get-started-btn:hover {
    transform: translate3d(0, -1px, 0);
    box-shadow: 0 6px 18px -3px light-dark(oklch(0 0 0 / 0.35), var(--accent-glow));
  }

  /* Stage Workspace (Fills the center area) */
  .stage-workspace {
    position: relative;
    flex: 1;
    width: 100%;
    margin-top: 1rem;
    overflow: visible;
  }

  /* Massive Editorial Headline */
  .headline-left,
  .headline-right {
    position: absolute;
    top: 0.5rem;
    font-size: clamp(3.2rem, 7.5vw, 6.8rem);
    font-weight: 850;
    line-height: 0.92;
    letter-spacing: -0.045em;
    color: light-dark(oklch(0.18 0.02 75), oklch(0.95 0.005 var(--brand-hue)));
    pointer-events: none;
    z-index: 2;
  }

  .headline-left {
    left: 0;
  }

  .headline-right {
    right: 0;
    text-align: right;
    max-width: 5.5ch;
  }

  /* Centerpiece 3D Tilt Wrapper */
  .centerpiece-tilt {
    position: absolute;
    top: 48%;
    left: 50%;
    margin-left: -175px;
    margin-top: -210px;
    width: 350px;
    height: 420px;
    z-index: 3;
    pointer-events: none;
    transform-style: preserve-3d;
    will-change: transform;
    transition: transform 0.15s cubic-bezier(0.2, 0, 0, 1);
  }

  /* Continuous GPU Floating Animation */
  .centerpiece-float {
    width: 100%;
    height: 100%;
    transform-style: preserve-3d;
    will-change: transform;
    animation: float-center 5s cubic-bezier(0.45, 0, 0.55, 1) infinite alternate;
  }

  @keyframes float-center {
    0% {
      transform: translate3d(0, 4px, 0) rotate(-1.5deg);
    }
    100% {
      transform: translate3d(0, -18px, 0) rotate(1.5deg);
    }
  }

  /* Ground Shadow Animated via Pure Transform & Opacity */
  .ground-shadow {
    position: absolute;
    bottom: -32px;
    left: 12%;
    width: 76%;
    height: 40px;
    border-radius: 50%;
    background: radial-gradient(ellipse at center, rgba(0, 0, 0, 0.32), transparent 70%);
    filter: blur(10px);
    pointer-events: none;
    will-change: transform, opacity;
    animation: float-shadow 5s cubic-bezier(0.45, 0, 0.55, 1) infinite alternate;
  }

  @keyframes float-shadow {
    0% {
      transform: translate3d(0, 0, 0) scale(1.05);
      opacity: 0.55;
    }
    100% {
      transform: translate3d(0, 0, 0) scale(0.85);
      opacity: 0.35;
    }
  }

  .emblem-3d {
    position: relative;
    width: 100%;
    height: 100%;
  }

  .emblem-depth {
    position: absolute;
    top: 6px;
    left: 4px;
    width: 100%;
    height: 100%;
    overflow: visible;
  }

  .emblem-front {
    position: absolute;
    top: 0;
    left: 0;
    width: 100%;
    height: 100%;
  }

  .emblem-plush {
    width: 100%;
    height: 100%;
    overflow: visible;
  }

  /* Shared Floating Glass Cards */
  .glass-card {
    position: absolute;
    z-index: 5;
    background: light-dark(
      oklch(0.97 0.015 75 / 0.75),
      oklch(0.2 0.02 var(--brand-hue) / 0.72)
    );
    backdrop-filter: blur(24px);
    -webkit-backdrop-filter: blur(24px);
    border: 1px solid light-dark(
      oklch(1 0 0 / 0.8),
      oklch(1 0 0 / 0.12)
    );
    border-radius: 20px;
    box-shadow:
      0 16px 36px -10px light-dark(oklch(0.3 0.03 75 / 0.14), oklch(0 0 0 / 0.45)),
      inset 0 1px 0 light-dark(oklch(1 0 0 / 0.9), oklch(1 0 0 / 0.1));
    will-change: transform;
    transition: transform 0.25s cubic-bezier(0.2, 0, 0, 1), box-shadow 0.25s ease;
  }
  .glass-card:hover {
    transform: translate3d(0, -3px, 0);
    animation-play-state: paused;
    box-shadow:
      0 20px 45px -8px light-dark(oklch(0.3 0.03 75 / 0.2), oklch(0 0 0 / 0.55)),
      inset 0 1px 0 light-dark(1 0 0 / 1, 1 0 0 / 0.15);
  }

  /* Widget 1: Top-Left "How It Works" */
  .widget-video {
    top: 24%;
    left: 0;
    display: flex;
    align-items: center;
    gap: 0.9rem;
    padding: 0.85rem 1.2rem 0.85rem 0.85rem;
    cursor: pointer;
    animation: card-float-1 6s ease-in-out infinite;
  }

  @keyframes card-float-1 {
    0%, 100% {
      transform: translate3d(0, 0, 0);
    }
    50% {
      transform: translate3d(0, -6px, 0);
    }
  }

  .phone-thumbnail {
    width: 42px;
    height: 60px;
    border-radius: 8px;
    background: light-dark(oklch(0.88 0.04 75), var(--surface-sunken));
    border: 1.5px solid light-dark(oklch(0.8 0.03 75), var(--line));
    padding: 3px;
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
  }

  .phone-screen {
    width: 100%;
    height: 100%;
    border-radius: 5px;
    background: linear-gradient(135deg, oklch(0.7 0.18 50), oklch(0.6 0.16 40));
    display: grid;
    place-items: center;
  }

  .play-indicator {
    width: 18px;
    height: 18px;
    border-radius: 50%;
    background: oklch(1 0 0 / 0.9);
    color: oklch(0.55 0.18 45);
    display: grid;
    place-items: center;
    padding-left: 2px;
  }

  .video-info {
    display: flex;
    flex-direction: column;
    gap: 0.15rem;
    max-width: 140px;
  }

  .video-title {
    font-size: 0.92rem;
    font-weight: 700;
    color: var(--text);
  }

  .video-sub {
    font-size: 0.76rem;
    line-height: 1.35;
    color: var(--text-secondary);
  }

  /* Widget 2: Mid-Left Task List Capsule */
  .widget-tasks-capsule {
    top: 48%;
    left: 8%;
    padding: 0.85rem 1.25rem;
    min-width: 250px;
    animation: card-float-2 5.5s ease-in-out 0.8s infinite;
  }

  @keyframes card-float-2 {
    0%, 100% {
      transform: translate3d(0, 0, 0);
    }
    50% {
      transform: translate3d(0, -8px, 0);
    }
  }

  .capsule-row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 0.85rem;
    font-size: 0.82rem;
    padding: 0.2rem 0;
  }

  .task-title {
    font-weight: 600;
    color: var(--text);
    min-width: 80px;
  }

  .priority-pill {
    padding: 0.2rem 0.55rem;
    border-radius: 9999px;
    font-size: 0.7rem;
    font-weight: 700;
    line-height: 1;
  }

  .pill-high {
    background: oklch(0.65 0.2 45);
    color: white;
  }

  .pill-med {
    background: oklch(0.72 0.18 70);
    color: white;
  }

  .status-name {
    font-size: 0.78rem;
    color: var(--text-secondary);
  }

  .task-duration {
    font-size: 0.8rem;
    font-weight: 600;
    color: var(--text-muted);
  }

  .capsule-divider {
    height: 1px;
    background: light-dark(oklch(0.88 0.02 75), var(--line));
    margin: 0.4rem 0;
  }

  /* Widget 3: Mid-Right "Today's Tasks" Board Card */
  .widget-board {
    top: 40%;
    right: 0;
    padding: 1.25rem;
    width: clamp(310px, 26vw, 360px);
    animation: card-float-3 6.5s ease-in-out 0.4s infinite;
  }

  @keyframes card-float-3 {
    0%, 100% {
      transform: translate3d(0, 0, 0);
    }
    50% {
      transform: translate3d(0, -7px, 0);
    }
  }

  .board-header {
    display: flex;
    align-items: flex-start;
    justify-content: space-between;
    margin-bottom: 1rem;
  }

  .board-title {
    font-size: 1.1rem;
    font-weight: 750;
    color: var(--text);
    line-height: 1.2;
  }

  .board-sub {
    font-size: 0.78rem;
    color: var(--text-secondary);
  }

  .board-arrow-btn {
    width: 1.9rem;
    height: 1.9rem;
    border-radius: 8px;
    background: light-dark(oklch(1 0 0 / 0.85), var(--surface-raised));
    border: 1px solid light-dark(oklch(0.85 0.02 75), var(--border));
    color: var(--text);
    display: grid;
    place-items: center;
    cursor: pointer;
    transition: transform 0.15s ease;
  }
  .board-arrow-btn:hover {
    transform: translate3d(2px, -2px, 0);
  }

  .board-cards-grid {
    display: grid;
    grid-template-columns: 1fr 1fr;
    gap: 0.65rem;
  }

  .task-subcard {
    background: light-dark(oklch(1 0 0 / 0.95), var(--surface-raised));
    border-radius: 12px;
    padding: 0.8rem 0.85rem;
    box-shadow: 0 4px 14px -3px light-dark(oklch(0 0 0 / 0.06), oklch(0 0 0 / 0.3));
    border: 1px solid light-dark(oklch(0.92 0.01 75), var(--line));
    display: flex;
    flex-direction: column;
    gap: 0.3rem;
  }

  .subcard-time {
    font-size: 0.68rem;
    font-weight: 600;
    color: var(--text-muted);
  }

  .subcard-title {
    font-size: 0.82rem;
    font-weight: 700;
    color: var(--text);
    line-height: 1.25;
  }

  .subcard-estimate {
    font-size: 0.72rem;
    color: var(--text-secondary);
    margin-bottom: 0.25rem;
  }

  .subcard-footer {
    display: flex;
    align-items: center;
    justify-content: space-between;
    margin-top: auto;
    padding-top: 0.35rem;
    border-top: 1px solid light-dark(oklch(0.94 0.01 75), var(--line));
  }

  .subcard-status {
    font-size: 0.72rem;
    color: var(--text-secondary);
  }

  .status-indicator-dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
  }
  .dot-orange {
    background: oklch(0.65 0.2 45);
  }
  .dot-yellow {
    background: oklch(0.75 0.18 75);
  }

  /* Bottom Narrative Copy & Interactive Carousel */
  .bottom-narrative {
    position: absolute;
    bottom: 0;
    left: 0;
    max-width: 440px;
    z-index: 6;
  }

  .slide-badge {
    font-size: 0.7rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    color: light-dark(oklch(0.5 0.15 45), var(--accent));
    margin-bottom: 0.35rem;
  }

  .slide-heading {
    font-size: 1.4rem;
    font-weight: 750;
    letter-spacing: -0.025em;
    color: var(--text);
    margin-bottom: 0.45rem;
    line-height: 1.2;
  }

  .slide-description {
    font-size: 0.86rem;
    line-height: 1.5;
    color: var(--text-secondary);
    margin-bottom: 1.15rem;
  }

  .carousel-nav {
    display: flex;
    align-items: center;
    gap: 0.75rem;
  }

  .carousel-arrow {
    width: 2rem;
    height: 2rem;
    border-radius: 50%;
    border: 1px solid light-dark(oklch(0.75 0.02 75), var(--line));
    background: light-dark(oklch(0.95 0.02 75), var(--surface-raised));
    color: var(--text);
    display: grid;
    place-items: center;
    cursor: pointer;
    transition: all 0.15s ease;
  }
  .carousel-arrow:hover {
    background: var(--hover);
    transform: scale(1.06);
  }

  .carousel-dots {
    display: flex;
    align-items: center;
    gap: 0.45rem;
  }

  .carousel-dot {
    width: 7px;
    height: 7px;
    border-radius: 50%;
    border: none;
    background: light-dark(oklch(0.75 0.02 75), var(--border));
    cursor: pointer;
    transition: all 0.2s ease;
    padding: 0;
  }
  .carousel-dot.active {
    width: 22px;
    border-radius: 4px;
    background: light-dark(oklch(0.2 0.02 75), var(--accent));
  }

  /* Bottom Right Tagline */
  .bottom-tagline {
    position: absolute;
    bottom: 0.25rem;
    right: 0;
    text-align: right;
    font-size: 0.7rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    line-height: 1.45;
    color: light-dark(oklch(0.4 0.02 75), var(--text-muted));
    z-index: 6;
  }

  /* Below-the-Stage Quick Documentation Directory */
  .quick-directory {
    width: 100%;
    max-width: 1360px;
    margin-top: 3.5rem;
    z-index: 2;
  }

  .directory-inner {
    padding: 0 0.5rem;
  }

  .directory-header {
    text-align: center;
    max-width: 650px;
    margin: 0 auto 2.5rem;
  }

  .directory-pill {
    display: inline-block;
    padding: 0.25rem 0.8rem;
    border-radius: 9999px;
    font-size: 0.72rem;
    font-weight: 700;
    letter-spacing: 0.08em;
    background: var(--accent-soft);
    color: var(--accent);
    margin-bottom: 0.75rem;
  }

  .directory-title {
    font-size: 2rem;
    font-weight: 800;
    letter-spacing: -0.03em;
    color: var(--text);
    margin-bottom: 0.65rem;
  }

  .directory-desc {
    font-size: 0.95rem;
    color: var(--text-secondary);
    line-height: 1.55;
  }

  .directory-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
    gap: 1.25rem;
  }

  .directory-card {
    background: var(--surface);
    border: 1px solid var(--line);
    border-radius: 20px;
    padding: 1.75rem 1.5rem;
    cursor: pointer;
    transition: transform 0.2s ease, border-color 0.2s ease, box-shadow 0.2s ease;
    display: flex;
    flex-direction: column;
  }
  .directory-card:hover {
    transform: translate3d(0, -3px, 0);
    border-color: var(--border);
    box-shadow: 0 12px 30px -8px light-dark(oklch(0 0 0 / 0.08), oklch(0 0 0 / 0.4));
  }

  .card-icon-tile {
    width: 2.75rem;
    height: 2.75rem;
    border-radius: 12px;
    background: var(--surface-raised);
    border: 1px solid var(--line);
    display: grid;
    place-items: center;
    font-size: 1.3rem;
    margin-bottom: 1rem;
  }

  .card-title {
    font-size: 1.15rem;
    font-weight: 700;
    color: var(--text);
    margin-bottom: 0.45rem;
  }

  .card-desc {
    font-size: 0.88rem;
    color: var(--text-secondary);
    line-height: 1.5;
    margin-bottom: 1.25rem;
  }

  .card-desc code {
    font-family: var(--font-mono);
    font-size: 0.85em;
    background: var(--surface-sunken);
    padding: 0.1rem 0.35rem;
    border-radius: 4px;
    border: 1px solid var(--line);
  }

  .card-action {
    margin-top: auto;
    font-size: 0.85rem;
    font-weight: 600;
    color: var(--accent);
    display: inline-flex;
    align-items: center;
    gap: 0.25rem;
  }
  .directory-card:hover .card-action {
    text-decoration: underline;
  }

  /* Responsive Staging for Tablets and Mobile (< 960px) */
  @media (max-width: 960px) {
    .hero-stage {
      height: auto;
      min-height: auto;
      padding: 1.5rem 1.25rem 2rem;
    }
    .stage-nav .nav-links {
      display: none;
    }
    .stage-workspace {
      display: flex;
      flex-direction: column;
      align-items: center;
      gap: 1.5rem;
      margin-top: 1.5rem;
    }
    .headline-left,
    .headline-right {
      position: static;
      font-size: clamp(2.8rem, 11vw, 4.2rem);
      text-align: center;
      max-width: none;
    }
    .headline-right {
      margin-top: -0.5rem;
    }
    .centerpiece-tilt {
      position: relative;
      top: auto;
      left: auto;
      margin: 1rem 0;
      width: 260px;
      height: 310px;
    }
    .widget-video,
    .widget-tasks-capsule,
    .widget-board {
      position: static;
      width: 100%;
      max-width: 440px;
      animation: none;
    }
    .bottom-narrative {
      position: static;
      max-width: 100%;
      text-align: center;
      margin-top: 1rem;
    }
    .carousel-nav {
      justify-content: center;
    }
    .bottom-tagline {
      position: static;
      text-align: center;
      margin-top: 1rem;
    }
  }
</style>
