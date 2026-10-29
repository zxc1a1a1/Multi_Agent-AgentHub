import { beforeEach, describe, expect, it, vi } from 'vitest'
import { recoverRun } from './client'
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
    await recoverRun('run-1', 2, (event) => seen.push(event), onComplete)
    expect(mockFetch).toHaveBeenCalledTimes(1)
    expect(mockFetch.mock.calls[0][0]).toBe('/api/runs/run-1/events?after=2&limit=1000')
    expect(seen.map((e) => e.seq)).toEqual([3, 4])
    expect(onComplete).toHaveBeenCalledTimes(1)
  })

  it('stops when the run status is terminal even without a terminal event', async () => {
    mockFetch
      .mockResolvedValueOnce(jsonResponse(200, { events: [] }))
      .mockResolvedValueOnce(jsonResponse(200, { status: 'completed' }))
    const onComplete = vi.fn()
    await recoverRun('run-2', 5, () => {}, onComplete)
    expect(onComplete).toHaveBeenCalledTimes(1)
    expect(mockFetch).toHaveBeenCalledTimes(2)
  })

  it('fails when replay endpoint is unavailable', async () => {
    mockFetch.mockResolvedValueOnce(jsonResponse(503, {}))
    await expect(recoverRun('run-3', 0, () => {})).rejects.toThrow('replay failed')
  })
})
