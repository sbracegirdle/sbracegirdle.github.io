# Rule set 2 — AI tells

Patterns drawn from Wikipedia's
[Signs of AI writing](https://en.wikipedia.org/wiki/Wikipedia:Signs_of_AI_writing)
and the de-slop guidance around it.

These matter more than the write-good rules. A passive sentence reads as
clumsy; these read as *not written by Simon*, which is worse on a personal
blog.

## Significance inflation

Sentences whose job is to tell you something mattered, rather than to say what
it does:

> stands as, serves as, is a testament to, plays a crucial/pivotal/vital role,
> underscores the importance of, reflects a broader, marks a turning point,
> leaves an indelible mark, the ever-evolving landscape of, in today's
> fast-paced world

Cut the sentence. It almost never carries information.

## Slop vocabulary

Words that spike in generated text:

> delve, leverage (as a verb), robust, seamless, crucial, pivotal, vital,
> intricate, interplay, tapestry, testament, meticulous, garner, boasts,
> bolster, underscore, vibrant, showcase, foster, harness, realm, landscape,
> myriad, plethora, navigate (figuratively), embark, unlock, elevate,
> game-changer, cutting-edge, groundbreaking, revolutionary, holistic,
> comprehensive, ensure (where "make sure" fits), align with, resonate with

Not banned outright — "robust" about a retry loop is fine — but each one needs
a reason beyond sounding impressive.

## Negative parallelism

The binary reframe:

> It's not X, it's Y. / Not just X, but Y. / X isn't about A — it's about B.

One in a post is a rhetorical choice. Two is a tic. Rewrite as a plain
statement. Simon's own corpus runs to roughly 20,000 words with none of these,
so on this site the bar is low — see also aphorism density below, which is the
same instinct stretched to a whole sentence.

## Rule of three

Triplets everywhere — "fast, simple, and reliable"; "discover, evaluate, and
adopt". Vary the count. If the third item is filler, drop it.

## Promotional register

Copy that sells where it should tell. A personal page slipping into
landing-page voice, every claim burnished and every noun given a superlative:

> is home ground, some of my proudest X, a greatest-hits list, the unglamorous
> parts matter more, built for the way you actually work, if any of this sounds
> like a problem you have

Simon is allowed to ask people to email him. Flag the salesman's phrasing of
the invitation, never the invitation itself.

Headings do it too. Sentence-case is the house rule, so the tell here is a
heading that performs instead of naming its section — usually a noun with a
participle bolted on after a comma, or a label chosen for punch over accuracy.

> Migrations, landed. Sharper teams. The whole stack. AI engineering, end to end.

Ask what the section is about, and check the heading says that.

## Editorialising asides

Phrases that announce importance instead of demonstrating it:

> It's important to note that, It's worth noting that, Notably, Importantly,
> Crucially, Here's the thing, The key takeaway is, Let's dive in, That said

Delete the wrapper and keep the claim.

## Self-answered rhetorical questions

"The result? A 40% speedup." "Why does this matter?" Replace with the
statement.

## Hollow conclusions

A final paragraph that restates the post and affirms its importance without
adding anything. Also "In conclusion", "In summary", "In this article we". End
on the last real point.

## Aphorism density

The quotable closer. A section ends on a tight, balanced maxim that sounds
earned but carries no new information:

> Modernising a codebase is a skill you practise, not a project you schedule.
> Good tests don't prove you're right, they tell you when you're wrong.
> The best architecture is the one you can delete.

One per post is a good line. Two is a model reaching for the pull-quote. The
giveaway is shape: two balanced halves hinged on a comma or a colon, the second
half turning against the first. And it lands exactly where a paragraph ends.
Watch for it clustering with negative parallelism, which is the same instinct at
sentence scale.

This is the hardest tell to cut, because the line usually reads *well*. Ask
whether it says anything the paragraph above it didn't. Usually the paragraph
was the point and the maxim is a bow on top of it.

## Participial flourishes

A trailing `-ing` or `that…` clause hung off the end of a sentence to make it
land, restating what the sentence already said instead of adding to it:

> the scrapers that keep the data flowing into all of it, instrumentation that
> paid for itself, the safety work that lets it face real users, a shift that
> marked a turning point

Test it by deleting the tail. If the sentence loses a fact, keep it — plenty of
these clauses do real work. If nothing is lost, it was there for cadence.
Suspect it most on the *last* item of a list: a comma-list whose final member
gets a payoff clause is a shape models reach for constantly.

## Symmetrical scaffolding

Every section the same length, every list the same shape,
`**Bold lead-in:** explanation` repeated down a bullet list, a "Challenges"
section followed by a "Future outlook" section. Real posts are lumpy.

## Copula avoidance

"serves as", "functions as", "represents", "boasts", "features" where "is" or
"has" is the honest verb.

## Formatting tells

Title Case In Headings (this site uses sentence case), bold scattered for
emphasis mid-paragraph, emoji as bullets, a horizontal rule before every
heading, tables where a sentence would do, curly quotes mixed into
straight-quote text, skipped heading levels (h2 → h4).

## Vague attribution

"Experts argue", "Industry reports suggest", "Studies show", "Many developers
find" — with no link. Either link the source or write it as your own opinion,
which on this blog it usually is.

## Unverifiable precision

The inverse of vague attribution, and easier to miss because specificity reads
as credibility. A confident number, percentage or scale that no one checked:

> a 30,000-line migration, cut ingestion costs by 40%, paid for itself in
> reduced bills, three times faster

Models fill these in to make a sentence land, and a wrong number is far worse
than a hedge, because it's a claim under Simon's name. On a CV or about page,
treat any figure that isn't sourced, linked or confirmed by Simon as must-fix:
ask him, or cut back to the claim he can stand behind. Two unquantified boasts
about one achievement ("cut costs", "paid for itself") usually mean nobody ever
measured it.

## Model artefacts

Leftover markers such as `contentReference`, `oaicite`, `turn0search0`,
`[cite: 1]`, `:::writing`, or a `utm_source=` parameter on a link. Always a
must-fix.

## Knowledge-cutoff and assistant voice

"As of my last update", "I hope this helps", "Certainly!", "Great question".
Always a must-fix.
