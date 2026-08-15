---
name: sports-update
description: Refresh static/sports.html — the hand-written, month-grouped calendar of the six sports Simon follows. Use when fixtures have passed, dates have moved, a venue is confirmed, or a new season needs adding. Covers where each date comes from, which parts of the page have to move together, and which edits are Simon's call rather than yours. Not for design or theme changes to the page.
---

# Keeping the sports page current

`static/sports.html` is a hand-written page. No feed backs it, no generator
touches it, and nothing tells you when it goes stale — it carries the date it
was compiled in the footer and rots quietly from there. This skill is how you
refresh it without losing the facts it already had right.

The page is one thing: the remaining 2026 fixtures across six sports — road
cycling, triathlon, marathon majors, Formula 1, rugby, and Australian and New
Zealand cricket — grouped by month. There is no intro, no summary, no per-sport
section; the lists *are* the page.

## The shape of the page

After the statusline, the h1 and the prompt, each month is an `<h2>` naming the
month, followed by one `<ul class="fixture-list">` of that month's events,
soonest first. An event belongs to the month it starts in:

```html
<li class="fixture hue-love">
  <span class="fixture-when">24 – 26 Jul</span>
  <span class="fixture-what">R11 · <a href="https://www.formula1.com/en/racing/2026">Hungarian Grand Prix</a> <span class="tag tag-love">f1</span></span>
  <p class="fixture-note"><span class="fixture-where">Hungaroring, Hungary</span>Seventy laps on Sunday, then four weeks off.</p>
</li>
```

- `.fixture-when` — the date or date range.
- `.fixture-what` — the linked event name, plus a `.tag` chip naming the sport.
- `.fixture-note` — `.fixture-where` for the venue, then one sentence on the event.

## Facts, and whose they are

**Every date comes from the organiser.** `references/sources.md` lists the
official calendar for each sport. Use those, not a summary of them, and not
memory — a governing body moves a race and the aggregators take weeks. When
something is unsettled, the note says so ("venue still to be announced",
"provisional"), which is better than a date that turns out to be wrong.

**The list is Simon's.** These are the six sports he follows and the events he
cares about. Don't add a sport or a headline event to round the list out, and
don't drop a fixture because it no longer interests him — changing what's on the
page is Simon's call; raise it and let him answer.

**Don't invent what Simon thinks.** The page describes fixtures; it never ranks
them or says which one he's waiting for. The prose review's
`references/ai-tells.md` sets this out. "A world championship three hours from
Perth" is a fact. "The one I've been waiting for" is invention. Write what the
event *is*, once.

## What a refresh is

**A refresh.** Fixtures have happened, a venue was confirmed, a date moved, a
round was cancelled. Most of the work. The structure stays; the facts change.

1. Take passed fixtures off their month's list — the page runs from the compile
   date forward, so anything already run no longer belongs.
2. Put new fixtures in their month and slot by date. An event goes under the
   month it starts in; a row out of order is the one way to make it lie.
3. Walk the parts that move together below.

**A new season.** A year's calendar goes on the page. Bigger, and it has a
decision in it — the h1, the head and the prompt all say "2026" and "what's left
of the season", so the framing has to be reworded. Give Simon the trade-off and
let him choose; whatever the shape, the page stays a list of months.

## The parts that move together

Change a fixture and walk this list — a fixture that only half-lands is worse
than one that never went on.

1. **The compiled date.** The footer says it ISO ("compiled 2026-07-25"). It is
   the day you did the work.
2. **Round numbers.** F1 rows are prefixed `R11 ·`. They come from the F1
   calendar and shift if a round is cancelled or added mid-season.
3. **Hues, tags, and month headings.** Each sport owns one hue — gold cycling,
   foam triathlon, iris marathons, love F1, rose rugby, pine cricket — and every
   row carries its `.hue-*` class **and** a chip naming the sport (`<span
   class="tag tag-gold">cycling</span>` and so on), which takes the hue as its
   outline and text. A month that gains its first event gains an `<h2>`; a month
   whose last event passes loses its `<h2>`. The headings are what makes a mixed
   list readable, so the rail and chip alone would be colour carrying meaning.
4. **Date order.** A row added mid-list takes its place by date; nothing groups
   by sport. Two events on the same day sit together in whatever order reads
   best.
5. **The head.** `<title>`, `og:title`, `description` and `og:description` name
   the year and the six sports. If the sports or the span change, they change.
6. **The homepage blurb.** `content/home.html` links the page and names the
   sports. It's prose, so it goes through the prose review with everything
   else.

## Weight

Measured on 15 August 2026: 22 KB of HTML, plus 38 KB of `theme.css`. The
condensed page is half what it was, so there's headroom — but say so before
writing a change that would balloon it. These numbers date from 15 August 2026;
measure the built page rather than trusting them.

## Before you call it done

The page is prose, it's markup, and editing it changes what a visitor sees:

- **prose review** — every fixture note you touched
- **responsive check** — look at `/sports.html` at 390px and 320px; rows stack
  and nothing should clip
- **design review** — if you touched markup or added a component

Then say what you changed: which fixtures moved and where the new dates came
from, which you removed, what you left alone because it was Simon's call, and
anything the organiser hasn't confirmed yet.
