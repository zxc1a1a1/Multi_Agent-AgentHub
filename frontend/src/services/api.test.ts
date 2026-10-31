import { beforeEach, describe, expect, it, vi } from 'vitest'

const mockFetch = vi.fn()
vi.stubGlobal('fetch', mockFetch)

function mockResponse(status: number, body: unknown) {
  return {
    ok: status >= 200 && status < 300,
    status,
    json: () => Promise.resolve(body),
  }
}

// We test the interface shape and error handling by extracting the internal
// request-building logic. Import the actual module to verify types.
import {
  createConversation,
  listConversationParticipants,
  regenerateMessage,
  updateConversationParticipants,
  type RegenerateRequest,
  type RegenerateResponse,
} from './api'

beforeEach(() => {
  vi.clearAllMocks()
})

describe('regenerateMessage', () => {
  const validRequest: RegenerateRequest = {
    runId: 'run-1',
    conversationId: 'conv-1',
    messageId: 'msg-3',
    pinnedMessageIds: ['msg-1'],
    context: [
      { id: 'msg-1', role: 'user', text: 'first question' },
      { id: 'msg-2', role: 'assistant', text: 'first answer' },
      { id: 'msg-3', role: 'assistant', text: 'bad answer to regenerate' },
    ],
  }

  it('includes context in request payload', async () => {
    const responseBody: RegenerateResponse = {
      messageId: 'msg-3',
      status: 'ready',
      preserveOriginal: true,
      pinnedMessageIds: ['msg-1'],
      context: [
        { id: 'msg-1', role: 'user', text: 'first question' },
        { id: 'msg-2', role: 'assistant', text: 'first answer' },
      ],
    }
    mockFetch.mockResolvedValueOnce(mockResponse(200, responseBody))

    const result = await regenerateMessage(validRequest)

    // Verify fetch was called with context in body
    expect(mockFetch).toHaveBeenCalledTimes(1)
    const callArgs = mockFetch.mock.calls[0] as [string, RequestInit]
    const body = JSON.parse(callArgs[1].body as string)
    expect(body.context).toBeDefined()
    expect(body.context.length).toBe(3)
    expect(body.context[0]).toEqual({ id: 'msg-1', role: 'user', text: 'first question' })
    expect(body.messageId).toBe('msg-3')
    expect(result.status).toBe('ready')
    expect(result.context).toBeDefined()
    expect(result.context!.length).toBe(2)
  })

  it('surfaces MESSAGE_NOT_FOUND error', async () => {
    mockFetch.mockResolvedValueOnce(
      mockResponse(404, { code: 'MESSAGE_NOT_FOUND', error: 'not found' }),
    )

    await expect(regenerateMessage(validRequest)).rejects.toThrow('MESSAGE_NOT_FOUND')
  })

  it('surfaces INVALID_TARGET_ROLE error', async () => {
    mockFetch.mockResolvedValueOnce(
      mockResponse(400, { code: 'INVALID_TARGET_ROLE', error: 'bad role' }),
    )

    await expect(regenerateMessage(validRequest)).rejects.toThrow('INVALID_TARGET_ROLE')
  })

  it('surfaces generic error for unknown status codes', async () => {
    mockFetch.mockResolvedValueOnce(mockResponse(500, {}))

    await expect(regenerateMessage(validRequest)).rejects.toThrow('Failed to regenerate message')
  })

  it('sends context even without pinnedMessageIds', async () => {
    const requestWithoutPins: RegenerateRequest = {
      conversationId: 'conv-1',
      messageId: 'msg-2',
      context: [
        { id: 'msg-1', role: 'user', text: 'hello' },
        { id: 'msg-2', role: 'assistant', text: 'world' },
      ],
    }
    mockFetch.mockResolvedValueOnce(
      mockResponse(200, { messageId: 'msg-2', status: 'ready', preserveOriginal: true }),
    )

    await regenerateMessage(requestWithoutPins)

    const callArgs = mockFetch.mock.calls[0] as [string, RequestInit]
    const body = JSON.parse(callArgs[1].body as string)
    expect(body.context).toBeDefined()
    expect(body.context.length).toBe(2)
  })

  it('auto-generates runId when not provided', async () => {
    const request: RegenerateRequest = {
      conversationId: 'conv-1',
      messageId: 'msg-2',
      context: [{ id: 'msg-1', role: 'user', text: 'hello' }],
    }
    mockFetch.mockResolvedValueOnce(
      mockResponse(200, { messageId: 'msg-2', status: 'ready', preserveOriginal: true }),
    )

    await regenerateMessage(request)

    const callArgs = mockFetch.mock.calls[0] as [string, RequestInit]
    const body = JSON.parse(callArgs[1].body as string)
    expect(body.runId).toMatch(/^regen-msg-2$/)
  })
})

describe('conversation participants API', () => {
  it('creates a group conversation with the full roster', async () => {
    mockFetch.mockResolvedValueOnce(
      mockResponse(201, {
        id: 'conv-group',
        agentName: 'code-agent',
        type: 'group',
        participants: [
          { agentName: 'code-agent', role: 'owner', selected: true },
          { agentName: 'review-agent', role: 'member', selected: true },
        ],
      }),
    )

    const conv = await createConversation('code-agent', ['code-agent', 'review-agent'])

    const callArgs = mockFetch.mock.calls[0] as [string, RequestInit]
    const body = JSON.parse(callArgs[1].body as string)
    expect(body.agentNames).toEqual(['code-agent', 'review-agent'])
    expect(body.type).toBe('group')
    expect(conv.type).toBe('group')
    expect(conv.participants).toHaveLength(2)
    expect(conv.participants?.[0]).toMatchObject({ agentName: 'code-agent', role: 'owner' })
  })

  it('lists participants from the roster endpoint', async () => {
    mockFetch.mockResolvedValueOnce(
      mockResponse(200, [
        { agentName: 'code-agent', role: 'owner', selected: true },
        { agentName: 'review-agent', role: 'member', selected: false },
      ]),
    )

    const participants = await listConversationParticipants('conv-1')

    const callArgs = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(callArgs[0]).toBe('/api/conversations/conv-1/participants')
    expect(participants).toHaveLength(2)
    expect(participants[1]).toMatchObject({ agentName: 'review-agent', selected: false })
  })

  it('updates participants with PATCH and rejects on failure', async () => {
    mockFetch.mockResolvedValueOnce(
      mockResponse(200, {
        id: 'conv-1',
        agentName: 'test-agent',
        type: 'single',
        participants: [{ agentName: 'test-agent', role: 'owner', selected: true }],
      }),
    )

    const conv = await updateConversationParticipants('conv-1', ['test-agent'])
    const callArgs = mockFetch.mock.calls[0] as [string, RequestInit]
    expect(callArgs[1].method).toBe('PATCH')
    expect(JSON.parse(callArgs[1].body as string)).toEqual({
      agentNames: ['test-agent'],
      type: 'single',
    })
    expect(conv.participants?.[0]?.agentName).toBe('test-agent')

    mockFetch.mockResolvedValueOnce(mockResponse(500, {}))
    await expect(updateConversationParticipants('conv-1', ['code-agent'])).rejects.toThrow(
      'Failed to update conversation participants',
    )
  })
})
