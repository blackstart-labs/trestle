import { describe, expect, it, vi, beforeEach, afterEach } from "vitest";
import { formatDuration, LiveTimer } from "./timer.svelte";

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
    vi.useFakeTimers();
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("starts a timer and tracks elapsed seconds and hours", () => {
    const timer = new LiveTimer();
    const task = {
      id: "task-1",
      human_key: "TSK-1",
      title: "Write documentation",
      project_id: "proj-1",
      workspace_id: "ws-1",
    };

    timer.start(task, "initial note");
    expect(timer.active).toBeTruthy();
    expect(timer.active?.taskHumanKey).toBe("TSK-1");
    expect(timer.elapsedSeconds).toBe(0);

    // Fast-forward 120 seconds
    vi.advanceTimersByTime(120000);
    expect(timer.elapsedSeconds).toBe(120);
    expect(timer.formattedDuration).toBe("00:02:00");
    expect(timer.elapsedHours).toBe(0.03);

    timer.discard();
    expect(timer.active).toBeNull();
    expect(timer.elapsedSeconds).toBe(0);
  });

  it("stops and logs time with API", async () => {
    const timer = new LiveTimer();
    let listenerCalled = false;
    timer.onLog((taskId, hours) => {
      expect(taskId).toBe("task-1");
      expect(hours).toBe(0.5);
      listenerCalled = true;
    });

    const task = {
      id: "task-1",
      human_key: "TSK-1",
      title: "Write documentation",
      project_id: "proj-1",
      workspace_id: "ws-1",
    };

    timer.start(task);
    vi.advanceTimersByTime(1800000); // 30 mins

    const mockFetch = vi.fn().mockResolvedValue({
      ok: true,
      json: async () => ({ spent_hours: 0.5 }),
    });
    global.fetch = mockFetch;

    const res = await timer.stopAndLog("test-csrf");
    expect(res.ok).toBe(true);
    expect(res.spentHours).toBe(0.5);
    expect(listenerCalled).toBe(true);
    expect(timer.active).toBeNull();

    expect(mockFetch).toHaveBeenCalledWith(
      "/api/v1/workspaces/ws-1/projects/proj-1/tasks/task-1/time-logs",
      expect.objectContaining({
        method: "POST",
        headers: expect.objectContaining({
          "X-CSRF-Token": "test-csrf",
        }),
      }),
    );
  });

  it("prevents duplicate logs and keeps a failed timer for retry", async () => {
    const timer = new LiveTimer();
    timer.start({
      id: "task-1",
      human_key: "CORE-1",
      title: "Task",
      project_id: "proj",
      workspace_id: "ws",
    });
    let respond!: (value: Response) => void;
    const fetchMock = vi.fn(
      () => new Promise<Response>((resolve) => (respond = resolve)),
    );
    vi.stubGlobal("fetch", fetchMock);
    const pending = timer.stopAndLog("csrf");
    expect(timer.saving).toBe(true);
    await timer.stopAndLog("csrf");
    timer.discard();
    expect(timer.active).not.toBeNull();
    expect(fetchMock).toHaveBeenCalledTimes(1);
    respond(
      new Response(JSON.stringify({ error: { message: "Try again" } }), {
        status: 500,
      }),
    );
    expect((await pending).ok).toBe(false);
    expect(timer.saving).toBe(false);
    expect(timer.active).not.toBeNull();
    timer.discard();
  });
});
