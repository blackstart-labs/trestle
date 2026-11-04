import { fireEvent, render, screen, waitFor } from "@testing-library/svelte";
import { expect, test, vi } from "vitest";
import ProjectsView from "./ProjectsView.svelte";

test("loads and displays active projects", async () => {
  const projects = [
    {
      id: "proj-1",
      workspace_id: "ws-1",
      name: "Core Platform",
      slug: "core-platform",
      key: "CORE",
      description: "Platform infrastructure",
      task_counter: 5,
      archived: false,
      created_at: "2026-11-01T00:00:00Z",
      updated_at: "2026-11-01T00:00:00Z",
    },
    {
      id: "proj-2",
      workspace_id: "ws-1",
      name: "Web Application",
      slug: "web-app",
      key: "WEB",
      description: "Client frontend",
      task_counter: 12,
      archived: false,
      created_at: "2026-11-01T00:00:00Z",
      updated_at: "2026-11-01T00:00:00Z",
    },
  ];

  const fetchMock = vi.fn().mockResolvedValue(
    new Response(JSON.stringify({ projects }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }),
  );
  vi.stubGlobal("fetch", fetchMock);

  render(ProjectsView, {
    workspace: {
      id: "ws-1",
      name: "Acme",
      slug: "acme",
      role: "owner",
    },
  });

  await waitFor(() => {
    expect(screen.getByText("Core Platform")).toBeTruthy();
    expect(screen.getByText("/core-platform")).toBeTruthy();
    expect(screen.getByText("CORE")).toBeTruthy();
    expect(screen.getByText("5 tasks created")).toBeTruthy();

    expect(screen.getByText("Web Application")).toBeTruthy();
    expect(screen.getByText("/web-app")).toBeTruthy();
    expect(screen.getByText("WEB")).toBeTruthy();
    expect(screen.getByText("12 tasks created")).toBeTruthy();
  });
});

test("shows empty state when no projects exist", async () => {
  const fetchMock = vi.fn().mockResolvedValue(
    new Response(JSON.stringify({ projects: [] }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }),
  );
  vi.stubGlobal("fetch", fetchMock);

  render(ProjectsView, {
    workspace: {
      id: "ws-1",
      name: "Acme",
      slug: "acme",
      role: "owner",
    },
  });

  await waitFor(() => {
    expect(screen.getByText(/no projects yet/i)).toBeTruthy();
  });
});

test("allows owner to create a new project", async () => {
  Object.defineProperty(document, "cookie", {
    writable: true,
    value: "trestle-csrf=test-csrf-token",
  });

  const projects = [
    {
      id: "proj-1",
      workspace_id: "ws-1",
      name: "Existing Project",
      slug: "existing-proj",
      key: "EXIST",
      description: "Existing",
      task_counter: 0,
      archived: false,
      created_at: "2026-11-01T00:00:00Z",
      updated_at: "2026-11-01T00:00:00Z",
    },
  ];

  const fetchMock = vi.fn().mockImplementation((url, init) => {
    if (init?.method === "POST") {
      return Promise.resolve(
        new Response(
          JSON.stringify({
            id: "proj-2",
            workspace_id: "ws-1",
            name: "Mobile App",
            slug: "mobile-app",
            key: "MOBI",
            description: "iOS and Android client",
            task_counter: 0,
            archived: false,
            created_at: "2026-11-02T00:00:00Z",
            updated_at: "2026-11-02T00:00:00Z",
          }),
          { status: 201, headers: { "Content-Type": "application/json" } },
        ),
      );
    }
    return Promise.resolve(
      new Response(JSON.stringify({ projects }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
  });
  vi.stubGlobal("fetch", fetchMock);

  render(ProjectsView, {
    workspace: {
      id: "ws-1",
      name: "Acme",
      slug: "acme",
      role: "owner",
    },
  });

  await waitFor(() => {
    expect(screen.getByText("Existing Project")).toBeTruthy();
  });

  // Open modal
  const newProjBtn = screen.getByRole("button", { name: /new project/i });
  await fireEvent.click(newProjBtn);

  await waitFor(() => {
    expect(screen.getByText("Create new project")).toBeTruthy();
  });

  const nameInput = screen.getByLabelText(/project name/i);
  await fireEvent.input(nameInput, { target: { value: "Mobile App" } });

  const submitBtn = screen.getByRole("button", { name: "Create project" });
  await fireEvent.click(submitBtn);

  await waitFor(() => {
    expect(screen.getByText(/created successfully/i)).toBeTruthy();
    expect(screen.getByText("Mobile App")).toBeTruthy();
    expect(screen.getByText("MOBI")).toBeTruthy();
  });
});
