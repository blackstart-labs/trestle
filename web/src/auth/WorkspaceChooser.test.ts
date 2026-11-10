import { fireEvent, render, screen, waitFor } from "@testing-library/svelte";
import { expect, test, vi } from "vitest";
import WorkspaceChooser from "./WorkspaceChooser.svelte";

test("loads and displays authorized workspaces", async () => {
  const workspaces = [
    {
      id: "ws-1",
      name: "Engineering",
      slug: "engineering",
      role: "owner",
    },
    {
      id: "ws-2",
      name: "Design",
      slug: "design",
      role: "member",
    },
  ];

  const fetchMock = vi.fn().mockResolvedValue(
    new Response(JSON.stringify({ workspaces }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }),
  );
  vi.stubGlobal("fetch", fetchMock);

  const onselect = vi.fn();
  render(WorkspaceChooser, { onselect });

  await waitFor(() => {
    expect(screen.getByText("Engineering")).toBeTruthy();
    expect(screen.getByText("/engineering")).toBeTruthy();
    expect(screen.getByText("Design")).toBeTruthy();
    expect(screen.getByText("/design")).toBeTruthy();
  });

  const buttons = screen.getAllByRole("button", { name: /open/i });
  expect(buttons.length).toBe(2);

  await fireEvent.click(buttons[0]);
  expect(onselect).toHaveBeenCalledWith(workspaces[0]);
});

test("displays empty state when user has no workspaces", async () => {
  const fetchMock = vi.fn().mockResolvedValue(
    new Response(JSON.stringify({ workspaces: [] }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }),
  );
  vi.stubGlobal("fetch", fetchMock);

  render(WorkspaceChooser, {});

  await waitFor(() => {
    expect(
      screen.getByText(/create a workspace, or ask an owner/i),
    ).toBeTruthy();
  });
});

test("displays error and allows retry when fetch fails", async () => {
  const fetchMock = vi
    .fn()
    .mockRejectedValueOnce(new Error("Network error"))
    .mockResolvedValueOnce(
      new Response(
        JSON.stringify({
          workspaces: [
            { id: "ws-1", name: "Recovered", slug: "recovered", role: "admin" },
          ],
        }),
        { status: 200, headers: { "Content-Type": "application/json" } },
      ),
    );
  vi.stubGlobal("fetch", fetchMock);

  render(WorkspaceChooser, {});

  await waitFor(() => {
    expect(screen.getByRole("alert").textContent).toContain(
      "Could not load workspaces",
    );
  });

  const retryButton = screen.getByRole("button", { name: /try again/i });
  await fireEvent.click(retryButton);

  await waitFor(() => {
    expect(screen.getByText("Recovered")).toBeTruthy();
  });
});

test("on arrival, the only workspace opens without a click", async () => {
  const only = { id: "ws-9", name: "Solo", slug: "solo", role: "owner" };
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue(
      new Response(JSON.stringify({ workspaces: [only] }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    ),
  );
  const onselect = vi.fn();
  render(WorkspaceChooser, { onselect, autoselect: true });
  await waitFor(() => expect(onselect).toHaveBeenCalledWith(only));
});

test("workspace creation preserves a rejected draft and selects the workspace after retry", async () => {
  document.cookie = "trestle-csrf=test-token";
  const ws = {
    id: "new",
    name: "New workspace",
    slug: "new-workspace",
    role: "owner",
  };
  let reject = true;
  const fetchMock = vi.fn(async (_url, init) => {
    if (init?.method === "POST")
      return new Response(
        JSON.stringify(
          reject ? { error: { message: "Choose another slug." } } : ws,
        ),
        { status: reject ? 409 : 201 },
      );
    return new Response(JSON.stringify({ workspaces: [] }));
  });
  vi.stubGlobal("fetch", fetchMock);
  const onselect = vi.fn();
  render(WorkspaceChooser, { onselect });
  await fireEvent.click(screen.getByRole("button", { name: "New workspace" }));
  await fireEvent.input(screen.getByLabelText("Workspace name"), {
    target: { value: ws.name },
  });
  await fireEvent.input(screen.getByLabelText("Workspace slug"), {
    target: { value: ws.slug },
  });
  await fireEvent.click(
    screen.getByRole("button", { name: "Create workspace" }),
  );
  await waitFor(() =>
    expect(screen.getByRole("alert").textContent).toContain(
      "Choose another slug",
    ),
  );
  expect(screen.getByLabelText("Workspace name")).toHaveProperty(
    "value",
    ws.name,
  );
  reject = false;
  await fireEvent.click(
    screen.getByRole("button", { name: "Create workspace" }),
  );
  await waitFor(() => expect(onselect).toHaveBeenCalledWith(ws));
  expect(fetchMock).toHaveBeenCalledWith(
    "/api/v1/workspaces",
    expect.objectContaining({
      headers: expect.objectContaining({ "X-CSRF-Token": "test-token" }),
    }),
  );
});
