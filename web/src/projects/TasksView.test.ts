import {
  fireEvent,
  render,
  screen,
  waitFor,
  within,
} from "@testing-library/svelte";
import { expect, test, vi } from "vitest";
import TasksView from "./TasksView.svelte";

const mockProject = {
  id: "proj-1",
  workspace_id: "ws-1",
  name: "Core Platform",
  slug: "core-platform",
  key: "CORE",
  description: "Platform core",
  task_counter: 2,
  archived: false,
  created_at: "2026-11-01T00:00:00Z",
  updated_at: "2026-11-01T00:00:00Z",
};

test("loads and displays project tasks with human keys and meta", async () => {
  const tasks = [
    {
      id: "task-2",
      workspace_id: "ws-1",
      project_id: "proj-1",
      project_key: "CORE",
      task_number: 2,
      human_key: "CORE-2",
      title: "Write documentation",
      description: "Document the public API",
      status: "unstarted",
      priority: "medium",
      due_date: "2026-11-20",
      created_by: "user-1",
      created_at: "2026-11-01T00:00:00Z",
      updated_at: "2026-11-01T00:00:00Z",
    },
    {
      id: "task-1",
      workspace_id: "ws-1",
      project_id: "proj-1",
      project_key: "CORE",
      task_number: 1,
      human_key: "CORE-1",
      title: "Set up PostgreSQL",
      description: "Migration scripts",
      status: "completed",
      priority: "urgent",
      due_date: null,
      created_by: "user-1",
      created_at: "2026-11-01T00:00:00Z",
      updated_at: "2026-11-01T00:00:00Z",
    },
  ];

  const fetchMock = vi.fn().mockResolvedValue(
    new Response(JSON.stringify({ tasks, total: 2, limit: 20, offset: 0 }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }),
  );
  vi.stubGlobal("fetch", fetchMock);

  render(TasksView, {
    workspace: {
      id: "ws-1",
      name: "Acme",
      slug: "acme",
      role: "owner",
    },
    project: mockProject,
  });

  await waitFor(() => {
    expect(screen.getByText("CORE-2")).toBeTruthy();
    expect(screen.getByText("Write documentation")).toBeTruthy();
    expect(screen.getByText("CORE-1")).toBeTruthy();
    expect(screen.getByText("Set up PostgreSQL")).toBeTruthy();
    expect(screen.getByText(/showing 1–2 of 2 tasks/i)).toBeTruthy();
  });
});

test("allows creating a new task", async () => {
  Object.defineProperty(document, "cookie", {
    writable: true,
    value: "trestle-csrf=test-csrf-token",
  });

  const tasks: unknown[] = [];

  const fetchMock = vi.fn().mockImplementation((url, init) => {
    if (init?.method === "POST") {
      return Promise.resolve(
        new Response(
          JSON.stringify({
            id: "task-3",
            workspace_id: "ws-1",
            project_id: "proj-1",
            project_key: "CORE",
            task_number: 3,
            human_key: "CORE-3",
            title: "Implement rate limiting",
            description: "Token bucket algorithm",
            status: "unstarted",
            priority: "high",
            due_date: null,
            created_by: "user-1",
            created_at: "2026-11-02T00:00:00Z",
            updated_at: "2026-11-02T00:00:00Z",
          }),
          { status: 201, headers: { "Content-Type": "application/json" } },
        ),
      );
    }
    return Promise.resolve(
      new Response(
        JSON.stringify({ tasks, total: tasks.length, limit: 20, offset: 0 }),
        {
          status: 200,
          headers: { "Content-Type": "application/json" },
        },
      ),
    );
  });
  vi.stubGlobal("fetch", fetchMock);

  render(TasksView, {
    workspace: {
      id: "ws-1",
      name: "Acme",
      slug: "acme",
      role: "owner",
    },
    project: mockProject,
  });

  await waitFor(() => {
    expect(screen.getByText(/no tasks match your selection/i)).toBeTruthy();
  });

  const newBtn = screen.getByRole("button", { name: /new task/i });
  await fireEvent.click(newBtn);

  await waitFor(() => {
    expect(screen.getByText(/new task in CORE/i)).toBeTruthy();
  });

  const titleInput = screen.getByLabelText(/title/i);
  await fireEvent.input(titleInput, {
    target: { value: "Implement rate limiting" },
  });

  const createSubmit = screen.getByRole("button", { name: "Save task" });
  await fireEvent.click(createSubmit);

  await waitFor(() => {
    expect(screen.getByText(/task CORE-3 created/i)).toBeTruthy();
  });
});

test("clicking a task opens the task detail drawer", async () => {
  const tasks = [
    {
      id: "task-10",
      workspace_id: "ws-1",
      project_id: "proj-1",
      project_key: "CORE",
      task_number: 10,
      human_key: "CORE-10",
      title: "Interactive detail test task",
      description: "Detailed description for drawer",
      status: "started",
      priority: "high",
      due_date: "2026-11-25",
      created_by: "user-1",
      labels: [],
      assignees: [],
      created_at: "2026-11-01T00:00:00Z",
      updated_at: "2026-11-01T00:00:00Z",
    },
  ];

  const fetchMock = vi.fn().mockImplementation((url: string) => {
    if (url.includes("/labels")) {
      return Promise.resolve(
        new Response(JSON.stringify([]), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    }
    if (url.includes("/members")) {
      return Promise.resolve(
        new Response(JSON.stringify({ members: [] }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    }
    if (url.includes("/tasks?")) {
      return Promise.resolve(
        new Response(
          JSON.stringify({ tasks, total: 1, limit: 20, offset: 0 }),
          {
            status: 200,
            headers: { "Content-Type": "application/json" },
          },
        ),
      );
    }
    return Promise.resolve(
      new Response(JSON.stringify({}), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
  });
  vi.stubGlobal("fetch", fetchMock);

  render(TasksView, {
    workspace: {
      id: "ws-1",
      name: "Acme",
      slug: "acme",
      role: "owner",
    },
    project: mockProject,
  });

  await waitFor(() => {
    expect(screen.getByText("Interactive detail test task")).toBeTruthy();
    expect(screen.getByText("CORE-10")).toBeTruthy();
  });

  // Clicking on task row opens drawer
  const taskRow = screen.getByText("Interactive detail test task");
  await fireEvent.click(taskRow);

  await waitFor(() => {
    // Drawer panel is open with dialog role and human key in header
    expect(screen.getByRole("dialog")).toBeTruthy();
    expect(
      screen.getByDisplayValue("Interactive detail test task"),
    ).toBeTruthy();
    expect(
      screen.getByDisplayValue("Detailed description for drawer"),
    ).toBeTruthy();
  });

  // Close drawer with Close button
  const closeBtn = screen.getByRole("button", { name: /close task details/i });
  await fireEvent.click(closeBtn);

  await waitFor(() => {
    expect(screen.queryByRole("dialog")).toBeNull();
  });
});

test("renders custom project statuses in top progress rail and allows filtering", async () => {
  const customStatuses = [
    {
      id: "s-1",
      project_id: "proj-1",
      workspace_id: "ws-1",
      name: "Backlog",
      category: "backlog",
      color: "#4c544c",
      position: 0,
    },
    {
      id: "s-2",
      project_id: "proj-1",
      workspace_id: "ws-1",
      name: "To Do",
      category: "unstarted",
      color: "#1e4d79",
      position: 1,
    },
    {
      id: "s-3",
      project_id: "proj-1",
      workspace_id: "ws-1",
      name: "In Progress",
      category: "started",
      color: "#775300",
      position: 2,
    },
    {
      id: "s-4",
      project_id: "proj-1",
      workspace_id: "ws-1",
      name: "Done",
      category: "completed",
      color: "#26633b",
      position: 3,
    },
    {
      id: "s-5",
      project_id: "proj-1",
      workspace_id: "ws-1",
      name: "In Review",
      category: "started",
      color: "#6366f1",
      position: 4,
    },
    {
      id: "s-6",
      project_id: "proj-1",
      workspace_id: "ws-1",
      name: "Cancelled",
      category: "cancelled",
      color: "#9f302b",
      position: 5,
    },
  ];

  const tasks = [
    {
      id: "task-100",
      workspace_id: "ws-1",
      project_id: "proj-1",
      project_key: "CORE",
      task_number: 100,
      human_key: "CORE-100",
      title: "Review PR",
      status: "started",
      status_id: "s-5",
      status_name: "In Review",
      status_color: "#6366f1",
      priority: "medium",
      due_date: null,
      created_by: "user-1",
      created_at: "2026-11-01T00:00:00Z",
      updated_at: "2026-11-01T00:00:00Z",
    },
  ];

  const fetchMock = vi.fn().mockImplementation((url: string) => {
    if (url.includes("/statuses")) {
      return Promise.resolve(
        new Response(JSON.stringify({ statuses: customStatuses }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    }
    if (url.includes("status_id=s-5")) {
      return Promise.resolve(
        new Response(JSON.stringify({ tasks, total: 1 }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    }
    if (url.includes("&status_id=")) {
      return Promise.resolve(
        new Response(JSON.stringify({ tasks: [], total: 0 }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    }
    if (url.includes("/tasks?")) {
      return Promise.resolve(
        new Response(
          JSON.stringify({ tasks, total: 1, limit: 20, offset: 0 }),
          {
            status: 200,
            headers: { "Content-Type": "application/json" },
          },
        ),
      );
    }
    return Promise.resolve(
      new Response(JSON.stringify({}), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
  });
  vi.stubGlobal("fetch", fetchMock);

  render(TasksView, {
    workspace: {
      id: "ws-1",
      name: "Acme",
      slug: "acme",
      role: "owner",
    },
    project: mockProject,
  });

  await waitFor(() => {
    const rail = screen.getByRole("group", { name: /filter by status/i });
    expect(
      within(rail).getByRole("button", { name: /in review/i }),
    ).toBeTruthy();
    expect(
      within(rail).getByRole("button", { name: /all tasks/i }),
    ).toBeTruthy();
    expect(within(rail).getByRole("button", { name: /backlog/i })).toBeTruthy();
    expect(within(rail).getByRole("button", { name: /to do/i })).toBeTruthy();
    expect(within(rail).getByRole("button", { name: /done/i })).toBeTruthy();
    expect(
      within(rail).getByRole("button", { name: /cancelled/i }),
    ).toBeTruthy();
    expect(within(rail).getByText(/0 of 1 tasks done/i)).toBeTruthy();
  });

  const rail = screen.getByRole("group", { name: /filter by status/i });
  const inReviewBtn = within(rail).getByRole("button", { name: /in review/i });
  await fireEvent.click(inReviewBtn);

  await waitFor(() => {
    expect(within(rail).getByText(/In Review tasks/i)).toBeTruthy();
    expect(fetchMock).toHaveBeenCalledWith(
      expect.stringContaining("status_id=s-5"),
      expect.anything(),
    );
  });
});

test("allows switching to Milestones view and displays project milestones with progress", async () => {
  const milestones = [
    {
      id: "m-1",
      workspace_id: "ws-1",
      project_id: "proj-1",
      title: "Q4 Launch",
      description: "Q4 public beta deliverables",
      start_date: "2026-10-01",
      due_date: "2026-12-31",
      end_date: "2026-12-31",
      color: "#6366f1",
      status: "open",
      total_tasks: 4,
      completed_tasks: 2,
      open_tasks: 2,
      created_at: "2026-10-01T00:00:00Z",
      updated_at: "2026-10-01T00:00:00Z",
    },
  ];

  const fetchMock = vi.fn().mockImplementation((url: string) => {
    if (url.includes("/milestones")) {
      return Promise.resolve(
        new Response(JSON.stringify({ milestones }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    }
    if (url.includes("/tasks?")) {
      return Promise.resolve(
        new Response(JSON.stringify({ tasks: [], total: 0 }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    }
    return Promise.resolve(
      new Response(JSON.stringify({}), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
  });
  vi.stubGlobal("fetch", fetchMock);

  render(TasksView, {
    workspace: {
      id: "ws-1",
      name: "Acme",
      slug: "acme",
      role: "owner",
    },
    project: mockProject,
  });

  const layoutGroup = screen.getByRole("group", { name: /task layout/i });
  const milestoneTabBtn = within(layoutGroup).getByRole("button", {
    name: /milestones/i,
  });
  await fireEvent.click(milestoneTabBtn);

  await waitFor(() => {
    expect(screen.getByText("Q4 Launch")).toBeTruthy();
    expect(screen.getByText("Q4 public beta deliverables")).toBeTruthy();
    expect(screen.getByText(/\(50%\)/)).toBeTruthy();
  });
});
