import { render, screen } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import StreamdownRenderer from './StreamdownRenderer'

vi.mock('streamdown/styles.css', () => ({}))

describe('StreamdownRenderer (real Streamdown 2.7.1)', () => {
  it('renders an unclosed code fence during streaming without crashing', async () => {
    const { container } = render(
      <StreamdownRenderer content={'```go\nfmt.Println("still open"'} fallback={<div>legacy</div>} />,
    )
    expect(await screen.findByText(/still open/)).toBeInTheDocument()
    expect(container.textContent).not.toContain('legacy')
  })

  it('renders CJK and inline math', async () => {
    render(<StreamdownRenderer content={'你好，世界。$x^2 + y^2$'} fallback={<div>legacy</div>} />)
    expect(await screen.findByText(/你好，世界/)).toBeInTheDocument()
  })

  it('keeps long content bounded without throwing', async () => {
    const long = 'AgentHub streaming text. '.repeat(1000)
    render(<StreamdownRenderer content={long} fallback={<div>legacy</div>} />)
    expect(await screen.findByText(/AgentHub streaming text\./)).toBeInTheDocument()
  })

  it('does not execute injected HTML/scripts or javascript: links', async () => {
    const hostile =
      '<script>window.__xss = 1</script><img src=x onerror="window.__xss=2">[click](javascript:alert(1))'
    const { container } = render(<StreamdownRenderer content={hostile} fallback={<div>legacy</div>} />)
    await screen.findByText(/click/)
    expect(container.querySelector('script')).toBeNull()
    expect(container.innerHTML).not.toContain('onerror')
    for (const link of Array.from(container.querySelectorAll('a'))) {
      expect(link.getAttribute('href') ?? '').not.toMatch(/^javascript:/i)
    }
    expect((window as unknown as { __xss?: number }).__xss).toBeUndefined()
  })
})
