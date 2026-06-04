import type { Page } from '@playwright/test'

interface AGUIEvent {
  type: string
  messageId?: string
  runId?: string
  content?: string
  delta?: string
  text?: string
  toolCallId?: string
  toolName?: string
  senderName?: string
  agentName?: string
  sender?: {
    type?: string
    name?: string
    displayName?: string
  }
  error?: string | { code?: string; message?: string }
}

function buildSSEBody(events: AGUIEvent[]): string {
  return events.map((event) => `data: ${JSON.stringify(event)}\n\n`).join('')
}

const CONVERSATION = {
  id: 'conv-1',
  title: 'New Conversation',
  agentName: 'code-agent',
  createdAt: '2026-01-01T00:00:00Z',
  updatedAt: '2026-01-01T00:00:01Z',
}

const HISTORY_MESSAGES = [
  {
    id: 'msg-1',
    conversationId: 'conv-1',
    senderType: 'user',
    senderName: '',
    content: 'Write a Go hello world',
    artifacts: null,
    aguiRunId: null,
    createdAt: '2026-01-01T00:00:00Z',
  },
  {
    id: 'msg-2',
    conversationId: 'conv-1',
    senderType: 'agent',
    senderName: 'Code Agent',
    content: 'Here is your Go hello world program:',
    artifacts: JSON.stringify([
      {
        type: 'code',
        title: 'main.go',
        content: 'package main\n\nimport "fmt"\n\nfunc main() {\n\tfmt.Println("hello world")\n}',
        metadata: { language: 'go', filename: 'main.go' },
      },
    ]),
    aguiRunId: 'run-1',
    createdAt: '2026-01-01T00:00:01Z',
  },
]

const AGENTS = [
  { name: 'code-agent', description: 'Generates and explains code' },
  { name: 'web-agent', description: 'Generates webpages and HTML previews' },
]

export function buildCodePreviewSSE(): string {
  const msgId = 'msg-agent-code-1'
  const toolCallId = 'tc-code-1'

  return buildSSEBody([
    { type: 'RUN_STARTED', runId: 'run-code-1' },
    {
      type: 'TEXT_MESSAGE_START',
      messageId: msgId,
      sender: { type: 'agent', name: 'code-agent', displayName: 'Code Agent' },
    },
    { type: 'TEXT_MESSAGE_CONTENT', messageId: msgId, delta: 'Here is your ' },
    { type: 'TEXT_MESSAGE_CONTENT', messageId: msgId, delta: 'Go hello world program:' },
    { type: 'TEXT_MESSAGE_END', messageId: msgId },
    {
      type: 'TOOL_CALL_START',
      toolCallId,
      toolName: 'code_preview',
      messageId: msgId,
    },
    {
      type: 'TOOL_CALL_ARGS',
      toolCallId,
      content: JSON.stringify({
        code: 'package main\n\nimport "fmt"\n\nfunc main() {\n\tfmt.Println("hello world")\n}',
        language: 'go',
        filename: 'main.go',
      }),
    },
    { type: 'TOOL_CALL_END', toolCallId, toolName: 'code_preview' },
    { type: 'RUN_FINISHED' },
  ])
}

export function buildWebPreviewSSE(): string {
  const msgId = 'msg-agent-web-1'
  const toolCallId = 'tc-web-1'

  return buildSSEBody([
    { type: 'RUN_STARTED', runId: 'run-web-1' },
    {
      type: 'TEXT_MESSAGE_START',
      messageId: msgId,
      sender: { type: 'agent', name: 'web-agent', displayName: 'Web Agent' },
    },
    {
      type: 'TEXT_MESSAGE_CONTENT',
      messageId: msgId,
      delta: '<section><h1>Demo Page</h1><p>safe html preview</p></section>',
    },
    { type: 'TEXT_MESSAGE_END', messageId: msgId },
    {
      type: 'TOOL_CALL_START',
      toolCallId,
      toolName: 'web_preview',
      messageId: msgId,
    },
    {
      type: 'TOOL_CALL_ARGS',
      toolCallId,
      content: JSON.stringify({
        title: 'demo.html',
        html: '<section><h1>Demo Page</h1><p>safe html preview</p></section>',
      }),
    },
    { type: 'TOOL_CALL_END', toolCallId, toolName: 'web_preview' },
    { type: 'RUN_FINISHED' },
  ])
}

export function buildMixedOrderedParallelSSE(): string {
  const webMsgId = 'msg-mixed-web'
  const codeMsgId = 'msg-mixed-code'
  const summaryMsgId = 'msg-mixed-summary'

  return buildSSEBody([
    { type: 'RUN_STARTED', runId: 'run-mixed-1' },

    // Task 1: Web Agent
    {
      type: 'TEXT_MESSAGE_START',
      messageId: webMsgId,
      runId: 'run-mixed-1',
      sender: { type: 'agent', name: 'web-agent', displayName: 'Web Agent' },
    },
    {
      type: 'TEXT_MESSAGE_CONTENT',
      messageId: webMsgId,
      delta: '<section><h1>Login Page</h1><form><input placeholder="email"/></form></section>',
    },
    { type: 'TEXT_MESSAGE_END', messageId: webMsgId },

    // Task 2: Code Agent — different messageId!
    {
      type: 'TEXT_MESSAGE_START',
      messageId: codeMsgId,
      runId: 'run-mixed-1',
      sender: { type: 'agent', name: 'code-agent', displayName: 'Code Agent' },
    },
    {
      type: 'TEXT_MESSAGE_CONTENT',
      messageId: codeMsgId,
      delta: 'package main\n\nimport (\n\t"net/http"\n\t"log"\n)\n\nfunc main() {\n\thttp.HandleFunc("/login", loginHandler)\n\tlog.Fatal(http.ListenAndServe(":8080", nil))\n}',
    },
    { type: 'TEXT_MESSAGE_END', messageId: codeMsgId },

    // Task 3: Orchestrator summary — different messageId!
    {
      type: 'TEXT_MESSAGE_START',
      messageId: summaryMsgId,
      runId: 'run-mixed-1',
      sender: { type: 'orchestrator', name: 'orchestrator', displayName: 'Orchestrator' },
    },
    {
      type: 'TEXT_MESSAGE_CONTENT',
      messageId: summaryMsgId,
      delta: 'All 2 task(s) completed successfully. Web Agent built the login page. Code Agent created the Go API server.',
    },
    { type: 'TEXT_MESSAGE_END', messageId: summaryMsgId },

    { type: 'RUN_FINISHED', runId: 'run-mixed-1' },
  ])
}

export function buildErrorSSE(): string {
  return buildSSEBody([
    { type: 'RUN_STARTED', runId: 'run-error-1' },
    { type: 'RUN_ERROR', runId: 'run-error-1', error: { code: 'AGUI_INTERNAL', message: 'assistant run failed' } },
  ])
}

function buildUnknownAgentErrorSSE(): string {
  return buildSSEBody([
    { type: 'RUN_STARTED', runId: 'run-error-unknown' },
    { type: 'RUN_ERROR', runId: 'run-error-unknown', error: { code: 'UNKNOWN_AGENT', message: 'unknown agent' } },
  ])
}

function buildChatSSE(agentName: string | undefined): string {
  if (!agentName || agentName === 'code-agent') {
    return buildCodePreviewSSE()
  }
  if (agentName === 'web-agent') {
    return buildWebPreviewSSE()
  }
  return buildUnknownAgentErrorSSE()
}

/**
 * Install API mocks on the page so E2E tests run without a real backend.
 * All routes use `page.route()` to intercept and respond with fixture data.
 */
export async function setupMocks(page: Page) {
  await page.route('**/api/conversations', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({ json: [CONVERSATION] })
      return
    }

    if (route.request().method() === 'POST') {
      const data = route.request().postDataJSON() as { agentName?: string; title?: string }
      const agentName = data.agentName === 'web-agent' ? 'web-agent' : 'code-agent'
      await route.fulfill({
        json: {
          ...CONVERSATION,
          title: data.title || 'New Conversation',
          agentName,
        },
      })
      return
    }

    await route.continue()
  })

  await page.route('**/api/conversations/*/messages', async (route) => {
    await route.fulfill({ json: HISTORY_MESSAGES })
  })

  await page.route('**/api/agents', async (route) => {
    await route.fulfill({ json: AGENTS })
  })

  await page.route('**/api/chat', async (route) => {
    const data = route.request().postDataJSON() as { agentName?: string }
    await route.fulfill({
      status: 200,
      headers: { 'Content-Type': 'text/event-stream' },
      body: buildChatSSE(data.agentName),
    })
  })
}
