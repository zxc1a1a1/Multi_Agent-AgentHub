import { useState, useCallback } from 'react'
import type { PlanApprovalData } from '../types/planApproval'

export type PlanApprovalStatus =
  | 'waiting'
  | 'approving'
  | 'cancelling'
  | 'cancelled'
  | 'executing'
  | 'failed'
  | 'revising'

export interface PlanApprovalCardProps {
  data: PlanApprovalData
  status?: PlanApprovalStatus
  onApprove: () => void | Promise<void>
  onCancel: () => void | Promise<void>
  onRevise?: (feedback: string) => void | Promise<void>
}

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

export default function PlanApprovalCard({
  data,
  status = 'waiting',
  onApprove,
  onCancel,
  onRevise,
}: PlanApprovalCardProps) {
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

  return (
    <div className="plan-approval-card rounded-lg border border-gray-200 bg-white shadow-sm overflow-hidden">
      {/* Header */}
      <div className="px-4 py-3 border-b border-gray-100 bg-gray-50/50 flex items-center justify-between flex-wrap gap-2">
        <div className="flex items-center gap-2 min-w-0">
          <h3 className="text-sm font-semibold text-gray-800 truncate">
            {data.title}
          </h3>
          {data.executionPath && data.executionPath !== 'unknown' && (
            <span className="inline-flex items-center px-2 py-0.5 rounded text-xs font-medium bg-indigo-50 text-indigo-600 border border-indigo-100 shrink-0">
              {EXECUTION_PATH_LABELS[data.executionPath] || data.executionPath}
            </span>
          )}
        </div>
        {data.planOwner && data.planOwner.agentName && (
          <span className="text-xs text-gray-500 shrink-0">
            by{' '}
            <span className="font-medium text-gray-700">
              {data.planOwner.type === 'main_agent' ? 'Main Agent' : data.planOwner.agentName}
            </span>
          </span>
        )}
        {data.executionOwner && data.executionOwner.type === 'agent' && data.executionOwner.agentName && (
          <span className="text-xs text-gray-400 shrink-0">
            · execute <span className="font-medium text-gray-600">{data.executionOwner.agentName}</span>
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

      {/* Steps / Tasks */}
      <div className="px-4 py-3 border-b border-gray-100">
        <h4 className="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">
          Steps ({data.tasks.length})
        </h4>
        {data.tasks.length === 0 ? (
          <p className="text-sm text-gray-400 italic">暂无详细步骤</p>
        ) : (
          <ol className="space-y-2">
            {data.tasks.map((task, idx) => (
              <li
                key={task.taskId || String(idx)}
                className="flex items-start gap-3 text-sm"
              >
                <span className="flex-shrink-0 w-5 h-5 rounded-full bg-indigo-100 text-indigo-600 text-xs font-medium flex items-center justify-center mt-0.5">
                  {idx + 1}
                </span>
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-2 flex-wrap">
                    {task.agentName && (
                      <span className="text-xs font-medium text-gray-500 bg-gray-100 px-1.5 py-0.5 rounded">
                        {task.agentName}
                      </span>
                    )}
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
            ))}
          </ol>
        )}
      </div>

      {/* Participants */}
      {data.participants.length > 0 && (
        <div className="px-4 py-3 border-b border-gray-100">
          <h4 className="text-xs font-semibold text-gray-500 uppercase tracking-wider mb-2">
            Participants ({data.participants.length})
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
          {/* Feedback input for revise — toggled by "提出修改意见" button */}
          {onRevise && showFeedbackInput && (
            <div>
              <textarea
                rows={2}
                placeholder="请输入你希望如何修改这个方案，例如：简化步骤、不要包含测试、先分析再执行……"
                value={feedback}
                onChange={(e) => setFeedback(e.target.value)}
                disabled={isBusy}
                className="w-full px-3 py-1.5 text-sm rounded-md border border-gray-300 bg-white text-gray-700 placeholder-gray-400 focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent disabled:opacity-50 disabled:cursor-not-allowed resize-none"
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
                提出修改意见
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
                  取消修改
                </button>
                <button
                  type="button"
                  disabled={isBusy || !feedback.trim()}
                  onClick={handleRevise}
                  className="px-3 py-1.5 text-sm rounded-md border border-amber-300 bg-amber-50 text-amber-700 hover:bg-amber-100 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
                >
                  {status === 'revising' ? 'Revising...' : '提交修改意见'}
                </button>
              </>
            )}
            <button
              type="button"
              disabled={isBusy}
              onClick={handleCancel}
              className="px-3 py-1.5 text-sm rounded-md border border-gray-300 bg-white text-gray-700 hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
            >
              {status === 'cancelling' ? 'Cancelling...' : '取消'}
            </button>
            <button
              type="button"
              disabled={isBusy}
              onClick={handleApprove}
              className="px-4 py-1.5 text-sm rounded-md bg-indigo-600 text-white hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors font-medium"
            >
              {status === 'approving' ? 'Approving...' : '同意并执行'}
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
