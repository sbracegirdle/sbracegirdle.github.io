import { test, expect } from "@playwright/test";
import { samplePages } from "../lib/site.js";

// The accessibility floor, asserted from the rendered DOM rather than from the
// CSS or the template. These are the checks that cost nothing to run and never
// regress silently afterwards; they are not a substitute for tabbing through a
// page and looking at it. The rules and their success criteria live in
// .agents/skills/design-review/references/accessibility.md.

for (const urlPath of samplePages()) {
  test(`semantics and names: ${urlPath}`, async ({ page }) => {
    await page.goto(urlPath);

    const report = await page.evaluate(() => {
      const text = (el) => (el.textContent || "").replace(/\s+/g, " ").trim();
      const named = (el) =>
        text(el) ||
        el.getAttribute("aria-label") ||
        [...el.querySelectorAll("img")].some((i) => (i.getAttribute("alt") || "").trim());

      return {
        lang: document.documentElement.getAttribute("lang"),
        title: (document.title || "").trim(),
        h1s: [...document.querySelectorAll("h1")].map(text),
        // Levels must descend without skipping: h2 → h4 leaves a hole in the
        // outline that a screen reader reads as a missing section.
        headingSkips: (() => {
          const skips = [];
          let last = 0;
          for (const h of document.querySelectorAll("h1,h2,h3,h4,h5,h6")) {
            const level = Number(h.tagName[1]);
            if (last && level > last + 1) skips.push(`${"h" + last} → ${"h" + level}: ${text(h)}`);
            last = level;
          }
          return skips;
        })(),
        namelessLinks: [...document.querySelectorAll("a[href]")]
          .filter((a) => !named(a))
          .map((a) => a.getAttribute("href")),
        // A link with no href is not a link; if it acts, it should be a button.
        hrefless: [...document.querySelectorAll("a:not([href])")].map(text),
        imgsWithoutAlt: [...document.querySelectorAll("img")]
          .filter((i) => i.getAttribute("alt") === null)
          .map((i) => i.getAttribute("src")),
        // aria-label on a bare div or span is dropped on the floor: role
        // `generic` does not support an accessible name. The site shipped
        // exactly this on its statusline, and it looked fine the whole time.
        labelledGenerics: [...document.querySelectorAll("div[aria-label], span[aria-label]")]
          .filter((el) => !el.hasAttribute("role"))
          .map((el) => `${el.tagName.toLowerCase()}.${el.className}`),
        // aria-hidden must never swallow something you can tab to.
        hiddenFocusables: [...document.querySelectorAll('[aria-hidden="true"]')]
          .filter((el) => el.querySelector("a[href], button, input, select, textarea, [tabindex]"))
          .map((el) => `${el.tagName.toLowerCase()}.${el.className}`),
      };
    });

    expect(report.lang, "html needs a lang attribute (SC 3.1.1)").toBeTruthy();
    expect(report.title, "every page needs a descriptive title (SC 2.4.2)").not.toBe("");
    expect(report.h1s.length, `expected exactly one h1, got ${report.h1s.join(" | ")}`).toBe(1);
    expect(report.headingSkips, "heading levels skip a step (SC 1.3.1)").toEqual([]);
    expect(report.namelessLinks, "links with no accessible name (SC 2.4.4)").toEqual([]);
    expect(report.hrefless, "anchors without href are not links").toEqual([]);
    expect(report.imgsWithoutAlt, "images need alt, empty if decorative (SC 1.1.1)").toEqual([]);
    expect(report.labelledGenerics, "aria-label on a roleless div/span is ignored").toEqual([]);
    expect(report.hiddenFocusables, "aria-hidden hides something focusable").toEqual([]);
  });
}

// Truncation is a CSS effect, so the full string stays in the accessibility
// tree and a screen reader reads all of it — but a sighted mouse user sees the
// stub. Anything that can ellipsis owes them a title with the same text.
test("every truncating element exposes its full text", async ({ page }) => {
  await page.goto("/index.html");

  const found = await page.evaluate(() =>
    [...document.querySelectorAll("body *")]
      .filter((el) => getComputedStyle(el).textOverflow === "ellipsis")
      .filter((el) => getComputedStyle(el).display !== "none")
      .map((el) => ({
        what: `${el.tagName.toLowerCase()}.${el.className}`,
        text: (el.textContent || "").trim(),
        title: (el.getAttribute("title") || "").trim(),
      })),
  );

  // Without this the assertion below passes on an empty set, and would keep
  // passing if the modifiers were renamed out from under it.
  expect(found.length, "no truncating elements found — has the selector gone stale?").toBeGreaterThan(0);
  expect(
    found.filter((el) => el.title !== el.text),
    "an element truncates without a matching title",
  ).toEqual([]);
});

// A focus ring you can't see is the same as no focus ring (SC 2.4.7), and it
// is the first thing an `outline: none` somewhere else in the CSS takes out.
test("keyboard focus is visible on links", async ({ page }) => {
  await page.goto("/index.html");
  const link = page.locator("main a, .post-list a").first();
  await link.focus();

  const ring = await link.evaluate((el) => {
    const s = getComputedStyle(el);
    return { style: s.outlineStyle, width: parseFloat(s.outlineWidth) || 0 };
  });

  expect(ring.style, "focused link has no outline style").not.toBe("none");
  expect(ring.width, "focused link has a zero-width outline").toBeGreaterThan(0);
});

// SC 2.5.8 asks for 24×24 CSS pixels of pointer target. Statusline segments are
// deliberately small chrome, so they are the ones worth measuring rather than
// assuming: they clear it on padding and line height, not by much.
test("interactive chrome meets the minimum target size", async ({ page }) => {
  await page.goto("/index.html");

  const targets = await page.evaluate(() =>
    [...document.querySelectorAll("header a, footer a")]
      .filter((el) => getComputedStyle(el).display !== "none")
      .map((el) => {
        const r = el.getBoundingClientRect();
        return { text: (el.textContent || "").trim(), w: Math.round(r.width), h: Math.round(r.height) };
      }),
  );

  expect(targets.length, "no chrome links found — has the markup changed?").toBeGreaterThan(0);
  expect(
    targets.filter((el) => el.h < 24 || el.w < 24),
    "chrome links smaller than 24×24 (SC 2.5.8)",
  ).toEqual([]);
});

// The truncation check above runs on the homepage. The hand-written pages in
// static/ each build their own statusline with their own .seg-shrink, and none
// of them was covered — so the rule held exactly where the generator applied it
// and nowhere else.
for (const urlPath of samplePages()) {
  test(`truncating elements expose their full text: ${urlPath}`, async ({ page }) => {
    await page.goto(urlPath);

    const found = await page.evaluate(() =>
      [...document.querySelectorAll("body *")]
        .filter((el) => getComputedStyle(el).textOverflow === "ellipsis")
        .filter((el) => getComputedStyle(el).display !== "none")
        .map((el) => ({
          what: `${el.tagName.toLowerCase()}.${el.className}`,
          text: (el.textContent || "").trim(),
          title: (el.getAttribute("title") || "").trim(),
        })),
    );

    expect(
      found.filter((el) => el.title !== el.text),
      "an element truncates without a matching title",
    ).toEqual([]);
  });
}

// title is the only affordance a no-JavaScript page has for a truncated label,
// and it reaches neither a keyboard user nor a touch user. That is why the rule
// is to keep .seg-shrink off anything focusable rather than to paper over it
// with a tooltip — a truncated link is unreadable for them with no way back.
// Making the filename segment a link is a one-character diff.
for (const urlPath of samplePages()) {
  test(`nothing that truncates is interactive: ${urlPath}`, async ({ page }) => {
    await page.goto(urlPath);

    const interactive = await page.$$eval(".seg-shrink", (els) =>
      els
        .filter(
          (el) =>
            el.matches("a[href], button, input, select, textarea, [tabindex]") ||
            el.querySelector("a[href], button, input, select, textarea, [tabindex]"),
        )
        .map((el) => `${el.tagName.toLowerCase()}.${el.className}: ${el.textContent.trim()}`),
    );

    expect(
      interactive,
      "a truncating element is focusable — title never reaches a keyboard or touch user",
    ).toEqual([]);
  });
}

// A region that scrolls is a region a keyboard user has to be able to reach and
// scroll, which means a tabindex and a name (SC 2.1.1). sports.html gets this
// right today; nothing kept it that way.
test("scrollable regions are keyboard reachable and named", async ({ page }) => {
  await page.goto("/sports.html");

  const regions = await page.$$eval(".scroll-x", (els) =>
    els.map((el) => ({
      what: el.className,
      scrolls: el.scrollWidth - el.clientWidth > 1,
      tabindex: el.getAttribute("tabindex"),
      role: el.getAttribute("role"),
      named: Boolean(
        el.getAttribute("aria-label") ||
          (el.getAttribute("aria-labelledby") &&
            document.getElementById(el.getAttribute("aria-labelledby"))),
      ),
    })),
  );

  expect(regions.length, "no .scroll-x region found — has the markup changed?").toBeGreaterThan(0);

  for (const r of regions) {
    if (!r.scrolls) continue;
    expect(r.tabindex, `${r.what} scrolls but is not focusable`).toBe("0");
    expect(r.role, `${r.what} scrolls but has no role`).toBeTruthy();
    expect(r.named, `${r.what} scrolls but has no accessible name`).toBe(true);
  }
});
