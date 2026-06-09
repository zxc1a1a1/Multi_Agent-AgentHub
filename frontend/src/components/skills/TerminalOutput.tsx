interface Props {
  command?: string
  stdout?: string
  stderr?: string
  exitCode?: number
}

export default function TerminalOutput({ command, stdout, stderr, exitCode }: Props) {
  return (
    <div className="mt-2 rounded-lg border border-gray-700 bg-gray-900 overflow-hidden">
      {/* Header */}
      <div className="flex items-center justify-between px-3 py-1.5 bg-gray-800 border-b border-gray-700">
        <span className="text-xs font-mono text-gray-300">
          {command || 'terminal'}
        </span>
        {exitCode !== undefined && (
          <span
            className={`text-xs px-1.5 py-0.5 rounded ${
              exitCode === 0
                ? 'bg-green-900/50 text-green-400'
                : 'bg-red-900/50 text-red-400'
            }`}
          >
            exit: {exitCode}
          </span>
        )}
      </div>

      {/* Output */}
      <div className="p-3 font-mono text-xs leading-relaxed">
        {stdout && (
          <pre className="text-green-400 whitespace-pre-wrap break-all m-0">
            {stdout}
          </pre>
        )}
        {stderr && (
          <pre className="text-red-400 whitespace-pre-wrap break-all m-0 mt-1">
            {stderr}
          </pre>
        )}
        {!stdout && !stderr && (
          <span className="text-gray-500 italic">No output</span>
        )}
      </div>
    </div>
  )
}
