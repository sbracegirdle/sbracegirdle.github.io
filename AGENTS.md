# AGENTS.md

Guidance for AI agents working in this repository.

## What this is

A small Go-based static site generator (SSG) that powers Simon Bracegirdle's
personal blog at `sbracegirdle.github.io`. It converts Markdown files in
`content/` into HTML in `build/` using a single HTML template, and generates an
index page listing all dated posts.

## Layout

- `main.go` — the generator (markdown → HTML, frontmatter parsing, index generation, optional `--watch` serve with live reload)
- `highlight.go` — tiny dependency-free syntax highlighter (rust, go, python, shell, yaml, js; unknown languages stay plain). At build time it renders every code block — fenced blocks in posts and `<script type="text/rust|shell">` blocks in static HTML — as the same line-numbered `pre.code` component
- `main_test.go`, `highlight_test.go`, `benchmark_test.go` — tests and benchmarks
- `content/` — blog posts as Markdown, named `yyyy-mm-dd-title.md`
- `static/` — standalone resources (e.g. self-contained HTML pages) copied verbatim into `build/` without going through the markdown/template pipeline; link to them from the homepage or posts
- `template.html` — HTML template with `{{title}}` (tab title), `{{heading}}` (visible h1), `{{file}}` (statusline filename), and `{{content}}` placeholders; links `/theme.css`
- `static/theme.css` — the site's single stylesheet: all tokens and components, linked by every page
- `build/` — generated output (git-ignored, not committed)
- `local-serve.sh` — local preview server (build + serve, optional `--watch`)
- `style.md` — Simon's writing style guidelines (apply when writing/editing posts)
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
```

## Conventions

- Posts live in `content/` named `yyyy-mm-dd-title.md`. The date prefix is parsed
  from the filename and drives index ordering (newest first); posts without a
  valid date are skipped from the index.
- Frontmatter (`title`, `description`) is optional. Without a title, one is
  derived from the filename; without a description, one is extracted from the
  first paragraph.
- Keep the generator dependency-light — it currently uses only `adrg/frontmatter`
  and `gomarkdown/markdown`. Prefer the standard library.
- When writing or editing post content, follow `style.md` (concise, approachable,
  informal, professional; avoid passive voice, weasel words, clichés).
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
  line height.
- `static/rust-quick-reference.html` and `static/style-guide.html` are
  standalone pages but link `/theme.css` like everything else.

## Deployment

Pushing to `main` triggers the GitHub Actions workflow, which runs tests, builds,
generates the site, and deploys to GitHub Pages. Do not commit the `build/`
directory.
