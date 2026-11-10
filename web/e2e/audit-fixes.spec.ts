import { expect, test, type Page } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

const workspace = {
  id: "ws",
  name: "Engineering",
  slug: "engineering",
  role: "owner",
};
const project = {
  id: "proj",
  workspace_id: "ws",
  name: "Core platform",
  slug: "core",
  key: "CORE",
  description: "1",
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
  description: "Release checklist",
  status: "unstarted",
  priority: "none",
  due_date: null,
  labels: [],
  assignees: [],
  subtask_count: 2,
  created_at: "2026-11-01T00:00:00Z",
  updated_at: "2026-11-01T00:00:00Z",
};

async function setup(page: Page, signedIn = true) {
  const writes: { path: string; body: any }[] = [];
  if (signedIn)
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
    if (["POST", "PATCH"].includes(req.method())) {
      const body = req.postDataJSON();
      writes.push({ path, body });
      if (path.endsWith("/session")) {
        await page.context().addCookies([
          {
            name: "trestle-csrf",
            value: "csrf",
            domain: "127.0.0.1",
            path: "/",
          },
        ]);
        return json({ id: "owner", username: "owner", display_name: "Owner" });
      }
      if (path.endsWith("/workspaces"))
        return json({ ...workspace, ...body, id: "new" }, 201);
      if (path.endsWith("/workspaces/ws"))
        return json({ ...workspace, ...body });
      if (path.endsWith("/tasks")) return json({ ...task, ...body }, 201);
      if (path.endsWith("/time-logs"))
        return json({ spent_hours: body.hours }, 201);
    }
    if (path.endsWith("/me"))
      return json({ id: "owner", username: "owner", display_name: "Owner" });
    if (path.endsWith("/workspaces")) {
      await new Promise((resolve) => setTimeout(resolve, 80));
      return json({ workspaces: [workspace] });
    }
    if (path.endsWith("/projects")) return json({ projects: [project] });
    if (path.endsWith("/projects/proj")) return json(project);
    if (path.endsWith("/tasks/task")) return json(task);
    if (path.endsWith("/tasks")) return json({ tasks: [task], total: 1 });
    if (path.endsWith("/members"))
      return json({
        members: [
          {
            user_id: "owner",
            username: "owner",
            display_name: "Owner",
            role: "owner",
          },
        ],
      });
    if (path.endsWith("/milestones"))
      return json({
        milestones: [
          {
            id: "release",
            title: "Release",
            status: "open",
            color: "#315b3b",
            total_tasks: 0,
            completed_tasks: 0,
          },
        ],
      });
    if (
      path.endsWith("/subtasks") ||
      path.endsWith("/activity") ||
      path.endsWith("/comments") ||
      path.endsWith("/time-logs")
    )
      return json([]);
    return json({ statuses: [], labels: [] });
  });
  return writes;
}

test("notification task links survive startup and sign-in without opening the dashboard (#47)", async ({
  page,
}) => {
  await setup(page, false);
  await page.goto("/#task/ws/proj/task");
  await page.getByLabel("Username", { exact: true }).fill("owner");
  await page.getByLabel("Password", { exact: true }).fill("password");
  await page.getByRole("button", { name: "Sign in", exact: true }).click();
  await expect(page.getByRole("dialog")).toBeVisible();
  await expect(
    page.getByRole("dialog").getByLabel("Title", { exact: true }),
  ).toHaveValue(task.title);
  await page.reload();
  await expect(page.getByRole("dialog")).toBeVisible();
});

test("failed task links can retry and do not block navigation", async ({
  page,
}) => {
  await setup(page);
  await page.route(
    "**/projects/proj",
    (route) => route.fulfill({ status: 503, json: {} }),
    { times: 1 },
  );
  await page.goto("/#task/ws/proj/task");
  await expect(page.getByText(/Could not open this task/)).toBeVisible();
  await page.getByRole("button", { name: "Retry task link" }).click();
  await expect(page.getByRole("dialog")).toBeVisible();
});

test("creation includes people and milestones; list and board show subtask counts (#35, #36)", async ({
  page,
}) => {
  const writes = await setup(page);
  await page.goto("/#task/ws/proj/task");
  await page.getByRole("button", { name: "Close task details" }).click();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await expect(page.getByText("2 subtasks", { exact: true })).toBeVisible();
  await expect(page.locator(".title-text")).not.toContainText("1");
  await page.getByRole("button", { name: "Board", exact: true }).click();
  await expect(page.getByText("2 subtasks", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "New task", exact: false }).click();
  const dialog = page.getByRole("dialog");
  await dialog.getByLabel("Title", { exact: true }).fill("New release task");
  await dialog.getByRole("button", { name: "Assignees", exact: true }).click();
  await page.getByRole("option", { name: /Owner/ }).click();
  await page.keyboard.press("Escape");
  await dialog.getByLabel("Milestone", { exact: true }).click();
  await page.getByRole("option", { name: "Release", exact: true }).click();
  await dialog.getByRole("button", { name: "Save task" }).click();
  await expect(dialog).toHaveCount(0);
  expect(writes.find((w) => w.path.endsWith("/tasks"))?.body).toMatchObject({
    assignee_ids: ["owner"],
    milestone_id: "release",
  });
});

test("workspace creation and editing are reachable (#38)", async ({ page }) => {
  const writes = await setup(page);
  await page.goto("/");
  await page.getByRole("button", { name: "Switch workspace" }).click();
  await page.getByRole("button", { name: "New workspace" }).click();
  await page.getByLabel("Workspace name", { exact: true }).fill("Design");
  await page.getByLabel("Workspace slug", { exact: true }).fill("design");
  await page
    .getByRole("button", { name: "Create workspace", exact: true })
    .click();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  expect(writes[0].body).toEqual({ name: "Design", slug: "design" });
  await page.goto("/#workspace");
  await page.reload();
  await page
    .getByLabel("Workspace name", { exact: true })
    .fill("Engineering team");
  await page.getByRole("button", { name: "Save changes", exact: true }).click();
  await expect(page.locator(".ws-name")).toContainText("Engineering team");
});

for (const theme of ["light", "dark"]) {
  for (const width of [390, 768, 1440]) {
    test(`shell and timer: ${theme} at ${width}px (#37, #41, #44, #51)`, async ({
      page,
    }) => {
      await setup(page);
      await page.setViewportSize({ width, height: 900 });
      await page.addInitScript(
        ({ theme }) => {
          localStorage.setItem("trestle.theme", theme);
          localStorage.setItem(
            "trestle.active_timer",
            JSON.stringify({
              taskId: "task",
              taskHumanKey: "CORE-1",
              taskTitle: "Ship release",
              projectId: "proj",
              workspaceId: "ws",
              startTime: Date.now() - 60000,
              description: "",
            }),
          );
        },
        { theme },
      );
      await page.goto("/#task/ws/proj/task");
      await page.getByRole("button", { name: "Close task details" }).click();
      await expect(page.getByRole("dialog")).toHaveCount(0);
      await expect(
        page.getByRole("button", { name: "Stop timer", exact: true }),
      ).toBeVisible();
      const icon = await page.request.get("/assets/favicon.svg");
      expect(icon.status()).toBe(200);
      expect(icon.headers()["content-type"]).toContain("image/svg+xml");
      expect(
        await page
          .locator(".topbar")
          .evaluate((el) => getComputedStyle(el).backgroundColor),
      ).not.toContain("rgba");
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
      ).toBe(true);
      expect(
        (
          await new AxeBuilder({ page })
            .withTags(["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"])
            .analyze()
        ).violations,
      ).toEqual([]);
      await page.evaluate(() => window.scrollTo(0, document.body.scrollHeight));
      for (const button of await page
        .locator(".live-timer-banner button")
        .all()) {
        const box = await button.boundingBox();
        expect(box!.height).toBeGreaterThanOrEqual(44);
        expect(box!.width).toBeGreaterThanOrEqual(44);
      }
      if (width === 390) {
        for (const button of await page
          .locator(".page-head .page-actions button")
          .all()) {
          expect((await button.boundingBox())!.width).toBeGreaterThanOrEqual(
            140,
          );
        }
      }
      await page.locator("main").focus();
      await page.screenshot({
        path: `/tmp/opencode/audit-${theme}-${width}.png`,
        fullPage: true,
        animations: "disabled",
      });
      if (width === 390) {
        await page.setViewportSize({ width: 320, height: 900 });
        await page.evaluate(() => {
          document.documentElement.style.fontSize = "200%";
        });
        const overflow = await page.evaluate(() =>
          [...document.querySelectorAll("main *, .topbar *")]
            .filter((el) => el.getBoundingClientRect().right > innerWidth)
            .map((el) => ({
              tag: el.tagName,
              class: el.className,
              right: el.getBoundingClientRect().right,
            })),
        );
        await page.screenshot({
          path: `/tmp/opencode/audit-zoom-${theme}.png`,
          fullPage: true,
        });
        expect(
          await page.evaluate(
            () => document.documentElement.scrollWidth <= innerWidth,
          ),
          JSON.stringify(overflow),
        ).toBe(true);
      }
      await page
        .getByRole("button", { name: "Stop timer", exact: true })
        .click();
      await expect(
        page.getByRole("button", { name: "Stop timer", exact: true }),
      ).toHaveCount(0);
    });
  }
}
