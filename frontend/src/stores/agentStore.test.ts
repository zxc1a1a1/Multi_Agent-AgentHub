import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useAgentStore } from './agentStore'
import * as api from '../services/api'

vi.mock('../services/api', () => ({
  listAgents: vi.fn(),
}))

describe('agentStore', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    useAgentStore.setState({
      options: [],
      loading: false,
      loaded: false,
    })
  })

  it('loads dynamic agents from /api/agents', async () => {
    vi.mocked(api.listAgents).mockResolvedValue([
      {
        name: 'doc-agent',
        displayName: 'Doc Agent',
        description: 'Writes docs',
        outputModes: ['text', 'markdown'],
      },
    ])

    await useAgentStore.getState().load()

    const state = useAgentStore.getState()
    expect(state.loaded).toBe(true)
    expect(state.loading).toBe(false)
    expect(state.options).toHaveLength(1)
    expect(state.options[0]).toMatchObject({
      name: 'doc-agent',
      displayName: 'Doc Agent',
      description: 'Writes docs',
    })
    expect(state.defaultAgentName()).toBe('doc-agent')
  })

  it('falls back to local static options when /api/agents fails', async () => {
    vi.mocked(api.listAgents).mockRejectedValue(new Error('network error'))

    await useAgentStore.getState().load()

    const state = useAgentStore.getState()
    expect(state.loaded).toBe(true)
    expect(state.loading).toBe(false)
    expect(state.options.length).toBeGreaterThanOrEqual(2)
    expect(state.options.some((option) => option.name === 'code-agent')).toBe(true)
    expect(state.options.some((option) => option.name === 'web-agent')).toBe(true)
  })
})
