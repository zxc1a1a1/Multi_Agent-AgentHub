import { create } from 'zustand'
import type { ActivitySnapshot } from '../types'

interface ActivityState {
  // Current pending activity per conversation.
  pendingActivityByConversation: Record<string, ActivitySnapshot>

  // Set or update activity for a conversation.
  setActivity: (conversationId: string, activity: ActivitySnapshot) => void

  // Update only the status field.
  updateActivityStatus: (conversationId: string, status: string) => void

  // Remove activity for a conversation.
  clearActivity: (conversationId: string) => void
}

export const useActivityStore = create<ActivityState>((set) => ({
  pendingActivityByConversation: {},

  setActivity: (conversationId, activity) =>
    set((state) => ({
      pendingActivityByConversation: {
        ...state.pendingActivityByConversation,
        [conversationId]: activity,
      },
    })),

  updateActivityStatus: (conversationId, status) =>
    set((state) => {
      const current = state.pendingActivityByConversation[conversationId]
      if (!current) return state
      return {
        pendingActivityByConversation: {
          ...state.pendingActivityByConversation,
          [conversationId]: { ...current, status },
        },
      }
    }),

  clearActivity: (conversationId) =>
    set((state) => {
      const next = { ...state.pendingActivityByConversation }
      delete next[conversationId]
      return { pendingActivityByConversation: next }
    }),
}))
