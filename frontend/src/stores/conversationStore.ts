import { create } from 'zustand'
import type { Conversation } from '../types'
import * as api from '../services/api'
import type { AgentName } from '../lib/agents'
import { overlayLocalTitles, persistTitle } from '../lib/conversationTitles'

interface ConversationState {
  conversations: Conversation[]
  activeId: string | null
  loading: boolean

  load: () => Promise<void>
  create: (agentName: AgentName) => Promise<Conversation>
  setActive: (id: string) => void
  updateTitle: (id: string, title: string) => void
}

export const useConversationStore = create<ConversationState>((set) => ({
  conversations: [],
  activeId: null,
  loading: false,

  load: async () => {
    set({ loading: true })
    try {
      const rawConversations = await api.listConversations()
      const conversations = overlayLocalTitles(rawConversations)
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

  updateTitle: (id: string, title: string) => {
    set((s) => ({
      conversations: s.conversations.map((conv) =>
        conv.id === id ? { ...conv, title, updatedAt: new Date().toISOString() } : conv,
      ),
    }))
  },
}))
