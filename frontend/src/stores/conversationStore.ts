import { create } from 'zustand'
import type { Conversation } from '../types'
import * as api from '../services/api'

interface ConversationState {
  conversations: Conversation[]
  activeId: string | null
  loading: boolean

  load: () => Promise<void>
  create: (agentName: string) => Promise<Conversation>
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

  create: async (agentName: string) => {
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
