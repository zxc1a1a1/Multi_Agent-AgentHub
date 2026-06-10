import { useEffect, useState } from 'react'
import { X, Copy, Loader2 } from 'lucide-react'
import * as api from '../../services/api'
import type { AgentManagement } from '../../types'

interface Props {
  agent: AgentManagement
  onClose: () => void
}

export default function AgentCardJsonDialog({ agent, onClose }: Props) {
  const [cardJson, setCardJson] = useState<string | null>(null)
  const [loading, setLoading] = useState(true)
  const [error, setError] = useState<string | null>(null)
  const [copied, setCopied] = useState(false)

  useEffect(() => {
    let cancelled = false
    setLoading(true)
    setError(null)

    api
      .getAgent(agent.name)
      .then((data) => {
        if (cancelled) return
        setCardJson(JSON.stringify(data, null, 2))
        setLoading(false)
      })
      .catch((err) => {
        if (cancelled) return
        setError(err instanceof Error ? err.message : 'Failed to load agent card')
        setLoading(false)
      })

    return () => {
      cancelled = true
    }
  }, [agent.name])

  const handleCopy = async () => {
    if (!cardJson) return
    try {
      await navigator.clipboard.writeText(cardJson)
      setCopied(true)
      setTimeout(() => setCopied(false), 2000)
    } catch {
      // Clipboard API may not be available
    }
  }

  // Close on Escape key.
  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose()
    }
    document.addEventListener('keydown', handleKeyDown)
    return () => document.removeEventListener('keydown', handleKeyDown)
  }, [onClose])

  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-black/40"
      onClick={(e) => {
        if (e.target === e.currentTarget) onClose()
      }}
    >
      <div className="bg-white rounded-lg shadow-xl w-full max-w-lg mx-4 max-h-[80vh] flex flex-col">
        {/* Header */}
        <div className="flex items-center justify-between px-4 py-3 border-b border-gray-200">
          <h3 className="text-sm font-semibold text-gray-800">
            Agent Card: {agent.displayName || agent.name}
          </h3>
          <button
            onClick={onClose}
            className="p-1 rounded hover:bg-gray-100 text-gray-400 hover:text-gray-600 transition-colors"
          >
            <X className="w-4 h-4" />
          </button>
        </div>

        {/* Body */}
        <div className="flex-1 overflow-auto p-4">
          {loading && (
            <div className="flex items-center justify-center py-8 text-gray-400 gap-2">
              <Loader2 className="w-5 h-5 animate-spin" />
              <span className="text-sm">Loading agent card…</span>
            </div>
          )}

          {error && (
            <div className="p-3 bg-red-50 border border-red-200 rounded text-sm text-red-600">
              {error}
            </div>
          )}

          {cardJson && (
            <pre className="text-xs text-gray-800 bg-gray-50 rounded-lg p-3 overflow-x-auto whitespace-pre-wrap break-all font-mono">
              {cardJson}
            </pre>
          )}
        </div>

        {/* Footer */}
        {cardJson && (
          <div className="px-4 py-2 border-t border-gray-200 flex justify-end">
            <button
              onClick={handleCopy}
              className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-gray-100 text-gray-700 text-sm hover:bg-gray-200 transition-colors"
            >
              <Copy className="w-3.5 h-3.5" />
              {copied ? 'Copied!' : 'Copy JSON'}
            </button>
          </div>
        )}
      </div>
    </div>
  )
}
