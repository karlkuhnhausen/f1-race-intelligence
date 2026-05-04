import { defineConfig } from "vite";
import path from "node:path";
import { fileURLToPath } from "node:url";
import tailwindcss from "@tailwindcss/vite";

const __dirname = path.dirname(fileURLToPath(import.meta.url));

export default defineConfig({
  plugins: [tailwindcss()],
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "./src")
    }
  },
  server: {
    port: 5173,
    proxy: process.env.VITE_BACKEND_PROXY
      ? { '/api': { target: process.env.VITE_BACKEND_PROXY, changeOrigin: true } }
      : undefined,
  },
  test: {
    environment: "jsdom",
    globals: true,
    pool: "threads",
    css: false,
    testTimeout: 30000,
    hookTimeout: 30000,
  }
});
