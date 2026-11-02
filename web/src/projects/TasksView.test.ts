import { fireEvent, render, screen, waitFor } from "@testing-library/svelte";
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
