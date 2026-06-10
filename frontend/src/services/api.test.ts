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
  regenerateMessage,
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
