import type { AGUIEvent, ReplyTo, Quote } from '../types'
import type { AgentName } from '../lib/agents'

export interface AGUIChatRequest {
  conversationId: string
  message: string
  agentName?: AgentName
  selectedAgentNames?: string[]
  mentions?: string[]
  requestedPath?: string
  replyTo?: ReplyTo
  quote?: Quote
  pinnedMessageIds?: string[]
  contextMessages?: Array<{ id?: string; role: string; text: string }>
}

function authHeaders(): Record<string, string> {
  const t = import.meta.env.VITE_AGENTHUB_API_TOKEN as string
  const h: Record<string, string> = { 'Content-Type': 'application/json' }
  if (t) h['Authorization'] = `Bearer ${t}`
  return h
}

/**
 * Sends a chat request to Gateway /api/chat and streams SSE events back.
 * Uses fetch + ReadableStream for POST-based SSE (EventSource only supports GET).
 */
export function runAgent(
  request: AGUIChatRequest,
  onEvent: (event: AGUIEvent) => void,
  onError?: (err: Error) => void,
  onComplete?: () => void,
): AbortController {
  const controller = new AbortController()
  let lastRunId = ''
  let lastSeq = 0
  let lastSubIndex = 0

  fetch('/api/chat', {
    method: 'POST',
    headers: authHeaders(),
    body: JSON.stringify(request),
    signal: controller.signal,
  })
    .then(async (response) => {
      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`)
      }
      const reader = response.body?.getReader()
      if (!reader) throw new Error('No response body')

      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) {
          buffer += decoder.decode()
          break
        }

        // Decode binary chunk and append to buffer
        buffer += decoder.decode(value, { stream: true })
        buffer = buffer.replace(/\r\n/g, '\n')

        const parsed = splitSSEBlocks(buffer)
        buffer = parsed.rest

        for (const block of parsed.blocks) {
          const parsedBlock = parseSSEBlock(block)
          if (!parsedBlock.data) continue
          try {
            const event: AGUIEvent = JSON.parse(parsedBlock.data)
            // If JSON has no type, fall back to SSE event name.
            if (!event.type && parsedBlock.eventName) {
              event.type = parsedBlock.eventName
            }
            // Normalize legacy SSE event names to AG-UI types for backward compat.
            if (!event.type) {
              // empty event — skip
              continue
            }
            if (event.runId && event.runId !== lastRunId) {
              lastRunId = event.runId
              lastSeq = 0
              lastSubIndex = 0
            }
            if (typeof event.seq === 'number') {
              const subIndex = typeof event.subIndex === 'number' ? event.subIndex : 0
              if (event.seq > lastSeq) {
                lastSeq = event.seq
                lastSubIndex = subIndex
              } else if (event.seq === lastSeq && subIndex > lastSubIndex) {
                lastSubIndex = subIndex
              }
            }
            onEvent(event)
          } catch {
            // skip malformed JSON and continue streaming
          }
        }
      }

      onComplete?.()
    })
    .catch(async (err) => {
      if (err.name === 'AbortError') return
      if (lastRunId) {
        try {
          await recoverRun(lastRunId, lastSeq, lastSubIndex, onEvent, onComplete)
          return
        } catch {
          // Fall through to the original error.
        }
      }
      onError?.(err)
    })

  return controller
}

function splitSSEBlocks(input: string): { blocks: string[]; rest: string } {
  const blocks: string[] = []
  let offset = 0

  while (true) {
    const idx = input.indexOf('\n\n', offset)
    if (idx === -1) {
      break
    }
    blocks.push(input.slice(offset, idx))
    offset = idx + 2
  }

  return {
    blocks,
    rest: input.slice(offset),
  }
}

function parseSSEBlock(block: string): { eventName: string; data: string | null } {
  let eventName = ''
  const dataLines: string[] = []

  for (const line of block.split('\n')) {
    if (line.startsWith('event:')) {
      const rawEvent = line.slice(6)
      eventName = rawEvent.startsWith(' ') ? rawEvent.slice(1).trim() : rawEvent.trim()
      continue
    }
    if (line.startsWith('data:')) {
      const rawData = line.slice(5)
      const value = rawData.startsWith(' ') ? rawData.slice(1) : rawData
      dataLines.push(value)
    }
  }

  if (dataLines.length === 0) {
    return { eventName, data: null }
  }
  const joined = dataLines.join('\n')
  if (joined.trim() === '') {
    return { eventName, data: null }
  }
  return { eventName, data: joined }
}

/**
 * Replays missing public events after a disconnect using the SQL cursor.
 * The run is durable; polling stops at a terminal status or RUN_FINISHED.
 */
export async function recoverRun(
  runId: string,
  afterSeq: number,
  afterSubIndex: number,
  onEvent: (event: AGUIEvent) => void,
  onComplete?: () => void,
  maxAttempts = 120,
): Promise<void> {
  if (!runId) throw new Error('runId is required')
  let cursor = afterSeq
  let cursorSubIndex = afterSubIndex
  for (let attempt = 0; attempt < maxAttempts; attempt++) {
    const response = await fetch(
      `/api/runs/${encodeURIComponent(runId)}/events?after=${cursor}&subIndex=${cursorSubIndex}&limit=1000`,
      { headers: authHeaders() },
    )
    if (!response.ok) throw new Error(`replay failed: HTTP ${response.status}`)
    const payload = (await response.json()) as { events?: AGUIEvent[] }
    const events = payload.events ?? []
    let terminal = false
    for (const event of events) {
      if (typeof event.seq === 'number') {
        const subIndex = typeof event.subIndex === 'number' ? event.subIndex : 0
        if (event.seq < cursor || (event.seq === cursor && subIndex <= cursorSubIndex)) continue
        cursor = event.seq
        cursorSubIndex = subIndex
      }
      onEvent(event)
      if (event.type === 'RUN_FINISHED' || event.type === 'RUN_ERROR') terminal = true
    }
    if (terminal) {
      onComplete?.()
      return
    }
    const status = await fetch(`/api/runs/${encodeURIComponent(runId)}/status`, { headers: authHeaders() })
    if (status.ok) {
      const state = (await status.json()) as { status?: string }
      if (state.status && ['completed', 'failed', 'cancelled'].includes(state.status)) {
        onComplete?.()
        return
      }
    }
    await new Promise((resolve) => setTimeout(resolve, 500))
  }
  throw new Error('replay timed out')
}
