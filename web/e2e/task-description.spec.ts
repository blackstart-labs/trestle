import { expect, test, type Page } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

const description =
  "## Project structure\n\n- **Review progress**\n- Decide what to implement next\n\n[Reference](https://example.com)\n\n```go\nreview(project)\n```\n\n- [x] Structure reviewed";

async function setup(page: Page) {
  const project = {
    id: "core",
    workspace_id: "ws",
    name: "FreshFlow",
    slug: "freshflow",
    key: "FRES",
    description: "Project planning",
    task_counter: 1,
    archived: false,
    created_at: "2026-11-07T00:00:00Z",
    updated_at: "2026-11-07T00:00:00Z",
  };
  let task = {
    id: "task",
    workspace_id: "ws",
    project_id: "core",
    project_key: "FRES",
    human_key: "FRES-1",
    task_number: 1,
    title: "Review the Project Progress",
    description,
    status: "unstarted",
    status_id: "todo",
    priority: "none",
    due_date: null,
    labels: [],
    assignees: [],
    created_at: "2026-11-07T00:00:00Z",
    updated_at: "2026-11-07T00:00:00Z",
  };
  const writes: Record<string, unknown>[] = [];
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
      return json({ id: "owner", username: "admin", display_name: "Admin" });
    if (path.endsWith("/workspaces"))
      return json({
        workspaces: [
          { id: "ws", name: "Engineering", slug: "engineering", role: "owner" },
        ],
      });
    if (path.endsWith("/projects")) return json({ projects: [project] });
    if (path.endsWith("/statuses"))
      return json({
        statuses: [
          {
            id: "todo",
            name: "To Do",
            category: "unstarted",
            color: "var(--cat-unstarted)",
            position: 0,
          },
          {
            id: "done",
            name: "Done",
            category: "completed",
            color: "var(--cat-completed)",
            position: 1,
          },
        ],
      });
    if (path.endsWith("/tasks/task") && request.method() === "PATCH") {
      expect(request.headers()["x-csrf-token"]).toBe("csrf");
      const patch = request.postDataJSON();
      writes.push(patch);
      if (delay) await new Promise((resolve) => setTimeout(resolve, delay));
      if (fail)
        return json({ error: { message: "Could not save. Try again." } }, 503);
      task = { ...task, ...patch };
      if (patch.status_id)
        task.status = patch.status_id === "done" ? "completed" : "unstarted";
      return json(task);
    }
    if (path.endsWith("/tasks/task")) return json(task);
    if (path.endsWith("/tasks")) {
      if (request.method() === "POST") {
        writes.push(request.postDataJSON());
        return json(task, 201);
      }
      const matches =
        !url.searchParams.has("status") ||
        url.searchParams.get("status") === task.status;
      return json({
        tasks: matches ? [task] : [],
        total: matches ? 1 : 0,
        limit: 50,
        offset: 0,
      });
    }
    if (path.endsWith("/subtasks")) return json([]);
    if (path.endsWith("/comments")) return json({ comments: [] });
    if (path.endsWith("/activity")) return json({ activities: [] });
    if (path.endsWith("/labels")) return json({ labels: [] });
    if (path.endsWith("/members")) return json({ members: [] });
    return json(project);
  });
  await page.goto("/");
  await page.getByRole("link", { name: "Projects", exact: true }).click();
  await page.getByRole("button", { name: "Open tasks" }).click();
  await page
    .getByRole("button", {
      name: "View task FRES-1: Review the Project Progress",
    })
    .click();
  return {
    writes,
    fail: (value: boolean) => (fail = value),
    delay: (value: number) => (delay = value),
  };
}

test("description preview is local; property changes preserve drafts and failed saves can retry", async ({
  page,
}) => {
  const api = await setup(page);
  const dialog = page.getByRole("dialog");
  const draft =
    "  **Unsaved structure**\n\n- Next step  \n\n[Reference](https://example.com/)";
  await dialog.getByLabel("Description", { exact: true }).fill(draft);
  await dialog.getByRole("button", { name: "Preview", exact: true }).click();
  await expect(dialog.locator(".markdown-preview strong")).toHaveText(
    "Unsaved structure",
  );
  expect(api.writes).toHaveLength(0);
  await page
    .context()
    .route("https://example.com/", (route) =>
      route.fulfill({ contentType: "text/html", body: "Reference" }),
    );
  const popupPromise = page.waitForEvent("popup");
  await dialog.getByRole("link", { name: "Reference", exact: true }).click();
  const popup = await popupPromise;
  await expect(popup).toHaveURL("https://example.com/");
  await popup.close();
  await expect(dialog.locator(".markdown-preview strong")).toHaveText(
    "Unsaved structure",
  );
  await dialog.getByLabel("Priority", { exact: true }).click();
  await page.getByRole("option", { name: "Low", exact: true }).click();
  await expect(dialog.getByLabel("Priority", { exact: true })).toContainText(
    "Low",
  );
  expect(api.writes).toEqual([{ priority: "low" }]);
  await dialog.getByRole("button", { name: "Write", exact: true }).click();
  await expect(dialog.getByLabel("Description", { exact: true })).toHaveValue(
    draft,
  );
  api.fail(true);
  await dialog.getByRole("button", { name: "Save details" }).click();
  await expect(dialog.getByText("Could not save. Try again.")).toBeVisible();
  await expect(dialog.getByLabel("Description", { exact: true })).toHaveValue(
    draft,
  );
  api.fail(false);
  api.delay(300);
  await dialog.getByRole("button", { name: "Save details" }).click();
  await expect(
    dialog.getByLabel("Description", { exact: true }),
  ).toBeDisabled();
  await expect(dialog.getByText("All changes saved")).toBeVisible();
  expect(api.writes).toEqual([
    { priority: "low" },
    { description: draft },
    { description: draft },
  ]);
  await dialog.getByLabel("Status", { exact: true }).click();
  await page.getByRole("option", { name: "Done", exact: true }).click();
  await expect(dialog.getByLabel("Status", { exact: true })).toContainText(
    "Done",
  );
  expect(api.writes.at(-1)).toEqual({ status_id: "done" });
  await dialog.getByRole("button", { name: /^Due date:/ }).click();
  await page.getByRole("button", { name: "Today", exact: true }).click();
  await expect
    .poll(() => api.writes.at(-1))
    .toEqual({ due_date: expect.stringMatching(/^\d{4}-\d{2}-\d{2}$/) });
  await expect(
    dialog.getByRole("button", { name: /^Due date:/ }),
  ).toBeEnabled();
  await dialog.getByRole("button", { name: /^Due date:/ }).click();
  await page.getByRole("button", { name: "Clear date", exact: true }).click();
  await expect.poll(() => api.writes.at(-1)).toEqual({ due_date: null });
});

test.describe("touch task creation", () => {
  test.use({ hasTouch: true });
  for (const theme of ["light", "dark"] as const) {
    for (const width of [390, 555, 768, 1440]) {
      test(`${theme} new task supports Markdown at ${width}px`, async ({
        page,
      }, testInfo) => {
        await page.setViewportSize({ width, height: 1000 });
        await page.emulateMedia({
          colorScheme: theme,
          reducedMotion: "reduce",
        });
        const api = await setup(page);
        await page.getByRole("button", { name: "Close task details" }).tap();
        await expect(page.getByRole("dialog")).toHaveCount(0);
        await page.getByRole("button", { name: "New task", exact: true }).tap();
        const dialog = page.getByRole("dialog", {
          name: "New task in FRES",
          exact: true,
        });
        const dueDate = dialog.getByRole("button", {
          name: "Due date",
          exact: true,
        });
        const expectDateAlignment = async () => {
          const geometry = await dueDate.evaluate((button) => ({
            height: button.getBoundingClientRect().height,
            centers: [...button.children].map((child) => {
              const rect = child.getBoundingClientRect();
              return rect.top + rect.height / 2;
            }),
          }));
          expect(
            Math.max(...geometry.centers) - Math.min(...geometry.centers),
          ).toBeLessThanOrEqual(1);
          const status = await dialog
            .getByLabel("Status", { exact: true })
            .boundingBox();
          expect(geometry.height).toBeCloseTo(status!.height, 0);
        };
        await expectDateAlignment();
        await dialog
          .getByLabel("Title", { exact: true })
          .fill("Plan next steps");
        const source = "  **Project structure**\n\n- Acceptance criteria  \n";
        await dialog.getByLabel("Description", { exact: true }).fill(source);
        await page.screenshot({
          path: testInfo.outputPath("create-write.png"),
          animations: "disabled",
        });
        await dialog
          .getByRole("button", { name: "Preview", exact: true })
          .tap();
        await expect(dialog.locator(".markdown-preview strong")).toHaveText(
          "Project structure",
        );
        await page.screenshot({
          path: testInfo.outputPath("create-preview.png"),
          animations: "disabled",
        });
        expect(api.writes).toHaveLength(0);
        page.once("dialog", (prompt) => prompt.dismiss());
        await dialog.getByRole("button", { name: "Close dialog" }).tap();
        await expect(dialog).toBeVisible();
        await dialog.getByRole("button", { name: "Write", exact: true }).tap();
        await expect(
          dialog.getByLabel("Description", { exact: true }),
        ).toHaveValue(source);
        await dueDate.tap();
        const picker = page.getByRole("dialog", {
          name: "Choose a date",
          exact: true,
        });
        await expect(picker).toBeVisible();
        const bounds = (await picker.boundingBox())!;
        expect(bounds.x).toBeGreaterThanOrEqual(0);
        expect(bounds.x + bounds.width).toBeLessThanOrEqual(width);
        await picker
          .getByRole("button", { name: "Tomorrow", exact: true })
          .tap();
        await expect(dueDate).toContainText("Tomorrow");
        await expectDateAlignment();
        await dueDate.tap();
        await picker
          .getByRole("button", { name: "Clear date", exact: true })
          .tap();
        await expect(dueDate).toContainText("No date");
        await expectDateAlignment();
        const result = await new AxeBuilder({ page })
          .include('[role="dialog"]')
          .withTags(["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"])
          .analyze();
        expect(result.violations).toEqual([]);
        await dialog
          .getByRole("button", { name: "Save task", exact: true })
          .tap();
        await expect(dialog).toHaveCount(0);
        expect(api.writes).toHaveLength(1);
        expect(api.writes[0].description).toBe(source);
      });
    }
  }
});

for (const theme of ["light", "dark"] as const) {
  for (const width of [390, 768, 1440]) {
    test(`${theme} task Markdown reflows and is accessible at ${width}px`, async ({
      page,
    }, testInfo) => {
      await page.setViewportSize({ width, height: 1000 });
      await page.emulateMedia({ colorScheme: theme, reducedMotion: "reduce" });
      await setup(page);
      const dialog = page.getByRole("dialog");
      await page.screenshot({
        path: testInfo.outputPath("write.png"),
        animations: "disabled",
      });
      await dialog
        .getByRole("button", { name: "Preview", exact: true })
        .focus();
      await page.keyboard.press("Enter");
      await expect(dialog.locator(".markdown-preview h2")).toHaveText(
        "Project structure",
      );
      await page.screenshot({
        path: testInfo.outputPath("preview.png"),
        animations: "disabled",
      });
      const results = await new AxeBuilder({ page })
        .include('[role="dialog"]')
        .withTags(["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"])
        .analyze();
      expect(results.violations).toEqual([]);
      await page.evaluate(() => {
        document.documentElement.style.fontSize = "200%";
      });
      await page.screenshot({
        path: testInfo.outputPath("enlarged.png"),
        animations: "disabled",
      });
      const overflow = await page.evaluate(() =>
        [...document.querySelectorAll("body *")]
          .filter((el) => el.getBoundingClientRect().right > innerWidth + 1)
          .map((el) => ({
            tag: el.tagName,
            class: el.className,
            right: el.getBoundingClientRect().right,
          }))
          .slice(0, 12),
      );
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= window.innerWidth,
        ),
        JSON.stringify(overflow),
      ).toBe(true);
      await dialog.getByRole("button", { name: "Write", exact: true }).click();
      await expect(
        dialog.getByLabel("Description", { exact: true }),
      ).toBeVisible();
      expect(
        await dialog.evaluate((el) => el.scrollWidth <= el.clientWidth + 1),
      ).toBe(true);
    });
  }
}
