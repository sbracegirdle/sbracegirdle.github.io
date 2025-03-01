---
layout: ../layouts/Post.astro
title: Are manual gates always bad?
date: 2023-07-20
tags: workflow agile software lean devops systems
author: Simon Bracegirdle
description: Waterfall and manual gates bad, agile good, as the industry wisdom goes. But is that always the case and are we glossing over some nuance?
image: gated-workflows
---

Gated workflows are where there's a manual hand-over or approval step that must occur before the next step in the process can start. Common examples of this is that design must occur before build, with a hand-off meeting or document. Another example is that testing must occur before deployment with a tester sign-off. This breakdown of the steps into linear sequential steps is also called [waterfall development](https://en.wikipedia.org/wiki/Waterfall_model).

This was the normal way to build software until the [agile manifesto](https://agilemanifesto.org/) came along, which promoted a different set of principles:

> - **Individuals and interactions** over processes and tools
> - **Working software** over comprehensive documentation
> - **Customer collaboration** over contract negotiation
> - **Responding to change** over following a plan

This is a big contrast to waterfall — you can't entirely do design before development if you need to **respond to change**, for example changing the design after the first iteration because you thought of a more user friendly approach.

You can't depend solely on big hand-over documents if your focus is on **individuals and interactions**. Writing specifications down is a good idea, but a two-way discussion is a better way to achieve a shared understanding, since participants can ask questions for clarification.

But is waterfall that bad? Is an agile approach best in all situations? Let's dig in and find out.

## Benefits of gated workflows

### Simple and intuitive

Traditional waterfall style workflows are simpler because they are easy to think about. Do A first, then move on to B. Simple, right?

This might work for some, but it can cause more harm than good, since working sequentially often doesn't work when the product needs iteration. For example, what if we find a technical constraint that forces us to re-think the design? What if we find a better design that is intuitive to use? If the project needs to revert back to a "design phase" every time this happens, it can slow down the entire project.

Having said that, I think this can work in environments where we have a high degree of confidence in what we're building. If you're 100% sure that the design fits the purpose and has no technical constraints, then a waterfall approach could work.

### More control

If the design or quality assurance teams require more control over changes to the product, then adding a manual gate provides this. This step empowers the approving party, since it prevents the other party from proceeding.

This is a natural step to take in dysfunctional environments where the output is not meeting a high enough standard. 

But this again can cause more harm than good because now everything must pass through the gate. This can increase the length of time to get feedback or changes out to customers. It also hints at a lack of trust between teams.

## Drawbacks of gated workflows

### Longer feedback loops

Gates create longer feedback loops. When you finish a task, you hand it off, or send it off for approval, then you wait for the other party. People move on to other tasks in the meantime. The other party also has their own set of tasks to complete and may take a while to get to yours.

When the other party does come back with feedback, it can then be frustrating to deal with because you've moved on to another task and need to context switch back to the original task to deal with the feedback. Then the whole process repeats until approved. The longer the feedback loop, the greater the frustration for everyone involved, and potentially your users who are also waiting.

Indeed, [Lean](https://en.wikipedia.org/wiki/Lean_software_development), originating from the [Toyota Production System](https://en.wikipedia.org/wiki/Toyota_Production_System), talks at great length about forms of waste that can slow down production of goods. I believe these ideas are also applicable to the software development process. Lean identifies different forms of waste that are applicable to hand-offs and manual gates:

- **Waiting** — Idle time that occurs when materials are not ready — waiting for approvals, feedback or hand-offs in our case .
- **Transport** — Unnecessary movement of materials — such as when we pass ownership of code changes to another team.
- **Motion** — Unnecessary movement of people — such as when people need to context switch back to a task after receiving feedback.
- **Defects** — Materials that are out of specification which require resources to correct — in our case this means bugs.

If we can find ways to remove these forms of waste then we'll be on our way to be able to **respond to change**.

### Siloed responsibility

When we break the development process into discrete parts that are each handled by a different party, it can result in hand-balling of responsibility. For example; "oh, no the designers are responsible for design" or "the testers are responsible for testing".

I've seen this first-hand where adding a dedicated testing team resulted in less testing by engineers because they started to depend on the testing team for that. As a result the quality of code produced dropped, and bugs increased.

### Discourages collaboration

Having hand-offs reduces collaboration because work is serially implemented, so there's less impetus for discussion. Everyone tends to follow the process, which might include an approval or hand-over document. It's easy to assume our understanding of the document is correct, unless we go out of our way to clarify it.

## What's the agile way then?

For teams where waterfall style methods aren't working, what's the alternative? How can we become more agile? This isn't about adopting scrum or other methodologies, but instead about some small habits that we can change to work more effectively.

### Create collaborative habits

I think this is the simplest and highest value thing to change — find ways to encourage continuous collaboration between designers, engineers, testers, and other individuals. One way is to organise regular catch-ups or stand-ups. This can be either asynchronous in Slack, or synchronous, as long as they're achieving increased two-way information flow.

If we automate this through repeating calendar invites, Slack bots, or other methods, then it helps by reducing the cognitive load and taking the impetus off the individual to remember.

### Set a high standard

Removing manual gates on its own isn't going to give you increased code quality. In fact, it can be dangerous to remove workflow controls without also making a change in expectations. This assumes that there's impetus within the team to improve how they work. If the team isn't interested in improving, then making big workflow changes will cause frustration.

If team members are to **share the responsibility** of design, testing, and other factors, then we should define the specifics of that responsibility in writing and communicate them. This means re-iteration of the expectations until it sinks in. 

Examples could include; should we commit test changes for every code change? What is our code coverage target? What happens if there's a bug in production?

The senior and experienced members of the team should be leading by example here. If the Lead Engineer is making code changes without testing, or with poor testing, what kind of message does that send to the team?

### Automate as much as possible

Setting a high standard, and automating that standard, can reduce the cognitive load of the team, and ensure that they exceed the bar by default. An example of this is to check that we meet the test coverage target in CI/CD before allowing code merges.

### Remove the gates

Once collaboration has improved, and we've raised our quality bar, then we can remove the gates.

## Summary

Waterfall style processes cause a lot of issues in team environments that work iteratively. But waterfall can work for you if you have a static product that is unlikely to evolve or change, and where the technical constraints are well known.

For the rest of us that are still figuring out our products through iteration and feedback, then we should consider adopting a more agile approach to working.

Thanks for reading.
