import { defineConfig } from "vite";

export default defineConfig({
  server: {
    port: 5173
  },
  test: {
    environment: "jsdom",
    globals: true,
    pool: "threads",
    testTimeout: 30000,
    hookTimeout: 30000,
  }
});
