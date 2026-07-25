# Lighthouse audits on a site shaped like this one

Lighthouse is written for the average site on the web: a framework, a bundle, a
font, a tag manager, a hero image. This site is a hand-written stylesheet and
some HTML. A good half of what Lighthouse reports is advice for somebody else's
problem. Repeat it verbatim and you bury the two lines that mattered.

This file is the filter. It says what each recurring audit means here, and
whether it is ever a finding.

## The categories

| Category | What a drop means here |
| --- | --- |
| **Performance** | Something now blocks, weighs or shifts. On a two-request page this is nearly always a new asset. Real, always worth chasing. |
| **Accessibility** | Lighthouse runs axe. A drop is a WCAG failure and outranks the design brief — see `design-review/references/accessibility.md`. Real, report it even when pre-existing. |
| **Best practices** | Console errors, deprecated APIs, insecure requests. On a zero-JavaScript site a drop means something genuinely new arrived. Real. |
| **SEO** | Title, description, crawlability, link text. Mostly guarded by `metadata.spec.js` already. A drop is usually a template regression. |

Scores are computed from a weighted subset of audits, so a failed audit and a
lower score are not the same event. Report the audit, not just the number.

## Audits that fire on every run — never a finding on their own

**`unminified-css`, `unused-css-rules`** — `theme.css` is one hand-written
design system serving every page on the site, documented by
`static/style-guide.html`. Lighthouse sees a single page using a third of it and
calls the rest waste; the next page uses a different third. Minifying it would
save around 11 KB of a 29 KB file and cost the ability to read the thing.
Only report when `maxStylesheetBytes` is actually breached — then the finding is
"theme.css grew", not "minify it".

**`cache-insight` / `uses-long-cache-ttl`** — `serve.mjs` sends
`Cache-Control: no-store` deliberately, so each audit measures a cold load
instead of a warm one. Cache headers in production are GitHub Pages' business
and are not observable from here.

**`document-latency-insight` / `server-response-time`** — a Node static server
on loopback. This measures the test harness.

**`render-blocking-insight`** — `theme.css` in the `<head>` is the one
render-blocking request, and that is the design. Inlining it would duplicate
~29 KB into every page to save one same-origin round trip on a dark theme where
a flash of unstyled content is exactly what the blocking link prevents. Becomes
a finding only if a *second* render-blocking request appears.

**`network-dependency-tree-insight`** — a diagnostic with no score. On a
two-request page there is no tree.

**`is-on-https`** — not applicable on localhost; Lighthouse treats it as secure
and the production site is HTTPS.

**`is-crawlable` on `/404.html`** — the generator emits the 404 page `noindex`
on purpose. Exempted in `lib/budgets.js` with that reason.

## Audits that are real when they fire

**`unsized-images`** — an `<img>` with no `width`/`height` reserves no space, so
the page reflows when it loads. CLS is currently 0 and worth keeping there. For
images the generator emits, fix it in the generator. For images hot-linked from
a post's prose, report it and let Simon decide — the dimensions aren't ours to
know.

**`link-in-text-block`** — a link inside a paragraph distinguishable from the
surrounding text by colour alone, with less than 3:1 contrast between them.
WCAG 2.2 SC 1.4.1 (Use of Colour), level A. Real, and it outranks the look.

**`target-size`** — a tap target under 24×24 CSS pixels without enough spacing
around it. WCAG 2.2 SC 2.5.8 (Target Size, Minimum), level AA. Real. Common on
chip rows and dense inline links.

**`color-contrast`** — axe measuring the rendered colours. The palette's
decorative hues (`--c-pine`, `--color-muted`) fail AA at body size by design and
are documented as decoration-only; using one for body text is the finding.

**`total-byte-weight`, `resource-summary`** — cross-check them against the
footprint line the runner prints. They count off-origin bytes too, which the
budgets deliberately don't.

**Anything in Best Practices about console errors or deprecated APIs** — this
site ships no JavaScript at all, so any hit here means something got added.

## What Lighthouse cannot tell you

It measures one cold load of one page on a simulated phone on an idle-ish
machine. It says nothing about how the page reads, whether the hierarchy works,
whether the statusline clipped a link, or whether the prose is any good. Those
belong to `design-review`, `browser-test` and `prose-review` respectively. Don't
answer their questions with this tool's output, and don't let a green run stand
in for theirs.
