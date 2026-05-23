import type { AGUIEvent } from '../types'

export interface AGUIRunRequest {
  threadId: string
  runId: string
  messages: { role: string; content: string }[]
  tools: { name: string }[]
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
    headers: { 'Content-Type': 'application/json' },
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
        if (done) break

        // Decode binary chunk and append to buffer
        buffer += decoder.decode(value, { stream: true })

        // Split by newlines (SSE events are separated by \n\n)
        const lines = buffer.split('\n')
        // Keep the last incomplete line in buffer
        buffer = lines.pop() || ''

        for (const line of lines) {
          // SSE format: "data: {json}"
          if (line.startsWith('data:')) {
            const data = line.slice(5).trim()
            if (!data) continue
            try {
              const event: AGUIEvent = JSON.parse(data)
              onEvent(event)
            } catch {
              // skip malformed JSON
            }
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
