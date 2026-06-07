import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useConversationStore } from './conversationStore'
import * as api from '../services/api'

vi.mock('../services/api', () => ({
  listConversations: vi.fn(),
  createConversation: vi.fn(),
  deleteConversation: vi.fn(),
}))

describe('conversationStore.delete', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useConversationStore.setState({
      conversations: [],
      activeId: null,
      loading: false,
    })
  })

  it('deletes a conversation and removes it from the list', async () => {
    const conv1 = { id: 'conv-1', title: 'Chat 1', agentName: 'auto', createdAt: '2026-01-01', updatedAt: '2026-01-02' }
    const conv2 = { id: 'conv-2', title: 'Chat 2', agentName: 'code-agent', createdAt: '2026-01-03', updatedAt: '2026-01-04' }
    useConversationStore.setState({ conversations: [conv1, conv2], activeId: 'conv-2' })

    vi.mocked(api.deleteConversation).mockResolvedValue(undefined)

    await useConversationStore.getState().delete('conv-1')

    const state = useConversationStore.getState()
    expect(state.conversations).toHaveLength(1)
    expect(state.conversations[0].id).toBe('conv-2')
    expect(api.deleteConversation).toHaveBeenCalledWith('conv-1')
  })

  it('falls back to most recent conversation when deleting current', async () => {
    const conv1 = { id: 'conv-1', title: 'First', agentName: 'auto', createdAt: '2026-01-01', updatedAt: '2026-01-02' }
    const conv2 = { id: 'conv-2', title: 'Second', agentName: 'code-agent', createdAt: '2026-01-03', updatedAt: '2026-01-04' }
    useConversationStore.setState({ conversations: [conv1, conv2], activeId: 'conv-2' })

    vi.mocked(api.deleteConversation).mockResolvedValue(undefined)

    await useConversationStore.getState().delete('conv-2')

    const state = useConversationStore.getState()
    expect(state.conversations).toHaveLength(1)
    expect(state.conversations[0].id).toBe('conv-1')
    // activeId falls back to remaining conversation
    expect(state.activeId).toBe('conv-1')
  })

  it('sets activeId to null when last conversation is deleted', async () => {
    const conv1 = { id: 'conv-1', title: 'Only', agentName: 'auto', createdAt: '2026-01-01', updatedAt: '2026-01-02' }
    useConversationStore.setState({ conversations: [conv1], activeId: 'conv-1' })

    vi.mocked(api.deleteConversation).mockResolvedValue(undefined)

    await useConversationStore.getState().delete('conv-1')

    const state = useConversationStore.getState()
    expect(state.conversations).toHaveLength(0)
    expect(state.activeId).toBeNull()
  })

  it('preserves activeId when deleting a non-current conversation', async () => {
    const conv1 = { id: 'conv-1', title: 'Chat 1', agentName: 'auto', createdAt: '2026-01-01', updatedAt: '2026-01-02' }
    const conv2 = { id: 'conv-2', title: 'Chat 2', agentName: 'code-agent', createdAt: '2026-01-03', updatedAt: '2026-01-04' }
    useConversationStore.setState({ conversations: [conv1, conv2], activeId: 'conv-1' })

    vi.mocked(api.deleteConversation).mockResolvedValue(undefined)

    await useConversationStore.getState().delete('conv-2')

    const state = useConversationStore.getState()
    expect(state.activeId).toBe('conv-1')
  })

  it('calls the API with the correct conversation ID', async () => {
    const conv = { id: 'del-me', title: 'To Delete', agentName: 'auto', createdAt: '2026-01-01', updatedAt: '2026-01-02' }
    useConversationStore.setState({ conversations: [conv], activeId: 'del-me' })

    vi.mocked(api.deleteConversation).mockResolvedValue(undefined)

    await useConversationStore.getState().delete('del-me')

    expect(api.deleteConversation).toHaveBeenCalledTimes(1)
    expect(api.deleteConversation).toHaveBeenCalledWith('del-me')
  })
})
