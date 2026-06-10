import { useEffect, useState, useCallback } from 'react'
import { useAgentStore } from '../../stores/agentStore'
import AgentStatusBadge from './AgentStatusBadge'
import AgentTagBadge from './AgentTagBadge'
import RegisterAgentPanel from './RegisterAgentPanel'
import AgentCardJsonDialog from './AgentCardJsonDialog'
import type { AgentManagement } from '../../types'
import {
  RefreshCw,
  Trash2,
  Eye,
  Power,
  PowerOff,
  Activity,
  Loader2,
  AlertTriangle,
  Bot,
  ChevronDown,
  ChevronUp,
  Server,
} from 'lucide-react'

export default function AgentDirectory() {
  const agents = useAgentStore((s) => s.agents)
  const mgmtLoading = useAgentStore((s) => s.mgmtLoading)
  const mgmtError = useAgentStore((s) => s.mgmtError)
  const mgmtLoaded = useAgentStore((s) => s.mgmtLoaded)
  const actionLoading = useAgentStore((s) => s.actionLoading)
  const loadManagement = useAgentStore((s) => s.loadManagement)
  const toggleEnable = useAgentStore((s) => s.toggleEnable)
  const unregister = useAgentStore((s) => s.unregister)
  const refresh = useAgentStore((s) => s.refresh)
  const check = useAgentStore((s) => s.check)
  const clearMgmtError = useAgentStore((s) => s.clearMgmtError)

  const [cardDialogAgent, setCardDialogAgent] = useState<AgentManagement | null>(null)
  const [deleteConfirm, setDeleteConfirm] = useState<string | null>(null)
  const [expanded, setExpanded] = useState<Record<string, boolean>>({})
  const [actionErrors, setActionErrors] = useState<Record<string, string>>({})

  useEffect(() => {
    loadManagement()
  }, [loadManagement])

  const handleToggle = useCallback(
    async (name: string, enabled: boolean) => {
      setActionErrors((prev) => {
        const next = { ...prev }
        delete next[name]
        return next
      })
      try {
        await toggleEnable(name, enabled)
      } catch (err) {
        setActionErrors((prev) => ({
          ...prev,
          [name]: err instanceof Error ? err.message : 'Action failed',
        }))
      }
    },
    [toggleEnable],
  )

  const handleRefresh = useCallback(
    async (name: string) => {
      setActionErrors((prev) => {
        const next = { ...prev }
        delete next[name]
        return next
      })
      try {
        await refresh(name)
      } catch (err) {
        setActionErrors((prev) => ({
          ...prev,
          [name]: err instanceof Error ? err.message : 'Refresh failed',
        }))
      }
    },
    [refresh],
  )

  const handleCheck = useCallback(
    async (name: string) => {
      setActionErrors((prev) => {
        const next = { ...prev }
        delete next[name]
        return next
      })
      try {
        await check(name)
      } catch (err) {
        setActionErrors((prev) => ({
          ...prev,
          [name]: err instanceof Error ? err.message : 'Health check failed',
        }))
      }
    },
    [check],
  )

  const handleDelete = useCallback(
    async (name: string) => {
      setActionErrors((prev) => {
        const next = { ...prev }
        delete next[name]
        return next
      })
      try {
        await unregister(name)
        setDeleteConfirm(null)
      } catch (err) {
        setActionErrors((prev) => ({
          ...prev,
          [name]: err instanceof Error ? err.message : 'Delete failed',
        }))
        setDeleteConfirm(null)
      }
    },
    [unregister],
  )

  const toggleExpand = (name: string) => {
    setExpanded((prev) => ({ ...prev, [name]: !prev[name] }))
  }

  // -----------------------------------------------------------------------
  // Loading state
  // -----------------------------------------------------------------------
  if (mgmtLoading && !mgmtLoaded) {
    return (
      <div className="flex-1 overflow-y-auto p-6">
        <div className="max-w-4xl mx-auto space-y-4">
          <div className="flex items-center justify-between mb-6">
            <h2 className="text-lg font-semibold text-gray-800">Agent Directory</h2>
          </div>
          {[1, 2, 3].map((i) => (
            <div key={i} className="border border-gray-100 rounded-lg p-4 animate-pulse">
              <div className="flex items-center gap-3">
                <div className="w-8 h-8 bg-gray-200 rounded-full" />
                <div className="flex-1 space-y-2">
                  <div className="h-4 bg-gray-200 rounded w-1/3" />
                  <div className="h-3 bg-gray-100 rounded w-1/2" />
                </div>
              </div>
            </div>
          ))}
        </div>
      </div>
    )
  }

  // -----------------------------------------------------------------------
  // Error state
  // -----------------------------------------------------------------------
  if (mgmtError && agents.length === 0) {
    return (
      <div className="flex-1 overflow-y-auto p-6">
        <div className="max-w-4xl mx-auto">
          <h2 className="text-lg font-semibold text-gray-800 mb-4">Agent Directory</h2>
          <div className="border border-red-200 bg-red-50 rounded-lg p-6 text-center">
            <AlertTriangle className="w-8 h-8 text-red-400 mx-auto mb-3" />
            <p className="text-red-700 font-medium text-sm mb-2">Failed to load agents</p>
            <p className="text-red-500 text-xs mb-4">{mgmtError}</p>
            <div className="flex gap-2 justify-center">
              <button
                onClick={() => {
                  clearMgmtError()
                  loadManagement()
                }}
                className="px-4 py-1.5 bg-red-600 text-white text-sm rounded-md hover:bg-red-700 transition-colors"
              >
                Retry
              </button>
            </div>
          </div>
        </div>
      </div>
    )
  }

  // -----------------------------------------------------------------------
  // Populated / Empty state
  // -----------------------------------------------------------------------
  return (
    <div className="flex-1 overflow-y-auto p-6">
      <div className="max-w-4xl mx-auto space-y-6">
        <div className="flex items-center justify-between">
          <div>
            <h2 className="text-lg font-semibold text-gray-800">Agent Directory</h2>
            <p className="text-xs text-gray-500 mt-0.5">
              {agents.length} agent{agents.length !== 1 ? 's' : ''} registered
            </p>
          </div>
          <button
            onClick={() => loadManagement()}
            disabled={mgmtLoading}
            className="inline-flex items-center gap-1.5 px-3 py-1.5 rounded-md bg-gray-100 text-gray-600 text-sm hover:bg-gray-200 disabled:opacity-50 transition-colors"
          >
            <RefreshCw className={`w-3.5 h-3.5 ${mgmtLoading ? 'animate-spin' : ''}`} />
            Refresh
          </button>
        </div>

        {/* Register agent form */}
        <RegisterAgentPanel />

        {/* Empty state */}
        {agents.length === 0 && (
          <div className="border border-gray-200 rounded-lg p-8 text-center bg-white">
            <Server className="w-10 h-10 text-gray-300 mx-auto mb-3" />
            <p className="text-gray-600 text-sm font-medium">No agents registered</p>
            <p className="text-gray-400 text-xs mt-1">
              Register your first agent using the form above.
            </p>
          </div>
        )}

        {/* Agent list */}
        {agents
          .filter((a) => a.name !== 'auto')
          .map((agent) => {
            const isExpanded = expanded[agent.name] || false
            const inFlight = actionLoading[agent.name] || false
            const err = actionErrors[agent.name] || null
            const isDynamic = agent.source === 'dynamic'
            const isDeleting = deleteConfirm === agent.name

            return (
              <div
                key={agent.name}
                className="border border-gray-200 rounded-lg bg-white overflow-hidden"
              >
                {/* Summary row */}
                <div className="flex items-center gap-3 px-4 py-3">
                  {/* Expand toggle */}
                  <button
                    onClick={() => toggleExpand(agent.name)}
                    className="p-0.5 text-gray-400 hover:text-gray-600 transition-colors"
                  >
                    {isExpanded ? (
                      <ChevronUp className="w-4 h-4" />
                    ) : (
                      <ChevronDown className="w-4 h-4" />
                    )}
                  </button>

                  {/* Icon */}
                  <div className="w-8 h-8 rounded-full bg-indigo-100 flex items-center justify-center flex-shrink-0">
                    <Bot className="w-4 h-4 text-indigo-500" />
                  </div>

                  {/* Name + source */}
                  <div className="min-w-0 flex-1">
                    <div className="flex items-center gap-2">
                      <span className="text-sm font-medium text-gray-800 truncate">
                        {agent.displayName || agent.name}
                      </span>
                      <AgentTagBadge label={agent.source} variant="source" />
                    </div>
                    <div className="flex items-center gap-2 mt-0.5">
                      <span className="text-xs text-gray-400 font-mono">{agent.name}</span>
                      <AgentStatusBadge enabled={agent.enabled} healthy={agent.healthy} />
                    </div>
                  </div>

                  {/* Action buttons */}
                  <div className="flex items-center gap-1 flex-shrink-0">
                    {/* Enable / Disable — dynamic only */}
                    {isDynamic && (
                      <button
                        onClick={() => handleToggle(agent.name, !agent.enabled)}
                        disabled={inFlight}
                        className="p-1.5 rounded text-gray-400 hover:text-gray-600 hover:bg-gray-100 disabled:opacity-50 transition-colors"
                        title={agent.enabled ? 'Disable' : 'Enable'}
                      >
                        {agent.enabled ? (
                          <PowerOff className="w-4 h-4" />
                        ) : (
                          <Power className="w-4 h-4" />
                        )}
                      </button>
                    )}

                    {/* Refresh */}
                    {isDynamic && (
                      <button
                        onClick={() => handleRefresh(agent.name)}
                        disabled={inFlight}
                        className="p-1.5 rounded text-gray-400 hover:text-gray-600 hover:bg-gray-100 disabled:opacity-50 transition-colors"
                        title="Refresh agent card"
                      >
                        <RefreshCw className={`w-4 h-4 ${inFlight ? 'animate-spin' : ''}`} />
                      </button>
                    )}

                    {/* Health check */}
                    <button
                      onClick={() => handleCheck(agent.name)}
                      disabled={inFlight}
                      className="p-1.5 rounded text-gray-400 hover:text-gray-600 hover:bg-gray-100 disabled:opacity-50 transition-colors"
                      title="Check health"
                    >
                      <Activity className="w-4 h-4" />
                    </button>

                    {/* View card JSON */}
                    <button
                      onClick={() => setCardDialogAgent(agent)}
                      className="p-1.5 rounded text-gray-400 hover:text-gray-600 hover:bg-gray-100 transition-colors"
                      title="View agent card JSON"
                    >
                      <Eye className="w-4 h-4" />
                    </button>

                    {/* Delete — dynamic only */}
                    {isDynamic &&
                      (isDeleting ? (
                        <div className="flex items-center gap-1 ml-1">
                          <button
                            onClick={() => handleDelete(agent.name)}
                            disabled={inFlight}
                            className="px-2 py-1 bg-red-500 text-white text-xs rounded hover:bg-red-600 disabled:opacity-50"
                          >
                            {inFlight ? (
                              <Loader2 className="w-3 h-3 animate-spin" />
                            ) : (
                              'Delete'
                            )}
                          </button>
                          <button
                            onClick={() => setDeleteConfirm(null)}
                            disabled={inFlight}
                            className="px-2 py-1 bg-gray-200 text-gray-600 text-xs rounded hover:bg-gray-300 disabled:opacity-50"
                          >
                            Cancel
                          </button>
                        </div>
                      ) : (
                        <button
                          onClick={() => setDeleteConfirm(agent.name)}
                          disabled={inFlight}
                          className="p-1.5 rounded text-gray-400 hover:text-red-500 hover:bg-red-50 disabled:opacity-50 transition-colors"
                          title="Delete agent"
                        >
                          <Trash2 className="w-4 h-4" />
                        </button>
                      ))}
                  </div>
                </div>

                {/* Action error */}
                {err && (
                  <div className="px-4 pb-2">
                    <div className="p-2 bg-red-50 border border-red-200 rounded text-xs text-red-600 flex items-center justify-between">
                      {err}
                      <button
                        className="underline text-red-500 hover:text-red-700 ml-2"
                        onClick={() =>
                          setActionErrors((prev) => {
                            const next = { ...prev }
                            delete next[agent.name]
                            return next
                          })
                        }
                      >
                        Dismiss
                      </button>
                    </div>
                  </div>
                )}

                {/* Expanded detail */}
                {isExpanded && (
                  <div className="border-t border-gray-100 px-4 py-3 bg-gray-50/50 space-y-2">
                    {/* Description */}
                    {agent.description && (
                      <div>
                        <span className="text-xs text-gray-400">Description</span>
                        <p className="text-sm text-gray-700 mt-0.5">{agent.description}</p>
                      </div>
                    )}

                    {/* Capabilities */}
                    {agent.capabilities && agent.capabilities.length > 0 && (
                      <div>
                        <span className="text-xs text-gray-400">Capabilities</span>
                        <div className="flex flex-wrap gap-1 mt-1">
                          {agent.capabilities.map((cap) => (
                            <AgentTagBadge key={cap} label={cap} />
                          ))}
                        </div>
                      </div>
                    )}

                    {/* Output modes */}
                    {agent.outputModes && agent.outputModes.length > 0 && (
                      <div>
                        <span className="text-xs text-gray-400">Output Modes</span>
                        <div className="flex flex-wrap gap-1 mt-1">
                          {agent.outputModes.map((mode) => (
                            <AgentTagBadge key={mode} label={mode} />
                          ))}
                        </div>
                      </div>
                    )}

                    {/* Base URL */}
                    {agent.baseURL && (
                      <div>
                        <span className="text-xs text-gray-400">Base URL</span>
                        <p className="text-sm text-gray-700 mt-0.5 font-mono text-xs truncate">
                          {agent.baseURL}
                        </p>
                      </div>
                    )}

                    {/* Timestamps */}
                    <div className="flex gap-6 text-xs text-gray-400">
                      {agent.createdAt && (
                        <span>
                          Created: {new Date(agent.createdAt).toLocaleString()}
                        </span>
                      )}
                      {agent.updatedAt && (
                        <span>
                          Updated: {new Date(agent.updatedAt).toLocaleString()}
                        </span>
                      )}
                    </div>

                    {/* Last error */}
                    {agent.lastError && (
                      <div className="p-2 bg-red-50 border border-red-100 rounded text-xs text-red-600">
                        <span className="font-medium">Last Error:</span> {agent.lastError}
                      </div>
                    )}
                  </div>
                )}
              </div>
            )
          })}
      </div>

      {/* Agent Card JSON dialog */}
      {cardDialogAgent && (
        <AgentCardJsonDialog
          agent={cardDialogAgent}
          onClose={() => setCardDialogAgent(null)}
        />
      )}
    </div>
  )
}
