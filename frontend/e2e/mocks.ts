import type { Page } from '@playwright/test'

interface AGUIEvent {
  type: string
  messageId?: string
  runId?: string
  content?: string
  toolCallId?: string
  toolName?: string
  error?: string
}

function buildSSEBody(events: AGUIEvent[]): string {
  return events.map((e) => `data: ${JSON.stringify(e)}\n\n`).join('')
}

// ── Mock data ──────────────────────────────────────────────

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
    senderName: 'code-agent',
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

const AGENTS = [{ name: 'code-agent', description: 'Generates and explains code' }]

// ── SSE stream builders ────────────────────────────────────

export function buildCodePreviewSSE(): string {
  const msgId = 'msg-agent-1'
  const tcId = 'tc-1'

  return buildSSEBody([
    { type: 'RUN_STARTED', runId: 'run-1' },
    { type: 'TEXT_MESSAGE_START', messageId: msgId },
    { type: 'TEXT_MESSAGE_CONTENT', messageId: msgId, content: 'Here is your ' },
    { type: 'TEXT_MESSAGE_CONTENT', messageId: msgId, content: 'Go hello world program:' },
    { type: 'TEXT_MESSAGE_END', messageId: msgId },
    {
      type: 'TOOL_CALL_START',
      toolCallId: tcId,
      toolName: 'code_preview',
      messageId: msgId,
    },
    {
      type: 'TOOL_CALL_ARGS',
      toolCallId: tcId,
      content: JSON.stringify({
        code: 'package main\n\nimport "fmt"\n\nfunc main() {\n\tfmt.Println("hello world")\n}',
        language: 'go',
        filename: 'main.go',
      }),
    },
    { type: 'TOOL_CALL_END', toolCallId: tcId },
    { type: 'RUN_FINISHED' },
  ])
}

export function buildTextOnlySSE(): string {
  const msgId = 'msg-agent-1'
  return buildSSEBody([
    { type: 'RUN_STARTED', runId: 'run-2' },
    { type: 'TEXT_MESSAGE_START', messageId: msgId },
    { type: 'TEXT_MESSAGE_CONTENT', messageId: msgId, content: 'Hello! How can I help?' },
    { type: 'TEXT_MESSAGE_END', messageId: msgId },
    { type: 'RUN_FINISHED' },
  ])
}

export function buildErrorSSE(): string {
  return buildSSEBody([
    { type: 'RUN_STARTED', runId: 'run-3' },
    { type: 'RUN_ERROR', error: 'Agent task failed' },
  ])
}

// ── Route setup ────────────────────────────────────────────

/**
 * Install API mocks on the page so E2E tests run without a real backend.
 * All routes use `page.route()` to intercept and respond with fixture data.
 */
export async function setupMocks(page: Page) {
  // Conversations list + create
  await page.route('**/api/conversations', async (route) => {
    if (route.request().method() === 'GET') {
      await route.fulfill({ json: [CONVERSATION] })
    } else if (route.request().method() === 'POST') {
      await route.fulfill({ json: CONVERSATION })
    } else {
      await route.continue()
    }
  })

  // Messages for a conversation
  await page.route('**/api/conversations/*/messages', async (route) => {
    await route.fulfill({ json: HISTORY_MESSAGES })
  })

  // Agents list
  await page.route('**/api/agents', async (route) => {
    await route.fulfill({ json: AGENTS })
  })

  // AG-UI SSE stream — default to code_preview response
  await page.route('**/api/agui/run', async (route) => {
    await route.fulfill({
      status: 200,
      headers: { 'Content-Type': 'text/event-stream' },
      body: buildCodePreviewSSE(),
    })
  })
}
