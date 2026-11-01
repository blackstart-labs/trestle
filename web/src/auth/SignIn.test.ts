import { fireEvent, render, screen, waitFor } from "@testing-library/svelte";
import { expect, test, vi } from "vitest";
import SignIn from "./SignIn.svelte";

test("renders accessible sign-in form with labels and submit button", () => {
  render(SignIn, { onsuccess: vi.fn() });

  expect(screen.getByLabelText("Username")).toBeTruthy();
  expect(screen.getByLabelText("Password")).toBeTruthy();
  expect(screen.getByRole("button", { name: /sign in/i })).toBeTruthy();
});

test("submits credentials to /api/v1/session and invokes onsuccess on 200", async () => {
  const onsuccess = vi.fn();
  const mockUser = {
    id: "123e4567-e89b-12d3-a456-426614174000",
    username: "alice",
    display_name: "Alice Smith",
    locale: "en",
    timezone: "UTC",
  };

  const fetchMock = vi.fn().mockResolvedValue(
    new Response(JSON.stringify(mockUser), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }),
  );
  vi.stubGlobal("fetch", fetchMock);

  render(SignIn, { onsuccess });

  await fireEvent.input(screen.getByLabelText("Username"), {
    target: { value: "alice" },
  });
  await fireEvent.input(screen.getByLabelText("Password"), {
    target: { value: "secret123" },
  });

  await fireEvent.submit(screen.getByRole("button", { name: /sign in/i }));

  await waitFor(() => expect(onsuccess).toHaveBeenCalledWith(mockUser));
  expect(fetchMock).toHaveBeenCalledWith(
    "/api/v1/session",
    expect.objectContaining({
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ username: "alice", password: "secret123" }),
    }),
  );
});

test("displays error message on 401 and zeroes password in state", async () => {
  const onsuccess = vi.fn();
  const fetchMock = vi.fn().mockResolvedValue(
    new Response(JSON.stringify({ error: { code: "unauthorized" } }), {
      status: 401,
      headers: { "Content-Type": "application/json" },
    }),
  );
  vi.stubGlobal("fetch", fetchMock);

  render(SignIn, { onsuccess });

  const passwordInput = screen.getByLabelText("Password") as HTMLInputElement;

  await fireEvent.input(screen.getByLabelText("Username"), {
    target: { value: "alice" },
  });
  await fireEvent.input(passwordInput, {
    target: { value: "wrongpassword" },
  });

  await fireEvent.submit(screen.getByRole("button", { name: /sign in/i }));

  await waitFor(() =>
    expect(screen.getByRole("alert").textContent).toContain(
      "Invalid username or password",
    ),
  );
  expect(onsuccess).not.toHaveBeenCalled();
  expect(passwordInput.value).toBe("");
});

test("displays rate limit message on 429 response", async () => {
  const onsuccess = vi.fn();
  const fetchMock = vi.fn().mockResolvedValue(
    new Response(JSON.stringify({ error: { code: "rate_limited" } }), {
      status: 429,
      headers: { "Content-Type": "application/json" },
    }),
  );
  vi.stubGlobal("fetch", fetchMock);

  render(SignIn, { onsuccess });

  await fireEvent.input(screen.getByLabelText("Username"), {
    target: { value: "badactor" },
  });
  await fireEvent.input(screen.getByLabelText("Password"), {
    target: { value: "pass" },
  });

  await fireEvent.submit(screen.getByRole("button", { name: /sign in/i }));

  await waitFor(() =>
    expect(screen.getByRole("alert").textContent).toContain(
      "Too many attempts",
    ),
  );
});
