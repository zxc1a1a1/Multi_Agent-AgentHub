import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useMessageStore } from './messageStore'
import type { AGUIRunRequest } from '../agui/client'

vi.mock('../services/api', () => ({
  listMessages: vi.fn().mockResolvedValue([]),
}))

const controllerMap = new Map<string, AbortController>()

vi.mock('../agui/client', () => ({
  runAgent: vi.fn(
    (
      request: AGUIRunRequest,
      _onEvent: (event: unknown) => void,
      onError?: (err: Error) => void,
      onComplete?: () => void,
    ) => {
      const controller = new AbortController()
      controller.signal.addEventListener('abort', () => {
        onError?.(new DOMException('aborted', 'AbortError'))
        onComplete?.()
      })
      controllerMap.set(request.threadId, controller)
      return controller
    },
  ),
}))

describe('messageStore per-conversation streaming', () => {
  beforeEach(() => {
    controllerMap.clear()
    useMessageStore.setState({
      messages: {},
      streamingByConversation: {},
      abortControllersByConversation: {},
    })
  })

  it('tracks streaming state independently across conversations', () => {
    const { sendMessage, isStreaming } = useMessageStore.getState()
    sendMessage('conv-a', 'hello a')
    sendMessage('conv-b', 'hello b')

    expect(isStreaming('conv-a')).toBe(true)
    expect(isStreaming('conv-b')).toBe(true)
  })

  it('stopping one conversation does not stop another conversation', () => {
    const { sendMessage, isStreaming, stopStreaming } = useMessageStore.getState()
    sendMessage('conv-a', 'hello a')
    sendMessage('conv-b', 'hello b')

    stopStreaming('conv-a')

    expect(controllerMap.get('conv-a')?.signal.aborted).toBe(true)
    expect(controllerMap.get('conv-b')?.signal.aborted).toBe(false)
    expect(isStreaming('conv-a')).toBe(false)
    expect(isStreaming('conv-b')).toBe(true)
  })
})
