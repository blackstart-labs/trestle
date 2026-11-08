import { fireEvent, render, screen, waitFor } from "@testing-library/svelte";
import { expect, test, vi } from "vitest";
import Integrations from "./Integrations.svelte";

const workspace = {
  id: "ws",
  name: "Engineering",
  slug: "engineering",
  role: "owner",
};
const projects = [
  { id: "alpha", name: "Alpha", key: "ALPHA", archived: false },
  { id: "beta", name: "Beta", key: "BETA", archived: false },
];
function mockAPI() {
  let saved: object[] = [];
  const fetch = vi.fn(async (url: string, init?: RequestInit) => {
    if (url.endsWith("/projects")) return Response.json({ projects });
    if (init?.method === "PUT") {
      saved = [
        {
          provider: url.split("/").at(-1),
          chat_id: "",
          last_attempt_at: null,
          last_error: "",
        },
      ];
      return Response.json(saved[0]);
    }
    if (init?.method === "DELETE") {
      saved = [];
      return new Response(null, { status: 204 });
    }
    if (url.endsWith("/test")) return Response.json({ sent: true });
    return Response.json({ integrations: saved });
  });
  vi.stubGlobal("fetch", fetch);
  return fetch;
}

test("saves a project connection, tests it, and confirms disconnect", async () => {
  const fetch = mockAPI();
  document.cookie = "trestle-csrf=csrf-token";
  render(Integrations, { workspace, signedIn: true, initialProjectId: "beta" });
  await fireEvent.click(
    await screen.findByRole("button", { name: "Connect Slack" }),
  );
  await fireEvent.input(screen.getByLabelText("Webhook URL"), {
    target: { value: "https://hooks.slack.com/services/T/B/SECRET" },
  });
  await fireEvent.click(
    screen.getByRole("button", { name: "Save integration" }),
  );
  await screen.findByRole("button", { name: "Replace Slack" });
  expect(fetch).toHaveBeenCalledWith(
    "/api/v1/workspaces/ws/projects/beta/integrations/slack",
    expect.objectContaining({
      method: "PUT",
      headers: expect.objectContaining({ "X-CSRF-Token": "csrf-token" }),
    }),
  );
  await waitFor(() =>
    expect(
      screen.queryByDisplayValue("https://hooks.slack.com/services/T/B/SECRET"),
    ).toBeNull(),
  );
  await fireEvent.click(screen.getByRole("button", { name: "Test Slack" }));
  await screen.findByText("Slack test message sent.");
  await waitFor(() =>
    expect(
      screen.getByRole("button", { name: "Disconnect Slack" }),
    ).toHaveProperty("disabled", false),
  );
  await fireEvent.click(
    screen.getByRole("button", { name: "Disconnect Slack" }),
  );
  expect(
    fetch.mock.calls.filter(([, init]) => init?.method === "DELETE"),
  ).toHaveLength(0);
  await fireEvent.click(
    screen.getByRole("button", { name: "Disconnect integration" }),
  );
  await screen.findByRole("button", { name: "Connect Slack" });
});

test("save errors preserve credentials and dirty close requires confirmation", async () => {
  const fetch = mockAPI();
  const original = fetch.getMockImplementation()!;
  fetch.mockImplementation(async (url, init) =>
    init?.method === "PUT"
      ? Response.json(
          { error: { message: "Check the webhook URL" } },
          { status: 400 },
        )
      : original(url, init),
  );
  const confirm = vi.spyOn(window, "confirm").mockReturnValue(false);
  render(Integrations, { workspace, signedIn: true });
  await fireEvent.click(
    await screen.findByRole("button", { name: "Connect Discord" }),
  );
  await fireEvent.input(screen.getByLabelText("Webhook URL"), {
    target: { value: "invalid" },
  });
  await fireEvent.click(
    screen.getByRole("button", { name: "Save integration" }),
  );
  await screen.findByRole("alert");
  expect(screen.getByLabelText("Webhook URL")).toHaveProperty(
    "value",
    "invalid",
  );
  await fireEvent.click(screen.getByRole("button", { name: "Cancel" }));
  expect(confirm).toHaveBeenCalled();
  expect(screen.getByRole("dialog")).toBeTruthy();
});

test("members cannot fetch or configure integrations", () => {
  const fetch = mockAPI();
  render(Integrations, {
    workspace: { ...workspace, role: "member" },
    signedIn: true,
  });
  expect(screen.getByText(/Only Workspace owners and admins/)).toBeTruthy();
  expect(fetch).not.toHaveBeenCalled();
});

test("loading failure offers retry and an empty Workspace offers the next action", async () => {
  const fetch = mockAPI();
  fetch.mockRejectedValueOnce(new Error("offline"));
  render(Integrations, { workspace, signedIn: true });
  await fireEvent.click(
    await screen.findByRole("button", { name: "Retry integrations" }),
  );
  await screen.findByRole("button", { name: "Connect Telegram" });
});

test("saves a Telegram connection with optional topic ID", async () => {
  const fetch = mockAPI();
  document.cookie = "trestle-csrf=csrf-token";
  render(Integrations, { workspace, signedIn: true, initialProjectId: "beta" });
  await fireEvent.click(
    await screen.findByRole("button", { name: "Connect Telegram" }),
  );
  await fireEvent.input(screen.getByLabelText("Bot token"), {
    target: { value: "123456:abcdefghijklmnopqrstuvwxyz_123456789" },
  });
  await fireEvent.input(screen.getByLabelText("Chat ID"), {
    target: { value: "-100123456" },
  });
  await fireEvent.input(screen.getByLabelText(/Topic ID/), {
    target: { value: "42" },
  });
  await fireEvent.click(
    screen.getByRole("button", { name: "Save integration" }),
  );
  await screen.findByRole("button", { name: "Replace Telegram" });
  expect(fetch).toHaveBeenCalledWith(
    "/api/v1/workspaces/ws/projects/beta/integrations/telegram",
    expect.objectContaining({
      method: "PUT",
      body: JSON.stringify({
        secret: "123456:abcdefghijklmnopqrstuvwxyz_123456789",
        chat_id: "-100123456",
        topic_id: "42",
      }),
    }),
  );
});
