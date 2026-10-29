import { beforeEach, describe, expect, it, vi } from 'vitest'
import { recoverRun, saveActiveRun, loadActiveRun, clearActiveRun } from './client'
import type { AGUIEvent } from '../types'

const mockFetch = vi.fn()
vi.stubGlobal('fetch', mockFetch)

function jsonResponse(status: number, body: unknown) {
  return { ok: status >= 200 && status < 300, status, json: () => Promise.resolve(body) }
}

beforeEach(() => {
  vi.clearAllMocks()
})

describe('recoverRun', () => {
  it('replays events after the cursor and stops at RUN_FINISHED', async () => {
    const events: AGUIEvent[] = [
      { type: 'TEXT_MESSAGE_CONTENT', runId: 'run-1', seq: 3, delta: 'hi' },
      { type: 'RUN_FINISHED', runId: 'run-1', seq: 4 },
    ]
    mockFetch.mockResolvedValueOnce(jsonResponse(200, { events }))
    const seen: AGUIEvent[] = []
    const onComplete = vi.fn()
    await recoverRun('run-1', 2, 0, (event) => seen.push(event), onComplete)
    expect(mockFetch).toHaveBeenCalledTimes(1)
    expect(mockFetch.mock.calls[0][0]).toBe('/api/runs/run-1/events?after=2&subIndex=0&limit=1000')
    expect(seen.map((e) => e.seq)).toEqual([3, 4])
    expect(onComplete).toHaveBeenCalledTimes(1)
  })

  it('replays remaining sub-index events at the cursor seq', async () => {
    const events: AGUIEvent[] = [
      { type: 'TEXT_MESSAGE_CONTENT', runId: 'run-4', seq: 2, subIndex: 0, delta: 'old' },
      { type: 'TEXT_MESSAGE_CONTENT', runId: 'run-4', seq: 2, subIndex: 1, delta: 'tail' },
      { type: 'RUN_FINISHED', runId: 'run-4', seq: 3, subIndex: 0 },
    ]
    mockFetch.mockResolvedValueOnce(jsonResponse(200, { events }))
    const seen: AGUIEvent[] = []
    await recoverRun('run-4', 2, 0, (event) => seen.push(event))
    expect(seen.map((e) => [e.seq, e.subIndex])).toEqual([
      [2, 1],
      [3, 0],
    ])
  })

  it('stops when the run status is terminal even without a terminal event', async () => {
    mockFetch
      .mockResolvedValueOnce(jsonResponse(200, { events: [] }))
      .mockResolvedValueOnce(jsonResponse(200, { status: 'completed' }))
    const onComplete = vi.fn()
    await recoverRun('run-2', 5, 0, () => {}, onComplete)
    expect(onComplete).toHaveBeenCalledTimes(1)
    expect(mockFetch).toHaveBeenCalledTimes(2)
  })

  it('fails when replay endpoint is unavailable', async () => {
    mockFetch.mockResolvedValueOnce(jsonResponse(503, {}))
    await expect(recoverRun('run-3', 0, 0, () => {})).rejects.toThrow('replay failed')
  })
})

describe('active run cursor', () => {
  beforeEach(() => {
    localStorage.clear()
  })

  it('round-trips and clears the cursor', () => {
    saveActiveRun('conv-1', 'run-1', 5, 2)
    expect(loadActiveRun('conv-1')).toMatchObject({
      runId: 'run-1',
      conversationId: 'conv-1',
      seq: 5,
      subIndex: 2,
    })
    clearActiveRun('conv-1')
    expect(loadActiveRun('conv-1')).toBeNull()
  })

  it('drops expired cursors', () => {
    saveActiveRun('conv-2', 'run-2', 1, 0)
    const key = 'agenthub.activeRun.conv-2'
    const cursor = JSON.parse(localStorage.getItem(key) || '{}')
    cursor.updatedAt = Date.now() - 31 * 60 * 1000
    localStorage.setItem(key, JSON.stringify(cursor))
    expect(loadActiveRun('conv-2')).toBeNull()
    expect(localStorage.getItem(key)).toBeNull()
  })
})
