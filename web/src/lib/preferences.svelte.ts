// Browser-local appearance preferences. Values are allowlisted before use and
// storage failures never block the page: changes still apply to this tab.

export type Theme = "system" | "light" | "dark";
export type Accent = "iris" | "sky" | "tide" | "moss" | "ember";

export const themes: readonly { value: Theme; label: string }[] = [
  { value: "system", label: "System preference" },
  { value: "light", label: "Light" },
  { value: "dark", label: "Dark" },
];

// Hues mirror the [data-accent] presets in tokens.css.
export const accents: readonly { value: Accent; label: string; hue: number }[] =
  [
    { value: "iris", label: "Iris", hue: 284 },
    { value: "sky", label: "Sky", hue: 246 },
    { value: "tide", label: "Tide", hue: 192 },
    { value: "moss", label: "Moss", hue: 142 },
    { value: "ember", label: "Ember", hue: 52 },
  ];

const THEME_KEY = "trestle.theme";
const ACCENT_KEY = "trestle.accent";

function isTheme(value: string | null): value is Theme {
  return value === "system" || value === "light" || value === "dark";
}

function isAccent(value: string | null): value is Accent {
  return accents.some((a) => a.value === value);
}

class Preferences {
  theme = $state<Theme>("system");
  accent = $state<Accent>("iris");
  storageError = $state(false);
  saved = $state(false);

  /** Effective scheme, resolving "system" against the device setting. */
  get resolvedDark(): boolean {
    if (this.theme === "dark") return true;
    if (this.theme === "light") return false;
    return (
      typeof matchMedia === "function" &&
      matchMedia("(prefers-color-scheme: dark)").matches
    );
  }

  init() {
    this.storageError = false;
    this.saved = false;
    try {
      const storedTheme = localStorage.getItem(THEME_KEY);
      const storedAccent = localStorage.getItem(ACCENT_KEY);
      this.applyTheme(isTheme(storedTheme) ? storedTheme : "system");
      this.applyAccent(isAccent(storedAccent) ? storedAccent : "iris");
    } catch {
      this.applyTheme("system");
      this.applyAccent("iris");
      this.storageError = true;
    }
  }

  private applyTheme(value: Theme) {
    this.theme = value;
    document.documentElement.dataset.theme = value;
  }

  private applyAccent(value: Accent) {
    this.accent = value;
    if (value === "iris") delete document.documentElement.dataset.accent;
    else document.documentElement.dataset.accent = value;
  }

  private persist(key: string, value: string) {
    try {
      localStorage.setItem(key, value);
      this.storageError = false;
      this.saved = true;
    } catch {
      this.storageError = true;
      this.saved = false;
    }
  }

  setTheme(value: string) {
    if (!isTheme(value)) return;
    this.applyTheme(value);
    this.persist(THEME_KEY, value);
  }

  setAccent(value: string) {
    if (!isAccent(value)) return;
    this.applyAccent(value);
    this.persist(ACCENT_KEY, value);
  }

  toggleScheme() {
    this.setTheme(this.resolvedDark ? "light" : "dark");
  }
}

export const preferences = new Preferences();
