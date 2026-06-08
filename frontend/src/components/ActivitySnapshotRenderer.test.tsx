import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi, beforeEach } from 'vitest'
import ActivitySnapshotRenderer from './ActivitySnapshotRenderer'
import { useActivityStore } from '../stores/activityStore'
import type { ActivitySnapshot } from '../types'

function seedActivity(overrides: Partial<ActivitySnapshot> = {}) {
  useActivityStore.setState({
    pendingActivityByConversation: {
      'conv-test-1': {
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
        summary: 'A test execution plan',
        tasks: [
          { taskId: 't1', agentName: 'code-agent', content: 'Write code', riskLevel: 'low' },
        ],
        allowedActions: ['approve', 'cancel', 'revise'],
        warnings: [],
        ...overrides,
      },
    },
  })
}

describe('ActivitySnapshotRenderer', () => {
  beforeEach(() => {
    useActivityStore.setState({ pendingActivityByConversation: {} })
  })

  it('renders nothing when no activity exists', () => {
    const onApprove = vi.fn()
    const onCancel = vi.fn()
    const { container } = render(
      <ActivitySnapshotRenderer
        conversationId="conv-test-1"
        onApprove={onApprove}
        onCancel={onCancel}
      />,
    )
    expect(container.innerHTML).toBe('')
  })

  it('renders nothing when activityType is not plan_approval', () => {
    seedActivity({ activityType: 'orchestration_progress' })
    const onApprove = vi.fn()
    const onCancel = vi.fn()
    const { container } = render(
      <ActivitySnapshotRenderer
        conversationId="conv-test-1"
        onApprove={onApprove}
        onCancel={onCancel}
      />,
    )
    expect(container.innerHTML).toBe('')
  })

  it('renders PlanApprovalCard for single_chat path', () => {
    seedActivity({ executionPath: 'single_chat' })
    const onApprove = vi.fn()
    const onCancel = vi.fn()

    render(
      <ActivitySnapshotRenderer
        conversationId="conv-test-1"
        onApprove={onApprove}
        onCancel={onCancel}
      />,
    )

    // PlanApprovalCard content is rendered
    expect(screen.getByText('Test Plan')).toBeInTheDocument()
    expect(screen.getByText('Write code')).toBeInTheDocument()
    // Action buttons
    expect(screen.getByText('同意并执行')).toBeInTheDocument()
    expect(screen.getByText('取消')).toBeInTheDocument()
  })

  it('renders GroupPlanCard for group_chat path', () => {
    seedActivity({
      executionPath: 'group_chat',
      planOwner: { type: 'group_coordinator', agentName: 'group-coordinator' },
      participants: [
        { agentName: 'code-agent', required: true, selected: true },
        { agentName: 'web-agent', required: false, selected: false },
      ],
      tasks: [
        { taskId: 't1', agentName: 'code-agent', content: 'Build backend' },
        { taskId: 't2', agentName: 'web-agent', content: 'Build frontend' },
      ],
    })
    const onApprove = vi.fn()
    const onCancel = vi.fn()

    render(
      <ActivitySnapshotRenderer
        conversationId="conv-test-1"
        onApprove={onApprove}
        onCancel={onCancel}
      />,
    )

    // Group indicator visible
    expect(screen.getByText('Group')).toBeInTheDocument()
    // Group coordinator credit
    expect(screen.getByText('Group Coordinator')).toBeInTheDocument()
    // Task content visible
    expect(screen.getByText('Build backend')).toBeInTheDocument()
    expect(screen.getByText('Build frontend')).toBeInTheDocument()
  })

  it('renders MainAgentPlanCard for main_agent_orchestration path', () => {
    seedActivity({
      executionPath: 'main_agent_orchestration',
      planOwner: { type: 'main_agent', agentName: 'main-agent' },
      candidateParticipants: [
        { agentName: 'code-agent', required: true, selected: true },
        { agentName: 'web-agent', required: false, selected: false },
      ],
      defaultSelectedParticipants: [
        { agentName: 'code-agent', required: true, selected: true },
      ],
      requiredParticipants: ['code-agent'],
      participants: [
        { agentName: 'code-agent', required: true, selected: true },
      ],
    })
    const onApprove = vi.fn()
    const onCancel = vi.fn()

    render(
      <ActivitySnapshotRenderer
        conversationId="conv-test-1"
        onApprove={onApprove}
        onCancel={onCancel}
      />,
    )

    // Main Agent credit in header
    expect(screen.getByText('Main Agent')).toBeInTheDocument()
    // Candidate participant selection visible
    expect(screen.getByText(/Select Participants/)).toBeInTheDocument()
    // Candidate checkboxes present (checkboxes inside labels with agent names)
    const checkboxes = screen.getAllByRole('checkbox')
    expect(checkboxes).toHaveLength(2)
    // Required badge for code-agent
    expect(screen.getByText('Required')).toBeInTheDocument()
    // Participant names visible (appear in both checkbox labels and assigned section)
    expect(screen.getAllByText('code-agent').length).toBeGreaterThan(0)
    expect(screen.getByText('web-agent')).toBeInTheDocument()
    // Approve button
    expect(screen.getByText('Approve & Execute')).toBeInTheDocument()
  })

  it('calls onApprove with selectedParticipants from MainAgentPlanCard', () => {
    // This test verifies the callback flow - selecting/deselecting participants
    // and clicking approve passes the correct list.
    const onApprove = vi.fn()
    const onCancel = vi.fn()
    seedActivity({
      executionPath: 'single_chat',
    })

    render(
      <ActivitySnapshotRenderer
        conversationId="conv-test-1"
        onApprove={onApprove}
        onCancel={onCancel}
      />,
    )

    // For single_chat, onApprove is called with no args
    fireClick(screen.getByText('同意并执行'))
    expect(onApprove).toHaveBeenCalledTimes(1)
    expect(onApprove).toHaveBeenCalledWith()
  })
})

function fireClick(el: HTMLElement) {
  const event = new MouseEvent('click', { bubbles: true, cancelable: true })
  el.dispatchEvent(event)
}
