import { Puzzle } from 'lucide-react'

interface Props {
  toolName?: string
  args?: unknown
}

/**
 * Fallback display for unknown tool/skill names.
 * Prevents crashes — always renders safely.
 */
export default function UnknownSkillFallback({ toolName, args }: Props) {
  // Sanitize args for display — never eval
  let argsDisplay = ''
  if (args !== undefined && args !== null) {
    try {
      argsDisplay = JSON.stringify(args, null, 2)
    } catch {
      argsDisplay = String(args)
    }
  }

  return (
    <div className="mt-2 rounded-lg border border-gray-200 bg-gray-50 p-3">
      <div className="flex items-center gap-2 mb-2">
        <Puzzle className="w-4 h-4 text-gray-400" />
        <span className="text-xs font-medium text-gray-600">
          Tool: {toolName || 'Unknown'}
        </span>
        <span className="text-[10px] px-1.5 py-0.5 rounded bg-gray-200 text-gray-500">
          raw output
        </span>
      </div>
      {argsDisplay && (
        <pre className="text-xs text-gray-500 bg-white border border-gray-200 rounded p-2 overflow-x-auto max-h-32 whitespace-pre-wrap break-all">
          {argsDisplay}
        </pre>
      )}
    </div>
  )
}
