import { fireEvent, render, screen, waitFor } from "@testing-library/svelte";
import { expect, test, vi } from "vitest";
import WorkspaceMembers from "./WorkspaceMembers.svelte";

test("loads and displays workspace members", async () => {
  const members = [
    {
      user_id: "user-1",
      username: "owner_user",
      display_name: "Owner User",
      role: "owner",
    },
    {
      user_id: "user-2",
      username: "alice",
      display_name: "Alice Cooper",
      role: "admin",
    },
  ];

  const fetchMock = vi.fn().mockResolvedValue(
    new Response(JSON.stringify({ members }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }),
  );
  vi.stubGlobal("fetch", fetchMock);

  render(WorkspaceMembers, {
    workspace: {
      id: "ws-1",
      name: "Engineering",
      slug: "engineering",
      role: "owner",
    },
    currentUserId: "user-1",
  });

  await waitFor(() => {
    expect(screen.getByText("Owner User")).toBeTruthy();
    expect(screen.getByText("@owner_user")).toBeTruthy();
    expect(screen.getByText("Alice Cooper")).toBeTruthy();
    expect(screen.getByText("@alice")).toBeTruthy();
  });
});

test("allows owner to add a new member", async () => {
  const members = [
    {
      user_id: "user-1",
      username: "owner_user",
      display_name: "Owner User",
      role: "owner",
    },
  ];

  const fetchMock = vi.fn().mockImplementation((url, init) => {
    if (init?.method === "POST") {
      return Promise.resolve(
        new Response(
          JSON.stringify({
            user_id: "user-2",
            username: "bob",
            display_name: "Bob Builder",
            role: "member",
          }),
          { status: 201, headers: { "Content-Type": "application/json" } },
        ),
      );
    }
    return Promise.resolve(
      new Response(JSON.stringify({ members }), {
        status: 200,
        headers: { "Content-Type": "application/json" },
      }),
    );
  });
  vi.stubGlobal("fetch", fetchMock);

  render(WorkspaceMembers, {
    workspace: {
      id: "ws-1",
      name: "Engineering",
      slug: "engineering",
      role: "owner",
    },
    currentUserId: "user-1",
  });

  await waitFor(() => {
    expect(screen.getByText("Owner User")).toBeTruthy();
  });

  const usernameInput = screen.getByLabelText(/username/i);
  await fireEvent.input(usernameInput, { target: { value: "bob" } });

  const submitBtn = screen.getByRole("button", { name: /add to workspace/i });
  await fireEvent.click(submitBtn);

  await waitFor(() => {
    expect(screen.getByText(/added @bob as member/i)).toBeTruthy();
    expect(screen.getByText("Bob Builder")).toBeTruthy();
  });
});

test("shows transfer ownership button for owner and opens modal", async () => {
  const members = [
    {
      user_id: "user-1",
      username: "owner_user",
      display_name: "Owner User",
      role: "owner",
    },
    {
      user_id: "user-2",
      username: "alice",
      display_name: "Alice Cooper",
      role: "admin",
    },
  ];

  const fetchMock = vi.fn().mockResolvedValue(
    new Response(JSON.stringify({ members }), {
      status: 200,
      headers: { "Content-Type": "application/json" },
    }),
  );
  vi.stubGlobal("fetch", fetchMock);

  render(WorkspaceMembers, {
    workspace: {
      id: "ws-1",
      name: "Engineering",
      slug: "engineering",
      role: "owner",
    },
    currentUserId: "user-1",
  });

  await waitFor(() => {
    expect(screen.getByText("Owner User")).toBeTruthy();
  });

  const transferBtn = screen.getByRole("button", {
    name: /transfer ownership/i,
  });
  await fireEvent.click(transferBtn);

  await waitFor(() => {
    expect(screen.getByLabelText(/new owner/i)).toBeTruthy();
    expect(screen.getByLabelText(/your password confirmation/i)).toBeTruthy();
  });
});
