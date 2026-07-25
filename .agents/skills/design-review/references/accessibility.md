# Accessibility standards
<!-- Companion to design-standards.md, which covers the rest of the general
     UI guidance. This file is the floor; that one is the craft. -->

The floor the `design-review` skill enforces. WCAG 2.2 AA is the target, but
most findings on a site like this one come from a shorter list: the wrong
element, a missing name, text you can see but can't read, and something you
can reach with a mouse but not a keyboard.

Sources:

- W3C, [WCAG 2.2](https://www.w3.org/TR/WCAG22/) — the normative standard
- WebAIM, [WCAG 2 checklist](https://webaim.org/standards/wcag/checklist) —
  the same success criteria in plain language, one line each
- W3C WAI, [Developing for web accessibility](https://www.w3.org/WAI/tips/developing/)
- W3C, [Using ARIA](https://www.w3.org/TR/using-aria/) and the
  [ARIA Authoring Practices Guide](https://www.w3.org/WAI/ARIA/apg/)
- MDN, [the `title` attribute](https://developer.mozilla.org/en-US/docs/Web/HTML/Reference/Global_attributes/title)
- Primer, [Truncate accessibility](https://primer.style/product/components/truncate/accessibility/)
- eBay, [the title tooltip anti-pattern](https://opensource.ebay.com/evo-web/accessibility/anti-patterns/title-tooltip)

## 1. Semantics before anything else

The element carries the meaning. Assistive technology reads the markup, not the
appearance, so a `div` styled as a button is invisible as a control no matter
how it looks.

- **Link or button?** A link navigates to a URL. A button performs an action in
  place. Swapping them breaks keyboard expectations (Enter vs Space) and the
  announced role.
- **Landmarks**: `header`, `nav`, `main`, `footer`, `aside`. One `main` per
  page. Where two landmarks share a role, distinguish them with `aria-label`.
- **Headings** describe structure, not size. One `h1`. Levels descend without
  skipping. Never pick a level for its font size.
- **Lists** are `ul`/`ol`/`dl`; tabular data is a `table` with `th` and scope.
- **Reading order must match DOM order** (WAI tip: "reflect the reading order in
  the code order"). CSS `order`, `flex-direction: row-reverse` and absolute
  positioning can put tab order out of step with what the eye sees.

## 2. Prefer native markup to ARIA

The first rule of ARIA is not to use ARIA: prefer the native element. Then:

- **`aria-label` on a generic element does nothing.** A bare `div` or `span`
  has role `generic`, which does not support naming — browsers and screen
  readers drop the label. This repo shipped exactly that:
  `<div class="statusline" aria-label="Site header">`. Either use the element
  whose role it is (`header`, `nav`) or drop the label.
- Never override a native role with a contradictory one.
- `aria-hidden="true"` must not be placed on anything focusable, and belongs on
  decoration only — the blinking prompt cursor is the right use.
- `role="presentation"` strips semantics; use it deliberately or not at all.

## 3. Every meaningful thing needs an accessible name

- Images: `alt` describing the *function* for meaningful images, `alt=""` for
  decoration. An image inside a link supplies that link's name.
- Icon-only controls need a name — visually hidden text or `aria-label`.
- Link text must make sense out of context (SC 2.4.4). "Read more" repeated
  down a page is a finding; "click here" always is.
- Form controls need a real `<label>`. `placeholder` is not a label, and
  neither is `title`.
- `<iframe>` needs `title`.

## 4. Truncation, ellipsis and tooltips

The rule the repo learned the hard way, and the one worth checking on every
statusline, table cell and card:

- **Prefer not to truncate.** Primer's order of preference: wrap the text, give
  the element more room, or expand it on demand. Truncation is the last option.
- **Never truncate a focusable element.** Primer is explicit: don't truncate
  links, buttons or anything else that can take focus. A control whose label is
  cut off can't be identified before it's activated.
- **CSS truncation does not remove text from the accessibility tree.** With
  `text-overflow: ellipsis` the full string is still in the DOM and a screen
  reader still reads it. The person who loses information is the sighted mouse
  or touch user.
- **If it truncates, expose the full text.** `title` is the conventional
  mitigation and what Primer, Carbon and EUI ship, so use it — but know its
  limits, which MDN and eBay both spell out: `title` tooltips never appear for
  keyboard users, never on touch, are inconsistently exposed by assistive
  technology, and are hard to trigger with impaired fine motor control.
- **Never use `title` as the only accessible name.** As a duplicate of visible
  text it is a hint; as the sole label it is a failure.
- A JavaScript-free page has no better hover affordance than `title`. A page
  that can afford a real tooltip should build one that opens on `:focus-visible`
  as well as `:hover` — this theme's `[data-tip]` component does.

## 5. Keyboard

- Every interactive element is reachable and operable by keyboard, in an order
  that matches the visual one.
- **Visible focus** on everything focusable (SC 2.4.7). Never
  `outline: none` without a replacement.
- **Focus not obscured** (SC 2.4.11, new in 2.2): a sticky header, footer or
  overlay must not cover the focused element.
- No keyboard traps. Custom widgets follow the APG pattern for their role.
- Where navigation is long, offer a skip link.

## 6. Colour and contrast

- Text: 4.5:1, or 3:1 at 18.66px bold / 24px regular and above (SC 1.4.3).
- Non-text — icons, focus rings, control boundaries, chart strokes: 3:1
  (SC 1.4.11).
- **Never convey meaning by colour alone** (SC 1.4.1). Pair it with text, shape
  or position.
- Decorative hues are decorative. A colour that fails AA at body size may not
  carry body text, however good it looks.

## 7. Zoom, reflow and text spacing

- **Reflow** (SC 1.4.10): usable at 320px equivalent with no horizontal
  scrolling, which is the same thing as 400% zoom at 1280px.
- **Text spacing** (SC 1.4.12): nothing is lost or clipped when line height goes
  to 1.5×, paragraph spacing to 2×, letter spacing to 0.12em and word spacing to
  0.16em. Fixed-height boxes holding text fail this.
- Size type in relative units so browser text settings still work.

## 8. Motion and timing

- Respect `prefers-reduced-motion: reduce` for every animation and transition.
- Nothing moving, blinking or auto-updating for more than five seconds without a
  control to stop it (SC 2.2.2).
- Nothing flashes more than three times a second (SC 2.3.1).

## 9. Target size

Pointer targets are at least 24×24 CSS pixels, or have equivalent spacing
(SC 2.5.8). Small chrome such as a statusline segment still has to clear this
once padding and line height are counted — measure it rather than assuming.

## 10. Page-level basics

- `<html lang>` set, and `lang` on any passage in another language (SC 3.1.1).
- A descriptive, unique `<title>` (SC 2.4.2).
- Consistent navigation and naming across pages.

## Checking it

Automated rules catch the mechanical half of this at best, so run the probes and
then keep going:

- Assert names, roles, `alt`, `lang` and heading order from the DOM in a
  Playwright probe — cheap, and they never regress silently afterwards.
- Tab through the page and record the focus order and the computed outline.
- Compute contrast from resolved colours rather than reading the token table.
- Compare each element's `scrollWidth` with its `clientWidth` to find text that
  is being cut off rather than wrapped.
- Then look at the screenshot. Overlap, clipping and a lost focus ring are all
  faster to see than to assert.
