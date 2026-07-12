import path from "node:path";
import { defineConfig } from "vitest/config";

export default defineConfig({
  resolve: {
    alias: {
      "@": path.resolve(__dirname, "."),
    },
  },
  test: {
    include: ["app/**/*.test.ts", "lib/**/*.test.ts", "features/**/*.test.ts"],
    setupFiles: ["./vitest.setup.ts"],
  },
});
