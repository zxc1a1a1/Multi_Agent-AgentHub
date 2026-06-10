import { useState, useMemo, useRef, useEffect } from 'react'
import { useAgentStore } from '../stores/agentStore'
import AgentAvatar from './AgentAvatar'
import { Bot, Search, Check, ChevronDown, X } from 'lucide-react'
import { normalizeAgentName, type AgentName } from '../lib/agents'

interface Props {
  selectedAgentNames: AgentName[]
  onChange: (names: AgentName[]) => void
  disabled?: boolean
}

export default function AgentPicker({ selectedAgentNames, onChange, disabled }: Props) {
  const agentOptions = useAgentStore((s) => s.options)
  const loadAgents = useAgentStore((s) => s.load)
  const isEnabled = useAgentStore((s) => s.isEnabled)
  const [open, setOpen] = useState(false)
  const [search, setSearch] = useState('')
  const containerRef = useRef<HTMLDivElement>(null)
  const inputRef = useRef<HTMLInputElement>(null)

  useEffect(() => {
    loadAgents()
  }, [loadAgents])

  // Close dropdown on outside click
  useEffect(() => {
    if (!open) return
    const handleClick = (e: MouseEvent) => {
      if (containerRef.current && !containerRef.current.contains(e.target as Node)) {
        setOpen(false)
        setSearch('')
      }
    }
    document.addEventListener('mousedown', handleClick)
    return () => document.removeEventListener('mousedown', handleClick)
  }, [open])

  // Close on Escape
  useEffect(() => {
    if (!open) return
    const handleKey = (e: KeyboardEvent) => {
      if (e.key === 'Escape') {
        setOpen(false)
        setSearch('')
      }
    }
    document.addEventListener('keydown', handleKey)
    return () => document.removeEventListener('keydown', handleKey)
  }, [open])

  const filteredOptions = useMemo(() => {
    const query = search.toLowerCase().trim()
    return agentOptions.filter((opt) => {
      if (!query) return true
      return (
        opt.name.toLowerCase().includes(query) ||
        opt.displayName.toLowerCase().includes(query) ||
        opt.description.toLowerCase().includes(query)
      )
    })
  }, [agentOptions, search])

  const selectedDisplay = useMemo(() => {
    if (selectedAgentNames.length === 0 || selectedAgentNames[0] === 'auto') {
      const autoOpt = agentOptions.find((o) => o.name === 'auto')
      return autoOpt?.displayName || 'Auto (Smart)'
    }
    if (selectedAgentNames.length === 1) {
      const opt = agentOptions.find((o) => o.name === selectedAgentNames[0])
      return opt?.displayName || selectedAgentNames[0]
    }
    return `${selectedAgentNames.length} agents selected`
  }, [selectedAgentNames, agentOptions])

  const handleToggle = (name: AgentName) => {
    if (name === 'auto') {
      onChange(['auto'])
      setOpen(false)
      setSearch('')
      return
    }

    const current = selectedAgentNames.includes('auto') ? [] : [...selectedAgentNames]
    const idx = current.indexOf(name)
    if (idx >= 0) {
      const next = current.filter((n) => n !== name)
      onChange(next.length === 0 ? ['auto'] : next)
    } else {
      // Remove 'auto' when adding concrete agents
      const withoutAuto = current.filter((n) => n !== 'auto')
      onChange([...withoutAuto, name])
    }
  }

  const handleRemoveAgent = (name: AgentName) => {
    const next = selectedAgentNames.filter((n) => n !== name)
    onChange(next.length === 0 ? ['auto'] : next)
  }

  return (
    <div ref={containerRef} className="relative">
      {/* Trigger button */}
      <button
        type="button"
        onClick={() => {
          if (!disabled) {
            setOpen(!open)
            setSearch('')
            setTimeout(() => inputRef.current?.focus(), 0)
          }
        }}
        disabled={disabled}
        className="flex items-center gap-1.5 px-2.5 py-1.5 rounded-lg border border-gray-300 bg-white text-sm hover:bg-gray-50 disabled:opacity-50 disabled:cursor-not-allowed transition-colors min-w-0"
      >
        <Bot className="w-4 h-4 text-indigo-500 flex-shrink-0" />
        <span className="truncate max-w-[120px]">{selectedDisplay}</span>
        <ChevronDown className={`w-3.5 h-3.5 text-gray-400 flex-shrink-0 transition-transform ${open ? 'rotate-180' : ''}`} />
      </button>

      {/* Dropdown */}
      {open && (
        <div className="absolute bottom-full left-0 mb-1 w-72 bg-white border border-gray-200 rounded-lg shadow-lg z-50">
          {/* Selected agents summary */}
          {selectedAgentNames.length > 0 && selectedAgentNames[0] !== 'auto' && (
            <div className="px-3 py-2 border-b border-gray-100 flex flex-wrap gap-1">
              {selectedAgentNames.map((name) => {
                const opt = agentOptions.find((o) => o.name === name)
                return (
                  <span
                    key={name}
                    className="inline-flex items-center gap-1 px-2 py-0.5 rounded-full bg-indigo-50 border border-indigo-200 text-xs text-indigo-700"
                  >
                    {opt?.displayName || name}
                    <button
                      type="button"
                      onClick={(e) => {
                        e.stopPropagation()
                        handleRemoveAgent(name)
                      }}
                      className="text-indigo-400 hover:text-indigo-600"
                    >
                      <X className="w-3 h-3" />
                    </button>
                  </span>
                )
              })}
            </div>
          )}

          {/* Search */}
          <div className="px-3 py-2 border-b border-gray-100">
            <div className="flex items-center gap-2 px-2 py-1 rounded-md bg-gray-50 border border-gray-200">
              <Search className="w-3.5 h-3.5 text-gray-400" />
              <input
                ref={inputRef}
                type="text"
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                placeholder="Search agents..."
                className="flex-1 bg-transparent text-sm outline-none placeholder:text-gray-400"
              />
            </div>
          </div>

          {/* Agent list */}
          <div className="max-h-48 overflow-y-auto py-1">
            {filteredOptions.length === 0 ? (
              <div className="px-3 py-4 text-center text-sm text-gray-400">
                No agents found
              </div>
            ) : (
              filteredOptions.map((opt) => {
                const enabled = isEnabled(opt.name)
                const isSelected =
                  opt.name === 'auto'
                    ? selectedAgentNames.includes('auto') || selectedAgentNames.length === 0
                    : selectedAgentNames.includes(opt.name)

                return (
                  <button
                    key={opt.name}
                    type="button"
                    onClick={() => enabled && handleToggle(opt.name)}
                    disabled={!enabled}
                    className={`w-full flex items-center gap-2.5 px-3 py-2 text-left transition-colors ${
                      enabled
                        ? 'hover:bg-gray-50 cursor-pointer'
                        : 'opacity-50 cursor-not-allowed'
                    }`}
                  >
                    {/* Checkbox / radio indicator */}
                    <span
                      className={`w-5 h-5 rounded border-2 flex items-center justify-center flex-shrink-0 transition-colors ${
                        isSelected
                          ? 'bg-indigo-600 border-indigo-600'
                          : 'border-gray-300'
                      }`}
                    >
                      {isSelected && <Check className="w-3 h-3 text-white" />}
                    </span>

                    {/* Avatar */}
                    <AgentAvatar name={opt.name} />

                    {/* Info */}
                    <div className="min-w-0 flex-1">
                      <div className="flex items-center gap-1.5">
                        <span className="text-sm font-medium text-gray-800 truncate">
                          {opt.displayName}
                        </span>
                        {!enabled && (
                          <span className="text-[10px] px-1 py-0.5 rounded bg-gray-200 text-gray-500 flex-shrink-0">
                            disabled
                          </span>
                        )}
                      </div>
                      <p className="text-xs text-gray-400 truncate">{opt.description}</p>
                    </div>
                  </button>
                )
              })
            )}
          </div>
        </div>
      )}
    </div>
  )
}
