import { expect, test, type Page } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

const names = [
  "Backlog",
  "To Do",
  "In Progress",
  "Done",
  "Cancelled",
  "In Review",
];

async function setup(page: Page, role = "owner") {
  let statuses = names.map((name, position) => ({
    id: `status-${position}`,
    name,
    position,
    category: [
      "backlog",
      "unstarted",
      "started",
      "completed",
      "cancelled",
      "started",
    ][position],
    color: "var(--text-secondary)",
  }));
  const project = {
    id: "project",
    workspace_id: "ws",
    name: "FreshFlow",
    slug: "freshflow",
    key: "FRES",
    archived: false,
    task_counter: 0,
  };
  const writes: string[][] = [];
  let fail = false;
  let delay = 0;
  await page
    .context()
    .addCookies([
      { name: "trestle-csrf", value: "csrf", domain: "127.0.0.1", path: "/" },
    ]);
  await page.route("**/api/v1/**", async (route) => {
    const request = route.request();
    const url = new URL(request.url());
    const path = url.pathname;
    const json = (body: object, status = 200) =>
      route.fulfill({ status, json: body });
    if (path.endsWith("/me"))
      return json({ id: "admin", username: "admin", display_name: "Admin" });
    if (path.endsWith("/workspaces"))
      return json({
        workspaces: [
          { id: "ws", name: "Engineering", slug: "engineering", role },
        ],
      });
    if (path.endsWith("/projects")) return json({ projects: [project] });
    if (path.endsWith("/statuses/positions")) {
      expect(request.method()).toBe("PUT");
      expect(request.headers()["x-csrf-token"]).toBe("csrf");
      const ids: string[] = request.postDataJSON().status_ids;
      writes.push(ids);
      if (delay) await new Promise((resolve) => setTimeout(resolve, delay));
      if (fail)
        return json(
          { error: { message: "Could not save status order. Try again." } },
          503,
        );
      statuses = ids.map((id, position) => ({
        ...statuses.find((s) => s.id === id)!,
        position,
      }));
      return route.fulfill({ status: 204 });
    }
    if (path.endsWith("/statuses")) return json({ statuses });
    if (path.endsWith("/tasks")) {
      const matches =
        !url.searchParams.has("status") ||
        url.searchParams.get("status") === "unstarted";
      const task = {
        id: "task",
        workspace_id: "ws",
        project_id: "project",
        project_key: "FRES",
        human_key: "FRES-1",
        task_number: 1,
        title: "Review progress",
        description: "",
        status: "unstarted",
        status_id: "status-1",
        priority: "none",
        due_date: null,
        labels: [],
        assignees: [],
        created_at: "2026-11-08T00:00:00Z",
        updated_at: "2026-11-08T00:00:00Z",
      };
      return json({
        tasks: matches ? [task] : [],
        total: matches ? 1 : 0,
        limit: 50,
        offset: 0,
      });
    }
    if (path.endsWith("/members")) return json({ members: [] });
    if (path.endsWith("/labels")) return json({ labels: [] });
    return json(project);
  });
  await page.goto("/");
  await page.getByRole("link", { name: "Projects", exact: true }).click();
  await page.getByRole("button", { name: "Open tasks" }).click();
  await page
    .getByRole("button", { name: "Workflow statuses", exact: true })
    .click();
  return {
    writes,
    fail: (value: boolean) => (fail = value),
    delay: (value: number) => (delay = value),
  };
}

test("saved arrow order stays in the modal, Board, status filter, and after reload", async ({
  page,
}) => {
  const api = await setup(page);
  const rows = page.locator(".status-name");
  await page.getByRole("button", { name: "Move In Review up" }).focus();
  await page.keyboard.press("Enter");
  const moved = [
    "Backlog",
    "To Do",
    "In Progress",
    "Done",
    "In Review",
    "Cancelled",
  ];
  await expect(rows).toHaveText(moved);
  await expect(
    page.getByRole("button", { name: "Move In Review down" }),
  ).toBeEnabled();
  expect(api.writes).toEqual([
    ["status-0", "status-1", "status-2", "status-3", "status-5", "status-4"],
  ]);
  await page.getByRole("button", { name: "Done", exact: true }).click();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await page.getByRole("button", { name: "Board", exact: true }).click();
  await expect(page.locator(".col-name")).toHaveText(moved);
  await page
    .getByRole("button", { name: "Filter tasks by custom status" })
    .click();
  await expect(page.getByRole("option")).toHaveText(["All statuses", ...moved]);
  await page.keyboard.press("Escape");
  await page.reload();
  await page.getByRole("link", { name: "Projects", exact: true }).click();
  await page.getByRole("button", { name: "Open tasks" }).click();
  await page
    .getByRole("button", { name: "Workflow statuses", exact: true })
    .click();
  await expect(rows).toHaveText(moved);
  await page.getByRole("button", { name: "Move In Review down" }).click();
  await expect(rows).toHaveText(names);
});

test("failed reorder restores the list, prevents overlapping moves, and can retry", async ({
  page,
}) => {
  const api = await setup(page);
  api.delay(1000);
  api.fail(true);
  await page.getByRole("button", { name: "Move In Review up" }).click();
  await expect(page.getByRole("dialog").getByRole("status")).toHaveText(
    "Saving status order…",
  );
  await expect(
    page.getByRole("button", { name: "Move To Do up" }),
  ).toBeDisabled();
  await expect(
    page.getByText("Could not save status order. Try again."),
  ).toBeVisible();
  await expect(page.locator(".status-name")).toHaveText(names);
  expect(api.writes).toHaveLength(1);
  api.fail(false);
  await page.getByRole("button", { name: "Move In Review up" }).click();
  await expect(
    page.getByRole("button", { name: "Move In Review down" }),
  ).toBeEnabled();
  await expect(
    page.getByText("Could not save status order. Try again."),
  ).toHaveCount(0);
  expect(api.writes).toHaveLength(2);
});

for (const theme of ["light", "dark"] as const) {
  for (const width of [390, 768, 1440]) {
    test(`${theme} workflow arrows at ${width}px`, async ({
      page,
    }, testInfo) => {
      await page.setViewportSize({ width, height: 1000 });
      await page.emulateMedia({ colorScheme: theme, reducedMotion: "reduce" });
      await setup(page);
      await page.getByRole("button", { name: "Move In Review up" }).click();
      await expect(
        page.getByRole("button", { name: "Move In Review down" }),
      ).toBeEnabled();
      await page.locator(".status-list").evaluate(async (list) => {
        await new Promise(requestAnimationFrame);
        await Promise.all(
          list
            .getAnimations({ subtree: true })
            .map((animation) => animation.finished.catch(() => {})),
        );
      });
      await page.screenshot({
        path: testInfo.outputPath("reordered.png"),
        animations: "disabled",
      });
      const result = await new AxeBuilder({ page })
        .include(".status-dialog")
        .withTags(["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"])
        .analyze();
      expect(result.violations).toEqual([]);
      await page.evaluate(() => {
        document.documentElement.style.fontSize = "200%";
      });
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
      ).toBe(true);
    });
  }
}

test.describe("touch workflow arrows", () => {
  test.use({ hasTouch: true, viewport: { width: 390, height: 1000 } });
  test("moves a custom status up and down by touch", async ({ page }) => {
    await setup(page);
    await page.getByRole("button", { name: "Move In Review up" }).tap();
    await expect(
      page.getByRole("button", { name: "Move In Review down" }),
    ).toBeEnabled();
    await page.getByRole("button", { name: "Move In Review down" }).tap();
    await expect(page.locator(".status-name")).toHaveText(names);
  });
});
