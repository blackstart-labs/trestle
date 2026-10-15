import { defineConfig } from "@playwright/test";

export default defineConfig({
  testDir: "./e2e",
  use: { baseURL: "http://127.0.0.1:4317", browserName: "chromium" },
  webServer: {
    command: "cargo run --locked --manifest-path ../Cargo.toml",
    url: "http://127.0.0.1:4317/health/live",
    reuseExistingServer: false,
    // Includes a cold native build when run directly from a clean checkout.
    timeout: 600000,
    env: {
      DATABASE_URL: "postgres://unavailable:unavailable@127.0.0.1:1/trestle",
      TRESTLE_BIND_ADDR: "127.0.0.1:4317",
      TRESTLE_DB_TIMEOUT_MS: "100",
    },
  },
});
