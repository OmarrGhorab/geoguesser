import path from "node:path";
import { defineConfig } from "vitest/config";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  test: {
    environment: "jsdom",
    globals: true,
    include: ["features/rooms/**/*.test.{ts,tsx}", "features/profile/**/*.test.{ts,tsx}"],
    passWithNoTests: true,
    setupFiles: ["./test/setup.ts"],
  },
  resolve: {
    alias: {
      "server-only": path.resolve(__dirname, "./test/server-only-stub.ts"),
      "@": path.resolve(__dirname, "."),
    },
  },
});
