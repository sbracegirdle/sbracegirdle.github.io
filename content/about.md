---
title: About me
description: A mini CV — the work I keep coming back to, and the skills behind it.
---

I'm Simon Bracegirdle — a software engineer in Perth, Western Australia. I've spent 20+ years building products: education software at [SEQTA](https://seqta.com.au), cloud consulting at [Mechanical Rock](https://mechanicalrock.io), and now government-relations tooling at [GovConnex](https://govconnex.com/).

## AI engineering

Since 2024 most of my work has been AI features that ship to paying customers. The biggest is a research assistant that orchestrates agents and tools, with a deep-research mode for longer questions. I also built the automated briefings that read the day's political events and write a summary overnight, and the embeddings pipeline behind semantic search.

I've built the parts around them too: per-user cost caps and usage reporting, [evals](2025-05-22-write-evals.html) to catch regressions, prompt caching to hold down latency and spend, and hardening against prompt injection.

<p><span class="tag">LLM agents</span> <span class="tag">tool use</span> <span class="tag">embeddings</span> <span class="tag">evals</span> <span class="tag">prompt design</span></p>

## Web applications

React and TypeScript on the front end, Node with NestJS and GraphQL behind it, and MySQL and Elasticsearch for storage and search. I've shipped a Microsoft 365 email integration for our CRM, and realtime alerting built on Elasticsearch percolation queries. I also look after the newsletter and monitoring pipelines, and the scrapers that collect parliamentary data.

<p><span class="tag">TypeScript</span> <span class="tag">React</span> <span class="tag">NestJS</span> <span class="tag">GraphQL</span> <span class="tag">Elasticsearch</span> <span class="tag">MySQL</span></p>

## Cloud and observability

I work in AWS most days: ECS services, Lambda and SAM stacks, SQS pipelines, Cognito for auth. In 2022 I rolled out OpenTelemetry across the platform, including the tail-based sampling that brought our trace ingestion bill down. I want systems that [tell you what's wrong](2023-05-18-important-logs.html) while there's still time to fix it. I've also written up [my take on OpenTelemetry](2023-03-23-opentel-thoughts.html).

<p><span class="tag">AWS</span> <span class="tag">serverless</span> <span class="tag">OpenTelemetry</span> <span class="tag">Datadog</span> <span class="tag">CI/CD</span></p>

## Migrations

A lot of my better work has been deletions. I moved a production webapp from JavaScript to TypeScript without pausing feature work. Since then I've taken the e2e suite from Cypress to Playwright, swapped ad-hoc SCSS for design tokens, and removed legacy features once their replacements proved out. I've found this kind of work goes better done continuously than saved up for a scheduled project.

## Team tooling

I spend a fair bit of time on the things that make a team faster: CI/CD pipelines, load testing, [parallel test jobs](2023-07-21-parallel-test-jobs.html), and [code review that goes past LGTM](2022-03-08-dont-lgtm-code-reviews.html). Lately that's meant agent-assisted workflows too — review agents, definition-of-done gates, and docs written so both people and machines can follow them. The habit under all of it is to [ship the thing](2023-05-17-ship-the-thing.html).

## Selected work

The last five years, most recent first:

<ul class="post-list">
<li><span class="date">2026</span>Agent workflows, deep-research mode, and prompt-injection hardening</li>
<li><span class="date">2025</span>AI research assistant from prototype to production<p>Report generation, usage caps and admin reporting, and a vector-search service.</p></li>
<li><span class="date">2024</span>Webapp to TypeScript, and the first embeddings work<p>The migration ran to about 30,000 lines.</p></li>
<li><span class="date">2023</span>Realtime alerting and CRM email integration<p>Elasticsearch percolation for instant matching, plus search-term tokenisation.</p></li>
<li><span class="date">2022</span>OpenTelemetry rollout — collector sidecars and tail-based sampling</li>
</ul>

## Elsewhere

My code is on [GitHub](https://github.com/sbracegirdle) and the rest of my writing is on the [blog](index.html). The home page has what I'm reading.
