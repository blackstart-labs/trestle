import { expect, test, type Page } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

const projects = [
  {
    id: "alpha",
    name: "Core Platform",
    key: "CORE",
    slug: "core",
    archived: false,
    updated_at: "2026-11-07T00:00:00Z",
  },
  {
    id: "beta",
    name: "Client Portal",
    key: "PORTAL",
    slug: "portal",
    archived: false,
    updated_at: "2026-11-07T00:00:00Z",
  },
];

async function setup(page: Page, role = "owner", empty = false) {
  await page.context().addCookies([
    {
      name: "trestle-csrf",
      value: "test-csrf",
      domain: "127.0.0.1",
      path: "/",
    },
  ]);
  let failSave = false;
  let failLoad = false;
  let loadDelay = 0;
  const connections: Record<string, Record<string, object>> = {
    alpha: {},
    beta: {},
  };
  const writes: { method: string; path: string; body: object | null }[] = [];
  await page.route("**/api/v1/**", async (route) => {
    const req = route.request();
    const path = new URL(req.url()).pathname;
    const method = req.method();
    const json = (body: object, status = 200) =>
      route.fulfill({ status, json: body });
    if (path.endsWith("/me"))
      return json({
        id: "owner",
        username: "owner",
        display_name: "Project Owner",
      });
    if (path === "/api/v1/workspaces")
      return json({
        workspaces: [
          { id: "ws", name: "Engineering", slug: "engineering", role },
        ],
      });
    if (path.endsWith("/projects"))
      return json({ projects: empty ? [] : projects });
    const integration = path.match(
      /\/projects\/(alpha|beta)\/integrations(?:\/(telegram|slack|discord)(\/test)?)?$/,
    );
    if (integration) {
      const [, project, provider, isTest] = integration;
      if (method === "GET") {
        if (loadDelay)
          await new Promise((resolve) => setTimeout(resolve, loadDelay));
        if (failLoad) return json({ error: { message: "Load failed" } }, 503);
        return json({ integrations: Object.values(connections[project]) });
      }
      expect(req.headers()["x-csrf-token"]).toBe("test-csrf");
      writes.push({ method, path, body: req.postDataJSON() });
      if (method === "PUT") {
        if (failSave)
          return json(
            { error: { message: "Check the webhook URL and try again." } },
            400,
          );
        connections[project][provider] = {
          provider,
          chat_id: req.postDataJSON().chat_id || "",
          last_attempt_at: null,
          last_error: "",
        };
        return json(connections[project][provider]);
      }
      if (method === "POST" && isTest) {
        connections[project][provider] = {
          ...connections[project][provider],
          last_attempt_at: new Date().toISOString(),
          last_error: "",
        };
        return json({ sent: true });
      }
      delete connections[project][provider];
      return route.fulfill({ status: 204 });
    }
    if (path.endsWith("/tasks"))
      return json({ tasks: [], total: 0, limit: 1, offset: 0 });
    return json({ statuses: [] });
  });
  await page.goto("/");
  await expect(page.locator(".ws-name")).toHaveText("Engineering");
  await page.getByRole("link", { name: "Settings", exact: true }).click();
  await page.getByRole("link", { name: "Integration", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Integration", exact: true }),
  ).toBeFocused();
  return {
    writes,
    connections,
    failSave: (v: boolean) => (failSave = v),
    failLoad: (v: boolean) => (failLoad = v),
    loadDelay: (v: number) => (loadDelay = v),
  };
}

test("project connections save, test, preserve errors, and disconnect with confirmation", async ({
  page,
}, testInfo) => {
  await page.emulateMedia({ colorScheme: "dark", reducedMotion: "reduce" });
  const api = await setup(page);
  for (const provider of ["Telegram", "Slack", "Discord"]) {
    await page.getByRole("button", { name: `Connect ${provider}` }).click();
    await page
      .getByLabel(provider === "Telegram" ? "Bot token" : "Webhook URL")
      .fill(
        provider === "Telegram"
          ? "123456:abcdefghijklmnopqrstuvwxyz_123456789"
          : provider === "Slack"
            ? "https://hooks.slack.com/services/T/B/TOKEN"
            : "https://discord.com/api/webhooks/123/token",
      );
    if (provider === "Telegram")
      await page.getByLabel("Chat ID", { exact: true }).fill("-100123");
    await page.getByRole("button", { name: "Save integration" }).click();
    await expect(page.getByRole("dialog")).toHaveCount(0);
    await expect(
      page.getByRole("button", { name: `Replace ${provider}` }),
    ).toBeVisible();
    await page.getByRole("button", { name: `Test ${provider}` }).click();
    await expect(
      page.getByText(`${provider} test message sent.`),
    ).toBeVisible();
  }
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.screenshot({
    path: testInfo.outputPath("configured.png"),
    fullPage: true,
    animations: "disabled",
  });
  await page.getByRole("button", { name: "Replace Slack" }).click();
  await expect(page.getByLabel("Webhook URL")).toHaveValue("");
  await page.getByLabel("Webhook URL").fill("bad-value");
  api.failSave(true);
  await page.getByRole("button", { name: "Save integration" }).click();
  await expect(page.getByRole("alert")).toHaveText(
    "Check the webhook URL and try again.",
  );
  await expect(page.getByLabel("Webhook URL")).toHaveValue("bad-value");
  await page.getByRole("dialog").screenshot({
    path: testInfo.outputPath("save-error.png"),
    animations: "disabled",
  });
  page.once("dialog", (dialog) => dialog.dismiss());
  await page.getByRole("button", { name: "Cancel", exact: true }).click();
  await expect(page.getByRole("dialog")).toBeVisible();
  page.once("dialog", (dialog) => dialog.accept());
  await page.getByRole("button", { name: "Cancel", exact: true }).click();
  await expect(page.getByRole("dialog")).toHaveCount(0);
  await page.getByRole("button", { name: "Project", exact: true }).click();
  await page.getByRole("option", { name: "Client Portal (PORTAL)" }).click();
  await expect(
    page.getByRole("button", { name: "Connect Slack" }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Project", exact: true }).click();
  await page.getByRole("option", { name: "Core Platform (CORE)" }).click();
  await page.getByRole("button", { name: "Disconnect Slack" }).click();
  expect(api.writes.filter((r) => r.method === "DELETE")).toHaveLength(0);
  await page
    .getByRole("button", { name: "Disconnect integration", exact: true })
    .click();
  await expect(
    page.getByRole("button", { name: "Connect Slack" }),
  ).toBeVisible();
});

test("integration empty and permission-limited states", async ({ page }) => {
  await setup(page, "member");
  await expect(
    page.getByText(/Only Workspace owners and admins/),
  ).toBeVisible();
  await expect(page.getByRole("button", { name: "Connect Slack" })).toHaveCount(
    0,
  );
});

test("Workspace without active Projects", async ({ page }) => {
  await setup(page, "owner", true);
  await expect(
    page.getByRole("heading", { name: "No active Projects" }),
  ).toBeVisible();
});

test("integration loading and load failure offer a working retry", async ({
  page,
}) => {
  const api = await setup(page);
  await page.getByRole("button", { name: "Project", exact: true }).click();
  api.failLoad(true);
  await page.getByRole("option", { name: "Client Portal (PORTAL)" }).click();
  await expect(page.getByRole("alert")).toHaveText(
    "Could not load integrations. Try again.",
  );
  api.failLoad(false);
  api.loadDelay(400);
  await page.getByRole("button", { name: "Retry integrations" }).click();
  await expect(page.getByRole("status")).toHaveText(
    "Loading project integrations…",
  );
  await expect(
    page.getByRole("button", { name: "Connect Telegram" }),
  ).toBeVisible();
});

for (const theme of ["dark", "light"] as const) {
  for (const width of [390, 768, 1440]) {
    test(`integration ${theme} at ${width}px supports keyboard, touch targets, and reflow`, async ({
      page,
    }, testInfo) => {
      await page.emulateMedia({ colorScheme: theme, reducedMotion: "reduce" });
      await page.setViewportSize({ width, height: 1000 });
      await setup(page);
      const panel = page.getByRole("region", {
        name: "Integration",
        exact: true,
      });
      await expect(
        page.getByRole("button", { name: "Connect Telegram" }),
      ).toBeVisible();
      await panel.screenshot({
        path: testInfo.outputPath(`integration-${theme}-${width}.png`),
        animations: "disabled",
      });
      await page.evaluate(() => window.scrollTo(0, 0));
      await page.screenshot({
        path: testInfo.outputPath(`integration-page-${theme}-${width}.png`),
        fullPage: true,
        animations: "disabled",
      });
      expect(
        (await new AxeBuilder({ page }).include(".integrations").analyze())
          .violations,
      ).toEqual([]);
      for (const button of await panel.getByRole("button").all()) {
        expect((await button.boundingBox())!.height).toBeGreaterThanOrEqual(44);
      }
      await page.getByRole("button", { name: "Connect Telegram" }).focus();
      await page.keyboard.press("Enter");
      await expect(page.getByLabel("Bot token")).toBeFocused();
      await page.keyboard.press("Tab");
      await expect(page.getByLabel("Chat ID", { exact: true })).toBeFocused();
      await expect(page.getByRole("dialog")).toHaveCSS("opacity", "1");
      await expect(page.locator(".scrim")).toHaveCSS("opacity", "1");
      await page.screenshot({
        path: testInfo.outputPath(`dialog-${theme}-${width}.png`),
        animations: "disabled",
      });
      expect(
        (await new AxeBuilder({ page }).include('[role="dialog"]').analyze())
          .violations,
      ).toEqual([]);
      await page.keyboard.press("Escape");
      await expect(
        page.getByRole("button", { name: "Connect Telegram" }),
      ).toBeFocused();
      await expect(page.getByRole("dialog")).toHaveCount(0);
      await page.evaluate(() => {
        document.documentElement.style.fontSize = "200%";
      });
      const overflow = await page.evaluate(() => ({
        fits: document.documentElement.scrollWidth <= innerWidth,
        elements: [...document.querySelectorAll("body *")]
          .filter((el) => el.getBoundingClientRect().right > innerWidth + 1)
          .map((el) => ({
            tag: el.tagName,
            class: el.className,
            text: el.textContent?.slice(0, 70),
          })),
      }));
      expect(overflow.fits, JSON.stringify(overflow.elements)).toBe(true);
    });
  }
}

test.describe("touch", () => {
  test.use({ hasTouch: true, viewport: { width: 390, height: 844 } });
  test("connect dialog opens and closes by touch", async ({ page }) => {
    await setup(page);
    await page.getByRole("button", { name: "Connect Discord" }).tap();
    await expect(page.getByRole("dialog")).toBeVisible();
    await page.getByRole("button", { name: "Cancel", exact: true }).tap();
    await expect(page.getByRole("dialog")).toHaveCount(0);
  });
});
