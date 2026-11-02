import { render, screen, waitFor } from '@testing-library/react'
import { describe, expect, test, vi, beforeEach } from 'vitest'
import mermaid from 'mermaid'
import MermaidDiagram from './MermaidDiagram'

vi.mock('mermaid', () => ({
  default: {
    initialize: vi.fn(),
    render: vi.fn(),
  },
}))

describe('MermaidDiagram', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  test('renders sanitized SVG and strips scripts/javascript links', async () => {
    vi.mocked(mermaid.render).mockResolvedValue({
      svg: '<svg xmlns="http://www.w3.org/2000/svg"><script>alert(1)</script><a href="javascript:alert(1)">x</a><rect width="10" height="10"/></svg>',
      bindFunctions: undefined,
    } as never)

    const { container } = render(<MermaidDiagram code="graph TD; A-->B" />)
    await waitFor(() => expect(container.querySelector('.mermaid-diagram svg')).not.toBeNull())
    expect(container.querySelector('script')).toBeNull()
    expect(container.querySelector('a[href^="javascript"]')).toBeNull()
    expect(container.querySelector('rect')).not.toBeNull()
    expect(vi.mocked(mermaid.initialize)).toHaveBeenCalledWith(
      expect.objectContaining({ securityLevel: 'strict' }),
    )
  })

  test('falls back to code text when mermaid fails', async () => {
    vi.mocked(mermaid.render).mockRejectedValue(new Error('parse error'))
    render(<MermaidDiagram code="graph TD; broken[" />)
    await waitFor(() => expect(screen.getByText('graph TD; broken[')).toBeInTheDocument())
  })
})
