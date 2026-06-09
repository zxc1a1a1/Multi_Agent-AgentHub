import { useState, useCallback } from 'react'
import type { PlanApprovalData } from '../types/planApproval'
import type { PlanApprovalStatus } from './PlanApprovalCard'

const EXECUTION_PATH_LABELS: Record<string, string> = {
  single_chat: 'Single Agent',
  group_chat: 'Group Chat',
  main_agent_orchestration: 'Auto',
  legacy: 'Legacy',
  unknown: 'Unknown',
}

const RISK_BADGES: Record<string, { bg: string; text: string }> = {
  high: { bg: 'bg-red-100', text: 'text-red-700' },
  medium: { bg: 'bg-amber-100', text: 'text-amber-700' },
  low: { bg: 'bg-green-100', text: 'text-green-700' },
}

export interface GroupPlanCardProps {
  data: PlanApprovalData
  status?: PlanApprovalStatus
  onApprove: () => void | Promise<void>
  onCancel: () => void | Promise<void>
  onRevise?: (feedback: string) => void | Promise<void>
}

export default function GroupPlanCard({
  data,
  status = 'waiting',
  onApprove,
  onCancel,
  onRevise,
}: GroupPlanCardProps) {
  const [acting, setActing] = useState(false)
  const [feedback, setFeedback] = useState('')
  const [showFeedbackInput, setShowFeedbackInput] = useState(false)

  const handleApprove = useCallback(async () => {
    if (acting) return
    setActing(true)
    try {
      await onApprove()
    } finally {
      setActing(false)
    }
  }, [acting, onApprove])

  const handleCancel = useCallback(async () => {
    if (acting) return
    setActing(true)
    try {
      await onCancel()
    } finally {
      setActing(false)
    }
  }, [acting, onCancel])

  const handleRevise = useCallback(async () => {
    if (acting || !feedback.trim() || !onRevise) return
    setActing(true)
    try {
      await onRevise(feedback.trim())
      setFeedback('')
      setShowFeedbackInput(false)
    } finally {
      setActing(false)
    }
  }, [acting, feedback, onRevise])

  const isBusy = acting || status === 'approving' || status === 'cancelling' || status === 'revising'
  const isTerminal = status === 'cancelled' || status === 'executing' || status === 'failed'

  // Group agents by task assignment for the coordinator view.
  const agentsByTask = new Map<string, PlanApprovalData['tasks'][number][]>()
  for (const task of data.tasks) {
    const name = task.agentName || 'unassigned'
    if (!agentsByTask.has(name)) agentsByTask.set(name, [])
    agentsByTask.get(name)!.push(task)
  }

  return (
    <div className="plan-approval-card rounded-lg border border-gray-200 bg-white shadow-sm overflow-hidden">
      {/* Header with group indicator */}
      <div className="px-4 py-3 border-b border-gray-100 bg-teal-50/50 flex items-center justify-between flex-wrap gap-2">
        <div className="flex items-center gap-2 min-w-0">
          <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-teal-100 text-teal-700 border border-teal-200 shrink-0">
            Group
          </span>
          <h3 className="text-sm font-semibold text-gray-800 truncate">
            {data.title}
          </h3>
          {data.executionPath && data.executionPath !== 'unknown' && (
            <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-teal-50 text-teal-600 border border-teal-100 shrink-0">
              {EXECUTION_PATH_LABELS[data.executionPath] || data.executionPath}
            </span>
          )}
        </div>
        <span className="text-xs text-gray-500 shrink-0">
          by <span className="font-medium text-gray-700">Group Coordinator</span>
        </span>
        {data.executionOwner && data.executionOwner.agentNames && data.executionOwner.agentNames.length > 0 && (
          <span className="text-xs text-gray-400 shrink-0">
            · execute{' '}
            <span className="font-medium text-gray-600">
              {data.executionOwner.agentNames.join(', ')}
            </span>
          </span>
        )}
      </div>

      {/* Summary */}
      {data.summary && (
        <div className="px-4 py-2 border-b border-gray-100">
          <p className="text-sm text-gray-600 leading-relaxed whitespace-pre-wrap break-words">
            {data.summary}
          </p>
        </div>
      )}

      {/* Participants */}
      {data.participants.length > 0 && (
        <div className="px-4 py-3 border-b border-gray-100">
          <h4 className="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">
            Team ({data.participants.length})
          </h4>
          <div className="flex flex-wrap gap-2">
            {data.participants.map((p) => (
              <span
                key={p.agentName}
                className={`inline-flex items-center gap-1 px-2 py-1 rounded text-xs font-medium border
                  ${p.required ? 'bg-blue-50 text-blue-700 border-blue-200' : 'bg-gray-50 text-gray-600 border-gray-200'}`}
              >
                {p.agentName}
                {p.required && (
                  <span className="text-blue-400" title="Required">*</span>
                )}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Tasks grouped by agent */}
      <div className="px-4 py-3 border-b border-gray-100">
        <h4 className="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">
          Tasks ({data.tasks.length})
        </h4>
        {data.tasks.length === 0 ? (
          <p className="text-sm text-gray-400 italic">No detailed tasks</p>
        ) : (
          <div className="space-y-3">
            {Array.from(agentsByTask.entries()).map(([agentName, tasks]) => (
              <div key={agentName}>
                <div className="flex items-center gap-2 mb-1">
                  <span className="text-xs font-semibold text-teal-700 bg-teal-50 px-2 py-0.5 rounded border border-teal-100">
                    {agentName}
                  </span>
                  <span className="text-xs text-gray-400">{tasks.length} task{tasks.length > 1 ? 's' : ''}</span>
                </div>
                <ol className="space-y-1.5 ml-4">
                  {tasks.map((task, idx) => {
                    const globalIdx = data.tasks.indexOf(task)
                    return (
                      <li key={task.taskId || String(idx)} className="flex items-start gap-2 text-sm">
                        <span className="flex-shrink-0 w-4 h-4 rounded-full bg-teal-100 text-teal-600 text-xs font-medium flex items-center justify-center mt-0.5">
                          {globalIdx + 1}
                        </span>
                        <div className="min-w-0 flex-1">
                          <div className="flex items-center gap-2 flex-wrap">
                            {task.riskLevel && (
                              <span
                                className={`text-xs px-1.5 py-0.5 rounded ${RISK_BADGES[task.riskLevel]?.bg || 'bg-gray-100'} ${RISK_BADGES[task.riskLevel]?.text || 'text-gray-600'}`}
                              >
                                {task.riskLevel}
                              </span>
                            )}
                            {task.priority !== undefined && (
                              <span className="text-xs text-gray-400">P{task.priority}</span>
                            )}
                          </div>
                          <p className="text-gray-700 mt-0.5 whitespace-pre-wrap break-words">
                            {task.content || '(no content)'}
                          </p>
                        </div>
                      </li>
                    )
                  })}
                </ol>
              </div>
            ))}
          </div>
        )}
      </div>

      {/* Warnings */}
      {data.warnings.length > 0 && (
        <div className="px-4 py-2 border-b border-gray-100 bg-amber-50/50">
          {data.warnings.map((w, i) => (
            <p key={i} className="text-xs text-amber-700 flex items-start gap-1">
              <span className="shrink-0 mt-0.5">!</span>
              <span>{w}</span>
            </p>
          ))}
        </div>
      )}

      {/* Actions */}
      {!isTerminal && (
        <div className="px-4 py-3 space-y-2">
          {onRevise && showFeedbackInput && (
            <div>
              <textarea
                rows={2}
                placeholder="Describe how you'd like to change the plan..."
                value={feedback}
                onChange={(e) => setFeedback(e.target.value)}
                disabled={isBusy}
                className="w-full px-3 py-1.5 text-sm rounded-md border border-gray-300 bg-white text-gray-700 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-teal-500 focus:border-transparent disabled:opacity-50 disabled:cursor-not-allowed resize-none"
              />
            </div>
          )}
          <div className="flex items-center justify-end gap-2">
            {onRevise && !showFeedbackInput && (
              <button
                type="button"
                disabled={isBusy}
                onClick={() => setShowFeedbackInput(true)}
                className="px-3 py-1.5 text-sm rounded-md border border-amber-300 bg-amber-50 text-amber-700 hover:bg-amber-100 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
              >
                Suggest Changes
              </button>
            )}
            {onRevise && showFeedbackInput && (
              <>
                <button
                  type="button"
                  disabled={isBusy}
                  onClick={() => { setShowFeedbackInput(false); setFeedback('') }}
                  className="px-3 py-1.5 text-sm rounded-md border border-gray-300 bg-white text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                >
                  Cancel Edit
                </button>
                <button
                  type="button"
                  disabled={isBusy || !feedback.trim()}
                  onClick={handleRevise}
                  className="px-3 py-1.5 text-sm rounded-md border border-amber-300 bg-amber-50 text-amber-700 hover:bg-amber-100 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                >
                  {status === 'revising' ? 'Revising...' : 'Submit Feedback'}
                </button>
              </>
            )}
            <button
              type="button"
              disabled={isBusy}
              onClick={handleCancel}
              className="px-3 py-1.5 text-sm rounded-md border border-gray-300 bg-white text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {status === 'cancelling' ? 'Cancelling...' : 'Cancel'}
            </button>
            <button
              type="button"
              disabled={isBusy}
              onClick={handleApprove}
              className="px-4 py-1.5 text-sm rounded-md bg-teal-600 text-white hover:bg-teal-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors font-medium"
            >
              {status === 'approving' ? 'Approving...' : 'Approve & Execute'}
            </button>
          </div>
        </div>
      )}

      {/* Terminal states */}
      {status === 'cancelled' && (
        <div className="px-4 py-2 bg-gray-50 text-center text-sm text-gray-500">
          Plan cancelled
        </div>
      )}
      {status === 'executing' && (
        <div className="px-4 py-2 bg-green-50 text-center text-sm text-green-700">
          Executing plan...
        </div>
      )}
      {status === 'failed' && (
        <div className="px-4 py-2 bg-red-50 text-center text-sm text-red-700">
          Execution failed
        </div>
      )}
    </div>
  )
}
