import { defineConfig, devices } from "@playwright/test";

const e2eBaseURL = process.env.E2E_BASE_URL || "http://localhost:8080";

export default defineConfig({
  testDir: "./tests/e2e",
  timeout: 60_000,
  expect: {
    timeout: 5_000,
  },
  fullyParallel: false,
  retries: 0,
  reporter: [["list"]],
  use: {
    baseURL: e2eBaseURL,
    trace: "retain-on-failure",
    screenshot: "only-on-failure",
    video: "retain-on-failure",
  },
  projects: [
    {
      name: "chrome",
      use: {
        ...devices["Desktop Chrome"],
        channel: "chrome",
      },
    },
    {
      name: "mddm-visual-parity",
      testDir: "./e2e",
      use: {
        ...devices["Desktop Chrome"],
        channel: "chrome",
        baseURL: "http://127.0.0.1:4173",
      },
    },
    {
      name: "parallel-flows",
      testDir: "./e2e/flows",
      workers: 3,
      fullyParallel: true,
      retries: 0,
      use: {
        ...devices["Desktop Chrome"],
        baseURL: e2eBaseURL,
        trace: "retain-on-failure",
      },
    },
    {
      name: "serial-clock",
      testDir: "./e2e/flows",
      testMatch: "*scheduled*",
      workers: 1,
      fullyParallel: false,
      retries: 0,
      use: {
        ...devices["Desktop Chrome"],
        baseURL: e2eBaseURL,
        trace: "retain-on-failure",
      },
    },
  ],
  webServer: {
    command: "go run ./apps/api/cmd/metaldocs-api",
    cwd: "../../..",
    env: {
      ...process.env,
      METALDOCS_E2E: "1",
    },
    url: "http://localhost:8080/healthz",
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
  },
});
