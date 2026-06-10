import { useMemo, useState } from 'react'
import { Send, X } from 'lucide-react'
import * as api from '../../services/api'

interface Props {
  runId?: string
  taskId?: string
  toolCallId?: string
  title?: string
  schema?: Record<string, unknown>
}

function schemaProperties(schema?: Record<string, unknown>): Record<string, Record<string, unknown>> {
  const props = schema?.properties
  if (!props || typeof props !== 'object' || Array.isArray(props)) return {}
  return props as Record<string, Record<string, unknown>>
}

function requiredFields(schema?: Record<string, unknown>): string[] {
  return Array.isArray(schema?.required) ? schema.required.map(String) : []
}

export default function FormInputCard({ runId, taskId, toolCallId, title, schema }: Props) {
  const props = useMemo(() => schemaProperties(schema), [schema])
  const required = useMemo(() => requiredFields(schema), [schema])
  const [values, setValues] = useState<Record<string, string>>({})
  const [submitting, setSubmitting] = useState(false)
  const [done, setDone] = useState<string | null>(null)
  const [error, setError] = useState<string | null>(null)

  const validate = () => {
    for (const field of required) {
      if (!String(values[field] || '').trim()) return `${field} is required`
    }
    return ''
  }

  const submit = async (cancelled = false) => {
    if (!runId || !toolCallId || submitting || done) return
    const validation = cancelled ? '' : validate()
    if (validation) { setError(validation); return }
    setSubmitting(true)
    setError(null)
    try {
      await api.sendToolResult({
        runId,
        taskId,
        toolCallId,
        status: cancelled ? 'cancelled' : 'success',
        contentType: 'application/json',
        data: cancelled ? { cancelled: true } : values,
      })
      setDone(cancelled ? 'cancelled' : 'submitted')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to submit form')
    } finally {
      setSubmitting(false)
    }
  }

  return (
    <div className="mt-2 rounded-lg border border-indigo-200 bg-white p-3">
      <h4 className="text-sm font-semibold text-gray-800 mb-2">{title || 'Input required'}</h4>
      {Object.keys(props).length === 0 && <p className="text-xs text-gray-500 mb-2">Unsupported schema; submitting empty JSON.</p>}
      <div className="space-y-2">
        {Object.entries(props).map(([name, def]) => (
          <label key={name} className="block text-xs text-gray-600">
            <span>{name}{required.includes(name) ? ' *' : ''}</span>
            <input
              className="mt-1 w-full rounded border border-gray-300 px-2 py-1 text-sm"
              value={values[name] || ''}
              placeholder={typeof def.description === 'string' ? def.description : ''}
              onChange={(e) => setValues((v) => ({ ...v, [name]: e.target.value }))}
            />
          </label>
        ))}
      </div>
      {error && <p className="text-xs text-red-600 mt-2">{error}</p>}
      {done ? <p className="text-xs text-green-600 mt-2">{done}</p> : (
        <div className="flex gap-2 mt-3">
          <button disabled={submitting || !runId || !toolCallId} onClick={() => submit(false)} className="inline-flex items-center gap-1 px-2 py-1 rounded bg-indigo-600 text-white text-xs disabled:opacity-50">
            <Send className="w-3 h-3" /> Submit
          </button>
          <button disabled={submitting || !runId || !toolCallId} onClick={() => submit(true)} className="inline-flex items-center gap-1 px-2 py-1 rounded bg-white border border-gray-300 text-gray-600 text-xs disabled:opacity-50">
            <X className="w-3 h-3" /> Cancel
          </button>
        </div>
      )}
    </div>
  )
}
