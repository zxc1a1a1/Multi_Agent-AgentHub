import type { AGUIEvent } from '../types'

export interface AGUIRunRequest {
  threadId: string
  runId: string
  messages: { role: string; content: string }[]
  tools: { name: string }[]
}

function authHeaders(): Record<string, string> {
  const t = import.meta.env.VITE_AGENTHUB_API_TOKEN as string
  const h: Record<string, string> = { 'Content-Type': 'application/json' }
  if (t) h['Authorization'] = `Bearer ${t}`
  return h
}

/**
 * Sends a run request to the AG-UI endpoint and streams SSE events back.
 * Uses fetch + ReadableStream for POST-based SSE (EventSource only supports GET).
 */
export function runAgent(
  request: AGUIRunRequest,
  onEvent: (event: AGUIEvent) => void,
  onError?: (err: Error) => void,
  onComplete?: () => void,
): AbortController {
  const controller = new AbortController()

  fetch('/api/agui/run', {
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
          const data = extractSSEData(block)
          if (!data) continue
          try {
            const event: AGUIEvent = JSON.parse(data)
            onEvent(event)
          } catch {
            // skip malformed JSON and continue streaming
          }
        }
      }

      onComplete?.()
    })
    .catch((err) => {
      if (err.name !== 'AbortError') {
        onError?.(err)
      }
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

function extractSSEData(block: string): string | null {
  const dataLines: string[] = []

  for (const line of block.split('\n')) {
    if (!line.startsWith('data:')) continue
    const raw = line.slice(5)
    const value = raw.startsWith(' ') ? raw.slice(1) : raw
    dataLines.push(value)
  }

  if (dataLines.length === 0) return null
  const joined = dataLines.join('\n')
  if (joined.trim() === '') return null
  return joined
}
