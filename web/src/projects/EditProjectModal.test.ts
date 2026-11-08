import { fireEvent, render, screen, waitFor } from "@testing-library/svelte";
import { expect, test, vi } from "vitest";
import EditProjectModal from "./EditProjectModal.svelte";
import type { Project } from "./ProjectsView.svelte";

const project: Project = {
  id: "project",
  workspace_id: "ws",
  name: "Core",
  slug: "core",
  key: "CORE",
  description: "Original",
  task_counter: 4,
  archived: false,
  created_at: "2026-11-08T00:00:00Z",
  updated_at: "2026-11-08T00:00:00Z",
};

function setup() {
  Object.defineProperty(document, "cookie", {
    configurable: true,
    writable: true,
    value: "trestle-csrf=csrf",
  });
  const onupdated = vi.fn();
  const onclose = vi.fn();
  render(EditProjectModal, {
    project,
    workspace: { id: "ws", role: "owner" },
    onupdated,
    onclose,
  });
  return { onupdated, onclose };
}

test("only changed metadata is saved and the project key is read-only", async () => {
  const fetchMock = vi
    .fn()
    .mockResolvedValue(
      new Response(JSON.stringify({ ...project, name: "Core Platform" })),
    );
  vi.stubGlobal("fetch", fetchMock);
  const { onupdated } = setup();
  expect(screen.getByLabelText("Project key")).toHaveProperty("readOnly", true);
  expect(screen.getByRole("button", { name: "Save project" })).toHaveProperty(
    "disabled",
    true,
  );
  await fireEvent.input(screen.getByLabelText("Project name"), {
    target: { value: " Core Platform " },
  });
  await fireEvent.click(screen.getByRole("button", { name: "Save project" }));
  await waitFor(() => expect(onupdated).toHaveBeenCalled());
  expect(fetchMock).toHaveBeenCalledWith(
    "/api/v1/workspaces/ws/projects/project",
    expect.objectContaining({
      method: "PATCH",
      body: JSON.stringify({ name: "Core Platform" }),
      headers: expect.objectContaining({ "X-CSRF-Token": "csrf" }),
    }),
  );
});

test("failed saves retain the draft, block duplicate submits, and can retry", async () => {
  let finish!: (response: Response) => void;
  const fetchMock = vi
    .fn()
    .mockImplementationOnce(
      () =>
        new Promise<Response>((resolve) => {
          finish = resolve;
        }),
    )
    .mockResolvedValue(
      new Response(JSON.stringify({ ...project, slug: "platform" })),
    );
  vi.stubGlobal("fetch", fetchMock);
  const { onupdated } = setup();
  await fireEvent.input(screen.getByLabelText("Slug"), {
    target: { value: "platform" },
  });
  await fireEvent.click(screen.getByRole("button", { name: "Save project" }));
  expect(screen.getByLabelText("Slug")).toHaveProperty("disabled", true);
  await fireEvent.submit(document.querySelector("form")!);
  expect(fetchMock).toHaveBeenCalledTimes(1);
  finish(
    new Response(
      JSON.stringify({ error: { message: "This slug is already used." } }),
      { status: 409 },
    ),
  );
  await waitFor(() =>
    expect(screen.getByRole("alert").textContent).toContain(
      "This slug is already used.",
    ),
  );
  expect(screen.getByLabelText("Slug")).toHaveProperty("value", "platform");
  expect(onupdated).not.toHaveBeenCalled();
  await fireEvent.click(screen.getByRole("button", { name: "Save project" }));
  await waitFor(() => expect(onupdated).toHaveBeenCalledTimes(1));
});

test("closing a dirty edit requires confirmation", async () => {
  const { onclose } = setup();
  const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
  await fireEvent.input(screen.getByLabelText("Description"), {
    target: { value: "Unsaved" },
  });
  await fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
  expect(confirm).toHaveBeenCalled();
  expect(onclose).not.toHaveBeenCalled();
  confirm.mockReturnValue(true);
  await fireEvent.keyDown(screen.getByRole("dialog"), { key: "Escape" });
  expect(onclose).toHaveBeenCalledTimes(1);
});

test("invalid slugs are explained without writing", async () => {
  const fetchMock = vi.fn();
  vi.stubGlobal("fetch", fetchMock);
  setup();
  await fireEvent.input(screen.getByLabelText("Slug"), {
    target: { value: "bad slug" },
  });
  await fireEvent.click(screen.getByRole("button", { name: "Save project" }));
  expect(screen.getByRole("alert").textContent).toContain(
    "Start and end with a letter or number.",
  );
  expect(fetchMock).not.toHaveBeenCalled();
});
