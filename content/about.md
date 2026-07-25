---
title: About me
description: A mini CV — the work I keep coming back to, and the skills behind it.
---

I'm Simon Bracegirdle — a software engineer in Perth, Western Australia. I've spent 20+ years building products: education software at [SEQTA](https://seqta.com.au), cloud consulting at [Mechanical Rock](https://mechanicalrock.io), and now government-relations tooling at [GovConnex](https://govconnex.com/).

This page is the short version — not a full CV, just the work I keep coming back to and the skills it built.

## AI engineering, end to end

Since 2024 I've built AI features that customers pay for, not demos. An AI research assistant with agent orchestration, tool use, and a deep-research mode. Automated briefings that read the day's political events and write the summary before the customer wakes up. An embeddings pipeline feeding semantic search.

The unglamorous parts matter more than the prompts: per-user cost caps and usage reporting, [evals](2025-05-22-write-evals.html) to catch regressions before customers do, prompt caching to hold down latency and spend, and hardening against prompt injection.

<p><span class="tag">LLM agents</span> <span class="tag">tool use</span> <span class="tag">embeddings</span> <span class="tag">evals</span> <span class="tag">prompt design</span></p>

## The whole stack

React and TypeScript up front; Node, NestJS, and GraphQL behind; MySQL and Elasticsearch under that. I've shipped a Microsoft 365 email integration for CRM, realtime alerting built on Elasticsearch percolation queries, newsletter and monitoring pipelines, and the scrapers that keep parliamentary data flowing into all of it.

<p><span class="tag">TypeScript</span> <span class="tag">React</span> <span class="tag">NestJS</span> <span class="tag">GraphQL</span> <span class="tag">Elasticsearch</span> <span class="tag">MySQL</span></p>

## Cloud and observability

AWS is home ground: ECS services, Lambda and SAM stacks, SQS pipelines, Cognito auth. I rolled out OpenTelemetry across the platform in 2022, including tail-based sampling that cut trace ingestion costs. I want systems that [tell you what's wrong](2023-05-18-important-logs.html) before customers do — I've written up [my take on OpenTelemetry](2023-03-23-opentel-thoughts.html).

<p><span class="tag">AWS</span> <span class="tag">serverless</span> <span class="tag">OpenTelemetry</span> <span class="tag">Datadog</span> <span class="tag">CI/CD</span></p>

## Migrations, landed

Some of my proudest diffs are net deletions. I moved an entire production webapp from JavaScript to TypeScript while features kept shipping, migrated the e2e suite from Cypress to Playwright, replaced ad-hoc SCSS with design tokens, and removed legacy features once their replacements proved out. Modernising a codebase is a skill you practise, not a project you schedule.

## Sharper teams

Tooling and habits that make everyone faster: CI/CD pipelines, load testing, [parallel test jobs](2023-07-21-parallel-test-jobs.html), a [code review culture beyond LGTM](2022-03-08-dont-lgtm-code-reviews.html), and lately agent-assisted workflows — review agents, definition-of-done gates, and docs written for both humans and machines. Above all: [ship the thing](2023-05-17-ship-the-thing.html).

## Selected work

A greatest-hits list, one line per year:

<ul class="post-list">
<li><span class="date">2026</span>Agent workflows and deep-research mode; automated daily briefings; prompt-injection hardening<p>Orchestration, streaming UX, and the safety work that lets an AI product face real users.</p></li>
<li><span class="date">2025</span>AI research assistant from prototype to production<p>Report generation, usage caps and admin reporting, evals, and a vector-search service to feed it.</p></li>
<li><span class="date">2024</span>Webapp to TypeScript; first embeddings work<p>A 30,000-line migration landed without pausing feature work, and the groundwork for semantic search.</p></li>
<li><span class="date">2023</span>Realtime alerting and CRM email integration<p>Elasticsearch percolation for instant matching, search term tokenisation, and Microsoft 365 email sync.</p></li>
<li><span class="date">2022</span>OpenTelemetry rollout<p>Collector sidecars, tail-based sampling, and instrumentation that paid for itself in reduced ingest bills.</p></li>
</ul>

## Elsewhere

Code at [GitHub](https://github.com/sbracegirdle), thoughts on the [blog](index.html), books on the home page. If any of this sounds like a problem you have, get in touch.
