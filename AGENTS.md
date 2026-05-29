# AGENTS.md

Guidance for AI agents working in this repository.

## What this is

A small Go-based static site generator (SSG) that powers Simon Bracegirdle's
personal blog at `sbracegirdle.github.io`. It converts Markdown files in
`content/` into HTML in `build/` using a single HTML template, and generates an
index page listing all dated posts.

## Layout

- `main.go` — the entire generator (markdown → HTML, frontmatter parsing, index generation)
- `main_test.go`, `benchmark_test.go` — tests and benchmarks
- `content/` — blog posts as Markdown, named `yyyy-mm-dd-title.md`
- `template.html` — HTML template with `{{title}}` and `{{content}}` placeholders
- `build/` — generated output (git-ignored, not committed)
- `local-serve.sh` — local preview server (build + serve, optional `--watch`)
- `style.md` — Simon's writing style guidelines (apply when writing/editing posts)
- `.github/workflows/deploy.yml` — CI: test, build, deploy to GitHub Pages on push to `main`

## Commands

```bash
go build -o ssg ./main.go   # build the generator
./ssg                        # generate site into ./build
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

## Theme

All styling lives in the inline `<style>` block of `template.html` and is driven
by CSS custom properties declared on `:root`, grouped by concern:

- **Colour** — `--color-bg`, `--color-text`, `--color-muted`, `--color-accent`,
  `--color-accent-hover`, `--color-code-bg`, `--color-border`. Overridden in the
  `prefers-color-scheme: dark` block for dark mode.
- **Fonts** — `--font-serif` (headings), `--font-sans` (body), `--font-mono`
  (code). System-font stacks only; no external font requests.
- **Type scale** — `--text-xs` … `--text-2xl`.
- **Line height / tracking** — `--leading-tight`, `--leading-normal`,
  `--tracking-tight`.
- **Weights** — `--weight-normal`, `--weight-medium`, `--weight-bold`.
- **Spacing scale** — `--space-1` … `--space-6`.
- **Layout / borders / motion** — `--layout-width`, `--layout-pad-y`,
  `--layout-pad-x`, `--border-width`, `--radius`, `--transition`.

When adjusting the look, change the token on `:root` rather than hard-coding
values in rules; dark mode only needs the `--color-*` tokens re-declared. Keep
the design subtle and plain — serif headings, sans body, muted off-white
background, soft teal accent.

## Deployment

Pushing to `main` triggers the GitHub Actions workflow, which runs tests, builds,
generates the site, and deploys to GitHub Pages. Do not commit the `build/`
directory.
