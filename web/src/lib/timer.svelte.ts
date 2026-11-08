// Live time tracking store for active task timers.
// State persists across page refreshes and route changes in localStorage.

export interface ActiveTimerData {
  taskId: string;
  taskHumanKey: string;
  taskTitle: string;
  projectId: string;
  workspaceId: string;
  startTime: number;
  description: string;
}

const TIMER_KEY = "trestle.active_timer";

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

export class LiveTimer {
  active = $state<ActiveTimerData | null>(null);
  now = $state<number>(Date.now());
  private intervalId: number | null = null;
  private listeners: ((taskId: string, spentHours: number) => void)[] = [];

  get elapsedSeconds(): number {
    if (!this.active) return 0;
    return Math.max(0, Math.floor((this.now - this.active.startTime) / 1000));
  }

  get formattedDuration(): string {
    return formatDuration(this.elapsedSeconds);
  }

  get elapsedHours(): number {
    if (!this.active) return 0;
    const ms = Math.max(1000, this.now - this.active.startTime);
    const hours = ms / 3600000;
    return Math.max(0.01, Math.round(hours * 100) / 100);
  }

  init() {
    try {
      const stored = localStorage.getItem(TIMER_KEY);
      if (stored) {
        const data = JSON.parse(stored) as ActiveTimerData;
        if (data && data.taskId && data.startTime) {
          this.active = data;
          this.startTicker();
        }
      }
    } catch {
      // Storage unavailable or invalid JSON; ignore.
    }
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

  start(
    task: {
      id: string;
      human_key: string;
      title: string;
      project_id: string;
      workspace_id: string;
    },
    description = "",
  ) {
    this.active = {
      taskId: task.id,
      taskHumanKey: task.human_key,
      taskTitle: task.title,
      projectId: task.project_id,
      workspaceId: task.workspace_id,
      startTime: Date.now(),
      description,
    };
    try {
      localStorage.setItem(TIMER_KEY, JSON.stringify(this.active));
    } catch {}
    this.startTicker();
  }

  setDescription(desc: string) {
    if (this.active) {
      this.active.description = desc;
      try {
        localStorage.setItem(TIMER_KEY, JSON.stringify(this.active));
      } catch {}
    }
  }

  discard() {
    this.active = null;
    this.stopTicker();
    try {
      localStorage.removeItem(TIMER_KEY);
    } catch {}
  }

  onLog(fn: (taskId: string, spentHours: number) => void): () => void {
    this.listeners.push(fn);
    return () => {
      this.listeners = this.listeners.filter((l) => l !== fn);
    };
  }

  async stopAndLog(
    csrfToken: string,
  ): Promise<{ ok: boolean; spentHours: number; error?: string }> {
    if (!this.active) return { ok: false, spentHours: 0 };
    const task = this.active;
    const hours = this.elapsedHours;
    const desc = task.description || "";
    const today = new Date().toISOString().slice(0, 10);

    try {
      const res = await fetch(
        `/api/v1/workspaces/${task.workspaceId}/projects/${task.projectId}/tasks/${task.taskId}/time-logs`,
        {
          method: "POST",
          headers: {
            "Content-Type": "application/json",
            "X-CSRF-Token": csrfToken,
          },
          body: JSON.stringify({
            hours,
            logged_date: today,
            description: desc,
          }),
        },
      );
      if (!res.ok) {
        const body = await res.json().catch(() => ({}));
        return {
          ok: false,
          spentHours: 0,
          error: body.error?.message || "Failed to log time",
        };
      }
      const data = await res.json();
      const spent =
        typeof data.spent_hours === "number" ? data.spent_hours : hours;
      this.discard();
      for (const listener of this.listeners) {
        listener(task.taskId, spent);
      }
      return { ok: true, spentHours: spent };
    } catch (e: any) {
      return {
        ok: false,
        spentHours: 0,
        error: e?.message || "Network error while logging time",
      };
    }
  }
}

export const liveTimer = new LiveTimer();
