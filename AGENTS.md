# AGENTS.md

Guidance for AI agents working in this repository.

## What this is

Go static site generator for Simon's blog, `sbracegirdle.github.io`.
Markdown in `content/` becomes HTML in `build/`, index included.

## Layout

- `main.go` generator; `tags.go` tags; `feed.go` feed/sitemap/robots
- `highlight.go` syntax highlighting; `*_test.go` Go tests
- `agents_test.go` guards skill wiring; `tests/browser/` Playwright
- `content/` posts; `static/` verbatim pages; `template.html` template
- `static/theme.css` stylesheet; `static/game.js` homepage arcade
- `static/style-guide.html` design spec; `static/sports.html` sports calendar
- `build/` generated; `local-serve.sh` preview; `style.md` voice
- `.github/workflows/deploy.yml` CI deploys Pages

## Skills and subagents

Skills live once under `.agents/skills/`, symlinked for both agents.
Invoke as needed; `agent-conventions` covers adding or changing.
`agents_test.go` enforces the wiring.

- `prose-review` — weak writing and AI tells
- `browser-test` — headless browser verification
- `design-review` — visual and accessibility quality
- `perf-audit` — page weight and Lighthouse vs budgets
- `sports-update` — refresh `static/sports.html`
- `agent-conventions` — add or change a skill

## Commands

```bash
go build -o ssg .             # build the generator
./ssg --watch [--port N]      # generate; serve with live reload
go test ./...                 # tests; CI fails on errors
./local-serve.sh              # build + serve on :8080

cd tests/browser
npm install                   # first time only
npx playwright install chromium
npm test                      # rebuild, run headless suite
node shot.mjs --both /        # screenshot pages, report
node lighthouse.mjs --sample  # Lighthouse vs budgets
```

## Conventions

- Do not edit AGENTS.md unless asked.
- Do not add tests until asked.
- Posts: `content/yyyy-mm-dd-title.md`; filename date orders index.
- Frontmatter optional; tags fold to sorted lowercase slugs.
- `renderPage` HTML-escapes every scalar — never bypass it.
- Site identity: one `const` block in `main.go`.
- Builds emit feed, sitemap, robots, and noindex 404.
- Feed dates track the newest post; rebuilds stay identical.
- Generated links are root-absolute (`/post.html`).
- Dependencies: `adrg/frontmatter` and `gomarkdown/markdown` only.
- Post prose follows `style.md`; `prose-review` checks it.
- Never attribute opinions to Simon; `ai-tells.md` has detail.
- `static/` copies verbatim; generated pages win name collisions.
- Don't name static files `index.html` or after posts.

## Theme

Dark only: night terminal on Rosé Pine, monospace, square corners.

- `static/style-guide.html` is the spec — read before styling.
- Keep the style guide in step with `theme.css` changes.
- All CSS in `theme.css`; tokens only, never raw hex.
- Each hue has one job; pine and muted stay decorative.
- Zero theme JS and external requests; the arcade excepted.
- Hierarchy from structure; headings descend one at a time.
- Nothing widens the page on a phone.
- Nothing clips navigation; wrap rather than hide overflow.
- Truncation carries `title`; never truncate anything focusable.
- `.wide` layout mode belongs to `static/` pages.
- Posts may hot-link images; the theme requests nothing.

## Deployment

Push to `main`: CI tests, builds, deploys GitHub Pages.
Never commit `build/`.
