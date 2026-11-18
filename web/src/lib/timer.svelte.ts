// Server-authoritative running timer.
// The server owns the start time, one-active-timer rule and the final log, so
// lost responses, other tabs and other devices cannot duplicate or lose time.
// A timer saved by the earlier browser-local version is detected but never
// imported silently.

export interface ActiveTimerData {
  id: string;
  taskId: string;
  taskHumanKey: string;
  taskTitle: string;
  projectId: string;
  workspaceId: string;
  startTime: number;
  pausedAt?: number | null;
  pausedSeconds?: number;
  description: string;
}

export interface TimerResult {
  ok: boolean;
  error?: string;
  spentHours?: number;
  cancelled?: boolean;
}

const TIMER_KEY = "trestle.active_timer";
const CHANNEL = "trestle.timer";

export function formatDuration(seconds: number): string {
  const total = Math.max(0, Math.floor(seconds));
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = total % 60;
  const hh = h.toString().padStart(2, "0");
  const mm = m.toString().padStart(2, "0");
  const ss = s.toString().padStart(2, "0");
  return `${hh}:${mm}:${ss}`;
}

function readCookie(name: string): string {
  if (typeof document === "undefined") return "";
  for (const part of document.cookie.split(";")) {
    const trimmed = part.trim();
    if (trimmed.startsWith(`${name}=`)) return trimmed.slice(name.length + 1);
  }
  return "";
}

async function errorMessage(res: Response, fallback: string): Promise<string> {
  const body = await res.json().catch(() => null);
  const message = body?.error?.message;
  return typeof message === "string" && message ? message : fallback;
}

export class LiveTimer {
  active = $state<ActiveTimerData | null>(null);
  saving = $state(false);
  loading = $state(false);
  legacyTimerFound = $state(false);
  now = $state<number>(Date.now());
  private intervalId: number | null = null;
  private initialized = false;
  private serverOffset = 0;
  private noteTimer: number | null = null;
  private listeners: ((taskId: string, spentHours: number) => void)[] = [];
  private channel: BroadcastChannel | null = null;
  private onWake = () => {
    if (
      typeof document !== "undefined" &&
      document.visibilityState === "hidden"
    )
      return;
    void this.load();
  };

  get elapsedSeconds(): number {
    if (!this.active) return 0;
    return Math.max(
      0,
      Math.floor(
        ((this.active.pausedAt ?? this.now) - this.active.startTime) / 1000 -
          (this.active.pausedSeconds || 0),
      ),
    );
  }

  get formattedDuration(): string {
    return formatDuration(this.elapsedSeconds);
  }

  init() {
    if (this.initialized) return;
    this.initialized = true;
    try {
      this.legacyTimerFound = localStorage.getItem(TIMER_KEY) !== null;
    } catch {
      this.legacyTimerFound = false;
    }
    if (typeof window !== "undefined") {
      window.addEventListener("focus", this.onWake);
      document.addEventListener("visibilitychange", this.onWake);
      if (typeof BroadcastChannel !== "undefined") {
        this.channel = new BroadcastChannel(CHANNEL);
        this.channel.onmessage = () => void this.load();
      }
    }
  }

  // Reads the server's timer. A transient failure keeps the current state;
  // an ended session clears it.
  async load(): Promise<void> {
    if (this.loading || this.saving) return;
    this.loading = true;
    try {
      const res = await fetch("/api/v1/timer", {
        headers: { Accept: "application/json" },
        cache: "no-store",
      });
      if (res.status === 401) {
        this.applyTimer(null);
        return;
      }
      if (!res.ok) return;
      const data = await res.json();
      const serverNow = data?.server_now ? Date.parse(data.server_now) : NaN;
      this.serverOffset = Number.isFinite(serverNow)
        ? Date.now() - serverNow
        : 0;
      this.applyTimer(data?.timer ?? null);
    } catch {
      // Offline: keep the last known timer and let the ticker continue.
    } finally {
      this.loading = false;
    }
  }

  dismissLegacy() {
    this.legacyTimerFound = false;
    try {
      localStorage.removeItem(TIMER_KEY);
    } catch {}
  }

  private applyTimer(timer: any) {
    if (!timer?.id || !timer.task_id || !timer.started_at) {
      this.clearActive();
      return;
    }
    const started = Date.parse(timer.started_at);
    this.active = {
      id: timer.id,
      taskId: timer.task_id,
      taskHumanKey: timer.task_key || "",
      taskTitle: timer.task_title || "",
      projectId: timer.project_id,
      workspaceId: timer.workspace_id,
      startTime:
        (Number.isFinite(started) ? started : Date.now()) + this.serverOffset,
      description: timer.note || "",
      pausedAt: timer.paused_at
        ? Date.parse(timer.paused_at) + this.serverOffset
        : null,
      pausedSeconds: Number(timer.paused_seconds) || 0,
    };
    this.startTicker();
  }

  private clearActive() {
    this.cancelNote();
    this.active = null;
    this.stopTicker();
  }

  private startTicker() {
    if (this.intervalId !== null || typeof window === "undefined") return;
    this.now = Date.now();
    this.intervalId = window.setInterval(() => {
      this.now = Date.now();
    }, 1000);
  }

  private stopTicker() {
    if (this.intervalId !== null) {
      clearInterval(this.intervalId);
      this.intervalId = null;
    }
  }

  private notify() {
    this.channel?.postMessage({ type: "changed" });
  }

  private cancelNote() {
    if (this.noteTimer !== null) {
      clearTimeout(this.noteTimer);
      this.noteTimer = null;
    }
  }

  async start(
    task: {
      id: string;
      human_key: string;
      title: string;
      project_id: string;
      workspace_id: string;
    },
    description = "",
  ): Promise<TimerResult> {
    if (this.saving) return { ok: false, error: "The timer is busy saving." };
    if (this.active?.taskId === task.id) return { ok: true };
    const csrf = readCookie("trestle-csrf");
    if (!csrf) {
      return {
        ok: false,
        error: "Your session expired. Reload the page and sign in again.",
      };
    }
    this.saving = true;
    try {
      let res = await this.postStart(task.id, description, csrf, false);
      if (res.status === 409) {
        const body = await res.json().catch(() => null);
        const running = body?.timer;
        const label = running?.task_key
          ? `${running.task_key}: ${running.task_title || "another task"}`
          : "another task";
        if (
          !confirm(
            `A timer is already running on ${label}. Starting this one will discard its unlogged time. Continue?`,
          )
        ) {
          return { ok: false, cancelled: true };
        }
        res = await this.postStart(task.id, description, csrf, true);
      }
      if (!res.ok) {
        return {
          ok: false,
          error: await errorMessage(res, "Could not start the timer."),
        };
      }
      const data = await res.json();
      this.applyTimer(data?.timer ?? null);
      this.notify();
      return { ok: true };
    } catch {
      return {
        ok: false,
        error:
          "Could not reach the server. Check your connection and try again.",
      };
    } finally {
      this.saving = false;
    }
  }

  private postStart(
    taskId: string,
    note: string,
    csrf: string,
    replaceActive: boolean,
  ) {
    return fetch("/api/v1/timer", {
      method: "POST",
      headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf },
      body: JSON.stringify({
        task_id: taskId,
        note,
        replace_active: replaceActive,
      }),
    });
  }

  setDescription(desc: string) {
    const current = this.active;
    if (!current) return;
    current.description = desc;
    this.cancelNote();
    const csrf = readCookie("trestle-csrf");
    if (!csrf) return;
    const timerId = current.id;
    this.noteTimer = window.setTimeout(() => {
      this.noteTimer = null;
      if (this.active?.id !== timerId) return;
      void fetch("/api/v1/timer", {
        method: "PATCH",
        headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf },
        body: JSON.stringify({ timer_id: timerId, note: desc }),
      }).catch(() => {
        // The note is retried with the final stop request.
      });
    }, 600);
  }

  async discard(): Promise<TimerResult> {
    if (!this.active) return { ok: false };
    if (this.saving)
      return { ok: false, error: "The timer is busy saving. Try again." };
    const csrf = readCookie("trestle-csrf");
    if (!csrf) {
      return {
        ok: false,
        error: "Your session expired. Reload the page and sign in again.",
      };
    }
    const timerId = this.active.id;
    this.saving = true;
    try {
      const res = await fetch("/api/v1/timer/discard", {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf },
        body: JSON.stringify({ timer_id: timerId }),
      });
      if (!res.ok) {
        return {
          ok: false,
          error: await errorMessage(res, "Could not discard the timer."),
        };
      }
      this.clearActive();
      this.notify();
      return { ok: true };
    } catch {
      return {
        ok: false,
        error: "Could not reach the server. The timer is still running.",
      };
    } finally {
      this.saving = false;
    }
  }

  onLog(fn: (taskId: string, spentHours: number) => void): () => void {
    this.listeners.push(fn);
    return () => {
      this.listeners = this.listeners.filter((l) => l !== fn);
    };
  }

  async setPaused(paused: boolean): Promise<TimerResult> {
    if (!this.active || this.saving) return { ok: false };
    this.saving = true;
    try {
      const res = await fetch("/api/v1/timer/pause", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          "X-CSRF-Token": readCookie("trestle-csrf"),
        },
        body: JSON.stringify({ timer_id: this.active.id, paused }),
      });
      if (!res.ok)
        return {
          ok: false,
          error: await errorMessage(
            res,
            "Could not change timer state. Retry.",
          ),
        };
      const data = await res.json();
      this.serverOffset = Date.now() - Date.parse(data.server_now);
      this.applyTimer(data.timer);
      this.notify();
      return { ok: true };
    } catch {
      return {
        ok: false,
        error:
          "Could not reach the server. Reload timer state before retrying.",
      };
    } finally {
      this.saving = false;
    }
  }

  async stopAndLog(csrfToken: string): Promise<TimerResult> {
    if (!this.active) return { ok: false };
    if (this.saving) {
      return { ok: false, error: "The timer is already being saved." };
    }
    this.saving = true;
    this.now = Date.now();
    const task = this.active;
    const note = task.description || "";
    const csrf = csrfToken || readCookie("trestle-csrf");

    try {
      const res = await fetch("/api/v1/timer/stop", {
        method: "POST",
        headers: { "Content-Type": "application/json", "X-CSRF-Token": csrf },
        body: JSON.stringify({ timer_id: task.id, note }),
      });
      if (!res.ok) {
        return {
          ok: false,
          error: await errorMessage(res, "Could not log time."),
        };
      }
      const data = await res.json();
      const spent =
        typeof data?.spent_hours === "number"
          ? data.spent_hours
          : (data?.time_log?.hours ?? 0);
      this.saving = false;
      this.clearActive();
      this.notify();
      for (const listener of this.listeners) listener(task.taskId, spent);
      return { ok: true, spentHours: spent };
    } catch {
      return {
        ok: false,
        error:
          "Could not log time. The timer is still running, so retrying will not duplicate the entry.",
      };
    } finally {
      this.saving = false;
    }
  }
}

export const liveTimer = new LiveTimer();
