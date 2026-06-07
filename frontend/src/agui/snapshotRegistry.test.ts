import { describe, it, expect, beforeEach } from 'vitest'
import {
  registerSnapshotRenderer,
  getSnapshotRenderer,
  hasSnapshotRenderer,
  clearSnapshotRegistry,
  type SnapshotRenderer,
  type SnapshotData,
  type RenderContext,
} from './snapshotRegistry'

const makeCtx = (): RenderContext => ({
  runId: 'run-1',
  location: 'inline',
})

const textRenderer: SnapshotRenderer = (data) =>
  `rendered:${data.snapshotType}:${JSON.stringify(data.payload)}`

describe('snapshotRegistry', () => {
  beforeEach(() => {
    clearSnapshotRegistry()
  })

  it('registers and retrieves a renderer', () => {
    registerSnapshotRenderer('code', textRenderer)
    const got = getSnapshotRenderer('code')
    expect(got).toBe(textRenderer)
  })

  it('hasSnapshotRenderer returns true for registered types', () => {
    registerSnapshotRenderer('webpage', textRenderer)
    expect(hasSnapshotRenderer('webpage')).toBe(true)
  })

  it('hasSnapshotRenderer returns false for unregistered types', () => {
    expect(hasSnapshotRenderer('unknown')).toBe(false)
  })

  it('returns undefined for unregistered type', () => {
    expect(getSnapshotRenderer('nonexistent')).toBeUndefined()
  })

  it('overwrites existing registration', () => {
    const rendererA: SnapshotRenderer = () => 'A'
    const rendererB: SnapshotRenderer = () => 'B'
    registerSnapshotRenderer('code', rendererA)
    registerSnapshotRenderer('code', rendererB)
    expect(getSnapshotRenderer('code')).toBe(rendererB)
  })

  it('clearSnapshotRegistry removes all renderers', () => {
    registerSnapshotRenderer('code', textRenderer)
    registerSnapshotRenderer('webpage', textRenderer)
    clearSnapshotRegistry()
    expect(hasSnapshotRenderer('code')).toBe(false)
    expect(hasSnapshotRenderer('webpage')).toBe(false)
    expect(getSnapshotRenderer('code')).toBeUndefined()
  })

  it('renderer receives snapshot data and context', () => {
    let capturedData: SnapshotData | undefined
    let capturedCtx: RenderContext | undefined
    const capturer: SnapshotRenderer = (data, ctx) => {
      capturedData = data
      capturedCtx = ctx
      return null
    }
    registerSnapshotRenderer('progress', capturer)
    const renderer = getSnapshotRenderer('progress')!
    const ctx = makeCtx()
    renderer(
      {
        runId: 'run-abc',
        taskId: 'task-1',
        snapshotType: 'progress',
        payload: { percent: 42 },
      },
      ctx,
    )
    expect(capturedData?.runId).toBe('run-abc')
    expect(capturedData?.payload).toEqual({ percent: 42 })
    expect(capturedCtx?.runId).toBe('run-1')
  })

  it('STATE_SNAPSHOT-like payload passes through unchanged', () => {
    registerSnapshotRenderer('activity', textRenderer)
    const renderer = getSnapshotRenderer('activity')!
    const result = renderer(
      {
        runId: 'run-x',
        snapshotType: 'activity',
        payload: { phase: 'planning', plannedAgents: ['code-agent'] },
      },
      makeCtx(),
    )
    expect(result).toContain('code-agent')
    expect(result).toContain('planning')
  })

  it('STATE_DELTA-like payload with delta field is handled', () => {
    let captured: SnapshotData | undefined
    const capturer: SnapshotRenderer = (data) => {
      captured = data
      return null
    }
    registerSnapshotRenderer('form', capturer)
    const renderer = getSnapshotRenderer('form')!
    const data: SnapshotData = {
      runId: 'run-y',
      snapshotType: 'form',
      payload: { field: 'original' },
      delta: { field: 'updated' },
    }
    renderer(data, makeCtx())
    // The renderer receives the full data including delta; merging is the consumer's job.
    expect(captured?.payload).toEqual({ field: 'original' })
    expect(captured?.delta).toEqual({ field: 'updated' })
  })

  it('unknown snapshot type is safely ignored by getSnapshotRenderer', () => {
    // The registry returns undefined for unknown types — consumer must not crash.
    const renderer = getSnapshotRenderer('unknown_type_xyz')
    expect(renderer).toBeUndefined()
    // Calling a missing renderer should be guarded by the consumer.
    if (renderer) {
      renderer({ runId: 'r', snapshotType: 'unknown_type_xyz', payload: {} }, makeCtx())
    }
    // No throw = pass.
  })
})
