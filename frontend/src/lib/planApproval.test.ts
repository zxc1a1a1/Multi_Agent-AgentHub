import { describe, expect, it } from 'vitest'
import { normalizePlanApprovalData } from './planApproval'
import type { PendingConfirmation } from '../stores/messageStore'

function buildConfirmation(
  overrides: Partial<PendingConfirmation> = {},
): PendingConfirmation {
  return {
    runId: 'run-1',
    actionId: 'plan-1',
    agentNames: ['code-agent'],
    tasks: [
      {
        taskId: 't1',
        agentName: 'code-agent',
        content: 'PLAN ONLY CONTENT - DISPLAY TO USER',
        priority: 1,
        riskLevel: 'low',
      },
    ],
    intentSummary: 'Test plan summary',
    strategy: 'single',
    status: 'pending',
    ...overrides,
  }
}

describe('normalizePlanApprovalData', () => {
  // 1. Phase 2 single_chat confirm_plan data
  it('parses Phase 2 single_chat confirm_plan data with proposal content', () => {
    const conf = buildConfirmation({
      executionPath: 'single_chat',
      tasks: [
        { taskId: 't1', agentName: 'code-agent', content: 'PLAN ONLY CONTENT - DISPLAY TO USER' },
      ],
      planOwner: { type: 'agent', agentName: 'code-agent' },
    })
    const data = normalizePlanApprovalData(conf)

    expect(data.executionPath).toBe('single_chat')
    expect(data.tasks).toHaveLength(1)
    expect(data.tasks[0].content).toBe('PLAN ONLY CONTENT - DISPLAY TO USER')
    expect(data.participants).toHaveLength(1)
    expect(data.participants[0].agentName).toBe('code-agent')
    expect(data.title).toBe('Test plan summary')
    expect(data.summary).toBe('Test plan summary')
    expect(data.planOwner).toEqual({ type: 'agent', agentName: 'code-agent' })
  })

  // 2. Legacy HITL data fallback
  it('handles legacy data with missing executionPath', () => {
    const conf = buildConfirmation({
      executionPath: undefined,
      tasks: [],
      intentSummary: '',
    })
    const data = normalizePlanApprovalData(conf)

    expect(data.executionPath).toBe('unknown')
    expect(data.title).toBe('确认执行方案')
    expect(data.tasks).toHaveLength(0)
    expect(data.summary).toBe('')
  })

  // 3. Missing tasks does not crash
  it('handles missing tasks gracefully', () => {
    const conf = buildConfirmation({ tasks: [] })
    const data = normalizePlanApprovalData(conf)
    expect(data.tasks).toHaveLength(0)
  })

  // 4. warnings display correctly
  it('shows warnings when present', () => {
    const conf = buildConfirmation({ warnings: ['risk: high', 'note: untested'] })
    const data = normalizePlanApprovalData(conf)
    expect(data.warnings).toHaveLength(2)
    expect(data.warnings[0]).toBe('risk: high')
  })

  it('omits warnings area when empty', () => {
    const conf = buildConfirmation()
    const data = normalizePlanApprovalData(conf)
    expect(data.warnings).toHaveLength(0)
  })

  // 5. participants deduplication
  it('deduplicates participants when multiple tasks share the same agent', () => {
    const conf = buildConfirmation({
      tasks: [
        { taskId: 't1', agentName: 'code-agent', content: 'write code' },
        { taskId: 't2', agentName: 'code-agent', content: 'test code' },
        { taskId: 't3', agentName: 'web-agent', content: 'render UI' },
      ],
    })
    const data = normalizePlanApprovalData(conf)
    expect(data.participants).toHaveLength(2)
    expect(data.participants.map((p) => p.agentName)).toEqual(['code-agent', 'web-agent'])
  })

  // 6. participants from explicit participants field
  it('uses explicit participants field when present', () => {
    const conf = buildConfirmation({
      participants: [
        { agentName: 'code-agent', required: true, selected: true },
        { agentName: 'web-agent', required: false },
      ],
    })
    const data = normalizePlanApprovalData(conf)
    expect(data.participants).toHaveLength(2)
    expect(data.participants[0].required).toBe(true)
    expect(data.participants[1].required).toBe(false)
  })

  // 7. planOwner from agentNames fallback
  it('infers planOwner from single agentNames', () => {
    const conf = buildConfirmation({
      agentNames: ['code-agent'],
      planOwner: undefined,
    })
    const data = normalizePlanApprovalData(conf)
    expect(data.planOwner).toEqual({ type: 'agent', agentName: 'code-agent' })
  })

  // 8. unknown executionPath handled
  it('maps unknown executionPath values to unknown', () => {
    const conf = buildConfirmation({ executionPath: 'invalid_path' })
    const data = normalizePlanApprovalData(conf)
    expect(data.executionPath).toBe('unknown')
  })

  // 9. long summary truncated in title
  it('truncates long summaries in title', () => {
    const longSummary = 'A'.repeat(100)
    const conf = buildConfirmation({ intentSummary: longSummary })
    const data = normalizePlanApprovalData(conf)
    expect(data.title!.length).toBeLessThanOrEqual(83) // 80 + '...'
    expect(data.title!.endsWith('...')).toBe(true)
  })

  // 10. revision and planId pass through
  it('passes through revision and planId', () => {
    const conf = buildConfirmation({
      planId: 'plan-abc',
      revision: 3,
    })
    const data = normalizePlanApprovalData(conf)
    expect(data.planId).toBe('plan-abc')
    expect(data.revision).toBe(3)
  })

  // 11. unknown executionPath falls back
  it('produces valid data for completely unknown input', () => {
    const conf: PendingConfirmation = {
      runId: 'r',
      actionId: 'a',
      agentNames: [],
      tasks: [],
      status: 'pending',
    }
    const data = normalizePlanApprovalData(conf)
    expect(data.executionPath).toBe('unknown')
    expect(data.title).toBe('确认执行方案')
    expect(data.warnings).toHaveLength(0)
    expect(data.participants).toHaveLength(0)
    expect(data.tasks).toHaveLength(0)
  })
})
