import { useMemo } from 'react'
import { useAgentStore } from '../stores/agentStore'
import AgentAvatar from './AgentAvatar'

interface Props {
  query: string
  onSelect: (name: string) => void
  knownAgentNames?: string[]
}

export default function MentionDropdown({ query, onSelect, knownAgentNames }: Props) {
  const agentOptions = useAgentStore((s) => s.options)
  const isEnabled = useAgentStore((s) => s.isEnabled)

  const filtered = useMemo(() => {
    const q = query.toLowerCase().trim()
    if (!q) {
      // Show all agents when dropdown first opens (empty query), exclude 'auto'
      return agentOptions.filter((opt) => {
        if (opt.name === 'auto') return false
        if (knownAgentNames && knownAgentNames.length > 0 && !knownAgentNames.includes(opt.name)) {
          return false
        }
        return isEnabled(opt.name)
      })
    }
    return agentOptions.filter((opt) => {
      if (opt.name === 'auto') return false
      if (knownAgentNames && knownAgentNames.length > 0 && !knownAgentNames.includes(opt.name)) {
        return false
      }
      if (!isEnabled(opt.name)) return false
      return (
        opt.name.toLowerCase().includes(q) ||
        opt.displayName.toLowerCase().includes(q)
      )
    })
  }, [agentOptions, query, knownAgentNames, isEnabled])

  if (filtered.length === 0) return null

  return (
    <div className="absolute bottom-full left-0 mb-1 w-64 bg-white border border-gray-200 rounded-lg shadow-lg z-50 max-h-48 overflow-y-auto py-1">
      {filtered.map((opt) => (
        <button
          key={opt.name}
          type="button"
          onClick={() => onSelect(opt.name)}
          className="w-full flex items-center gap-2 px-3 py-2 text-left hover:bg-gray-50 transition-colors"
        >
          <AgentAvatar name={opt.name} />
          <div className="min-w-0 flex-1">
            <span className="text-sm font-medium text-gray-800 block truncate">
              {opt.displayName}
            </span>
            <span className="text-xs text-gray-400 font-mono">@{opt.name}</span>
          </div>
        </button>
      ))}
    </div>
  )
}
