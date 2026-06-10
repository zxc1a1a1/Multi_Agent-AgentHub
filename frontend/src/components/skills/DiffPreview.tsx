interface Props {
  diffText: string
  filename?: string
}

/**
 * Lightweight unified diff display.
 * DISPLAY ONLY — no Apply/merge functionality.
 */
export default function DiffPreview({ diffText, filename }: Props) {
  const lines = diffText.split('\n')

  return (
    <div className="mt-2 rounded-lg border border-gray-300 overflow-hidden bg-white">
      {/* Header */}
      <div className="flex items-center px-3 py-1.5 bg-gray-50 border-b border-gray-200">
        <span className="text-xs font-mono text-gray-600">
          {filename || 'diff'}
        </span>
        <span className="ml-auto text-[10px] text-gray-400">diff view</span>
      </div>

      {/* Diff content */}
      <div className="overflow-x-auto">
        <pre className="text-xs font-mono leading-relaxed m-0 p-0">
          {lines.map((line, i) => {
            let lineClass = 'px-3 py-0'
            if (line.startsWith('+') && !line.startsWith('+++')) {
              lineClass += ' bg-green-50 text-green-800'
            } else if (line.startsWith('-') && !line.startsWith('---')) {
              lineClass += ' bg-red-50 text-red-800'
            } else if (line.startsWith('@@')) {
              lineClass += ' bg-blue-50 text-blue-700'
            }
            return (
              <div key={i} className={lineClass}>
                {line || ' '}
              </div>
            )
          })}
        </pre>
      </div>
    </div>
  )
}
