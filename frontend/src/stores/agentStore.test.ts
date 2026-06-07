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

  it('loads server agents and keeps Auto (Smart) first', async () => {
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
    expect(state.options.length).toBeGreaterThanOrEqual(2)
    expect(state.options[0]).toMatchObject({
      name: 'auto',
      displayName: 'Auto (Smart)',
    })
    expect(state.options[1]).toMatchObject({
      name: 'doc-agent',
      displayName: 'Doc Agent',
      description: 'Writes docs',
    })
  })

  it('injects Auto even when server omits auto', async () => {
    vi.mocked(api.listAgents).mockResolvedValue([
      { name: 'code-agent', displayName: 'Code Agent', description: 'Codes', outputModes: ['text', 'code'] },
      { name: 'web-agent', displayName: 'Web Agent', description: 'Builds web', outputModes: ['text', 'webpage'] },
    ])

    await useAgentStore.getState().load()

    const state = useAgentStore.getState()
    expect(state.options[0].name).toBe('auto')
    expect(state.options[0].displayName).toBe('Auto (Smart)')
    expect(state.options.some((o) => o.name === 'code-agent')).toBe(true)
    expect(state.options.some((o) => o.name === 'web-agent')).toBe(true)
  })

  it('deduplicates auto when server also returns auto', async () => {
    vi.mocked(api.listAgents).mockResolvedValue([
      { name: 'auto', displayName: 'Server Auto', description: 'Server version', outputModes: [] },
      { name: 'code-agent', displayName: 'Code Agent', description: 'Codes', outputModes: ['text'] },
    ])

    await useAgentStore.getState().load()

    const state = useAgentStore.getState()
    const autoOptions = state.options.filter((o) => o.name === 'auto')
    expect(autoOptions).toHaveLength(1)
    expect(state.options[0].name).toBe('auto')
    // Should use the local AUTO_AGENT_OPTION, not the server version.
    expect(state.options[0].displayName).toBe('Auto (Smart)')
  })

  it('default selected agent is auto after loading server agents', async () => {
    vi.mocked(api.listAgents).mockResolvedValue([
      { name: 'code-agent', displayName: 'Code Agent', description: 'Codes', outputModes: ['text'] },
    ])

    await useAgentStore.getState().load()

    const state = useAgentStore.getState()
    expect(state.defaultAgentName()).toBe('auto')
  })

  it('findOption falls back to auto for unknown agent name', async () => {
    vi.mocked(api.listAgents).mockResolvedValue([
      { name: 'code-agent', displayName: 'Code Agent', description: 'Codes', outputModes: ['text'] },
    ])

    await useAgentStore.getState().load()

    const state = useAgentStore.getState()
    const found = state.findOption('nonexistent-agent')
    expect(found).toBeDefined()
    expect(found?.name).toBe('auto')
  })

  it('findOption returns the correct real agent when it exists', async () => {
    vi.mocked(api.listAgents).mockResolvedValue([
      { name: 'code-agent', displayName: 'Code Agent', description: 'Codes', outputModes: ['text'] },
      { name: 'web-agent', displayName: 'Web Agent', description: 'Builds web', outputModes: ['webpage'] },
    ])

    await useAgentStore.getState().load()

    const state = useAgentStore.getState()
    const found = state.findOption('web-agent')
    expect(found).toBeDefined()
    expect(found?.name).toBe('web-agent')
    expect(found?.displayName).toBe('Web Agent')
  })

  it('falls back to local static options when /api/agents fails', async () => {
    vi.mocked(api.listAgents).mockRejectedValue(new Error('network error'))

    await useAgentStore.getState().load()

    const state = useAgentStore.getState()
    expect(state.loaded).toBe(true)
    expect(state.loading).toBe(false)
    expect(state.options.length).toBeGreaterThanOrEqual(2)
    expect(state.options[0].name).toBe('auto')
    expect(state.options.some((option) => option.name === 'code-agent')).toBe(true)
    expect(state.options.some((option) => option.name === 'web-agent')).toBe(true)
  })
})
