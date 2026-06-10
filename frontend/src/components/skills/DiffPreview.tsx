import { useState } from 'react'
import * as api from '../../services/api'

interface Props {
  diffText: string
  filename?: string
}

/**
 * Unified diff display with safe Phase 8 dry-run/apply handshake.
 * Gateway only proxies; Orchestrator validates path/diff and records apply.
 */
export default function DiffPreview({ diffText, filename }: Props) {
  const lines = diffText.split('\n')
  const [dryRunId, setDryRunId] = useState<string>('')
  const [status, setStatus] = useState<string>('')
  const [error, setError] = useState<string>('')
  const [loading, setLoading] = useState(false)
  const path = filename || extractPath(diffText) || 'src/unknown.patch'

  const dryRun = async () => {
    setLoading(true)
    setError('')
    try {
      const res = await api.dryRunDiff({ path, diffText })
      setDryRunId(res.dryRunId || '')
      setStatus(`Dry-run OK: ${res.files.join(', ')}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Dry-run failed')
    } finally {
      setLoading(false)
    }
  }

  const apply = async () => {
    if (!dryRunId) {
      setError('Run dry-run before apply')
      return
    }
    if (!window.confirm(`Apply validated diff for ${path}?`)) {
      return
    }
    setLoading(true)
    setError('')
    try {
      const res = await api.applyDiff({ path, diffText, dryRunId, confirmed: true })
      setStatus(`Apply accepted: ${res.status}`)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Apply failed')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="mt-2 rounded-lg border border-gray-300 overflow-hidden bg-white">
      <div className="flex items-center px-3 py-1.5 bg-gray-50 border-b border-gray-200 gap-2">
        <span className="text-xs font-mono text-gray-600 truncate">
          {path}
        </span>
        <span className="ml-auto text-[10px] text-gray-400">diff view</span>
        <button disabled={loading} onClick={dryRun} className="text-[10px] px-2 py-0.5 rounded border border-gray-300 bg-white hover:bg-gray-100 disabled:opacity-50">
          Dry-run
        </button>
        <button disabled={loading || !dryRunId} onClick={apply} className="text-[10px] px-2 py-0.5 rounded border border-indigo-300 bg-indigo-50 text-indigo-700 hover:bg-indigo-100 disabled:opacity-50">
          Apply
        </button>
      </div>
      {(status || error) && (
        <div className={`px-3 py-1 text-xs border-b ${error ? 'bg-red-50 text-red-600 border-red-100' : 'bg-green-50 text-green-700 border-green-100'}`}>
          {error || status}
        </div>
      )}
      <div className="overflow-x-auto">
        <pre className="text-xs font-mono leading-relaxed m-0 p-0 max-h-96 overflow-y-auto">
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

function extractPath(diffText: string): string {
  const match = diffText.match(/^\+\+\+\s+b\/(.+)$/m) || diffText.match(/^---\s+a\/(.+)$/m)
  return match?.[1] || ''
}
