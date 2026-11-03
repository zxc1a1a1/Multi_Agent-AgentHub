import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import StreamdownRenderer from './StreamdownRenderer'

vi.mock('streamdown/styles.css', () => ({}))
vi.mock('streamdown', () => ({
  Streamdown: () => {
    throw new Error('streamdown chunk failed')
  },
}))

describe('StreamdownRenderer failure boundary', () => {
  it('falls back to the legacy renderer when Streamdown throws', async () => {
    render(<StreamdownRenderer content="boom" fallback={<div>legacy fallback</div>} />)
    expect(await screen.findByText('legacy fallback')).toBeInTheDocument()
  })
})
