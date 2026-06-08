import { render, screen, fireEvent, waitFor, act } from '@testing-library/react'
import { describe, expect, it, vi, beforeEach, beforeAll } from 'vitest'
import ChatWindow from './ChatWindow'
import { useMessageStore } from '../stores/messageStore'
import { useConversationStore } from '../stores/conversationStore'
import { useAgentStore } from '../stores/agentStore'
import { useActivityStore } from '../stores/activityStore'
import type { ActivitySnapshot } from '../types'

// Mock useSendMessage to avoid SSE side effects
vi.mock('../agui/events', () => ({
  useSendMessage: () => ({
    sendMessage: vi.fn(),
    isStreaming: () => false,
    stopStreaming: vi.fn(),
  }),
}))

// jsdom polyfill
beforeAll(() => {
  if (!HTMLElement.prototype.scrollIntoView) {
    HTMLElement.prototype.scrollIntoView = vi.fn() as unknown as typeof HTMLElement.prototype.scrollIntoView
  }
})

const CONV_ID = 'conv-test-1'

function seedPendingConfirmation(overrides: Record<string, unknown> = {}) {
  useMessageStore.setState({
    messages: { [CONV_ID]: [] },
    confirmationByConversation: {
      [CONV_ID]: {
        runId: 'run-test-1',
        actionId: 'plan-test-1',
        planId: 'plan-test-1',
        revision: 1,
        executionPath: 'single_chat',
        agentNames: ['code-agent'],
        planOwner: { type: 'agent', agentName: 'code-agent' },
        tasks: [
          {
            taskId: 't1',
            agentName: 'code-agent',
            content: 'Write Go HTTP server code',
            priority: 1,
            riskLevel: 'low',
          },
          {
            taskId: 't2',
            agentName: 'code-agent',
            content: 'Add unit tests',
            priority: 2,
            riskLevel: 'medium',
          },
        ],
        participants: [
          { agentName: 'code-agent', required: true, selected: true },
        ],
        intentSummary: 'Test confirmation plan',
        strategy: 'single',
        warnings: [],
        status: 'pending',
        ...overrides,
      },
    },
  })
  // Also seed activityStore so ActivitySnapshotRenderer can read the data.
  useActivityStore.setState({
    pendingActivityByConversation: {
      [CONV_ID]: {
        activityId: (overrides.planId as string) || 'plan-test-1',
        activityType: 'plan_approval',
        status: 'awaiting_confirmation',
        executionPath: (overrides.executionPath as string) || 'single_chat',
        planId: (overrides.planId as string) || 'plan-test-1',
        revision: (overrides.revision as number) || 1,
        planOwner: (overrides.planOwner as ActivitySnapshot['planOwner']) || { type: 'agent', agentName: 'code-agent' },
        participants: (overrides.participants as ActivitySnapshot['participants']) || [
          { agentName: 'code-agent', required: true, selected: true },
        ],
        candidateParticipants: (overrides.candidateParticipants as ActivitySnapshot['candidateParticipants']) || [],
        defaultSelectedParticipants: (overrides.defaultSelectedParticipants as ActivitySnapshot['defaultSelectedParticipants']) || [],
        requiredParticipants: (overrides.requiredParticipants as string[]) || [],
        title: (overrides.intentSummary as string) || 'Test confirmation plan',
        summary: (overrides.intentSummary as string) || 'Test confirmation plan',
        tasks: (overrides.tasks as ActivitySnapshot['tasks']) || [
          { taskId: 't1', agentName: 'code-agent', content: 'Write Go HTTP server code', priority: 1, riskLevel: 'low' },
          { taskId: 't2', agentName: 'code-agent', content: 'Add unit tests', priority: 2, riskLevel: 'medium' },
        ],
        allowedActions: ['approve', 'cancel', 'revise'],
        warnings: (overrides.warnings as string[]) || [],
      },
    },
  })
}

function seedConversation() {
  useConversationStore.setState({
    conversations: [
      {
        id: CONV_ID,
        title: 'Test Conversation',
        agentName: 'code-agent',
        createdAt: '2026-06-01T00:00:00Z',
        updatedAt: '2026-06-08T00:00:00Z',
      },
    ],
    activeId: CONV_ID,
    loading: false,
  })
}

describe('ChatWindow — HITL PlanApprovalCard integration', () => {
  beforeEach(() => {
    // Reset stores
    useMessageStore.setState({
      messages: {},
      confirmationByConversation: {},
      streamingByConversation: {},
      abortControllersByConversation: {},
      orchestrationByConversation: {},
    })
    useConversationStore.setState({
      conversations: [],
      activeId: null,
      loading: false,
    })
    // Seed agent store with known options so the agent selector renders
    useAgentStore.setState({
      options: [
        { name: 'auto', displayName: 'Auto (Smart)', description: 'Auto select', outputModes: ['text', 'code'] },
        { name: 'code-agent', displayName: 'Code Agent', description: 'Code generation', outputModes: ['text', 'code'] },
        { name: 'web-agent', displayName: 'Web Agent', description: 'Web development', outputModes: ['text', 'webpage'] },
      ],
      loading: false,
      loaded: true,
    })
  })

  it('renders PlanApprovalCard when pending confirmation exists', () => {
    seedConversation()
    seedPendingConfirmation()

    render(<ChatWindow conversationId={CONV_ID} />)

    // Card header (use heading role to avoid duplicate text matches)
    const title = screen.getByRole('heading', { name: 'Test confirmation plan' })
    expect(title).toBeInTheDocument()
    expect(title.tagName).toBe('H3')
    // Single Agent badge
    expect(screen.getByText('Single Agent')).toBeInTheDocument()
    // Task content
    expect(screen.getByText('Write Go HTTP server code')).toBeInTheDocument()
    expect(screen.getByText('Add unit tests')).toBeInTheDocument()
    // Action buttons
    expect(screen.getByText('同意并执行')).toBeInTheDocument()
    expect(screen.getByText('取消')).toBeInTheDocument()
  })

  it('clicking approve calls confirmPlan with confirmed=true', async () => {
    const confirmPlanSpy = vi.spyOn(useMessageStore.getState(), 'confirmPlan')
      .mockResolvedValue(undefined)

    seedConversation()
    seedPendingConfirmation()

    render(<ChatWindow conversationId={CONV_ID} />)

    fireEvent.click(screen.getByText('同意并执行'))

    await waitFor(() => {
      expect(confirmPlanSpy).toHaveBeenCalledWith(
        CONV_ID,
        'run-test-1',
        'plan-test-1',
        true,
        undefined,
        undefined,
        undefined,
        undefined,
        undefined,
      )
    })

    confirmPlanSpy.mockRestore()
  })

  it('clicking cancel calls confirmPlan with confirmed=false and reason', async () => {
    const confirmPlanSpy = vi.spyOn(useMessageStore.getState(), 'confirmPlan')
      .mockResolvedValue(undefined)

    seedConversation()
    seedPendingConfirmation()

    render(<ChatWindow conversationId={CONV_ID} />)

    fireEvent.click(screen.getByText('取消'))

    await waitFor(() => {
      expect(confirmPlanSpy).toHaveBeenCalledWith(
        CONV_ID,
        'run-test-1',
        'plan-test-1',
        false,
        'User cancelled',
      )
    })

    confirmPlanSpy.mockRestore()
  })

  it('does not render PlanApprovalCard when no pending confirmation', () => {
    seedConversation()
    // No confirmation seeded

    render(<ChatWindow conversationId={CONV_ID} />)

    expect(screen.queryByText('同意并执行')).not.toBeInTheDocument()
    expect(screen.queryByText('取消')).not.toBeInTheDocument()
  })

  it('renders confirmed status banner when status is confirmed', () => {
    seedConversation()
    seedPendingConfirmation({ status: 'confirmed' })

    render(<ChatWindow conversationId={CONV_ID} />)

    expect(screen.getByText(/Plan confirmed/)).toBeInTheDocument()
    // Action buttons should not be visible when confirmed
    expect(screen.queryByText('同意并执行')).not.toBeInTheDocument()
  })

  it('renders rejected status banner when status is rejected', () => {
    seedConversation()
    seedPendingConfirmation({ status: 'rejected', rejectReason: 'too complex' })

    render(<ChatWindow conversationId={CONV_ID} />)

    expect(screen.getByText(/Plan rejected/)).toBeInTheDocument()
    expect(screen.getByText(/too complex/)).toBeInTheDocument()
  })

  it('calls confirmPlan with action=revise when onRevise triggered', async () => {
    const confirmPlanSpy = vi.spyOn(useMessageStore.getState(), 'confirmPlan')
      .mockResolvedValue(undefined)

    seedConversation()
    seedPendingConfirmation({ revision: 1 })

    render(<ChatWindow conversationId={CONV_ID} />)

    // Click the revise button to show feedback input
    fireEvent.click(screen.getByText('提出修改意见'))

    // Type feedback in the textarea
    const textarea = screen.getByPlaceholderText(/请输入你希望如何修改这个方案/)
    fireEvent.change(textarea, { target: { value: '简化步骤' } })

    // Click submit
    fireEvent.click(screen.getByText('提交修改意见'))

    await waitFor(() => {
      expect(confirmPlanSpy).toHaveBeenCalledWith(
        CONV_ID,
        'run-test-1',
        'plan-test-1',
        false,
        '',
        'revise',
        '简化步骤',
        1,
        undefined,
      )
    })

    confirmPlanSpy.mockRestore()
  })

  it('confirmStatus resets to waiting when new pending confirmation arrives', async () => {
    seedConversation()
    seedPendingConfirmation({ status: 'pending', revision: 1, planId: 'plan-v1' })

    render(<ChatWindow conversationId={CONV_ID} />)

    // Verify PlanApprovalCard is visible with plan-v1 content
    expect(screen.getByText('Write Go HTTP server code')).toBeInTheDocument()

    // Simulate a revised plan arriving (new TOOL_CALL_END with different planId)
    // by directly updating confirmationByConversation to a new pending state.
    // Wrap in act() so React processes the resulting useEffect.
    act(() => {
      useMessageStore.setState({
        confirmationByConversation: {
          [CONV_ID]: {
            runId: 'run-test-1',
            actionId: 'plan-v2',
            planId: 'plan-v2',
            revision: 2,
            executionPath: 'single_chat',
            agentNames: ['code-agent'],
            planOwner: { type: 'agent', agentName: 'code-agent' },
            tasks: [
              {
                taskId: 't1',
                agentName: 'code-agent',
                content: 'Write a simple Go HTTP server',
                priority: 1,
                riskLevel: 'low',
              },
            ],
            participants: [
              { agentName: 'code-agent', required: true, selected: true },
            ],
            intentSummary: 'Revised test plan',
            strategy: 'single',
            warnings: [],
            status: 'pending',
          },
        },
      })
      useActivityStore.setState({
        pendingActivityByConversation: {
          [CONV_ID]: {
            activityId: 'plan-v2',
            activityType: 'plan_approval',
            status: 'awaiting_confirmation',
            executionPath: 'single_chat',
            planId: 'plan-v2',
            revision: 2,
            planOwner: { type: 'agent', agentName: 'code-agent' },
            participants: [
              { agentName: 'code-agent', required: true, selected: true },
            ],
            candidateParticipants: [],
            defaultSelectedParticipants: [],
            requiredParticipants: [],
            title: 'Revised test plan',
            summary: 'Revised test plan',
            tasks: [
              { taskId: 't1', agentName: 'code-agent', content: 'Write a simple Go HTTP server', priority: 1, riskLevel: 'low' },
            ],
            allowedActions: ['approve', 'cancel', 'revise'],
            warnings: [],
          },
        },
      })
    })

    // The useEffect should have reset confirmStatus to 'waiting'
    // The new plan content should be visible after React re-renders.
    await waitFor(() => {
      expect(screen.getByText('Write a simple Go HTTP server')).toBeInTheDocument()
    })
    // The old plan content should NOT be visible anymore (overwritten)
    expect(screen.queryByText('Write Go HTTP server code')).not.toBeInTheDocument()
  })
})
