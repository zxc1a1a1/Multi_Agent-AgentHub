import { useState } from 'react'
import { useAgentStore } from '../../stores/agentStore'
import { Plus, Loader2 } from 'lucide-react'

export default function RegisterAgentPanel() {
  const register = useAgentStore((s) => s.register)
  const actionLoading = useAgentStore((s) => s.actionLoading)

  const [name, setName] = useState('')
  const [url, setUrl] = useState('')
  const [replace, setReplace] = useState(false)
  const [error, setError] = useState<string | null>(null)
  const [success, setSuccess] = useState<string | null>(null)

  const trimmedName = name.trim()
  const trimmedUrl = url.trim()
  const isValid = trimmedName !== '' && trimmedUrl !== ''
  const inFlight = actionLoading[trimmedName] || false

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault()
    if (!isValid || inFlight) return

    setError(null)
    setSuccess(null)

    try {
      const agent = await register({
        name: trimmedName,
        url: trimmedUrl,
        replace: replace || undefined,
      })
      setSuccess(`Agent "${agent.displayName || agent.name}" registered successfully.`)
      setName('')
      setUrl('')
      setReplace(false)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Registration failed')
    }
  }

  return (
    <div className="border border-gray-200 rounded-lg p-4 bg-white">
      <h3 className="text-sm font-semibold text-gray-700 mb-3 flex items-center gap-1.5">
        <Plus className="w-4 h-4" />
        Register Agent
      </h3>

      {success && (
        <div className="mb-3 p-2 bg-green-50 border border-green-200 rounded text-sm text-green-700 flex items-center justify-between">
          {success}
          <button
            className="text-green-600 hover:text-green-800 underline text-xs ml-2"
            onClick={() => setSuccess(null)}
          >
            Dismiss
          </button>
        </div>
      )}

      {error && (
        <div className="mb-3 p-2 bg-red-50 border border-red-200 rounded text-sm text-red-600 flex items-center justify-between">
          {error}
          <button
            className="text-red-600 hover:text-red-800 underline text-xs ml-2"
            onClick={() => setError(null)}
          >
            Dismiss
          </button>
        </div>
      )}

      <form onSubmit={handleSubmit} className="space-y-3">
        <div>
          <label htmlFor="agent-name" className="block text-xs text-gray-500 mb-1">
            Agent Name
          </label>
          <input
            id="agent-name"
            type="text"
            value={name}
            onChange={(e) => setName(e.target.value)}
            placeholder="e.g. my-code-agent"
            disabled={inFlight}
            className="w-full rounded-md border border-gray-300 px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent disabled:bg-gray-50"
          />
        </div>

        <div>
          <label htmlFor="agent-url" className="block text-xs text-gray-500 mb-1">
            Agent URL
          </label>
          <input
            id="agent-url"
            type="text"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="http://host:port"
            disabled={inFlight}
            className="w-full rounded-md border border-gray-300 px-3 py-1.5 text-sm focus:outline-none focus:ring-2 focus:ring-indigo-500 focus:border-transparent disabled:bg-gray-50"
          />
        </div>

        <div className="flex items-center justify-between">
          <label className="flex items-center gap-2 text-xs text-gray-500">
            <input
              type="checkbox"
              checked={replace}
              onChange={(e) => setReplace(e.target.checked)}
              disabled={inFlight}
              className="rounded border-gray-300"
            />
            Replace if exists
          </label>

          <button
            type="submit"
            disabled={!isValid || inFlight}
            className="inline-flex items-center gap-1.5 px-4 py-1.5 rounded-md bg-indigo-600 text-white text-sm font-medium hover:bg-indigo-700 disabled:opacity-50 disabled:cursor-not-allowed transition-colors"
          >
            {inFlight ? (
              <>
                <Loader2 className="w-3.5 h-3.5 animate-spin" />
                Registering…
              </>
            ) : (
              <>
                <Plus className="w-3.5 h-3.5" />
                Register
              </>
            )}
          </button>
        </div>
      </form>
    </div>
  )
}
