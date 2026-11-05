import { fireEvent, render, screen, waitFor } from "@testing-library/svelte";
import { expect, test, vi } from "vitest";
import TaskDetailDrawer from "./TaskDetailDrawer.svelte";
import type { Task } from "./TasksView.svelte";

const mockWorkspace = {
  id: "ws-1",
  name: "Acme Corp",
  slug: "acme-corp",
  role: "owner",
};

const mockProject = {
  id: "proj-1",
  workspace_id: "ws-1",
  name: "Platform Core",
  slug: "platform-core",
  key: "CORE",
  description: "Core infrastructure",
  task_counter: 10,
  archived: false,
  created_at: "2026-11-01T00:00:00Z",
  updated_at: "2026-11-01T00:00:00Z",
};

const mockTask: Task = {
  id: "task-101",
  workspace_id: "ws-1",
  project_id: "proj-1",
  project_key: "CORE",
  task_number: 42,
  human_key: "CORE-42",
  title: "Implement database connection pooling",
  description: "Use deadpool-postgres for connection pooling",
  status: "started",
  priority: "high",
  due_date: "2026-11-15",
  created_by: "user-alice",
  created_at: "2026-11-02T10:00:00Z",
  updated_at: "2026-11-02T10:00:00Z",
  labels: [
    {
      id: "label-1",
      workspace_id: "ws-1",
      project_id: "proj-1",
      name: "Backend",
      color: "#315b3b",
      created_at: "2026-11-01T00:00:00Z",
    },
  ],
  assignees: [
    {
      task_id: "task-101",
      user_id: "user-alice",
      username: "alice",
      display_name: "Alice Engineer",
      created_at: "2026-11-02T10:05:00Z",
    },
  ],
};

function setupFetchMock(customResponses?: Record<string, unknown>) {
  Object.defineProperty(document, "cookie", {
    writable: true,
    value: "trestle-csrf=test-csrf-token",
  });

  const mockSubtasks = [
    {
      id: "subtask-1",
      workspace_id: "ws-1",
      project_id: "proj-1",
      project_key: "CORE",
      task_number: 43,
      human_key: "CORE-43",
      title: "Write healthcheck query",
      description: "",
      status: "completed",
      priority: "medium",
      due_date: null,
      parent_id: "task-101",
      created_by: "user-alice",
      created_at: "2026-11-02T10:10:00Z",
      updated_at: "2026-11-02T10:15:00Z",
    },
  ];

  const mockComments = [
    {
      id: "comment-1",
      workspace_id: "ws-1",
      task_id: "task-101",
      user_id: "user-alice",
      author_username: "alice",
      author_name: "Alice Engineer",
      content: "Initial pooled tests show 3x throughput improvement.",
      created_at: "2026-11-02T10:20:00Z",
      updated_at: "2026-11-02T10:20:00Z",
    },
  ];

  const mockActivities = [
    {
      id: "act-1",
      workspace_id: "ws-1",
      task_id: "task-101",
      actor_id: "user-alice",
      actor_name: "Alice Engineer",
      action: "task.status_changed",
      details: { status: "started" },
      created_at: "2026-11-02T10:05:00Z",
    },
  ];

  const mockLabels = [
    {
      id: "label-1",
      workspace_id: "ws-1",
      project_id: "proj-1",
      name: "Backend",
      color: "#315b3b",
      created_at: "2026-11-01T00:00:00Z",
    },
    {
      id: "label-2",
      workspace_id: "ws-1",
      project_id: "proj-1",
      name: "Database",
      color: "#1e4d79",
      created_at: "2026-11-01T00:00:00Z",
    },
  ];

  const mockMembers = [
    {
      id: "mem-1",
      user_id: "user-alice",
      username: "alice",
      display_name: "Alice Engineer",
      role: "owner",
    },
    {
      id: "mem-2",
      user_id: "user-bob",
      username: "bob",
      display_name: "Bob Architect",
      role: "member",
    },
  ];

  const fetchMock = vi
    .fn()
    .mockImplementation((url: string, init?: RequestInit) => {
      if (url.includes("/subtasks") && init?.method === "POST") {
        return Promise.resolve(
          new Response(
            JSON.stringify({
              id: "subtask-2",
              human_key: "CORE-44",
              title: "Configure max pool size",
              status: "unstarted",
              parent_id: "task-101",
            }),
            { status: 201, headers: { "Content-Type": "application/json" } },
          ),
        );
      }
      if (url.includes("/subtasks")) {
        return Promise.resolve(
          new Response(JSON.stringify(mockSubtasks), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          }),
        );
      }
      if (url.includes("/comments") && init?.method === "POST") {
        return Promise.resolve(
          new Response(
            JSON.stringify({
              id: "comment-2",
              task_id: "task-101",
              user_id: "user-alice",
              author_username: "alice",
              author_display_name: "Alice Engineer",
              content: "Benchmark completed without leaks.",
              created_at: "2026-11-02T10:30:00Z",
              updated_at: "2026-11-02T10:30:00Z",
            }),
            { status: 201, headers: { "Content-Type": "application/json" } },
          ),
        );
      }
      if (url.includes("/comments") && init?.method === "DELETE") {
        return Promise.resolve(new Response(null, { status: 204 }));
      }
      if (url.includes("/comments")) {
        return Promise.resolve(
          new Response(JSON.stringify({ comments: mockComments }), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          }),
        );
      }
      if (url.includes("/activity")) {
        return Promise.resolve(
          new Response(JSON.stringify({ activities: mockActivities }), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          }),
        );
      }
      if (url.includes("/labels") && init?.method === "POST") {
        return Promise.resolve(
          new Response(
            JSON.stringify({
              id: "label-3",
              workspace_id: "ws-1",
              project_id: "proj-1",
              name: "Performance",
              color: "#6b4c9a",
            }),
            { status: 201, headers: { "Content-Type": "application/json" } },
          ),
        );
      }
      if (url.includes("/labels")) {
        return Promise.resolve(
          new Response(JSON.stringify(mockLabels), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          }),
        );
      }
      if (url.includes("/members")) {
        return Promise.resolve(
          new Response(JSON.stringify({ members: mockMembers }), {
            status: 200,
            headers: { "Content-Type": "application/json" },
          }),
        );
      }
      if (url.includes("/assignees") && init?.method === "POST") {
        return Promise.resolve(new Response(null, { status: 204 }));
      }
      if (url.includes("/assignees") && init?.method === "DELETE") {
        return Promise.resolve(new Response(null, { status: 204 }));
      }
      if (url.includes("/tasks/task-101") && init?.method === "PATCH") {
        return Promise.resolve(
          new Response(
            JSON.stringify({
              ...mockTask,
              title: "Implement database connection pooling (Updated)",
            }),
            { status: 200, headers: { "Content-Type": "application/json" } },
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
  return fetchMock;
}

test("renders task details, human key, and sidebar metadata", async () => {
  setupFetchMock();
  const onclose = vi.fn();

  render(TaskDetailDrawer, {
    task: mockTask,
    workspace: mockWorkspace,
    project: mockProject,
    currentUserId: "user-alice",
    onclose,
  });

  expect(screen.getByText("CORE-42")).toBeTruthy();
  expect(screen.getByText("Platform Core")).toBeTruthy();
  expect(
    screen.getByDisplayValue("Implement database connection pooling"),
  ).toBeTruthy();
  expect(
    screen.getByDisplayValue("Use deadpool-postgres for connection pooling"),
  ).toBeTruthy();

  await waitFor(() => {
    // Subtask loaded
    expect(screen.getByText("Write healthcheck query")).toBeTruthy();
    expect(screen.getByText("CORE-43")).toBeTruthy();
    // Comment loaded
    expect(
      screen.getByText("Initial pooled tests show 3x throughput improvement."),
    ).toBeTruthy();
    expect(screen.getAllByText("Alice Engineer").length).toBeGreaterThan(0);
    // Activity timeline loaded
    expect(screen.getByText(/updated status to started/i)).toBeTruthy();
  });
});

test("allows creating a new comment", async () => {
  setupFetchMock();
  const onclose = vi.fn();

  render(TaskDetailDrawer, {
    task: mockTask,
    workspace: mockWorkspace,
    project: mockProject,
    currentUserId: "user-alice",
    onclose,
  });

  const commentInput = screen.getByLabelText(/write a comment/i);
  await fireEvent.input(commentInput, {
    target: { value: "Benchmark completed without leaks." },
  });

  const postBtn = screen.getByRole("button", { name: /post comment/i });
  await fireEvent.click(postBtn);

  await waitFor(() => {
    expect(screen.getByText("Comment posted.")).toBeTruthy();
  });
});

test("allows adding a subtask", async () => {
  setupFetchMock();
  const onclose = vi.fn();

  render(TaskDetailDrawer, {
    task: mockTask,
    workspace: mockWorkspace,
    project: mockProject,
    currentUserId: "user-alice",
    onclose,
  });

  const subtaskInput = screen.getByLabelText(/new subtask title/i);
  await fireEvent.input(subtaskInput, {
    target: { value: "Configure max pool size" },
  });

  const addBtn = screen.getByRole("button", { name: "Add subtask" });
  await fireEvent.click(addBtn);

  await waitFor(() => {
    expect(screen.getByText("Subtask added.")).toBeTruthy();
  });
});

test("closes drawer on Escape key press", async () => {
  setupFetchMock();
  const onclose = vi.fn();

  render(TaskDetailDrawer, {
    task: mockTask,
    workspace: mockWorkspace,
    project: mockProject,
    currentUserId: "user-alice",
    onclose,
  });

  await fireEvent.keyDown(window, { key: "Escape" });
  expect(onclose).toHaveBeenCalled();
});

test("unsaved title edits are flagged and guarded before closing", async () => {
  setupFetchMock();
  const onclose = vi.fn();
  const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);

  render(TaskDetailDrawer, {
    task: mockTask,
    workspace: mockWorkspace,
    project: mockProject,
    currentUserId: "user-alice",
    onclose,
  });

  expect(screen.getByText("All changes saved")).toBeTruthy();
  await fireEvent.input(screen.getByLabelText("Title"), {
    target: { value: "Implement pooled connections" },
  });
  expect(screen.getByText("Unsaved changes")).toBeTruthy();
  expect(screen.getByRole("button", { name: "Save details" })).toHaveProperty(
    "disabled",
    false,
  );

  // Declining the prompt keeps the drawer, and the edit, open.
  await fireEvent.click(
    screen.getByRole("button", { name: /close task details/i }),
  );
  expect(confirm).toHaveBeenCalled();
  expect(onclose).not.toHaveBeenCalled();

  // Discard restores the saved title without a prompt.
  await fireEvent.click(screen.getByRole("button", { name: "Discard" }));
  expect(screen.queryByText("Unsaved changes")).toBeNull();
  await fireEvent.click(
    screen.getByRole("button", { name: /close task details/i }),
  );
  expect(onclose).toHaveBeenCalledTimes(1);
  confirm.mockRestore();
});
