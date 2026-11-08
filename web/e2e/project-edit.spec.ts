import { expect, test, type Page } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

async function setup(page: Page, role = "owner") {
  let project = {
    id: "core",
    workspace_id: "ws",
    name: "Core",
    slug: "core",
    key: "CORE",
    description: "Original description",
    task_counter: 1,
    archived: false,
    created_at: "2026-11-08T00:00:00Z",
    updated_at: "2026-11-08T00:00:00Z",
  };
  const writes: Record<string, string>[] = [];
  let fail = false;
  let delay = 0;
  await page
    .context()
    .addCookies([
      { name: "trestle-csrf", value: "csrf", domain: "127.0.0.1", path: "/" },
    ]);
  await page.route("**/api/v1/**", async (route) => {
    const request = route.request();
    const path = new URL(request.url()).pathname;
    const json = (body: object, status = 200) =>
      route.fulfill({ status, json: body });
    if (path.endsWith("/me"))
      return json({
        id: "owner",
        username: "owner",
        display_name: "Project Owner",
      });
    if (path.endsWith("/workspaces"))
      return json({
        workspaces: [
          { id: "ws", name: "Engineering", slug: "engineering", role },
        ],
      });
    if (path.endsWith("/projects")) return json({ projects: [project] });
    if (path.endsWith("/projects/core") && request.method() === "PATCH") {
      expect(request.headers()["x-csrf-token"]).toBe("csrf");
      const patch = request.postDataJSON();
      writes.push(patch);
      if (delay) await new Promise((resolve) => setTimeout(resolve, delay));
      if (fail)
        return json(
          {
            error: {
              message:
                "A project with this slug already exists in the workspace",
            },
          },
          409,
        );
      project = { ...project, ...patch, updated_at: "2026-11-08T01:00:00Z" };
      return json(project);
    }
    if (path.endsWith("/projects/core")) return json(project);
    if (path.endsWith("/tasks"))
      return json({ tasks: [], total: 0, limit: 50, offset: 0 });
    if (path.endsWith("/statuses")) return json({ statuses: [] });
    if (path.endsWith("/labels")) return json({ labels: [] });
    if (path.endsWith("/members")) return json({ members: [] });
    return json({});
  });
  await page.goto("/");
  await page.getByRole("link", { name: "Projects", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Core", exact: true }),
  ).toBeVisible();
  return {
    writes,
    fail: (value: boolean) => (fail = value),
    delay: (value: number) => (delay = value),
  };
}

test("project edits update the card, open-project header and breadcrumbs, then survive reload", async ({
  page,
}) => {
  const api = await setup(page);
  await page
    .getByRole("button", { name: "Edit project Core", exact: true })
    .click();
  const dialog = page.getByRole("dialog", {
    name: "Edit project",
    exact: true,
  });
  await expect(
    dialog.getByLabel("Project key", { exact: true }),
  ).toHaveAttribute("readonly", "");
  await expect(
    dialog.getByRole("button", { name: "Save project" }),
  ).toBeDisabled();
  await dialog
    .getByLabel("Project name", { exact: true })
    .fill("Core Platform");
  await dialog.getByLabel("Slug", { exact: true }).fill("platform");
  await dialog
    .getByLabel("Description", { exact: true })
    .fill("Updated project context");
  await dialog.getByRole("button", { name: "Save project" }).click();
  await expect(dialog).toHaveCount(0);
  expect(api.writes).toEqual([
    {
      name: "Core Platform",
      slug: "platform",
      description: "Updated project context",
    },
  ]);
  await expect(
    page.getByRole("heading", { name: "Core Platform", exact: true }),
  ).toBeVisible();
  await expect(page.getByText("/platform", { exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Open tasks" }).click();
  await page.getByRole("button", { name: "Edit project", exact: true }).click();
  await dialog
    .getByLabel("Project name", { exact: true })
    .fill("Platform Engineering");
  await dialog.getByLabel("Description", { exact: true }).fill("");
  await dialog.getByRole("button", { name: "Save project" }).click();
  await expect(dialog).toHaveCount(0);
  await expect(
    page.getByRole("heading", {
      name: "Tasks for Platform Engineering",
      exact: true,
    }),
  ).toBeVisible();
  await expect(page.locator(".crumb-current")).toHaveText(
    "Platform Engineering",
  );
  await expect(page).toHaveTitle(/Platform Engineering/);
  expect(api.writes[1]).toEqual({
    name: "Platform Engineering",
    description: "",
  });
  await page.reload();
  await page.getByRole("link", { name: "Projects", exact: true }).click();
  await page
    .getByRole("button", {
      name: "Edit project Platform Engineering",
      exact: true,
    })
    .click();
  await expect(dialog.getByLabel("Project name", { exact: true })).toHaveValue(
    "Platform Engineering",
  );
  await expect(dialog.getByLabel("Description", { exact: true })).toHaveValue(
    "",
  );
  await expect(dialog.getByLabel("Project key", { exact: true })).toHaveValue(
    "CORE",
  );
});

test("project edit errors retain the draft, dirty close is guarded, and retry succeeds", async ({
  page,
}) => {
  const api = await setup(page);
  await page
    .getByRole("button", { name: "Edit project Core", exact: true })
    .click();
  const dialog = page.getByRole("dialog", {
    name: "Edit project",
    exact: true,
  });
  await dialog.getByLabel("Slug", { exact: true }).fill("platform");
  page.once("dialog", (prompt) => prompt.dismiss());
  await page.keyboard.press("Escape");
  await expect(dialog).toBeVisible();
  api.fail(true);
  await dialog.getByRole("button", { name: "Save project" }).click();
  await expect(dialog.getByRole("alert")).toContainText(
    "this slug already exists",
  );
  await expect(dialog.getByLabel("Slug", { exact: true })).toHaveValue(
    "platform",
  );
  api.fail(false);
  api.delay(500);
  await dialog.getByRole("button", { name: "Save project" }).click();
  await expect(dialog.getByLabel("Slug", { exact: true })).toBeDisabled();
  await page.keyboard.press("Escape");
  await expect(dialog).toBeVisible();
  await expect(dialog).toHaveCount(0);
  expect(api.writes).toHaveLength(2);
});

test("workspace members do not see project edit actions", async ({ page }) => {
  await setup(page, "member");
  await expect(page.getByRole("button", { name: /Edit project/ })).toHaveCount(
    0,
  );
  await page.getByRole("button", { name: "Open tasks" }).click();
  await expect(
    page.getByRole("button", { name: "Edit project", exact: true }),
  ).toHaveCount(0);
});

for (const theme of ["light", "dark"] as const) {
  for (const width of [390, 768, 1440]) {
    test(`${theme} project editor at ${width}px supports keyboard and reflow`, async ({
      page,
    }, testInfo) => {
      await page.setViewportSize({ width, height: 1000 });
      await page.emulateMedia({ colorScheme: theme, reducedMotion: "reduce" });
      await setup(page);
      await page
        .getByRole("button", { name: "Edit project Core", exact: true })
        .focus();
      await page.keyboard.press("Enter");
      const dialog = page.getByRole("dialog", {
        name: "Edit project",
        exact: true,
      });
      await expect(
        dialog.getByLabel("Project name", { exact: true }),
      ).toBeFocused();
      await page.keyboard.press("Tab");
      await expect(dialog.getByLabel("Slug", { exact: true })).toBeFocused();
      const slugBounds = (await dialog
        .getByLabel("Slug", { exact: true })
        .boundingBox())!;
      const keyBounds = (await dialog
        .getByLabel("Project key", { exact: true })
        .boundingBox())!;
      expect(Math.abs(slugBounds.y - keyBounds.y)).toBeLessThanOrEqual(1);
      expect(
        Math.abs(slugBounds.height - keyBounds.height),
      ).toBeLessThanOrEqual(1);
      await page.screenshot({
        path: testInfo.outputPath("project-edit.png"),
        animations: "disabled",
      });
      const result = await new AxeBuilder({ page })
        .include('[role="dialog"]')
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
      await page.keyboard.press("Escape");
      await expect(dialog).toHaveCount(0);
      await expect(
        page.getByRole("button", { name: "Edit project Core", exact: true }),
      ).toBeFocused();
    });
  }
}

test.describe("touch project editing", () => {
  test.use({ hasTouch: true, viewport: { width: 390, height: 1000 } });
  test("edits a project from its card by touch", async ({ page }) => {
    await setup(page);
    await page
      .getByRole("button", { name: "Edit project Core", exact: true })
      .tap();
    const dialog = page.getByRole("dialog", {
      name: "Edit project",
      exact: true,
    });
    await dialog
      .getByLabel("Project name", { exact: true })
      .fill("Touch project");
    await dialog.getByRole("button", { name: "Save project" }).tap();
    await expect(dialog).toHaveCount(0);
    await expect(
      page.getByRole("heading", { name: "Touch project", exact: true }),
    ).toBeVisible();
  });
});
