import { defineConfig, devices } from '@playwright/test';

const baseURL = process.env.E2E_BASE_URL ?? 'http://localhost:8080';

const sharedUse = {
  ...devices['Desktop Chrome'],
  viewport: { width: 1280, height: 800 },
  locale: 'pt-BR',
};

const startBackend = process.env.START_BACKEND === '1';

export default defineConfig({
  testDir: './e2e/flows',
  fullyParallel: true,
  retries: 0,
  expect: {
    timeout: 5000,
  },
  reporter: [['html'], ['list']],
  use: {
    baseURL,
    trace: 'retain-on-failure',
    video: 'retain-on-failure',
  },
  projects: [
    {
      name: 'parallel-flows',
      workers: 3,
      use: {
        ...sharedUse,
      },
    },
    {
      name: 'serial-clock',
      workers: 1,
      testMatch: ['**/scheduled_publish.spec.ts'],
      use: {
        ...sharedUse,
      },
    },
  ],
  ...(startBackend
    ? {
        webServer: {
          command: 'go run ./cmd/api',
          url: 'http://localhost:8080/healthz',
          reuseExistingServer: !process.env.CI,
          timeout: 120_000,
        },
      }
    : {}),
});
