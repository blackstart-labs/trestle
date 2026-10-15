import { fireEvent, render, screen, waitFor } from "@testing-library/svelte";
import { expect, test, vi } from "vitest";
import App from "./App.svelte";

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
