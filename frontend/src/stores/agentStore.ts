import { create } from 'zustand'
import * as api from '../services/api'
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
  options: AgentOption[]
  loading: boolean
  loaded: boolean
  load: () => Promise<void>
  defaultAgentName: () => AgentName
  findOption: (name: string | null | undefined) => AgentOption | undefined
  getDisplayName: (name: string | null | undefined) => string
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
}))
