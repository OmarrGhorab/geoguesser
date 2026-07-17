import { defineConfig, devices } from "@playwright/test";

const isCI = Boolean(process.env.CI);
const e2eBackendURL = "http://127.0.0.1:8081/api/v1";

const nextWebServer = {
  command: isCI ? "pnpm build && pnpm start" : "pnpm dev",
  url: "http://127.0.0.1:3000",
  reuseExistingServer: !isCI,
  timeout: 120_000,
  ...(isCI ? { env: { BACKEND_API_URL: e2eBackendURL } } : {}),
};

export default defineConfig({
  testDir: "./tests/e2e",
  fullyParallel: true,
  forbidOnly: Boolean(process.env.CI),
  retries: process.env.CI ? 2 : 0,
  // The App Router dev server compiles auth routes on demand. Keep local runs
  // aligned with CI so parallel first-load navigations do not abort each other.
  workers: 2,
  reporter: "list",
  use: {
    baseURL: "http://127.0.0.1:3000",
    trace: "on-first-retry",
  },
  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"] },
    },
  ],
  webServer: isCI
    ? [
        {
          command: "node tests/e2e/mock-backend.mjs",
          url: "http://127.0.0.1:8081/health",
          reuseExistingServer: false,
          timeout: 30_000,
        },
        nextWebServer,
      ]
    : nextWebServer,
});
