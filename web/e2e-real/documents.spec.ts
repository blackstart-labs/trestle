import { expect, test } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";

test("real documents: hierarchy, backlinks, conflicts, revisions, archive and export", async ({
  page,
  context,
}, testInfo) => {
  await page.goto("/#signin");
  await page.getByLabel("Username", { exact: true }).fill("boardtester");
  await page
    .getByLabel("Password", { exact: true })
    .fill("board-browser-test-only");
  await page.getByRole("button", { name: "Sign in", exact: true }).click();
  await page.getByRole("link", { name: "Documents", exact: true }).click();
  await page.getByRole("button", { name: "New document", exact: true }).click();
  const title = `Architecture ${Date.now()}`;
  await page.getByLabel("Title", { exact: true }).fill(title);
  const markdown =
    "# Architecture\n\n- [ ] Review\n\n| Decision | Owner |\n| --- | --- |\n| Portable Markdown | Team |\n\n<script>alert('unsafe')</script>\n";
  await page.getByLabel("Document content", { exact: true }).fill(markdown);
  const creation = page.waitForResponse(
    (r) => r.request().method() === "POST" && r.url().endsWith("/documents"),
  );
  await page
    .getByRole("button", { name: "Save document", exact: true })
    .click();
  const parent = await (await creation).json();
  await expect(page.getByText("Unsaved changes", { exact: true })).toHaveCount(
    0,
  );
  await page
    .getByRole("button", { name: "New child page", exact: true })
    .click();
  await page.getByLabel("Title", { exact: true }).fill("Implementation notes");
  await page
    .getByLabel("Document content", { exact: true })
    .fill(`[[${parent.id}|Architecture]]\n\nChild body`);
  await page
    .getByRole("button", { name: "Save document", exact: true })
    .click();
  await expect(page.getByText("Unsaved changes", { exact: true })).toHaveCount(
    0,
  );
  await page
    .getByRole("button", { name: "Open parent page", exact: true })
    .click();
  await expect(page.getByLabel("Title", { exact: true })).toHaveValue(title);
  await expect(
    page.getByRole("button", { name: "Implementation notes", exact: true }),
  ).toBeVisible();
  const cookies = await context.cookies();
  const csrf = cookies.find((c) => c.name === "trestle-csrf")!.value;
  const path = `/api/v1/workspaces/${parent.workspace_id}/documents/${parent.id}`;
  const response = await context.request.put(path, {
    headers: { "X-CSRF-Token": csrf },
    data: { ...parent, title: "Remote edit" },
  });
  expect(response.status()).toBe(200);
  await page.getByLabel("Title", { exact: true }).fill("Local draft");
  await page
    .getByRole("button", { name: "Save document", exact: true })
    .click();
  await expect(page.getByText(/This document changed/)).toBeVisible();
  await expect(page.getByLabel("Title", { exact: true })).toHaveValue(
    "Local draft",
  );
  page.once("dialog", (d) => d.dismiss());
  await page.getByRole("link", { name: "Projects", exact: true }).click();
  await expect(page.getByLabel("Title", { exact: true })).toHaveValue(
    "Local draft",
  );
  page.once("dialog", (d) => d.accept());
  await page
    .getByRole("button", { name: "Reload saved page", exact: true })
    .click();
  await expect(page.getByLabel("Title", { exact: true })).toHaveValue(
    "Remote edit",
  );
  await page
    .getByRole("button", { name: "Revision history", exact: true })
    .click();
  page.once("dialog", (d) => d.accept());
  await page
    .getByRole("button", { name: "Load revision 1", exact: true })
    .click();
  await expect(page.getByLabel("Title", { exact: true })).toHaveValue(title);
  await page
    .getByRole("button", { name: "Save document", exact: true })
    .click();
  await expect(page.getByText("Revision 3", { exact: true })).toBeVisible();
  const download = page.waitForEvent("download");
  await page
    .getByRole("button", { name: "Export Markdown", exact: true })
    .click();
  const stream = await (await download).createReadStream();
  const chunks: Buffer[] = [];
  for await (const chunk of stream!) chunks.push(Buffer.from(chunk));
  expect(Buffer.concat(chunks).toString()).toBe(markdown);
  // Both themes and core breakpoints, plus 320px enlarged text.
  for (const theme of ["dark", "light"]) {
    await page.evaluate((t) => {
      document.documentElement.dataset.theme = t;
      document.documentElement.style.colorScheme = t;
    }, theme);
    for (const width of [390, 768, 1440]) {
      await page.setViewportSize({ width, height: 1000 });
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
      ).toBe(true);
      const results = await new AxeBuilder({ page }).analyze();
      expect(results.violations).toEqual([]);
      await page.screenshot({
        path: testInfo.outputPath(`documents-${theme}-${width}.png`),
        fullPage: true,
      });
    }
  }
  await page.setViewportSize({ width: 320, height: 900 });
  await page.evaluate(() => (document.documentElement.style.fontSize = "200%"));
  expect(
    await page.evaluate(
      () => document.documentElement.scrollWidth <= innerWidth,
    ),
  ).toBe(true);
  await page.evaluate(() => (document.documentElement.style.fontSize = ""));
  page.once("dialog", (d) => d.accept());
  await page
    .getByRole("button", { name: "Archive document", exact: true })
    .click();
  await expect(page.getByLabel("Title", { exact: true })).toBeDisabled();
  page.once("dialog", (d) => d.accept());
  await page
    .getByRole("button", { name: "Restore document", exact: true })
    .click();
  await expect(page.getByLabel("Title", { exact: true })).toBeEnabled();
  await page.reload();
  await page.getByRole("link", { name: "Documents", exact: true }).click();
  await page.getByLabel("Search titles and content").fill(title);
  await page
    .getByRole("button", { name: "Search documents", exact: true })
    .click();
  await page
    .getByRole("button", { name: `Open ${title}`, exact: true })
    .click();
  await expect(
    page.getByLabel("Document content", { exact: true }),
  ).toHaveValue(markdown);
});
