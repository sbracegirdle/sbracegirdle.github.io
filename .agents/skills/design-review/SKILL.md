---
name: design-review
description: Review the visual, semantic and accessibility quality of the generated site. The caller invokes it on changes that touch static/theme.css, template.html, static/*.html, or generator markup that affects layout — and whenever a new component or page appears. Judges fidelity to the site's own design system, visual hierarchy, text fit and containment, responsive behaviour, semantic HTML and ARIA, accessible names, keyboard access, and the WCAG 2.2 AA floor. Reports findings with concrete CSS or markup fixes; never edits the site.
---

# Design review

The Go tests prove the generator emits what it meant to. The browser suite
proves the result renders, navigates and doesn't overflow. Neither asks whether
it *looks right*. This skill does.

A page can pass every assertion in `tests/browser/specs/` and still be broken
design: navigation clipped out of existence, a heading that outweighs the
content under it, a token invented instead of reused, tap targets a thumb
can't hit. Those are the findings this review exists to produce.

## Scope

Review the design of anything a visitor sees:

- `static/theme.css` — tokens and components, the source of truth for the look
- `template.html` — page chrome, the statusline header and footer
- `static/*.html` — the style guide, the Rust quick reference, any standalone page
- Generator markup that carries layout: `main.go`, `tags.go`, `highlight.go`
- Rendered pages as a whole — a post, the index, a tag listing, the 404

Not in scope: prose wording (that's `prose-review`), whether a page loads at
all or a link 404s (that's `browser-test`), Go internals, tests, build tooling.

Where this overlaps `browser-test`, defer to it on mechanics and keep judgement
for yourself. `browser-test` asks "does the page overflow the viewport?".
This skill asks "is anything unreadable, unreachable, or off-system?" — which
includes the failure mode where a container clips its own content and the page
width stays perfectly correct.

## The design system is the brief

This site has already committed to an aesthetic: a dark-only "night terminal"
on the Rosé Pine palette, monospace throughout, square corners, near-zero
motion. That commitment is the brief. **Do not review it as though the site
were still choosing a look**, and never propose a redesign.

Before reviewing, read both:

1. `static/style-guide.html` — the token reference, component demos, usage
   rules and verified contrast ratios. It is the specification.
2. `static/theme.css` — what is actually implemented.

Two reference files sit beside this skill:

- `references/design-standards.md` — general UI standards, distilled from the
  vendor guides on agent-authored interfaces.
- `references/accessibility.md` — the WCAG 2.2 AA floor, semantics and ARIA,
  accessible names, truncation and tooltips, keyboard, contrast, target size.

The general standards apply on top of the house system, never against it. Where
a general rule and the house system disagree, the house system wins and the
general rule is a false positive — see "Do not flag". **Accessibility is the
exception: it outranks taste, the house system and the diff alike.** A
contrast, semantics or keyboard failure is a finding even when it is
pre-existing and even when the look was deliberate.

## Process

1. Read `static/style-guide.html`, then `static/theme.css`.
2. Get the change — `git diff`, `git diff --cached` for staged work,
   `git diff main...HEAD` for a branch. If the caller named specific pages,
   review those.
3. Work out which pages the change reaches. One `theme.css` rule reaches every
   page; a component only used by the style guide reaches one.
4. **Look at the pages.** Capture them and open the PNGs:

   ```bash
   cd tests/browser
   node shot.mjs --both / /tags.html /style-guide.html   # 1280px and 390px
   node shot.mjs --width 320 /some-post.html             # narrowest supported
   node shot.mjs --scroll 2000 /some-post.html           # below the fold
   ```

   A review written without opening a screenshot is not a design review. The
   tooling and its flags are documented in the `browser-test` skill; this skill
   reuses them rather than defining its own.
5. Measure anything the eye can't settle. Throwaway probes go in
   `tests/browser/scratch/` (git-ignored) — they must live under
   `tests/browser/` so Node can resolve `@playwright/test`:

   ```js
   import { chromium } from "@playwright/test";
   import { startServer } from "../serve.mjs";

   const { baseURL, close } = await startServer();
   const browser = await chromium.launch({ headless: true });
   const page = await browser.newPage({ viewport: { width: 390, height: 844 } });
   await page.goto(baseURL + "/index.html");
   // getBoundingClientRect, getComputedStyle, scrollWidth vs clientWidth,
   // tab through focus order, toggle prefers-reduced-motion…
   await browser.close();
   await close();
   ```
6. Report in the output format below. Clean up `scratch/` when you finish.

**Headless, always.** Never `--headed`, never `--debug`, never
`headless: false`. A window that opens on the host steals focus from whoever is
at the keyboard, and `tests/browser/global-setup.js` refuses the run anyway.

## What to review

### 1. System fidelity

Every value comes from a token. A raw hex, a raw pixel, a one-off font size or
a hand-rolled shade is a finding — name the token it should have used. New
components belong in `static/theme.css`, not inline in a page, and anything new
or changed must be documented in `static/style-guide.html`. A component that
exists in the CSS but not the guide is a finding; so is a guide that describes
behaviour the CSS no longer has.

Each hue has exactly one job: foam for links and primary, gold for numbers and
dates, rose for title fill and emphasis, iris for h3 and forms, love for
danger. A hue used off-duty is a finding. `--c-pine` and `--color-muted` are
decorative only and fail AA at body size — flag either one carrying real text.

### 2. Hierarchy and composition

Rank should come from structure, not decoration: the inverted rose h1 block,
gold auto-numbered h2 with its fill rule, iris-sidebar h3, small-caps h4.
Check that heading levels descend without skipping, that the most important
thing on the page is the most prominent, and that a scan of the headings alone
tells you what the page is about.

Spacing should read as rhythm, not accident — related things closer than
unrelated things, consistent gaps between peers. Match display size to its
container: a heading sized for a full-width page looks wrong inside a compact
panel.

### 3. Fit and containment

The highest-yield section, and the one automated checks miss most often.

- **Text must fit its element.** If it can't, it wraps, shrinks or truncates
  visibly — it never silently disappears.
- **Truncation is the last resort, and it owes the reader the full text.**
  Wrapping or more room beats an ellipsis. Where something does truncate, it
  carries a `title` with the same string, and it is never a link or a button —
  see the truncation section of `references/accessibility.md`, which is the
  house rule as well as the general one.
- **Nothing interactive may be clipped.** A container with `overflow: hidden`
  and unshrinkable flex children hides links while the page width stays
  perfectly correct, so `theme.spec.js` passes and the link is simply gone.
  Check every flex or grid row that holds links: does its content width exceed
  its box at any viewport?
- **Nothing may overlap** unintentionally — text over a border, a chip over a
  rule, one column into another.
- **Layout must be stable.** Hover, focus and dynamic content must not resize
  or shift anything around them.

### 4. Responsive behaviour

Check 1280px, 390px and 320px at minimum. Nothing may widen the page: inline
`code`, links and `h1` break mid-token; `pre` scrolls internally; flex rows
wrap. Remember `overflow-wrap: break-word` does not shrink a shrink-to-fit box
such as the `h1` block — only `anywhere` does. Hidden decoration must be hidden
with `display`, because an invisible box still occupies layout and still widens
the page.

Sizing should be container-relative, not viewport-scaled. Tap targets want
roughly 24px of hit area or better; statusline segments are small by design,
but they must still be reachable.

### 5. Semantics and accessibility

The floor, and the one dimension that outranks everything else here. Work
through `references/accessibility.md` — it carries the detail, the success
criteria and the sources. The short version, in the order findings actually
turn up:

- **The right element.** A link navigates, a button acts. Headings mark
  structure, not size, and levels descend without skipping. Landmarks are
  `header`/`nav`/`main`/`footer`, one `main`, one `h1`. A `div` doing a
  control's job is a must-fix however well it is styled.
- **ARIA only where native markup can't reach**, and never contradicting it.
  Watch for `aria-label` on a bare `div` or `span` — role `generic` doesn't
  support naming, so the label is silently dropped. This repo shipped that on
  its statusline.
- **Accessible names.** Every link, button, image and control has one.
  Link text stands on its own without the surrounding sentence. Meaningful
  images have `alt`; decorative ones have `alt=""` or `aria-hidden`.
- **Truncated text keeps a route to the whole string** — the `title` rule in
  §3. `title` is a hint on top of visible text, never the only name.
- **Keyboard.** Everything focusable is reachable, in reading order, with a
  visible focus ring, and nothing sticky covers the focused element.
- **Contrast** — 4.5:1 for body text, 3:1 for large text, 3:1 for icons, focus
  rings and control boundaries. The style guide records the verified ratios;
  recompute from the resolved colours when one moves. Never let colour alone
  carry meaning.
- **Target size** — about 24×24px of hit area. Statusline segments are small by
  design; measure them rather than assuming they clear it.
- **Reflow and text spacing** — usable at 320px, and nothing clipped when line
  height, word and letter spacing are increased. A fixed-height box holding
  text usually fails this.
- **Motion** — everything animated is disabled under `prefers-reduced-motion`.

Check these by probing the DOM, not by reading the CSS: names, roles, `alt`,
`lang`, heading order and `scrollWidth` vs `clientWidth` are all one
`page.evaluate` away, and a probe that finds nothing is still worth writing
down as checked.

### 6. Motion

This site has essentially one animation, the prompt cursor. Restraint is the
house position. Flag new motion that isn't earned, motion that fires on scroll
for decoration, and any transition that moves layout rather than appearance.

### 7. Writing as interface

Not prose wording — that belongs to `prose-review` — but the interface text as
a design element. Labels use plain active verbs, sentence case, consistent
terminology. Link text says where it goes. Empty states point at the next
action rather than apologising.

## Do not flag

The general standards were written for teams choosing a look. This site has
chosen one. Reporting any of these wastes the author's attention:

- **Monospace everywhere.** Deliberate, and the whole point of the theme.
  Never propose a display or body typeface pairing.
- **Square corners.** `--radius: 0` is the house position, not an oversight.
- **The single dark palette, and no light mode.** Dark-only is a decision.
- **Positive letter-spacing** from `--tracking-wide`. It is a token doing its
  job on small-caps and labels.
- **"Add gradients, atmosphere, depth, texture, a hero image."** The theme
  makes zero external requests and uses flat surfaces on purpose.
- **"Add motion, entrance animations, scroll effects."** See above.
- **Terminal metaphors** — statusline chrome, `❯` prompts, `eof`, `~/blog`
  paths, ISO dates. These carry the brief.
- **Off-origin images in post prose.** The author's call; the "no external
  requests" rule is about the theme.
- **Anything the diff didn't touch**, unless it's a genuine accessibility or
  containment defect, which is always worth raising.
- **Taste you can't tie to a rule** in this skill, the style guide, or
  `references/design-standards.md`. If you can't name the rule, drop it.

Accessibility outranks the rest, but it grows its own false positives. These
are not findings either:

- **ARIA that restates a native role** — `role="navigation"` on `<nav>`,
  `role="button"` on `<button>`. The absence of redundant ARIA is correct.
- **AAA criteria.** AA is the bar. A 6:1 ratio is a pass, not a near miss.
- **"Add a light mode for accessibility."** A dark theme that meets AA is
  compliant. Nothing in WCAG requires a second one.
- **A missing skip link** on page chrome that is two links long.
- **`title` duplicating visible text** on something that truncates. That is the
  prescribed mitigation, not a defect — flag the truncation of a *focusable*
  element instead, which is the real rule.
- **`tabindex` proposed on a static element** so a tooltip can be focused. That
  invents a tab stop that leads nowhere, which is worse than the tooltip.

Precision beats recall, same as the prose review. Ten findings the author acts
on beat forty they scroll past.

## Dispatch

- **Claude Code** — delegate to the `design-reviewer` subagent
  (`.claude/agents/design-reviewer.md`), which runs this skill.
- **Codex** — spawn the `design-reviewer` agent by name
  (`.codex/agents/design-reviewer.toml`), which runs this skill.
- **No subagent support** — run this skill directly.

## Output

Start with a verdict line, then findings, worst first:

```
VERDICT: pass | changes requested  (N must-fix, N should-fix, N optional)
```

Each finding:

```
[must-fix|should-fix|optional] static/theme.css:283 — rule name (dimension)
  page: which page(s) and viewport(s) show it
  saw:  what you observed, and how — the screenshot, or the probe you ran
  fix:  the concrete change, as CSS or markup
  why:  one sentence, only if the rule name doesn't already say it
```

Severities:

- **must-fix** — content or navigation unreachable, an accessibility floor
  breached, layout broken at a supported width, or a token/system violation
  that will spread.
- **should-fix** — a clear hierarchy, spacing, fit or consistency problem that
  makes the page worse.
- **optional** — a judgement call worth the author's glance.

Say which pages and widths you looked at, and which you didn't — an unexamined
page is not a passing page. Mark each finding as a regression from this change
or pre-existing. If the change is clean, say so in one line and stop; don't
manufacture findings, and don't pad with a summary of what the design does
well.

You are read-only on the site. Propose the fix; the caller applies it.
