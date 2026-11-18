import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/svelte";
import BoardsView from "./BoardsView.svelte";

describe("BoardsView", () => {
  const mockWorkspace = {
    id: "ws-1",
    name: "Engineering",
    role: "owner",
  };

  const sampleBoards = [
    {
      id: "board-1",
      workspace_id: "ws-1",
      title: "Architecture Diagram",
      kind: "diagram",
      version: 1,
      archived: false,
      updated_at: new Date(Date.now() - 3600000).toISOString(),
    },
    {
      id: "board-2",
      workspace_id: "ws-1",
      title: "Sprint Whiteboard",
      kind: "whiteboard",
      version: 1,
      archived: false,
      updated_at: new Date(Date.now() - 7200000).toISOString(),
    },
  ];

  it("renders in grid view by default and allows toggling to list view", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn(
        async () =>
          new Response(
            JSON.stringify({ boards: sampleBoards, has_more: false }),
            { headers: { "Content-Type": "application/json" } },
          ),
      ),
    );

    const { container } = render(BoardsView, {
      workspace: mockWorkspace,
      mode: "all",
    });

    // Wait for boards to load
    await screen.findByText("Architecture Diagram");

    // Verify grid layout is active by default
    const gridBtn = screen.getByRole("button", { name: "Grid" });
    const listBtn = screen.getByRole("button", { name: "List" });
    expect(gridBtn.getAttribute("aria-pressed")).toBe("true");
    expect(listBtn.getAttribute("aria-pressed")).toBe("false");
    expect(container.querySelector(".board-grid")).toBeTruthy();
    expect(container.querySelector(".board-list")).toBeNull();

    // Verify boards are rendered with Open and Delete buttons
    expect(screen.getByText("Sprint Whiteboard")).toBeTruthy();
    expect(screen.getAllByRole("button", { name: "Open" })).toHaveLength(2);
    expect(
      screen.getByRole("button", { name: "Delete Architecture Diagram" }),
    ).toBeTruthy();
    expect(
      screen.getByRole("button", { name: "Delete Sprint Whiteboard" }),
    ).toBeTruthy();

    // Verify modified time is shown
    expect(screen.getAllByText(/Modified/i).length).toBeGreaterThan(0);

    // Switch to list view
    await fireEvent.click(listBtn);
    expect(gridBtn.getAttribute("aria-pressed")).toBe("false");
    expect(listBtn.getAttribute("aria-pressed")).toBe("true");
    expect(container.querySelector(".board-list")).toBeTruthy();
    expect(container.querySelector(".board-grid")).toBeNull();
  });
});
