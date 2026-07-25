import { test, expect } from "@playwright/test";

// The homepage arcade is the site's only JavaScript, so it is the only part of
// the site that can fail in ways being static rules out. These are the four
// things that would be worth a rollback: it never draws, it draws outside its
// panel, it takes the keyboard without being asked, or it takes the keyboard
// and won't give it back.

const panel = ".arcade";
const frame = ".arcade-frame";

test("the panel draws itself and keeps animating", async ({ page }) => {
  await page.goto("/index.html");

  // The controls are in the markup either way, so their being there proves
  // nothing about the script. A repainted grid is the proof.
  await expect(page.locator(".arcade-play")).toBeVisible();
  await expect(page.locator(".arcade-pause")).toBeVisible();

  const first = await page.locator(frame).innerHTML();
  await page.waitForTimeout(500);
  const second = await page.locator(frame).innerHTML();
  expect(first, "the grid never repainted — the attract loop isn't running").not.toBe(second);
  expect(second.length, "the grid is empty").toBeGreaterThan(0);
});

test("the grid fits its panel at every width", async ({ page }) => {
  for (const width of [1280, 390, 320]) {
    await page.setViewportSize({ width, height: 800 });
    await page.goto("/index.html");
    await page.waitForTimeout(300);

    const fit = await page.locator(frame).evaluate((el) => ({
      overflow: el.scrollWidth - el.clientWidth,
      lines: el.textContent.split("\n").length,
      rows: Number(getComputedStyle(el).getPropertyValue("--arcade-rows")),
      // The grid states its height in CSS so the panel is its final size
      // before a frame is drawn. If the two ever disagree the page shifts
      // under the visitor when the script starts.
      height: el.clientHeight,
      line: parseFloat(getComputedStyle(el).lineHeight),
    }));

    expect(fit.overflow, `the grid overflows its panel at ${width}px`).toBeLessThanOrEqual(1);
    expect(fit.lines, `the grid drew ${fit.lines} rows at ${width}px`).toBe(fit.rows);
    expect(Math.abs(fit.height - fit.rows * fit.line), `panel height at ${width}px`).toBeLessThan(2);
  }
});

// Nothing below the panel may move when the script starts. The grid's height
// is stated in CSS and both buttons are in the markup for this reason: when
// they were revealed instead, the HUD gained a third row at 390px and pushed
// the page down 28px — a layout shift that only escaped being measured as one
// because the script happened to finish before the first paint.
test("the panel is its final size before the script runs", async ({ page }) => {
  await page.setViewportSize({ width: 390, height: 844 });
  const height = () =>
    page.evaluate(() => Math.round(document.querySelector(".arcade").getBoundingClientRect().height));

  await page.route("**/game.js", (route) => route.abort());
  await page.goto("/index.html");
  const cold = await height();

  await page.unroute("**/game.js");
  await page.goto("/index.html");
  await page.waitForTimeout(300);

  expect(cold, "the panel has no height before the script runs").toBeGreaterThan(100);
  expect(await height(), "the panel changed size when the script started").toBe(cold);
});

test("the keyboard is only taken once the visitor asks", async ({ page }) => {
  await page.goto("/index.html");

  // Attract mode must not swallow the arrow keys: someone reading the page has
  // to be able to scroll past the panel.
  await page.locator("body").click({ position: { x: 5, y: 5 } });
  await page.keyboard.press("ArrowDown");
  await page.waitForTimeout(100);
  expect(await page.evaluate(() => window.scrollY), "attract mode ate the arrow keys").toBeGreaterThan(0);

  await page.evaluate(() => window.scrollTo(0, 0));
  await page.locator(".arcade-screen").click();
  await expect(page.locator(panel)).toHaveClass(/playing/);
  await expect(page.locator(".arcade-play")).toHaveText("stop");

  // Measured from wherever the click left the page: bringing the panel into
  // view is Playwright's business, and it moves by a pixel doing it.
  const at = await page.evaluate(() => window.scrollY);
  await page.keyboard.press("ArrowDown");
  await page.waitForTimeout(100);
  expect(await page.evaluate(() => window.scrollY), "the page scrolled while playing").toBe(at);
});

test("escape and a click elsewhere both hand the ship back", async ({ page }) => {
  await page.goto("/index.html");

  await page.locator(".arcade-screen").click();
  await expect(page.locator(panel)).toHaveClass(/playing/);
  await page.keyboard.press("Escape");
  await expect(page.locator(panel)).not.toHaveClass(/playing/);

  await page.locator(".arcade-screen").click();
  await expect(page.locator(panel)).toHaveClass(/playing/);
  await page.locator("main p").first().click();
  await expect(page.locator(panel)).not.toHaveClass(/playing/);
  await expect(page.locator(".arcade-play")).toHaveText("play");
});

// SC 2.2.2. The panel starts on its own and runs for as long as the page is
// open, next to text someone is trying to read, so it owes them a way to halt
// it. "stop" isn't that way — it hands the ship back to the autopilot, which
// carries on flying.
test("pause halts the panel outright, and playing resumes it", async ({ page }) => {
  await page.goto("/index.html");
  const frameHTML = () => page.locator(frame).innerHTML();

  await page.locator(".arcade-pause").click();
  await expect(page.locator(".arcade-pause")).toHaveText("resume");
  const halted = await frameHTML();
  await page.waitForTimeout(500);
  expect(await frameHTML(), "the panel kept animating after pause").toBe(halted);

  // Taking the controls is also a way of asking for it to move again.
  await page.locator(".arcade-screen").click();
  await expect(page.locator(".arcade-pause")).toHaveText("pause");
  await page.waitForTimeout(400);
  expect(await frameHTML(), "playing did not resume the panel").not.toBe(halted);
});

// The panel holds the arrow keys, so it has to let go the moment it stops
// being something the visitor can see.
test("scrolling the panel out of view hands the keyboard back", async ({ page }) => {
  await page.setViewportSize({ width: 1280, height: 700 });
  await page.goto("/index.html");

  await page.locator(".arcade-screen").click();
  await expect(page.locator(panel)).toHaveClass(/playing/);

  await page.mouse.wheel(0, 1400);
  await expect(page.locator(panel)).not.toHaveClass(/playing/);

  const at = await page.evaluate(() => window.scrollY);
  await page.keyboard.press("ArrowDown");
  await page.waitForTimeout(100);
  expect(
    await page.evaluate(() => window.scrollY),
    "an off-screen panel is still eating the arrow keys",
  ).toBeGreaterThan(at);
});

// pointerdown fires at the start of a scroll flick. Taking the controls there
// put `touch-action: none` under the thumb, and a band of the home page
// stopped scrolling for good.
test("a touch flick scrolls the page instead of taking the controls", async ({ browser }) => {
  const context = await browser.newContext({
    viewport: { width: 390, height: 844 },
    hasTouch: true,
    isMobile: true,
  });
  const page = await context.newPage();
  await page.goto("/index.html");

  const box = await page.locator(".arcade-screen").boundingBox();
  const x = Math.round(box.x + box.width / 2);
  await page.touchscreen.tap(x, Math.round(box.y + box.height / 2));
  await expect(page.locator(panel)).toHaveClass(/playing/);
  await page.keyboard.press("Escape");

  // A flick that starts on the panel: down, moved, up.
  await page.evaluate(
    ([px, py]) => {
      const el = document.elementFromPoint(px, py);
      const opts = { pointerType: "touch", bubbles: true, clientX: px, clientY: py, pointerId: 1 };
      el.dispatchEvent(new PointerEvent("pointerdown", opts));
      el.dispatchEvent(new PointerEvent("pointerup", { ...opts, clientY: py - 120 }));
    },
    [x, Math.round(box.y + box.height / 2)],
  );
  await expect(page.locator(panel)).not.toHaveClass(/playing/);
  expect(
    await page.evaluate(() => getComputedStyle(document.querySelector(".arcade-screen")).touchAction),
    "the panel is blocking touch scrolling when nobody is playing",
  ).not.toBe("none");

  await context.close();
});

// A panel that animates by itself is exactly what prefers-reduced-motion is
// about, so it doesn't — until the visitor presses play, which is them asking
// for it. Emulated per page rather than through `test.use({ reducedMotion })`:
// that form is silently ignored inside a describe block here, and the test
// passed for the wrong reason.
test("reduced motion means nothing moves until the visitor presses play", async ({ page }) => {
  await page.emulateMedia({ reducedMotion: "reduce" });
  await page.goto("/index.html");
  await page.waitForTimeout(400);

  const still = await page.locator(frame).innerHTML();
  await page.waitForTimeout(400);
  expect(await page.locator(frame).innerHTML(), "the attract loop ran anyway").toBe(still);

  // Reduced motion is the panel starting out halted rather than a second kind
  // of stopped, so the button says what state it is in. Offering to "pause"
  // something that isn't moving is a control that does nothing.
  await expect(page.locator(".arcade-pause")).toHaveText("resume");

  await page.locator(".arcade-play").click();
  await page.waitForTimeout(400);
  expect(
    await page.locator(frame).innerHTML(),
    "the game stayed frozen after being asked to play",
  ).not.toBe(still);
  await expect(page.locator(".arcade-pause")).toHaveText("pause");
});
