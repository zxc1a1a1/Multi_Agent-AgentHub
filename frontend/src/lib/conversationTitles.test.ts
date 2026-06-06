import { describe, expect, it, beforeEach, vi } from 'vitest'
import { loadTitles, persistTitle, overlayLocalTitles } from '../lib/conversationTitles'

// Mock localStorage
const store: Record<string, string> = {}

beforeEach(() => {
  Object.keys(store).forEach((key) => delete store[key])
  vi.stubGlobal('localStorage', {
    getItem: vi.fn((key: string) => store[key] ?? null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value
    }),
    removeItem: vi.fn((key: string) => {
      delete store[key]
    }),
  })
})

describe('conversationTitles localStorage', () => {
  it('persists a title and loads it back', () => {
    persistTitle('conv-1', 'Build a login page')
    const titles = loadTitles()
    expect(titles['conv-1']).toBe('Build a login page')
  })

  it('overwrites existing title for same conversationId', () => {
    persistTitle('conv-1', 'First title')
    persistTitle('conv-1', 'Updated title')
    const titles = loadTitles()
    expect(titles['conv-1']).toBe('Updated title')
  })

  it('overlays local title when Gateway title is "New Conversation"', () => {
    persistTitle('conv-2', 'My custom title')
    const conversations = [
      { id: 'conv-2', title: 'New Conversation' },
      { id: 'conv-3', title: 'New Conversation' },
    ]
    const result = overlayLocalTitles(conversations)
    expect(result[0].title).toBe('My custom title')
    expect(result[1].title).toBe('New Conversation') // No local title for this one
  })

  it('respects Gateway title when it is a real title', () => {
    persistTitle('conv-4', 'Local override')
    const conversations = [
      { id: 'conv-4', title: 'Gateway authoritative title' },
    ]
    const result = overlayLocalTitles(conversations)
    // Gateway title is real (not default), so local title should NOT override
    expect(result[0].title).toBe('Gateway authoritative title')
  })

  it('returns empty object for corrupted JSON', () => {
    store['agenthub.conversationTitles.v1'] = '{corrupted'
    const titles = loadTitles()
    expect(titles).toEqual({})
  })

  it('returns empty object when localStorage is empty', () => {
    const titles = loadTitles()
    expect(titles).toEqual({})
  })

  it('skips non-string values in stored object', () => {
    store['agenthub.conversationTitles.v1'] = JSON.stringify({ 'conv-5': 123, 'conv-6': 'valid' })
    const titles = loadTitles()
    expect(titles['conv-5']).toBeUndefined()
    expect(titles['conv-6']).toBe('valid')
  })
})
