import { defineConfig, devices } from "@playwright/test";

const port = Number(process.env.PORT || 4321);
const baseURL = `http://127.0.0.1:${port}`;

export default defineConfig({
  testDir: "./specs",
  fullyParallel: true,
  forbidOnly: !!process.env.CI,
  retries: 0,
  workers: process.env.CI ? 2 : undefined,
  reporter: process.env.CI ? [["github"], ["list"]] : [["list"]],

  // Headless is non-negotiable: these tests must never pop a window on the
  // host. `globalSetup` re-checks the resolved config so `--headed` on the
  // command line fails the run instead of quietly overriding this.
  globalSetup: "./global-setup.js",

  use: {
    baseURL,
    headless: true,
    screenshot: "only-on-failure",
    trace: "retain-on-failure",
    video: "off",
  },

  projects: [
    {
      name: "chromium",
      use: { ...devices["Desktop Chrome"], headless: true },
    },
  ],

  webServer: {
    command: "bash ./build-site.sh && node serve.mjs",
    url: baseURL,
    cwd: ".",
    reuseExistingServer: false,
    stdout: "pipe",
    stderr: "pipe",
    timeout: 120_000,
  },
});
