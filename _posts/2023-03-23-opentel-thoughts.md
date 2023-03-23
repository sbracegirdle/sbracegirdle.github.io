---
layout: post
title: OpenTelemetry and the future of monitoring and observability
date: 2023-03-23
tags: aws fargate ecs openelemetry observability monitoring traces sidecar container
author: Simon Bracegirdle
description: I share my personal thoughts and experience with OpenTelemetry in 2023 — the benefits, limitations, and impact on monitoring and observability by this game-changing, vendor-agnostic framework.
image: opentel-thoughts
---

<!-- > What is OpenTelemetry, and why should people in the tech industry be interested in learning more about it? -->

[OpenTelemetry](https://opentelemetry.io/docs/) is a collection of standards and tools designed to help you add telemetry to your application or service. By telemetry, I mean metrics, traces, and logs, which are critical for understanding the state of any deployed software system.

<!-- > How has the development of OpenTelemetry impacted the monitoring and observability landscape, and what are its key benefits for developers and organizations? -->

The key benefits that OpenTelemetry brings is standardisation and vendor agnosticism. You can instrument your code once and then with the help of the [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector) create configurations that pipe your telemetry data to one or more back-ends. There's integrations with popular SaaS vendors in the space such as DataDog, AWS CloudWatch, Honeycomb, and others.

<!-- > What are some common use cases for implementing OpenTelemetry, and how does it help developers gain insights into their applications' performance and reliability? -->

For example, let's say your deploying a new web application as a container to a cloud service such as AWS ECS, Flyio, GCP containers, or one of the other options. To understand the behaviour of that application in production, and help debug issues, or receive alerts when something is going wrong, telemetry is _essential_.

To start with OpenTelemetry you would add [instrumentation](https://opentelemetry.io/docs/instrumentation/) to your code. A good starting point is to use the auto instrumentation tooling provided by the community. These are available via the [SDK and API libraries](https://opentelemetry.io/docs/instrumentation/) provided for each language supported by OpenTelemetry.

You may also decide to run a [OpenTelemetry Collector](https://github.com/open-telemetry/opentelemetry-collector), which is a separate process that can add processing such as batching and sampling, to help prepare, filter, and massage your data before sending it to your monitoring and observability back-end. A common configuration is to deploy the collector as a side car container on the same host as your application. They would then communicate over OTLP — the OpenTelemetry protocol — over HTTP or gRPC.

With that context in place, let's take a look at the limitations and challenges with OpenTelemetry.

## Limitations and challenges

<!-- > What are some limitations or challenges developers might encounter when using OpenTelemetry, and how can they overcome these obstacles? -->

The biggest issue for me so far is that some tooling isn't yet mature and has some rough edges. For instance, I encountered a strange issue with exporting logs with the DataDog exporter, and ended up debugging the problem myself and submitting a PR to the [contrib repository](https://github.com/open-telemetry/opentelemetry-collector-contrib). It's notable that this is also a benefit of the ecosystem — that the community can contribute fixes and improvements to the project — so I think over time the community will achieve a stable and reliable tool, but there's some challenges as it stands today in 2023.

To illustrate this further, if you browse through the [Collector contrib](https://github.com/open-telemetry/opentelemetry-collector-contrib) repository , you'll notice components in the ecosystem are in beta or alpha state. These components are evolving fast, with breaking changes occurring. This can create difficulties in maintaining up-to-date dependencies on the ecosystem.

<!-- > With these limitations in mind, what strategies would you recommend for developers looking to stay on top of updates and improvements? -->

To mange that it's wise to set a regular cadence for applying updates, and setting time aside for working through any major changes that impact you.

Another challenge is that adding instrumentation to new services can be a bit more work than using libraries from specific vendors. Those vendors have had time to optimise their on-boarding and developer experience, understandable given it's criticality to their business. But I think once you have deployed OpenTelemetry to at least one service in your environment, you'll have established a pattern that makes it easier to copy to other services, plus the generative text tools available these days can help to cut down on boilerplate.

For example, setting up the infrastructure for the OpenTelemetry Collector can be a bit of added work. First you need to choose a distribution for the collector, such as the [contrib distirbution](https://github.com/open-telemetry/opentelemetry-collector-contrib), one from a vendor such as AWS, or by [building one yourself with the builder tool](https://opentelemetry.io/docs/collector/custom-collector/). Then you need to add a YML configuration for your collector, which defines how to receive, process and export telemetry data. Then you need to build an image for the collector that embeds your configuration and publish it to a container repository in your ecosystem. Then we deploy the collector along side the application, which we configure to pipe telemetry data to the collector via OTLP, which then forwards the data on to a backend.

This is a lot of steps and moving parts, and I'm sure it's daunting for newcomers, but once you've stepped through the process and familiarised with it, I don't think it's too arduous or complex.

## Highlights and strengths

<!-- > What are some of the most notable strengths of OpenTelemetry, and how do these strengths set it apart from other monitoring and observability solutions available in the market? -->

I think the biggest strength of OpenTelemetry is the amount of flexibility and power in the tools provided. For instance, the community has provided a lot of components for the Collector that allow processing data in a range of ways. For example, if we want to sample our traces before sending them off to DataDog, which can be quite expensive if you're ingesting every single trace and span, then it's a matter of adding the `tail_sampling` component and adding a configuration like below:

```yml
receivers:
  # ...

processors:
  tail_sampling:
    decision_wait: 10s
    num_traces: 1000
    expected_new_traces_per_sec: 10
    policies:
      - name: sample
        type: string_attribute
        rate_limiting:
          spans_per_second: 100
        string_attribute:
          key: "sample-me"
          values: ["true"]

exporters:
  # ...

pipeline:
  # ...
```

Using `tail_sampling`, we can create a wide range of policies that will sample traces with specific attributes. An example could be to include less traces from health checks, but ensure errors are always included, or that slow requests are always included.

Another powerful feature of OpenTelemetry is the ability to test locally, and to export to open source tools such as Jaeger for traces, without the need to work with a third party vendor that might charge you for data ingestion. This can help you to iterate faster during development before deploying.

<!-- > How does the OpenTelemetry project promote collaboration and community involvement, and what are some examples of how the community has contributed to its growth and success? -->

The OpenTelemetry project has collaboration and community involvement at its core. Being an open-source project, it encourages developers and organisations to actively participate in its growth and development. The project maintains a strong presence on GitHub, where developers can contribute to the codebase, report issues, and suggest improvements under the Apache license Version 2.0, which is permissive license that allows commercial use.

The community's involvement has led to the development of integrations, exporters, and instrumentation libraries for a range of programming languages and platforms. Examples of community-driven contributions include the development of OpenTelemetry SDKs for languages like Java, Python, Go, JavaScript, also exporters for popular backends like Jaeger, Prometheus, Zipkin, and DataDog. The community also contributes to documentation, tutorials, and sharing best practices to help other developers adopt OpenTelemetry.

## Conclusion

In conclusion, the current state of OpenTelemetry shows great promise in standardising and democratising observability and monitoring across different platforms and languages. Challenges exist, such as adding instrumentation to new services being more involved than equal libraries from vendors. But despite these on-boarding challenges, I think the benefits of vendor agnosticism, and the powerful tools available, will compound in the long run.

As the OpenTelemetry ecosystem continues to evolve and mature, we can anticipate further improvements in its tooling and developer experience, solidifying its position in the world of observability and monitoring. I predict, and hope, that in a few years, it'll become the default method for software teams intrumenting their code.

Thanks for reading! Please reach out on the socials if you'd like to discuss further.
