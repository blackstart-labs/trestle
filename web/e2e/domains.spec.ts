import { expect, test, type Page } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

async function mockOperator(page: Page) {
  await page
    .context()
    .addCookies([
      { name: "trestle-csrf", value: "test", url: "http://127.0.0.1:4317" },
    ]);
  await page.route("**/api/v1/me", (route) =>
    route.fulfill({
      json: {
        id: "operator",
        username: "operator",
        display_name: "Instance operator",
        locale: "en",
        timezone: "UTC",
        instance_operator: true,
      },
    }),
  );
  await page.route("**/api/v1/workspaces", (route) =>
    route.fulfill({ json: { workspaces: [] } }),
  );
}

for (const width of [390, 768, 1440]) {
  for (const theme of ["light", "dark"] as const) {
    test.describe(`Domains at ${width}px in ${theme}`, () => {
      test.use({ viewport: { width, height: 900 }, hasTouch: width === 390 });
      test("fits, is operable and has no main-content axe violations", async ({
        page,
      }, testInfo) => {
        await page.emulateMedia({ colorScheme: theme });
        await mockOperator(page);
        await page.route("**/api/v1/instance/domain", (route) =>
          route.fulfill({
            json: { public_origin: "", listen_address: "0.0.0.0:3000" },
          }),
        );
        await page.goto("/#settings");
        const link = page
          .getByRole("navigation", { name: "Settings sections" })
          .getByRole("link", { name: "Domains" });
        if (width === 390) await link.tap();
        else {
          await link.focus();
          await page.keyboard.press("Enter");
        }
        const origin = page.getByRole("textbox", { name: "Public URL" });
        await expect(origin).toBeVisible();
        expect(
          (await new AxeBuilder({ page }).include("main").analyze()).violations,
        ).toEqual([]);
        expect(
          await page.evaluate(() => document.documentElement.scrollWidth),
        ).toBeLessThanOrEqual(width);
        const nav = page.getByRole("navigation", { name: "Application" });
        await expect(
          nav.getByRole("link", { name: "Settings" }),
        ).toHaveAttribute("aria-current", "page");
        await expect(link).toHaveAttribute("aria-current", "page");
        await page.screenshot({
          path: testInfo.outputPath("domains.png"),
          fullPage: true,
        });
        if (width === 1440) {
          await page
            .getByRole("link", { name: "Profile", exact: true })
            .click();
          await expect(
            page.getByRole("heading", { name: "Profile", exact: true }),
          ).toBeVisible();
          await page.screenshot({
            path: testInfo.outputPath("profile-reference.png"),
            fullPage: true,
          });
        }
      });
    });
  }
}

test("Domains keeps an unsaved URL after a failed save and offers a retry", async ({
  page,
}) => {
  await mockOperator(page);
  let attempts = 0;
  await page.route("**/api/v1/instance/domain", async (route) => {
    const publicOrigin = "https://trestle.example.com";
    if (route.request().method() === "GET") {
      await route.fulfill({
        json: { public_origin: "", listen_address: "127.0.0.1:3000" },
      });
      return;
    }
    expect(route.request().headers()["x-csrf-token"]).toBe("test");
    expect(route.request().postDataJSON()).toEqual({
      public_origin: publicOrigin,
      previous_origin: "",
    });
    attempts++;
    if (attempts === 1) {
      await route.fulfill({
        status: 503,
        json: {
          error: { message: "Could not save the public URL. Try again." },
        },
      });
    } else {
      await route.fulfill({
        json: { public_origin: publicOrigin, listen_address: "127.0.0.1:3000" },
      });
    }
  });
  await page.goto("/#domains");
  const field = page.getByRole("textbox", { name: "Public URL" });
  await field.fill("https://trestle.example.com");
  const save = page.getByRole("button", { name: "Save public URL" });
  await save.click();
  await expect(page.getByRole("alert")).toContainText("Try again");
  await expect(field).toHaveValue("https://trestle.example.com");
  await expect(field).not.toHaveAttribute("aria-invalid", "true");
  await page.getByRole("button", { name: "Discard changes" }).click();
  await expect(field).toHaveValue("");
  await expect(page.getByRole("alert")).toHaveCount(0);
  await field.fill("https://trestle.example.com");
  await save.click();
  await expect(page.getByText("Public URL saved.")).toBeVisible();
  await expect(save).toBeDisabled();
});
