---
layout: ../layouts/Post.astro
title: Scaling out test jobs in GitHub Actions
date: 2023-07-21
tags: github actions ci cd tests devops
author: Simon Bracegirdle
description: Distributing your test suite over concurrent jobs in GitHub Actions isn't as straightforward as it first seems.
image: parallel-test-jobs
---

## Introduction

It's a common problem that tests and other tasks are slow in CI/CD, and I think it's always good to find the root cause of the problem before jumping to solutions. Sometimes it might be because of inefficient code or latency, but other times it could be compute constraints. In that case a bigger box would make it run faster.

[GitHub added more compute options to Actions](https://docs.github.com/en/actions/using-github-hosted-runners/about-larger-runners), giving us the ability to use larger runners where necessary, which was a welcome addition. Another way that we can add more compute is scaling horizontally by adding workers to our jobs. But, if you want to automatically distribute work across those workers, there isn't an out-of-the-box solution to that.

In this post I'll show you how you can automatically distribute tests across parallel jobs using the matrix strategy and some bash script magic in GitHub Actions.

## Adding workers

The way that you scale out a GitHub action job is to use the `matrix` strategy. The most common use case of this is to run a job with a range of configurations. For example, running across a set of browsers:

```yml
strategy:
  matrix:
    browsers: [chrome, firefox]
```

But sometimes you want to distribute a single piece of work, such as a test suite, across concurrent jobs. This is not as straightforward as it first seems.

## Parallel tests

Let's keep in mind that popular test runners like Jest already support parallelisation natively within a single job, so you'd consider concurrent jobs or a larger worker when you have compute constraints. Horizontal scaling (more workers) makes more sense than vertical scaling (larger worker) in cases where there's contention within a single job. In my case I had an end to end test suite that required a full browser environment, which causes contention.

The simplest solution to add more jobs would be to list every test file in a matrix:

```yml
strategy:
  matrix:
    test_name: [authors.test.js, books.test.js]

steps:
  # ...
  - run: npm run test -- ${{ matrix.test_name }}
```

But this means you need to remember to update your action file whenever you add a test. It also limits you to a single test per job and could get cumbersome if you have a large number of tests in your code base, so I don't think this is ideal for a lot of cases.

Let's look at another option...

## Distributing tests

My solution was to come up with a script that reliably and deterministically selected tests based on the index of the worker across a known count of workers. This means we need to provide two options: a `matrix` of job indexes, and a count of the total number of jobs:

```yml
env:
  JOB_COUNT: 6 # Needs to match count of strategy.matrix.job_idx

strategy:
  matrix:
    job_idx: [1, 2, 3, 4, 5, 6] # Needs to be numeric IDs in sequence.
```

Then we need a script to grab the tests (specs) to run in each of the current jobs. This script should reliably select the same tests on each run, so that if we re-run the job it'll run the same tests. It should also not duplicate any tests, so they run once across the set.

Here's the script:

```yml
steps:
  # ...
  - name: Get spec list for concurrent job
    id: specs
     env:
       JOB_COUNT: ${{ env.JOB_COUNT }}
       JOB_IDX: ${{ matrix.job_idx }}
     run: |
       all_specs=$(find src/ -name '*.test.js' | sort)
       spec_count=$(echo "$all_specs" | wc -l)
       per_runner=$(( spec_count / $JOB_COUNT ))
    
       if [[ $JOB_IDX -lt $JOB_COUNT ]]; then
         spec_list=$(echo "$all_specs" | sed -n "$(( ($JOB_IDX-1) * per_runner + 1 )),$(( $JOB_IDX * per_runner ))p")
       else
         spec_list=$(echo "$all_specs" | sed -n "$(( ($JOB_IDX-1) * per_runner + 1 )),$spec_count p")
       fi

       # Convert the newline-separated list of specs into comma-separated for the action's input.
       spec_list=$(echo "$spec_list" | paste -sd, -)

       echo "spec_list=$spec_list" >> $GITHUB_OUTPUT
```

There's a fair bit going on here so let's break this down:

- The `all_specs=` line uses `find` to match any files in `src/` that are tests using the `*.test.js` pattern. You'd adjust this to match your test file pattern and folder structure.
- `spec_count` uses `wc` to count the number of spec files found
- `per_runner` is some math to get the count of specs to run per runner
- The `if` and `else` block uses `sed` to extract the names of the tests using line numbers and the job index.
- `spec_list` converts from a line separated list of test files into comma separated
- Then we put it in `GITHUB_OUTPUT`

To use this the generated spec list, we can refer to the output of the step in another step using the `id`:

```yml
steps:
  # ...
  - run: npm run test -- ${{ steps.specs.outputs.spec_list }}
```

This script is a bit messy, so it might be nicer to break it out into an independent action to clean it up. As of the writing of this post, I haven't done that yet.

## Summary

We used a bash script to select the tests to run on a per-job basis. We could do this because we provided the position of the job within the set of jobs. This information allows us to select tests reproducibly and without duplication. The bash script itself is a bit messy, so we could tidy it up by putting it behind a custom action.

Thanks for reading, I hope this helps you with improving your CI workflow.
