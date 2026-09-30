---
title: A basic LLM harness in TypeScript
description: A starting point for a tutor agent with the Vercel AI SDK, covering tools, saved conversations and streamed progress.
---

Suppose you're building a tutor agent to help students understand their course material. To answer a question, it may need to read a lesson, search for an example, then explain what it found. The student should be able to follow that work as it happens.

The harness is the application code that orchestrates the agent's work to complete the student's request. It calls the model, runs the tools the model requests, and passes their results back so the agent can continue. It also saves the conversation, streams progress and handles interruptions such as tool approval or a lost connection.

The [Vercel AI SDK](https://ai-sdk.dev/docs/agents/loop-control) provides the model loop and message types for this. The examples below build on those, leaving HTTP routing and database implementations to the application.

## Model configuration

Start with a model and instructions for the tutor agent. This example uses [GPT-6 Sol](https://developers.openai.com/api/docs/models/gpt-6-sol) through the SDK's OpenAI provider, with Zod for tool input schemas:

```sh
npm install ai@7 @ai-sdk/openai@4 zod@4
```

With `OPENAI_API_KEY` set in the server environment, the first call can answer a student question:

```ts
import { openai } from '@ai-sdk/openai';
import { generateText } from 'ai';

const model = openai('gpt-6-sol');
const system = `You are a student tutoring agent.
Help the student understand their course material, not do the work for them.
Read the lesson before explaining it. Cite sources for web examples.
Ask the student questions to verify their understanding, one at a time.`;

const result = await generateText({
  model, system,
  prompt: 'Explain why 1/2 is equal to 2/4.',
});
```

That gives you a first answer, but the model still can't read a lesson or search the web. Those need tools.

## Course tools

Give the tutor agent a way to read the lesson it's explaining. This tool searches the student's course and returns the relevant text:

```ts
import { tool } from 'ai';
import { z } from 'zod';

const courseTools = {
  readLesson: tool({
    description: 'Find relevant lesson text in the student’s current course',
    inputSchema: z.object({ topicSearch: z.string().max(100) }),
    execute: ({ topicSearch }, { abortSignal }) =>
      lessons.searchCourse(studentId, courseId, topicSearch, abortSignal),
  }),
};
```

This keeps the explanation grounded in the student's course. The tutor agent chooses what topic to look up, while your application controls which course material it can access.

## The model loop

Helping with a lesson may take a few attempts to find and use the right material. Give the tutor agent a loop so it can choose a tool, read the result and decide what to do next:

```ts
import { stepCountIs } from 'ai';

const result = await generateText({
  model, system, messages,
  tools: courseTools,
  stopWhen: stepCountIs(8),
  maxOutputTokens: 1_500,
  abortSignal: signal,
});

if (result.finishReason !== 'stop') {
  throw new Error(`Tutor agent stopped before completion: ${result.finishReason}`);
}
```

The tutor agent can keep using tools within the eight-call limit set by `stepCountIs(8)`. The loop ends sooner if it finishes its response or you cancel through `abortSignal`.

## Web search

The student might understand the lesson but ask, “Where would I use this?” Add OpenAI's [web search tool](https://ai-sdk.dev/providers/ai-sdk-providers/openai#web-search-tool) to help the tutor agent find an example beyond the course:

```ts
const tools = {
  ...courseTools,
  web_search: openai.tools.webSearch({}),
};
```

Use `tools` in the loop to make both lookups available. OpenAI runs search for you, with no local `execute` function, and returns source URLs to cite. Pages can also contain instructions aimed at the agent, so your application still needs to enforce access checks.

## Sub-agents as tools

If finding an example takes several searches, you don't need all those results in the tutor agent's conversation. Give that job to a [sub-agent](https://ai-sdk.dev/docs/agents/subagents), which returns a short explanation and sources through a tool:

```ts
const research = tool({
  description: 'Find a sourced example to help explain a lesson',
  inputSchema: z.object({ question: z.string() }),
  execute: async ({ question }, { abortSignal }) => {
    const result = await generateText({
      model,
      system: 'Find one relevant example. Return a paragraph and URLs.',
      prompt: question,
      tools: { web_search: tools.web_search },
      stopWhen: stepCountIs(3),
      maxOutputTokens: 800,
      abortSignal,
    });
    if (result.finishReason !== 'stop') throw new Error('Research incomplete');
    return { text: result.text, sources: result.sources };
  },
});

const tutorTools = { ...tools, research };
```

The student is still making one request, even when research runs in another conversation. Use the same `abortSignal` and count both agents' calls towards one budget, so cancellation and spending limits cover the whole task.

## Messages and session state

Alongside the explanation, the student can see which lessons and sources the tutor agent used. The SDK's [`UIMessage`](https://ai-sdk.dev/docs/reference/ai-sdk-core/ui-message) represents a message as it appears in the interface, with an ordered `parts` array for text, sources and tools still running.

You can save these UI messages and show them in the browser. When the student asks a follow-up, [`convertToModelMessages`](https://ai-sdk.dev/docs/reference/ai-sdk-ui/convert-to-model-messages) turns that conversation into `ModelMessage[]`, ready for the next model call:

```ts
import type { InferUITools, UIMessage } from 'ai';

type TutorMessage = UIMessage<
  unknown,
  never,
  InferUITools<typeof tutorTools>
>;

type Session = {
  id: string;
  messages: TutorMessage[];
  status: 'idle' | 'running' | 'waiting' | 'completed' | 'cancelled' | 'error';
  error?: string;
};
```

## Streaming and saving progress

The student shouldn't have to wait for every tool call before seeing anything. Switch to [`streamText`](https://ai-sdk.dev/docs/reference/ai-sdk-core/stream-text) to show progress, then use [`readUIMessageStream`](https://ai-sdk.dev/docs/reference/ai-sdk-ui/read-ui-message-stream) to assemble text and tool updates into `UIMessage` snapshots you can save.

Start with the student's new question in the saved conversation and mark the session `running`. The `store` helpers below save each update:

```ts
import {
  convertToModelMessages, readUIMessageStream, streamText,
} from 'ai';

const result = streamText({
  model, system,
  messages: await convertToModelMessages(session.messages, { tools: tutorTools }),
  tools: tutorTools,
  stopWhen: stepCountIs(8),
  maxOutputTokens: 1_500,
  abortSignal: signal,
});

const uiStream = result.toUIMessageStream({ sendSources: true });
const updates = readUIMessageStream<TutorMessage>({
  stream: uiStream,
  terminateOnError: true,
});

for await (const message of updates) {
  await store.saveMessage(session.id, message);
}

signal.throwIfAborted();
if (await result.finishReason !== 'stop') {
  throw new Error('Tutor agent stopped before completion');
}
await store.setStatus(session.id, 'completed');
```

Saving a partial answer lets the student return to what they've already read if something interrupts the request. For longer answers, batch these saves to reduce database writes, and flush the last batch before marking the request complete.

For actions the student should review first, `needsApproval` lets you ask permission before a tool runs.

## Keeping the connection separate

A dropped connection shouldn't make the student start over. Run the tutor agent independently of the browser connection and let the browser subscribe to saved progress through [server-sent events](https://html.spec.whatwg.org/multipage/server-sent-events.html) (SSE). Reconnecting then catches up with the existing work.

Sending full snapshots is simple but repeats the conversation on every update. For longer chats, send [deltas](https://ai-sdk.dev/docs/ai-sdk-ui/stream-protocol#text-parts) containing only new content. Save those events with IDs so a reconnect can [replay what was missed](https://html.spec.whatwg.org/multipage/server-sent-events.html#the-last-event-id-header). Your endpoint still checks session ownership before sending updates.

## Summary

For the student, these pieces come together as a conversation they can follow and return to. The tutor agent can consult their course, research an example and help them work through the lesson. The harness keeps that work going and saves the conversation, ready for their next question.

This example is deliberately small. To make it ready for students, you'd add a fuller toolset, a chat UI and persistent storage for conversations and progress. You'd also need authentication, permission controls, spending limits and recovery for interrupted work. An evaluation process would check the tutor agent's accuracy and teaching quality using representative student questions.
