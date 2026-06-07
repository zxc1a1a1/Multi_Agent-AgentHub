import { useState } from 'react'
import { Brain, ChevronDown, ChevronRight, AlertTriangle } from 'lucide-react'

export interface OrchestrationInfo {
  intent?: string
  reasoning?: string
  strategy?: string
  taskCount?: number
  plannerSource?: string
  plannerSourceLabel?: string
  plannerModel?: string
  // Current execution phase (planning, thinking, executing, error, etc.)
  phase?: string
  // Actual agent selection from orchestrator STATE_UPDATE.
  selectedAgentName?: string
  selectedAgentDisplayName?: string
  // Task-level agent assignments (comma-separated or array in raw state).
  taskAgentNames?: string
  // HITL plan confirmation fields.
  requiresConfirmation?: boolean
  confirmationActionId?: string
  plannedAgents?: string[]
  plannedTasks?: Array<{
    taskId?: string
    agentName?: string
    content?: string
    dependsOn?: string[]
    priority?: number
    riskLevel?: string
  }>
  // Error tracking fields.
  errorCode?: string
  errorPhase?: string
}

interface Props {
  info: OrchestrationInfo
}

function phaseLabel(phase: string): string {
  switch (phase) {
    case 'planning': return '规划阶段'
    case 'thinking': return '分析阶段'
    case 'confirming': return '确认阶段'
    case 'executing': return '执行阶段'
    case 'dispatching': return '调度阶段'
    case 'synthesizing': return '汇总阶段'
    case 'error': return '错误'
    default: return phase
  }
}

function statusSummary(info: OrchestrationInfo): string {
  // Show phase-based status for lifecycle states.
  if (info.phase === 'error') {
    return '自动编排失败'
  }
  if (info.phase === 'thinking') {
    return '分析中…'
  }
  if (info.phase === 'planning') {
    return '正在规划…'
  }
  if (info.phase === 'responding') {
    return '正在回复…'
  }

  if (!info.strategy && !info.intent) {
    return 'Planning…'
  }

  // Show actual agent selection when available from orchestrator STATE_UPDATE.
  if (info.selectedAgentDisplayName || info.selectedAgentName) {
    const agentLabel = info.selectedAgentDisplayName || info.selectedAgentName || ''
    if (info.taskCount && info.taskCount > 0) {
      return `已选择 ${agentLabel} · ${info.taskCount} 个任务`
    }
    return `已选择 ${agentLabel}`
  }

  // Fallback to strategy-based summary.
  // ordered_parallel and sequential both involve multiple agents.
  let strategy: string
  if (info.strategy === 'ordered_parallel' || info.strategy === 'sequential') {
    strategy = '多 Agent 编排'
  } else {
    strategy = '单 Agent'
  }
  if (info.taskCount && info.taskCount > 0) {
    return `${strategy} · ${info.taskCount} 个任务`
  }
  return strategy
}

export default function OrchestrationCard({ info }: Props) {
  const [expanded, setExpanded] = useState(false)

  // Show the card when there's intent, strategy, phase, or error info.
  if (!info.intent && !info.strategy && !info.phase) return null

  const isError = info.phase === 'error'
  const hasDetail =
    !!info.reasoning || !!info.plannerSource || !!info.plannerModel || isError

  const borderClass = isError
    ? 'border-red-200 bg-red-50/50'
    : 'border-indigo-200 bg-indigo-50/50'
  const headerHoverClass = isError ? 'hover:bg-red-50' : 'hover:bg-indigo-50'
  const iconClass = isError ? 'text-red-500' : 'text-indigo-500'
  const textClass = isError ? 'text-red-700' : 'text-indigo-700'

  return (
    <div className={`rounded-lg border ${borderClass} text-sm`}>
      {/* Collapsed header — always visible */}
      <button
        onClick={() => hasDetail && setExpanded(!expanded)}
        className={`w-full flex items-center justify-between px-4 py-2 ${headerHoverClass} transition-colors text-left`}
      >
        <div className="flex items-center gap-2">
          {isError ? (
            <AlertTriangle className={`w-4 h-4 ${iconClass}`} />
          ) : (
            <Brain className={`w-4 h-4 ${iconClass}`} />
          )}
          <span className={`text-xs ${textClass}`}>{statusSummary(info)}</span>
        </div>
        {hasDetail && (
          <span className={isError ? 'text-red-400' : 'text-indigo-400'}>
            {expanded ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
          </span>
        )}
      </button>

      {/* Expanded detail area */}
      {expanded && (
        <div className={`px-4 pb-3 border-t ${isError ? 'border-red-100' : 'border-indigo-100'}`}>
          {isError && (
            <>
              <div className="mt-2">
                <span className="text-gray-500 text-xs">错误码：</span>
                <span className="text-red-700 text-xs font-medium">{info.errorCode || 'UNKNOWN'}</span>
              </div>
              {info.errorPhase && info.errorPhase !== 'unknown' && (
                <div className="mt-1">
                  <span className="text-gray-500 text-xs">阶段：</span>
                  <span className="text-gray-800 text-xs">{phaseLabel(info.errorPhase)}</span>
                </div>
              )}
              <div className="mt-2 text-xs text-red-600 leading-relaxed border-t border-red-100 pt-2">
                请重试或选择手动指定 Agent 来执行任务。
              </div>
            </>
          )}

          {!isError && info.intent && (
            <div className="mt-2">
              <span className="text-gray-500 text-xs">意图：</span>
              <span className="text-gray-800 text-xs font-medium">{info.intent}</span>
            </div>
          )}

          {!isError && info.plannerSource && (
            <div className="mt-1 flex items-center gap-2 text-xs">
              <span className="text-gray-500">
                来源：{info.plannerSourceLabel || info.plannerSource}
              </span>
              {info.plannerModel && (
                <span className="text-gray-400">模型：{info.plannerModel}</span>
              )}
            </div>
          )}

          {!isError && info.reasoning && (
            <div className="mt-2 text-xs text-gray-500 leading-relaxed border-t border-indigo-100 pt-2">
              {info.reasoning}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
