---
name: sports-update
description: Refresh static/sports.html — the hand-written calendar of six sports and the athletes and teams Simon follows. Use when fixtures have passed, dates have moved, a venue is confirmed, or a new season needs adding. Covers where each date comes from, which parts of the page are derived from other parts and have to move together, and which edits are Simon's call rather than yours. Not for design or theme changes to the page.
---

# Keeping the sports page current

`static/sports.html` is a hand-written page. No feed backs it, no generator
touches it, and nothing tells you when it goes stale — it carries the date it
was compiled and rots quietly from there. This skill is how you refresh it
without losing the facts it already had right.

The page covers six sports: road cycling, triathlon, marathon majors, Formula 1,
rugby, and Australian and New Zealand cricket. It lists fixtures, and under most
of them the athletes and teams Simon follows.

## What kind of job this is

**A refresh.** Fixtures have happened, a venue was confirmed, a date moved, a
round was cancelled. Most of the work. The structure stays; the facts change.

**A new season.** A year's calendar is published and goes on the page. Bigger,
and it has a decision in it — see *Adding a season* below.

Either way, read the whole page first. Its sections cross-reference each other,
and the second half of this skill lists the places where that bites.

## Facts, and whose they are

**Every date comes from the organiser.** `references/sources.md` lists the
official calendar for each sport. Use those, not a summary of them, and not
memory — a governing body moves a race and the aggregators take weeks. When
something is unsettled, the page says so in the note ("venue still to be
announced", "provisional"), which is better than a date that turns out to be
wrong.

**The people are Simon's.** The `.who` chips are who he follows. Don't add a
name because they're winning, don't drop one because they retired or had a bad
year, and don't round the list out to make a section look complete. Correcting a
misspelling or a team change is a fix; changing who is on the list is Simon's
call — raise it and let him answer. Same for adding or dropping a sport.

**Don't invent what Simon thinks.** The page describes fixtures; it never ranks
them or says which one he's waiting for. The prose review's
`references/ai-tells.md` sets this out, and this page is where it goes wrong most
easily: a fixture note is one sentence long and an opinion fills the space so
neatly. "A world championship three hours from Perth" is a fact. "The one I've
been waiting for" is invention. Write what the event *is*, once.

## The parts that move together

Change a fixture and walk this list — a fixture that only half-lands is worse
than one that never went on.

1. **The compiled date, twice.** The `Snapshot` callout says it long-form
   ("Compiled 25 July 2026"), the footer says it ISO ("compiled 2026-07-25").
   Both are the day you did the work, and they must agree.
2. **The season table.** One count per fixture, in the month it starts. A
   fixture listed under two sports counts once — the trans-Tasman Test series
   appears in both cricket sections and is one entry in the December cell.
   `.bar-N` exists for 1 to 5 only (`theme.css`); a sixth event in a month needs
   a new rule and a new wash, which is a theme change and a design review.
   Zero is `<span class="n none">0</span>`, with no bar.
3. **The glance paragraph** above the table names the heaviest month and what's
   in it. Re-derive it from the table you just edited rather than trusting the
   sentence that's there.
4. **What's next.** The soonest fixtures across all six sports, in date order,
   starting from the compile date. Anything already run comes off. It mixes
   sports, so every row carries a `.tag` chip naming the sport as well as its
   hue — the rail alone would be colour carrying meaning.
5. **The section lists.** Each sport's `.fixture-list` keeps past fixtures until
   they fall out of the page's window, because the table counts them and the
   season reads as a whole. Removing one means fixing its cell in the table.
6. **Anchors.** The table's row headers and the What's next rows link to
   `#cycling`, `#triathlon`, `#marathon`, `#f1`, `#rugby`, `#cricket`. Rename a
   section and both sets of links break; `links.spec.js` catches it.
7. **Hues.** Each sport owns one — gold cycling, foam triathlon, iris marathons,
   love F1, rose rugby, pine cricket — and the table row, the section's list and
   the What's next rows all use it. `.who` chips are the exception: there the
   hue is the athlete's nationality, and it repeats the `.nat` code beside it.
8. **The head.** `<title>`, `og:title`, `description` and `og:description` name
   the year and the six sports. If the sports or the span change, they change.
9. **The homepage blurb**, in `main.go` near the reading shelves, names the year
   and three example fixtures. It's a Go string literal and it's prose, so it
   goes through the prose review with everything else.
10. **The scroll hint breakpoint.** The page's one inline rule reveals the hint
    at 565px, measured for a seven-column table. Add a column and the number is
    wrong — re-measure it at the width where the last column clips, and say in
    the comment what you measured.

## Adding a season

Simon may ask for next year's events on the page before this year's are done.
The page is built around one season and a table of six months, so it needs a
decision from him. Give him the trade-off and let him choose.

The cheap version, and the default if he has no preference: keep one page,
slide the table's six-month window forward from the compile month, and label
the columns that cross the boundary with the year (`Dec`, `Jan '27`). The h1,
the title and the descriptions then name a span rather than a year, and the
"what's left of the season" framing has to go — reworded, and through the prose
review.

The expensive version is a page per season, which means a naming scheme, a link
between them, an archive page nobody asked for, and two pages going stale
instead of one. Don't reach for it unprompted.

Whatever the shape, keep the six-month window: the table's breakpoint (point 10)
and the `.bar-5` ceiling (point 2) both assume it.

## Weight

Measured on 25 July 2026: 42.4 KB of HTML, plus 42.5 KB of `theme.css`, for
85.0 KB of the 160 KB `maxPageBytes` budget. Only `/style-guide.html` and the
Rust quick reference are heavier. That leaves room for about one more season of
fixtures and no more, so say so before writing a change that would double the
page. `npx playwright test specs/footprint.spec.js` prints the current number —
measure rather than trusting the one above.

Watch `theme.css` harder than the page. It sits on the critical path of every
page on the site and it is within a few kilobytes of its own 45 KB budget, so a
new rule here — a `.bar-6` for a busy month, a component for a new kind of
fixture — costs more than it looks like it costs. Both budgets are Simon's to
raise, never yours.

## Before you call it done

The page is prose, it's markup, and editing it changes what a visitor sees, so
put the result through the review skills as needed:

- **prose review** — every fixture note and section intro you touched
- **browser test** — `npm test`, then look at `/sports.html` at 390px and 320px
- **design review** — if you touched markup or added a component
- **perf audit** — if the page grew materially, or gained a request

Then say what you changed: which fixtures moved and where the new dates came
from, which you removed, what you left alone because it was Simon's call, and
anything the organiser hasn't confirmed yet.
