import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useMessageStore } from './messageStore'
import * as api from '../services/api'

vi.mock('../services/api', () => ({
  listMessages: vi.fn().mockResolvedValue([]),
}))

describe('message replay (page refresh recovery)', () => {
  beforeEach(() => {
    useMessageStore.setState({
      messages: {},
      streamingByConversation: {},
      abortControllersByConversation: {},
    })
  })

  it('restores 3 separate agent messages from replay data after refresh', async () => {
    const replayData = [
      {
        id: 'db-msg-1', conversationId: 'conv-replay',
        senderType: 'user', senderDisplayName: '',
        role: 'user', content: 'build a login page and go api',
        text: 'build a login page and go api',
        createdAt: '2026-06-04T00:00:00Z',
      },
      {
        id: 'db-msg-2', conversationId: 'conv-replay',
        runId: 'run-001', sseMessageId: 'msg-web',
        senderType: 'agent', senderName: 'web-agent',
        senderDisplayName: 'Web Agent', agentName: 'web-agent',
        role: 'assistant', content: '<section><h1>Login Page</h1></section>',
        text: '<section><h1>Login Page</h1></section>',
        status: 'sent', createdAt: '2026-06-04T00:00:01Z',
      },
      {
        id: 'db-msg-3', conversationId: 'conv-replay',
        runId: 'run-001', sseMessageId: 'msg-code',
        senderType: 'agent', senderName: 'code-agent',
        senderDisplayName: 'Code Agent', agentName: 'code-agent',
        role: 'assistant', content: 'package main\n\nimport "net/http"',
        text: 'package main\n\nimport "net/http"',
        status: 'sent', createdAt: '2026-06-04T00:00:02Z',
      },
      {
        id: 'db-msg-4', conversationId: 'conv-replay',
        runId: 'run-001', sseMessageId: 'msg-summary',
        senderType: 'agent', senderName: 'orchestrator',
        senderDisplayName: 'Orchestrator', agentName: 'orchestrator',
        role: 'assistant', content: 'All 2 task(s) completed successfully.',
        text: 'All 2 task(s) completed successfully.',
        status: 'sent', createdAt: '2026-06-04T00:00:03Z',
      },
    ]

    vi.mocked(api.listMessages).mockResolvedValueOnce(replayData as any)

    const { loadMessages } = useMessageStore.getState()
    await loadMessages('conv-replay')

    const messages = useMessageStore.getState().messages['conv-replay'] || []

    // 1 user + 3 agent messages = 4 total
    expect(messages).toHaveLength(4)

    const userMsg = messages[0]
    expect(userMsg.senderType).toBe('user')
    expect(userMsg.content).toBe('build a login page and go api')

    const webMsg = messages[1]
    expect(webMsg.senderType).toBe('agent')
    expect(webMsg.senderName).toBe('Web Agent')
    expect(webMsg.agentName).toBe('web-agent')
    expect(webMsg.content).toContain('<section><h1>Login Page</h1>')
    expect(webMsg.runId).toBe('run-001')
    expect(webMsg.sseMessageId).toBe('msg-web')

    const codeMsg = messages[2]
    expect(codeMsg.senderName).toBe('Code Agent')
    expect(codeMsg.agentName).toBe('code-agent')
    expect(codeMsg.content).toContain('package main')

    const orchMsg = messages[3]
    expect(orchMsg.senderName).toBe('Orchestrator')
    expect(orchMsg.agentName).toBe('orchestrator')
    expect(orchMsg.content).toContain('All 2 task(s) completed')
  })

  it('resolves senderName from senderDisplayName when available', async () => {
    const replayData = [
      {
        id: 'msg-1', conversationId: 'conv-disp',
        senderType: 'agent', senderName: 'web-agent',
        senderDisplayName: 'Web Agent', role: 'assistant',
        content: 'web output', createdAt: '2026-01-01T00:00:00Z',
      },
    ]

    vi.mocked(api.listMessages).mockResolvedValueOnce(replayData as any)

    const { loadMessages } = useMessageStore.getState()
    await loadMessages('conv-disp')

    const messages = useMessageStore.getState().messages['conv-disp'] || []
    expect(messages).toHaveLength(1)
    expect(messages[0].senderName).toBe('Web Agent')
  })

  it('falls back senderName to author for backward compat', async () => {
    const replayData = [
      {
        id: 'msg-legacy', conversationId: 'conv-legacy',
        author: 'assistant', role: 'assistant',
        text: 'legacy merged response', createdAt: '2026-01-01T00:00:00Z',
      },
    ]

    vi.mocked(api.listMessages).mockResolvedValueOnce(replayData as any)

    const { loadMessages } = useMessageStore.getState()
    await loadMessages('conv-legacy')

    const messages = useMessageStore.getState().messages['conv-legacy'] || []
    expect(messages).toHaveLength(1)
    expect(messages[0].senderType).toBe('agent')
    expect(messages[0].senderName).toBe('assistant')
    expect(messages[0].content).toBe('legacy merged response')
  })

  it('carries failed status and error fields from replay', async () => {
    const replayData = [
      {
        id: 'msg-fail', conversationId: 'conv-err-replay',
        senderType: 'agent', senderName: 'code-agent',
        senderDisplayName: 'Code Agent', role: 'assistant',
        content: 'partial', status: 'failed',
        errorCode: 'AGUI_INTERNAL', errorMessage: 'internal error',
        createdAt: '2026-01-01T00:00:00Z',
      },
    ]

    vi.mocked(api.listMessages).mockResolvedValueOnce(replayData as any)

    const { loadMessages } = useMessageStore.getState()
    await loadMessages('conv-err-replay')

    const messages = useMessageStore.getState().messages['conv-err-replay'] || []
    expect(messages).toHaveLength(1)
    expect(messages[0].status).toBe('failed')
    expect(messages[0].errorCode).toBe('AGUI_INTERNAL')
    expect(messages[0].errorMessage).toBe('internal error')
  })

  it('carries runId, stepId, and sseMessageId from replay', async () => {
    const replayData = [
      {
        id: 'msg-link', conversationId: 'conv-link',
        senderType: 'agent', senderName: 'web-agent',
        senderDisplayName: 'Web Agent', role: 'assistant',
        content: 'hello', runId: 'run-abc', stepId: 'step-1',
        sseMessageId: 'msg-web', status: 'sent',
        createdAt: '2026-01-01T00:00:00Z',
      },
    ]

    vi.mocked(api.listMessages).mockResolvedValueOnce(replayData as any)

    const { loadMessages } = useMessageStore.getState()
    await loadMessages('conv-link')

    const messages = useMessageStore.getState().messages['conv-link'] || []
    expect(messages).toHaveLength(1)
    expect(messages[0].runId).toBe('run-abc')
    expect(messages[0].stepId).toBe('step-1')
    expect(messages[0].sseMessageId).toBe('msg-web')
  })
})
