import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { formatDuration, LiveTimer } from "./timer.svelte";

const serverTimer = {
  id: "timer-1",
  workspace_id: "ws-1",
  project_id: "proj-1",
  task_id: "task-1",
  task_key: "CORE-1",
  task_title: "Write documentation",
  note: "initial note",
  started_at: "2026-11-12T09:59:00Z",
};

function json(body: unknown, status = 200) {
  return new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
}

describe("formatDuration", () => {
  it("formats seconds, minutes, and hours with zero padding", () => {
    expect(formatDuration(0)).toBe("00:00:00");
    expect(formatDuration(5)).toBe("00:00:05");
    expect(formatDuration(65)).toBe("00:01:05");
    expect(formatDuration(3661)).toBe("01:01:01");
    expect(formatDuration(7200)).toBe("02:00:00");
  });
});

describe("LiveTimer", () => {
  beforeEach(() => {
    localStorage.clear();
    document.cookie = "trestle-csrf=csrf; path=/";
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-11-12T10:00:00Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
    vi.unstubAllGlobals();
    vi.restoreAllMocks();
  });

  it("loads the server timer and counts elapsed time from the server start", async () => {
    vi.stubGlobal(
      "fetch",
      vi.fn().mockResolvedValue(
        json({
          timer: serverTimer,
          server_now: "2026-11-12T10:00:00Z",
        }),
      ),
    );
    const timer = new LiveTimer();
    await timer.load();

    expect(timer.active?.id).toBe("timer-1");
    expect(timer.active?.taskHumanKey).toBe("CORE-1");
    expect(timer.active?.description).toBe("initial note");
    expect(timer.elapsedSeconds).toBe(60);
    expect(timer.formattedDuration).toBe("00:01:00");

    vi.advanceTimersByTime(120000);
    expect(timer.elapsedSeconds).toBe(180);
    expect(timer.formattedDuration).toBe("00:03:00");
  });

  it("clears the timer when the server has none", async () => {
    vi.stubGlobal(
      "fetch",
      vi
        .fn()
        .mockResolvedValue(
          json({ timer: null, server_now: new Date().toISOString() }),
        ),
    );
    const timer = new LiveTimer();
    await timer.load();
    expect(timer.active).toBeNull();
  });

  it("starts on the server and sends the task and note", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(json({ timer: serverTimer }, 201));
    vi.stubGlobal("fetch", fetchMock);
    const timer = new LiveTimer();

    const res = await timer.start(
      {
        id: "task-1",
        human_key: "CORE-1",
        title: "Write documentation",
        project_id: "proj-1",
        workspace_id: "ws-1",
      },
      "initial note",
    );

    expect(res.ok).toBe(true);
    expect(timer.active?.id).toBe("timer-1");
    expect(fetchMock).toHaveBeenCalledWith(
      "/api/v1/timer",
      expect.objectContaining({
        method: "POST",
        headers: expect.objectContaining({ "X-CSRF-Token": "csrf" }),
      }),
    );
    const body = JSON.parse(fetchMock.mock.calls[0][1].body);
    expect(body).toEqual({
      task_id: "task-1",
      note: "initial note",
      replace_active: false,
    });
  });

  it("asks before replacing another running timer", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        json(
          {
            error: { code: "timer_active" },
            timer: {
              ...serverTimer,
              task_id: "task-2",
              task_key: "CORE-2",
              task_title: "Other work",
            },
          },
          409,
        ),
      )
      .mockResolvedValueOnce(
        json({ timer: { ...serverTimer, id: "timer-2" } }, 201),
      );
    vi.stubGlobal("fetch", fetchMock);
    vi.stubGlobal("confirm", vi.fn().mockReturnValue(true));
    const timer = new LiveTimer();

    const res = await timer.start({
      id: "task-1",
      human_key: "CORE-1",
      title: "Write documentation",
      project_id: "proj-1",
      workspace_id: "ws-1",
    });

    expect(res.ok).toBe(true);
    expect(fetchMock).toHaveBeenCalledTimes(2);
    expect(JSON.parse(fetchMock.mock.calls[1][1].body).replace_active).toBe(
      true,
    );
    expect(timer.active?.id).toBe("timer-2");
  });

  it("does not start when replacing is cancelled", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValue(
        json({ error: { code: "timer_active" }, timer: serverTimer }, 409),
      );
    vi.stubGlobal("fetch", fetchMock);
    vi.stubGlobal("confirm", vi.fn().mockReturnValue(false));
    const timer = new LiveTimer();

    const res = await timer.start({
      id: "task-1",
      human_key: "CORE-1",
      title: "Write documentation",
      project_id: "proj-1",
      workspace_id: "ws-1",
    });

    expect(res.ok).toBe(false);
    expect(res.cancelled).toBe(true);
    expect(timer.active).toBeNull();
    expect(fetchMock).toHaveBeenCalledTimes(1);
  });

  it("keeps no local timer when the start request fails", async () => {
    vi.stubGlobal("fetch", vi.fn().mockRejectedValue(new Error("offline")));
    const timer = new LiveTimer();
    const res = await timer.start({
      id: "task-1",
      human_key: "CORE-1",
      title: "Write documentation",
      project_id: "proj-1",
      workspace_id: "ws-1",
    });
    expect(res.ok).toBe(false);
    expect(res.error).toContain("reach the server");
    expect(timer.active).toBeNull();
  });

  it("stops with the server timer id and never posts a second time log", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        json({ timer: serverTimer, server_now: "2026-11-12T10:00:00Z" }),
      )
      .mockResolvedValueOnce(
        json({ time_log: { id: "log-1", hours: 0.5 }, spent_hours: 0.5 }),
      );
    vi.stubGlobal("fetch", fetchMock);
    const timer = new LiveTimer();
    await timer.load();

    let logged: { taskId: string; hours: number } | null = null;
    timer.onLog((taskId, hours) => (logged = { taskId, hours }));

    vi.advanceTimersByTime(1800000);
    const res = await timer.stopAndLog("csrf");

    expect(res.ok).toBe(true);
    expect(res.spentHours).toBe(0.5);
    expect(timer.active).toBeNull();
    expect(logged).toEqual({ taskId: "task-1", hours: 0.5 });
    const stopCall = fetchMock.mock.calls[1];
    expect(stopCall[0]).toBe("/api/v1/timer/stop");
    expect(JSON.parse(stopCall[1].body)).toEqual({
      timer_id: "timer-1",
      note: "initial note",
    });
    expect(
      fetchMock.mock.calls.some(([url]) => String(url).includes("/time-logs")),
    ).toBe(false);
  });

  it("keeps the timer after a failed stop so retry can reclaim the log", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        json({ timer: serverTimer, server_now: "2026-11-12T10:00:00Z" }),
      )
      .mockResolvedValueOnce(json({ error: { message: "Try again" } }, 500));
    vi.stubGlobal("fetch", fetchMock);
    const timer = new LiveTimer();
    await timer.load();

    const res = await timer.stopAndLog("csrf");
    expect(res.ok).toBe(false);
    expect(res.error).toBe("Try again");
    expect(timer.active?.id).toBe("timer-1");
    expect(timer.saving).toBe(false);
  });

  it("only clears the timer after the server confirms a discard", async () => {
    const fetchMock = vi
      .fn()
      .mockResolvedValueOnce(
        json({ timer: serverTimer, server_now: "2026-11-12T10:00:00Z" }),
      )
      .mockResolvedValueOnce(json({ status: "discarded" }));
    vi.stubGlobal("fetch", fetchMock);
    const timer = new LiveTimer();
    await timer.load();

    const res = await timer.discard();
    expect(res.ok).toBe(true);
    expect(timer.active).toBeNull();

    const failed = new LiveTimer();
    fetchMock.mockResolvedValueOnce(
      json({ timer: serverTimer, server_now: "2026-11-12T10:00:00Z" }),
    );
    await failed.load();
    fetchMock.mockResolvedValueOnce(
      json({ error: { message: "Offline" } }, 500),
    );
    const bad = await failed.discard();
    expect(bad.ok).toBe(false);
    expect(failed.active?.id).toBe("timer-1");
  });

  it("does not import a timer saved by the previous browser-local version", () => {
    localStorage.setItem(
      "trestle.active_timer",
      JSON.stringify({ taskId: "old", startTime: Date.now() - 60000 }),
    );
    const timer = new LiveTimer();
    timer.init();
    expect(timer.legacyTimerFound).toBe(true);
    expect(timer.active).toBeNull();
    timer.dismissLegacy();
    expect(timer.legacyTimerFound).toBe(false);
    expect(localStorage.getItem("trestle.active_timer")).toBeNull();
  });
});
