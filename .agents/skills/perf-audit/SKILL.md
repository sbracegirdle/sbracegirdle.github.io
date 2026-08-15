---
name: perf-audit
description: Measure what the site costs to load and audit it with Lighthouse. The caller invokes it on significant non-textual changes — a theme or template change, a new page or component, a new asset or request, a generator change that alters emitted markup, or a dependency of any kind. Reports category scores, Core Web Vitals, page weight and request count against the budgets in tests/browser/lib/budgets.js. Not for editing the words in a post. Headless only; it must never open a window on the host.
---

# Performance and footprint audit

This site's whole argument is that a blog is HTML and one stylesheet. Nothing
enforces that on its own. Weight arrives one reasonable decision at a time — a
font here, a widget there — and every one of them is defensible in isolation.
This skill is where the site gets weighed.

The browser suite asks whether a page works and the design review asks whether
it's right. This asks what it costs.

## When this runs

Run it for a change that alters what the browser has to fetch, parse or lay out:

- `static/theme.css` — every page on the site loads it
- `template.html`, or generator changes to emitted markup
- a new page in `static/`, or a new component anywhere
- any new asset, request, script, font or dependency
- anything that adds an image to a post, or a page that renders images

Don't run it for prose edits, docs, tests, or Go internals with no rendered
output — editing the words in a post changes nothing the browser has to fetch.

## Headless, always

Never `--headed`, never `--debug`, never `headless: false`. A window on the host
steals focus from whoever is at the keyboard. Both phases below run headless and
there is no reason to change that.

## One browser workload at a time

Run the two phases one after the other, and never alongside `npm test`, another
audit, or anything else that drives a browser. Each Lighthouse pass launches its
own Chromium, and Playwright runs a worker per core; together they took Simon's
machine down hard enough to reboot it. Wait for one command to finish before
starting the next, and prefer `--sample` over `--all` unless you have a reason.
This is a hard rule, not a performance tip.

## Phase 1 — the budget, in the suite

```bash
cd tests/browser
npx playwright test specs/footprint.spec.js    # or npm test for the whole suite
```

`footprint.spec.js` loads each sample page with off-origin requests blocked and
adds up what we serve: same-origin bytes, same-origin request count, and
`theme.css` on its own because it sits on the critical path of every page. It is
fast, deterministic, and part of `npm test`, so it keeps checking long after
this review is over.

It blocks off-origin requests rather than measuring them. A post may hot-link an
image from anywhere — the author's call — and a budget that moved with someone
else's CDN would be a coin toss.

## Phase 2 — the Lighthouse pass

```bash
cd tests/browser
node lighthouse.mjs --sample              # the standard page set, mobile
node lighthouse.mjs --both / /about.html  # named pages, both form factors
node lighthouse.mjs --desktop --all       # every page (slow: ~15s per page)
node lighthouse.mjs --no-build --json /   # reuse build/, keep the full report
```

First run on a machine: `npm install && npx playwright install chromium`.

Mobile is the default because it is the harsher measurement — an emulated phone
on throttled 4G with a slowed CPU. Desktop numbers look better because the
conditions are easier.

For each page it prints the four category scores, the metrics behind the
performance number, the page's footprint largest-asset-first, and every audit
Lighthouse marked as failed. It exits non-zero when a score or a byte budget in
`lib/budgets.js` is breached.

Audit what the change actually reaches. A `theme.css` rule reaches every page,
so sweep the sample. A new standalone page in `static/` only needs itself — and
it needs itself at both form factors, because it is the one page nobody has ever
measured.

## Reading the report

**Scores** are the headline and the least interesting line. A static, monospace,
zero-JavaScript site should score 100 on all four. Treat any drop as a
regression with a cause, and go find the cause rather than reporting the number.

**Metrics** — FCP, LCP, TBT, CLS, SI — are how the performance score was
reached. Compare them against a run on `main`, not against an absolute. The
throttled numbers are a simulation, and a busy machine moves them.

**Footprint** is the number that matters most here, and the one a loaded machine
can't move. Same-origin bytes and request count, largest asset first. Two
requests and ~35 KB is the current shape of a page here. A third request needs a
reason.

**Failed audits** are listed whether or not they moved a score, because the
diagnostics Lighthouse leaves unscored are often the interesting ones. Read
`references/lighthouse-audits.md` before reporting any of them — it says which
ones mean something on a site shaped like this one and which are noise you must
not report.

**Off-origin weight** is reported and never budgeted, but it is still a finding
when it's absurd. A megabyte of hot-linked images on the homepage is worth
saying out loud even though no budget covers it; the fix is the author's call.

## Raising a budget is Simon's call

`tests/browser/lib/budgets.js` holds every number in one place, sitting at
roughly 1.5–2x what the site ships today: enough room for a post with a long
code block, not enough for a font or a framework to arrive unnoticed.

Raising one is a decision. If a change genuinely needs more room, say what got
heavier and why it was worth it, and let Simon raise the number — never raise a
budget to make your own change pass. `scoreExemptions` works the same way: an
entry needs a reason that still reads as true in a year.

## False positives — do not report these

`references/lighthouse-audits.md` has the full catalogue and the reasoning.
The ones that come up every single run:

- **Minify CSS / reduce unused CSS.** `theme.css` is a hand-written design
  system loaded by every page. "Unused" on one page is used on the next, and the
  file is meant to be read. Only a real budget breach makes this a finding.
- **Use efficient cache lifetimes.** The test server sends `Cache-Control:
  no-store` on purpose. Caching is GitHub Pages' business and cannot be
  measured from here.
- **Document request latency / server response time.** That is a Node server on
  localhost. It is measuring the harness.
- **Render-blocking requests**, for `theme.css` alone. One same-origin
  stylesheet on a dark theme is the design: inlining it would duplicate the
  bytes on every page and trade a flash of unstyled dark for nothing.
- **Anything asking for a light mode, a font, a CDN, or a build step.** The
  theme is the brief, exactly as it is for the design review.

Accessibility findings are the exception. Report them — they are a floor, not a
preference — and tie them to the criterion in
`.agents/skills/design-review/references/accessibility.md`.

## Dispatch

- **Claude Code** — delegate to the `perf-auditor` subagent
  (`.claude/agents/perf-auditor.md`), which runs this skill.
- **Codex** — spawn the `perf-auditor` agent by name
  (`.codex/agents/perf-auditor.toml`), which runs this skill.
- **No subagent support** — run this skill directly.

The auditor reports; it doesn't fix the site and it doesn't touch the budgets.

## Reporting

For each finding: the page, the form factor, the number you measured, the number
you measured it against, and whether the change caused it or it was already
there. A pre-existing failure is still a finding; say that it's pre-existing.

Give the before and after when you have both. "Performance 100, page 36.0 KB
over 2 requests, unchanged from `main`" is a complete report and a short one.
Say which pages you audited and which you didn't — an unaudited page is not a
passing page. If nothing moved, say so in a line rather than padding it out.
