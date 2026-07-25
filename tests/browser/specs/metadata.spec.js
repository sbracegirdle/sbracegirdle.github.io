import { test, expect } from "@playwright/test";
import { aTaggedPost, samplePages, tagPages } from "../lib/site.js";

const siteURL = "https://letsbuild.cloud";

async function head(page, selector, attr = "content") {
  return page.locator(selector).first().getAttribute(attr);
}

for (const urlPath of samplePages()) {
  test(`head metadata: ${urlPath}`, async ({ page }) => {
    await page.goto(urlPath);

    expect(await page.title()).not.toBe("");
    expect(await head(page, 'meta[name="description"]')).toBeTruthy();
    expect(await head(page, 'meta[name="author"]')).toBe("Simon Bracegirdle");

    const canonical = await head(page, 'link[rel="canonical"]', "href");
    expect(canonical, `${urlPath} needs a canonical URL`).toContain(siteURL);

    expect(await head(page, 'meta[property="og:title"]')).toBeTruthy();
    expect(await head(page, 'meta[property="og:description"]')).toBeTruthy();
    expect(await head(page, 'meta[property="og:url"]')).toBe(canonical);
    expect(await head(page, 'meta[property="og:site_name"]')).toBe("LetsBuild.cloud");
    expect(await head(page, 'meta[name="twitter:card"]')).toBe("summary");

    // The feed is discoverable from every page.
    expect(await head(page, 'link[rel="alternate"][type="application/rss+xml"]', "href")).toBe(
      "/feed.xml",
    );
  });
}

test("canonical URL matches the page's own path", async ({ page }) => {
  const post = aTaggedPost();
  await page.goto(post);
  expect(await head(page, 'link[rel="canonical"]', "href")).toBe(siteURL + post);

  await page.goto("/index.html");
  expect(await head(page, 'link[rel="canonical"]', "href")).toBe(siteURL + "/");
});

test("posts are og:type article with a published date", async ({ page }) => {
  const post = aTaggedPost();
  await page.goto(post);
  expect(await head(page, 'meta[property="og:type"]')).toBe("article");
  const published = await head(page, 'meta[property="article:published_time"]');
  expect(published).toMatch(/^\d{4}-\d{2}-\d{2}T/);
  // The date in the URL and the date in the metadata must agree.
  expect(post).toContain(published.slice(0, 10));
});

test("listing pages are og:type website", async ({ page }) => {
  for (const urlPath of ["/index.html", "/tags.html", tagPages()[0]]) {
    await page.goto(urlPath);
    expect(await head(page, 'meta[property="og:type"]'), urlPath).toBe("website");
  }
});

test("a description containing quotes does not break the head", async ({ page }) => {
  // The escaping regression this guards against: one raw `"` in a description
  // closes the content attribute and swallows the rest of <head>.
  await page.goto("/2022-03-08-dont-lgtm-code-reviews.html");
  const description = await head(page, 'meta[name="description"]');
  expect(description).toContain('"');
  // If the attribute had broken out, these later tags would be gone.
  expect(await head(page, 'meta[name="twitter:card"]')).toBe("summary");
  await expect(page.locator('link[rel="stylesheet"][href="/theme.css"]')).toHaveCount(1);
});
