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
      screen.getByText(/not currently a member of any workspaces/i),
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
