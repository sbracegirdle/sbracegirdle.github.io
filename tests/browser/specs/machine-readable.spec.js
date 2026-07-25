import { test, expect } from "@playwright/test";
import { postPages } from "../lib/site.js";

const siteURL = "https://letsbuild.cloud";

test("feed.xml is well-formed RSS with items", async ({ request, page }) => {
  const response = await request.get("/feed.xml");
  expect(response.status()).toBe(200);
  const xml = await response.text();

  // Parse it with the browser's XML parser — a malformed feed fails here the
  // same way it would in a reader.
  const parsed = await page.evaluate((source) => {
    const doc = new DOMParser().parseFromString(source, "application/xml");
    const error = doc.querySelector("parsererror");
    return {
      error: error ? error.textContent : null,
      channelTitle: doc.querySelector("channel > title")?.textContent ?? null,
      channelLink: doc.querySelector("channel > link")?.textContent ?? null,
      items: [...doc.querySelectorAll("item")].map((item) => ({
        title: item.querySelector("title")?.textContent ?? "",
        link: item.querySelector("link")?.textContent ?? "",
        pubDate: item.querySelector("pubDate")?.textContent ?? "",
      })),
    };
  }, xml);

  expect(parsed.error).toBeNull();
  expect(parsed.channelTitle).toBeTruthy();
  expect(parsed.channelLink).toBe(siteURL + "/");
  expect(parsed.items.length).toBe(postPages().length);

  for (const item of parsed.items) {
    expect(item.title).not.toBe("");
    expect(item.link).toContain(siteURL);
    expect(Number.isNaN(Date.parse(item.pubDate)), `bad pubDate: ${item.pubDate}`).toBe(false);
  }
});

test("every feed link resolves", async ({ request, page }) => {
  const xml = await (await request.get("/feed.xml")).text();
  const links = await page.evaluate((source) => {
    const doc = new DOMParser().parseFromString(source, "application/xml");
    return [...doc.querySelectorAll("item > link")].map((n) => n.textContent ?? "");
  }, xml);

  const broken = [];
  for (const link of links) {
    const path = link.replace(siteURL, "");
    const response = await request.get(path);
    if (response.status() !== 200) broken.push(`${link} [${response.status()}]`);
  }
  expect(broken).toEqual([]);
});

test("sitemap.xml lists reachable URLs", async ({ request, page }) => {
  const response = await request.get("/sitemap.xml");
  expect(response.status()).toBe(200);
  const xml = await response.text();

  const urls = await page.evaluate((source) => {
    const doc = new DOMParser().parseFromString(source, "application/xml");
    if (doc.querySelector("parsererror")) return null;
    return [...doc.querySelectorAll("url > loc")].map((n) => n.textContent ?? "");
  }, xml);

  expect(urls, "sitemap.xml did not parse").not.toBeNull();
  expect(urls.length).toBeGreaterThan(0);

  const broken = [];
  for (const url of urls) {
    expect(url.startsWith(siteURL), `sitemap URL not absolute: ${url}`).toBe(true);
    const response = await request.get(url.replace(siteURL, "") || "/");
    if (response.status() !== 200) broken.push(`${url} [${response.status()}]`);
  }
  expect(broken).toEqual([]);
});

test("the sitemap does not advertise the 404 page", async ({ request }) => {
  const xml = await (await request.get("/sitemap.xml")).text();
  expect(xml).not.toContain("404.html");
});

test("robots.txt points at the sitemap", async ({ request }) => {
  const response = await request.get("/robots.txt");
  expect(response.status()).toBe(200);
  expect(await response.text()).toContain(`Sitemap: ${siteURL}/sitemap.xml`);
});

test("unknown paths serve the 404 page with a 404 status", async ({ page }) => {
  const response = await page.goto("/no-such-page-here");
  expect(response?.status()).toBe(404);
  await expect(page.locator("h1")).toContainText("404");
  // Search engines must not index it.
  await expect(page.locator('meta[name="robots"][content*="noindex"]')).toHaveCount(1);
});
