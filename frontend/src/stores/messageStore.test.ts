import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useMessageStore } from './messageStore'
import type { AGUIChatRequest } from '../agui/client'
import type { AGUIEvent } from '../types'
import * as api from '../services/api'

vi.mock('../services/api', () => ({
  listMessages: vi.fn().mockResolvedValue([]),
  confirmHITL: vi.fn().mockResolvedValue(undefined),
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

    // Code-agent message (msg-1) 鈥?different messageId!
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

    // Orchestrator summary (msg-2) 鈥?different messageId!
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

    // Second agent message 鈥?different messageId, must be separate bubble
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

describe('web preview auto-mode triggering', () => {
  beforeEach(() => {
    streamByConversation.clear()
    useMessageStore.setState({
      messages: {},
      streamingByConversation: {},
      abortControllersByConversation: {},
    })
  })

  it('creates web preview in auto mode when web_preview tool call occurs', () => {
    const { sendMessage } = useMessageStore.getState()
    // Auto mode 鈥?no agentName passed
    sendMessage('conv-auto-tool', 'build a login page')

    emit('conv-auto-tool', { type: 'TEXT_MESSAGE_START', messageId: 'msg-1' })
    emit('conv-auto-tool', {
      type: 'tool.call',
      messageId: 'msg-1',
      toolCall: {
        id: 'tc-1',
        name: 'web_preview',
        arguments: { title: 'login.html', html: '<form><input type="text"></form>' },
      },
    })
    emit('conv-auto-tool', { type: 'TEXT_MESSAGE_END', messageId: 'msg-1' })

    const messages = useMessageStore.getState().messages['conv-auto-tool'] || []
    const agentMsg = messages.find((msg) => msg.senderType === 'agent')
    expect(agentMsg).toBeDefined()
    expect(agentMsg?.webPreviews).toHaveLength(1)
    expect(agentMsg?.webPreviews?.[0].title).toBe('login.html')
  })

  it('creates web preview in auto mode when artifact.delta has webpage type', () => {
    const { sendMessage } = useMessageStore.getState()
    sendMessage('conv-auto-artifact', 'make a page')

    emit('conv-auto-artifact', { type: 'TEXT_MESSAGE_START', messageId: 'msg-a' })
    emit('conv-auto-artifact', {
      type: 'artifact.delta',
      messageId: 'msg-a',
      artifact: {
        type: 'webpage',
        title: 'artifact-page.html',
        content: '<html><body><h1>Artifact</h1></body></html>',
      },
    })
    emit('conv-auto-artifact', { type: 'TEXT_MESSAGE_END', messageId: 'msg-a' })

    const messages = useMessageStore.getState().messages['conv-auto-artifact'] || []
    const agentMsg = messages.find((msg) => msg.senderType === 'agent')
    expect(agentMsg).toBeDefined()
    expect(agentMsg?.webPreviews).toHaveLength(1)
    expect(agentMsg?.webPreviews?.[0].title).toBe('artifact-page.html')
  })

  it('creates web preview in auto mode when SSE resolves agentName to web-agent and content is HTML', () => {
    const { sendMessage } = useMessageStore.getState()
    sendMessage('conv-auto-resolved', 'create a page')

    // SSE event with explicit agentName from orchestrator
    emit('conv-auto-resolved', {
      type: 'TEXT_MESSAGE_START',
      messageId: 'msg-w',
      agentName: 'web-agent',
      sender: { type: 'agent', name: 'web-agent', displayName: 'Web Agent' },
    })
    emit('conv-auto-resolved', {
      type: 'TEXT_MESSAGE_CONTENT',
      messageId: 'msg-w',
      delta: '<html><body><h1>Auto Page</h1></body></html>',
      agentName: 'web-agent',
    })
    emit('conv-auto-resolved', { type: 'TEXT_MESSAGE_END', messageId: 'msg-w' })

    const messages = useMessageStore.getState().messages['conv-auto-resolved'] || []
    const agentMsg = messages.find((msg) => msg.senderType === 'agent')
    expect(agentMsg).toBeDefined()
    expect(agentMsg?.webPreviews).toHaveLength(1)
    expect(agentMsg?.webPreviews?.[0].html).toContain('<h1>Auto Page</h1>')
  })

  it('does NOT create web preview from markdown code fence HTML in auto mode', () => {
    const { sendMessage } = useMessageStore.getState()
    sendMessage('conv-auto-codefence', 'show me an example')

    // Content is a markdown code block containing HTML 鈥?should NOT trigger web preview.
    emit('conv-auto-codefence', { type: 'TEXT_MESSAGE_START', messageId: 'msg-cf' })
    emit('conv-auto-codefence', {
      type: 'TEXT_MESSAGE_CONTENT',
      messageId: 'msg-cf',
      delta: 'Here is an HTML example:\n\n```html\n<html>\n<body>\n<h1>Example</h1>\n</body>\n</html>\n```\n\nThat is a code sample.',
    })
    emit('conv-auto-codefence', { type: 'TEXT_MESSAGE_END', messageId: 'msg-cf' })

    const messages = useMessageStore.getState().messages['conv-auto-codefence'] || []
    const agentMsg = messages.find((msg) => msg.senderType === 'agent')
    expect(agentMsg).toBeDefined()
    // Should NOT have web previews 鈥?HTML is inside a markdown code fence.
    expect(agentMsg?.webPreviews).toBeUndefined()
  })

  it('does NOT create web preview in auto mode when only code-agent HTML example is in content', () => {
    const { sendMessage } = useMessageStore.getState()
    sendMessage('conv-auto-codeagent', 'explain HTML forms')

    // SSE resolves to code-agent; content happens to mention HTML
    emit('conv-auto-codeagent', {
      type: 'TEXT_MESSAGE_START',
      messageId: 'msg-ca',
      agentName: 'code-agent',
      sender: { type: 'agent', name: 'code-agent', displayName: 'Code Agent' },
    })
    emit('conv-auto-codeagent', {
      type: 'TEXT_MESSAGE_CONTENT',
      messageId: 'msg-ca',
      delta: 'To create a form, use:\n\n```html\n<form>\n<input type="text">\n</form>\n```\n\nThis is standard HTML.',
      agentName: 'code-agent',
    })
    emit('conv-auto-codeagent', { type: 'TEXT_MESSAGE_END', messageId: 'msg-ca' })

    const messages = useMessageStore.getState().messages['conv-auto-codeagent'] || []
    const agentMsg = messages.find((msg) => msg.senderType === 'agent')
    expect(agentMsg).toBeDefined()
    expect(agentMsg?.codeBlocks).toBeUndefined()
    // Should NOT extract web preview from code-agent HTML examples in code fences.
    expect(agentMsg?.webPreviews).toBeUndefined()
  })
})

describe('plan confirmation via TOOL_CALL events', () => {
  beforeEach(() => {
    streamByConversation.clear()
    useMessageStore.setState({
      messages: {},
      streamingByConversation: {},
      abortControllersByConversation: {},
      orchestrationByConversation: {},
      confirmationByConversation: {},
    })
  })

  it('sets pending confirmation on confirm_plan TOOL_CALL_START/ARGS/END', () => {
    const { sendMessage } = useMessageStore.getState()
    sendMessage('conv-confirm', 'build login page')

    emit('conv-confirm', {
      type: 'TOOL_CALL_START',
      runId: 'run-001',
      toolCallId: 'plan-abc',
      toolName: 'confirm_plan',
      toolCall: { id: 'plan-abc', name: 'confirm_plan' },
    })
    emit('conv-confirm', {
      type: 'TOOL_CALL_ARGS',
      runId: 'run-001',
      toolCallId: 'plan-abc',
      delta: JSON.stringify({
        runId: 'run-001',
        planId: 'plan-abc',
        strategy: 'ordered_parallel',
        plannedAgents: ['code-agent', 'web-agent'],
        tasks: [
          { taskId: 'task-1', agentName: 'code-agent', content: 'Build Go API' },
          { taskId: 'task-2', agentName: 'web-agent', content: 'Build login page' },
        ],
        intentSummary: 'Build a full-stack login feature',
        requiresConfirmation: true,
      }),
    })
    emit('conv-confirm', {
      type: 'TOOL_CALL_END',
      runId: 'run-001',
      toolCallId: 'plan-abc',
    })

    const confirmation = useMessageStore.getState().getConfirmation('conv-confirm')
    expect(confirmation).not.toBeNull()
    expect(confirmation?.status).toBe('pending')
    expect(confirmation?.runId).toBe('run-001')
    expect(confirmation?.actionId).toBe('plan-abc')
    expect(confirmation?.agentNames).toEqual(['code-agent', 'web-agent'])
    expect(confirmation?.tasks).toHaveLength(2)
    expect(confirmation?.strategy).toBe('ordered_parallel')
    expect(confirmation?.intentSummary).toBe('Build a full-stack login feature')
  })

  it('ignores TOOL_CALL_END for non-confirm_plan tools', () => {
    const { sendMessage } = useMessageStore.getState()
    sendMessage('conv-other-tool', 'write code')

    emit('conv-other-tool', {
      type: 'TOOL_CALL_START',
      runId: 'run-002',
      toolCallId: 'tc-1',
      toolName: 'code_preview',
      toolCall: { id: 'tc-1', name: 'code_preview' },
    })
    emit('conv-other-tool', {
      type: 'TOOL_CALL_ARGS',
      runId: 'run-002',
      toolCallId: 'tc-1',
      delta: JSON.stringify({ code: 'func main() {}', language: 'go' }),
    })
    emit('conv-other-tool', {
      type: 'TOOL_CALL_END',
      runId: 'run-002',
      toolCallId: 'tc-1',
    })

    const confirmation = useMessageStore.getState().getConfirmation('conv-other-tool')
    expect(confirmation).toBeNull()
  })
})

describe('plan confirmation via STATE_UPDATE metadata', () => {
  beforeEach(() => {
    streamByConversation.clear()
    useMessageStore.setState({
      messages: {},
      streamingByConversation: {},
      abortControllersByConversation: {},
      orchestrationByConversation: {},
      confirmationByConversation: {},
    })
  })

  it('captures requiresConfirmation and plannedAgents from STATE_UPDATE', () => {
    const { sendMessage } = useMessageStore.getState()
    sendMessage('conv-state-confirm', 'build dashboard')

    emit('conv-state-confirm', {
      type: 'STATE_UPDATE',
      runId: 'run-003',
      state: {
        phase: 'awaiting_confirmation',
        requiresConfirmation: true,
        confirmationActionId: 'plan-state-1',
        plannedAgents: ['code-agent', 'web-agent', 'document-agent'],
        tasks: [
          { taskId: 't1', agentName: 'code-agent', content: 'Build API' },
          { taskId: 't2', agentName: 'web-agent', content: 'Build dashboard UI' },
          { taskId: 't3', agentName: 'document-agent', content: 'Write API docs' },
        ],
        strategy: 'sequential',
        intentSummary: 'Build a dashboard with docs',
        taskCount: 3,
      },
    })

    const orch = useMessageStore.getState().orchestrationByConversation['conv-state-confirm']
    expect(orch).toBeDefined()
    expect(orch?.requiresConfirmation).toBe(true)
    expect(orch?.confirmationActionId).toBe('plan-state-1')
    expect(orch?.plannedAgents).toEqual(['code-agent', 'web-agent', 'document-agent'])
    expect(orch?.plannedTasks).toHaveLength(3)
    expect(orch?.strategy).toBe('sequential')
    expect(orch?.taskCount).toBe(3)
  })

  it('preserves STATE_UPDATE metadata alongside confirm_plan tool events', () => {
    const { sendMessage } = useMessageStore.getState()
    sendMessage('conv-dual', 'build feature')

    // STATE_UPDATE first (backward compat path)
    emit('conv-dual', {
      type: 'STATE_UPDATE',
      runId: 'run-004',
      state: {
        requiresConfirmation: true,
        confirmationActionId: 'plan-dual',
        plannedAgents: ['code-agent'],
        tasks: [{ taskId: 't1', agentName: 'code-agent', content: 'Implement feature' }],
        strategy: 'single',
      },
    })

    // Then TOOL_CALL events (AG-UI standard path)
    emit('conv-dual', {
      type: 'TOOL_CALL_START',
      runId: 'run-004',
      toolCallId: 'plan-dual',
      toolName: 'confirm_plan',
      toolCall: { id: 'plan-dual', name: 'confirm_plan' },
    })
    emit('conv-dual', {
      type: 'TOOL_CALL_ARGS',
      runId: 'run-004',
      toolCallId: 'plan-dual',
      delta: JSON.stringify({
        runId: 'run-004',
        planId: 'plan-dual',
        strategy: 'single',
        plannedAgents: ['code-agent'],
        tasks: [{ taskId: 't1', agentName: 'code-agent', content: 'Implement feature' }],
        intentSummary: 'Implement the feature',
        requiresConfirmation: true,
      }),
    })
    emit('conv-dual', {
      type: 'TOOL_CALL_END',
      runId: 'run-004',
      toolCallId: 'plan-dual',
    })

    // Both paths should have populated data
    const orch = useMessageStore.getState().orchestrationByConversation['conv-dual']
    expect(orch?.requiresConfirmation).toBe(true)

    const confirmation = useMessageStore.getState().getConfirmation('conv-dual')
    expect(confirmation).not.toBeNull()
    expect(confirmation?.status).toBe('pending')
    expect(confirmation?.runId).toBe('run-004')
  })
})

describe('confirmPlan API integration', () => {
  beforeEach(() => {
    streamByConversation.clear()
    vi.clearAllMocks()
    useMessageStore.setState({
      messages: {},
      streamingByConversation: {},
      abortControllersByConversation: {},
      orchestrationByConversation: {},
      confirmationByConversation: {},
    })
  })

  it('confirmPlan calls api.confirmHITL with confirmed=true', async () => {
    const { confirmHITL } = await import('../services/api')

    // Set up a pending confirmation first
    useMessageStore.setState({
      confirmationByConversation: {
        'conv-api-1': {
          runId: 'run-api-1',
          actionId: 'plan-api-1',
          agentNames: ['code-agent'],
          tasks: [{ taskId: 't1', agentName: 'code-agent', content: 'Task' }],
          strategy: 'single',
          status: 'pending',
        },
      },
    })

    const { confirmPlan } = useMessageStore.getState()
    await confirmPlan('conv-api-1', 'run-api-1', 'plan-api-1', true)

    expect(confirmHITL).toHaveBeenCalledWith({
      runId: 'run-api-1',
      actionId: 'plan-api-1',
      confirmed: true,
      rejectReason: '',
    })

    const confirmation = useMessageStore.getState().getConfirmation('conv-api-1')
    expect(confirmation?.status).toBe('confirmed')
  })

  it('confirmPlan calls api.confirmHITL with confirmed=false and reason', async () => {
    const { confirmHITL } = await import('../services/api')

    useMessageStore.setState({
      confirmationByConversation: {
        'conv-api-2': {
          runId: 'run-api-2',
          actionId: 'plan-api-2',
          agentNames: ['web-agent'],
          tasks: [{ taskId: 't2', agentName: 'web-agent', content: 'Build UI' }],
          strategy: 'single',
          status: 'pending',
        },
      },
    })

    const { confirmPlan } = useMessageStore.getState()
    await confirmPlan('conv-api-2', 'run-api-2', 'plan-api-2', false, 'not needed right now')

    expect(confirmHITL).toHaveBeenCalledWith({
      runId: 'run-api-2',
      actionId: 'plan-api-2',
      confirmed: false,
      rejectReason: 'not needed right now',
    })

    const confirmation = useMessageStore.getState().getConfirmation('conv-api-2')
    expect(confirmation?.status).toBe('rejected')
    expect(confirmation?.rejectReason).toBe('not needed right now')
  })

  it('confirmPlan handles API errors with sanitized message', async () => {
    const { confirmHITL } = await import('../services/api')
    vi.mocked(confirmHITL).mockRejectedValueOnce(
      new Error('orchestrator at http://internal:8090 failed: token=' + 'sk-' + 'abc123def456ghi789jkl012'),
    )

    useMessageStore.setState({
      confirmationByConversation: {
        'conv-api-3': {
          runId: 'run-api-3',
          actionId: 'plan-api-3',
          agentNames: ['code-agent'],
          tasks: [],
          strategy: 'single',
          status: 'pending',
        },
      },
    })

    const { confirmPlan } = useMessageStore.getState()
    await expect(
      confirmPlan('conv-api-3', 'run-api-3', 'plan-api-3', true),
    ).rejects.toThrow()

    const confirmation = useMessageStore.getState().getConfirmation('conv-api-3')
    expect(confirmation?.status).toBe('pending') // still pending after error
    expect(confirmation?.error).toBeDefined()
    // Error must be sanitized
    expect(confirmation?.error).not.toContain('internal:8090')
    expect(confirmation?.error).not.toContain('sk-abc123')
  })

  it('clearConfirmation removes confirmation state', () => {
    useMessageStore.setState({
      confirmationByConversation: {
        'conv-clear': {
          runId: 'run-clear',
          actionId: 'plan-clear',
          agentNames: ['code-agent'],
          tasks: [],
          status: 'confirmed',
        },
      },
    })

    const { clearConfirmation } = useMessageStore.getState()
    clearConfirmation('conv-clear')

    expect(useMessageStore.getState().getConfirmation('conv-clear')).toBeNull()
  })

  it('getConfirmation returns null for unknown conversation', () => {
    const confirmation = useMessageStore.getState().getConfirmation('nonexistent')
    expect(confirmation).toBeNull()
  })
})

