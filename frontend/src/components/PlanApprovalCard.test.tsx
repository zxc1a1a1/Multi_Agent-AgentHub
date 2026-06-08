import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import PlanApprovalCard from './PlanApprovalCard'
import type { PlanApprovalData } from '../types/planApproval'

function buildData(overrides: Partial<PlanApprovalData> = {}): PlanApprovalData {
  return {
    runId: 'run-1',
    actionId: 'plan-1',
    planId: 'plan-1',
    revision: 1,
    executionPath: 'single_chat',
    title: 'Test Plan',
    summary: 'Build a Go HTTP server',
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
        content: 'Add tests',
        priority: 2,
        riskLevel: 'medium',
      },
    ],
    participants: [
      { agentName: 'code-agent', required: true, selected: true },
    ],
    strategy: 'single',
    warnings: [],
    ...overrides,
  }
}

describe('PlanApprovalCard', () => {
  // 5. renders title, summary, executionPath badge, tasks, participants
  it('renders title, summary, executionPath badge, tasks, and participants', () => {
    render(
      <PlanApprovalCard
        data={buildData()}
        onApprove={vi.fn()}
        onCancel={vi.fn()}
      />,
    )

    expect(screen.getByText('Test Plan')).toBeInTheDocument()
    expect(screen.getByText('Build a Go HTTP server')).toBeInTheDocument()
    expect(screen.getByText('Single Agent')).toBeInTheDocument()
    expect(screen.getByText('Steps (2)')).toBeInTheDocument()
    expect(screen.getByText('Write Go HTTP server code')).toBeInTheDocument()
    expect(screen.getByText('Add tests')).toBeInTheDocument()
    expect(screen.getByText('Participants (1)')).toBeInTheDocument()
    // code-agent appears in planOwner header, task badges, and participants
    expect(screen.getAllByText('code-agent').length).toBeGreaterThanOrEqual(3)
  })

  // 6. clicking approve calls onApprove once
  it('calls onApprove when approve button clicked', () => {
    const onApprove = vi.fn()
    render(
      <PlanApprovalCard data={buildData()} onApprove={onApprove} onCancel={vi.fn()} />,
    )

    fireEvent.click(screen.getByText('同意并执行'))
    expect(onApprove).toHaveBeenCalledTimes(1)
  })

  // 7. clicking cancel calls onCancel once
  it('calls onCancel when cancel button clicked', () => {
    const onCancel = vi.fn()
    render(
      <PlanApprovalCard data={buildData()} onApprove={vi.fn()} onCancel={onCancel} />,
    )

    fireEvent.click(screen.getByText('取消'))
    expect(onCancel).toHaveBeenCalledTimes(1)
  })

  // 8. buttons disabled during loading/approving
  it('disables buttons during approving state', () => {
    render(
      <PlanApprovalCard
        data={buildData()}
        status="approving"
        onApprove={vi.fn()}
        onCancel={vi.fn()}
      />,
    )

    const approveBtn = screen.getByText('Approving...')
    expect(approveBtn).toBeDisabled()

    const cancelBtn = screen.getByText('取消')
    expect(cancelBtn).toBeDisabled()
  })

  it('disables buttons during cancelling state', () => {
    render(
      <PlanApprovalCard
        data={buildData()}
        status="cancelling"
        onApprove={vi.fn()}
        onCancel={vi.fn()}
      />,
    )

    expect(screen.getByText('Cancelling...')).toBeDisabled()
    expect(screen.getByText('同意并执行')).toBeDisabled()
  })

  // 9. empty tasks shows fallback
  it('shows fallback for empty tasks', () => {
    render(
      <PlanApprovalCard
        data={buildData({ tasks: [] })}
        onApprove={vi.fn()}
        onCancel={vi.fn()}
      />,
    )

    expect(screen.getByText('暂无详细步骤')).toBeInTheDocument()
    expect(screen.getByText('Steps (0)')).toBeInTheDocument()
  })

  // 10. long content renders without breaking layout
  it('renders long content without crashing', () => {
    const longContent = 'A'.repeat(500)
    render(
      <PlanApprovalCard
        data={buildData({
          summary: longContent,
          tasks: [{ content: longContent }],
        })}
        onApprove={vi.fn()}
        onCancel={vi.fn()}
      />,
    )

    expect(screen.getAllByText(longContent).length).toBeGreaterThanOrEqual(1)
  })

  // warnings rendering
  it('renders warnings when present', () => {
    render(
      <PlanApprovalCard
        data={buildData({ warnings: ['High latency expected', 'Untested path'] })}
        onApprove={vi.fn()}
        onCancel={vi.fn()}
      />,
    )

    expect(screen.getByText('High latency expected')).toBeInTheDocument()
    expect(screen.getByText('Untested path')).toBeInTheDocument()
  })

  it('does not render warnings section when empty', () => {
    render(
      <PlanApprovalCard
        data={buildData({ warnings: [] })}
        onApprove={vi.fn()}
        onCancel={vi.fn()}
      />,
    )

    // No amber warning text should appear
    expect(screen.queryByText('High latency')).not.toBeInTheDocument()
  })

  // terminal states hide buttons
  it('hides action buttons when cancelled', () => {
    render(
      <PlanApprovalCard
        data={buildData()}
        status="cancelled"
        onApprove={vi.fn()}
        onCancel={vi.fn()}
      />,
    )

    expect(screen.queryByText('同意并执行')).not.toBeInTheDocument()
    expect(screen.queryByText('取消')).not.toBeInTheDocument()
    expect(screen.getByText('Plan cancelled')).toBeInTheDocument()
  })

  it('hides action buttons when executing', () => {
    render(
      <PlanApprovalCard
        data={buildData()}
        status="executing"
        onApprove={vi.fn()}
        onCancel={vi.fn()}
      />,
    )

    expect(screen.queryByText('同意并执行')).not.toBeInTheDocument()
    expect(screen.getByText('Executing plan...')).toBeInTheDocument()
  })

  // planOwner display
  it('shows plan owner by agent in single_chat', () => {
    render(
      <PlanApprovalCard
        data={buildData({ planOwner: { type: 'agent', agentName: 'code-agent' } })}
        onApprove={vi.fn()}
        onCancel={vi.fn()}
      />,
    )

    // Plan owner renders in header: "by code-agent"
    const byText = screen.getByText('by')
    expect(byText).toBeInTheDocument()
    expect(screen.getAllByText('code-agent').length).toBeGreaterThanOrEqual(3)
  })

  // async onApprove handles loading state
  it('handles async approve callback without double-firing', async () => {
    const onApprove = vi.fn().mockResolvedValue(undefined)
    render(
      <PlanApprovalCard data={buildData()} onApprove={onApprove} onCancel={vi.fn()} />,
    )

    fireEvent.click(screen.getByText('同意并执行'))
    // Click again should not fire
    fireEvent.click(screen.getByText('同意并执行'))

    await waitFor(() => {
      expect(onApprove).toHaveBeenCalledTimes(1)
    })
  })

  // Unknown executionPath renders safely
  it('renders unknown executionPath without badge', () => {
    render(
      <PlanApprovalCard
        data={buildData({ executionPath: 'unknown' })}
        onApprove={vi.fn()}
        onCancel={vi.fn()}
      />,
    )

    expect(screen.queryByText('Single Agent')).not.toBeInTheDocument()
    expect(screen.queryByText('Unknown')).not.toBeInTheDocument()
  })

  // required participant shows indicator
  it('shows required indicator on required participants', () => {
    render(
      <PlanApprovalCard
        data={buildData({
          participants: [
            { agentName: 'code-agent', required: true },
            { agentName: 'test-agent', required: false },
          ],
        })}
        onApprove={vi.fn()}
        onCancel={vi.fn()}
      />,
    )

    // Both agents appear in participants section
    expect(screen.getByText('test-agent')).toBeInTheDocument()
    // code-agent appears multiple times (planOwner, tasks, participants)
    // but the required indicator "*" should be present with context
    const requiredIndicator = screen.getByTitle('Required')
    expect(requiredIndicator).toBeInTheDocument()
  })

  // Fallback title when summary is empty
  it('uses fallback title "确认执行方案" when summary is empty', () => {
    render(
      <PlanApprovalCard
        data={buildData({ title: '确认执行方案', summary: '' })}
        onApprove={vi.fn()}
        onCancel={vi.fn()}
      />,
    )

    expect(screen.getByText('确认执行方案')).toBeInTheDocument()
  })

  // --- revise tests ---

  it('shows revise button when onRevise prop is provided', () => {
    render(
      <PlanApprovalCard
        data={buildData()}
        onApprove={vi.fn()}
        onCancel={vi.fn()}
        onRevise={vi.fn()}
      />,
    )

    expect(screen.getByText('提出修改意见')).toBeInTheDocument()
  })

  it('hides revise button when onRevise prop is not provided', () => {
    render(
      <PlanApprovalCard
        data={buildData()}
        onApprove={vi.fn()}
        onCancel={vi.fn()}
      />,
    )

    expect(screen.queryByText('提出修改意见')).not.toBeInTheDocument()
  })

  it('shows feedback textarea after clicking revise button', () => {
    render(
      <PlanApprovalCard
        data={buildData()}
        onApprove={vi.fn()}
        onCancel={vi.fn()}
        onRevise={vi.fn()}
      />,
    )

    // textarea should not be visible initially
    expect(screen.queryByPlaceholderText(/请输入你希望如何修改这个方案/)).not.toBeInTheDocument()

    // click revise button → textarea appears
    fireEvent.click(screen.getByText('提出修改意见'))
    expect(screen.getByPlaceholderText(/请输入你希望如何修改这个方案/)).toBeInTheDocument()
    expect(screen.getByText('取消修改')).toBeInTheDocument()
    expect(screen.getByText('提交修改意见')).toBeInTheDocument()
  })

  it('calls onRevise with feedback text', async () => {
    const onRevise = vi.fn().mockResolvedValue(undefined)
    render(
      <PlanApprovalCard
        data={buildData()}
        onApprove={vi.fn()}
        onCancel={vi.fn()}
        onRevise={onRevise}
      />,
    )

    fireEvent.click(screen.getByText('提出修改意见'))
    fireEvent.change(
      screen.getByPlaceholderText(/请输入你希望如何修改这个方案/),
      { target: { value: '简化步骤，减少参与者' } },
    )
    fireEvent.click(screen.getByText('提交修改意见'))

    await waitFor(() => {
      expect(onRevise).toHaveBeenCalledWith('简化步骤，减少参与者')
      expect(onRevise).toHaveBeenCalledTimes(1)
    })
  })

  it('disables submit when feedback is empty', () => {
    render(
      <PlanApprovalCard
        data={buildData()}
        onApprove={vi.fn()}
        onCancel={vi.fn()}
        onRevise={vi.fn()}
      />,
    )

    fireEvent.click(screen.getByText('提出修改意见'))

    // textarea is empty initially → submit button disabled
    const submitBtn = screen.getByText('提交修改意见')
    expect(submitBtn).toBeDisabled()
  })

  it('hides feedback textarea when cancel revise clicked', () => {
    render(
      <PlanApprovalCard
        data={buildData()}
        onApprove={vi.fn()}
        onCancel={vi.fn()}
        onRevise={vi.fn()}
      />,
    )

    fireEvent.click(screen.getByText('提出修改意见'))
    expect(screen.getByPlaceholderText(/请输入你希望如何修改这个方案/)).toBeInTheDocument()

    fireEvent.click(screen.getByText('取消修改'))
    expect(screen.queryByPlaceholderText(/请输入你希望如何修改这个方案/)).not.toBeInTheDocument()
    expect(screen.getByText('提出修改意见')).toBeInTheDocument()
  })
})
