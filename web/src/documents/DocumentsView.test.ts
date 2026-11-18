import { render, screen, fireEvent, waitFor } from "@testing-library/svelte";
import { afterEach, expect, it, vi } from "vitest";
import DocumentsView from "./DocumentsView.svelte";
const workspace = { id: "ws", name: "Engineering", role: "owner" };
afterEach(() => vi.restoreAllMocks());
it("preserves a failed draft, blocks navigation and sends Markdown unchanged", async () => {
  let body: Record<string, unknown> = {};
  vi.stubGlobal(
    "fetch",
    vi.fn(async (_url: string, options?: RequestInit) => {
      if (options?.method === "POST") {
        body = JSON.parse(String(options.body));
        return new Response(
          JSON.stringify({
            error: { message: "Retry after connection recovery" },
          }),
          { status: 503 },
        );
      }
      return new Response(JSON.stringify({ documents: [], has_more: false }));
    }),
  );
  render(DocumentsView, { workspace });
  await screen.findByText("Build your knowledge base");
  await fireEvent.click(screen.getByRole("button", { name: "New document" }));
  await fireEvent.input(screen.getByLabelText("Title", { exact: true }), {
    target: { value: "Research" },
  });
  await fireEvent.input(
    screen.getByLabelText("Document content", { exact: true }),
    { target: { value: "  # Draft\n\nKeep whitespace  " } },
  );
  await fireEvent.click(screen.getByRole("button", { name: "Save document" }));
  await screen.findByText("Retry after connection recovery");
  expect(body.content).toBe("  # Draft\n\nKeep whitespace  ");
  expect(
    (screen.getByLabelText("Title", { exact: true }) as HTMLInputElement).value,
  ).toBe("Research");
  vi.spyOn(window, "confirm").mockReturnValue(false);
  const event = new Event("trestle:before-navigate", { cancelable: true });
  expect(window.dispatchEvent(event)).toBe(false);
});
it("offers guests read and export without create controls", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn(
      async () =>
        new Response(JSON.stringify({ documents: [], has_more: false })),
    ),
  );
  render(DocumentsView, { workspace: { ...workspace, role: "guest" } });
  await waitFor(() =>
    expect(screen.queryByText("Loading documents…")).toBeNull(),
  );
  expect(screen.queryByRole("button", { name: "New document" })).toBeNull();
  expect(screen.getByText(/Guests can read and export/)).toBeTruthy();
});
