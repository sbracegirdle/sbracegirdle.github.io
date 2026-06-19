# AGENTS.md

Guidance for AI agents working in this repository.

## What this is

A small Go-based static site generator (SSG) that powers Simon Bracegirdle's
personal blog at `sbracegirdle.github.io`. It converts Markdown files in
`content/` into HTML in `build/` using a single HTML template, and generates an
index page listing all dated posts.

## Layout

- `main.go` — the entire generator (markdown → HTML, frontmatter parsing, index generation, optional `--watch` serve with live reload)
- `main_test.go`, `benchmark_test.go` — tests and benchmarks
- `content/` — blog posts as Markdown, named `yyyy-mm-dd-title.md`
- `static/` — standalone resources (e.g. self-contained HTML pages) copied verbatim into `build/` without going through the markdown/template pipeline; link to them from the homepage or posts
- `template.html` — HTML template with `{{title}}` (tab title), `{{heading}}` (visible h1), and `{{content}}` placeholders
- `build/` — generated output (git-ignored, not committed)
- `local-serve.sh` — local preview server (build + serve, optional `--watch`)
- `style.md` — Simon's writing style guidelines (apply when writing/editing posts)
- `.github/workflows/deploy.yml` — CI: test, build, deploy to GitHub Pages on push to `main`

## Commands

```bash
go build -o ssg ./main.go   # build the generator
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
- Keep `main.go` dependency-light — it currently uses only `adrg/frontmatter`
  and `gomarkdown/markdown`. Prefer the standard library.
- When writing or editing post content, follow `style.md` (concise, approachable,
  informal, professional; avoid passive voice, weasel words, clichés).
- Standalone, self-contained resources live in `static/` and are copied as-is
  into `build/` (no rendering, no template wrapping). Use this for pages that
  ship their own markup/styling and shouldn't be wrapped in the blog template.
  Generated pages win on name collisions, so don't name a static file
  `index.html` or after a post.

## Theme

All styling lives in the inline `<style>` block of `template.html` and is driven
by CSS custom properties declared on `:root`, grouped by concern:

- **Colour** — `--color-bg`, `--color-text`, `--color-muted`, `--color-accent`,
  `--color-accent-hover`, `--color-code-bg`, `--color-border`. Overridden in the
  `prefers-color-scheme: dark` block for dark mode. A set of seven gruvbox
  accent tokens (`--c-red`, `--c-green`, `--c-yellow`, `--c-blue`, `--c-purple`,
  `--c-aqua`, `--c-orange`) drive the divider, headings, list markers, and panel
  borders — reused throughout for colour consistency.
- **Fonts** — `--font-serif` (headings), `--font-sans` (body), `--font-mono`
  (code). All three resolve to the same monospace stack — the TUI aesthetic is
  monospace throughout. System-font stacks only; no external font requests.
- **Type scale** — `--text-xs` … `--text-2xl`. Tightened for monospace widths.
- **Line height / tracking** — `--leading-tight`, `--leading-normal`,
  `--tracking-tight`.
- **Weights** — `--weight-normal`, `--weight-medium`, `--weight-bold`.
- **Spacing scale** — `--space-1` … `--space-6`.
- **Layout / borders / motion** — `--layout-width`, `--layout-pad-y`,
  `--layout-pad-x`, `--border-width`, `--radius`, `--transition`.

When adjusting the look, change the token on `:root` rather than hard-coding
values in rules; dark mode only needs the `--color-*` tokens re-declared. The
theme is a terminal (TUI) aesthetic: monospace everywhere, gruvbox-inspired
accents on neutral light/dark surfaces (green accent in light, phosphor-green in
dark), square corners
(`--radius: 0`), a header title bar, a blinking block cursor on the site
prompt (respects `prefers-reduced-motion`), and `›` markers on the post list.
Keep it readable — long-form prose still needs generous line height.

## Deployment

Pushing to `main` triggers the GitHub Actions workflow, which runs tests, builds,
generates the site, and deploys to GitHub Pages. Do not commit the `build/`
directory.
