import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useMessageStore } from './messageStore'
import type { AGUIChatRequest } from '../agui/client'
import type { AGUIEvent } from '../types'

vi.mock('../services/api', () => ({
  listMessages: vi.fn().mockResolvedValue([]),
}))

type StreamHarness = {
  request: AGUIChatRequest
  onEvent: (event: AGUIEvent) => void
  onError?: (error: Error) => void
  onComplete?: () => void
  controller: AbortController
}

const streamByConversation = new Map<string, StreamHarness>()

vi.mock('../agui/client', () => ({
  runAgent: vi.fn(
    (
      request: AGUIChatRequest,
      onEvent: (event: AGUIEvent) => void,
      onError?: (error: Error) => void,
      onComplete?: () => void,
    ) => {
      const controller = new AbortController()
      controller.signal.addEventListener('abort', () => {
        onError?.(new DOMException('aborted', 'AbortError'))
        onComplete?.()
      })
      streamByConversation.set(request.conversationId, {
        request,
        onEvent,
        onError,
        onComplete,
        controller,
      })
      return controller
    },
  ),
}))

function emit(conversationId: string, event: AGUIEvent) {
  const harness = streamByConversation.get(conversationId)
  if (!harness) {
    throw new Error(`missing stream harness for ${conversationId}`)
  }
  harness.onEvent(event)
}

describe('messageStore per-conversation streaming', () => {
  beforeEach(() => {
    streamByConversation.clear()
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

    expect(streamByConversation.get('conv-a')?.controller.signal.aborted).toBe(true)
    expect(streamByConversation.get('conv-b')?.controller.signal.aborted).toBe(false)
    expect(isStreaming('conv-a')).toBe(false)
    expect(isStreaming('conv-b')).toBe(true)
  })

  it('sends selected agentName to /api/chat request', () => {
    const { sendMessage } = useMessageStore.getState()
    sendMessage('conv-web', 'build web page', { agentName: 'web-agent' })

    const request = streamByConversation.get('conv-web')?.request
    expect(request).toBeDefined()
    expect(request?.agentName).toBe('web-agent')
    expect(request?.conversationId).toBe('conv-web')
    expect(request?.message).toBe('build web page')
  })

  it('keeps agentName optional in /api/chat request', () => {
    const { sendMessage } = useMessageStore.getState()
    sendMessage('conv-default', 'default route')

    const request = streamByConversation.get('conv-default')?.request
    expect(request).toBeDefined()
    expect(request?.agentName).toBeUndefined()
  })

  it('falls back senderName from selected agent when stream has no sender fields', () => {
    const { sendMessage } = useMessageStore.getState()
    sendMessage('conv-web', 'render html', { agentName: 'web-agent' })

    emit('conv-web', { type: 'TEXT_MESSAGE_CONTENT', content: 'hello from web agent' })
    emit('conv-web', { type: 'TEXT_MESSAGE_END' })

    const messages = useMessageStore.getState().messages['conv-web']
    const agentMessage = messages.find((message) => message.senderType === 'agent')
    expect(agentMessage).toBeDefined()
    expect(agentMessage?.senderName).toBe('Web Agent')
    expect(agentMessage?.agentName).toBe('web-agent')
  })

  it('captures web_preview tool payload as safe web preview block', () => {
    const { sendMessage } = useMessageStore.getState()
    sendMessage('conv-web', 'preview html', { agentName: 'web-agent' })

    emit('conv-web', {
      type: 'tool.call',
      toolCall: {
        id: 'tc-1',
        name: 'web_preview',
        arguments: {
          title: 'index.html',
          html: '<section><h1>hello</h1><script>alert(1)</script></section>',
        },
      },
    })

    const messages = useMessageStore.getState().messages['conv-web']
    const agentMessage = messages.find((message) => message.senderType === 'agent')
    expect(agentMessage).toBeDefined()
    expect(agentMessage?.webPreviews).toHaveLength(1)
    expect(agentMessage?.webPreviews?.[0].title).toBe('index.html')
    expect(agentMessage?.webPreviews?.[0].html).toContain('<script>alert(1)</script>')
  })
})
