import { defineConfig, devices } from '@playwright/test'

// Browser checks of the built site (`pnpm build` first): accessibility, the
// keyboard, the pickers without JavaScript and reflow. The preview server
// serves dist/.
export default defineConfig({
  testDir: 'e2e',
  timeout: 60_000,
  retries: process.env.CI ? 1 : 0,
  reporter: process.env.CI ? [['list'], ['html', { open: 'never' }]] : 'list',
  use: {
    baseURL: 'http://localhost:5289',
    trace: 'retain-on-failure',
    // A Chrome other than Playwright's own (PW_EXECUTABLE), where its build isn't installed.
    launchOptions: process.env.PW_EXECUTABLE ? { executablePath: process.env.PW_EXECUTABLE } : {},
  },
  webServer: { command: 'pnpm preview', url: 'http://localhost:5289', reuseExistingServer: !process.env.CI },
  projects: [
    { name: 'desktop', use: { ...devices['Desktop Chrome'], viewport: { width: 1440, height: 900 } } },
    { name: 'phone', use: { ...devices['Pixel 7'], browserName: 'chromium' } },
  ],
})
