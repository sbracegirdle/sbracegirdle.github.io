import { test, expect } from "@playwright/test";
import { aTaggedPost, postPages, samplePages, tagPages } from "../lib/site.js";

// Reads a custom property the way the browser resolves it, rather than
// hard-coding a hex in the test — the token stays the single source of truth.
async function resolveToken(page, token) {
  return page.evaluate((name) => {
    const probe = document.createElement("div");
    probe.style.backgroundColor = `var(${name})`;
    document.documentElement.appendChild(probe);
    const value = getComputedStyle(probe).backgroundColor;
    probe.remove();
    return value;
  }, token);
}

test("theme.css is applied, not just linked", async ({ page }) => {
  await page.goto("/index.html");

  const tokenBg = await resolveToken(page, "--color-bg");
  expect(tokenBg, "--color-bg did not resolve — theme.css is missing or 404ing").not.toBe(
    "rgba(0, 0, 0, 0)",
  );

  const styles = await page.evaluate(() => {
    const body = getComputedStyle(document.body);
    return { background: body.backgroundColor, font: body.fontFamily };
  });
  expect(styles.background).toBe(tokenBg);
  expect(styles.font.toLowerCase()).toContain("mono");
});

test("static pages share the same stylesheet", async ({ page }) => {
  for (const urlPath of ["/style-guide.html", "/rust-quick-reference.html"]) {
    await page.goto(urlPath);
    await expect(page.locator('link[rel="stylesheet"][href="/theme.css"]'), urlPath).toHaveCount(1);
    const bg = await page.evaluate(() => getComputedStyle(document.body).backgroundColor);
    expect(bg, urlPath).toBe(await resolveToken(page, "--color-bg"));
  }
});

// Long code blocks, wide tables, unbreakable words in a title and hidden
// decoration are the usual culprits for blowing out a narrow layout. 390px is a
// current phone; 320px is the narrowest still worth supporting and catches
// things 390px lets through.
for (const width of [390, 320]) {
  test.describe(`layout at ${width}px`, () => {
    test.use({ viewport: { width, height: 844 } });

    for (const urlPath of new Set([...postPages(), ...samplePages()])) {
      test(`no horizontal overflow: ${urlPath}`, async ({ page }) => {
        await page.goto(urlPath);
        const overflow = await page.evaluate(
          () => document.documentElement.scrollWidth - window.innerWidth,
        );
        expect(
          overflow,
          `${urlPath} overflows by ${overflow}px at ${width}px wide`,
        ).toBeLessThanOrEqual(1);
      });
    }
  });
}

// Page-width overflow is only half the story. A flex row that hides its own
// overflow keeps the page exactly the right width and throws away whatever
// didn't fit — which is how the footer statusline dropped "style guide" and
// half of "feed" at every width, desktop included, without failing a thing.
// 1280px is here deliberately: the bug was not mobile-only.
for (const width of [1280, 700, 390, 320]) {
  test.describe(`statusline at ${width}px`, () => {
    test.use({ viewport: { width, height: 844 } });

    // A long post filename is what stresses the header bar.
    for (const urlPath of ["/index.html", "/2022-06-14-robo-sim-pipe-github-actions.html"]) {
      test(`no segment is clipped or lost: ${urlPath}`, async ({ page }) => {
        await page.goto(urlPath);

        const bars = await page.evaluate(() =>
          [...document.querySelectorAll(".statusline")].map((bar) => {
            const right = bar.getBoundingClientRect().right;
            return {
              overflow: bar.scrollWidth - bar.clientWidth,
              // A segment escaping the bar's right edge is unreachable, and a
              // link that shows an ellipsis is fine — one that shows nothing
              // of its label is not.
              lost: [...bar.children]
                .filter((el) => getComputedStyle(el).display !== "none")
                .filter((el) => el.getBoundingClientRect().right > right + 1)
                .map((el) => el.textContent.trim()),
              emptyLinks: [...bar.querySelectorAll("a")]
                .filter((el) => getComputedStyle(el).display !== "none")
                .filter((el) => el.getBoundingClientRect().width < 8)
                .map((el) => el.textContent.trim()),
            };
          }),
        );

        expect(bars.length).toBeGreaterThan(0);
        for (const bar of bars) {
          expect(bar.lost, `segments pushed past the bar edge: ${bar.lost.join(", ")}`).toEqual([]);
          expect(
            bar.emptyLinks,
            `links squeezed to nothing: ${bar.emptyLinks.join(", ")}`,
          ).toEqual([]);
          expect(bar.overflow, "statusline hides its own overflow").toBeLessThanOrEqual(1);
        }
      });
    }
  });
}

// Every link the footer offers must be reachable, not merely present in the
// markup — the clipped version still had all five in the DOM.
for (const width of [1280, 390, 320]) {
  test.describe(`footer links at ${width}px`, () => {
    test.use({ viewport: { width, height: 844 } });

    // .seg-grow is the colophon label, which is allowed to truncate and is
    // hidden outright on a phone. Everything else in the bar navigates.
    test("every footer link is visible and clickable", async ({ page }) => {
      await page.goto("/index.html");
      const links = page.locator("footer .statusline a:not(.seg-grow)");
      const count = await links.count();
      expect(count, "footer should offer top, home, tags, feed and style guide").toBe(5);

      for (let i = 0; i < count; i++) {
        const link = links.nth(i);
        const label = (await link.textContent()).trim();
        await expect(link, `footer link "${label}" is not visible`).toBeVisible();
        const box = await link.boundingBox();
        expect(box, `footer link "${label}" has no box`).not.toBeNull();
        expect(box.width, `footer link "${label}" is ${box.width}px wide`).toBeGreaterThan(8);
      }
    });
  });
}

test("tag chips render and lead to their tag page", async ({ page }) => {
  const post = aTaggedPost();
  await page.goto(post);

  const chips = page.locator(".tag-list a.tag");
  expect(await chips.count()).toBeGreaterThan(0);

  const label = (await chips.first().textContent())?.trim();
  await chips.first().click();

  await expect(page).toHaveURL(/\/tags\/[a-z0-9-]+\.html$/);
  await expect(page.locator("h1")).toContainText(label ?? "");
  // The tag page lists the post we came from.
  await expect(page.locator(`.post-list a[href="${post}"]`)).toHaveCount(1);
});

test("the tag index links every tag page with a count", async ({ page }) => {
  await page.goto("/tags.html");

  const chips = page.locator(".tag-cloud a.tag");
  expect(await chips.count()).toBe(tagPages().length);

  for (const chip of await chips.all()) {
    await expect(chip.locator(".tag-count")).toHaveText(/^\d+$/);
  }
});

test("footer navigation works from a nested tag page", async ({ page }) => {
  // Tag pages live one directory deep — root-absolute links must still resolve.
  await page.goto(tagPages()[0]);
  await page.getByRole("link", { name: "tags", exact: true }).click();
  await expect(page).toHaveURL(/\/tags\.html$/);
  await expect(page.locator("h1")).toBeVisible();
});
