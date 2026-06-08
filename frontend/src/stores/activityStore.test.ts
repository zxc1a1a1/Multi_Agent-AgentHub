import { describe, expect, it, beforeEach } from 'vitest'
import { useActivityStore } from './activityStore'
import type { ActivitySnapshot } from '../types'

const baseActivity: ActivitySnapshot = {
  activityId: 'plan-test-1',
  activityType: 'plan_approval',
  status: 'awaiting_confirmation',
  executionPath: 'single_chat',
  planId: 'plan-test-1',
  revision: 1,
  planOwner: { type: 'agent', agentName: 'code-agent' },
  participants: [{ agentName: 'code-agent', required: true, selected: true }],
  candidateParticipants: [],
  defaultSelectedParticipants: [],
  requiredParticipants: [],
  title: 'Test Plan',
  summary: 'A test plan',
  tasks: [{ taskId: 't1', agentName: 'code-agent', content: 'Write code' }],
  allowedActions: ['approve', 'cancel', 'revise'],
  warnings: [],
}

describe('activityStore', () => {
  beforeEach(() => {
    useActivityStore.setState({ pendingActivityByConversation: {} })
  })

  it('setActivity stores activity for a conversation', () => {
    useActivityStore.getState().setActivity('conv-1', baseActivity)
    const state = useActivityStore.getState()
    expect(state.pendingActivityByConversation['conv-1']).toEqual(baseActivity)
  })

  it('setActivity overwrites previous activity for same conversation', () => {
    const revised: ActivitySnapshot = {
      ...baseActivity,
      revision: 2,
      planId: 'plan-v2',
      summary: 'Revised plan',
    }
    useActivityStore.getState().setActivity('conv-1', baseActivity)
    useActivityStore.getState().setActivity('conv-1', revised)
    const state = useActivityStore.getState()
    expect(state.pendingActivityByConversation['conv-1'].revision).toBe(2)
    expect(state.pendingActivityByConversation['conv-1'].planId).toBe('plan-v2')
  })

  it('setActivity supports multiple conversations independently', () => {
    useActivityStore.getState().setActivity('conv-1', baseActivity)
    useActivityStore.getState().setActivity('conv-2', {
      ...baseActivity,
      planId: 'plan-other',
      executionPath: 'group_chat',
    })
    const state = useActivityStore.getState()
    expect(state.pendingActivityByConversation['conv-1'].planId).toBe('plan-test-1')
    expect(state.pendingActivityByConversation['conv-2'].planId).toBe('plan-other')
    expect(state.pendingActivityByConversation['conv-2'].executionPath).toBe('group_chat')
  })

  it('updateActivityStatus changes only the status field', () => {
    useActivityStore.getState().setActivity('conv-1', baseActivity)
    useActivityStore.getState().updateActivityStatus('conv-1', 'executing')
    const state = useActivityStore.getState()
    expect(state.pendingActivityByConversation['conv-1'].status).toBe('executing')
    expect(state.pendingActivityByConversation['conv-1'].planId).toBe('plan-test-1')
  })

  it('updateActivityStatus is a no-op for unknown conversation', () => {
    useActivityStore.getState().updateActivityStatus('nonexistent', 'executing')
    const state = useActivityStore.getState()
    expect(state.pendingActivityByConversation['nonexistent']).toBeUndefined()
  })

  it('clearActivity removes activity for a conversation', () => {
    useActivityStore.getState().setActivity('conv-1', baseActivity)
    useActivityStore.getState().clearActivity('conv-1')
    const state = useActivityStore.getState()
    expect(state.pendingActivityByConversation['conv-1']).toBeUndefined()
  })

  it('clearActivity does not affect other conversations', () => {
    useActivityStore.getState().setActivity('conv-1', baseActivity)
    useActivityStore.getState().setActivity('conv-2', { ...baseActivity, planId: 'plan-2' })
    useActivityStore.getState().clearActivity('conv-1')
    const state = useActivityStore.getState()
    expect(state.pendingActivityByConversation['conv-1']).toBeUndefined()
    expect(state.pendingActivityByConversation['conv-2']).toBeDefined()
    expect(state.pendingActivityByConversation['conv-2'].planId).toBe('plan-2')
  })
})
