import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AgentPicker from './AgentPicker'
import { useAgentStore } from '../stores/agentStore'

vi.mock('../services/api', () => ({
  listAgents: vi.fn(),
  listAgentsManagement: vi.fn().mockResolvedValue([
    {
      name: 'code-agent',
      displayName: 'Code Agent',
      description: 'Codes',
      source: 'static',
      enabled: true,
      healthy: true,
    },
    {
      name: 'disabled-agent',
      displayName: 'Disabled Agent',
      description: 'Off',
      source: 'dynamic',
      enabled: false,
      healthy: false,
    },
  ]),
}))

describe('AgentPicker', () => {
  beforeEach(() => {
    useAgentStore.setState({
      options: [
        { name: 'auto', displayName: 'Auto (Smart)', description: 'Auto', outputModes: ['text'] },
        { name: 'code-agent', displayName: 'Code Agent', description: 'Codes', outputModes: ['text'] },
        { name: 'disabled-agent', displayName: 'Disabled Agent', description: 'Off', outputModes: ['text'] },
      ],
      agents: [],
      loading: false,
      loaded: true,
      mgmtLoading: false,
      mgmtLoaded: false,
    })
  })

  it('does not select an agent that management marks disabled', async () => {
    const onChange = vi.fn()
    render(<AgentPicker selectedAgentNames={['auto']} onChange={onChange} />)

    fireEvent.click(screen.getByText('Auto (Smart)'))
    await waitFor(() => {
      expect(useAgentStore.getState().agents.length).toBe(2)
    })
    const disabled = screen.getByText('Disabled Agent').closest('button')
    expect(disabled).toBeDisabled()
    fireEvent.click(screen.getByText('Disabled Agent'))
    expect(onChange).not.toHaveBeenCalled()
  })
})
