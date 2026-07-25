---
name: browser-test
description: Verify the generated site in a real headless browser before shipping. Use whenever a change affects what visitors see — posts in content/, template.html, static/theme.css, static/*.html, or the generator itself. Runs the Playwright suite and then an exploratory pass over the changed pages, looking at screenshots and probing layout. Headless only; it must never open a window on the host.
---

# Browser testing

Go tests prove the generator emits the bytes it meant to. They say nothing
about whether the result renders, wraps, or navigates. This skill is how the
site gets checked in a browser before it ships.

Two phases, both mandatory. A green suite with no exploratory pass is an
incomplete report, and so is the reverse.

## Headless, always

Never pass `--headed` or `--debug`, never launch with `headless: false`, never
open a browser on the host machine — a window that steals focus interrupts the
person you're working for. `tests/browser/global-setup.js` refuses a headed run,
so attempting one only wastes a cycle. To inspect a failure, read the screenshot
and trace Playwright already saved.

## Phase 1 — the static suite

```bash
cd tests/browser
npm test          # regenerates build/, then runs every spec
```

Run it from `tests/browser/`. Running Playwright from the repo root picks up a
different, unconfigured copy of the package and fails with "did not expect
test() to be called here".

First time on a machine: `npm install && npx playwright install chromium`.

To iterate on one area: `npx playwright test specs/theme.spec.js`, or
`npx playwright test --grep "tag chips"`.

What the specs cover:

| Spec | Covers |
| --- | --- |
| `pages.spec.js` | every generated page renders — 200, one `h1`, page chrome, no console errors, no uncaught exceptions, no off-origin code |
| `metadata.spec.js` | title, description, canonical, Open Graph, Twitter card, feed discovery; `article` vs `website` og:type; the escaping regression a quoted description would cause |
| `theme.spec.js` | theme.css actually applies, monospace body, no horizontal overflow at 390px **or 320px** on any post or listing, tag chips navigate, tag index counts |
| `links.spec.js` | no internal link 404s anywhere on the site, generator-emitted links are root-absolute, external links open safely |
| `a11y.spec.js` | the mechanical accessibility floor — `lang`, one `h1`, heading order, accessible names, `alt`, no `aria-label` on a bare generic, nothing focusable inside `aria-hidden` |
| `machine-readable.spec.js` | feed.xml parses and every item link resolves, sitemap.xml URLs resolve, robots.txt points at the sitemap, unknown paths return a real 404 |
| `footprint.spec.js` | same-origin page weight, request count and theme.css size against the budgets in `lib/budgets.js`. The cheap half of the `perf-audit` skill — a breach here belongs to that gate, not this one |

When a spec fails, diagnose before changing anything. Failures leave a
screenshot and a trace under `tests/browser/test-results/`; read the screenshot.
A failing assertion is more often a real site bug than a bad test — decide
which, and say why you decided it.

## Phase 2 — exploratory testing

The suite only catches what someone already thought to assert. This phase is for
what nobody asserted. Go in with a question, not a checklist.

```bash
node shot.mjs --both / /tags.html /some-post.html   # desktop + mobile
node shot.mjs --width 320 /2021-12-20-cdk-cr.html   # narrowest supported width
node shot.mjs --scroll 2000 /some-post.html         # inspect below the fold
node shot.mjs --no-build --mobile /404              # reuse the current build
```

For each page `shot.mjs` writes a PNG to `tests/browser/.shots/` and reports
status, console errors, failed requests, off-origin requests and horizontal
overflow. **Then open the PNGs and look at them.** The report catches
machine-visible problems; you are there for the rest — text colliding with a
border, a heading that reads wrong at phone width, highlighting that looks off,
a chip row that wraps into a mess, contrast that technically passes and still
looks bad.

Shots are viewport-sized by default, which is what you want: `--full` on a long
post produces a 10,000px strip that downscales into an unreadable smear. Walk
down a long page with `--scroll` instead.

Focus on what changed:

- **A new or edited post** — read it at both widths, follow its tag chips, check
  its code blocks, images, and any long URLs in the prose.
- **A theme change** — sweep a post, the homepage, a tag page, the style guide
  and the 404. One rule reaches all of them.
- **A generator change** — check the pages whose markup it emits, including the
  nested `/tags/` pages where root-absolute links matter.

For anything a screenshot can't answer, write a throwaway script in
`tests/browser/scratch/` (git-ignored) and run it:

```js
import { chromium } from "@playwright/test";
import { startServer } from "../serve.mjs";

const { baseURL, close } = await startServer();
const browser = await chromium.launch({ headless: true });
const page = await browser.newPage({ viewport: { width: 390, height: 844 } });
await page.goto(baseURL + "/index.html");
// measure, click, tab through focus order, toggle prefers-reduced-motion…
console.log(await page.evaluate(() => document.documentElement.scrollWidth));
await browser.close();
await close();
```

The script must live under `tests/browser/` — Node resolves `@playwright/test`
from the script's own directory, so a script in `/tmp` cannot import it.

Worth probing when it's relevant to the change: keyboard focus order and visible
focus rings, `prefers-reduced-motion`, 2560px viewports, long unbroken tokens in
prose and titles, a post with no tags, the first and last item in a list, hover
and focus states. Two repeat offenders on this site: hidden decoration, because
an invisible element still occupies layout and can widen the page; and
shrink-to-fit boxes, because `overflow-wrap: break-word` does not reduce their
min-content width — only `anywhere` does.

## Reporting

For each finding give: the page, what you observed, how you observed it (spec
name, or the probe you ran), and whether it's a regression from the current
change or pre-existing. Say which pages you looked at and which you didn't — an
unexamined page is not a passing page.

Distinguish a real defect from a too-strict assertion, and call it. If the suite
is green and the exploratory pass found nothing, say so in one line rather than
padding it out. Clean up `tests/browser/scratch/` when you're done.
