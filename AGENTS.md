# AGENTS.md

Guidance for AI agents working in this repository.

## What this is

A small Go-based static site generator (SSG) that powers Simon Bracegirdle's
personal blog at `sbracegirdle.github.io`. It converts Markdown files in
`content/` into HTML in `build/` using a single HTML template, and generates an
index page listing all dated posts.

## Layout

- `main.go` — the generator (markdown → HTML, frontmatter parsing, page rendering, index/archive/404 generation, optional `--watch` serve with live reload)
- `tags.go` — tag parsing and normalisation, the tag chips on posts, per-tag listing pages under `build/tags/`, and the `tags.html` index
- `feed.go` — machine-readable outputs: the RSS feed (`feed.xml`), `sitemap.xml`, and `robots.txt`
- `highlight.go` — tiny dependency-free syntax highlighter (rust, go, python, shell, yaml, js; unknown languages stay plain). At build time it renders every code block — fenced blocks in posts and `<script type="text/rust|shell">` blocks in static HTML — as the same line-numbered `pre.code` component
- `main_test.go`, `tags_test.go`, `feed_test.go`, `highlight_test.go`, `benchmark_test.go` — Go tests and benchmarks
- `agents_test.go` — guards the cross-agent wiring below: every skill symlinked into both agent directories, every subagent shipped for both
- `tests/browser/` — headless Playwright suite (`specs/`) plus the exploratory tooling (`shot.mjs` screenshots and reports, `serve.mjs` a GitHub-Pages-shaped static server). Its own npm package; not part of the Go module
- `content/` — blog posts as Markdown, named `yyyy-mm-dd-title.md`
- `static/` — standalone resources (e.g. self-contained HTML pages) copied verbatim into `build/` without going through the markdown/template pipeline; link to them from the homepage or posts
- `template.html` — HTML template with `{{title}}` (tab title and `og:title`), `{{heading}}` (visible h1), `{{file}}` (statusline filename), `{{description}}`, `{{canonical}}`, `{{ogtype}}`, `{{head_extra}}` (raw extra `<head>` markup) and `{{content}}` placeholders; links `/theme.css`
- `static/theme.css` — the site's single stylesheet: all tokens and components, linked by every page
- `build/` — generated output (git-ignored, not committed)
- `local-serve.sh` — local preview server (build + serve, optional `--watch`)
- `style.md` — Simon's writing style guidelines (apply when writing/editing posts)
- `.agents/skills/prose-review/` — the prose review skill (scope, process, house voice, output format, plus `references/write-good.md` and `references/ai-tells.md`); symlinked into `.claude/skills/` and `.codex/skills/` so both agents discover the one copy
- `.agents/skills/browser-test/` — the browser testing skill (the two phases, commands, spec coverage, reporting); symlinked into `.claude/skills/` and `.codex/skills/` the same way
- `.agents/skills/design-review/` — the design review skill (design-system fidelity, hierarchy, fit and containment, responsive, semantics and the WCAG 2.2 AA floor, plus `references/design-standards.md` and `references/accessibility.md`); symlinked the same way
- `.claude/agents/prose-reviewer.md`, `.codex/agents/prose-reviewer.toml` — the same reviewer subagent for Claude Code and Codex; both are thin wrappers that run the `prose-review` skill
- `.claude/agents/browser-tester.md`, `.codex/agents/browser-tester.toml` — the same tester subagent for both, thin wrappers that run the `browser-test` skill
- `.claude/agents/design-reviewer.md`, `.codex/agents/design-reviewer.toml` — the same design reviewer for both, thin wrappers that run the `design-review` skill
- `static/style-guide.html` — visual design system reference (tokens, components, usage rules); deployed at `/style-guide.html`, linked from the site footer
- `.github/workflows/deploy.yml` — CI: test, build, deploy to GitHub Pages on push to `main`

## Commands

```bash
go build -o ssg .            # build the generator
./ssg                        # generate site into ./build
./ssg --watch [--port N]     # native watch + serve with live reload (Goodreads cached for 10m)
go test -v ./...             # run tests (CI runs this and fails on errors)
go test -bench=. ./...       # run benchmarks
./local-serve.sh             # build + serve on :8080 (--port N, --watch)

cd tests/browser
npm install                  # first time only
npx playwright install chromium
npm test                     # rebuild the site, then run the headless suite
node shot.mjs --both /       # exploratory: screenshot pages and report on them
```

## Cross-agent skills and subagents

Skills and subagents in this repo work in both Claude Code and Codex. Each
skill exists once; the two agents reach it through symlinks. Follow this
whenever you add or change one — `agents_test.go` fails the build if you don't.

```
.agents/skills/<skill-name>/SKILL.md        the one real copy
.agents/skills/<skill-name>/references/     supporting material, optional
.claude/skills/<skill-name>   ->  ../../.agents/skills/<skill-name>
.codex/skills/<skill-name>    ->  ../../.agents/skills/<skill-name>
.claude/agents/<agent-name>.md              Claude Code wrapper
.codex/agents/<agent-name>.toml             Codex wrapper
```

To add one:

1. Write `.agents/skills/<skill-name>/SKILL.md`. Its frontmatter `name:` must
   match the directory name, and `description:` should say when to use it —
   that line is all an agent sees when deciding whether to load the skill.
   Put anything long in `references/` and link to it, so the skill itself stays
   short enough to be read every time.
2. Symlink it into both agent directories, relative so the tree stays portable:

   ```bash
   ln -s ../../.agents/skills/<skill-name> .claude/skills/<skill-name>
   ln -s ../../.agents/skills/<skill-name> .codex/skills/<skill-name>
   ```

3. If a subagent drives the skill, write both wrappers. Keep them thin: the
   skill holds the process, and the wrapper only says who the agent is, what to
   review, and the rules of engagement. Never let the two wrappers drift into
   different instructions — the whole point is that both agents behave the same.
   - `.claude/agents/<agent-name>.md` — YAML frontmatter (`name`, `description`,
     `tools`, `model: inherit`), then the instructions as the body. Write
     `description` so it says when the agent must run; Claude Code uses it to
     decide when to delegate.
   - `.codex/agents/<agent-name>.toml` — the same content as `name`,
     `description`, `sandbox_mode` and a `developer_instructions = """…"""`
     block. Use `read-only` for a reviewer that only reads files, and
     `workspace-write` for one that has to build the site or drive a browser.
4. Add the skill and both wrappers to the Layout list above, and give the skill
   its own section below if it's a gate.
5. Run `go test ./...`. `agents_test.go` checks the frontmatter name, that both
   symlinks exist and resolve to the shared copy, that every subagent exists for
   both agents, and that each wrapper names the skill it runs.

Reviewers report; they don't fix. A subagent proposes changes and the caller
applies them — so the caller stays responsible for what lands, and a wrong
finding gets argued with rather than silently committed.

## Prose review is mandatory

Any change that adds or edits prose **must** be reviewed by the `prose-reviewer`
subagent before you report the work as done. This is a gate, not a suggestion —
treat it like the test suite.

Prose means: `content/*.md` body text and the `title`/`description` frontmatter;
`README.md`, `AGENTS.md`, `style.md`; user-visible copy in `template.html` and
`static/*.html`; and Go string literals that render as page text. Code,
identifiers, and code fences are not prose. The skill has the exact scope.

The review itself lives in the `prose-review` skill at
`.agents/skills/prose-review/`, symlinked into `.claude/skills/` and
`.codex/skills/`. One copy, both agents.

- **Claude Code** — delegate to the `prose-reviewer` subagent
  (`.claude/agents/prose-reviewer.md`), which runs the skill.
- **Codex** — spawn the `prose-reviewer` agent by name
  (`.codex/agents/prose-reviewer.toml`), which runs the skill.
- **No subagent support** — invoke the `prose-review` skill yourself, or read
  `.agents/skills/prose-review/SKILL.md` directly. Either way make it a separate
  pass over the diff, after drafting. Never fold it into the drafting step; the
  review has to look at finished text.

Run it on the diff once the prose is drafted. Apply the must-fix findings, use
your judgement on should-fix and optional, and tell Simon what you changed and
what you left. The reviewer is read-only — it proposes replacements, you apply
them. If a finding is wrong, say why rather than applying it; the skill lists
the false positives it is meant to suppress, so a bad finding is a signal the
skill needs an edit.

## Browser testing is mandatory

Any change that affects what a visitor sees **must** be verified in a real
headless browser before you report the work as done. This is a gate, not a
suggestion — the same standing as the Go tests and the prose review.

That means: posts and pages in `content/`, `template.html`, `static/theme.css`,
`static/*.html`, and any generator change that alters emitted markup. A change
confined to Go internals, tests, or docs with no rendered output doesn't need it.

Two phases, both required:

1. **The static suite** — `cd tests/browser && npm test`. Rebuilds the site and
   runs every spec against it: pages rendering, head metadata, theme applied,
   layout at 390px and 320px, statusline segments neither clipped nor lost, the
   accessibility floor (`a11y.spec.js` — semantics, accessible names, heading
   order, truncation, focus ring, target size), tag navigation, internal links,
   feed/sitemap/robots/404.
2. **An exploratory pass** — `node shot.mjs --both <paths>`, then *look at* the
   screenshots. The suite only catches what someone already thought to assert;
   this phase is for everything else.

The process lives in the `browser-test` skill at `.agents/skills/browser-test/`,
symlinked into `.claude/skills/` and `.codex/skills/`. One copy, both agents.

- **Claude Code** — delegate to the `browser-tester` subagent
  (`.claude/agents/browser-tester.md`), which runs the skill.
- **Codex** — spawn the `browser-tester` agent by name
  (`.codex/agents/browser-tester.toml`), which runs the skill.
- **No subagent support** — invoke the `browser-test` skill yourself, or read
  `.agents/skills/browser-test/SKILL.md` directly.

**Headless only.** Never `--headed`, never `--debug`, never
`headless: false` — a browser window on the host steals focus from whoever is
at the keyboard. `tests/browser/global-setup.js` enforces this by refusing to
run when the resolved config asks for a headed browser, so `--headed` fails the
run rather than opening a window. Read the saved screenshot and trace instead.

The tester reports; it doesn't fix the site. Apply the findings yourself, and
say which you applied and which you left.

## Design review is mandatory

Any change to how the site **looks**, or to the markup it emits, must go through
the `design-reviewer` subagent before you report the work as done. Third gate,
same standing as the other two.

That means `static/theme.css`, `template.html`, `static/*.html`, and generator
changes that alter layout — plus any new component or page. Editing a post's
words isn't a design change; adding a component to render them is.

The browser suite and the design review look at the same pages for different
things. `browser-test` asks whether the page works: it renders, it doesn't
error, links resolve, the page doesn't overflow. `design-review` asks whether
it's *right*: hierarchy, token discipline, text fit, contrast, focus, spacing.
Between them sits the failure mode neither the Go tests nor a page-width
assertion can see — a container clipping its own content, where the page width
stays correct and the content is simply gone. The footer statusline shipped
that way: `style guide` was unreachable at every width, desktop included.

The process lives in the `design-review` skill at
`.agents/skills/design-review/`, wired for both agents per the standard above.
Delegate to `design-reviewer` in Claude Code, spawn `design-reviewer` in Codex,
or invoke the skill yourself if you have no subagents.

**The design system is the brief.** `static/style-guide.html` is the
specification; read it before reviewing or changing anything. The reviewer
enforces the system that exists — it never proposes a redesign, a typeface
pairing, a gradient or a light mode, and the skill lists those false positives
explicitly. General standards in `references/design-standards.md` apply on top
of the house system, never against it.

**Accessibility is the one thing that outranks the brief.** Semantics, ARIA,
accessible names, keyboard access, contrast and target size are a floor, and a
failure there is a finding even when it is pre-existing and even when the look
was deliberate. `references/accessibility.md` holds the
detail — WCAG 2.2 AA, the rules for truncation and tooltips, and the false
positives that come with the territory (redundant ARIA, AAA ratios, "add a
light mode"). `a11y.spec.js` in the browser suite fixes the mechanical half of
it in place.

Headless only, exactly as above. The reviewer reports; it doesn't fix the site.

## Conventions

- Posts live in `content/` named `yyyy-mm-dd-title.md`. The date prefix is parsed
  from the filename and drives index ordering (newest first); posts without a
  valid date are skipped from the index.
- Frontmatter (`title`, `description`, `tags`) is optional. Without a title, one
  is derived from the filename; without a description, one is extracted from the
  first paragraph. `tags` accepts a space-separated scalar
  (`tags: devops ci aws`) or a YAML list; either way tags are folded to
  lowercase hyphenated slugs, deduplicated and sorted, so `code_review` and
  `Code Review` land on the same tag.
- Every page goes through `renderPage`, which fills the template placeholders
  and HTML-escapes each scalar. Titles and descriptions reach attribute values
  (`meta description`, `og:*`), so never bypass it with a raw `strings.Replace`
  — one unescaped quote in a description silently breaks the whole `<head>`.
  `Content` and `HeadExtra` are inserted raw and substituted last, which is why
  a post can safely contain the literal text `{{title}}` in a code block.
- Site identity (URL, name, author, description) lives in one `const` block at
  the top of `main.go`. Canonical URLs, the feed, and the sitemap all derive
  from it — `canonicalURL` is the only place a public URL is assembled.
- Every build also emits `feed.xml` (RSS 2.0, summaries not full text),
  `sitemap.xml`, `robots.txt`, and a noindex `404.html` that GitHub Pages serves
  for unknown paths. The feed's `lastBuildDate` tracks the newest post rather
  than the wall clock, so an unchanged site rebuilds byte-identically.
- Links between generated pages are root-absolute (`/post.html`), so the same
  markup works from the site root and from the nested `/tags/` pages.
- Keep the generator dependency-light — it currently uses only `adrg/frontmatter`
  and `gomarkdown/markdown`. Prefer the standard library.
- When writing or editing post content, follow `style.md` (concise, approachable,
  informal, professional; avoid passive voice, weasel words, clichés), then put
  the result through the mandatory prose review above.
- Standalone, self-contained resources live in `static/` and are copied into
  `build/` without rendering or template wrapping — except that
  `<script type="text/rust|shell">` blocks in static HTML are pre-rendered to
  highlighted code by `highlight.go`. Use static/ for pages that ship their own
  markup/styling. Generated pages win on name collisions, so don't name a
  static file `index.html` or after a post.

## Theme

Dark only — "night terminal" on the Rosé Pine palette. All CSS lives in one
file: `static/theme.css` (tokens + every component), linked as `/theme.css` by
the template, the style guide, and static pages — never duplicate its rules
into a page. `static/style-guide.html` documents it (token reference, component
demos, usage rules, verified contrast ratios); read the guide before any
styling change, and keep its docs in step when theme.css changes. Pages may
carry a few page-specific rules inline (e.g. the quick ref's two-column TOC).

- Tokens are grouped by concern: colour (surfaces `--color-bg/surface/overlay`,
  borders `--color-border[-strong]`, text `--color-text/subtle/muted`, hues
  `--c-love/gold/rose/pine/foam/iris`), typography (`--font-mono`,
  `--text-xs`…`--text-3xl`, leading, tracking, weights), spacing
  (`--space-1`…`--space-6`), layout, borders & motion. Never hard-code a hex or
  magic number in a rule — always go through a token.
- Each hue has one job: foam links/primary, gold numbers/dates, rose title
  fill/hover/emphasis, iris h3 sidebar/forms, love danger. `--c-pine` and
  `--color-muted` are decorative only — they fail AA for body-size text.
- Hierarchy comes from structure, not glyph prefixes: inverted rose h1 block,
  gold auto-numbered h2 with a fill rule, iris-sidebar h3, small-caps h4.
  Page chrome is a statusline header/footer; the index is a `.post-list`
  (gold ISO dates, dashed separators).
- Syntax tokens (`.t-kw` iris, `.t-type/.t-fn` foam, `.t-str/.t-macro/.t-flag`
  gold, `.t-num/.t-lifetime` rose, `.t-comment` subtle italic — never muted)
  are emitted by `highlight.go` and styled in theme.css.
- Monospace everywhere, square corners (`--radius: 0`), zero JS and zero
  external requests in the theme; one animation (prompt cursor blink,
  disabled under `prefers-reduced-motion`). Long-form prose keeps generous
  line height. Post prose may hot-link images, which is the author's call —
  the "no external requests" rule is about the theme, and the browser suite
  only fails on off-origin *code*.
- Nothing may widen the page on a phone. Inline `code`, links and `h1` break
  mid-token; code blocks scroll inside `pre`; flex rows (`.pager`, the style
  guide's specimen rows) wrap; and hidden decoration is hidden with `display`
  rather than `opacity` alone, because an invisible box still sits in the
  scrollable overflow. Note that `overflow-wrap: break-word` does not shrink a
  shrink-to-fit box like the `h1` block — that needs `anywhere`.
  `theme.spec.js` checks every page at 390px and 320px.
- Nothing may clip navigation either. A page-width check can't see a container
  hiding its own overflow, which is how the footer statusline lost `style guide`
  and half of `feed` at every width, desktop included. The statusline wraps
  instead, and prefer wrapping to `overflow: hidden` on any row that holds
  links.
- **Give a link a row before you make it give ground.** A bar that carries links
  as well as status gets a second row: `.statusline-stack` on the bar, each line
  wrapped in a `.statusline-row` that draws the surface tone itself, so no
  trailing `.fill` is needed. Header and footer both do this — status above,
  links below. The colophon was a *truncated link* at every desktop width
  (308 of 449 pixels at 1280) until it got a row of its own.
- The nav row is the command line beneath the status bar. `.statusline-nav`
  drops the fill tone, the foam block and the chips for a foam `:` and plain
  links, and drops the segment padding so the `:` sits on the page's left rail
  with `❯`, the `h1` block and the prose. Two rows of solid segments read as
  equals. Each nav is a landmark and needs its own `aria-label` — `Primary` and
  `Footer`.
- `.seg-note` wraps inside its own box rather than truncating; it's for prose in
  a bar of `nowrap` segments. `.seg-shrink` is for the one-token label that
  genuinely can't wrap (a filename), truncates to an ellipsis, is hidden below
  640px, and never goes on a link or anything else focusable.
- Anything that truncates carries a `title` with the same text. The ellipsis is
  a CSS effect, so the whole string stays in the accessibility tree and a screen
  reader reads it — the person left with a stub is the sighted mouse user, and a
  page with no JavaScript has no other hover affordance. `title` never reaches a
  keyboard or touch user, which is why the rule is to keep `.seg-shrink` off
  anything clickable rather than to paper over one with a tooltip.
- Heading levels are the document outline, so they descend one at a time and
  nothing joins them for its looks. Use `.label` for a small-caps caption that
  isn't a section — the do/don't cards in the style guide were ten h4s under an
  h2, which both skipped a level and buried the real structure.
- `static/rust-quick-reference.html` and `static/style-guide.html` are
  standalone pages but link `/theme.css` like everything else.

## Deployment

Pushing to `main` triggers the GitHub Actions workflow, which runs tests, builds,
generates the site, and deploys to GitHub Pages. Do not commit the `build/`
directory.
