# General UI design standards

Distilled from the two vendor guides on agent-authored interfaces, plus the
WCAG floor both of them assume. These are the general rules the `design-review`
skill applies **on top of** this site's own design system.

Where a rule here disagrees with `static/style-guide.html`, the style guide
wins — see the "Do not flag" list in `SKILL.md`. Most of the aesthetic-direction
advice in both sources is aimed at projects still choosing a look, which this
one isn't. What survives that filter is the mechanical and accessibility
material below, which applies to any interface.

Sources:

- Anthropic, [frontend-design skill](https://github.com/anthropics/claude-code/blob/main/plugins/frontend-design/skills/frontend-design/SKILL.md)
  and [Improving frontend design through Skills](https://claude.com/blog/improving-frontend-design-through-skills)
- OpenAI, [Frontend prompt instructions](https://developers.openai.com/api/docs/guides/frontend-prompt)
  and [Designing delightful frontends](https://developers.openai.com/blog/designing-delightful-frontends-with-gpt-5-4)

## Fit, containment and stability

The rules that catch real bugs most often. All four are from OpenAI's frontend
guidance, and all four are invisible to a page-width overflow check.

- Text must fit within its UI element. If it doesn't fit, break it to a new
  line or size it dynamically — never let it be silently cut off.
- Text must not overlap the content before or after it.
- Define stable dimensions with responsive constraints — `aspect-ratio`, grid
  tracks, `min-`/`max-` bounds — so content arriving or changing doesn't jump
  the layout.
- Hover states, labels and dynamic content must not shift surrounding layout.

To that add the failure mode this repo has already hit twice: a container that
hides its own overflow. `overflow: hidden` on a flex row whose children can't
shrink below their min-content width destroys the tail of the row — and because
nothing escapes the viewport, every page-level overflow assertion still passes.
Check content width against box width directly.

## Typography

- Match display size to container size — smaller headings inside compact
  panels, not one scale reused everywhere.
- Size type relative to its container, not the viewport width.
- Set an intentional scale. Hierarchy comes from deliberate steps, not from
  nudging sizes until it looks fine.
- Letter-spacing should not be negative. (Positive tracking on labels and
  small-caps is fine and is a token here.)

## Layout and composition

- One primary job per section.
- A viewport should read as one composition, not a scatter of panels.
- Prefer full-width bands and unframed sections. Reserve cards for repeated
  items, modals and genuinely framed tools; never nest a card in a card.
- Structural devices must inform, not decorate. Numbered markers only where the
  content is genuinely sequential.
- Favour whitespace and contrast over decorative chrome.

## Colour

- Keep the palette cohesive and give each hue a job. Dominant colours with
  sharp accents beat evenly distributed ones.
- Don't let a decorative hue carry meaning, and don't convey state by colour
  alone.

## Components and interaction

- Pick the control that matches the job: toggles and checkboxes for binary
  settings, sliders and steppers for numeric values, tabs for views, segmented
  controls for modes, menus for option sets.
- Keep border radius at 8px or less unless the design system says otherwise.
  (This one says 0.)
- Link and button text should name the action or destination.

## Motion

- Orchestrated beats scattered. A few intentional moments — an entrance, a
  hover that clarifies an affordance — beat micro-interactions everywhere.
- Over-animation is itself a tell that nobody made a decision.
- Motion should reinforce hierarchy, never add noise, and never move layout.

## Quality floor

Anthropic states these as non-negotiable, and they match the WCAG 2.2 AA
baseline. `accessibility.md` beside this file expands every one of them, adds
semantics, ARIA, accessible names and truncation, and cites the standards:

- Responsive down to mobile.
- Visible keyboard focus on every interactive element.
- `prefers-reduced-motion` respected.
- Built and checked with real content, not lorem ipsum.
- Contrast at least 4.5:1 for body text, 3:1 for large text and UI boundaries.
- Tap targets around 24px minimum, with spacing where they're smaller.

## Writing as design material

- Active voice on controls: "Save changes", not "Submit".
- Consistent terminology across a flow — one name per concept.
- Errors are specific and instructive, never vague.
- Empty states invite the next action rather than setting a mood.
- Sentence case, plain verbs, conversational register.

## Verify by looking

Both vendors land in the same place: the model should render the page in a real
browser, screenshot it at desktop and mobile widths, and inspect the result for
blank sections, overlap, clipped content, broken responsive behaviour, contrast
and tap target size. A design claim not backed by a rendered page is a guess.
