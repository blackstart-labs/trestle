import { expect, test } from "@playwright/test";

test("real timer survives reload and a lost stop response does not duplicate time", async ({
  page,
  context,
}) => {
  await page.goto("/#signin");
  await page.getByLabel("Username", { exact: true }).fill("boardtester");
  await page
    .getByLabel("Password", { exact: true })
    .fill("board-browser-test-only");
  await page.getByRole("button", { name: "Sign in", exact: true }).click();
  await expect(
    page.getByRole("link", { name: "Boards", exact: true }),
  ).toBeVisible();

  const cookies = await context.cookies();
  const csrf = cookies.find((c) => c.name === "trestle-csrf")!.value;
  const headers = { "X-CSRF-Token": csrf };

  // A previous run may have left a timer on this disposable database.
  const stale = await (await context.request.get("/api/v1/timer")).json();
  if (stale.timer) {
    const discarded = await context.request.post("/api/v1/timer/discard", {
      headers,
      data: { timer_id: stale.timer.id },
    });
    expect(discarded.status()).toBe(200);
    await page.reload();
    await expect(
      page.getByRole("link", { name: "Boards", exact: true }),
    ).toBeVisible();
  }

  const workspaces = await (
    await context.request.get("/api/v1/workspaces")
  ).json();
  const workspace = workspaces.workspaces[0];
  const projectResponse = await context.request.post(
    `/api/v1/workspaces/${workspace.id}/projects`,
    {
      headers,
      data: {
        name: `Timer test ${Date.now()}`,
        slug: `timer-${Date.now()}`,
        key: `T${String(Date.now()).slice(-6)}`,
      },
    },
  );
  expect(projectResponse.status()).toBe(201);
  const project = await projectResponse.json();
  const taskResponse = await context.request.post(
    `/api/v1/workspaces/${workspace.id}/projects/${project.id}/tasks`,
    { headers, data: { title: "Track this task" } },
  );
  expect(taskResponse.status()).toBe(201);
  const task = await taskResponse.json();

  await page.goto(`/#task/${workspace.id}/${project.id}/${task.id}`);
  await page
    .getByRole("dialog")
    .getByRole("button", { name: "Start live timer" })
    .click();
  await expect(
    page.getByRole("button", { name: "Stop timer", exact: true }),
  ).toBeVisible();

  // The server, not the tab, remembers the running timer.
  await page.reload();
  await expect(
    page.getByRole("button", { name: "Stop timer", exact: true }),
  ).toBeVisible();

  const active = await (await context.request.get("/api/v1/timer")).json();
  expect(active.timer?.task_id).toBe(task.id);

  // Simulate a stop whose response never reached this tab. The UI still shows
  // the timer; stopping again must return the same log, not a second one.
  const firstStop = await context.request.post("/api/v1/timer/stop", {
    headers,
    data: { timer_id: active.timer.id },
  });
  expect(firstStop.status()).toBe(200);
  const firstLog = await firstStop.json();

  await page
    .getByRole("dialog")
    .getByRole("button", { name: "Stop & Log time" })
    .click();
  await expect(
    page.getByRole("button", { name: "Stop timer", exact: true }),
  ).toHaveCount(0);

  const logs = await (
    await context.request.get(
      `/api/v1/workspaces/${workspace.id}/projects/${project.id}/tasks/${task.id}/time-logs`,
    )
  ).json();
  expect(logs.time_logs).toHaveLength(1);
  expect(logs.time_logs[0].id).toBe(firstLog.time_log.id);
  expect(logs.time_logs[0].hours).toBeGreaterThan(0);
});
