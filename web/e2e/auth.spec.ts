import { expect, test } from "@playwright/test";

test.describe("Authentication flow", () => {
  test("navigates to sign-in form and focuses username", async ({ page }) => {
    await page.goto("/");

    // Click Sign in link
    await page.getByRole("link", { name: "Sign in" }).first().click();

    await expect(
      page.getByRole("heading", { name: "Sign in", exact: true }),
    ).toBeVisible();

    const usernameInput = page.getByLabel("Username");
    const passwordInput = page.getByLabel("Password");
    const submitButton = page.getByRole("button", { name: "Sign in" });

    await expect(usernameInput).toBeVisible();
    await expect(passwordInput).toBeVisible();
    await expect(submitButton).toBeVisible();
  });

  test("keyboard-first navigation through sign-in form", async ({ page }) => {
    await page.goto("/#signin");

    const usernameInput = page.getByLabel("Username");
    const passwordInput = page.getByLabel("Password");
    const submitButton = page.getByRole("button", { name: "Sign in" });

    await usernameInput.focus();
    await expect(usernameInput).toBeFocused();

    await page.keyboard.press("Tab");
    await expect(passwordInput).toBeFocused();

    await page.keyboard.press("Tab");
    await expect(submitButton).toBeFocused();
  });

  test("submitting invalid credentials displays an error alert", async ({
    page,
  }) => {
    await page.goto("/#signin");

    await page.getByLabel("Username").fill("nonexistent_user");
    await page.getByLabel("Password").fill("somepassword123");
    await page.getByRole("button", { name: "Sign in" }).click();

    // In this webServer test environment, the database is unavailable, so we get the connection error alert.
    const alert = page.getByRole("alert");
    await expect(alert).toBeVisible();
    expect(await alert.textContent()).toMatch(/could not sign in|invalid/i);
  });

  test("sign-in page reflows without horizontal scroll on phone viewport", async ({
    page,
  }) => {
    await page.setViewportSize({ width: 360, height: 740 });
    await page.goto("/#signin");

    await expect(
      page.getByRole("heading", { name: "Sign in", exact: true }),
    ).toBeVisible();
    await expect(page.getByLabel("Username")).toBeVisible();
    await expect(page.getByLabel("Password")).toBeVisible();
    await expect(page.getByRole("button", { name: "Sign in" })).toBeVisible();

    const scrollWidth = await page.evaluate(
      () => document.documentElement.scrollWidth,
    );
    expect(scrollWidth).toBeLessThanOrEqual(360);
  });
});
