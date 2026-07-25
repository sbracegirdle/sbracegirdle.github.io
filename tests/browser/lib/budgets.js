// The site's performance budget, in one place so the spec and the Lighthouse
// runner can never disagree about what "too heavy" means.
//
// The numbers are deliberately close to what the site actually ships. A budget
// with a lot of slack in it never fires, and a budget that fires on every
// commit gets raised until it never fires. These sit roughly 1.5–2x current
// weight: enough room for a post with a long code block, not enough for a web
// font, an analytics tag, or a framework to slip in unnoticed.
//
// Raising a number is a decision, not a fix. Say in the commit message what got
// heavier and why it was worth it.

// Bytes a single page may pull from its own origin — the HTML plus theme.css
// plus anything else the page links. Off-origin images hot-linked from a post's
// prose are the author's call and are counted separately, never budgeted.
export const maxPageBytes = 160 * 1024;

// theme.css is loaded by every page on the site, so it is the one asset where a
// few kilobytes is a few kilobytes everywhere.
export const maxStylesheetBytes = 45 * 1024;

// Requests to our own origin. The site is HTML + one stylesheet; the headroom
// is for a favicon and an inline-able extra, not for a bundle.
export const maxSameOriginRequests = 5;

// Lighthouse category scores, 0–100. A static, monospace, zero-JavaScript site
// has no excuse for less than a perfect run, so these are set where a real
// regression is the only thing that can breach them.
export const minScores = {
  performance: 98,
  accessibility: 100,
  "best-practices": 100,
  seo: 100,
};

// Where a category score is deliberately not perfect. Each entry needs a reason
// that would survive being read back in a year — this is the escape hatch that
// turns into "we exempted everything" if it is used casually.
export const scoreExemptions = {
  "/404.html": {
    seo: "the 404 page is noindex on purpose, so is-crawlable fails by design",
  },
};

export function formatBytes(n) {
  return n < 1024 ? `${n} B` : `${(n / 1024).toFixed(1)} KB`;
}
