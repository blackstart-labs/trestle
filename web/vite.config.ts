import { svelte } from "@sveltejs/vite-plugin-svelte";
import { defineConfig } from "vitest/config";

export default defineConfig({
  plugins: [svelte()],
  build: { license: { fileName: "assets/third-party-licenses.txt" } },
  server: { proxy: { "/health": "http://127.0.0.1:3000" } },
  test: {
    environment: "jsdom",
    include: ["src/**/*.test.ts"],
    setupFiles: ["src/test-setup.ts"],
    server: { deps: { inline: ["svelte"] } },
  },
  resolve: { conditions: ["browser"] },
});
