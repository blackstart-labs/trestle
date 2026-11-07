import { fireEvent, render, screen } from "@testing-library/svelte";
import { expect, test } from "vitest";
import MarkdownEditor from "./MarkdownEditor.svelte";

test("formats a selection and previews the unsaved draft without losing source", async () => {
  render(MarkdownEditor, { id: "description", value: "Project structure" });
  const input = screen.getByLabelText("Description") as HTMLTextAreaElement;
  input.setSelectionRange(0, 7);
  await fireEvent.click(screen.getByRole("button", { name: "Bold" }));
  expect(input.value).toBe("**Project** structure");
  expect(document.activeElement).toBe(input);
  await fireEvent.click(screen.getByRole("button", { name: "Preview" }));
  const preview = screen.getByRole("region", { name: "Description preview" });
  expect(preview.querySelector("strong")?.textContent).toBe("Project");
  await fireEvent.click(screen.getByRole("button", { name: "Write" }));
  expect(
    (screen.getByLabelText("Description") as HTMLTextAreaElement).value,
  ).toBe("**Project** structure");
});

test("empty previews and read-only descriptions explain their state", async () => {
  const view = render(MarkdownEditor, { id: "description" });
  await fireEvent.click(screen.getByRole("button", { name: "Preview" }));
  expect(
    screen.getByText("Nothing to preview. Write a description first."),
  ).toBeTruthy();
  await view.rerender({
    id: "description",
    readonly: true,
    value: "**Read only**",
  });
  expect(screen.queryByRole("button")).toBeNull();
  expect(screen.getByRole("region").querySelector("strong")?.textContent).toBe(
    "Read only",
  );
});
