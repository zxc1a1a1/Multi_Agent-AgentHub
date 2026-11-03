import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'

vi.mock('streamdown/styles.css', () => ({}))
vi.mock('streamdown', () => ({
  Streamdown: ({ children }: { children: string }) => <div data-testid="streamdown">{children}</div>,
}))

describe('StreamingText Streamdown flag', () => {
  it('flag off keeps the legacy renderer and never renders Streamdown', async () => {
    vi.stubEnv('VITE_STREAMDOWN_ENABLED', 'false')
    vi.resetModules()
    const { default: StreamingText } = await import('./StreamingText')
    render(<StreamingText content={'legacy **bold** text'} />)
    expect(screen.getByText(/legacy/)).toBeInTheDocument()
    expect(screen.queryByTestId('streamdown')).toBeNull()
  })

  it('flag on lazily renders Streamdown', async () => {
    vi.stubEnv('VITE_STREAMDOWN_ENABLED', 'true')
    vi.resetModules()
    const { default: StreamingText } = await import('./StreamingText')
    render(<StreamingText content={'streamdown body'} />)
    expect(await screen.findByTestId('streamdown')).toHaveTextContent('streamdown body')
  })
})
