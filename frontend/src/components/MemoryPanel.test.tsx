import { beforeEach, describe, expect, it, vi } from 'vitest'
import { fireEvent, render, screen, waitFor } from '@testing-library/react'

const api = vi.hoisted(() => ({
  getMemoryPolicy: vi.fn(),
  listMemoryEntries: vi.fn(),
  updateMemoryPolicy: vi.fn(),
  confirmMemory: vi.fn(),
  updateMemory: vi.fn(),
  forgetMemory: vi.fn(),
  forgetMemoryScope: vi.fn(),
  memoryScopeSnapshot: vi.fn(() => ({ env: 'test', workspaceId: 'default', principalId: 'demo-user' })),
}))

vi.mock('../services/api', () => api)

import MemoryPanel from './MemoryPanel'
import type { MemoryEntry, MemoryPolicy } from '../services/api'

const policy: MemoryPolicy = {
  mode: 'scoped_long_term',
  read: true,
  propose: true,
  confirm: true,
  forget: true,
  maxInjectedEntries: 5,
  maxCoreEntries: 2,
  maxInjectedTokens: 2048,
  maxEntryBytes: 4096,
  maxActivePerScope: 200,
  candidateTTL: 86400000000000,
  defaultRecheckAfter: 2592000000000000,
}

const candidate: MemoryEntry = {
  memoryId: 'm-candidate',
  kind: 'preference',
  logicalKey: 'favorite-lang',
  content: 'user likes Go',
  revision: 1,
  status: 'candidate',
  sourceRefs: ['msg-1'],
  createdAt: '2026-10-29T00:00:00Z',
  verifiedAt: '2026-10-29T00:00:00Z',
}

const active: MemoryEntry = {
  ...candidate,
  memoryId: 'm-active',
  logicalKey: 'timezone',
  content: 'UTC+8',
  status: 'active',
  revision: 2,
}

describe('MemoryPanel', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    api.getMemoryPolicy.mockResolvedValue(policy)
    api.listMemoryEntries.mockResolvedValue([candidate, active])
    api.confirmMemory.mockResolvedValue({ ...candidate, status: 'active' })
    api.updateMemory.mockResolvedValue({ ...active, revision: 3 })
    api.forgetMemory.mockResolvedValue({ recallDisabled: true, purgeStatus: 'pending' })
    api.forgetMemoryScope.mockResolvedValue({ revokedCount: 2, recallDisabled: true, purgeStatus: 'pending' })
    api.updateMemoryPolicy.mockResolvedValue(policy)
    vi.stubGlobal('confirm', vi.fn(() => true))
    vi.stubGlobal('prompt', vi.fn(() => 'UTC+9'))
  })

  it('loads scope, mode and entries', async () => {
    render(<MemoryPanel />)
    expect(await screen.findByText('favorite-lang')).toBeInTheDocument()
    expect(screen.getByText('timezone')).toBeInTheDocument()
    expect(screen.getByText(/test \/ default \/ demo-user/)).toBeInTheDocument()
    expect((screen.getByLabelText('Memory mode') as HTMLSelectElement).value).toBe('scoped_long_term')
    expect(screen.getAllByText(/sources: msg-1/)).toHaveLength(2)
  })

  it('confirms a candidate with its revision', async () => {
    render(<MemoryPanel />)
    await screen.findByText('favorite-lang')
    fireEvent.click(screen.getByTitle('Confirm memory'))
    await waitFor(() => expect(api.confirmMemory).toHaveBeenCalledWith('m-candidate', 1))
    expect(await screen.findByText('Memory confirmed and active.')).toBeInTheDocument()
  })

  it('corrects an active memory and surfaces revision conflicts', async () => {
    render(<MemoryPanel />)
    await screen.findByText('timezone')
    fireEvent.click(screen.getByTitle('Correct memory'))
    await waitFor(() => expect(api.updateMemory).toHaveBeenCalledWith('m-active', 'UTC+9', 2))
    expect(await screen.findByText('Memory corrected with a new revision.')).toBeInTheDocument()

    api.updateMemory.mockRejectedValueOnce(new Error('memory operation failed'))
    fireEvent.click(screen.getByTitle('Correct memory'))
    expect(await screen.findByText('memory operation failed')).toBeInTheDocument()
  })

  it('forgets memory and reports purge status without claiming source deletion', async () => {
    render(<MemoryPanel />)
    await screen.findByText('timezone')
    fireEvent.click(screen.getAllByTitle('Forget memory')[1])
    await waitFor(() => expect(api.forgetMemory).toHaveBeenCalledWith('m-active'))
    expect(await screen.findByText(/purge status: pending/)).toBeInTheDocument()
    expect(screen.getByText(/does not delete the original conversation/)).toBeInTheDocument()
  })

  it('forgets the whole scope and reports revoked count/purge status', async () => {
    render(<MemoryPanel />)
    await screen.findByText('favorite-lang')
    fireEvent.click(screen.getByText('Forget all in this scope'))
    await waitFor(() => expect(api.forgetMemoryScope).toHaveBeenCalled())
    expect(await screen.findByText(/Revoked 2 memories/)).toBeInTheDocument()
  })

  it('shows load errors instead of an empty success state', async () => {
    api.listMemoryEntries.mockRejectedValueOnce(new Error('memory unavailable'))
    render(<MemoryPanel />)
    expect(await screen.findByText('memory unavailable')).toBeInTheDocument()
  })

  it('changes mode and persists the full policy', async () => {
    render(<MemoryPanel />)
    await screen.findByText('favorite-lang')
    fireEvent.change(screen.getByLabelText('Memory mode'), { target: { value: 'off' } })
    await waitFor(() => expect(api.updateMemoryPolicy).toHaveBeenCalled())
    const saved = api.updateMemoryPolicy.mock.calls[0][0] as MemoryPolicy
    expect(saved.mode).toBe('off')
    expect(saved.read).toBe(false)
  })
})
