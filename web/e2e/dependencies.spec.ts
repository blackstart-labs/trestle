import { expect, test, type Page } from "@playwright/test";

const workspace = {
  id: "ws",
  name: "Engineering",
  slug: "engineering",
  role: "member",
};
const project = {
  id: "proj",
  workspace_id: "ws",
  name: "Core platform",
  slug: "core",
  key: "CORE",
  description: "",
  task_counter: 3,
  archived: false,
  created_at: "2026-11-01T00:00:00Z",
  updated_at: "2026-11-01T00:00:00Z",
};
const task = {
  id: "task",
  workspace_id: "ws",
  project_id: "proj",
  project_key: "CORE",
  task_number: 1,
  human_key: "CORE-1",
  title: "Ship release",
  description: "",
  status: "started",
  priority: "none",
  due_date: null,
  labels: [],
  assignees: [],
  subtask_count: 0,
  spent_hours: 0,
  created_at: "2026-11-01T00:00:00Z",
  updated_at: "2026-11-01T00:00:00Z",
};
const blocker = {
  id: "web-task",
  workspace_id: "ws",
  project_id: "web",
  project_name: "Website",
  project_key: "WEB",
  task_number: 1,
  human_key: "WEB-1",
  title: "Build page",
  status: "started",
  status_name: "In progress",
  status_color: "#315b3b",
  created_at: "2026-11-01T00:00:00Z",
  updated_at: "2026-11-01T00:00:00Z",
};
const blocked = {
  id: "report-task",
  workspace_id: "ws",
  project_id: "proj",
  project_name: "Core platform",
  project_key: "CORE",
  task_number: 3,
  human_key: "CORE-3",
  title: "Write report",
  status: "unstarted",
  status_name: "Unstarted",
  status_color: "#4c544c",
  created_at: "2026-11-01T00:00:00Z",
  updated_at: "2026-11-01T00:00:00Z",
};

async function setup(page: Page, role = "member") {
  const writes: { method: string; path: string; body: any }[] = [];
  const links: { blocked_by: any[]; blocks: any[] } = {
    blocked_by: [],
    blocks: [],
  };
  await page
    .context()
    .addCookies([
      { name: "trestle-csrf", value: "csrf", domain: "127.0.0.1", path: "/" },
    ]);
  await page.route("**/api/v1/**", async (route) => {
    const req = route.request();
    const path = new URL(req.url()).pathname;
    const json = (body: object, status = 200) =>
      route.fulfill({ status, json: body });
    const method = req.method();

    if (path.endsWith("/dependencies") && path.includes("/tasks/")) {
      if (method === "POST") {
        const body = req.postDataJSON();
        writes.push({ method, path, body });
        const other = body.related_task_id === "web-task" ? blocker : blocked;
        const link = {
          id: `dep-${writes.length}`,
          task: other,
          created_at: "",
        };
        if (body.direction === "blocked_by") links.blocked_by.push(link);
        else links.blocks.push(link);
        return json({ dependency: link }, 201);
      }
      return json(links);
    }
    if (path.includes("/dependencies/") && method === "DELETE") {
      writes.push({ method, path, body: null });
      const id = path.split("/").pop();
      links.blocked_by = links.blocked_by.filter((l) => l.id !== id);
      links.blocks = links.blocks.filter((l) => l.id !== id);
      return route.fulfill({ status: 204 });
    }
    if (path.endsWith("/workspaces/ws/tasks")) {
      const query = new URL(req.url()).searchParams.get("query") || "";
      if (query === "") return json({ tasks: [] });
      const pool = [blocker, blocked].filter(
        (t) =>
          t.title.toLowerCase().includes(query.toLowerCase()) ||
          t.human_key.toLowerCase().includes(query.toLowerCase()),
      );
      return json({ tasks: pool });
    }
    if (method === "POST" || method === "PATCH" || method === "PUT") {
      writes.push({ method, path, body: req.postDataJSON() });
    }

    if (path.endsWith("/workspaces/ws")) return json({ ...workspace, role });
    if (path.endsWith("/workspaces")) {
      return json({ workspaces: [{ ...workspace, role }] });
    }
    if (path.endsWith("/projects/proj")) return json(project);
    if (path.endsWith("/projects")) return json({ projects: [project] });
    if (path.endsWith("/tasks/task")) return json(task);
    if (path.endsWith("/tasks")) return json({ tasks: [task], total: 1 });
    if (path.endsWith("/members"))
      return json({
        members: [
          {
            user_id: "owner",
            username: "owner",
            display_name: "Owner",
            role,
          },
        ],
      });
    if (path.endsWith("/statuses"))
      return json({
        statuses: [
          {
            id: "started",
            name: "In progress",
            category: "started",
            color: "#315b3b",
            position: 1,
          },
        ],
      });
    if (path.endsWith("/labels")) return json({ labels: [] });
    if (path.endsWith("/milestones")) return json({ milestones: [] });
    if (
      path.endsWith("/subtasks") ||
      path.endsWith("/activity") ||
      path.endsWith("/comments") ||
      path.endsWith("/time-logs")
    )
      return json([]);
    if (path.endsWith("/me"))
      return json({ id: "owner", username: "owner", display_name: "Owner" });
    return json({});
  });
  return writes;
}

async function openDrawer(page: Page) {
  await page.goto("/#task/ws/proj/task");
  await expect(page.getByRole("dialog")).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Dependencies", exact: true }),
  ).toBeVisible();
}

test("members can link both directions, search inside and across projects, and remove a link", async ({
  page,
}) => {
  const writes = await setup(page);
  await openDrawer(page);

  await expect(page.getByText("Nothing blocks this task.")).toBeVisible();
  await expect(page.getByText("This task blocks nothing.")).toBeVisible();

  // Search reaches another project and offers its high-level context.
  await page.getByLabel("Search tasks").fill("page");
  const result = page.getByRole("button", { name: /WEB-1.*Build page/ });
  await expect(result).toBeVisible();
  await result.click();

  await expect(page.locator(".dep-key", { hasText: "WEB-1" })).toBeVisible();
  await expect(
    page.getByRole("button", { name: "Remove blocker WEB-1" }),
  ).toBeVisible();
  await expect(page.getByText("This task blocks nothing.")).toBeVisible();

  // The link direction is explicit in the request.
  const added = writes.find((w) => w.method === "POST");
  expect(added?.body).toEqual({
    related_task_id: "web-task",
    direction: "blocked_by",
  });

  // The same picker can record the inverse relationship.
  await page.getByRole("button", { name: "Relationship" }).click();
  await page.getByRole("option", { name: /^blocks/ }).click();
  await page.getByLabel("Search tasks").fill("report");
  await page.getByRole("button", { name: /CORE-3.*Write report/ }).click();
  await expect(
    page.getByRole("button", { name: "Remove blocked task CORE-3" }),
  ).toBeVisible();
  expect(writes.filter((w) => w.method === "POST")[1].body).toEqual({
    related_task_id: "report-task",
    direction: "blocks",
  });

  // Removing confirms by result, not by leaving a stale row.
  await page.getByRole("button", { name: "Remove blocker WEB-1" }).click();
  await expect(page.locator(".dep-key", { hasText: "WEB-1" })).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: "Remove blocked task CORE-3" }),
  ).toBeVisible();
  expect(writes.some((w) => w.method === "DELETE")).toBe(true);
});

test("the picker explains an empty search without inventing results", async ({
  page,
}) => {
  await setup(page);
  await openDrawer(page);
  await page.getByLabel("Search tasks").fill("zzz");
  await expect(page.getByText(/No matching tasks/)).toBeVisible();
  await expect(page.locator(".dep-results li")).toHaveCount(0);
});

test("guests read dependencies but cannot change them", async ({ page }) => {
  await setup(page, "guest");
  await openDrawer(page);
  await expect(page.getByLabel("Search tasks")).toHaveCount(0);
  await expect(
    page.getByRole("button", { name: /Remove blocker/ }),
  ).toHaveCount(0);
  await expect(page.getByText("Nothing blocks this task.")).toBeVisible();
});
