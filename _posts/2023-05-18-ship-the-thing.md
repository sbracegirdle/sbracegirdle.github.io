---
layout: post
title: Ship the thing — what's getting in the way?
date: 2023-05-18
tags: waste lean devops systems stoicism
author: Simon Bracegirdle
description: What's stopping us from shipping the thing? Are there friction points or waste getting in the way?
image: ship-the-thing
---

Now and then I think it's worth taking a step back and reflecting on what's stopping us from reaching our goal of shipping a new product or feature to customers. The stated goal "ship the thing," seems so simple, almost condescending, so why do we find it so hard?

Sometimes it's the sheer volume of work — we need to write code, test with users, iterate after receiving feedback. Everyone has a long list of tasks that they need to complete.

Hidden amongst any product journey is work that's not *really* needed, things that get in our way, distract us, or slow us down. In some of my experiences in the past, *most* of the work of the project fell into this bucket. A lot of the time it's accepted as okay, normal, or even expected.

The [Google Site Reliability Engineering (SRE) book](https://sre.google/workbook/table-of-contents/) introduced a similar idea that they call *toil* — repetitive manual work necessary to keep the system running, but doesn't contribute to the system's stability or strategic development. In other words, it has no *value*.

In the [Lean Manufacturing and Toyota Product System](https://en.wikipedia.org/wiki/Lean_manufacturing) they talk about continuous improvement and the need to remove waste. They even define seven forms of waste that can happen in manufacturing, but most of which are also relevant to software — transportation, inventory, motion, waiting, over-production, over-processing, defects.

Going even further back, Marcus Aurelius of the Stoics made this astounding statement that's still relevant to our industry today:

> "The impediment to action advances action. What stands in the way becomes the way".

What things get in our way? Sometimes it's obvious — we'll experience some friction or frustration that needs fixing up. But other times it's not obvious. That's why it can be valuable to set time aside to reflect on or map out the tools, systems and processes that we use and analyse where the pain points are.

This is starting to sound a lot like systems thinking — a method for looking at systems, such as organisations, software teams, or CI/CD pipelines, as a whole and breaking them down into sub-systems and components. Deming wrote about this in 1986, but the ideas go even further back.

The other way that we can get slowed down is when we've adapted to the inefficiency of tool or system. I'm calling this "learned waste" — waste that we don't even realise is there because everyone's been doing it this way for so long they forget that there's better ways to work. I suspect there's cases where the entire industry has learned waste on certain tools and methods.

We've talked a lot about waste and friction at this point, so let's look at a few examples from my own experience.

Do we need that overcomplicated or overabstracted architecture? A lot has been said about the whole microservices versus monolith debate, but I think it's entirely pointless arguing about it without considering context.

If we don't have any users yet, or we have less than say 20 engineers, then a complicated microservices architecture with 30+ repositories, 20 data stores and 10000 lines of AWS CloudFormation code should be a massive red flag. But if we're Google and we need to handle 10 quadrillion requests per second, then a monolith could be a red flag.

What's the simplest possible architecture that's fit for our purpose and has the least friction in our context? At earlier points in our product lifecycle we need to focus on shipping fast. We can build our "dream architecture" later when we're actually making money with real users and discover that we actually need to handle more than one request a minute.

Do we need those bugs? When we're building a prototype, a personal project or an early stage product we can accept bugs, because we don't know if the thing is going to survive at all. But for established teams bugs slow us down and frustrate our users. In that case there are ways to increase code quality, so why aren't we using them?

I'm a proponent for writing tests, but again it depends on the context. A mature product with real users needs to be stable and reliable, and the half-dozen team of engineers need to be able to make regular changes with confidence they won't break the system — tests are paramount for this team. We can do this by writing tests either first (TDD), or after. I don't have time for dogma, I want to know what's going to help us ship and continue to ship.

It's worth noting that even if we're building a prototype, code written without tests can make writing tests harder later. This is why people will advocate for discarding a prototype and re-writing it once we know this is going to be a real product with real users.

Do we need slow CI? This is a common one. If we make code changes throughout the day and need to wait 20 minutes for each code change to build, test, and deploy, how much do we think that's going to add up to over time? It takes some simple math to realise it's worth spending time optimising this process.

Is our language, library or framework choice fit for purpose? Is there a lot of boilerplate or overhead that will slow down the building of features? Does the framework tend to lead to verbose or complex code that's harder to work with? Again, a lot has been said about the Ruby on Rails/Laravel vs React debate, but I think both are valid (or poor) choices depending on context.

In summary, we've all got friction points holding us back in our context. Let's use our intuition, but also don't — consider using systems thinking and other analytical methods for finding the pain points in our ecosystem. The impediments should become "the way," work hard to remove them, but with the goal of enabling us to *ship the thing*.