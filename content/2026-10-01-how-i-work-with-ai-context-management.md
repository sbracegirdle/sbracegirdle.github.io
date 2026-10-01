---
title: "How I work with AI, part 1: Managing context"
description: Choosing what an AI agent needs for a task, where to keep instructions, and when to compact or start a fresh conversation.
tags: [ai, workflow]
---

This is the first post in a series about how I work with AI agents in software engineering teams. It starts with context management, including where to keep instructions and how to decide what an agent needs to read for a task. It also covers choosing a context window and preserving progress when a conversation needs to be compacted or restarted.

To see why context management matters, consider a hypothetical incident investigation. An agent begins with an issue and a few log excerpts. As it follows the evidence, the conversation grows to include deployment details and possible explanations for the failure. This material sits alongside your instructions in the model's context window, which holds what it can use for its current response. The window has a token limit and also needs room for the response and, depending on the model, reasoning tokens.

What belongs in that window changes during the investigation. Early logs may help distinguish between possible causes. Once the agent has reproduced the failure, the fix may need only that reproduction and the accepted design. Carrying every discarded explanation into the implementation adds context that no longer helps, while losing an important constraint can lead to the wrong fix.

Managing context means making those choices as the work progresses. The starting point is the instructions that accompany each task, before the agent begins gathering evidence.

## Keep base instructions short and specific

Base instruction files such as `AGENTS.md` or `CLAUDE.md` accompany tasks within their scope. A long release procedure in repository instructions can therefore take up context even when the agent is fixing a typo. Keep these files focused on rules that apply across tasks. Global instructions can hold preferences you want across projects, while repository instructions hold the project's conventions and recurring pitfalls.

Within those files, be specific about the behaviour you expect. A general request to follow good engineering practices leaves the agent to choose what that means for your project. State the action and when it's required. For example, if API changes need integration testing, write “After changing a public endpoint, run its integration tests. Report the command and result, including any checks you couldn't run.” This gives the agent a completion requirement and gives you evidence to review. Anthropic's [instruction-writing guidance](https://code.claude.com/docs/en/memory#write-effective-instructions) recommends concrete, verifiable rules.

The same principle applies to project constraints. Record the project knowledge an agent needs to act correctly, especially conventions it would otherwise have to rediscover. Explain the reason for each rule so it can understand the consequence of ignoring it. For example, some projects generate files during a build. An edit to that output will be lost when the next build recreates it, so the instruction should say to change the source instead. Include a path when it helps identify which files the rule applies to.

Requiring the agent to read every document before every task fills context with material it may never use. A typo fix doesn't need the API compatibility policy. “Read `docs/api-compatibility.md` before changing a public endpoint” makes that policy available for the changes it governs. OpenAI's [guidance on updating instructions](https://developers.openai.com/blog/rethinking-skills-and-prompts-for-gpt-6-astra) recommends reading documents when the task needs them.

Extra wording can obscure an otherwise clear rule. Once the action and reason are present, trim the sentence. “If you have changed any files in the build directory, make sure not to include them in your commit, since they will be generated again when the site builds” can become “Don't commit `build/`. It contains generated files.” The shorter version keeps both the rule and its reason.

Completion requirements also need a boundary. Suppose the agent is fixing a checkout bug and finds a reporting-service test that also fails on the unchanged main branch. “Fix every failing test” could send it into a second investigation. Say how to handle failures outside the task, such as “Fix test failures caused by the checkout change. Report failures that also occur on main separately.” That keeps the failure visible without expanding the repair. For a review, be equally clear about whether you want findings or edits to the code.

Duplicated instructions take up context and can contradict each other when one copy changes. Keep each rule in one place and point to it where needed. If writing guidance lives in `style.md`, repository instructions can say “Read `style.md` before writing or editing a post.” Copying the full guide into `AGENTS.md` would load it for code changes too, and leave two copies to maintain.

Instructions can become inaccurate as the project changes. A renamed test command or a moved reference document can send the agent down a dead end. Include instruction updates in the same pull request as changes to the commands or conventions they describe. When an agent repeatedly misunderstands a rule, revise that rule at its source and check it on a similar task before merging the update.

For an active repository, a monthly review is a useful starting cadence. It can be part of an existing team retrospective. Use recent agent tasks to identify repeated corrections and instructions that loaded without being relevant. Ask the agent to propose edits, then have a maintainer review them against the current project. Remove duplication and check that referenced documents and commands still exist. Adjust the frequency if changes or recurring mistakes warrant a review sooner.

Model and tool updates are another reason to review the guidance. A workaround added after one bad response may no longer help, or may interfere with behaviour the new model handles well. Try representative tasks after an update and check which instructions still serve a purpose. Remove obsolete rules and repair broken references, while preserving project constraints that still apply.

## Choose between instructions, skills and docs

Keeping base instructions short leaves a question about where the detail belongs. A release checklist and an incident response procedure need different context, so neither needs to accompany every task. Skills provide a way to load each procedure when the agent is doing that work.

Preparing a release might involve retrieving milestone issues, checking linked pull requests, inspecting CI and drafting release notes. An agent fixing a typo doesn't need that procedure. Put the repeatable workflow in a release-preparation skill so the agent can load it for release work. Keep the current release number and milestone in the request. They change between runs and don't belong in permanent instructions.

Use a skill as a playbook for a repeatable procedure with a defined result. A database migration review skill might check compatibility with existing clients, deployment ordering and rollback behaviour, then return findings against the proposed migration. Keep it focused on that review rather than turning it into a general database skill.

Use plain Markdown reference docs when the agent needs facts, constraints or rationale. An architecture decision record explaining why the checkout service owns payment state belongs in a document. So do the service's SLOs and dependency map. These documents can also preserve the history of decisions, including alternatives considered and why they were rejected. An incident-investigation skill can point to those references without copying them into its instructions.

Those references can also help the agent gather more context as the investigation develops. If checkout logs point to duplicate payment attempts, the agent can read the decision record about payment ownership and retry behaviour. It doesn't need every service's architecture history before examining the first error.

The distinction is whether you need the agent to consult information or perform a workflow. A one-off question about payment ownership can point straight to the document. A recurring investigation that checks ownership, traces a request and gathers evidence benefits from a skill.

Give skills precise triggers. “Use when reviewing a database migration” is more useful than “Use for database work”, which could invoke a migration procedure while the agent is explaining a query. [OpenAI's skill guidance](https://developers.openai.com/blog/rethinking-skills-and-prompts-for-gpt-6-astra#better-skills) uses this distinction to show how broad triggers load irrelevant instructions.

Moving text into a separate file only helps context management if the agent reads it when needed. Don't move a long procedure out of base instructions and then require it at the start of every task. Claude Code's [startup imports](https://code.claude.com/docs/en/memory#write-effective-instructions), for example, split the files but still load their contents.

## Give the agent clear scope boundaries

Even with well-organised instructions and references, the agent needs to know which question it's answering. A broad request can lead it through material that has little bearing on the result you need. Tell it what you want to find out and where to start looking. Explain what to leave out and what you expect back, so it can choose what to read before search results and logs fill the conversation.

For a release review, “find blockers” needs a definition of what would prevent the release. An open issue may have been deferred, and a failing optional check may not be a release requirement. Point the agent to the team's release checklist so it can judge the evidence against the agreed criteria:

> Review the payments milestone against `docs/release-checklist.md`. Check issues marked `release-blocker`, pull requests required for this release that haven't merged, and failing or missing required CI checks on the release commit. Apply the checklist's criteria and any recorded waivers before calling something a blocker. Follow linked dependencies outside the milestone only if they prevent a release requirement from being met. Return confirmed blockers with source links and the unmet requirement. List anything you couldn't verify separately. Don't change issues or code.

For an incident investigation:

> Investigate the checkout errors between 14:00 and 14:20 UTC. Start with the incident issue, the checkout logs for that interval and the deployment immediately before it. Identify the likely cause and supporting evidence. Don't change code or configuration. Follow dependencies beyond checkout if the evidence points there, and explain why.

The second example leaves room to follow evidence while setting an initial service and time range. “Investigate checkout” could send the agent through months of issues and logs without answering the incident question.

Search results can fill the window even when only a few entries matter. Ask the agent to limit what it reads to the question it's investigating. For the release review, it can filter the issue tracker by milestone and blocker label before opening discussions. It doesn't need to download the whole backlog to find the few issues that could prevent this release.

A build log can contain thousands of lines of successful steps around a short failure. Ask the agent to save the full log to a file and read the failure output with the surrounding lines. Keep the file path so it can read more if the cause isn't clear. For web research, ask it to read the source passage supporting a finding and keep the link for follow-up. Summarising a huge log after reading it doesn't recover the context already used by that log.

This follows Anthropic's [guidance on retrieving context as needed](https://www.anthropic.com/engineering/effective-context-engineering-for-ai-agents). Scope the investigation without prescribing every search or command.

## When to use 200k or 1M

The size of the window matters when the task needs more material available together than the smaller window can hold. A focused implementation, review or workflow run should usually have a bounded set of relevant inputs, with room left for further tool results and output. Start with 200k unless the task has a specific need for 1M.

Use 1M when the work needs a large amount of original material available together. For example:

- Reviewing a migration across several services, where compatibility depends on details spread across their implementations and contracts.
- Comparing incident reports, issue discussions and deployment histories to find a recurring failure pattern.
- Reconciling a collection of specifications where exact wording and exceptions matter, and summaries would remove evidence you need to compare.
- Continuing an investigation whose accumulated evidence remains relevant and would be costly to reconstruct after compaction.

Before expanding, check whether the material needs to be present together. A release workflow that processes independent issues may only need a running list of findings. A compatibility review may need both sides of every interface available for comparison. The second case gives you a stronger reason to use 1M.

Validate a larger window on representative tasks by checking whether the agent uses the relevant evidence and preserves constraints. The original [Lost in the Middle study](https://arxiv.org/abs/2307.03172) tested earlier models and doesn't establish how today's frontier models use long context. A [2026 positional evaluation](https://arxiv.org/abs/2605.23170) found that newer releases reduced some position effects, while other effects persisted depending on the surrounding material. It didn't test the latest GPT or Claude models, so it can't establish a cutoff for them either.

## When to compact or start fresh

As a conversation grows, its history may contain both evidence you still need and exploration you've finished with. Compaction reduces that history to a shorter representation so the same conversation can continue. Use it when the objective remains the same and the agent still needs the accumulated state. [OpenAI's Codex guidance](https://developers.openai.com/blog/mastering-codex-remote-for-engineering#7-manage-context-before-it-becomes-a-problem) recommends compaction for this situation.

A fresh conversation is useful when the next task needs different context or the history keeps steering the agent towards discarded approaches. [Claude Code's session guidance](https://code.claude.com/docs/en/best-practices#manage-context-aggressively) recommends clearing between unrelated tasks. At a task or phase boundary, if you can't explain why the old exploration is needed for the next step, default to a fresh conversation. For unfinished work, save a checked handoff of the useful state before clearing. Keep links to the original evidence so the new session can retrieve details if needed.

For example:

- During an incident, keep the conversation while the agent is comparing live hypotheses. Once you reproduce the cause and decide on a fix, save the reproduction and decision. Compact if the exploratory logs are crowding out the implementation work.
- After preparing a release report, start a fresh conversation for an unrelated API review. The report's issue discussions and CI output don't help the review.
- If the agent keeps proposing a migration approach you've rejected, save the accepted design and the reason for rejecting the alternative. Start fresh with that request rather than adding another correction to the same history.

Check context usage at milestones and before a large batch of reading. If the work is progressing and the history remains useful, keep going and let automatic compaction handle capacity where the tool supports it. Repeated confusion about the current goal or rejected decisions is a reason to intervene earlier.

Caching is a reason to keep a productive conversation intact while its history remains useful. When a later request reuses an unchanged beginning of the conversation, a cache hit can make that input cheaper to process. As of October 2026, [GPT-6.1 Sol's API cache-read rate](https://developers.openai.com/api/docs/models/gpt-6.1-sol) is 5% of its uncached input rate, down from [10% for GPT-6 Sol](https://developers.openai.com/api/docs/models/gpt-6-sol). [DeepSeek V4.1 Flash's cache-hit rate](https://api-docs.deepseek.com/quick_start/pricing/) is 2% of its cache-miss rate. These discounts reduce the cost of reusing history when the requests hit the cache.

Clearing or compacting can interrupt that reuse. OpenAI's [caching guidance](https://developers.openai.com/api/docs/guides/prompt-caching#compaction-can-reduce-cache-reuse) explains how compaction can change the cached prefix, but also notes that fewer input tokens can still reduce total cost. A lower cache-read price doesn't make an obsolete hypothesis useful. Preserve the state that matters when moving on, and let the next task's needs determine what to carry forward.

Where the tool supports it, tell compaction what to preserve. In Claude Code, for example:

```text
/compact Preserve the current goal, constraints, modified files,
verification results, rejected approaches and the next step.
```

## Save a handoff before clearing

A fresh session won't have the previous conversation's working state unless you provide it. A handoff records that state in a document you can review and give to the next session or another agent. Use compaction to continue within the same conversation. Use a handoff when restarting or transferring unfinished work, or when you need a durable record of progress outside the chat. You can also save one before compaction if an important decision needs to survive independently of the generated summary.

Before clearing, ask the agent to write a short handoff and check it against the files and results. Anthropic's [work on long-running agents](https://www.anthropic.com/engineering/effective-harnesses-for-long-running-agents) uses progress notes and Git history to help each session recover the current state.

A handoff can use this structure:

```text
Goal: The outcome and how to verify it.
Constraints: Requirements and what is out of scope.
Current state: Files changed and work still pending.
Decisions: Chosen approach and why. Rejected approaches and why.
Evidence: Checks run, results and anything unverified.
Open questions: Unknowns or decisions still needed.
Next step: The next concrete action.
References: Files, source links and logs to consult.
```

Keep uncertainty explicit. “Likely caused by a stale cache” must not become “caused by a stale cache” in the handoff. Preserve exact error messages or source excerpts when the next step depends on their wording.

Start the new conversation with the handoff and access to the current work. Ask the agent to inspect the relevant files before continuing, since the notes may no longer match them.

## Break tasks at useful boundaries

Handoffs are easier when the work has reached a point where its result can be stated clearly. For a migration, an investigation might establish the dependency map before design work begins. Once the contract is agreed, implementation can proceed without carrying every design discussion into its context.

Split when the next phase needs the findings but little of the exploration that produced them. Finish each phase by recording its result and remaining uncertainties. This follows the [milestone approach in OpenAI's long-running task guidance](https://developers.openai.com/blog/run-long-horizon-tasks-with-codex).

Keep work together when the parts depend on details that are difficult to transfer. Changing an interface and updating its callers may belong in one task. Splitting every file into a separate conversation can create more handoff work than it saves.

The same boundary applies when handing work to a sub-agent. Its task needs an outcome you can check and enough context to work independently.

## Conclusion

Context management starts with the information that accompanies every task. Short, specific base instructions establish recurring constraints. Skills supply procedures for particular kinds of work, while reference docs hold the background and decision history. Keeping those sources current and loading them when needed gives the agent useful guidance without making every task carry the project's entire documentation.

During the task, a clear scope helps the agent choose what to read. A larger window helps when a large body of evidence needs to be compared in detail and summaries could lose exact wording or exceptions. As the work changes, compaction can preserve progress within the conversation, while a checked handoff lets a fresh session continue from the accepted decisions. The agent can then carry on from an agreed state, with the evidence available for you to check.

Later posts in this series will cover verification, including how to check an agent's conclusions and the work it produces. They'll also look at guardrails for actions that need approval, and at using sub-agents with clear responsibilities and results that can be brought back into the main task.
