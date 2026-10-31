import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, expect, test, vi } from 'vitest'
import GroupPlanCard from './GroupPlanCard'
import type { PlanApprovalData } from '../types/planApproval'

function groupPlan(overrides: Partial<PlanApprovalData> = {}): PlanApprovalData {
  return {
    runId: 'run-group',
    planId: 'plan-group',
    revision: 1,
    executionPath: 'group_chat',
    title: 'Group plan',
    tasks: [
      { taskId: 't1', agentName: 'code-agent', content: 'write code' },
      { taskId: 't2', agentName: 'review-agent', content: 'review code' },
      { taskId: 't3', agentName: 'test-agent', content: 'test code' },
    ],
    participants: [
      { agentName: 'code-agent', required: true, selected: true },
      { agentName: 'review-agent', selected: true },
      { agentName: 'test-agent', selected: true },
    ],
    warnings: [],
    ...overrides,
  }
}

describe('GroupPlanCard participant selection', () => {
  test('approves only the selected optional participants', async () => {
    const onApprove = vi.fn().mockResolvedValue(undefined)
    render(<GroupPlanCard data={groupPlan()} onApprove={onApprove} onCancel={vi.fn()} />)

    const required = screen.getByRole('checkbox', { name: 'code-agent' })
    expect(required).toBeDisabled()
    expect(required).toHaveAttribute('aria-checked', 'true')

    const review = screen.getByRole('checkbox', { name: 'review-agent' })
    fireEvent.click(review)
    expect(review).toHaveAttribute('aria-checked', 'false')

    fireEvent.click(screen.getByText('Approve & Execute'))
    await waitFor(() => {
      expect(onApprove).toHaveBeenCalledWith(['code-agent', 'test-agent'])
    })
  })

  test('required participant cannot be deselected', () => {
    const onApprove = vi.fn()
    render(<GroupPlanCard data={groupPlan()} onApprove={onApprove} onCancel={vi.fn()} />)
    const required = screen.getByRole('checkbox', { name: 'code-agent' })
    fireEvent.click(required)
    expect(required).toHaveAttribute('aria-checked', 'true')
  })

  test('revision feedback carries the current selection', async () => {
    const onRevise = vi.fn().mockResolvedValue(undefined)
    render(
      <GroupPlanCard
        data={groupPlan()}
        onApprove={vi.fn()}
        onCancel={vi.fn()}
        onRevise={onRevise}
      />,
    )
    fireEvent.click(screen.getByRole('checkbox', { name: 'test-agent' }))
    fireEvent.click(screen.getByText('Suggest Changes'))
    fireEvent.change(screen.getByPlaceholderText(/describe how/i), {
      target: { value: 'swap reviewer' },
    })
    fireEvent.click(screen.getByText('Submit Feedback'))
    await waitFor(() => {
      expect(onRevise).toHaveBeenCalledWith('swap reviewer', ['code-agent', 'review-agent'])
    })
  })
})
