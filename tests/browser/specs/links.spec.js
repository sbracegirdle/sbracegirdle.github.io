import { test, expect } from "@playwright/test";
import { htmlPages, read, samplePages, tagPages } from "../lib/site.js";

// Collects every internal href/src across the generated site, then checks each
// distinct target once. The server resolves extensionless URLs the way GitHub
// Pages does, so a link that works here works in production.
function internalTargets() {
  const targets = new Map(); // url -> the page(s) that link to it
  const attr = /(?:href|src)="([^"]+)"/g;

  for (const pagePath of htmlPages()) {
    const html = read(pagePath);
    for (const [, raw] of html.matchAll(attr)) {
      if (/^(https?:|mailto:|tel:|data:|#)/.test(raw)) continue;
      const url = raw.split("#")[0];
      if (url === "") continue;
      // Everything the generator emits is root-absolute; flag anything that
      // isn't rather than silently resolving it.
      const resolved = url.startsWith("/") ? url : new URL(url, `http://x${pagePath}`).pathname;
      if (!targets.has(resolved)) targets.set(resolved, []);
      targets.get(resolved).push(pagePath);
    }
  }
  return targets;
}

test("no internal link 404s", async ({ request }) => {
  const targets = internalTargets();
  const broken = [];

  for (const [url, sources] of targets) {
    const response = await request.get(url);
    if (response.status() !== 200) {
      broken.push(`${url} [${response.status()}] linked from ${sources.join(", ")}`);
    }
  }

  expect(broken, `${targets.size} internal links checked`).toEqual([]);
});

test("generator-emitted links are root-absolute", async ({ page }) => {
  // The chrome and listings the generator emits must be root-absolute, because
  // the same markup is served from /tags/ one directory deep. Links written by
  // hand inside post prose are the author's business and aren't checked here —
  // "no internal link 404s" already covers whether they resolve.
  const chrome = ".statusline a, .post-list a, .tag-list a, .tag-cloud a";
  const relative = [];

  for (const urlPath of [...samplePages(), ...tagPages().slice(0, 3)]) {
    await page.goto(urlPath);
    const hrefs = await page.$$eval(chrome, (links) =>
      links.map((a) => a.getAttribute("href")).filter(Boolean),
    );
    for (const href of hrefs) {
      if (!/^(https?:|mailto:|tel:|#|\/)/.test(href)) relative.push(`${urlPath} -> ${href}`);
    }
  }

  expect(relative).toEqual([]);
});

test("external links open safely", async () => {
  const unsafe = [];
  for (const pagePath of htmlPages()) {
    const html = read(pagePath);
    for (const [tag] of html.matchAll(/<a\b[^>]*href="https?:\/\/[^"]*"[^>]*>/g)) {
      if (tag.includes('target="_blank"') && !tag.includes("noopener")) {
        unsafe.push(`${pagePath}: ${tag}`);
      }
    }
  }
  expect(unsafe).toEqual([]);
});
