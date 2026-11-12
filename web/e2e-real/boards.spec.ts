import { expect, test } from "@playwright/test";

test("real sign-in, diagram persistence, concurrent edits and archive/restore", async ({
  page,
  context,
}) => {
  await page.goto("/#signin");
  await page.getByLabel("Username", { exact: true }).fill("boardtester");
  await page
    .getByLabel("Password", { exact: true })
    .fill("board-browser-test-only");
  await page.getByRole("button", { name: "Sign in", exact: true }).click();
  await page.getByRole("link", { name: "Boards", exact: true }).click();
  await page.getByRole("button", { name: "New diagram", exact: true }).click();
  const title = `Release flow ${Date.now()}`;
  await page.getByLabel("Board title", { exact: true }).fill(title);
  await page
    .getByRole("button", { name: "Add rectangle", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Object properties", exact: true })
    .click();
  await page.getByLabel("Object label", { exact: true }).fill("Build and test");
  const saved = page.waitForResponse(
    (r) => r.request().method() === "POST" && r.url().endsWith("/boards"),
  );
  await page.getByRole("button", { name: "Save board", exact: true }).click();
  const board = await (await saved).json();
  await expect(
    page.getByText("All changes saved", { exact: true }),
  ).toBeVisible();
  await page.reload();
  await page.getByRole("link", { name: "Boards", exact: true }).click();
  await page
    .getByRole("article")
    .filter({ has: page.getByRole("heading", { name: title, exact: true }) })
    .getByRole("button", { name: "Open board", exact: true })
    .click();
  await expect(page.locator('svg[aria-label="Board canvas"] text')).toHaveText(
    "Build and test",
  );
  // A second real API writer commits while this browser holds the old revision.
  const cookies = await context.cookies();
  const csrf = cookies.find((c) => c.name === "trestle-csrf")!.value;
  const response = await context.request.put(
    `/api/v1/workspaces/${board.workspace_id}/boards/${board.id}`,
    {
      headers: { "X-CSRF-Token": csrf },
      data: { ...board, title: `${title} from another editor` },
    },
  );
  expect(response.status()).toBe(200);
  await page
    .getByLabel("Board title", { exact: true })
    .fill(`${title} local draft`);
  await page.getByRole("button", { name: "Save board", exact: true }).click();
  await expect(page.getByText(/This board changed/)).toBeVisible();
  await expect(page.getByLabel("Board title", { exact: true })).toHaveValue(
    `${title} local draft`,
  );
  page.once("dialog", (d) => d.accept());
  await page
    .getByRole("button", { name: "Reload saved board", exact: true })
    .click();
  await expect(page.getByLabel("Board title", { exact: true })).toHaveValue(
    `${title} from another editor`,
  );
  page.once("dialog", (d) => d.accept());
  await page.getByRole("button", { name: "Export board", exact: true }).click();
  await page
    .getByRole("option", { name: "Archive board", exact: true })
    .click();
  await expect(page.getByLabel("Board title", { exact: true })).toBeDisabled();
  page.once("dialog", (d) => d.accept());
  await page.getByRole("button", { name: "Export board", exact: true }).click();
  await page
    .getByRole("option", { name: "Restore board", exact: true })
    .click();
  await expect(page.getByLabel("Board title", { exact: true })).toBeEnabled();
  await expect(page.locator('svg[aria-label="Board canvas"] text')).toHaveText(
    "Build and test",
  );
});
