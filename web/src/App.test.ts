import { fireEvent, render, screen, waitFor } from "@testing-library/svelte";
import { beforeEach, expect, test, vi } from "vitest";
import App from "./App.svelte";

beforeEach(() => {
  localStorage.clear();
  document.documentElement.removeAttribute("data-theme");
});

test("appearance applies and saves a local override through its labelled control", async () => {
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue(new Response("", { status: 503 })),
  );
  render(App);
  await fireEvent.change(screen.getByLabelText("Theme"), {
    target: { value: "dark" },
  });
  expect(document.documentElement.dataset.theme).toBe("dark");
  expect(localStorage.getItem("trestle.theme")).toBe("dark");
  expect(screen.getByText("Theme saved on this browser.")).toBeTruthy();
});

test.each([
  ["dark", "dark"],
  ["light", "light"],
  ["system", "system"],
  ["untrusted value", "system"],
])("saved theme %s restores safely as %s", async (stored, expected) => {
  localStorage.setItem("trestle.theme", stored);
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue(new Response("", { status: 503 })),
  );
  render(App);
  await waitFor(() =>
    expect(document.documentElement.dataset.theme).toBe(expected),
  );
  expect(screen.getByLabelText("Theme")).toHaveProperty("value", expected);
  expect(document.body.textContent).not.toContain("untrusted value");
});

test("blocked storage reads preserve a usable system-themed shell", async () => {
  vi.spyOn(Storage.prototype, "getItem").mockImplementation(() => {
    throw new Error("private storage details");
  });
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue(new Response("", { status: 503 })),
  );
  render(App);
  expect(screen.getByRole("alert").textContent).toContain(
    "Theme storage is unavailable",
  );
  expect(document.documentElement.dataset.theme).toBe("system");
  await waitFor(() =>
    expect(screen.getByRole("status").textContent).toContain("unavailable"),
  );
  expect(document.body.textContent).not.toContain("private storage details");
});

test("failed preference writes apply temporarily and recover on another change", async () => {
  const write = vi
    .spyOn(Storage.prototype, "setItem")
    .mockImplementation(() => {
      throw new Error("private quota details");
    });
  vi.stubGlobal(
    "fetch",
    vi.fn().mockResolvedValue(new Response("", { status: 503 })),
  );
  render(App);
  await fireEvent.change(screen.getByLabelText("Theme"), {
    target: { value: "dark" },
  });
  expect(document.documentElement.dataset.theme).toBe("dark");
  expect(screen.getByRole("alert").textContent).toMatch(
    /may not\s+survive a reload/,
  );
  expect(document.body.textContent).not.toContain("private quota details");
  write.mockRestore();
  await fireEvent.change(screen.getByLabelText("Theme"), {
    target: { value: "light" },
  });
  expect(screen.queryByRole("alert")).toBeNull();
  expect(localStorage.getItem("trestle.theme")).toBe("light");
  expect(screen.getByText("Theme saved on this browser.")).toBeTruthy();
});

test("checks actual database readiness on opening the foundation page", async () => {
  const fetch = vi.fn().mockResolvedValue(
    new Response('{"status":"ok"}', {
      headers: { "Content-Type": "application/json" },
    }),
  );
  vi.stubGlobal("fetch", fetch);
  render(App);
  await waitFor(() =>
    expect(screen.getByRole("status").textContent).toContain(
      "Database connection ready",
    ),
  );
  expect(fetch).toHaveBeenCalledWith(
    "/health/ready",
    expect.objectContaining({ cache: "no-store" }),
  );
});

test("an unavailable database can be checked again and recover", async () => {
  const fetch = vi
    .fn()
    .mockResolvedValueOnce(new Response("", { status: 503 }))
    .mockResolvedValueOnce(
      new Response('{"status":"ok"}', {
        headers: { "Content-Type": "application/json" },
      }),
    );
  vi.stubGlobal("fetch", fetch);
  render(App);
  await waitFor(() =>
    expect(screen.getByRole("status").textContent).toContain(
      "Database connection unavailable",
    ),
  );
  await fireEvent.click(screen.getByRole("button", { name: "Check again" }));
  await waitFor(() =>
    expect(screen.getByRole("status").textContent).toContain(
      "Database connection ready",
    ),
  );
  expect(fetch).toHaveBeenCalledTimes(2);
});

test.each([
  () => Promise.reject(new Error("secret network details")),
  () =>
    Promise.resolve(
      new Response('{"status":"unexpected"}', {
        headers: { "Content-Type": "application/json" },
      }),
    ),
  () =>
    Promise.resolve(
      new Response("not JSON", {
        headers: { "Content-Type": "application/json" },
      }),
    ),
  () =>
    Promise.resolve(
      new Response('{"status":"ok"}', {
        status: 500,
        headers: { "Content-Type": "application/json" },
      }),
    ),
  () =>
    Promise.resolve(
      new Response('{"status":"ok"}', {
        headers: { "Content-Type": "text/html" },
      }),
    ),
])("network/protocol failures show a safe retry message", async (reply) => {
  vi.stubGlobal("fetch", vi.fn().mockImplementation(reply));
  render(App);
  await waitFor(() =>
    expect(screen.getByRole("status").textContent).toContain(
      "Could not check readiness. Try again.",
    ),
  );
  expect(document.body.textContent).not.toContain("secret network details");
  expect((screen.getByRole("button") as HTMLButtonElement).disabled).toBe(
    false,
  );
});

test("a pending check prevents duplicates and is aborted after five seconds", async () => {
  vi.useFakeTimers();
  let signal: AbortSignal | undefined;
  vi.stubGlobal(
    "fetch",
    vi.fn(
      (_url, options) =>
        new Promise((_resolve, reject) => {
          signal = options.signal;
          signal!.addEventListener("abort", () => reject(new Error("aborted")));
        }),
    ),
  );
  try {
    render(App);
    expect((screen.getByRole("button") as HTMLButtonElement).disabled).toBe(
      true,
    );
    await vi.advanceTimersByTimeAsync(5000);
    expect(signal?.aborted).toBe(true);
    expect(screen.getByRole("status").textContent).toContain(
      "Could not check readiness",
    );
  } finally {
    vi.useRealTimers();
  }
});
