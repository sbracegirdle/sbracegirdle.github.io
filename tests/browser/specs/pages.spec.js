import { test, expect } from "@playwright/test";
import { htmlPages } from "../lib/site.js";

// Every generated page gets opened in a real browser. Catches template
// regressions, unclosed <head> tags, and pages that render but throw.
for (const urlPath of htmlPages()) {
  test(`renders: ${urlPath}`, async ({ page }) => {
    const consoleErrors = [];
    const pageErrors = [];
    const offOriginCode = [];
    const failed = [];

    page.on("console", (msg) => msg.type() === "error" && consoleErrors.push(msg.text()));
    page.on("pageerror", (err) => pageErrors.push(err.message));
    page.on("requestfailed", (req) => failed.push(`${req.url()} — ${req.failure()?.errorText}`));
    page.on("request", (req) => {
      const url = req.url();
      if (url.startsWith("http://127.0.0.1:") || url.startsWith("http://localhost:")) return;
      // Off-origin *code* is what the theme forbids: no CDN fonts, no
      // analytics, no third-party scripts. Images hot-linked from a post's
      // prose are the author's call, so they don't fail the build.
      if (["script", "stylesheet", "font", "xhr", "fetch", "websocket"].includes(req.resourceType())) {
        offOriginCode.push(`${req.resourceType()} ${url}`);
      }
    });

    const response = await page.goto(urlPath);
    expect(response?.status(), `${urlPath} should serve`).toBe(200);

    // Page chrome: the statusline header and footer wrap every page.
    await expect(page.locator(".statusline").first()).toBeVisible();
    await expect(page.locator("h1")).toHaveCount(1);
    expect(await page.title()).not.toBe("");

    expect(pageErrors, `uncaught errors on ${urlPath}`).toEqual([]);
    expect(consoleErrors, `console errors on ${urlPath}`).toEqual([]);
    expect(failed, `failed requests on ${urlPath}`).toEqual([]);
    expect(offOriginCode, `off-origin code loaded by ${urlPath}`).toEqual([]);
  });
}
