import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useMessageStore } from './messageStore'
import type { ActiveRunCursor } from '../agui/client'
import type { ActivitySnapshot } from '../types'

const mocks = vi.hoisted(() => ({
  active: null as ActiveRunCursor | null,
  listMessages: vi.fn(),
  recoverRun: vi.fn(),
  getRunStatus: vi.fn(),
  loadActiveRun: vi.fn(),
  saveActiveRun: vi.fn(),
  clearActiveRun: vi.fn(),
}))

vi.mock('../services/api', () => ({
  listMessages: mocks.listMessages,
  confirmHITL: vi.fn().mockResolvedValue(undefined),
  regenerateMessage: vi.fn(),
}))

vi.mock('../agui/client', () => ({
  runAgent: vi.fn(),
  recoverRun: mocks.recoverRun,
  getRunStatus: mocks.getRunStatus,
  loadActiveRun: mocks.loadActiveRun,
  saveActiveRun: mocks.saveActiveRun,
  clearActiveRun: mocks.clearActiveRun,
}))

const conversationId = 'conv-resume'
const userMessage = {
  id: 'u-1',
  conversationId,
  senderType: 'user',
  content: 'hello',
  createdAt: '2026-01-01T00:00:00Z',
}
const assistantMessage = {
  id: 'a-1',
  conversationId,
  senderType: 'agent',
  senderName: 'code-agent',
  content: 'persisted answer',
  createdAt: '2026-01-01T00:00:01Z',
}

function planActivity(): ActivitySnapshot {
  return {
    activityId: 'act-1',
    activityType: 'plan_approval',
    status: 'awaiting_confirmation',
    executionPath: 'main_agent_orchestration',
    planId: 'plan-1',
    revision: 1,
    participants: [{ agentName: 'code-agent', required: true, selected: true }],
    tasks: [{ taskId: 'task-1', agentName: 'code-agent', content: 'write code' }],
    allowedActions: ['approve', 'cancel'],
  }
}

beforeEach(() => {
  mocks.active = null
  mocks.listMessages.mockReset()
  mocks.recoverRun.mockReset()
  mocks.getRunStatus.mockReset()
  mocks.loadActiveRun.mockReset()
  mocks.saveActiveRun.mockReset()
  mocks.clearActiveRun.mockReset()
  mocks.loadActiveRun.mockImplementation(() => mocks.active)
  mocks.clearActiveRun.mockImplementation(() => {
    mocks.active = null
  })
  useMessageStore.setState({
    messages: {},
    streamingByConversation: {},
    abortControllersByConversation: {},
    orchestrationByConversation: {},
    confirmationByConversation: {},
    pinnedMessageIdsByConversation: {},
  })
})

describe('resumeActiveRun', () => {
  it('replays a paused run on load and restores its plan card', async () => {
    mocks.active = {
      runId: 'run-paused',
      conversationId,
      seq: 3,
      subIndex: 1,
      updatedAt: Date.now(),
    }
    mocks.listMessages.mockResolvedValue([userMessage])
    mocks.getRunStatus.mockResolvedValue('awaiting_approval')
    mocks.recoverRun.mockImplementation(async (runId: string, _seq, _sub, onEvent) => {
      onEvent({ type: 'ACTIVITY_SNAPSHOT', runId, seq: 4, activity: planActivity() })
    })

    await useMessageStore.getState().loadMessages(conversationId)
    await vi.waitFor(() => {
      expect(useMessageStore.getState().confirmationByConversation[conversationId]?.planId).toBe('plan-1')
    })
    expect(mocks.recoverRun).toHaveBeenCalledWith(
      'run-paused',
      0,
      0,
      expect.any(Function),
      expect.any(Function),
    )
    expect(mocks.clearActiveRun).not.toHaveBeenCalled()
  })

  it('replays to completion and reloads persisted history', async () => {
    mocks.active = {
      runId: 'run-live',
      conversationId,
      seq: 2,
      subIndex: 0,
      updatedAt: Date.now(),
    }
    mocks.listMessages.mockResolvedValueOnce([userMessage]).mockResolvedValue([userMessage, assistantMessage])
    mocks.getRunStatus.mockResolvedValue('executing')
    mocks.recoverRun.mockImplementation(async (runId: string, _seq, _sub, onEvent, onComplete) => {
      onEvent({ type: 'TEXT_MESSAGE_CONTENT', runId, seq: 3, delta: 'replayed' })
      onComplete?.()
    })

    await useMessageStore.getState().loadMessages(conversationId)
    await vi.waitFor(() => {
      expect(
        useMessageStore.getState().messages[conversationId]?.some((m) => m.content === 'persisted answer'),
      ).toBe(true)
    })
    expect(mocks.clearActiveRun).toHaveBeenCalled()
  })

  it('follows a recovered run to completion after approval', async () => {
    mocks.active = {
      runId: 'run-follow',
      conversationId,
      seq: 4,
      subIndex: 0,
      updatedAt: Date.now(),
    }
    mocks.getRunStatus.mockResolvedValueOnce('executing').mockResolvedValue('completed')
    mocks.listMessages.mockResolvedValue([userMessage, assistantMessage])
    useMessageStore.setState({
      confirmationByConversation: {
        [conversationId]: {
          runId: 'run-follow',
          actionId: 'plan-1',
          planId: 'plan-1',
          agentNames: [],
          tasks: [],
          status: 'pending',
        },
      },
    })

    await useMessageStore.getState().confirmPlan(conversationId, 'run-follow', 'plan-1', true)

    await vi.waitFor(() => {
      expect(
        useMessageStore.getState().messages[conversationId]?.some((m) => m.content === 'persisted answer'),
      ).toBe(true)
    })
    expect(mocks.clearActiveRun).toHaveBeenCalled()
  })

  it('clears a finished run without replaying it', async () => {
    mocks.active = {
      runId: 'run-done',
      conversationId,
      seq: 9,
      subIndex: 0,
      updatedAt: Date.now(),
    }
    mocks.listMessages.mockResolvedValueOnce([userMessage]).mockResolvedValue([userMessage, assistantMessage])
    mocks.getRunStatus.mockResolvedValue('completed')

    await useMessageStore.getState().loadMessages(conversationId)
    await vi.waitFor(() => {
      expect(
        useMessageStore.getState().messages[conversationId]?.some((m) => m.content === 'persisted answer'),
      ).toBe(true)
    })
    expect(mocks.recoverRun).not.toHaveBeenCalled()
    expect(mocks.clearActiveRun).toHaveBeenCalled()
  })
})
