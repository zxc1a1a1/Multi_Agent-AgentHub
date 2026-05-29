import { afterEach, describe, expect, it, vi } from 'vitest'
import { runAgent, type AGUIChatRequest } from './client'
import type { AGUIEvent } from '../types'

function createStream(chunks: string[]): ReadableStream<Uint8Array> {
  const encoder = new TextEncoder()
  return new ReadableStream<Uint8Array>({
    start(controller) {
      for (const chunk of chunks) {
        controller.enqueue(encoder.encode(chunk))
      }
      controller.close()
    },
  })
}

function createResponseFromChunks(chunks: string[]): Response {
  return new Response(createStream(chunks), {
    status: 200,
    headers: { 'Content-Type': 'text/event-stream' },
  })
}

const request: AGUIChatRequest = {
  conversationId: 'conv-1',
  message: 'hello',
}

async function runAndCollect(chunks: string[]): Promise<{
  events: AGUIEvent[]
  errors: Error[]
  completed: boolean
}> {
  const events: AGUIEvent[] = []
  const errors: Error[] = []
  let completed = false

  vi.stubGlobal(
    'fetch',
    vi.fn().mockResolvedValue(createResponseFromChunks(chunks)),
  )

  await new Promise<void>((resolve, reject) => {
    const timer = setTimeout(() => {
      reject(new Error('Timed out waiting stream completion'))
    }, 1000)

    runAgent(
      request,
      (event) => {
        events.push(event)
      },
      (err) => {
        errors.push(err)
        clearTimeout(timer)
        resolve()
      },
      () => {
        completed = true
        clearTimeout(timer)
        resolve()
      },
    )
  })

  return { events, errors, completed }
}

describe('runAgent SSE parsing', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    vi.unstubAllGlobals()
  })

  it('supports multiple SSE blocks in one chunk', async () => {
    const { events, errors, completed } = await runAndCollect([
      [
        'data: {"type":"RUN_STARTED","runId":"run-1"}\n\n',
        'data: {"type":"TEXT_MESSAGE_CONTENT","content":"A"}\n\n',
        'data: {"type":"RUN_FINISHED","runId":"run-1"}\n\n',
      ].join(''),
    ])

    expect(events.map((e) => e.type)).toEqual([
      'RUN_STARTED',
      'TEXT_MESSAGE_CONTENT',
      'RUN_FINISHED',
    ])
    expect(errors).toHaveLength(0)
    expect(completed).toBe(true)
  })

  it('falls back to SSE event name when payload has no type', async () => {
    const { events } = await runAndCollect([
      'event: message.delta\ndata: {"text":"A"}\n\n',
    ])

    expect(events).toHaveLength(1)
    expect(events[0]).toMatchObject({
      type: 'message.delta',
      text: 'A',
    })
  })

  it('supports data line split across chunks with tail buffer', async () => {
    const { events, completed } = await runAndCollect([
      'data: {"type":"TEXT_MESSAGE_CONTENT","content":"hel',
      'lo"}\n\n',
    ])

    expect(events).toHaveLength(1)
    expect(events[0]).toMatchObject({
      type: 'TEXT_MESSAGE_CONTENT',
      content: 'hello',
    })
    expect(completed).toBe(true)
  })

  it('supports multi-line data inside one block', async () => {
    const { events } = await runAndCollect([
      [
        'data: {"type":"TEXT_MESSAGE_CONTENT",',
        'data: "content":"line1\\nline2"}',
        '\n\n',
      ].join('\n'),
    ])

    expect(events).toHaveLength(1)
    expect(events[0]).toMatchObject({
      type: 'TEXT_MESSAGE_CONTENT',
      content: 'line1\nline2',
    })
  })

  it('skips empty data blocks and continues stream', async () => {
    const { events } = await runAndCollect([
      'data:\n\n',
      'data:   \n\n',
      'data: {"type":"RUN_FINISHED","runId":"run-1"}\n\n',
    ])

    expect(events).toHaveLength(1)
    expect(events[0]).toMatchObject({
      type: 'RUN_FINISHED',
      runId: 'run-1',
    })
  })

  it('malformed JSON does not break whole stream', async () => {
    const { events, errors } = await runAndCollect([
      'data: {"type":"TEXT_MESSAGE_CONTENT","content":"ok"}\n\n',
      'data: {"type":"BROKEN"\n\n',
      'data: {"type":"RUN_FINISHED","runId":"run-1"}\n\n',
    ])

    expect(events.map((e) => e.type)).toEqual([
      'TEXT_MESSAGE_CONTENT',
      'RUN_FINISHED',
    ])
    expect(errors).toHaveLength(0)
  })
})
