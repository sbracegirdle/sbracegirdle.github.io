---
name: prose-review
description: Review prose for weak writing and AI-generated tells before it ships. The caller invokes it on changes that add or edit English prose in this repo — blog posts in content/, the README/AGENTS/style docs, or user-visible copy in template.html, static/*.html and Go string literals. Applies style.md, the write-good rules, and an AI-writing-tell catalogue, then reports findings with concrete rewrites. Not for code, identifiers, or the contents of code fences.
---

# Prose review

Catch writing that is weak, wordy, or reads as machine-generated, before it
ships under a real person's name on sbracegirdle.github.io.

`style.md` in the repo root states the voice this site aims for. It is the
target; this skill is how to check a diff against it.

## Scope

Prose is any text a human reads as English on the site or in the repo:

- `content/*.md` — body text, plus the `title` and `description` frontmatter
  (descriptions ship into `<meta>` and the RSS feed, so they get reviewed too)
- `README.md`, `AGENTS.md`, `style.md`
- User-visible copy in `template.html` and `static/*.html` — headings, body
  copy, footer text, `alt` text
- Go string literals that render as page text (e.g. the 404 copy, index
  headings in `main.go`)

Not prose, do not review: code, code fences and their contents, identifiers,
file paths, CSS, test fixtures, generated output under `build/`.

## Process

1. Read `style.md`.
2. Establish the review scope. The caller sets it; there are two shapes.
   - **Named scope** — the caller named files, sections, or passages ("review
     the style guide prose", "review the intro to post X"). What they named is
     the scope, in full. Age is irrelevant: a line that has sat in the file for
     a year is in scope exactly like one written this morning, and "it was
     already there" is not a reason to pass over a rule hit.
   - **Diff scope** — the default when the caller named nothing. `git diff` for
     unstaged work, `git diff --cached` for staged, `git diff main...HEAD` for a
     branch. Only the changed lines are in scope.

   State which shape you used in one line at the top of the report, so the
   caller can tell a clean diff from a clean file.
3. Apply the three rule sets to every passage in scope. Read surrounding
   paragraphs to judge flow, but report only on what's in scope.
   - `references/write-good.md` — the mechanical prose checks
   - `references/ai-tells.md` — the machine-generated-writing catalogue
   - House voice, below

   Under a named scope, walk the rule sets against the text rather than reading
   for what jumps out. The tells that survive longest are the ones that read
   well — negative parallelism and aphorisms in particular, which like the
   bolded lead-in and the end of a paragraph.

   Two passes are worth making on their own, because both need you to look
   somewhere other than the sentence in front of you. Collect every first-person
   claim in scope and ask of each one where Simon said it. Then read each
   section's opening line against the heading above it and the page title, and
   cut what they already said.
4. Report in the output format below. Do not edit the files — propose the
   replacement text and let the caller apply it.

## Dispatch

The review runs as its own pass over finished text, after drafting — never
folded into the drafting step, because it has to look at completed prose.

- **Claude Code** — delegate to the `prose-reviewer` subagent
  (`.claude/agents/prose-reviewer.md`), which runs this skill.
- **Codex** — spawn the `prose-reviewer` agent by name
  (`.codex/agents/prose-reviewer.toml`), which runs this skill.
- **No subagent support** — run this skill directly.

The reviewer is read-only: it proposes replacements, and whoever invoked it
applies them — must-fix always, should-fix and optional by judgement, and a
wrong finding gets argued with rather than applied. The "Do not flag" list
names the false positives the review is meant to suppress, so a bad finding is
a signal this skill needs an edit.

## House voice

From `style.md`: concise, approachable, informal, professional. Plus what the
existing corpus shows:

- **First person, singular.** Simon writes "I" and "you". Not "we" (unless it
  genuinely means Simon and a team), not "one", not "the reader".
- **"I" is a real person's word.** Every sentence in Simon's voice reporting
  what he feels, prefers, ranks, plans or can get to has to trace back to
  something he actually said. Inventing one is a must-fix, not a style note —
  see *invented stance* and *manufactured stakes* in `references/ai-tells.md`.
  When a passage needs a stance the source doesn't supply, the fix is to state
  the fact and stop, not to find a milder feeling.
- **Opinions stated plainly**, hedged honestly. "I think X is a mistake" over
  "some might argue that X could be suboptimal".
- **Concrete over abstract.** Name the tool, show the command, give the number.
- **Contractions are correct here** — "don't", "it's", "you'll". Their absence
  is itself an AI tell.
- **Sentence case headings.** No trailing colons, no title case.
- **Em dashes are part of this voice.** The corpus runs about 5 per 1,000 words,
  and individual posts reach 11. Do not flag em dashes as an AI tell. Only
  mention them above roughly 12 per 1,000 words in a single post, or where two
  land in one sentence.

## Do not flag

`write-good` is a naive linter and the AI-tell lists over-trigger. Every one of
these is a false positive, and reporting them wastes the author's attention:

- Anything inside a code fence, inline code, a URL, a file path, an identifier,
  or frontmatter keys.
- Technical terms that happen to appear on a list — "vital signs" in a
  monitoring post, "landscape" about an actual landscape, "several" where the
  exact count is genuinely unknown and unimportant.
- Passive voice where the actor is unknown or irrelevant.
- Quoted material, book and article titles, error messages — never rewrite a
  quote.
- Prose outside the review scope. Under diff scope, adjacent lines are context,
  not scope. This never applies under a named scope — there, everything the
  caller named is fair game however old it is.
- Style preferences with no rule behind them. If you can't name the rule from
  this skill, drop the finding.

Precision beats recall. Ten real findings the author acts on is a better review
than forty they scroll past.

## Output

Start with the scope you reviewed and a verdict line, then the findings, worst
first:

```
SCOPE: named — static/style-guide.html, whole file    (or: diff — git diff, 3 files)
VERDICT: pass | changes requested  (N must-fix, N should-fix, N optional)
```

Each finding:

```
[must-fix|should-fix|optional] file.md:12 — rule name (rule set)
  now: "the exact text as written"
  fix: "the exact replacement text"
  why: one sentence, only if it isn't obvious from the rule name
```

Severities:

- **must-fix** — model artefacts, assistant voice, lexical illusions, broken
  meaning, an unverified number or claim about Simon's own work, an opinion or
  feeling attributed to him that he never expressed, or anything that reads as
  machine-generated to a casual reader.
- **should-fix** — a clear rule-set hit that makes the writing weaker.
- **optional** — a judgement call worth the author's glance.

If everything in scope is clean, say so in one line and stop. Do not manufacture
findings to look thorough, and do not pad the report with a summary of what the
prose does well.
