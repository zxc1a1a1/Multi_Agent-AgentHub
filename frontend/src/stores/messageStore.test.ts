import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useMessageStore } from './messageStore'
import type { AGUIChatRequest } from '../agui/client'
import type { AGUIEvent } from '../types'
import * as api from '../services/api'

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

describe('multi-agent mixed ordered_parallel', () => {
  beforeEach(() => {
    streamByConversation.clear()
    useMessageStore.setState({
      messages: {},
      streamingByConversation: {},
      abortControllersByConversation: {},
    })
  })

  it('creates 3 separate agent messages for web-agent, code-agent, orchestrator', () => {
    const { sendMessage } = useMessageStore.getState()
    sendMessage('conv-mixed', 'build login page and go api')

    // Web-agent message (msg-0)
    emit('conv-mixed', {
      type: 'TEXT_MESSAGE_START',
      runId: 'run-001',
      messageId: 'msg-0',
      sender: { type: 'agent', name: 'web-agent', displayName: 'Web Agent' },
      author: 'web-agent',
    })
    emit('conv-mixed', {
      type: 'TEXT_MESSAGE_CONTENT',
      runId: 'run-001',
      messageId: 'msg-0',
      delta: '<section><h1>Login</h1>',
      sender: { type: 'agent', name: 'web-agent' },
    })
    emit('conv-mixed', {
      type: 'TEXT_MESSAGE_END',
      runId: 'run-001',
      messageId: 'msg-0',
    })

    // Code-agent message (msg-1) — different messageId!
    emit('conv-mixed', {
      type: 'TEXT_MESSAGE_START',
      runId: 'run-001',
      messageId: 'msg-1',
      sender: { type: 'agent', name: 'code-agent', displayName: 'Code Agent' },
      author: 'code-agent',
    })
    emit('conv-mixed', {
      type: 'TEXT_MESSAGE_CONTENT',
      runId: 'run-001',
      messageId: 'msg-1',
      delta: 'package main\n\nimport "net/http"',
      sender: { type: 'agent', name: 'code-agent' },
    })
    emit('conv-mixed', {
      type: 'TEXT_MESSAGE_END',
      runId: 'run-001',
      messageId: 'msg-1',
    })

    // Orchestrator summary (msg-2) — different messageId!
    emit('conv-mixed', {
      type: 'TEXT_MESSAGE_START',
      runId: 'run-001',
      messageId: 'msg-summary',
      sender: { type: 'orchestrator', name: 'orchestrator', displayName: 'Orchestrator' },
      author: 'orchestrator',
    })
    emit('conv-mixed', {
      type: 'TEXT_MESSAGE_CONTENT',
      runId: 'run-001',
      messageId: 'msg-summary',
      delta: 'All 2 task(s) completed successfully.',
      sender: { type: 'orchestrator', name: 'orchestrator' },
    })
    emit('conv-mixed', {
      type: 'TEXT_MESSAGE_END',
      runId: 'run-001',
      messageId: 'msg-summary',
    })

    // Run finished
    emit('conv-mixed', { type: 'RUN_FINISHED', runId: 'run-001' })

    const messages = useMessageStore.getState().messages['conv-mixed'] || []

    // Should have: 1 user message + 3 agent messages = 4 total
    const agentMessages = messages.filter((msg) => msg.senderType === 'agent')
    expect(agentMessages).toHaveLength(3)

    // Each agent message should have distinct content from the right sender
    const webMsg = agentMessages.find((msg) => msg.senderName === 'Web Agent' || msg.agentName === 'web-agent')
    const codeMsg = agentMessages.find((msg) => msg.senderName === 'Code Agent' || msg.agentName === 'code-agent')
    const orchMsg = agentMessages.find((msg) => msg.senderName === 'Orchestrator' || msg.agentName === 'orchestrator')

    expect(webMsg).toBeDefined()
    expect(codeMsg).toBeDefined()
    expect(orchMsg).toBeDefined()

    expect(webMsg?.content).toContain('<section><h1>Login</h1>')
    expect(codeMsg?.content).toContain('package main')
    expect(orchMsg?.content).toContain('All 2 task(s) completed successfully')
  })

  it('does not merge different messageId into same bubble', () => {
    const { sendMessage } = useMessageStore.getState()
    sendMessage('conv-sep', 'test')

    // First agent message
    emit('conv-sep', {
      type: 'TEXT_MESSAGE_START',
      messageId: 'msg-a',
      sender: { type: 'agent', name: 'agent-a' },
    })
    emit('conv-sep', {
      type: 'TEXT_MESSAGE_CONTENT',
      messageId: 'msg-a',
      delta: 'content from A',
    })
    emit('conv-sep', { type: 'TEXT_MESSAGE_END', messageId: 'msg-a' })

    // Second agent message — different messageId, must be separate bubble
    emit('conv-sep', {
      type: 'TEXT_MESSAGE_START',
      messageId: 'msg-b',
      sender: { type: 'agent', name: 'agent-b' },
    })
    emit('conv-sep', {
      type: 'TEXT_MESSAGE_CONTENT',
      messageId: 'msg-b',
      delta: 'content from B',
    })
    emit('conv-sep', { type: 'TEXT_MESSAGE_END', messageId: 'msg-b' })

    const messages = useMessageStore.getState().messages['conv-sep'] || []
    const agentMessages = messages.filter((msg) => msg.senderType === 'agent')
    expect(agentMessages).toHaveLength(2)

    // Messages must have different content and different IDs
    expect(agentMessages[0].id).not.toBe(agentMessages[1].id)
    expect(agentMessages[0].content).not.toBe(agentMessages[1].content)
  })

  it('handles RUN_ERROR with safe error message', () => {
    const { sendMessage } = useMessageStore.getState()
    sendMessage('conv-err', 'trigger error')

    emit('conv-err', { type: 'RUN_STARTED', runId: 'run-err' })
    emit('conv-err', {
      type: 'TEXT_MESSAGE_START',
      messageId: 'msg-err',
      sender: { type: 'agent', name: 'test-agent' },
    })
    emit('conv-err', {
      type: 'TEXT_MESSAGE_CONTENT',
      messageId: 'msg-err',
      delta: 'partial response before error',
    })
    emit('conv-err', {
      type: 'RUN_ERROR',
      runId: 'run-err',
      error: { code: 'AGUI_INTERNAL', message: 'assistant run failed' },
    })

    const messages = useMessageStore.getState().messages['conv-err'] || []
    // Should have user message + failed agent message
    const agentMessages = messages.filter((msg) => msg.senderType === 'agent')
    expect(agentMessages.length).toBeGreaterThanOrEqual(1)

    const lastAgentMsg = agentMessages[agentMessages.length - 1]
    expect(lastAgentMsg.status).toBe('failed')
    // Error message should not contain internal details
    expect(lastAgentMsg.content).not.toContain('stack')
    expect(lastAgentMsg.content).not.toContain('token')
  })

  it('propagates sender object fields to agent message', () => {
    const { sendMessage } = useMessageStore.getState()
    sendMessage('conv-sender', 'test sender')

    emit('conv-sender', {
      type: 'TEXT_MESSAGE_START',
      messageId: 'msg-sender',
      sender: { type: 'agent', name: 'web-agent', displayName: 'Web Agent' },
    })
    emit('conv-sender', {
      type: 'TEXT_MESSAGE_CONTENT',
      messageId: 'msg-sender',
      delta: 'agent content',
      sender: { type: 'agent', name: 'web-agent' },
    })
    emit('conv-sender', { type: 'TEXT_MESSAGE_END', messageId: 'msg-sender' })

    const messages = useMessageStore.getState().messages['conv-sender'] || []
    const agentMsg = messages.find((msg) => msg.senderType === 'agent')
    expect(agentMsg).toBeDefined()
    expect(agentMsg?.senderName).toBe('Web Agent')
    expect(agentMsg?.agentName).toBe('web-agent')
  })

  it('propagates orchestrator sender type for summary messages', () => {
    const { sendMessage } = useMessageStore.getState()
    sendMessage('conv-orch', 'test orchestrator')

    emit('conv-orch', {
      type: 'TEXT_MESSAGE_START',
      messageId: 'msg-orch',
      sender: { type: 'orchestrator', name: 'orchestrator', displayName: 'Orchestrator' },
    })
    emit('conv-orch', {
      type: 'TEXT_MESSAGE_CONTENT',
      messageId: 'msg-orch',
      delta: 'All tasks completed.',
      sender: { type: 'orchestrator', name: 'orchestrator' },
    })
    emit('conv-orch', { type: 'TEXT_MESSAGE_END', messageId: 'msg-orch' })

    const messages = useMessageStore.getState().messages['conv-orch'] || []
    const orchMsg = messages.find((msg) => msg.senderType === 'agent' && msg.senderName === 'Orchestrator')
    expect(orchMsg).toBeDefined()
    expect(orchMsg?.content).toContain('All tasks completed')
  })

  it('strips sensitive secrets from error message text', () => {
    const { sendMessage } = useMessageStore.getState()
    sendMessage('conv-secret', 'trigger error')

    const fakeOpenAIKey = 'sk-' + 'abc123def45678901234567890'
    emit('conv-secret', {
      type: 'RUN_ERROR',
      runId: 'run-secret',
      error: { code: 'INTERNAL', message: 'OPENAI_API_KEY=' + fakeOpenAIKey },
    })

    const messages = useMessageStore.getState().messages['conv-secret'] || []
    const failedMsg = messages.find((msg) => msg.status === 'failed')
    expect(failedMsg).toBeDefined()
    expect(failedMsg?.content).not.toContain('sk-abc')
    expect(failedMsg?.content).not.toContain('OPENAI_API_KEY')
  })

  it('strips file paths from error message text', () => {
    const { sendMessage } = useMessageStore.getState()
    sendMessage('conv-path', 'trigger error')

    emit('conv-path', {
      type: 'RUN_ERROR',
      runId: 'run-path',
      error: { code: 'INTERNAL', message: 'panic at C:\\Users\\service\\main.go:42' },
    })

    const messages = useMessageStore.getState().messages['conv-path'] || []
    const failedMsg = messages.find((msg) => msg.status === 'failed')
    expect(failedMsg).toBeDefined()
    expect(failedMsg?.content).not.toContain('C:\\Users')
  })

  it('replaces panic/stack trace with generic error message', () => {
    const { sendMessage } = useMessageStore.getState()
    sendMessage('conv-panic', 'trigger error')

    emit('conv-panic', {
      type: 'RUN_ERROR',
      runId: 'run-panic',
      error: { code: 'INTERNAL', message: 'panic: runtime error: invalid memory address\nstack trace:\ngoroutine 1...' },
    })

    const messages = useMessageStore.getState().messages['conv-panic'] || []
    const failedMsg = messages.find((msg) => msg.status === 'failed')
    expect(failedMsg).toBeDefined()
    expect(failedMsg?.content).not.toContain('panic')
    expect(failedMsg?.content).not.toContain('stack trace')
  })

  it('strips sk-prefixed token from error message', () => {
    const { sendMessage } = useMessageStore.getState()
    sendMessage('conv-sktoken', 'trigger error')

    const fakeProjectToken = 'sk-' + 'proj-' + 'abcdefghijklmnopqrstuvwxyz123456'
    emit('conv-sktoken', {
      type: 'RUN_ERROR',
      runId: 'run-sktoken',
      error: { code: 'INTERNAL', message: 'auth failed with token ' + fakeProjectToken },
    })

    const messages = useMessageStore.getState().messages['conv-sktoken'] || []
    const failedMsg = messages.find((msg) => msg.status === 'failed')
    expect(failedMsg).toBeDefined()
    expect(failedMsg?.content).not.toContain('sk-proj')
  })

  it('creates code preview based on toolName, not agentName', () => {
    const { sendMessage } = useMessageStore.getState()
    // Send as web-agent, but emit code_preview tool call
    sendMessage('conv-toolname', 'write code', { agentName: 'web-agent' })

    emit('conv-toolname', { type: 'TEXT_MESSAGE_START', messageId: 'msg-tool' })
    emit('conv-toolname', {
      type: 'tool.call',
      messageId: 'msg-tool',
      toolCall: {
        id: 'tc-1',
        name: 'code_preview',
        arguments: { code: 'package main', language: 'go', filename: 'main.go' },
      },
      agentName: 'web-agent',
    })
    emit('conv-toolname', { type: 'TEXT_MESSAGE_END', messageId: 'msg-tool' })

    const messages = useMessageStore.getState().messages['conv-toolname'] || []
    const agentMsg = messages.find((msg) => msg.senderType === 'agent')
    expect(agentMsg).toBeDefined()
    // Code preview should be created even though agentName is web-agent
    expect(agentMsg?.codeBlocks).toHaveLength(1)
    expect(agentMsg?.codeBlocks?.[0].filename).toBe('main.go')
  })

  it('creates web preview based on toolName, not agentName', () => {
    const { sendMessage } = useMessageStore.getState()
    // Send as code-agent, but emit web_preview tool call
    sendMessage('conv-toolname-web', 'build page', { agentName: 'code-agent' })

    emit('conv-toolname-web', { type: 'TEXT_MESSAGE_START', messageId: 'msg-tool-web' })
    emit('conv-toolname-web', {
      type: 'tool.call',
      messageId: 'msg-tool-web',
      toolCall: {
        id: 'tc-2',
        name: 'web_preview',
        arguments: { title: 'page.html', html: '<h1>Page</h1>' },
      },
      agentName: 'code-agent',
    })
    emit('conv-toolname-web', { type: 'TEXT_MESSAGE_END', messageId: 'msg-tool-web' })

    const messages = useMessageStore.getState().messages['conv-toolname-web'] || []
    const agentMsg = messages.find((msg) => msg.senderType === 'agent')
    expect(agentMsg).toBeDefined()
    // Web preview should be created even though agentName is code-agent
    expect(agentMsg?.webPreviews).toHaveLength(1)
    expect(agentMsg?.webPreviews?.[0].title).toBe('page.html')
  })
})
