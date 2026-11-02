import { test, expect, type Page, type Route } from '@playwright/test'

interface Participant {
  agentName: string
  role?: string
  selected?: boolean
  required?: boolean
}

interface ChatState {
  conversations: Array<Record<string, unknown>>
  participants: Participant[]
  messages: Array<Record<string, unknown>>
  chatBodies: Array<Record<string, unknown>>
  confirmBodies: Array<Record<string, unknown>>
  participantPatches: Array<Record<string, unknown>>
}

const conversationId = 'conv-e2e'
const runId = 'run-e2e'
const planId = 'plan-e2e'

const agents = [
  { name: 'auto', displayName: 'Auto (Smart)', description: 'Auto route', outputModes: ['text'], source: 'static', enabled: true, healthy: true },
  { name: 'code-agent', displayName: 'Code Agent', description: 'Writes code', outputModes: ['text'], source: 'static', enabled: true, healthy: true },
  { name: 'review-agent', displayName: 'Review Agent', description: 'Reviews code', outputModes: ['text'], source: 'static', enabled: true, healthy: true },
]

function json(route: Route, status: number, body: unknown) {
  return route.fulfill({
    status,
    contentType: 'application/json',
    body: JSON.stringify(body),
  })
}

function conversation(state: ChatState) {
  return {
    id: conversationId,
    title: 'Group Test',
    agentName: state.participants[0]?.agentName || 'code-agent',
    type: state.participants.length > 1 ? 'group' : 'single',
    participants: state.participants,
    createdAt: '2026-11-02T00:00:00Z',
    updatedAt: '2026-11-02T00:00:00Z',
  }
}

async function installApiMocks(page: Page, state: ChatState) {
  await page.route('**/api/**', async (route) => {
    const request = route.request()
    const url = new URL(request.url())
    const path = url.pathname
    const method = request.method()

    if (path === '/api/agents') {
      return json(route, 200, agents)
    }
    if (path === '/api/conversations' && method === 'GET') {
      return json(route, 200, state.conversations)
    }
    if (path === '/api/conversations' && method === 'POST') {
      const body = JSON.parse(request.postData() || '{}')
      state.participants = []
      const conv = conversation(state)
      state.conversations = [conv]
      return json(route, 201, { ...conv, agentName: body.agentName || 'auto' })
    }
    if (path === `/api/conversations/${conversationId}/participants`) {
      if (method === 'GET') return json(route, 200, state.participants)
      if (method === 'PATCH') {
        const body = JSON.parse(request.postData() || '{}')
        state.participantPatches.push(body)
        const names: string[] = Array.isArray(body.agentNames) ? body.agentNames : []
        state.participants = names.map((name, index) => ({
          agentName: name,
          role: index === 0 ? 'owner' : 'member',
          selected: true,
        }))
        return json(route, 200, conversation(state))
      }
    }
    if (path === `/api/conversations/${conversationId}/messages` && method === 'GET') {
      return json(route, 200, state.messages)
    }
    if (path === '/api/chat' && method === 'POST') {
      const body = JSON.parse(request.postData() || '{}')
      state.chatBodies.push(body)
      state.messages = [
        {
          id: 'msg-user',
          conversationId,
          senderType: 'user',
          senderName: 'user',
          content: body.message,
          createdAt: '2026-11-02T00:00:01Z',
        },
        {
          id: 'msg-code',
          conversationId,
          senderType: 'agent',
          senderName: 'code-agent',
          content: 'code answer',
          runId,
          createdAt: '2026-11-02T00:00:02Z',
        },
        {
          id: 'msg-review',
          conversationId,
          senderType: 'agent',
          senderName: 'review-agent',
          content: 'review ok',
          runId,
          createdAt: '2026-11-02T00:00:03Z',
        },
      ]
      const activity = {
        activityId: planId,
        activityType: 'plan_approval',
        status: 'awaiting_confirmation',
        executionPath: 'group_chat',
        planId,
        revision: 1,
        planOwner: { type: 'group', agentName: 'code-agent' },
        participants: [
          { agentName: 'code-agent', required: true, selected: true },
          { agentName: 'review-agent', selected: true },
        ],
        candidateParticipants: [],
        defaultSelectedParticipants: [
          { agentName: 'code-agent', selected: true },
          { agentName: 'review-agent', selected: true },
        ],
        requiredParticipants: ['code-agent'],
        title: 'Group plan',
        summary: 'Run two agents',
        tasks: [
          { taskId: 't1', agentName: 'code-agent', content: 'write code' },
          { taskId: 't2', agentName: 'review-agent', content: 'review code' },
        ],
        allowedActions: ['approve', 'cancel', 'revise'],
        warnings: [],
      }
      const sse = [
        `data: ${JSON.stringify({ type: 'RUN_STARTED', runId, seq: 1 })}\n\n`,
        `data: ${JSON.stringify({ type: 'ACTIVITY_SNAPSHOT', runId, seq: 2, activity })}\n\n`,
        `data: ${JSON.stringify({ type: 'RUN_FINISHED', runId, seq: 3, state: { status: 'awaiting_confirmation' } })}\n\n`,
      ].join('')
      return route.fulfill({
        status: 200,
        contentType: 'text/event-stream',
        body: sse,
      })
    }
    if (path === `/api/runs/${runId}/confirm` && method === 'POST') {
      state.confirmBodies.push(JSON.parse(request.postData() || '{}'))
      return json(route, 200, { status: 'acknowledged' })
    }
    if (path === `/api/runs/${runId}/status`) {
      return json(route, 200, { status: 'cancelled' })
    }
    return json(route, 404, { error: `unhandled ${method} ${path}` })
  })
}

test('group chat create → send → cancel → refresh keeps participants and history', async ({ page }) => {
  const state: ChatState = {
    conversations: [],
    participants: [],
    messages: [],
    chatBodies: [],
    confirmBodies: [],
    participantPatches: [],
  }
  await installApiMocks(page, state)
  await page.goto('/')

  await page.getByTitle('New conversation').click()
  await expect(page.getByPlaceholder('Type a message... Use @agent-name to mention an agent. Enter to send.')).toBeVisible()

  // Select two agents in the picker.
  await page.getByTestId('agent-picker-trigger').click()
  await page.getByTestId('agent-option-code-agent').click()
  await page.getByTestId('agent-option-review-agent').click()
  await expect(page.getByText('2 agents selected')).toBeVisible()
  expect(state.participantPatches.some((body) => Array.isArray(body.agentNames) && body.agentNames.length === 2)).toBeTruthy()

  // Send a message; the mocked SSE raises a group plan awaiting confirmation.
  const input = page.getByPlaceholder('Type a message... Use @agent-name to mention an agent. Enter to send.')
  await input.fill('hello group')
  await input.press('Enter')
  await expect(page.getByRole('heading', { name: 'Group plan' })).toBeVisible()
  expect(state.chatBodies[0]?.selectedAgentNames).toEqual(
    expect.arrayContaining(['code-agent', 'review-agent']),
  )

  // Cancel the plan from the group card.
  await page.getByRole('button', { name: 'Cancel' }).evaluate((element) => (element as HTMLElement).click())
  await expect.poll(() => state.confirmBodies.length).toBeGreaterThan(0)
  expect(state.confirmBodies[0]?.action).toBe('cancel')

  // Refresh: participants and per-agent history are restored.
  await page.reload()
  await expect(page.getByText('2 agents selected')).toBeVisible()
  await expect(page.getByText('code answer')).toBeVisible()
  await expect(page.getByText('review ok')).toBeVisible()
})
