import { defineConfig, devices } from '@playwright/test'

// The smoke test (review 2026-11 FE-8): against a running, seeded server,
// SMOKE_URL (http://localhost:8080 in CI, the Vite dev server locally).
export default defineConfig({
  testDir: 'e2e',
  timeout: 60_000,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL: process.env.SMOKE_URL ?? 'http://localhost:5173',
    trace: 'retain-on-failure',
    // A Chrome other than Playwright's own (PW_EXECUTABLE), where its build isn't installed.
    launchOptions: process.env.PW_EXECUTABLE ? { executablePath: process.env.PW_EXECUTABLE } : {},
  },
  projects: [
    // The demo owner signs in once for the run (e2e/sign-in.setup.ts).
    { name: 'sign-in', testMatch: /sign-in\.setup\.ts$/ },
    { name: 'desktop', dependencies: ['sign-in'], use: { ...devices['Desktop Chrome'], viewport: { width: 1440, height: 900 } } },
    { name: 'phone', dependencies: ['sign-in'], use: { ...devices['Pixel 7'], browserName: 'chromium' } },
  ],
})
