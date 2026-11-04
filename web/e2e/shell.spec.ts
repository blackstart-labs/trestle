import { expect, test } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

test.describe("touch navigation", () => {
  test.use({ hasTouch: true, viewport: { width: 390, height: 844 } });
  test("touch users can reach instance status and retry the real check", async ({
    page,
  }) => {
    await page.goto("/");
    await page.getByRole("link", { name: "Settings", exact: true }).tap();
    await expect(page.getByRole("status")).toHaveText(
      "Database connection unavailable.",
    );
    const heading = page.getByRole("heading", { name: "Instance status" });
    await heading.scrollIntoViewIfNeeded();
    await expect(heading).toBeInViewport();
    const response = page.waitForResponse((r) =>
      r.url().endsWith("/health/ready"),
    );
    await page.getByRole("button", { name: "Check again" }).tap();
    expect((await response).status()).toBe(503);
    await expect(page.getByRole("status")).toHaveText(
      "Database connection unavailable.",
    );
  });
});

test("keyboard users can skip navigation to the real main content", async ({
  page,
}) => {
  await page.goto("/");
  await page.keyboard.press("Tab");
  await expect(
    page.getByRole("link", { name: "Skip to content" }),
  ).toBeFocused();
  await page.keyboard.press("Enter");
  await expect(page.getByRole("main")).toBeFocused();
  await expect(
    page.getByRole("navigation", { name: "Application" }),
  ).toBeVisible();
});

test("system follows the device, explicit themes survive reload and can return to system", async ({
  page,
}) => {
  await page.emulateMedia({ colorScheme: "dark" });
  await page.goto("/#settings");
  const root = page.locator("html");
  const card = (name: string) =>
    page.getByRole("radio", { name: new RegExp(`^${name}\\b`) });
  await expect(root).toHaveCSS("color-scheme", "dark");
  await card("Light").check();
  await expect(root).toHaveCSS("color-scheme", "light");
  await page.reload();
  await expect(card("Light")).toBeChecked();
  await expect(root).toHaveCSS("color-scheme", "light");
  await card("Dark").check();
  await page.emulateMedia({ colorScheme: "light" });
  await expect(root).toHaveCSS("color-scheme", "dark");
  await card("System").check();
  await expect(root).toHaveCSS("color-scheme", "light");
  await page.emulateMedia({ colorScheme: "dark" });
  await expect(root).toHaveCSS("color-scheme", "dark");
});

for (const theme of ["light", "dark"]) {
  test(`${theme} semantic token pairs meet contrast thresholds`, async ({
    page,
  }) => {
    await page.goto("/#settings");
    await page
      .getByRole("radio", { name: new RegExp(`^${theme}\\b`, "i") })
      .check();
    const pairs: [string, string, number][] = [];
    for (const background of [
      "surface",
      "surface-raised",
      "surface-nav",
      "selected",
    ]) {
      for (const foreground of [
        "text",
        "text-secondary",
        "text-muted",
        "accent",
      ]) {
        pairs.push([foreground, background, 4.5]);
      }
      pairs.push(["focus", background, 3], ["border", background, 3]);
    }
    for (const foreground of ["success", "warning", "danger"])
      pairs.push([foreground, "surface-raised", 4.5]);
    pairs.push(["button-text", "button-bg", 4.5]);
    const measured = await page.evaluate((pairs) => {
      const probe = document.createElement("span");
      document.body.append(probe);
      // Tokens are authored in OKLCH; a canvas resolves any CSS colour space
      // to the sRGB bytes that are actually displayed.
      const canvas = document.createElement("canvas");
      canvas.width = canvas.height = 1;
      const ctx = canvas.getContext("2d", { willReadFrequently: true })!;
      function toSrgb(color: string) {
        ctx.clearRect(0, 0, 1, 1);
        ctx.fillStyle = color;
        ctx.fillRect(0, 0, 1, 1);
        return [...ctx.getImageData(0, 0, 1, 1).data.slice(0, 3)];
      }
      function luminance(color: string) {
        const rgb = toSrgb(color).map((v) => {
          const s = v / 255;
          return s <= 0.04045 ? s / 12.92 : ((s + 0.055) / 1.055) ** 2.4;
        });
        return rgb[0] * 0.2126 + rgb[1] * 0.7152 + rgb[2] * 0.0722;
      }
      const results = pairs.map(([foreground, background, minimum]) => {
        probe.style.color = `var(--${foreground})`;
        probe.style.backgroundColor = `var(--${background})`;
        const style = getComputedStyle(probe);
        const a = luminance(style.color),
          b = luminance(style.backgroundColor);
        return {
          foreground,
          background,
          minimum,
          ratio: (Math.max(a, b) + 0.05) / (Math.min(a, b) + 0.05),
        };
      });
      probe.remove();
      return results;
    }, pairs);
    for (const pair of measured)
      expect(
        pair.ratio,
        `${theme} ${pair.foreground}/${pair.background}`,
      ).toBeGreaterThanOrEqual(pair.minimum);
  });

  test(`${theme} shell has no automated accessibility violations`, async ({
    page,
  }) => {
    await page.goto("/#settings");
    await page
      .getByRole("radio", { name: new RegExp(`^${theme}\\b`, "i") })
      .check();
    await expect(page.getByRole("status")).toHaveText(
      "Database connection unavailable.",
    );
    // Settings carries the most controls; the overview is the landing page.
    for (const hash of ["#settings", "#overview", "#signin"]) {
      await page.goto(`/${hash}`);
      await page.waitForTimeout(500);
      const result = await new AxeBuilder({ page })
        .withTags(["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"])
        .analyze();
      expect(result.violations, hash).toEqual([]);
    }
  });
}

for (const [width, height] of [
  [320, 640],
  [360, 800],
  [390, 844],
  [844, 390],
  [768, 1024],
  [1440, 900],
]) {
  test(`shell reflows at ${width}x${height} with enlarged text`, async ({
    page,
  }) => {
    await page.setViewportSize({ width, height });
    await page.goto("/");
    // Test-only text enlargement through CSSOM; production CSP is not disabled.
    await page.evaluate(() => {
      document.documentElement.style.fontSize = "200%";
    });
    await page.getByRole("link", { name: "Settings", exact: true }).click();
    await expect(
      page.getByRole("heading", { name: "Settings", level: 1 }),
    ).toBeFocused();
    await expect(page.getByRole("status")).toHaveText(
      "Database connection unavailable.",
    );
    await expect(page.getByRole("radio", { name: /^Light\b/ })).toBeVisible();
    expect(
      await page.evaluate(
        () => document.documentElement.scrollWidth <= innerWidth,
      ),
    ).toBe(true);
    for (const control of await page.locator("nav a, button, select").all()) {
      expect((await control.boundingBox())!.height).toBeGreaterThanOrEqual(44);
    }
  });
}
