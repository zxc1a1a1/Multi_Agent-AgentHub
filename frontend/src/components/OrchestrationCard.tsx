import { useState } from 'react'
import { Brain, ChevronDown, ChevronRight, Code, Globe, Wrench, AlertTriangle } from 'lucide-react'

export interface OrchestrationInfo {
  intent?: string
  reasoning?: string
  strategy?: string
  taskCount?: number
  plannerSource?: string
  plannerSourceLabel?: string
  plannerModel?: string
  // Actual agent selection from orchestrator STATE_UPDATE.
  selectedAgentName?: string
  selectedAgentDisplayName?: string
  // Task-level agent assignments (comma-separated or array in raw state).
  taskAgentNames?: string
}

interface Props {
  info: OrchestrationInfo
}

function statusSummary(info: OrchestrationInfo): string {
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
  const strategy = info.strategy === 'ordered_parallel' ? '多 Agent 协作' : '单 Agent'
  if (info.taskCount && info.taskCount > 0) {
    return `${strategy} · ${info.taskCount} 个任务`
  }
  return strategy
}

export default function OrchestrationCard({ info }: Props) {
  const [expanded, setExpanded] = useState(false)

  if (!info.intent && !info.strategy) return null

  const hasDetail =
    !!info.reasoning || !!info.plannerSource || !!info.plannerModel

  return (
    <div className="rounded-lg border border-indigo-200 bg-indigo-50/50 text-sm">
      {/* Collapsed header — always visible */}
      <button
        onClick={() => hasDetail && setExpanded(!expanded)}
        className="w-full flex items-center justify-between px-4 py-2 hover:bg-indigo-50 transition-colors text-left"
      >
        <div className="flex items-center gap-2">
          <Brain className="w-4 h-4 text-indigo-500" />
          <span className="text-xs text-indigo-700">{statusSummary(info)}</span>
        </div>
        {hasDetail && (
          <span className="text-indigo-400">
            {expanded ? <ChevronDown className="w-4 h-4" /> : <ChevronRight className="w-4 h-4" />}
          </span>
        )}
      </button>

      {/* Expanded detail area */}
      {expanded && (
        <div className="px-4 pb-3 border-t border-indigo-100">
          {info.intent && (
            <div className="mt-2">
              <span className="text-gray-500 text-xs">意图：</span>
              <span className="text-gray-800 text-xs font-medium">{info.intent}</span>
            </div>
          )}

          {info.plannerSource && (
            <div className="mt-1 flex items-center gap-2 text-xs">
              <span className="text-gray-500">
                来源：{info.plannerSourceLabel || info.plannerSource}
              </span>
              {info.plannerModel && (
                <span className="text-gray-400">模型：{info.plannerModel}</span>
              )}
            </div>
          )}

          {info.reasoning && (
            <div className="mt-2 text-xs text-gray-500 leading-relaxed border-t border-indigo-100 pt-2">
              {info.reasoning}
            </div>
          )}
        </div>
      )}
    </div>
  )
}
