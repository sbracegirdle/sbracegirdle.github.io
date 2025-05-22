---
title: We should be writing evals
description: As software engineers increasingly integrate LLMs into products, prompt evaluations—using tools like promptfoo—are essential for systematically testing and ensuring prompt reliability, much like traditional software tests.
---

The role of software engineering is broadening — traditionally we design APIs, debug performance issues, write front-ends, document technical designs, unit test functions, among many other tasks. But now we're also straying into the area of machine learning engineering with the growing popularity and hype of LLMs, and having to integrate these systems into products and services.

If you're building these kinds of systems it's natural enough to "vibe it out" — spot check a few examples to see if it's producing something useful and correct. But if you need a reliable system that you'll be taking to production with actual users, then vibing isn't good enough. You need a systematic way to test your prompts that will give you confidence in its reliability.

This is where evals come in — they're a method of running a suite of scenarios against a prompt and model, and receiving a score for the defined assertions. This score allows you to make informed decisions about model selection and prompt optimisations.

Let's use the library [promptfoo](https://www.promptfoo.dev/docs/intro/) as an example, which can be installed with:

```sh
npm install -g promptfoo
```

You provide your prompt and one or more models (e.g. in `promptfooconfig.yaml`):

```yml
prompts:
  - file://prompts/greeting.txt
providers:
  - openai:gpt-4o-mini
```

Define a set of test scenarios, along with the assertions for each scenario:

```yml
tests:
  - vars:
      name: "Alice"
    assert:
      - type: equals
        value: "Hello, Alice!" # Response must exactly equal this
      - type: cost
        threshold: 0.001 # Cost must be under this
```

Model graded assertions specifically are very useful:

```yml
# normalized perplexity, i.e. how certain it is about it's predictions
- type: perplexity-score
  threshold: 0.8

# rubric-style criteria. i.e. have another LLM grade the response from the first LLM against a set of criteria
- type: llm-rubric
  value: "Use formal tone and mention metrics"

# closed-QA style. i.e. similar to llm-rubric but uses OpenAI closed QA eval.
- type: model-graded-closedqa
  value: "References Australia in some way"

# relevance check. i.e. uses a combination of embedding similarity and LLM evaluation to determine relevance.
- type: answer-relevance
  value: "Output should explain Promptfoo usage"
```

And then execute the evals:

```sh
promptfoo eval
```

The result is a report that scores the performance of each prompt/model, view it in your browser with:

```sh
promptfoo view
```

I see them as a necessity going forwards for integrating LLMs. In the same way that we expect tests written for traditional software, we should expect evals written for prompts.