import { Brain, Code, Globe, Wrench, AlertTriangle } from 'lucide-react'

export interface OrchestrationInfo {
  intent?: string
  reasoning?: string
  strategy?: string
  taskCount?: number
  plannerSource?: string
  plannerSourceLabel?: string
  plannerModel?: string
}

interface Props {
  info: OrchestrationInfo
}

const strategyIcons: Record<string, typeof Brain> = {
  single: Brain,
  ordered_parallel: Code,
}

const sourceColors: Record<string, string> = {
  llm: 'bg-blue-100 text-blue-700 border-blue-300',
  rule: 'bg-gray-100 text-gray-600 border-gray-300',
  fallback: 'bg-orange-100 text-orange-700 border-orange-300',
}

const sourceIcons: Record<string, typeof Brain> = {
  llm: Brain,
  rule: Wrench,
  fallback: AlertTriangle,
}

export default function OrchestrationCard({ info }: Props) {
  if (!info.intent && !info.strategy) return null

  const SourceIcon = sourceIcons[info.plannerSource || ''] || Brain
  const sourceColor = sourceColors[info.plannerSource || ''] || sourceColors.rule

  return (
    <div className="rounded-lg border border-indigo-200 bg-indigo-50/50 px-4 py-3 text-sm">
      {/* Header */}
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-2">
          <Brain className="w-4 h-4 text-indigo-500" />
          <span className="font-medium text-indigo-700">编排分析</span>
        </div>
        {info.plannerSourceLabel && (
          <span className={`text-xs px-2 py-0.5 rounded-full border ${sourceColor} flex items-center gap-1`}>
            <SourceIcon className="w-3 h-3" />
            {info.plannerSourceLabel}
          </span>
        )}
      </div>

      {/* Intent */}
      {info.intent && (
        <div className="mb-2">
          <span className="text-gray-500">意图：</span>
          <span className="text-gray-800 font-medium">{info.intent}</span>
        </div>
      )}

      {/* Strategy & task count */}
      {info.strategy && (
        <div className="flex items-center gap-4 mb-2 text-xs">
          <span className="text-gray-500">
            策略：
            <span className="text-gray-700 font-medium ml-1">
              {info.strategy === 'single' ? '单 Agent' : '多 Agent 协作'}
            </span>
          </span>
          {info.taskCount && info.taskCount > 0 && (
            <span className="text-gray-500">
              任务数：
              <span className="text-gray-700 font-medium ml-1">{info.taskCount}</span>
            </span>
          )}
        </div>
      )}

      {/* Reasoning (collapsible) */}
      {info.reasoning && (
        <details className="text-xs text-gray-500 mt-1">
          <summary className="cursor-pointer hover:text-gray-700">查看推理过程</summary>
          <p className="mt-1 text-gray-600 leading-relaxed">{info.reasoning}</p>
        </details>
      )}

      {/* Model info */}
      {info.plannerModel && (
        <div className="text-xs text-gray-400 mt-2">
          模型：{info.plannerModel}
        </div>
      )}
    </div>
  )
}
