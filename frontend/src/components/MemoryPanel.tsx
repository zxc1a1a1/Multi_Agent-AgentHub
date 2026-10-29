import { useCallback, useEffect, useState } from 'react'
import { AlertTriangle, Brain, Check, Pencil, RefreshCw, Trash2 } from 'lucide-react'
import {
  confirmMemory,
  forgetMemory,
  forgetMemoryScope,
  getMemoryPolicy,
  listMemoryEntries,
  memoryScopeSnapshot,
  updateMemory,
  updateMemoryPolicy,
  type MemoryEntry,
  type MemoryMode,
  type MemoryPolicy,
} from '../services/api'

const MODE_LABEL: Record<MemoryMode, string> = {
  off: 'Off',
  session_only: 'Session only',
  scoped_long_term: 'Scoped long-term',
}

function withMode(policy: MemoryPolicy, mode: MemoryMode): MemoryPolicy {
  const enabled = mode === 'scoped_long_term'
  return { ...policy, mode, read: enabled, propose: enabled, confirm: enabled, forget: enabled }
}

export default function MemoryPanel() {
  const [policy, setPolicy] = useState<MemoryPolicy | null>(null)
  const [entries, setEntries] = useState<MemoryEntry[]>([])
  const [loading, setLoading] = useState(false)
  const [error, setError] = useState('')
  const [notice, setNotice] = useState('')

  const scope = memoryScopeSnapshot()

  const load = useCallback(async () => {
    setLoading(true)
    setError('')
    try {
      const [nextPolicy, nextEntries] = await Promise.all([getMemoryPolicy(), listMemoryEntries()])
      setPolicy(nextPolicy)
      setEntries(nextEntries)
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to load memory')
    } finally {
      setLoading(false)
    }
  }, [])

  useEffect(() => {
    void load()
  }, [load])

  const changeMode = async (mode: MemoryMode) => {
    if (!policy) return
    setError('')
    setNotice('')
    try {
      const saved = await updateMemoryPolicy(withMode(policy, mode))
      setPolicy(saved)
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update memory mode')
    }
  }

  const confirmEntry = async (entry: MemoryEntry) => {
    setError('')
    setNotice('')
    try {
      await confirmMemory(entry.memoryId, entry.revision)
      setNotice('Memory confirmed and active.')
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to confirm memory')
    }
  }

  const editEntry = async (entry: MemoryEntry) => {
    const next = window.prompt('Correct memory content', entry.content)
    if (next === null || next.trim() === '' || next === entry.content) return
    setError('')
    setNotice('')
    try {
      await updateMemory(entry.memoryId, next, entry.revision)
      setNotice('Memory corrected with a new revision.')
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to update memory (revision conflict?)')
    }
  }

  const forgetScope = async () => {
    const ok = window.confirm(
      'Forget all long-term memories in this scope? This does not delete source chats or backups.',
    )
    if (!ok) return
    setError('')
    setNotice('')
    try {
      const result = await forgetMemoryScope()
      setNotice(
        `Revoked ${result.revokedCount} memories; recall disabled: ${result.recallDisabled ? 'yes' : 'no'}; purge status: ${result.purgeStatus}.`,
      )
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to forget scope')
    }
  }

  const forgetEntry = async (entry: MemoryEntry) => {
    const ok = window.confirm(
      'Stop recalling this memory? This does not delete the original chat or its source messages.',
    )
    if (!ok) return
    setError('')
    setNotice('')
    try {
      const result = await forgetMemory(entry.memoryId)
      setNotice(
        `Recall disabled: ${result.recallDisabled ? 'yes' : 'no'}; purge status: ${result.purgeStatus}.`,
      )
      await load()
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Failed to forget memory')
    }
  }

  return (
    <div className="flex h-full flex-col bg-white">
      <header className="flex items-center justify-between border-b border-gray-200 px-6 py-4">
        <div className="flex items-center gap-2">
          <Brain className="h-5 w-5 text-indigo-600" />
          <div>
            <h2 className="text-lg font-semibold text-gray-900">Memory</h2>
            <p className="text-xs text-gray-500">
              Scope: {scope.env} / {scope.workspaceId} / {scope.principalId}
            </p>
          </div>
        </div>
        <button
          type="button"
          onClick={() => void load()}
          className="inline-flex items-center gap-1 rounded-md border border-gray-300 px-3 py-1.5 text-sm text-gray-700 hover:bg-gray-50"
          disabled={loading}
        >
          <RefreshCw className="h-4 w-4" /> Refresh
        </button>
      </header>

      <div className="flex-1 overflow-y-auto px-6 py-4">
        {policy && (
          <div className="mb-4 rounded-lg border border-gray-200 p-4">
            <label className="text-sm font-medium text-gray-700" htmlFor="memory-mode">
              Memory mode
            </label>
            <select
              id="memory-mode"
              value={policy.mode}
              onChange={(e) => void changeMode(e.target.value as MemoryMode)}
              className="mt-1 block rounded-md border border-gray-300 px-3 py-1.5 text-sm"
            >
              {(Object.keys(MODE_LABEL) as MemoryMode[]).map((mode) => (
                <option key={mode} value={mode}>
                  {MODE_LABEL[mode]}
                </option>
              ))}
            </select>
            <p className="mt-2 text-xs text-gray-500">
              {policy.mode === 'off' && 'No past session reuse. Current request and safety data still apply.'}
              {policy.mode === 'session_only' && 'Only the current authorized thread; no cross-session recall.'}
              {policy.mode === 'scoped_long_term' && 'Active memory in this scope only; new entries require confirmation.'}
            </p>
            <button
              type="button"
              onClick={() => void forgetScope()}
              className="mt-3 rounded-md border border-red-300 px-3 py-1.5 text-sm text-red-700 hover:bg-red-50"
            >
              Forget all in this scope
            </button>
          </div>
        )}

        {error && (
          <div className="mb-3 flex items-start gap-2 rounded-md border border-red-200 bg-red-50 p-3 text-sm text-red-700">
            <AlertTriangle className="mt-0.5 h-4 w-4" /> {error}
          </div>
        )}
        {notice && <div className="mb-3 rounded-md border border-green-200 bg-green-50 p-3 text-sm text-green-700">{notice}</div>}

        {entries.length === 0 && !loading && (
          <p className="text-sm text-gray-500">No memory entries in this scope.</p>
        )}
        <ul className="space-y-3">
          {entries.map((entry) => (
            <li key={entry.memoryId} className="rounded-lg border border-gray-200 p-4">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <div className="flex items-center gap-2">
                    <span className="text-sm font-medium text-gray-900">{entry.logicalKey}</span>
                    <span
                      className={`rounded-full px-2 py-0.5 text-xs ${
                        entry.status === 'active'
                          ? 'bg-green-100 text-green-700'
                          : entry.status === 'candidate'
                            ? 'bg-amber-100 text-amber-700'
                            : 'bg-gray-100 text-gray-600'
                      }`}
                    >
                      {entry.status}
                    </span>
                    <span className="text-xs text-gray-400">rev {entry.revision}</span>
                  </div>
                  <p className="mt-1 whitespace-pre-wrap text-sm text-gray-700">{entry.content}</p>
                  <p className="mt-1 text-xs text-gray-500">
                    {entry.kind}
                    {entry.expiresAt ? ` · expires ${new Date(entry.expiresAt).toLocaleString()}` : ''}
                    {entry.sourceRefs.length > 0 ? ` · sources: ${entry.sourceRefs.join(', ')}` : ''}
                  </p>
                </div>
                <div className="flex flex-shrink-0 items-center gap-1">
                  {entry.status === 'candidate' && (
                    <button
                      type="button"
                      onClick={() => void confirmEntry(entry)}
                      className="rounded p-1.5 text-green-600 hover:bg-green-50"
                      title="Confirm memory"
                    >
                      <Check className="h-4 w-4" />
                    </button>
                  )}
                  {entry.status === 'active' && (
                    <button
                      type="button"
                      onClick={() => void editEntry(entry)}
                      className="rounded p-1.5 text-gray-600 hover:bg-gray-50"
                      title="Correct memory"
                    >
                      <Pencil className="h-4 w-4" />
                    </button>
                  )}
                  {entry.status !== 'revoked' && (
                    <button
                      type="button"
                      onClick={() => void forgetEntry(entry)}
                      className="rounded p-1.5 text-red-600 hover:bg-red-50"
                      title="Forget memory"
                    >
                      <Trash2 className="h-4 w-4" />
                    </button>
                  )}
                </div>
              </div>
            </li>
          ))}
        </ul>
        <p className="mt-6 text-xs text-gray-500">
          Forgetting memory stops future recall; it does not delete the original conversation or backup data.
        </p>
      </div>
    </div>
  )
}
