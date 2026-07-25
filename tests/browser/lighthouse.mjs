// Lighthouse and page-footprint auditing for the generated site.
//
//   node tests/browser/lighthouse.mjs /                      # mobile, the default
//   node tests/browser/lighthouse.mjs --both / /style-guide.html
//   node tests/browser/lighthouse.mjs --desktop --sample     # the standard page set
//   node tests/browser/lighthouse.mjs --no-build --json /    # keep the full report
//
// For each page it prints the four Lighthouse category scores, the metrics
// behind the performance number, every audit that did not pass, and what the
// page actually weighs — same-origin bytes and requests, largest asset first.
// It exits non-zero when a score or a byte budget in lib/budgets.js is breached.
//
// Flags:
//   --mobile      emulated phone with slow 4G throttling (default)
//   --desktop     desktop viewport, desktop throttling
//   --both        run each page twice, once per form factor
//   --sample      audit the standard sample of pages instead of naming them
//   --all         audit every generated page (slow: ~15s per page per preset)
//   --json        also write the full Lighthouse JSON under .lighthouse/
//   --no-build    skip regenerating the site (use the existing build/)
//
// Headless always — this must never open a window on the host machine.

import { chromium } from "@playwright/test";
import lighthouse from "lighthouse";
import desktopConfig from "lighthouse/core/config/desktop-config.js";
import { execFileSync } from "node:child_process";
import { createServer } from "node:net";
import { mkdir, writeFile } from "node:fs/promises";
import path from "node:path";
import { fileURLToPath } from "node:url";
import { startServer } from "./serve.mjs";
import { htmlPages, samplePages } from "./lib/site.js";
import {
  formatBytes,
  maxPageBytes,
  maxSameOriginRequests,
  minScores,
  scoreExemptions,
} from "./lib/budgets.js";

const here = path.dirname(fileURLToPath(import.meta.url));
const categories = ["performance", "accessibility", "best-practices", "seo"];

function parseArgs(argv) {
  const opts = { desktop: false, both: false, json: false, build: true, set: null, paths: [] };
  for (let i = 0; i < argv.length; i++) {
    const a = argv[i];
    if (a === "--desktop") opts.desktop = true;
    else if (a === "--mobile") opts.desktop = false;
    else if (a === "--both") opts.both = true;
    else if (a === "--sample") opts.set = "sample";
    else if (a === "--all") opts.set = "all";
    else if (a === "--json") opts.json = true;
    else if (a === "--no-build") opts.build = false;
    else if (a.startsWith("--")) throw new Error(`unknown flag ${a}`);
    else opts.paths.push(a.startsWith("/") ? a : `/${a}`);
  }
  return opts;
}

// freePort asks the OS for an unused port and hands it straight to Chrome.
// There is a race in principle; in practice nothing else on the machine is
// racing us for it, and the alternative is parsing DevToolsActivePort out of a
// throwaway profile directory.
function freePort() {
  return new Promise((resolve, reject) => {
    const server = createServer();
    server.once("error", reject);
    server.listen(0, "127.0.0.1", () => {
      const { port } = server.address();
      server.close(() => resolve(port));
    });
  });
}

// Chrome opens the debugging port a moment after launch returns.
async function waitForCDP(port, timeoutMs = 15_000) {
  const deadline = Date.now() + timeoutMs;
  for (;;) {
    try {
      const res = await fetch(`http://127.0.0.1:${port}/json/version`);
      if (res.ok) return;
    } catch {
      // not listening yet
    }
    if (Date.now() > deadline) {
      throw new Error(`Chrome never opened its debugging port on ${port}`);
    }
    await new Promise((r) => setTimeout(r, 100));
  }
}

function slug(urlPath) {
  const s = urlPath.replace(/^\//, "").replace(/[^a-zA-Z0-9._-]+/g, "-");
  return s === "" ? "index" : s.replace(/\.html$/, "");
}

// audit runs one Lighthouse pass against an already-running Chrome.
//
// Lighthouse occasionally comes back with no scores at all — a loaded machine,
// a page it decided it never saw load. That result is indistinguishable from a
// clean run unless you check for it, so check for it, retry once, and fail
// loudly rather than printing a row of `n/a` that reads like a pass.
async function audit(baseURL, urlPath, port, formFactor, attempt = 1) {
  const flags = {
    port,
    output: "json",
    logLevel: "error",
    onlyCategories: categories,
    // A fresh profile per run. Without this the second page in a run reads
    // theme.css from the memory cache and scores a load nobody will ever get.
    disableStorageReset: false,
  };
  const config = formFactor === "desktop" ? desktopConfig : undefined;
  const { lhr } = await lighthouse(baseURL + urlPath, flags, config);

  const scored = categories.some((c) => typeof lhr.categories[c]?.score === "number");
  if (lhr.runtimeError || !scored) {
    const why = lhr.runtimeError?.message ?? "Lighthouse returned no category scores";
    if (attempt === 1) {
      console.log(`   .. ${urlPath} [${formFactor}] produced no result (${why}) — retrying`);
      return audit(baseURL, urlPath, port, formFactor, 2);
    }
    throw new Error(`${urlPath} [${formFactor}]: ${why}`);
  }
  return lhr;
}

// footprint splits Lighthouse's network log into what we serve and what a
// post's prose hot-links from elsewhere. Only the first is budgeted.
function footprint(lhr, baseURL) {
  const items = lhr.audits["network-requests"]?.details?.items ?? [];
  const own = [];
  const offOrigin = [];
  for (const r of items) {
    const size = r.transferSize ?? r.resourceSize ?? 0;
    (r.url.startsWith(baseURL) ? own : offOrigin).push({
      url: r.url.replace(baseURL, "") || "/",
      type: r.resourceType || "Other",
      size,
    });
  }
  own.sort((a, b) => b.size - a.size);
  return {
    own,
    offOrigin,
    bytes: own.reduce((n, r) => n + r.size, 0),
    requests: own.length,
  };
}

const metricKeys = [
  ["first-contentful-paint", "FCP"],
  ["largest-contentful-paint", "LCP"],
  ["total-blocking-time", "TBT"],
  ["cumulative-layout-shift", "CLS"],
  ["speed-index", "SI"],
];

function report(urlPath, formFactor, lhr, fp) {
  const lines = [];
  const problems = [];
  const exemptions = [];

  const exempt = scoreExemptions[urlPath] ?? {};
  const scores = categories.map((c) => {
    const raw = lhr.categories[c]?.score;
    const value = raw === null || raw === undefined ? null : Math.round(raw * 100);
    if (value !== null && value < minScores[c] && !exempt[c]) {
      problems.push(`${c} ${value} < ${minScores[c]}`);
    }
    if (exempt[c]) exemptions.push(`${c} ${value ?? "n/a"} — ${exempt[c]}`);
    return `${c} ${value === null ? "n/a" : value}`;
  });

  if (fp.bytes > maxPageBytes) {
    problems.push(`page weight ${formatBytes(fp.bytes)} > ${formatBytes(maxPageBytes)}`);
  }
  if (fp.requests > maxSameOriginRequests) {
    problems.push(`${fp.requests} same-origin requests > ${maxSameOriginRequests}`);
  }

  lines.push(`${problems.length ? "!! " : "ok "}${urlPath}  [${formFactor}]`);
  lines.push(`   scores: ${scores.join("  ")}`);
  lines.push(
    `   metrics: ${metricKeys
      .map(([id, label]) => `${label} ${lhr.audits[id]?.displayValue ?? "?"}`)
      .join("  ")}`,
  );
  lines.push(
    `   footprint: ${formatBytes(fp.bytes)} over ${fp.requests} same-origin request(s)` +
      (fp.offOrigin.length ? `, plus ${fp.offOrigin.length} off-origin` : ""),
  );
  for (const r of fp.own.slice(0, 4)) {
    lines.push(`     ${formatBytes(r.size).padStart(9)}  ${r.type}  ${r.url}`);
  }

  // Every audit Lighthouse marked as failed, whether or not it moved a score.
  // The diagnostics it leaves unscored are often the interesting ones.
  const failed = Object.values(lhr.audits).filter(
    (a) => a.score !== null && a.score < 1 && a.scoreDisplayMode !== "notApplicable",
  );
  for (const a of failed) {
    lines.push(`   -- ${a.id}: ${a.title}${a.displayValue ? ` (${a.displayValue})` : ""}`);
  }
  for (const e of exemptions) lines.push(`   ~~ exempt: ${e}`);
  for (const p of problems) lines.push(`   !! over budget: ${p}`);

  return { text: lines.join("\n"), problems: problems.length };
}

const opts = parseArgs(process.argv.slice(2));

if (opts.build) {
  execFileSync("bash", [path.join(here, "build-site.sh")], { stdio: "inherit" });
}

let paths = opts.paths;
if (opts.set === "all") paths = htmlPages();
else if (opts.set === "sample" || paths.length === 0) paths = samplePages();

const formFactors = opts.both ? ["mobile", "desktop"] : [opts.desktop ? "desktop" : "mobile"];

const { baseURL, close } = await startServer();
// Lighthouse drives Chrome over CDP rather than through Playwright's API, so
// the browser has to be launched with a debugging port it can be told about.
// Playwright's own endpoint is its protocol, not CDP, so pick the port here.
const port = await freePort();
const browser = await chromium.launch({
  headless: true,
  args: [`--remote-debugging-port=${port}`],
});
await waitForCDP(port);

const outDir = path.join(here, ".lighthouse");
if (opts.json) await mkdir(outDir, { recursive: true });

let problems = 0;
try {
  for (const urlPath of paths) {
    for (const formFactor of formFactors) {
      const lhr = await audit(baseURL, urlPath, port, formFactor);
      const fp = footprint(lhr, baseURL);
      const r = report(urlPath, formFactor, lhr, fp);
      console.log(r.text);
      problems += r.problems;
      if (opts.json) {
        const file = path.join(outDir, `${slug(urlPath)}--${formFactor}.json`);
        await writeFile(file, JSON.stringify(lhr, null, 2));
        console.log(`   json: ${file}`);
      }
    }
  }
} finally {
  await browser.close();
  await close();
}

console.log(
  problems === 0
    ? `\nAll ${paths.length} page(s) within budget. Read the failed audits above anyway — not every one moves a score.`
    : `\n${problems} budget breach(es) — see above.`,
);
process.exitCode = problems === 0 ? 0 : 1;
