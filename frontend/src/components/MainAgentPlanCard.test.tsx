import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import MainAgentPlanCard from './MainAgentPlanCard'
import type { PlanApprovalData } from '../types/planApproval'
import type { ActivityParticipant } from '../types'

function buildData(overrides: Partial<PlanApprovalData> = {}): PlanApprovalData {
  return {
    runId: 'run-1',
    actionId: 'plan-1',
    planId: 'plan-1',
    revision: 1,
    executionPath: 'main_agent_orchestration',
    title: 'Orchestration Plan',
    summary: 'Build full-stack login feature',
    planOwner: { type: 'main_agent', agentName: 'main-agent', isMainAgent: true },
    tasks: [
      { taskId: 't1', agentName: 'code-agent', content: 'Write backend code', priority: 1, riskLevel: 'low' },
      { taskId: 't2', agentName: 'web-agent', content: 'Write frontend code', priority: 2, riskLevel: 'low' },
      { taskId: 't3', agentName: 'reviewer', content: 'Review code', priority: 3, riskLevel: 'medium' },
    ],
    participants: [
      { agentName: 'code-agent', required: true, selected: true },
      { agentName: 'web-agent', required: false },
      { agentName: 'reviewer', required: false },
    ],
    strategy: 'sequential',
    warnings: [],
    ...overrides,
  }
}

function buildCandidateParticipants(): ActivityParticipant[] {
  return [
    { agentName: 'code-agent' },
    { agentName: 'web-agent' },
    { agentName: 'reviewer' },
  ]
}

const defaultSelected: ActivityParticipant[] = [
  { agentName: 'code-agent' },
  { agentName: 'web-agent' },
]

const requiredNames = ['code-agent']

describe('MainAgentPlanCard', () => {
  // 1. required participant checkbox is disabled
  it('disables checkbox for required participants', () => {
    render(
      <MainAgentPlanCard
        data={buildData()}
        candidateParticipants={buildCandidateParticipants()}
        defaultSelectedParticipants={defaultSelected}
        requiredParticipants={requiredNames}
        onApprove={vi.fn()}
        onCancel={vi.fn()}
        onRevise={vi.fn()}
      />,
    )

    const checkboxes = screen.getAllByRole('checkbox')
    // code-agent (required) checkbox should be disabled
    const codeCheckbox = checkboxes[0]
    expect(codeCheckbox).toBeDisabled()
  })

  // 2. optional participant checkbox is enabled
  it('enables checkbox for optional participants', () => {
    render(
      <MainAgentPlanCard
        data={buildData()}
        candidateParticipants={buildCandidateParticipants()}
        defaultSelectedParticipants={defaultSelected}
        requiredParticipants={requiredNames}
        onApprove={vi.fn()}
        onCancel={vi.fn()}
        onRevise={vi.fn()}
      />,
    )

    const checkboxes = screen.getAllByRole('checkbox')
    // web-agent (optional, recommended) checkbox should be enabled
    const webCheckbox = checkboxes[1]
    expect(webCheckbox).not.toBeDisabled()
  })

  // 3. selection unchanged → calls onApprove directly (not onRevise)
  it('calls onApprove when selection is unchanged from defaults', async () => {
    const onApprove = vi.fn().mockResolvedValue(undefined)
    const onRevise = vi.fn().mockResolvedValue(undefined)

    render(
      <MainAgentPlanCard
        data={buildData()}
        candidateParticipants={buildCandidateParticipants()}
        defaultSelectedParticipants={defaultSelected}
        requiredParticipants={requiredNames}
        onApprove={onApprove}
        onCancel={vi.fn()}
        onRevise={onRevise}
      />,
    )

    fireEvent.click(screen.getByText('Approve & Execute'))

    await waitFor(() => {
      expect(onApprove).toHaveBeenCalledTimes(1)
      expect(onApprove).toHaveBeenCalledWith(['code-agent', 'web-agent'])
      expect(onRevise).not.toHaveBeenCalled()
    })
  })

  // 4. selection changed → routes through revise (calls onRevise, not onApprove)
  it('routes through revise when selection differs from defaults', async () => {
    const onApprove = vi.fn().mockResolvedValue(undefined)
    const onRevise = vi.fn().mockResolvedValue(undefined)

    render(
      <MainAgentPlanCard
        data={buildData()}
        candidateParticipants={buildCandidateParticipants()}
        defaultSelectedParticipants={defaultSelected}
        requiredParticipants={requiredNames}
        onApprove={onApprove}
        onCancel={vi.fn()}
        onRevise={onRevise}
      />,
    )

    // Uncheck optional web-agent checkbox
    const checkboxes = screen.getAllByRole('checkbox')
    fireEvent.click(checkboxes[1]) // web-agent (currently checked → unchecked)

    // Button should now say "Request Revision"
    const reviseBtn = screen.getByText('Request Revision')
    fireEvent.click(reviseBtn)

    await waitFor(() => {
      expect(onRevise).toHaveBeenCalledTimes(1)
      expect(onRevise).toHaveBeenCalledWith(
        expect.stringContaining('Participant selection changed'),
        ['code-agent'],
      )
      expect(onApprove).not.toHaveBeenCalled()
    })
  })

  // 5. shows warning when selection differs from defaults
  it('shows warning text when selection differs from defaults', () => {
    render(
      <MainAgentPlanCard
        data={buildData()}
        candidateParticipants={buildCandidateParticipants()}
        defaultSelectedParticipants={defaultSelected}
        requiredParticipants={requiredNames}
        onApprove={vi.fn()}
        onCancel={vi.fn()}
        onRevise={vi.fn()}
      />,
    )

    // Uncheck optional web-agent
    const checkboxes = screen.getAllByRole('checkbox')
    fireEvent.click(checkboxes[1])

    expect(screen.getByText(/Selection differs from defaults/)).toBeInTheDocument()
  })

  // 6. does not show warning when selection matches defaults
  it('does not show warning when selection matches defaults', () => {
    render(
      <MainAgentPlanCard
        data={buildData()}
        candidateParticipants={buildCandidateParticipants()}
        defaultSelectedParticipants={defaultSelected}
        requiredParticipants={requiredNames}
        onApprove={vi.fn()}
        onCancel={vi.fn()}
        onRevise={vi.fn()}
      />,
    )

    expect(screen.queryByText(/Selection differs from defaults/)).not.toBeInTheDocument()
  })

  // 7. buttons disabled during approving state
  it('disables buttons during approving state', () => {
    render(
      <MainAgentPlanCard
        data={buildData()}
        candidateParticipants={buildCandidateParticipants()}
        defaultSelectedParticipants={defaultSelected}
        requiredParticipants={requiredNames}
        status="approving"
        onApprove={vi.fn()}
        onCancel={vi.fn()}
        onRevise={vi.fn()}
      />,
    )

    const approveBtn = screen.getByText('Approving...')
    expect(approveBtn).toBeDisabled()

    const cancelBtn = screen.getByText('Cancel')
    expect(cancelBtn).toBeDisabled()
  })

  // 8. hides action buttons in terminal states
  it('hides action buttons when cancelled', () => {
    render(
      <MainAgentPlanCard
        data={buildData()}
        candidateParticipants={buildCandidateParticipants()}
        defaultSelectedParticipants={defaultSelected}
        requiredParticipants={requiredNames}
        status="cancelled"
        onApprove={vi.fn()}
        onCancel={vi.fn()}
        onRevise={vi.fn()}
      />,
    )

    expect(screen.queryByText('Approve & Execute')).not.toBeInTheDocument()
    expect(screen.queryByText('Cancel')).not.toBeInTheDocument()
    expect(screen.getByText('Plan cancelled')).toBeInTheDocument()
  })

  // 9. cancel calls onCancel
  it('calls onCancel when cancel button is clicked', async () => {
    const onCancel = vi.fn().mockResolvedValue(undefined)

    render(
      <MainAgentPlanCard
        data={buildData()}
        candidateParticipants={buildCandidateParticipants()}
        defaultSelectedParticipants={defaultSelected}
        requiredParticipants={requiredNames}
        onApprove={vi.fn()}
        onCancel={onCancel}
        onRevise={vi.fn()}
      />,
    )

    fireEvent.click(screen.getByText('Cancel'))

    await waitFor(() => {
      expect(onCancel).toHaveBeenCalledTimes(1)
    })
  })

  // 10. Shows "Auto" badge for main_agent_orchestration executionPath
  it('shows execution path badge for main_agent_orchestration', () => {
    render(
      <MainAgentPlanCard
        data={buildData({ executionPath: 'main_agent_orchestration' })}
        candidateParticipants={buildCandidateParticipants()}
        defaultSelectedParticipants={defaultSelected}
        requiredParticipants={requiredNames}
        onApprove={vi.fn()}
        onCancel={vi.fn()}
      />,
    )

    expect(screen.getByText('Auto')).toBeInTheDocument()
  })
})
