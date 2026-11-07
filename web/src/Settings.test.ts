import { render, screen } from "@testing-library/svelte";
import { expect, test, vi } from "vitest";
import Settings from "./Settings.svelte";

test("Integration has a dedicated Settings view and its active navigation", () => {
  render(Settings, { section: "integration" });
  expect(screen.getByRole("heading", { name: "Integration" })).toBeTruthy();
  expect(screen.queryByRole("heading", { name: "Appearance" })).toBeNull();
  expect(
    screen
      .getByRole("link", { name: "Integration" })
      .getAttribute("aria-current"),
  ).toBe("page");
  expect(screen.getByText(/Sign in/)).toBeTruthy();
});

test("default Settings retains Appearance and Instance status", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue(new Response("", { status: 503 })),
  );
  render(Settings);
  expect(screen.getByRole("heading", { name: "Appearance" })).toBeTruthy();
  expect(screen.getByRole("heading", { name: "Instance status" })).toBeTruthy();
  expect(screen.queryByRole("heading", { name: "Integration" })).toBeNull();
});
