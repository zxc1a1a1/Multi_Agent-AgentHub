import { Bot, CheckCircle, XCircle, Clock, Loader2 } from 'lucide-react'

interface AgentTask {
  agentName: string
  taskId?: string
  status: 'completed' | 'failed' | 'running' | 'pending' | 'skipped'
  content?: string
  duration?: string
}

interface Props {
  agents?: string[]
  tasks?: AgentTask[]
  runStatus?: string
  duration?: string
  artifactCount?: number
}

const statusIcon: Record<string, React.ReactNode> = {
  completed: <CheckCircle className="w-3.5 h-3.5 text-green-500" />,
  failed: <XCircle className="w-3.5 h-3.5 text-red-500" />,
  running: <Loader2 className="w-3.5 h-3.5 text-blue-500 animate-spin" />,
  pending: <Clock className="w-3.5 h-3.5 text-gray-300" />,
  skipped: <Clock className="w-3.5 h-3.5 text-gray-300" />,
}

const statusLabel: Record<string, string> = {
  completed: 'Done',
  failed: 'Failed',
  running: 'Running',
  pending: 'Pending',
  skipped: 'Skipped',
}

/**
 * Post-run summary displayed after RUN_FINISHED.
 * Summarizes which agents were used and task outcomes.
 */
export default function OrchestrationSummary({
  agents,
  tasks,
  runStatus,
  duration,
  artifactCount,
}: Props) {
  const hasData = (agents && agents.length > 0) || (tasks && tasks.length > 0)

  return (
    <div className="my-4 rounded-lg border border-indigo-200 bg-indigo-50/50 p-4">
      {/* Header */}
      <div className="flex items-center justify-between mb-3">
        <div className="flex items-center gap-2">
          <Bot className="w-4 h-4 text-indigo-500" />
          <h4 className="text-sm font-semibold text-indigo-800">
            Orchestration Summary
          </h4>
        </div>
        {runStatus && (
          <span className="text-xs px-2 py-0.5 rounded-full bg-indigo-100 text-indigo-700 border border-indigo-200">
            {runStatus}
          </span>
        )}
      </div>

      {/* Agents used */}
      {agents && agents.length > 0 && (
        <div className="mb-2">
          <span className="text-xs text-gray-500">Agents used:</span>
          <div className="flex flex-wrap gap-1 mt-1">
            {agents.map((name) => (
              <span
                key={name}
                className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-white border border-gray-200 text-xs text-gray-700"
              >
                <Bot className="w-3 h-3 text-indigo-400" />
                {name}
              </span>
            ))}
          </div>
        </div>
      )}

      {/* Tasks */}
      {tasks && tasks.length > 0 && (
        <div className="space-y-1.5 mt-2">
          <span className="text-xs text-gray-500">Tasks:</span>
          {tasks.map((task, i) => (
            <div
              key={task.taskId || i}
              className="flex items-center gap-2 px-2 py-1.5 rounded bg-white border border-gray-100 text-xs"
            >
              {statusIcon[task.status] || statusIcon.pending}
              <span className="text-gray-600 font-mono text-xs">
                {task.agentName}
              </span>
              {task.content && (
                <span className="text-gray-400 truncate flex-1">
                  — {task.content}
                </span>
              )}
              <span className="text-gray-400 flex-shrink-0">
                {statusLabel[task.status] || task.status}
              </span>
              {task.duration && (
                <span className="text-gray-300 flex-shrink-0">
                  {task.duration}
                </span>
              )}
            </div>
          ))}
        </div>
      )}

      {/* Footer stats */}
      <div className="flex items-center gap-4 mt-3 pt-2 border-t border-indigo-100 text-xs text-gray-400">
        {duration && (
          <span>Duration: {duration}</span>
        )}
        {artifactCount !== undefined && artifactCount > 0 && (
          <span>{artifactCount} artifact{artifactCount !== 1 ? 's' : ''}</span>
        )}
        {!hasData && (
          <span>No orchestration data available for this run.</span>
        )}
      </div>
    </div>
  )
}
