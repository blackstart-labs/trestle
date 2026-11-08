import { fireEvent, render, screen, waitFor } from "@testing-library/svelte";
import { beforeEach, expect, test, vi } from "vitest";
import MilestoneModal, { type Milestone } from "./MilestoneModal.svelte";

const mockWorkspace = {
  id: "ws-1",
  name: "Engineering",
  role: "admin",
};

const mockProject = {
  id: "proj-1",
  name: "Core Platform",
  key: "CORE",
};

const mockMilestones: Milestone[] = [
  {
    id: "m-1",
    workspace_id: "ws-1",
    project_id: "proj-1",
    title: "Alpha Launch",
    description: "Initial alpha testing targets",
    start_date: "2026-11-01",
    due_date: "2026-12-01",
    end_date: "2026-12-01",
    color: "#6366f1",
    status: "open",
    total_tasks: 5,
    completed_tasks: 2,
    open_tasks: 3,
    created_at: "2026-11-01T00:00:00Z",
    updated_at: "2026-11-01T00:00:00Z",
  },
  {
    id: "m-2",
    workspace_id: "ws-1",
    project_id: "proj-1",
    title: "Prototype",
    description: "Foundational spike",
    start_date: null,
    due_date: null,
    end_date: null,
    color: "#26633b",
    status: "closed",
    total_tasks: 3,
    completed_tasks: 3,
    open_tasks: 0,
    created_at: "2026-10-15T00:00:00Z",
    updated_at: "2026-10-25T00:00:00Z",
  },
];

beforeEach(() => {
  Object.defineProperty(document, "cookie", {
    configurable: true,
    writable: true,
    value: "trestle-csrf=test-csrf-token",
  });
});

test("renders list of milestones with progress and status", async () => {
  const fetchMock = vi.fn().mockImplementation((url: string) => {
    if (url.includes("/milestones")) {
      return Promise.resolve(
        new Response(JSON.stringify({ milestones: mockMilestones }), {
          status: 200,
          headers: { "Content-Type": "application/json" },
        }),
      );
    }
    return Promise.reject(new Error("unknown URL"));
  });
  vi.stubGlobal("fetch", fetchMock);

  render(MilestoneModal, {
    workspace: mockWorkspace,
    project: mockProject,
    onclose: vi.fn(),
  });

  await waitFor(() => {
    expect(screen.getByText("Alpha Launch")).toBeTruthy();
    expect(screen.getByText("Prototype")).toBeTruthy();
  });

  expect(screen.getByText("Open")).toBeTruthy();
  expect(screen.getByText("Closed")).toBeTruthy();
  expect(screen.getByText("2 of 5 completed (40%)")).toBeTruthy();
  expect(screen.getByText("3 of 3 completed (100%)")).toBeTruthy();
});

test("creates a new milestone and submits POST with CSRF token", async () => {
  let created = false;
  const fetchMock = vi.fn().mockImplementation((url: string, opts?: any) => {
    if (opts?.method === "POST" && url.includes("/milestones")) {
      created = true;
      const body = JSON.parse(opts.body);
      return Promise.resolve(
        new Response(
          JSON.stringify({
            id: "m-3",
            workspace_id: "ws-1",
            project_id: "proj-1",
            title: body.title,
            description: body.description,
            due_date: body.due_date,
            status: "open",
            total_tasks: 0,
            completed_tasks: 0,
            open_tasks: 0,
            created_at: "2026-11-08T00:00:00Z",
            updated_at: "2026-11-08T00:00:00Z",
          }),
          { status: 201, headers: { "Content-Type": "application/json" } },
        ),
      );
    }
    return Promise.resolve(
      new Response(
        JSON.stringify({
          milestones: created
            ? [
                ...mockMilestones,
                {
                  id: "m-3",
                  workspace_id: "ws-1",
                  project_id: "proj-1",
                  title: "Beta Release",
                  description: "Feature complete beta",
                  due_date: null,
                  status: "open",
                  total_tasks: 0,
                  completed_tasks: 0,
                  open_tasks: 0,
                  created_at: "2026-11-08T00:00:00Z",
                  updated_at: "2026-11-08T00:00:00Z",
                },
              ]
            : mockMilestones,
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
  });
  vi.stubGlobal("fetch", fetchMock);

  const onmilestonesupdated = vi.fn();
  render(MilestoneModal, {
    workspace: mockWorkspace,
    project: mockProject,
    onclose: vi.fn(),
    onmilestonesupdated,
  });

  await waitFor(() => {
    expect(screen.getByText("Alpha Launch")).toBeTruthy();
  });

  await fireEvent.click(screen.getByRole("button", { name: "New milestone" }));

  expect(screen.getByRole("heading", { name: "New milestone" })).toBeTruthy();

  await fireEvent.input(screen.getByLabelText("Title"), {
    target: { value: "Beta Release" },
  });
  await fireEvent.input(screen.getByLabelText("Description"), {
    target: { value: "Feature complete beta" },
  });

  await fireEvent.click(
    screen.getByRole("button", { name: "Create milestone" }),
  );

  await waitFor(() => {
    expect(onmilestonesupdated).toHaveBeenCalled();
  });

  expect(fetchMock).toHaveBeenCalledWith(
    "/api/v1/workspaces/ws-1/projects/proj-1/milestones",
    expect.objectContaining({
      method: "POST",
      headers: expect.objectContaining({
        "X-CSRF-Token": "test-csrf-token",
      }),
      body: JSON.stringify({
        title: "Beta Release",
        description: "Feature complete beta",
        start_date: null,
        end_date: null,
        due_date: null,
        color: "#6366f1",
      }),
    }),
  );
});

test("confirms and deletes a milestone", async () => {
  let deleted = false;
  const fetchMock = vi.fn().mockImplementation((url: string, opts?: any) => {
    if (opts?.method === "DELETE" && url.includes("/milestones/m-1")) {
      deleted = true;
      return Promise.resolve(new Response(null, { status: 204 }));
    }
    return Promise.resolve(
      new Response(
        JSON.stringify({
          milestones: deleted ? [mockMilestones[1]] : mockMilestones,
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
  });
  vi.stubGlobal("fetch", fetchMock);

  const onmilestonesupdated = vi.fn();
  render(MilestoneModal, {
    workspace: mockWorkspace,
    project: mockProject,
    onclose: vi.fn(),
    onmilestonesupdated,
  });

  await waitFor(() => {
    expect(screen.getByText("Alpha Launch")).toBeTruthy();
  });

  await fireEvent.click(
    screen.getByRole("button", { name: "Delete milestone Alpha Launch" }),
  );

  expect(
    screen.getByRole("heading", {
      name: 'Delete milestone "Alpha Launch"?',
    }),
  ).toBeTruthy();

  await fireEvent.click(
    screen.getByRole("button", { name: "Delete milestone" }),
  );

  await waitFor(() => {
    expect(onmilestonesupdated).toHaveBeenCalled();
  });

  expect(fetchMock).toHaveBeenCalledWith(
    "/api/v1/workspaces/ws-1/projects/proj-1/milestones/m-1",
    expect.objectContaining({
      method: "DELETE",
      headers: expect.objectContaining({
        "X-CSRF-Token": "test-csrf-token",
      }),
    }),
  );
});
