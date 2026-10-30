import { expect, test } from "@playwright/test";

test("Rust serves the compiled foundation with a real unavailable readiness check", async ({
  page,
}) => {
  const errors: string[] = [];
  page.on("pageerror", (error) => errors.push(error.message));
  const assets: number[] = [];
  page.on("response", (response) => {
    if (response.url().includes("/assets/")) assets.push(response.status());
  });
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: "Trestle", exact: true }),
  ).toBeVisible();
  await expect(page.getByRole("status")).toHaveText(
    "Database connection unavailable.",
  );
  // The shell now has navigation before this control. Anchor navigation must
  // focus its heading, then Tab reaches the readiness action naturally.
  await page
    .getByRole("link", { name: "Instance status", exact: true })
    .click();
  await page.keyboard.press("Tab");
  await expect(page.getByRole("button", { name: "Check again" })).toBeFocused();
  const readiness = page.waitForResponse((response) =>
    response.url().endsWith("/health/ready"),
  );
  await page.keyboard.press("Enter");
  expect((await readiness).status()).toBe(503);
  await expect(page.getByRole("status")).toHaveText(
    "Database connection unavailable.",
  );
  expect(assets.length).toBeGreaterThanOrEqual(2);
  expect(assets.every((status) => status === 200)).toBe(true);
  expect(errors).toEqual([]);
});

test("foundation remains readable at phone width", async ({ page }) => {
  await page.setViewportSize({ width: 320, height: 640 });
  await page.goto("/");
  await expect(page.getByRole("button", { name: "Check again" })).toBeVisible();
  await expect(page.getByRole("status")).toHaveText(
    "Database connection unavailable.",
  );
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
});
