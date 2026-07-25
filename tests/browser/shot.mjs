// Exploratory browser tool: build the site, serve it, open real pages in a
// headless Chromium and capture what a visitor would see.
//
//   node tests/browser/shot.mjs / /tags.html /2022-03-08-dont-lgtm-code-reviews.html
//   node tests/browser/shot.mjs --both /tags/devops.html
//   node tests/browser/shot.mjs --viewport --width 1440 /style-guide.html
//
// For each page it writes a PNG and prints a short report — console errors,
// uncaught exceptions, failed requests, off-origin requests, and whether the
// layout overflows horizontally. Open the PNGs to actually *look* at the page;
// the report only catches what a machine can see.
//
// Flags:
//   --both        capture desktop and mobile widths (default: desktop only)
//   --mobile      capture mobile only (390x844)
//   --width N     desktop width (default 1280)
//   --height N    desktop viewport height (default 900)
//   --full        capture the whole scroll height instead of the viewport.
//                 Careful: a long post is 10,000+ px tall and downscales to an
//                 unreadable strip. Prefer viewport shots with --scroll.
//   --scroll N    scroll N pixels down before the shot (repeatable per page by
//                 running the tool again) — how you inspect below the fold
//   --out DIR     where PNGs land (default tests/browser/.shots)
//   --no-build    skip regenerating the site (use the existing build/)
//
// Headless always — this must never open a window on the host machine.

import { chromium } from "@playwright/test";
import { execFileSync } from "node:child_process";
import { mkdir } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { startServer } from "./serve.mjs";

const here = path.dirname(fileURLToPath(import.meta.url));

function parseArgs(argv) {
  const opts = {
    both: false,
    mobile: false,
    width: 1280,
    height: 900,
    fullPage: false,
    scroll: 0,
    out: path.join(here, ".shots"),
    build: true,
    paths: [],
  };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === "--both") opts.both = true;
    else if (a === "--mobile") opts.mobile = true;
    else if (a === "--full") opts.fullPage = true;
    else if (a === "--no-build") opts.build = false;
    else if (a === "--width") opts.width = Number(argv[++i]);
    else if (a === "--height") opts.height = Number(argv[++i]);
    else if (a === "--scroll") opts.scroll = Number(argv[++i]);
    else if (a === "--out") opts.out = path.resolve(argv[++i]);
    else if (a.startsWith("--")) throw new Error(`unknown flag ${a}`);
    else opts.paths.push(a.startsWith("/") ? a : `/${a}`);
  }
  if (opts.paths.length === 0) opts.paths = ["/"];
  return opts;
}

function slug(urlPath) {
  const s = urlPath.replace(/^\//, "").replace(/[^a-zA-Z0-9._-]+/g, "-");
  return s === "" ? "index" : s.replace(/\.html$/, "");
}

// inspect loads one page and reports everything a machine can notice about it.
async function inspect(context, baseURL, urlPath, viewport, opts) {
  const page = await context.newPage();
  await page.setViewportSize(viewport);

  const consoleErrors = [];
  const pageErrors = [];
  const failed = [];
  const offOrigin = [];

  page.on("console", (msg) => {
    if (msg.type() === "error") consoleErrors.push(msg.text());
  });
  page.on("pageerror", (err) => pageErrors.push(err.message));
  page.on("requestfailed", (req) => failed.push(`${req.url()} — ${req.failure()?.errorText}`));
  page.on("request", (req) => {
    if (!req.url().startsWith(baseURL)) offOrigin.push(`${req.resourceType()} ${req.url()}`);
  });

  const response = await page.goto(baseURL + urlPath, { waitUntil: "load" });
  const status = response ? response.status() : 0;

  const overflow = await page.evaluate(
    () => document.documentElement.scrollWidth - window.innerWidth,
  );
  const title = await page.title();

  if (opts.scroll) {
    await page.evaluate((y) => window.scrollTo(0, y), opts.scroll);
  }

  const suffix = opts.scroll ? `--y${opts.scroll}` : "";
  const name = `${slug(urlPath)}--${viewport.width}x${viewport.height}${suffix}.png`;
  const file = path.join(opts.out, name);
  await page.screenshot({ path: file, fullPage: opts.fullPage });
  await page.close();

  return { urlPath, status, title, overflow, consoleErrors, pageErrors, failed, offOrigin, file };
}

function report(r) {
  const lines = [];
  const flag = r.status === 200 ? "ok " : "!! ";
  lines.push(`${flag}${r.urlPath}  [${r.status}] "${r.title}"`);
  lines.push(`   shot: ${r.file}`);
  if (r.overflow > 1) lines.push(`   !! horizontal overflow: ${r.overflow}px wider than the viewport`);
  for (const e of r.pageErrors) lines.push(`   !! uncaught: ${e}`);
  for (const e of r.consoleErrors) lines.push(`   !! console: ${e}`);
  for (const e of r.failed) lines.push(`   !! request failed: ${e}`);
  // Off-origin images come from post prose and are only worth noting; anything
  // else off-origin is a broken promise of the theme.
  const images = r.offOrigin.filter((u) => u.startsWith("image "));
  const code = r.offOrigin.filter((u) => !u.startsWith("image "));
  if (images.length) lines.push(`   -- ${images.length} off-origin image(s), e.g. ${images[0]}`);
  for (const u of code) lines.push(`   !! off-origin ${u}`);
  return lines.join("\n");
}

const opts = parseArgs(process.argv.slice(2));

if (opts.build) {
  execFileSync("bash", [path.join(here, "build-site.sh")], { stdio: "inherit" });
}
await mkdir(opts.out, { recursive: true });

const { baseURL, close } = await startServer();
const browser = await chromium.launch({ headless: true });
const context = await browser.newContext();

const viewports = [];
if (!opts.mobile) viewports.push({ width: opts.width, height: opts.height });
if (opts.mobile || opts.both) viewports.push({ width: 390, height: 844 });

let problems = 0;
for (const urlPath of opts.paths) {
  for (const viewport of viewports) {
    const r = await inspect(context, baseURL, urlPath, viewport, opts);
    console.log(report(r));
    if (
      r.status !== 200 ||
      r.overflow > 1 ||
      r.pageErrors.length ||
      r.consoleErrors.length ||
      r.failed.length ||
      r.offOrigin.some((u) => !u.startsWith("image "))
    ) {
      problems++;
    }
  }
}

await context.close();
await browser.close();
await close();

console.log(
  problems === 0
    ? `\nNo machine-visible problems. Now open the PNGs and look at them.`
    : `\n${problems} page/viewport combination(s) reported problems — see above, then open the PNGs.`,
);
