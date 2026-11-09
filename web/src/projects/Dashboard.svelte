<script lang="ts">
  import { onMount } from "svelte";
  import Button from "../ui/Button.svelte";
  import Feedback from "../ui/Feedback.svelte";
  import Icon, { type IconName } from "../ui/Icon.svelte";
  import Avatar from "../ui/Avatar.svelte";
  import type { Project } from "./ProjectsView.svelte";
  import type { Task } from "./TasksView.svelte";
  import type { User } from "../auth/SignIn.svelte";
  import {
    formatDueDate,
    identityHue,
    initials,
    isOverdue,
    relativeTime,
  } from "../lib/identity";

  let {
    workspace,
    user,
    onopenproject,
    onopenprojects,
  }: {
    workspace: { id: string; name: string; slug: string; role: string };
    user: User;
    onopenproject: (project: Project, taskId?: string) => void;
    onopenprojects: () => void;
  } = $props();

  // Bounded: at most this many projects and tasks per project are read.
  const PROJECT_CAP = 12;
  const TASK_CAP = 100;
  const DAY = 86400000;

  type Category = Task["status"];
  const categories: Category[] = [
    "backlog",
    "unstarted",
    "started",
    "completed",
    "cancelled",
  ];
  const categoryLabel: Record<Category, string> = {
    backlog: "Backlog",
    unstarted: "Unstarted",
    started: "In progress",
    completed: "Completed",
    cancelled: "Cancelled",
  };

  let projects = $state<Project[]>([]);
  let tasksByProject = $state<Record<string, Task[]>>({});
  let truncated = $state(false);
  let loading = $state(true);
  let errorMessage = $state<string | null>(null);

  const today = new Date();
  const startOfToday = new Date(
    today.getFullYear(),
    today.getMonth(),
    today.getDate(),
  );

  function parseDue(value: string): Date | null {
    const [y, m, d] = value.split("-").map(Number);
    if (!y || !m || !d) return null;
    return new Date(y, m - 1, d);
  }

  function daysFromToday(value: string): number | null {
    const due = parseDue(value);
    if (!due) return null;
    return Math.round((due.getTime() - startOfToday.getTime()) / DAY);
  }

  const isOpen = (t: Task) =>
    t.status !== "completed" && t.status !== "cancelled";

  const allTasks = $derived(Object.values(tasksByProject).flat());
  const projectById = $derived(
    Object.fromEntries(projects.map((p) => [p.id, p])),
  );

  const openTasks = $derived(allTasks.filter(isOpen));

  const categoryCounts = $derived.by(() => {
    const counts = Object.fromEntries(categories.map((c) => [c, 0])) as Record<
      Category,
      number
    >;
    for (const t of allTasks) counts[t.status]++;
    return counts;
  });

  // Seven days starting today: how many open tasks fall due on each.
  const week = $derived.by(() =>
    Array.from({ length: 7 }, (_, i) => {
      const date = new Date(startOfToday.getTime() + i * DAY);
      const count = openTasks.filter(
        (t) => t.due_date && daysFromToday(t.due_date) === i,
      ).length;
      return {
        i,
        count,
        label: date.toLocaleDateString(undefined, { weekday: "narrow" }),
        full: date.toLocaleDateString(undefined, {
          weekday: "long",
          month: "short",
          day: "numeric",
        }),
      };
    }),
  );
  const weekMax = $derived(Math.max(1, ...week.map((d) => d.count)));
  const dueThisWeek = $derived(week.reduce((sum, d) => sum + d.count, 0));
  const overdue = $derived(
    openTasks.filter((t) => isOverdue(t.due_date)).length,
  );

  const decided = $derived(allTasks.length - categoryCounts.cancelled);
  const completion = $derived(
    decided > 0 ? Math.round((categoryCounts.completed / decided) * 100) : 0,
  );

  // My work: open tasks assigned to me, the most urgent first.
  const myWork = $derived(
    openTasks
      .filter((t) => t.assignees?.some((a) => a.user_id === user.id))
      .sort((a, b) => {
        const da = a.due_date ? (daysFromToday(a.due_date) ?? 999) : 999;
        const db = b.due_date ? (daysFromToday(b.due_date) ?? 999) : 999;
        if (da !== db) return da - db;
        return priorityRank(b.priority) - priorityRank(a.priority);
      })
      .slice(0, 6),
  );
  const myOpenCount = $derived(
    openTasks.filter((t) => t.assignees?.some((a) => a.user_id === user.id))
      .length,
  );

  function priorityRank(p: Task["priority"]) {
    return { none: 0, low: 1, medium: 2, high: 3, urgent: 4 }[p];
  }

  // Deadlines: the next fourteen days as a timeline.
  const SPAN = 14;
  const timeline = $derived.by(() => {
    const placed = openTasks
      .filter((t) => {
        if (!t.due_date) return false;
        const d = daysFromToday(t.due_date);
        return d !== null && d >= 0 && d < SPAN;
      })
      .sort((a, b) => (a.due_date! < b.due_date! ? -1 : 1))
      .slice(0, 7);
    return placed.map((t, row) => ({
      task: t,
      day: daysFromToday(t.due_date!)!,
      row,
    }));
  });
  const ticks = $derived(
    Array.from({ length: SPAN }, (_, i) => {
      const date = new Date(startOfToday.getTime() + i * DAY);
      return {
        i,
        day: date.getDate(),
        month: date.toLocaleDateString(undefined, { month: "short" }),
      };
    }),
  );

  const projectRows = $derived(
    projects.map((p) => {
      const list = tasksByProject[p.id] ?? [];
      const done = list.filter((t) => t.status === "completed").length;
      const cancelled = list.filter((t) => t.status === "cancelled").length;
      const open = list.length - done - cancelled;
      const counted = list.length - cancelled;
      return {
        project: p,
        open,
        total: list.length,
        pct: counted > 0 ? Math.round((done / counted) * 100) : 0,
      };
    }),
  );

  const greeting = $derived.by(() => {
    const h = today.getHours();
    const part = h < 12 ? "morning" : h < 18 ? "afternoon" : "evening";
    return `Good ${part}, ${user.display_name.split(" ")[0]}`;
  });

  const statusIcon: Record<Category, IconName> = {
    backlog: "layers",
    unstarted: "circle",
    started: "pulse",
    completed: "circle-check",
    cancelled: "close",
  };

  function dueLabel(t: Task): string {
    if (!t.due_date) return "No due date";
    const d = daysFromToday(t.due_date);
    if (d === null) return t.due_date;
    if (d < 0) return `Overdue · ${formatDueDate(t.due_date)}`;
    if (d === 0) return "Due today";
    if (d === 1) return "Due tomorrow";
    return `Due ${formatDueDate(t.due_date)}`;
  }

  async function load() {
    loading = true;
    errorMessage = null;
    try {
      const res = await fetch(`/api/v1/workspaces/${workspace.id}/projects`, {
        headers: { Accept: "application/json" },
        cache: "no-store",
      });
      if (!res.ok) {
        errorMessage = "Could not load this workspace. Please try again.";
        return;
      }
      const data = await res.json();
      projects = (data.projects || []) as Project[];
      const sample = projects.slice(0, PROJECT_CAP);
      const results = await Promise.all(
        sample.map(async (p) => {
          const r = await fetch(
            `/api/v1/workspaces/${workspace.id}/projects/${p.id}/tasks?limit=${TASK_CAP}&offset=0`,
            { headers: { Accept: "application/json" }, cache: "no-store" },
          );
          if (!r.ok) return [p.id, [] as Task[], 0] as const;
          const body = await r.json();
          return [p.id, (body.tasks || []) as Task[], body.total || 0] as const;
        }),
      );
      truncated =
        projects.length > PROJECT_CAP ||
        results.some(([, list, total]) => total > list.length);
      tasksByProject = Object.fromEntries(
        results.map(([id, list]) => [id, list]),
      );
    } catch {
      errorMessage = "Could not load this workspace. Check your connection.";
    } finally {
      loading = false;
    }
  }

  onMount(() => {
    void load();
  });
</script>

<section class="dashboard" aria-labelledby="dashboard-heading">
  <div class="page-head">
    <div>
      <h2 id="dashboard-heading" tabindex="-1">{greeting}</h2>
      <p>
        {#if loading}
          Gathering {workspace.name}…
        {:else if projects.length === 0}
          {workspace.name} has no projects yet.
        {:else}
          {myOpenCount === 0
            ? "Nothing is assigned to you."
            : `${myOpenCount} open ${myOpenCount === 1 ? "task is" : "tasks are"} yours`}
          {#if dueThisWeek > 0}
            · {dueThisWeek} due this week across {workspace.name}{/if}.
        {/if}
      </p>
    </div>
    <div class="page-actions">
      <Button variant="primary" onclick={onopenprojects}>
        {#snippet children()}
          <Icon name="folder" size={16} />
          All projects
        {/snippet}
      </Button>
    </div>
  </div>

  {#if errorMessage}
    <div class="stack">
      <Feedback tone="danger" announcement="alert">{errorMessage}</Feedback>
      <div>
        <Button onclick={load}>
          {#snippet children()}Try again{/snippet}
        </Button>
      </div>
    </div>
  {:else if loading}
    <div class="stats" aria-hidden="true">
      {#each [0, 1, 2, 3] as i (i)}
        <div class="card stat skeleton"></div>
      {/each}
    </div>
    <div class="grid-main" aria-hidden="true">
      <div class="card skeleton tall"></div>
      <div class="card skeleton tall"></div>
    </div>
    <Feedback tone="neutral" announcement="status">Loading dashboard…</Feedback>
  {:else if projects.length === 0}
    <div class="empty">
      <span class="empty-mark"><Icon name="folder" size={22} /></span>
      <h3>Start with a project</h3>
      <p>
        The dashboard fills in from real projects and tasks: open work, due
        dates and progress.
      </p>
      <Button variant="primary" onclick={onopenprojects}>
        {#snippet children()}Go to projects{/snippet}
      </Button>
    </div>
  {:else}
    <div class="stats">
      <article class="card stat">
        <header class="stat-head">
          <span class="tile"><Icon name="folder" size={17} /></span>
          <h3>Active projects</h3>
        </header>
        <p class="stat-value num">{projects.length}</p>
        <p class="stat-note">In {workspace.name}</p>
        <div class="stat-viz keys" aria-hidden="true">
          {#each projects.slice(0, 4) as p (p.id)}
            <span class="key-chip" style:--project-hue={identityHue(p.key)}
              >{p.key}</span
            >
          {/each}
          {#if projects.length > 4}
            <span class="key-chip more">+{projects.length - 4}</span>
          {/if}
        </div>
      </article>

      <article class="card stat">
        <header class="stat-head">
          <span class="tile"><Icon name="list" size={17} /></span>
          <h3>Open tasks</h3>
        </header>
        <p class="stat-value num">{openTasks.length}</p>
        <p class="stat-note">{categoryCounts.started} in progress</p>
        <div
          class="stat-viz mini-rail"
          role="img"
          aria-label="Tasks by status: {categories
            .map((c) => `${categoryLabel[c]} ${categoryCounts[c]}`)
            .join(', ')}"
        >
          {#each categories as c (c)}
            {#if categoryCounts[c] > 0}
              <span
                style:flex-grow={categoryCounts[c]}
                style:background="var(--cat-{c})"
              ></span>
            {/if}
          {/each}
        </div>
      </article>

      <article class="card stat">
        <header class="stat-head">
          <span class="tile"><Icon name="calendar" size={17} /></span>
          <h3>Due this week</h3>
        </header>
        <p class="stat-value num">{dueThisWeek}</p>
        <p class="stat-note" class:alert={overdue > 0}>
          {overdue > 0 ? `${overdue} overdue` : "Nothing overdue"}
        </p>
        <div
          class="stat-viz week"
          role="img"
          aria-label="Open tasks due each day: {week
            .map((d) => `${d.full} ${d.count}`)
            .join(', ')}"
        >
          {#each week as d (d.i)}
            <span class="week-day" class:today={d.i === 0}>
              <span
                class="week-bar"
                class:zero={d.count === 0}
                style:height="{Math.max(8, (d.count / weekMax) * 100)}%"
              ></span>
              <span class="week-label">{d.label}</span>
            </span>
          {/each}
        </div>
      </article>

      <article class="card stat">
        <header class="stat-head">
          <span class="tile"><Icon name="target" size={17} /></span>
          <h3>Completion</h3>
        </header>
        <p class="stat-value num">{completion}%</p>
        <p class="stat-note">
          {categoryCounts.completed} of {decided} tasks done
        </p>
        <svg class="stat-viz ring" viewBox="0 0 44 44" aria-hidden="true">
          <circle class="ring-track" cx="22" cy="22" r="18" />
          <circle
            class="ring-value"
            cx="22"
            cy="22"
            r="18"
            style:stroke-dasharray="{(completion / 100) * 113.1} 113.1"
          />
        </svg>
      </article>
    </div>

    <div class="grid-main">
      <section class="card panel-block rim" aria-labelledby="mywork-heading">
        <header class="block-head">
          <h3 id="mywork-heading">My work</h3>
          <span class="note num">{myOpenCount} open</span>
        </header>
        {#if myWork.length === 0}
          <div class="block-empty">
            <span class="tile"><Icon name="circle-check" size={17} /></span>
            <p>
              Nothing assigned to you is open. Pick something up in a project.
            </p>
          </div>
        {:else}
          <ul class="work-list" role="list">
            {#each myWork as t (t.id)}
              {@const late = isOverdue(t.due_date)}
              <li>
                <button
                  type="button"
                  class="inset work-row"
                  onclick={() => {
                    const p = projectById[t.project_id];
                    if (p) onopenproject(p, t.id);
                  }}
                >
                  <span class="tile tile-sm" aria-hidden="true">
                    <Icon name={statusIcon[t.status]} size={15} />
                  </span>
                  <span class="work-text">
                    <span class="work-title">{t.title}</span>
                    <span class="work-meta">
                      <span class="key">{t.human_key}</span>
                      <span>{t.status_name || categoryLabel[t.status]}</span>
                      <span class:late>{dueLabel(t)}</span>
                    </span>
                  </span>
                  {#if t.priority !== "none"}
                    <span class="priority" data-priority={t.priority}
                      >{t.priority}</span
                    >
                  {/if}
                </button>
              </li>
            {/each}
          </ul>
        {/if}
      </section>

      <section class="card panel-block" aria-labelledby="deadlines-heading">
        <header class="block-head">
          <h3 id="deadlines-heading">Deadlines</h3>
          <span class="note">Next 14 days</span>
        </header>
        {#if timeline.length === 0}
          <div class="block-empty">
            <span class="tile"><Icon name="calendar" size={17} /></span>
            <p>No open task is due in the next two weeks.</p>
          </div>
        {:else}
          <div class="timeline-wrap">
            <div
              class="timeline"
              style:--rows={Math.max(3, timeline.length)}
              role="group"
              aria-label="Open tasks due in the next 14 days"
            >
              <div class="grid-lines" aria-hidden="true">
                {#each ticks as tick (tick.i)}
                  <span class:today={tick.i === 0}></span>
                {/each}
              </div>
              <span class="today-line" aria-hidden="true"></span>
              {#each timeline as item (item.task.id)}
                <button
                  type="button"
                  class="inset event"
                  class:flip={item.day >= 9}
                  style:--day={item.day}
                  style:--row={item.row}
                  onclick={() => {
                    const p = projectById[item.task.project_id];
                    if (p) onopenproject(p, item.task.id);
                  }}
                  aria-label="{item.task.human_key} {item.task
                    .title}, {dueLabel(item.task)}"
                >
                  <span class="event-text">
                    <span class="event-title">{item.task.title}</span>
                    <span class="event-meta"
                      >{item.task.human_key} · {dueLabel(item.task)}</span
                    >
                  </span>
                  {#if item.task.assignees && item.task.assignees.length > 0}
                    <span class="avatar-stack" aria-hidden="true">
                      {#each item.task.assignees.slice(0, 3) as a (a.user_id)}
                        <Avatar
                          id={a.user_id}
                          name={a.display_name}
                          avatarUrl={a.avatar_url}
                          size="sm"
                        />
                      {/each}
                    </span>
                  {/if}
                </button>
              {/each}
            </div>
            <div class="ticks" aria-hidden="true">
              {#each ticks as tick (tick.i)}
                <span class:today={tick.i === 0}
                  >{tick.i === 0 || tick.day === 1
                    ? `${tick.day} ${tick.month}`
                    : tick.day}</span
                >
              {/each}
            </div>
          </div>
        {/if}
      </section>
    </div>

    <section
      class="card panel-block projects-block"
      aria-labelledby="dash-projects-heading"
    >
      <header class="block-head">
        <h3 id="dash-projects-heading">Projects</h3>
        <button type="button" class="btn-link" onclick={onopenprojects}>
          View all
          <Icon name="arrow-right" size={15} />
        </button>
      </header>
      <ul class="project-rows" role="list">
        {#each projectRows as row (row.project.id)}
          <li>
            <button
              type="button"
              class="project-row"
              onclick={() => onopenproject(row.project)}
            >
              {#if row.project.image_url}
                <img
                  src={row.project.image_url}
                  alt="{row.project.name} icon"
                  class="project-img"
                />
              {:else}
                <span
                  class="key-chip"
                  style:--project-hue={identityHue(row.project.key)}
                  >{row.project.key}</span
                >
              {/if}
              <span class="pr-name">
                <span class="pr-title">{row.project.name}</span>
                <span class="pr-sub"
                  >Updated {relativeTime(row.project.updated_at)}</span
                >
              </span>
              <span class="pr-open num">
                <strong>{row.open}</strong> open
              </span>
              <span class="pr-progress">
                <span class="bar" aria-hidden="true">
                  <span style:width="{row.pct}%"></span>
                </span>
                <span class="num pct">{row.pct}%</span>
              </span>
              <Icon name="chevron-right" size={16} class="pr-chevron" />
            </button>
          </li>
        {/each}
      </ul>
    </section>

    {#if truncated}
      <p class="note sample-note">
        Figures cover the first {PROJECT_CAP} projects and up to {TASK_CAP} tasks
        in each.
      </p>
    {/if}
  {/if}
</section>

<style>
  .dashboard {
    display: grid;
  }

  .page-head h2 {
    font-size: clamp(1.625rem, 1.3rem + 1vw, 2.125rem);
    font-weight: 480;
    letter-spacing: -0.03em;
  }

  /* ── Stats ───────────────────────────────────────────────────── */
  .stats {
    display: grid;
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: var(--space-4);
    margin-bottom: var(--space-4);
  }

  .stat {
    position: relative;
    display: grid;
    grid-template-columns: minmax(0, 1fr) auto;
    grid-template-rows: auto 1fr auto;
    gap: 0 var(--space-3);
    min-height: 9.5rem;
    padding: var(--space-4) var(--space-5) var(--space-4);
    overflow: hidden;
    isolation: isolate;
  }

  /* Light pooling in the lower right, as on a lit surface. */
  .stat::before {
    content: "";
    position: absolute;
    inset: 0;
    z-index: -1;
    background: radial-gradient(
      70% 80% at 100% 110%,
      var(--glow-accent),
      transparent 70%
    );
  }

  .stat-head {
    grid-column: 1 / -1;
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  .stat-head h3 {
    font-size: var(--text-ui);
    font-weight: 500;
    color: var(--text-secondary);
  }

  .stat-value {
    align-self: end;
    margin-top: var(--space-4);
    font-size: 2.25rem;
    line-height: 1;
    font-weight: 460;
    letter-spacing: -0.035em;
    color: var(--text);
  }

  .stat-note {
    grid-column: 1;
    margin-top: var(--space-2);
    font-size: var(--text-xs);
    color: var(--success);
    font-weight: 540;
  }

  .stat-note.alert {
    color: var(--danger);
  }

  .stat-viz {
    grid-column: 2;
    grid-row: 2 / 4;
    align-self: end;
  }

  .keys {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: 3px;
  }

  .key-chip {
    --project-hue: var(--brand-hue);
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 3.25rem;
    height: 1.375rem;
    padding: 0 0.4rem;
    border-radius: 0.4rem;
    font-size: 0.6875rem;
    font-weight: 680;
    letter-spacing: 0.05em;
    background: light-dark(
      oklch(0.92 0.05 var(--project-hue)),
      oklch(0.34 0.08 var(--project-hue))
    );
    color: light-dark(
      oklch(0.35 0.1 var(--project-hue)),
      oklch(0.93 0.05 var(--project-hue))
    );
    flex-shrink: 0;
  }

  .key-chip.more {
    background: var(--selected);
    color: var(--text-muted);
  }

  .project-img {
    width: 2rem;
    height: 2rem;
    border-radius: var(--radius-xs);
    object-fit: cover;
    border: var(--border-width) solid var(--line);
    background: var(--surface-sunken);
    flex-shrink: 0;
    justify-self: start;
  }

  .mini-rail {
    display: flex;
    gap: 2px;
    width: 6.5rem;
    height: 0.5rem;
    margin-bottom: 0.4rem;
    border-radius: var(--radius-full);
    overflow: hidden;
    background: var(--surface-sunken);
  }

  .mini-rail span {
    flex-basis: 0;
    min-width: 3px;
  }

  .week {
    display: flex;
    align-items: flex-end;
    gap: 4px;
    height: 4.25rem;
  }

  .week-day {
    display: grid;
    grid-template-rows: 1fr auto;
    justify-items: center;
    gap: 4px;
    height: 100%;
    width: 0.75rem;
  }

  .week-bar {
    align-self: end;
    width: 100%;
    border-radius: 3px;
    background: linear-gradient(
      180deg,
      var(--accent),
      light-dark(
        oklch(0.7 calc(var(--brand-chroma) * 0.8) var(--brand-hue)),
        oklch(0.5 calc(var(--brand-chroma) * 0.9) var(--brand-hue))
      )
    );
  }

  .week-bar.zero {
    background: var(--line-strong);
  }

  .week-label {
    font-size: 0.625rem;
    color: var(--text-muted);
    line-height: 1;
  }

  .week-day.today .week-label {
    color: var(--accent);
    font-weight: 700;
  }

  .ring {
    width: 4.25rem;
    height: 4.25rem;
    transform: rotate(-90deg);
  }

  .ring circle {
    fill: none;
    stroke-width: 5;
  }

  .ring-track {
    stroke: var(--surface-sunken);
  }

  .ring-value {
    stroke: var(--accent);
    stroke-linecap: round;
    filter: drop-shadow(0 0 4px var(--glow-accent));
    transition: stroke-dasharray var(--duration-slow) var(--ease-out);
  }

  .skeleton {
    min-height: 9.5rem;
    animation: pulse 1.4s var(--ease-in-out) infinite alternate;
  }

  .skeleton.tall {
    min-height: 22rem;
  }

  @keyframes pulse {
    to {
      opacity: 0.5;
    }
  }

  /* ── Main grid ───────────────────────────────────────────────── */
  .grid-main {
    display: grid;
    grid-template-columns: minmax(0, 5fr) minmax(0, 7fr);
    gap: var(--space-4);
    margin-bottom: var(--space-4);
  }

  .panel-block {
    display: flex;
    flex-direction: column;
    padding: var(--space-4);
    min-width: 0;
  }

  .block-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
    padding: var(--space-1) var(--space-1) var(--space-4);
  }

  .block-head h3 {
    font-size: var(--text-section);
    font-weight: 500;
  }

  .block-empty {
    flex: 1;
    display: grid;
    place-items: center;
    align-content: center;
    gap: var(--space-3);
    min-height: 12rem;
    text-align: center;
  }

  .block-empty p {
    max-width: 26ch;
    font-size: var(--text-small);
    color: var(--text-muted);
  }

  .work-list {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
    gap: var(--space-2);
  }

  .work-row {
    width: 100%;
    justify-content: flex-start;
    gap: var(--space-3);
    min-height: 3.5rem;
    padding: var(--space-2) var(--space-3) var(--space-2) var(--space-2);
    text-align: left;
    font-weight: 400;
    box-shadow: none;
  }

  .work-row:hover:not(:disabled) {
    --veil: var(--hover-veil);
  }

  .work-text {
    display: grid;
    gap: 0.125rem;
    flex: 1;
    min-width: 0;
  }

  .work-title {
    font-weight: 520;
    color: var(--text);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .work-meta {
    display: flex;
    gap: var(--space-2);
    font-size: var(--text-xs);
    color: var(--text-muted);
    white-space: nowrap;
    overflow: hidden;
  }

  .work-meta .key {
    font-size: inherit;
  }

  .late {
    color: var(--danger);
    font-weight: 560;
  }

  /* ── Timeline ────────────────────────────────────────────────── */
  .timeline-wrap {
    flex: 1;
    display: grid;
    grid-template-rows: 1fr auto;
    min-height: 0;
    overflow-x: auto;
    overscroll-behavior-x: contain;
  }

  .timeline {
    position: relative;
    min-width: 34rem;
    min-height: calc(var(--rows) * 3.25rem + 1rem);
  }

  .grid-lines {
    position: absolute;
    inset: 0;
    display: grid;
    grid-template-columns: repeat(14, 1fr);
  }

  .grid-lines span {
    border-left: 1px dashed var(--line);
  }

  .today-line {
    position: absolute;
    top: 0;
    bottom: 0;
    left: 0;
    width: 2px;
    margin-left: -1px;
    background: linear-gradient(180deg, var(--accent), transparent);
  }

  .today-line::before {
    content: "";
    position: absolute;
    top: -4px;
    left: -4px;
    width: 10px;
    height: 10px;
    border-radius: 50%;
    background: var(--accent);
    box-shadow: 0 0 0 4px var(--accent-soft);
  }

  .event {
    position: absolute;
    left: calc(var(--day) / 14 * 100%);
    top: calc(var(--row) * 3.25rem + 0.75rem);
    max-width: min(17rem, calc(100% - var(--day) / 14 * 100%));
    min-height: 2.75rem;
    justify-content: flex-start;
    gap: var(--space-3);
    padding: 0.3rem 0.4rem 0.3rem 0.75rem;
    margin-left: 4px;
    border-radius: var(--radius);
    text-align: left;
    font-weight: 400;
    box-shadow: 0 8px 20px -12px oklch(0 0 0 / 0.6);
  }

  /* Late in the window, an event hangs left of its date so it stays in view. */
  .event.flip {
    max-width: min(17rem, calc(var(--day) / 14 * 100%));
    transform: translateX(-100%);
    margin-left: -4px;
  }

  .event-text {
    display: grid;
    min-width: 0;
    line-height: 1.25;
  }

  .event-title {
    font-size: var(--text-small);
    font-weight: 540;
    color: var(--text);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .event-meta {
    font-size: 0.6875rem;
    color: var(--text-muted);
    white-space: nowrap;
  }

  .ticks {
    display: grid;
    grid-template-columns: repeat(14, 1fr);
    min-width: 34rem;
    padding-top: var(--space-2);
  }

  .ticks span {
    font-size: 0.6875rem;
    color: var(--text-muted);
    white-space: nowrap;
    font-variant-numeric: tabular-nums;
  }

  .ticks span.today {
    color: var(--accent);
    font-weight: 650;
  }

  /* ── Projects ────────────────────────────────────────────────── */
  .project-rows {
    list-style: none;
    margin: 0;
    padding: 0;
    display: grid;
  }

  .project-row {
    width: 100%;
    display: grid;
    grid-template-columns: auto minmax(0, 1fr) 6rem minmax(8rem, 14rem) auto;
    align-items: center;
    gap: var(--space-4);
    min-height: 3.75rem;
    padding: var(--space-2) var(--space-2);
    border: 0;
    border-top: var(--border-width) solid var(--line);
    border-radius: 0;
    background: transparent;
    box-shadow: none;
    text-align: left;
    font-weight: 400;
  }

  li:first-child .project-row {
    border-top: 0;
  }

  .project-row:hover:not(:disabled) {
    background: var(--hover);
    border-radius: var(--radius-sm);
  }

  .pr-name {
    display: grid;
    min-width: 0;
  }

  .pr-title {
    font-weight: 540;
    color: var(--text);
  }

  .pr-sub {
    font-size: var(--text-xs);
    color: var(--text-muted);
  }

  .pr-open {
    font-size: var(--text-small);
    color: var(--text-muted);
  }

  .pr-open strong {
    font-weight: 600;
  }

  .pr-progress {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }

  .bar {
    flex: 1;
    height: 0.375rem;
    border-radius: var(--radius-full);
    background: var(--surface-sunken);
    overflow: hidden;
  }

  .bar span {
    display: block;
    height: 100%;
    border-radius: inherit;
    background: linear-gradient(90deg, var(--accent-soft), var(--accent));
  }

  .pct {
    min-width: 2.5rem;
    text-align: right;
    font-size: var(--text-small);
    color: var(--text-secondary);
  }

  .project-row :global(.pr-chevron) {
    color: var(--text-muted);
  }

  .sample-note {
    margin-top: var(--space-3);
  }

  /* ── Responsive ──────────────────────────────────────────────── */
  @media (max-width: 80rem) {
    .stats {
      grid-template-columns: repeat(2, minmax(0, 1fr));
    }
  }

  @media (max-width: 64rem) {
    .grid-main {
      grid-template-columns: minmax(0, 1fr);
    }
  }

  @media (max-width: 40rem) {
    .stats {
      gap: var(--space-3);
    }

    .stat {
      min-height: 8.5rem;
      padding: var(--space-3) var(--space-4);
    }

    .stat-viz {
      display: none;
    }

    .stat-value {
      font-size: 1.875rem;
    }

    .stat-head h3 {
      font-size: var(--text-small);
    }

    .project-row {
      grid-template-columns: auto minmax(0, 1fr) auto;
    }

    .pr-open,
    .project-row :global(.pr-chevron) {
      display: none;
    }

    .pr-progress {
      grid-column: 2 / -1;
    }
  }
</style>
