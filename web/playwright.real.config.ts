import { defineConfig } from "@playwright/test";

const database = process.env.TRESTLE_E2E_DATABASE_URL;
if (
  !database ||
  !/^\/trestle_e2e_[a-z0-9_]+$/.test(new URL(database).pathname)
) {
  throw new Error(
    "Set TRESTLE_E2E_DATABASE_URL to a disposable PostgreSQL database named trestle_e2e_*. Never use a production database.",
  );
}

export default defineConfig({
  testDir: "./e2e-real",
  globalSetup: "./e2e-real/setup.ts",
  workers: 1,
  use: { baseURL: "http://127.0.0.1:4318", browserName: "chromium" },
  webServer: {
    command: "go run ..",
    url: "http://127.0.0.1:4318/health/live",
    reuseExistingServer: false,
    timeout: 600000,
    env: {
      DATABASE_URL: database,
      TRESTLE_DEV_MODE: "true",
      TRESTLE_BIND_ADDR: "127.0.0.1:4318",
    },
  },
});
