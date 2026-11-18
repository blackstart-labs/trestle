<script lang="ts">
  import { onMount } from "svelte";
  import Button from "../ui/Button.svelte";
  import DatePicker from "../ui/DatePicker.svelte";
  import Select from "../ui/Select.svelte";
  import Feedback from "../ui/Feedback.svelte";
  import { api, message } from "../lib/api";
  let { workspace }: { workspace: { id: string; name: string; role: string } } =
    $props();
  const manager = $derived(["owner", "admin"].includes(workspace.role));
  const iso = (d: Date) =>
    `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, "0")}-${String(d.getDate()).padStart(2, "0")}`;
  const today = new Date();
  let from = $state(iso(new Date(today.getFullYear(), today.getMonth(), 1))),
    to = $state(iso(today)),
    project = $state(""),
    user = $state(""),
    offset = $state(0),
    more = $state(false),
    loading = $state(false),
    exporting = $state(false),
    error = $state(""),
    total = $state(0),
    count = $state(0),
    applied = $state("");
  type Entry = {
    id: string;
    date: string;
    user_name: string;
    project_name: string;
    task_key: string;
    task_title: string;
    hours: number;
    description: string;
    task_id: string;
    project_id: string;
  };
  let entries = $state<Entry[]>([]),
    projects = $state<{ id: string; name: string }[]>([]),
    members = $state<{ user_id: string; display_name: string }[]>([]);
  function params() {
    return new URLSearchParams({
      from,
      to,
      ...(project ? { project_id: project } : {}),
      ...(manager && user ? { user_id: user } : {}),
    }).toString();
  }
  async function load(newSearch = true) {
    loading = true;
    error = "";
    if (newSearch) {
      offset = 0;
      applied = params();
    }
    try {
      const data = await api(
        `/api/v1/workspaces/${workspace.id}/time-report?${applied}&offset=${offset}`,
      );
      entries = data.entries;
      total = data.total_hours;
      count = data.total_entries;
      more = data.has_more;
    } catch (e) {
      error = message(e);
    } finally {
      loading = false;
    }
  }
  function period(p: string) {
    const d = new Date();
    if (p === "day") {
      from = iso(d);
      to = iso(d);
    }
    if (p === "week") {
      const start = new Date(d);
      start.setDate(start.getDate() - ((start.getDay() + 6) % 7));
      from = iso(start);
      to = iso(d);
    }
    if (p === "month") {
      from = iso(new Date(d.getFullYear(), d.getMonth(), 1));
      to = iso(d);
    }
    void load();
  }
  async function exportCSV() {
    if (!applied || exporting) return;
    exporting = true;
    error = "";
    try {
      const res = await fetch(
        `/api/v1/workspaces/${workspace.id}/time-report?${applied}&format=csv`,
        { cache: "no-store" },
      );
      if (!res.ok) {
        const d = await res.json();
        throw new Error(d?.error?.message || "Could not export this report.");
      }
      const url = URL.createObjectURL(await res.blob());
      const a = document.createElement("a");
      a.href = url;
      a.download = "trestle-timesheet.csv";
      a.click();
      setTimeout(() => URL.revokeObjectURL(url), 1000);
    } catch (e) {
      error = message(e);
    } finally {
      exporting = false;
    }
  }
  onMount(() => {
    if (workspace.role === "guest") return;
    void load();
    void Promise.all([
      api(`/api/v1/workspaces/${workspace.id}/projects`),
      manager
        ? api(`/api/v1/workspaces/${workspace.id}/members`)
        : Promise.resolve({ members: [] }),
    ])
      .then(([p, m]) => {
        projects = p.projects;
        members = m.members;
      })
      .catch((e) => (error = message(e)));
  });
</script>

<div class="page-head">
  <div>
    <h2>Time reports</h2>
    <p>{manager ? "Workspace" : "Your"} recorded work in {workspace.name}.</p>
  </div>
  <div class="page-actions">
    <Button
      disabled={loading || exporting || !applied || workspace.role === "guest"}
      onclick={exportCSV}>{exporting ? "Exporting…" : "Export CSV"}</Button
    >
  </div>
</div>
{#if workspace.role === "guest"}<Feedback
    >Time reports are available to contributors. Owners and admins can report on
    the full team.</Feedback
  >{:else}
  <div class="report-toolbar">
    <Button disabled={loading} onclick={() => period("day")}>Today</Button
    ><Button disabled={loading} onclick={() => period("week")}>This week</Button
    ><Button disabled={loading} onclick={() => period("month")}
      >This month</Button
    >
  </div>
  <form
    class="report-toolbar"
    onsubmit={(e) => {
      e.preventDefault();
      void load();
    }}
  >
    <DatePicker
      label="From date"
      value={from}
      onchange={(v) => (from = v)}
      disabled={loading}
    /><DatePicker
      label="To date"
      value={to}
      onchange={(v) => (to = v)}
      disabled={loading}
    /><Select
      label="Report project"
      value={project}
      options={[
        { value: "", label: "All projects" },
        ...projects.map((p) => ({ value: p.id, label: p.name })),
      ]}
      onchange={(v) => (project = v)}
      disabled={loading}
    />{#if manager}<Select
        label="Report member"
        value={user}
        options={[
          { value: "", label: "All members" },
          ...members.map((m) => ({ value: m.user_id, label: m.display_name })),
        ]}
        onchange={(v) => (user = v)}
        disabled={loading}
      />{/if}<Button type="submit" variant="primary" disabled={loading}
      >Apply filters</Button
    >
  </form>
  {#if error}<Feedback tone="danger" announcement="alert">{error}</Feedback
    ><Button disabled={loading} onclick={() => load()}>Retry report</Button
    >{/if}
  {#if loading}<Feedback announcement="status">Loading recorded time…</Feedback>
    <div class="panel report-loading" aria-hidden="true"></div>{:else if !error}
    <div class="report-totals panel">
      <div>
        <h3>Total hours</h3>
        <strong>{total.toFixed(2)}</strong>
      </div>
      <div>
        <h3>Time entries</h3>
        <strong>{count}</strong>
      </div>
      <p>
        Totals cover all entries matching the applied filters. Up to 100 entries
        are shown per page.
      </p>
    </div>
    {#each entries as entry (entry.id)}<article class="card report-row">
        <div>
          <p>{entry.date} · {entry.user_name}</p>
          <h3>
            <a
              href={`#task/${workspace.id}/${entry.project_id}/${entry.task_id}`}
              >{entry.task_key} · {entry.task_title}</a
            >
          </h3>
          <p>{entry.project_name}</p>
          {#if entry.description}<p>{entry.description}</p>{/if}
        </div>
        <strong>{entry.hours.toFixed(2)} h</strong>
      </article>{:else}<div class="empty">
        <h3>No recorded time</h3>
        <p>Track time on a task or change the date range.</p>
      </div>{/each}
    {#if more || offset}<div class="report-toolbar">
        <Button
          disabled={loading || offset === 0}
          onclick={() => {
            offset -= 100;
            void load(false);
          }}>Previous page</Button
        >
        <p>Page {offset / 100 + 1}</p>
        <Button
          disabled={loading || !more}
          onclick={() => {
            offset += 100;
            void load(false);
          }}>Next page</Button
        >
      </div>{/if}{/if}{/if}

<style>
  .report-toolbar {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-3);
    align-items: end;
    margin-block: var(--space-4);
  }
  .report-totals {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-6);
    padding: var(--space-5);
    margin-bottom: var(--space-4);
  }
  .report-totals strong {
    font-size: var(--text-title);
  }
  .report-totals p {
    flex: 1 1 18rem;
  }
  .report-row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-4);
    padding: var(--space-4);
    margin-bottom: var(--space-3);
  }
  .report-row > div {
    flex: 1 1 14rem;
    min-width: 0;
    overflow-wrap: anywhere;
  }
  .report-row p {
    margin-block: var(--space-2);
  }
  .report-loading {
    min-height: 12rem;
    background: var(--surface-sunken);
  }
</style>
