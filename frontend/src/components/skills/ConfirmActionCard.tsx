import { useState } from 'react'
import { AlertTriangle, Check, X } from 'lucide-react'
import * as api from '../../services/api'

interface Props {
  runId?: string
  taskId?: string
  toolCallId?: string
  title?: string
  message?: string
  riskLevel?: 'low' | 'medium' | 'high'
}

export default function ConfirmActionCard({ runId, taskId, toolCallId, title, message, riskLevel = 'medium' }: Props) {
  const [submitting, setSubmitting] = useState(false)
  const [done, setDone] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const submit = async (confirmed: boolean) => {
    if (!runId || !toolCallId || submitting || done) return
    setSubmitting(true)
    setError(null)
    try {
      await api.sendToolResult({
        runId,
        taskId,
        toolCallId,
        status: confirmed ? 'success' : 'cancelled',
        contentType: 'application/json',
        data: { confirmed },
      })
      setDone(confirmed ? 'confirmed' : 'cancelled')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to submit confirmation')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="mt-2 rounded-lg border border-amber-200 bg-amber-50 p-3">
      <div className="flex items-center gap-2 mb-2">
        <AlertTriangle className="w-4 h-4 text-amber-600" />
        <span className="text-sm font-semibold text-amber-900">{title || 'Confirm action'}</span>
        <span className="text-[10px] uppercase rounded bg-white/70 border border-amber-200 px-1.5 py-0.5 text-amber-700">{riskLevel}</span>
      </div>
      {message && <p className="text-xs text-amber-900 whitespace-pre-wrap mb-3">{message}</p>}
      {error && <p className="text-xs text-red-600 mb-2">{error}</p>}
      {done ? (
        <p className="text-xs text-amber-800">Submitted: {done}</p>
      ) : (
        <div className="flex gap-2">
          <button disabled={submitting || !runId || !toolCallId} onClick={() => submit(true)} className="inline-flex items-center gap-1 px-2 py-1 rounded bg-amber-600 text-white text-xs disabled:opacity-50">
            <Check className="w-3 h-3" /> Confirm
          </button>
          <button disabled={submitting || !runId || !toolCallId} onClick={() => submit(false)} className="inline-flex items-center gap-1 px-2 py-1 rounded bg-white border border-amber-300 text-amber-800 text-xs disabled:opacity-50">
            <X className="w-3 h-3" /> Cancel
          </button>
        </div>
      )}
    </div>
  )
}
