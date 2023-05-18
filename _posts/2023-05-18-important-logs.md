---
layout: post
title: Remembering the important bits to log
date: 2023-05-18
tags: logs devops observability monitoring software
author: Simon Bracegirdle
description: Explore a useful mnemonic for remembering the essential information to log in code for supporting debugging, without overcomplicating it.
image: important-logs
---

Logging can be a mixed bag, I've seen it done well and not well, and I've been guilty of both myself. Even though it's one of the tools to achieve observability, I still think it's important for engineering teams operating products and services.

When there's not enough logs, or other kinds of observability telemetry, then it can be difficult to understand what's going on inside a system, which can be painful when trying to debug an issue. If our database connections are failing and we didn't log the error, it might take us longer to understand what's happening.

When logs are too verbose and noisy, it can make it difficult to search for and find the information we need, and it can increase costs depending on how we're ingesting and storing logs.

Achieving the right quantity and quality of logging is difficult to master, and I'm not claiming to be a master myself, but the guidance in this post helped me, so I hope it helps you too.

Today I present a handy acronym — **REDIT** (in case we needed another one) — to help us remember what I think are the key bits of information to log. Let's explore it further.

## R — Request/response

*Log key context from the request and response payloads.*

Capturing the input and output of a system is fundamental to understanding its usage. This might include capturing the requesting user ID, the record ID, date range, user's browser, or source IP address. This isn't an exhaustive list!

Add anything that might be conceivably useful when diagnosing a production issue, or trying to understand how users use the system.

Whether we log the entire request or a subset depends on context. We don't want to log the entire payload when a user is submitting large content, such as for a blog post or multimedia, but perhaps logging the content length or key attributes would be helpful.

Be careful to avoid logging personally identifiable information (PII), or anything sensitive or private — this could lead to regulatory issues, privacy issues, or losing the user's trust. It's important that we're mindful of the sensitivity of what we put into logs.

Below is an example of setting up Python's Flake framework to log every request. Note that we don't log every header, or the request body, as they may contain sensitive information such as authentication tokens or private data. We'd also log the response and attributes like status code, response time, and error code, but I've excluded an example of that for brevity.

```python
@app.before_request
def log_request_info():
    headers = request.headers
    # Selectively log safe headers to avoid leaking sensitive information into the log
    required_headers = {k: headers.get(k) for k in ["User-Agent", "Accept-Language"]}

    log_data = {
        "remote_address": request.remote_addr,
        "url": request.path,
        "method": request.method,
        "headers": required_headers,
    }
    app.logger.info(json.dumps(log_data))

# ... also log after_request ...

# Requests to the following route will be logged:
@app.route('/')
def hello_world():
    return 'Hello, World!'
```

*CALLOUT — All code samples in this post are un-tested pseudo code for demonstration purposes.*


## E — Errors

*Log errors, and any other surrounding context such as stack traces and identifiers.*

When something goes wrong, we need to understand the problem so that we can work to resolve the underlying issue. If we don't understand the problem then our hands are tied and we don't know where to look, or what even happened.

An error that gets swallowed is a disaster, we'll find out about these when a user reports a problem with the system, and we'll be powerless to solve the problem because the logs don't indicate anything is wrong. When in this situation we must resort to trial and error to isolate the issue, or by fixing the logs.

Logging errors is essential and the bare minimum for any sane production system.

Context is important with errors too, so be sure to include the error message itself along with any pertinent IDs, stack traces and any other context that help us understand what was happening, where, and when the issue happened.

Logging errors is one place where we can afford to be a bit more verbose too, since they shouldn't, in theory, be happening too often, so there's more value in maximising information about the error, with little cost.

Here's an example of logging an error with a good amount of context in Python:

```python
def create_user(user):
    try:
        # ... Create user code

    except Exception as e:
        log_data = {
            "event": "user::create::error",
            "message": str(e),
            "user_id": user.id if user else None,
            "username": user.username if user else None,
            "error": repr(e),
            "stack_trace": str(sys.exc_info())
            # Add any other userful context you want to log
        }

        logging.error(json.dumps(log_data))
        
        # Handle the error etc...
```

## D — Dependencies

*Log calls to third-party dependencies such as external APIs or cloud services.*

Trying to understand an issue in a distributed system can be challenging, so it's critical to understand any interaction with third party APIs such as AWS, SendGrid, GitHub, or anything else you're using.

Logging the entry and exit points can be a big help when trying to understand the flow of data in the system and across systems. Of course, traces is, in general, a better tool for this job. But having logs is also a good idea for local debugging, or in case we sample out the span, or the span doesn't contain the attributes we need. Redundancy is nice.

To give you a concrete example — I encountered an issue in a call to the AWS `SQS.batchMessage` endpoint. The call site wasn't checking the response, but instead expected it to throw an error when an issue occurred. But this is a batch endpoint, and doesn't throw errors, but instead returns them in the payload. This lead to a bug, and we didn't any logging to help us understand what was happening, and the investigation took longer that it should have.

This highlights the need to log key attributes from the response at the call site. Not everything will bubble up into an error when something goes wrong.

Here's an example of logging an API call site:

```python
def create_user(user):
	log_data = {
		"event": "user::create::api_call",
		"message": "Calling the user service API."
		"user_id": user.id,
		"username": user.username,
		# Add any other userful context you want to log
	}
	logging.info(json.dumps(log_data))
	
	response = user_service.call_api('create', user)

	# ALSO LOG THE RESPONSE HERE!
	response_log_data = { 
        # Any useful context from the response
    }
	logging.info(json.dumps(response_log_data))

	# ...
```

## I — Important events

*Log any important system or business events that occur.*

It can also be useful to log business events that occur. For example, if you're building a book management system, you might log business events such as — user left a review, book created, user signed up, etc.

These events provide context that help debug problems. If the user created a book, and then later failed to cancel the book, finding the original create event could help to understand why the cancel failed. It's rare that anything happens in isolation, having extra context is helpful. Perhaps the book creation used values we didn't expect, capturing pertinent attributes would be helpful in this case.

An example:

```python
def create_book(book):
	# Book creation process goes here...

	# If successful:
	log_data = {
		"event": "book::created",
		"message": "Book successfully created"
		"book_id": book.get('id'),
		"title": book.get('title'),
		"author": book.get('author'),
		"genre": book.get('genre'),
		# Add any other userful context you want to log — but nothing SENSITIVE or PRIVATE.
	}
	logging.info(json.dumps(log_data))
```

## T — Trace IDs

*Log any IDs that help you to trace a request as it passes across a distributed system.*

Logging is just one tool to achieve observability within your operational systems. Traces are another tool that are quite good for understanding how a request passes across services and layers of a service.

In fact, people are now claiming that traces are only thing you need, due to the idea of *wide events*. These are traces packed with enough metadata that they're useful for diagnosing issues on their own. I think there's some validity to this, but logs can still be helpful and complementary.

Traces and logs can work together by linking them with what's called a correlation ID or trace ID. Some libraries provide this functionality for you. [OpenTelemetry for JS](https://opentelemetry.io/docs/instrumentation/js/instrumentation/) and the [winston instrumentation package](https://www.npmjs.com/package/@opentelemetry/instrumentation-winston) provide an option to inject trace and span IDs into logs, which are then correlated in your monitoring tool of choice. As an example, DataDog provides a logs tab within their trace viewer.

In OpenTelemetry specifically, there's the concept of events, which are simple log-like objects nestable within spans, co-locating the data, which reduces the chance of missing logs due to differing sampling rules by data type. I think the guidance in this post also applies to events.

## Summary

To recap, the REDIT acronym stands for:

- R — Capture **r**equest/response metadata
- E — Capture **e**rrors
- D — Capture calls to external **d**ependencies
- I — Capture **i**mportant business and system events
- T — Capture **t**race IDs and link traces to logs to increase observability even further

I hope this can serve as a useful mnemonic for remembering the important bits to log and send you on the way to observability nirvana. Thanks for reading.