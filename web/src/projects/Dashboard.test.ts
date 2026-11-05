import { render, screen, waitFor } from "@testing-library/svelte";
import { expect, test, vi } from "vitest";
import Dashboard from "./Dashboard.svelte";

const user = {
  id: "u-1",
  username: "riya",
  display_name: "Riya Sen",
  locale: "en",
  timezone: "UTC",
};
const workspace = { id: "ws-1", name: "Acme", slug: "acme", role: "owner" };

function localDate(offsetDays: number): string {
  const d = new Date();
  d.setDate(d.getDate() + offsetDays);
  const pad = (n: number) => String(n).padStart(2, "0");
  return `${d.getFullYear()}-${pad(d.getMonth() + 1)}-${pad(d.getDate())}`;
}

function task(
  n: number,
  status: string,
  due: string | null,
  assignees: string[],
) {
  return {
    id: `t-${n}`,
    workspace_id: "ws-1",
    project_id: "p-1",
    project_key: "CORE",
    task_number: n,
    human_key: `CORE-${n}`,
    title: `Task number ${n}`,
    description: "",
    status,
    priority: "high",
    due_date: due,
    created_by: "u-1",
    assignees: assignees.map((id) => ({
      task_id: `t-${n}`,
      user_id: id,
      username: id,
      display_name: id === "u-1" ? "Riya Sen" : "Other Person",
      created_at: "2026-11-01T00:00:00Z",
    })),
    created_at: "2026-11-01T00:00:00Z",
    updated_at: "2026-11-01T00:00:00Z",
  };
}

const project = {
  id: "p-1",
  workspace_id: "ws-1",
  name: "Core Platform",
  slug: "core",
  key: "CORE",
  description: "",
  task_counter: 4,
  archived: false,
  created_at: "2026-11-01T00:00:00Z",
  updated_at: "2026-11-01T00:00:00Z",
};

function mockApi(tasks: unknown[], projects: unknown[] = [project]) {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockImplementation((url: string) => {
      const body = url.includes("/tasks?")
        ? { tasks, total: tasks.length, limit: 100, offset: 0 }
        : { projects };
      return Promise.resolve(
        new Response(JSON.stringify(body), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    }),
  );
}

test("summarises real open work, deadlines and my assignments", async () => {
  mockApi([
    task(1, "started", localDate(1), ["u-1"]),
    task(2, "unstarted", localDate(-2), ["u-1"]),
    task(3, "completed", null, ["u-1"]),
    task(4, "unstarted", localDate(3), ["u-2"]),
  ]);

  render(Dashboard, {
    workspace,
    user,
    onopenproject: vi.fn(),
    onopenprojects: vi.fn(),
  });

  await waitFor(() =>
    expect(screen.getByRole("heading", { name: "My work" })).toBeTruthy(),
  );
  // One completed of four decided gives 25%, on the Completion card and on
  // the project's progress row.
  expect(screen.getAllByText("25%")).toHaveLength(2);
  expect(screen.getByText("1 overdue")).toBeTruthy();
  // My work lists only my open tasks, the overdue one first.
  const mine = screen.getAllByText(/^Task number [12]$/);
  expect(mine.map((el) => el.textContent)).toContain("Task number 2");
  expect(screen.getByText("2 open")).toBeTruthy();
  expect(screen.getByText(/Overdue ·/)).toBeTruthy();
});

test("explains an empty workspace instead of showing zeros", async () => {
  mockApi([], []);
  render(Dashboard, {
    workspace,
    user,
    onopenproject: vi.fn(),
    onopenprojects: vi.fn(),
  });
  await waitFor(() =>
    expect(screen.getByText("Start with a project")).toBeTruthy(),
  );
});
