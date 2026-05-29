import { create } from 'zustand'
import type { Conversation } from '../types'
import * as api from '../services/api'
import type { AgentName } from '../lib/agents'

interface ConversationState {
  conversations: Conversation[]
  activeId: string | null
  loading: boolean

  load: () => Promise<void>
  create: (agentName: AgentName) => Promise<Conversation>
  setActive: (id: string) => void
}

export const useConversationStore = create<ConversationState>((set) => ({
  conversations: [],
  activeId: null,
  loading: false,

  load: async () => {
    set({ loading: true })
    try {
      const conversations = await api.listConversations()
      set({ conversations, loading: false })
    } catch {
      set({ loading: false })
    }
  },

  create: async (agentName: AgentName) => {
    const conv = await api.createConversation(agentName)
    set((s) => ({
      conversations: [conv, ...s.conversations],
      activeId: conv.id,
    }))
    return conv
  },

  setActive: (id: string) => {
    set({ activeId: id })
  },
}))
