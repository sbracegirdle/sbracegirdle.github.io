---
title: Teaching back
description: How I stay intellectually engaged with AI-generated work by explaining it back to the agent as I review it.
---

AI agents make it easy to become intellectually lazy. I can skim an output, recognise the shape of something plausible and move on without absorbing or understanding it deeply.

Sometimes I use a slower process:

1. I prompt the agent.
2. The agent produces the code, UI or other artifact.
3. I inspect the result, writing my understanding back to the agent and adding questions or feedback as I go.

I think of that review process as teaching back.

## Explaining it in my own words

[Healthcare provides a useful example of teach-back](https://www.ahrq.gov/patient-safety/reports/engage/teachback.html). A patient explains instructions in their own words, which helps the clinician find gaps in the explanation or the patient's understanding.

My process is similar. The agent produces the work, and I explain my understanding back to it. For a code change, I might write:

> My understanding is that the handler validates the request, writes the new record, then invalidates the account cache. The transaction keeps the write and audit entry together. I don't see what prevents two requests from creating the same record.

Writing that forces me to follow the implementation, explain why it exists and compare it with the original intent. It also gives the agent something precise to respond to: correct my reading, explain a missing constraint or fix the race.

The same approach works for a UI:

> The page puts the current status first, with the history as supporting detail. The retry action is available only for failed jobs. I expected the error message to sit beside that action rather than in the activity list.

Or for a document:

> The proposal assumes the migration can happen before clients adopt the new API. If both versions need to run together, the rollout section is missing that period of overlap.

In each case, I write down my understanding as I review the result, including anything I can't explain.

## Why it helps

Paul Graham's [Putting Ideas into Words](https://paulgraham.com/words.html) describes writing as a test that exposes incomplete thoughts. A peer-reviewed [meta-analysis of 28 learning-by-teaching studies](https://doi.org/10.1111/jpr.12221) found that preparing to teach and then teaching improved both surface and deep learning compared with studying alone.

The agent makes the review interactive. As I write back my understanding, I can ask:

- What have I misunderstood?
- Which assumption is doing the most work here?
- Why does this abstraction exist?
- What breaks if that condition changes?

Those questions borrow from the [Socratic habit of probing an explanation](https://serc.carleton.edu/teachearth/teaching_methods/socratic/fourth.html). They turn review into a conversation about the work instead of a verdict on it.

This process is slower because I spend more time with the result. By the time I approve it, I can explain what was built and why. That makes me feel closer to the work.
