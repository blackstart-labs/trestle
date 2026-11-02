import { fireEvent, render, screen, waitFor } from "@testing-library/svelte";
import { expect, test, vi } from "vitest";
import StatusManagerModal, {
  type TaskStatus,
} from "./StatusManagerModal.svelte";

const mockWorkspace = {
  id: "ws-1",
  name: "Engineering",
  slug: "engineering",
  role: "admin",
};

const mockProject = {
  id: "proj-1",
  name: "Core Platform",
  key: "CORE",
};

const mockStatuses: TaskStatus[] = [
  {
    id: "st-1",
    workspace_id: "ws-1",
    project_id: "proj-1",
    name: "Backlog",
    category: "backlog",
    color: "#4c544c",
    position: 0,
    created_at: "2026-11-01T00:00:00Z",
    updated_at: "2026-11-01T00:00:00Z",
  },
  {
    id: "st-2",
    workspace_id: "ws-1",
    project_id: "proj-1",
    name: "To Do",
    category: "unstarted",
    color: "#1e4d79",
    position: 1,
    created_at: "2026-11-01T00:00:00Z",
    updated_at: "2026-11-01T00:00:00Z",
  },
  {
    id: "st-3",
    workspace_id: "ws-1",
    project_id: "proj-1",
    name: "In Progress",
    category: "started",
    color: "#775300",
    position: 2,
    created_at: "2026-11-01T00:00:00Z",
    updated_at: "2026-11-01T00:00:00Z",
  },
  {
    id: "st-4",
    workspace_id: "ws-1",
    project_id: "proj-1",
    name: "Done",
    category: "completed",
    color: "#26633b",
    position: 3,
    created_at: "2026-11-01T00:00:00Z",
    updated_at: "2026-11-01T00:00:00Z",
  },
];

test("renders list of statuses with category badges and controls", () => {
  render(StatusManagerModal, {
    props: {
      workspace: mockWorkspace,
      project: mockProject,
      statuses: mockStatuses,
      onclose: vi.fn(),
    },
  });

  expect(screen.getByText("Workflow statuses & transitions")).toBeTruthy();
  expect(screen.getAllByText("Backlog").length).toBeGreaterThanOrEqual(1);
  expect(screen.getByText("To Do")).toBeTruthy();
  expect(screen.getByText("In Progress")).toBeTruthy();
  expect(screen.getAllByText("Done").length).toBeGreaterThanOrEqual(1);
  expect(screen.getByText("4 statuses configured")).toBeTruthy();
});

test("allows creating a new custom status", async () => {
  document.cookie = "trestle-csrf=test-csrf; path=/";

  const fetchMock = vi.fn().mockImplementation((url, opts) => {
    if (opts?.method === "POST") {
      return Promise.resolve(
        new Response(
          JSON.stringify({
            id: "st-new",
            name: "In Review",
            category: "started",
            color: "#6366f1",
            position: 4,
          }),
          { status: 201, headers: { "Content-Type": "application/json" } },
        ),
      );
    }
    // reload statuses
    return Promise.resolve(
      new Response(
        JSON.stringify({
          statuses: [
            ...mockStatuses,
            {
              id: "st-new",
              name: "In Review",
              category: "started",
              color: "#6366f1",
              position: 4,
              created_at: "2026-11-01T00:00:00Z",
              updated_at: "2026-11-01T00:00:00Z",
            },
          ],
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
  });
  vi.stubGlobal("fetch", fetchMock);

  render(StatusManagerModal, {
    props: {
      workspace: mockWorkspace,
      project: mockProject,
      statuses: mockStatuses,
      onclose: vi.fn(),
    },
  });

  const addBtn = screen.getByRole("button", { name: /Add custom status/i });
  await fireEvent.click(addBtn);

  const nameInput = screen.getByLabelText(/Status name/i);
  await fireEvent.input(nameInput, { target: { value: "In Review" } });

  const saveBtn = screen.getByRole("button", { name: /Save status/i });
  await fireEvent.click(saveBtn);

  await waitFor(() => {
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/workspaces/ws-1/projects/proj-1/statuses",
      expect.objectContaining({
        method: "POST",
      }),
    );
  });
});

test("prompts for safe task remap target when deleting a status", async () => {
  document.cookie = "trestle-csrf=test-csrf; path=/";

  render(StatusManagerModal, {
    props: {
      workspace: mockWorkspace,
      project: mockProject,
      statuses: mockStatuses,
      onclose: vi.fn(),
    },
  });

  const deleteBtn = screen.getByRole("button", { name: /Delete In Progress/i });
  await fireEvent.click(deleteBtn);

  expect(screen.getByText('Delete "In Progress" status')).toBeTruthy();
  expect(screen.getByLabelText(/Reassign active tasks to:/i)).toBeTruthy();
  expect(
    screen.getByRole("button", { name: /Reassign & Delete/i }),
  ).toBeTruthy();
});
