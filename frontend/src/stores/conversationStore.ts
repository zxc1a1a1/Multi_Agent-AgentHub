import { create } from 'zustand'
import type { Conversation } from '../types'
import * as api from '../services/api'
import type { AgentName } from '../lib/agents'
import { overlayLocalTitles, persistTitle } from '../lib/conversationTitles'

interface ConversationState {
  conversations: Conversation[]
  activeId: string | null
  loading: boolean
  searchQuery: string

  load: () => Promise<void>
  create: (agentName: AgentName) => Promise<Conversation>
  setActive: (id: string) => void
  updateTitle: (id: string, title: string) => void
  delete: (id: string) => Promise<void>
  pinConversation: (id: string, pinned: boolean) => Promise<void>
  setSearchQuery: (query: string) => void
}

function sortConversations(conversations: Conversation[]): Conversation[] {
  return [...conversations].sort((a, b) => {
    // Pinned first
    if (a.pinned && !b.pinned) return -1
    if (!a.pinned && b.pinned) return 1
    // Then by updatedAt DESC
    return new Date(b.updatedAt).getTime() - new Date(a.updatedAt).getTime()
  })
}

const ACTIVE_CONVERSATION_KEY = 'agenthub.activeConversation'

function readStoredActiveId(): string | null {
  try {
    return typeof window === 'undefined' ? null : window.localStorage.getItem(ACTIVE_CONVERSATION_KEY)
  } catch {
    return null
  }
}

function writeStoredActiveId(id: string | null): void {
  try {
    if (typeof window === 'undefined') return
    if (id) window.localStorage.setItem(ACTIVE_CONVERSATION_KEY, id)
    else window.localStorage.removeItem(ACTIVE_CONVERSATION_KEY)
  } catch {
    // Storage may be unavailable; selection still works in memory.
  }
}

export const useConversationStore = create<ConversationState>((set, get) => ({
  conversations: [],
  activeId: readStoredActiveId(),
  loading: false,
  searchQuery: '',

  load: async () => {
    set({ loading: true })
    try {
      const rawConversations = await api.listConversations()
      const conversations = sortConversations(overlayLocalTitles(rawConversations))
      set({ conversations, loading: false })
      const activeId = get().activeId
      if (activeId && !conversations.some((conv) => conv.id === activeId)) {
        writeStoredActiveId(null)
        set({ activeId: null })
      }
    } catch {
      set({ loading: false })
    }
  },

  create: async (agentName: AgentName) => {
    const conv = await api.createConversation(agentName)
    writeStoredActiveId(conv.id)
    set((s) => ({
      conversations: sortConversations([conv, ...s.conversations]),
      activeId: conv.id,
    }))
    return conv
  },

  setActive: (id: string) => {
    writeStoredActiveId(id)
    set({ activeId: id })
  },

  updateTitle: (id: string, title: string) => {
    set((s) => ({
      conversations: sortConversations(
        s.conversations.map((conv) =>
          conv.id === id ? { ...conv, title, updatedAt: new Date().toISOString() } : conv,
        ),
      ),
    }))
  },

  delete: async (id: string) => {
    await api.deleteConversation(id)
    set((s) => {
      const remaining = s.conversations.filter((conv) => conv.id !== id)
      let nextActiveId = s.activeId
      if (s.activeId === id) {
        nextActiveId = remaining.length > 0 ? remaining[0].id : null
        writeStoredActiveId(nextActiveId)
      }
      return {
        conversations: remaining,
        activeId: nextActiveId,
      }
    })
  },

  pinConversation: async (id: string, pinned: boolean) => {
    const updated = await api.pinConversation(id, pinned)
    set((s) => ({
      conversations: sortConversations(
        s.conversations.map((conv) =>
          conv.id === id
            ? { ...updated, updatedAt: new Date().toISOString() }
            : conv,
        ),
      ),
    }))
  },

  setSearchQuery: (query: string) => {
    set({ searchQuery: query })
  },
}))
