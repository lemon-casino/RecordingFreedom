import {defineConfig, devices} from '@playwright/test'

const port = Number(process.env.WAILS_VITE_PORT) || 9245

export default defineConfig({
  testDir: './tests',
  timeout: 60_000,
  workers: process.env.CI ? 1 : undefined,
  expect: {
    timeout: 10_000,
  },
  fullyParallel: false,
  reporter: process.env.CI ? [['list'], ['html', {open: 'never'}]] : [['list']],
  use: {
    baseURL: `http://127.0.0.1:${port}`,
    trace: 'retain-on-failure',
    screenshot: 'only-on-failure',
    viewport: {width: 1280, height: 820},
  },
  webServer: {
    command: `npm run dev -- --host 127.0.0.1 --port ${port}`,
    url: `http://127.0.0.1:${port}`,
    reuseExistingServer: !process.env.CI,
    timeout: 120_000,
  },
  projects: [
    {
      name: 'chromium',
      use: {...devices['Desktop Chrome']},
    },
  ],
})
