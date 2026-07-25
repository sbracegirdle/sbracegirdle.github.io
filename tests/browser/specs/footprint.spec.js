import { test, expect } from "@playwright/test";
import { statSync } from "node:fs";
import path from "node:path";
import { buildDir, samplePages } from "../lib/site.js";
import {
  formatBytes,
  maxPageBytes,
  maxSameOriginRequests,
  maxStylesheetBytes,
} from "../lib/budgets.js";

// What each page costs to load, asserted from real responses. This is the
// cheap, deterministic half of performance work — it runs in the normal suite
// and catches the regression that matters most on a static site: something new
// getting linked from every page. The Lighthouse pass (`node lighthouse.mjs`)
// is the other half; it measures what this can't, and it is too slow to live
// here.
//
// Off-origin requests are blocked rather than measured. A post may hot-link an
// image from anywhere, which is the author's call, and a budget that moved with
// someone else's CDN would be a coin toss. The budgets are in lib/budgets.js.

for (const urlPath of samplePages()) {
  test(`page weight: ${urlPath}`, async ({ page, baseURL }) => {
    const seen = new Map();

    await page.route("**", async (route, request) => {
      if (request.url().startsWith(baseURL)) return route.continue();
      await route.abort();
    });

    page.on("response", async (response) => {
      if (!response.url().startsWith(baseURL)) return;
      try {
        seen.set(response.url(), (await response.body()).length);
      } catch {
        // A response with no retrievable body contributes nothing to weight.
      }
    });

    await page.goto(urlPath, { waitUntil: "load" });

    const bytes = [...seen.values()].reduce((n, b) => n + b, 0);
    const detail = [...seen.entries()]
      .map(([url, b]) => `${url.replace(baseURL, "")} ${formatBytes(b)}`)
      .join(", ");

    expect(
      seen.size,
      `${urlPath} makes ${seen.size} same-origin requests: ${detail}`,
    ).toBeLessThanOrEqual(maxSameOriginRequests);

    expect(
      bytes,
      `${urlPath} weighs ${formatBytes(bytes)} — ${detail}`,
    ).toBeLessThanOrEqual(maxPageBytes);
  });
}

// theme.css is on the critical path of every page on the site, so it gets its
// own budget rather than hiding inside a per-page total.
test("theme.css stays within its budget", () => {
  const bytes = statSync(path.join(buildDir, "theme.css")).size;
  expect(
    bytes,
    `theme.css is ${formatBytes(bytes)}, budget ${formatBytes(maxStylesheetBytes)}`,
  ).toBeLessThanOrEqual(maxStylesheetBytes);
});
