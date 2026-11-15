import { execFileSync } from "node:child_process";

export default function setup() {
  const env = {
    ...process.env,
    DATABASE_URL: process.env.TRESTLE_E2E_DATABASE_URL,
    TRESTLE_DEV_MODE: "true",
    TRESTLE_BIND_ADDR: "127.0.0.1:4318",
    TRESTLE_BOOTSTRAP_USERNAME: "boardtester",
    TRESTLE_BOOTSTRAP_PASSWORD: "board-browser-test-only",
    TRESTLE_BOOTSTRAP_WORKSPACE_NAME: "Board test studio",
    TRESTLE_BOOTSTRAP_WORKSPACE_SLUG: "board-test-studio",
  };
  execFileSync("go", ["run", "..", "migrate"], { env, stdio: "inherit" });
  execFileSync("go", ["run", "..", "bootstrap"], { env, stdio: "inherit" });
}
