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
  for (const urlPath of ["/style-guide.html", "/rust-quick-reference.html", "/sports.html"]) {
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

    // The colophon lives on the status row and the links on the nav row below
    // it, so the two can no longer compete for the same space at any width.
    test("every footer link is visible and clickable", async ({ page }) => {
      await page.goto("/index.html");
      const links = page.locator("footer .statusline-nav a");
      const count = await links.count();
      expect(count, "footer should offer top, home, feed and style guide").toBe(4);

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

// The header carries the way in — the key pages — on a nav row of its own.
for (const width of [1280, 390, 320]) {
  test.describe(`header nav at ${width}px`, () => {
    test.use({ viewport: { width, height: 844 } });

    test("every header nav link is visible and clickable", async ({ page }) => {
      await page.goto("/index.html");
      const links = page.locator("header .statusline-nav a");
      const labels = (await links.allTextContents()).map((t) => t.trim());
      expect(labels, "header nav does not offer the pages it should").toEqual([
        "~/blog",
        "posts",
        "tags",
        "sports",
        "about",
      ]);

      for (const link of await links.all()) {
        const label = (await link.textContent()).trim();
        await expect(link, `header link "${label}" is not visible`).toBeVisible();
        const box = await link.boundingBox();
        expect(box.width, `header link "${label}" is ${box.width}px wide`).toBeGreaterThan(8);
      }
    });
  });
}

// The two nav landmarks are distinct destinations, so each needs its own name;
// "navigation" twice tells a screen reader user nothing about which is which.
test("the two nav landmarks are named and distinct", async ({ page }) => {
  await page.goto("/index.html");
  const names = await page
    .locator("nav")
    .evaluateAll((els) => els.map((el) => el.getAttribute("aria-label")));
  expect(names.length, "expected a header nav and a footer nav").toBe(2);
  expect(names.filter(Boolean).length, "every nav needs an accessible name").toBe(2);
  expect(new Set(names).size, `nav labels are not distinct: ${names.join(", ")}`).toBe(2);
});

// Giving the links their own row is what buys this: the colophon used to be a
// truncated *link* at every desktop width, 308px of its 449px shown at 1280.
for (const width of [1280, 768, 390, 320]) {
  test.describe(`footer colophon at ${width}px`, () => {
    test.use({ viewport: { width, height: 844 } });

    test("the colophon is shown whole, not truncated", async ({ page }) => {
      await page.goto("/index.html");
      const colophon = page.locator("footer .seg-note");
      await expect(colophon).toBeVisible();

      const cut = await colophon.evaluate((el) => ({
        x: el.scrollWidth - el.clientWidth,
        y: el.scrollHeight - el.clientHeight,
        text: el.textContent.trim(),
      }));
      expect(cut.x, `colophon clipped horizontally: "${cut.text}"`).toBeLessThanOrEqual(1);
      expect(cut.y, `colophon clipped vertically: "${cut.text}"`).toBeLessThanOrEqual(1);
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

// The statusline check above knows which container to look inside. This one
// doesn't need to: it asks the general question — is any element hiding its own
// overflow? — which is the shape of the footer bug rather than the instance of
// it. The theme has exactly one deliberate `overflow: hidden` today
// (.seg-shrink, guarded by the title rule in a11y.spec.js), so the next one
// someone adds is caught here instead of shipping.
for (const width of [1280, 700, 390, 320]) {
  test.describe(`containment at ${width}px`, () => {
    test.use({ viewport: { width, height: 844 } });

    for (const urlPath of samplePages()) {
      test(`no container hides its own content: ${urlPath}`, async ({ page }) => {
        await page.goto(urlPath);

        const clipped = await page.evaluate(() =>
          [...document.querySelectorAll("body *")]
            .filter((el) => {
              const s = getComputedStyle(el);
              if (s.display === "none" || s.visibility === "hidden") return false;
              return /hidden|clip/.test(`${s.overflowX} ${s.overflowY}`);
            })
            // .seg-shrink truncates on purpose; it carries a matching title and
            // is never focusable, both asserted in a11y.spec.js.
            .filter((el) => !el.classList.contains("seg-shrink"))
            .filter(
              (el) =>
                el.scrollWidth - el.clientWidth > 1 || el.scrollHeight - el.clientHeight > 1,
            )
            .map(
              (el) =>
                `${el.tagName.toLowerCase()}.${el.className} clips "${(el.textContent || "")
                  .trim()
                  .slice(0, 60)}"`,
            ),
        );

        expect(clipped, "an element hides content it should show or scroll").toEqual([]);
      });
    }
  });
}

// The season table carries every count twice: as the printed number a reader
// sees, and as the bar-N class that draws the column behind it. Thirty-six
// hand-written cells, and a chart that disagrees with its own numbers looks
// completely normal. The bar is aria-hidden, so no accessibility check reaches
// it either.
test("season bars match their printed counts", async ({ page }) => {
  await page.goto("/sports.html");

  const cells = await page.$$eval(".season tbody td", (tds) =>
    tds.map((td) => {
      const n = td.querySelector(".n");
      const barClass = [...(td.querySelector(".bar")?.classList ?? [])].find((c) =>
        c.startsWith("bar-"),
      );
      return {
        printed: n?.textContent.trim() ?? "",
        // A zero month prints "0" and marks it .none — there is no column to
        // draw, so the absent bar is correct rather than a mismatch.
        zero: Boolean(n?.classList.contains("none")),
        bar: barClass ? barClass.slice("bar-".length) : "",
      };
    }),
  );

  expect(cells.length, "no season cells found — has the markup changed?").toBeGreaterThan(0);
  expect(
    cells.some((c) => c.bar !== ""),
    "no bars found at all — has the chart markup changed?",
  ).toBe(true);

  const wrong = cells.flatMap((c) => {
    if (c.zero) {
      // Marked empty: must print 0 and draw nothing.
      if (c.printed !== "0" || c.bar !== "") {
        return [`cell marked empty prints "${c.printed}" and draws bar ${c.bar || "none"}`];
      }
      return [];
    }
    if (c.bar !== c.printed) {
      return [`printed ${c.printed || '""'}, bar ${c.bar || "none"}`];
    }
    return [];
  });

  expect(wrong, "the season chart disagrees with its own numbers").toEqual([]);
});

// The scroll hint is shown by a hand-measured media query, so it drifts the
// moment a column is added or a cell gets longer. Either half of the mismatch
// is a real failure: a table that scrolls with no hint hides half the year, and
// a hint with nothing to scroll sends the reader looking for content that is
// already on screen.
for (const width of [1280, 700, 565, 390, 320]) {
  test.describe(`season table at ${width}px`, () => {
    test.use({ viewport: { width, height: 844 } });

    test("the scroll hint appears exactly when the table scrolls", async ({ page }) => {
      await page.goto("/sports.html");

      const state = await page.evaluate(() => {
        const box = document.querySelector(".scroll-x");
        const hint = document.querySelector(".scroll-hint");
        return {
          // Any overflow at all means the reader has content off screen. The
          // breakpoint is calibrated to the pixel — 566px fits exactly, 565px
          // overflows by one — so a tolerance here would blind the test to the
          // boundary it exists to watch.
          scrolls: box ? box.scrollWidth - box.clientWidth > 0 : null,
          hinted: hint ? getComputedStyle(hint).display !== "none" : null,
        };
      });

      expect(state.scrolls, "no .scroll-x region found on the page").not.toBeNull();
      expect(state.hinted, "no .scroll-hint found on the page").not.toBeNull();
      expect(
        state.hinted,
        state.scrolls
          ? "the table scrolls but no hint says so"
          : "a scroll hint is shown but the table fits",
      ).toBe(state.scrolls);
    });
  });
}
