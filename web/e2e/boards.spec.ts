import { expect, test, type Page, type Download } from "@playwright/test";
import AxeBuilder from "@axe-core/playwright";
import type { Board, Scene } from "../src/boards/scene";

async function content(download: Download) {
  const stream = await download.createReadStream();
  const chunks: Buffer[] = [];
  for await (const chunk of stream!) chunks.push(Buffer.from(chunk));
  return Buffer.concat(chunks);
}
async function setup(page: Page, role = "owner") {
  let boards: Board[] = [];
  let fail = false;
  let conflict = false;
  let failList = false;
  const writes: Record<string, unknown>[] = [];
  await page
    .context()
    .addCookies([
      { name: "trestle-csrf", value: "csrf", domain: "127.0.0.1", path: "/" },
    ]);
  await page.route("**/api/v1/**", async (route) => {
    const req = route.request(),
      url = new URL(req.url()),
      path = url.pathname;
    const json = (body: object, status = 200) =>
      route.fulfill({ status, json: body });
    if (path.endsWith("/me"))
      return json({ id: "owner", username: "owner", display_name: "Owner" });
    if (path.endsWith("/workspaces"))
      return json({
        workspaces: [
          { id: "ws", name: "Engineering", slug: "engineering", role },
        ],
      });
    if (path.endsWith("/boards") && req.method() === "GET")
      return failList
        ? json({ error: { message: "Could not load boards. Try again." } }, 503)
        : json({
            boards: boards.filter(
              (b) =>
                b.archived === (url.searchParams.get("archived") === "true"),
            ),
            has_more: false,
          });
    if (path.includes("/boards") && ["POST", "PUT"].includes(req.method())) {
      expect(req.headers()["x-csrf-token"]).toBe("csrf");
      const body = req.postDataJSON();
      writes.push(body);
      if (fail)
        return json(
          { error: { message: "Could not save board. Try again." } },
          503,
        );
      if (conflict)
        return json(
          {
            error: {
              message:
                "This board changed. Export your draft, then reload the latest version before editing again.",
            },
          },
          409,
        );
      const board: Board = {
        ...body,
        id: "board",
        workspace_id: "ws",
        version: (body.version || 0) + 1,
        updated_at: "2026-11-11T12:00:00Z",
      };
      boards = [board];
      return json(board, req.method() === "POST" ? 201 : 200);
    }
    if (path.endsWith("/boards/board")) return json(boards[0]);
    if (path.endsWith("/projects")) return json({ projects: [] });
    return json({});
  });
  await page.goto("/");
  await page.getByRole("link", { name: "Boards", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Boards", exact: true }),
  ).toBeVisible();
  return {
    writes,
    fail: (v: boolean) => (fail = v),
    conflict: (v: boolean) => (conflict = v),
    failList: (v: boolean) => (failList = v),
    seed: (scene: Scene) =>
      (boards = [
        {
          id: "board",
          workspace_id: "ws",
          title: "Shared flow",
          kind: "diagram",
          version: 1,
          archived: false,
          updated_at: "2026-11-11T12:00:00Z",
          scene,
        },
      ]),
  };
}
async function properties(page: Page) {
  const button = page.getByRole("button", {
    name: "Object properties",
    exact: true,
  });
  if ((await button.getAttribute("aria-expanded")) !== "true")
    await button.click();
}
async function exportFile(page: Page, label: string) {
  const pending = page.waitForEvent("download");
  await page.getByRole("button", { name: "Export board", exact: true }).click();
  await page
    .getByRole("listbox", { name: "Export board", exact: true })
    .getByRole("option", { name: label, exact: true })
    .click();
  return content(await pending);
}

test("diagrams preserve failures, connect shapes, undo, and round-trip complete JSON/SVG/PNG", async ({
  page,
}) => {
  const api = await setup(page);
  await page.getByRole("button", { name: "New diagram", exact: true }).click();
  await page.getByLabel("Board title", { exact: true }).fill("Release process");
  await page
    .getByRole("button", { name: "Add rectangle", exact: true })
    .click();
  await properties(page);
  await page.getByLabel("Object label", { exact: true }).fill("Review");
  await page.getByLabel("Position X", { exact: true }).fill("100");
  await page.getByLabel("Position X", { exact: true }).press("Tab");
  await page.getByRole("button", { name: "Add diamond", exact: true }).click();
  await page.getByLabel("Object label", { exact: true }).fill("Approved?");
  await page.getByLabel("Position X", { exact: true }).fill("600");
  await page.getByLabel("Position X", { exact: true }).press("Tab");
  await page.getByRole("button", { name: "Arrow from", exact: true }).click();
  await page
    .getByRole("listbox", { name: "Arrow from", exact: true })
    .getByRole("option", { name: "Review", exact: true })
    .click();
  await page.getByRole("button", { name: "Arrow to", exact: true }).click();
  await page
    .getByRole("listbox", { name: "Arrow to", exact: true })
    .getByRole("option", { name: "Approved?", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Connect with arrow", exact: true })
    .click();
  await expect(page.locator("g.connector")).toHaveCount(1);
  await page.getByRole("button", { name: "Undo", exact: true }).click();
  await expect(page.locator("g.connector")).toHaveCount(0);
  await page.getByRole("button", { name: "Redo", exact: true }).click();
  api.fail(true);
  await page.getByRole("button", { name: "Save board", exact: true }).click();
  await expect(
    page.getByText("Could not save board. Try again.", { exact: true }),
  ).toBeVisible();
  await expect(page.getByLabel("Board title", { exact: true })).toHaveValue(
    "Release process",
  );
  api.fail(false);
  await page.getByRole("button", { name: "Save board", exact: true }).click();
  await expect(
    page.getByText("All changes saved", { exact: true }),
  ).toBeVisible();
  expect(api.writes.at(-1)?.creation_key).toBe(api.writes.at(-2)?.creation_key);
  const portable = await exportFile(page, "Export JSON draft");
  expect(JSON.parse(portable.toString()).scene.elements).toHaveLength(3);
  const svg = (await exportFile(page, "Export SVG")).toString();
  expect(svg).toContain("Review");
  expect(svg).not.toContain("data-editor-only");
  expect(svg).not.toContain("127.0.0.1");
  expect(
    await page.evaluate(
      (source) =>
        new DOMParser()
          .parseFromString(source, "image/svg+xml")
          .querySelector("parsererror")?.textContent || "",
      svg,
    ),
  ).toBe("");
  const png = await exportFile(page, "Export PNG");
  expect([...png.subarray(0, 8)]).toEqual([137, 80, 78, 71, 13, 10, 26, 10]);
  await page
    .getByRole("button", { name: "Back to boards", exact: true })
    .click();
  await page.getByLabel("Import board", { exact: true }).setInputFiles({
    name: "board.json",
    mimeType: "application/json",
    buffer: portable,
  });
  await expect(page.getByLabel("Board title", { exact: true })).toHaveValue(
    "Release process",
  );
  await expect(page.locator("g.connector")).toHaveCount(1);
});

test("direct manipulation supports resize, multi-select, group, duplicate, keyboard moves and inline text", async ({
  page,
}) => {
  const api = await setup(page);
  await page.getByRole("button", { name: "New diagram", exact: true }).click();
  await page
    .getByRole("button", { name: "Add rectangle", exact: true })
    .click();
  const rect = page.locator("g[data-object].shape > rect").first();
  const oldWidth = Number(await rect.getAttribute("width"));
  const handle = page.locator('[data-handle="se"]');
  const r = (await handle.boundingBox())!;
  await page.mouse.move(r.x + r.width / 2, r.y + r.height / 2);
  await page.mouse.down();
  await page.mouse.move(r.x + r.width / 2 + 60, r.y + r.height / 2 + 40, {
    steps: 5,
  });
  await page.mouse.up();
  expect(Number(await rect.getAttribute("width"))).toBeGreaterThan(oldWidth);
  await rect.dblclick();
  await page
    .getByLabel("Edit shape text", { exact: true })
    .fill("Review the design");
  await page
    .getByLabel("Edit shape text", { exact: true })
    .press("Control+Enter");
  await expect(page.locator("g[data-object] text")).toHaveText(
    "Review the design",
  );
  await page.getByRole("button", { name: "Add ellipse", exact: true }).click();
  const canvas = page.getByRole("application", { name: "Board canvas" });
  await canvas.focus();
  await page.keyboard.press("Control+a");
  await page.keyboard.press("Control+g");
  await page.keyboard.press("Control+d");
  await expect(page.locator("g.shape")).toHaveCount(4);
  await page.keyboard.press("ArrowRight");
  await page.keyboard.press("Control+s");
  await expect(
    page.getByText("All changes saved", { exact: true }),
  ).toBeVisible();
  const elements = (api.writes.at(-1)?.scene as Scene).elements;
  expect(elements.every((e) => !!e.group)).toBe(true);
  expect(new Set(elements.map((e) => e.group)).size).toBe(2);
});

test("whiteboards support pointer and precision drawing, guarded navigation and conflict recovery", async ({
  page,
}) => {
  const api = await setup(page);
  await page
    .getByRole("button", { name: "New whiteboard", exact: true })
    .click();
  await properties(page);
  await page.getByText("Draw with coordinates", { exact: true }).click();
  await page
    .getByLabel("Stroke points", { exact: true })
    .fill("100,100 200,160 300,100");
  await page
    .getByRole("button", { name: "Add stroke from points", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Draw freehand", exact: true })
    .click();
  const canvas = page.getByRole("application", { name: "Board canvas" }),
    box = (await canvas.boundingBox())!;
  await page.mouse.move(box.x + 80, box.y + 80);
  await page.mouse.down();
  await page.mouse.move(box.x + 200, box.y + 140, { steps: 8 });
  await page.mouse.up();
  await expect(canvas.locator("polyline")).toHaveCount(2);
  page.once("dialog", (d) => d.dismiss());
  await page
    .getByRole("button", { name: "Back to boards", exact: true })
    .click();
  await expect(
    page.getByRole("region", { name: "Whiteboard editor", exact: true }),
  ).toBeVisible();
  await page.keyboard.press("Control+k");
  await page.getByRole("combobox").fill("All projects");
  page.once("dialog", (d) => d.dismiss());
  await page.getByRole("combobox").press("Enter");
  await expect(
    page.getByRole("region", { name: "Whiteboard editor", exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Save board", exact: true }).click();
  await expect(
    page.getByText("All changes saved", { exact: true }),
  ).toBeVisible();
  await page.getByLabel("Board title", { exact: true }).fill("My draft");
  api.conflict(true);
  await page.getByRole("button", { name: "Save board", exact: true }).click();
  await expect(page.getByText(/This board changed/)).toBeVisible();
  await expect(page.getByLabel("Board title", { exact: true })).toHaveValue(
    "My draft",
  );
  page.once("dialog", (d) => d.accept());
  await page
    .getByRole("button", { name: "Reload saved board", exact: true })
    .click();
  await expect(page.getByLabel("Board title", { exact: true })).toHaveValue(
    "Untitled whiteboard",
  );
});

test("guest boards are read-only", async ({ page }) => {
  const api = await setup(page, "guest");
  api.seed({ version: 1, elements: [] });
  await page.getByRole("link", { name: "Projects", exact: true }).click();
  await page.getByRole("link", { name: "Boards", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "New diagram", exact: true }),
  ).toHaveCount(0);
  await page.getByRole("button", { name: "Open board", exact: true }).click();
  await expect(page.getByLabel("Board title", { exact: true })).toBeDisabled();
  await expect(
    page.getByRole("button", { name: "Add rectangle", exact: true }),
  ).toBeDisabled();
  await expect(
    page.getByRole("button", { name: "Save board", exact: true }),
  ).toHaveCount(0);
  expect(api.writes).toHaveLength(0);
});

test("load errors retry and page filters have an empty state", async ({
  page,
}) => {
  const api = await setup(page);
  await expect(
    page.getByRole("heading", { name: "No boards yet", exact: true }),
  ).toBeVisible();
  api.seed({ version: 1, elements: [] });
  api.failList(true);
  await page.getByRole("link", { name: "Projects", exact: true }).click();
  await page.getByRole("link", { name: "Boards", exact: true }).click();
  await expect(
    page.getByText("Could not load boards. Try again.", { exact: true }),
  ).toBeVisible();
  api.failList(false);
  await page.getByRole("button", { name: "Retry boards", exact: true }).click();
  await expect(
    page.getByRole("heading", { name: "Shared flow", exact: true }),
  ).toBeVisible();
  await page
    .getByLabel("Filter this page", { exact: true })
    .fill("no such board");
  await expect(
    page.getByRole("heading", { name: "No matching boards", exact: true }),
  ).toBeVisible();
  await page.getByRole("button", { name: "Clear filter", exact: true }).click();
  await expect(
    page.getByRole("button", { name: "Open board", exact: true }),
  ).toBeVisible();
});

test.describe("touch boards", () => {
  test.use({
    hasTouch: true,
    isMobile: true,
    viewport: { width: 390, height: 844 },
  });
  test("draws a freehand stroke by touch", async ({ page }) => {
    await setup(page);
    await page
      .getByRole("button", { name: "New whiteboard", exact: true })
      .tap();
    await page
      .getByRole("button", { name: "Draw freehand", exact: true })
      .tap();
    const canvas = page.getByRole("application", { name: "Board canvas" }),
      box = (await canvas.boundingBox())!;
    const cdp = await page.context().newCDPSession(page);
    await cdp.send("Input.dispatchTouchEvent", {
      type: "touchStart",
      touchPoints: [{ x: box.x + 80, y: box.y + 80 }],
    });
    await cdp.send("Input.dispatchTouchEvent", {
      type: "touchMove",
      touchPoints: [{ x: box.x + 180, y: box.y + 130 }],
    });
    await cdp.send("Input.dispatchTouchEvent", {
      type: "touchEnd",
      touchPoints: [],
    });
    await expect(canvas.locator("polyline")).toHaveCount(1);
    await expect(
      page.getByRole("button", { name: "Undo", exact: true }),
    ).toBeEnabled();
    await cdp.detach();
  });
  test("creates and saves shapes by touch", async ({ page }) => {
    const api = await setup(page);
    await page
      .getByRole("button", { name: "New whiteboard", exact: true })
      .tap();
    await page
      .getByRole("button", { name: "Shapes and layers", exact: true })
      .tap();
    await page
      .getByRole("button", { name: "Add rectangle", exact: true })
      .tap();
    await page.getByRole("button", { name: "Save board", exact: true }).tap();
    await expect(
      page.getByText("All changes saved", { exact: true }),
    ).toBeVisible();
    expect((api.writes[0].scene as Scene).elements[0].kind).toBe("rectangle");
  });
});

test("canvas-first flowchart workspace is accessible and reflows in both themes", async ({
  page,
}, testInfo) => {
  await setup(page);
  await page.getByRole("button", { name: "New diagram", exact: true }).click();
  await page
    .getByRole("button", { name: "Start from template", exact: true })
    .click();
  await page
    .getByRole("button", { name: "Three-lane process", exact: true })
    .click();
  await expect(page.locator("g.connector")).toHaveCount(4);
  for (const theme of ["light", "dark"]) {
    await page.evaluate(
      (t) => (document.documentElement.dataset.theme = t),
      theme,
    );
    for (const width of [1440, 768, 390]) {
      await page.setViewportSize({ width, height: 1000 });
      const library = page.getByRole("button", {
        name: "Shapes and layers",
        exact: true,
      });
      if (
        ((await library.getAttribute("aria-expanded")) === "true") !==
        width > 960
      )
        await library.click();
      await page
        .getByRole("button", { name: "Fit drawing", exact: true })
        .click();
      expect(
        await page.evaluate(
          () => document.documentElement.scrollWidth <= innerWidth,
        ),
      ).toBe(true);
      const box = (await page
        .getByRole("application", { name: "Board canvas" })
        .boundingBox())!;
      expect(box.width).toBeGreaterThan(width * 0.6);
      expect(box.height).toBeGreaterThan(500);
      const results = await new AxeBuilder({ page })
        .withTags(["wcag2a", "wcag2aa", "wcag21aa", "wcag22aa"])
        .analyze();
      expect(results.violations).toEqual([]);
      await page.screenshot({
        path: testInfo.outputPath(`boards-${theme}-${width}.png`),
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
});
