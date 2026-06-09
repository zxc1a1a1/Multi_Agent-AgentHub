import { create } from 'zustand'
import * as api from '../services/api'
import type { AgentManagement, RegisterAgentRequest } from '../types'
import {
  AGENT_OPTIONS,
  DEFAULT_AGENT_NAME,
  buildAgentOptionsFromSummary,
  getAgentDisplayName,
  normalizeAgentName,
  type AgentName,
  type AgentOption,
} from '../lib/agents'

interface AgentState {
  // Pickers / dropdown (existing)
  options: AgentOption[]
  loading: boolean
  loaded: boolean
  load: () => Promise<void>
  defaultAgentName: () => AgentName
  findOption: (name: string | null | undefined) => AgentOption | undefined
  getDisplayName: (name: string | null | undefined) => string

  // Management / directory
  agents: AgentManagement[]
  mgmtLoading: boolean
  mgmtLoaded: boolean
  mgmtError: string | null
  actionLoading: Record<string, boolean> // agentName → in-flight

  loadManagement: () => Promise<void>
  register: (req: RegisterAgentRequest) => Promise<AgentManagement>
  unregister: (name: string) => Promise<void>
  toggleEnable: (name: string, enabled: boolean) => Promise<AgentManagement>
  refresh: (name: string) => Promise<AgentManagement>
  check: (name: string) => Promise<AgentManagement>
  isEnabled: (name: string | null | undefined) => boolean
  clearMgmtError: () => void
}

function fallbackOptions(): AgentOption[] {
  return AGENT_OPTIONS.map((option) => ({
    ...option,
    outputModes: [...option.outputModes],
  }))
}

export const useAgentStore = create<AgentState>((set, get) => ({
  options: fallbackOptions(),
  loading: false,
  loaded: false,

  load: async () => {
    if (get().loading || get().loaded) {
      return
    }
    set({ loading: true })
    try {
      const summaries = await api.listAgents()
      const options = buildAgentOptionsFromSummary(summaries)
      set({
        options: options.length > 0 ? options : fallbackOptions(),
        loading: false,
        loaded: true,
      })
    } catch {
      set({
        options: fallbackOptions(),
        loading: false,
        loaded: true,
      })
    }
  },

  defaultAgentName: () => {
    return DEFAULT_AGENT_NAME
  },

  findOption: (name: string | null | undefined) => {
    const normalized = normalizeAgentName(name)
    const found = get().options.find((option) => option.name === normalized)
    if (found) {
      return found
    }
    // Fall back to the auto option (always first in the list).
    return get().options[0]
  },

  getDisplayName: (name: string | null | undefined) => {
    const normalized = normalizeAgentName(name)
    const option = get().options.find((item) => item.name === normalized)
    if (option) {
      return option.displayName
    }
    return getAgentDisplayName(normalized)
  },

  // -----------------------------------------------------------------------
  // Management / directory
  // -----------------------------------------------------------------------

  agents: [],
  mgmtLoading: false,
  mgmtLoaded: false,
  mgmtError: null,
  actionLoading: {},

  loadManagement: async () => {
    set({ mgmtLoading: true, mgmtError: null })
    try {
      const agents = await api.listAgentsManagement()
      set({ agents, mgmtLoading: false, mgmtLoaded: true, mgmtError: null })
    } catch (err) {
      const message = err instanceof Error ? err.message : 'Failed to load agents'
      set({ mgmtLoading: false, mgmtError: message })
    }
  },

  register: async (req: RegisterAgentRequest) => {
    setActionLoading(set, get, req.name, true)
    try {
      const agent = await api.registerAgent(req)
      set((s) => ({
        agents: [agent, ...s.agents.filter((a) => a.name !== agent.name)],
      }))
      setActionLoading(set, get, req.name, false)
      return agent
    } catch (err) {
      setActionLoading(set, get, req.name, false)
      throw err
    }
  },

  unregister: async (name: string) => {
    setActionLoading(set, get, name, true)
    try {
      await api.deleteAgent(name)
      set((s) => ({
        agents: s.agents.filter((a) => a.name !== name),
      }))
      setActionLoading(set, get, name, false)
    } catch (err) {
      setActionLoading(set, get, name, false)
      throw err
    }
  },

  toggleEnable: async (name: string, enabled: boolean) => {
    setActionLoading(set, get, name, true)
    try {
      const agent = await api.setAgentEnabled(name, enabled)
      set((s) => ({
        agents: s.agents.map((a) => (a.name === name ? agent : a)),
      }))
      setActionLoading(set, get, name, false)
      return agent
    } catch (err) {
      setActionLoading(set, get, name, false)
      throw err
    }
  },

  refresh: async (name: string) => {
    setActionLoading(set, get, name, true)
    try {
      const agent = await api.refreshAgent(name)
      set((s) => ({
        agents: s.agents.map((a) => (a.name === name ? agent : a)),
      }))
      setActionLoading(set, get, name, false)
      return agent
    } catch (err) {
      setActionLoading(set, get, name, false)
      throw err
    }
  },

  check: async (name: string) => {
    setActionLoading(set, get, name, true)
    try {
      const agent = await api.checkAgent(name)
      set((s) => ({
        agents: s.agents.map((a) => (a.name === name ? agent : a)),
      }))
      setActionLoading(set, get, name, false)
      return agent
    } catch (err) {
      setActionLoading(set, get, name, false)
      throw err
    }
  },

  isEnabled: (name: string | null | undefined) => {
    if (!name) return true
    const normalized = normalizeAgentName(name)
    if (normalized === 'auto') return true
    const agent = get().agents.find((a) => a.name === normalized)
    // If not in management list, default to enabled (static agents are always enabled unless toggled)
    if (!agent) return true
    return agent.enabled
  },

  clearMgmtError: () => {
    set({ mgmtError: null })
  },
}))

/**
 * Helper to set a per-agent action loading state.
 */
function setActionLoading(
  set: (updater: Partial<AgentState> | ((state: AgentState) => Partial<AgentState>)) => void,
  get: () => AgentState,
  name: string,
  loading: boolean,
) {
  set((s) => ({
    actionLoading: {
      ...s.actionLoading,
      [name]: loading,
    },
  }))
}
