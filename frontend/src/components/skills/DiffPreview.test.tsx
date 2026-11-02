import { render, screen, fireEvent, waitFor } from '@testing-library/react'
import { describe, expect, test, vi, beforeEach } from 'vitest'
import DiffPreview from './DiffPreview'
import * as api from '../../services/api'

vi.mock('../../services/api', () => ({
  dryRunDiff: vi.fn(),
  applyDiff: vi.fn(),
}))

const diffText = [
  'diff --git a/file.go b/file.go',
  'index 1111111..2222222 100644',
  '--- a/file.go',
  '+++ b/file.go',
  '@@ -1,3 +1,3 @@',
  ' package main',
  '-old line',
  '+new line',
  ' // end',
  '',
].join('\n')

describe('DiffPreview', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  test('parses diff2html and renders file/hunk/line text', () => {
    render(<DiffPreview diffText={diffText} filename="file.go" />)
    expect(screen.getAllByText(/file\.go/).length).toBeGreaterThan(0)
    expect(screen.getByText(/@@ -1,3 \+1,3 @@/)).toBeInTheDocument()
    expect(screen.getByText('-old line')).toBeInTheDocument()
    expect(screen.getByText('+new line')).toBeInTheDocument()
    expect(screen.getByText((_, el) => el?.textContent === ' package main')).toBeInTheDocument()
    expect(screen.getByText('+1')).toBeInTheDocument()
    expect(screen.getByText('-1')).toBeInTheDocument()
  })

  test('keeps dry-run then apply handshake', async () => {
    vi.mocked(api.dryRunDiff).mockResolvedValue({ dryRunId: 'dry-1', status: 'valid', files: ['file.go'] })
    vi.mocked(api.applyDiff).mockResolvedValue({ status: 'applied', files: ['file.go'] })
    vi.spyOn(window, 'confirm').mockReturnValue(true)

    render(<DiffPreview diffText={diffText} filename="file.go" />)
    const applyButton = screen.getByRole('button', { name: 'Apply' })
    expect(applyButton).toBeDisabled()

    fireEvent.click(screen.getByRole('button', { name: 'Dry-run' }))
    await waitFor(() => expect(api.dryRunDiff).toHaveBeenCalledWith({ path: 'file.go', diffText }))
    await waitFor(() => expect(applyButton).toBeEnabled())

    fireEvent.click(applyButton)
    await waitFor(() =>
      expect(api.applyDiff).toHaveBeenCalledWith({
        path: 'file.go',
        diffText,
        dryRunId: 'dry-1',
        confirmed: true,
      }),
    )
  })

  test('renders diff content as text, not HTML', () => {
    const malicious = [
      '--- a/evil.html',
      '+++ b/evil.html',
      '@@ -1 +1 @@',
      '-old',
      '+<img src=x onerror="window.__xss=1"><script>alert(1)</script>',
      '',
    ].join('\n')
    const { container } = render(<DiffPreview diffText={malicious} filename="evil.html" />)
    expect(container.querySelector('img')).toBeNull()
    expect(container.querySelector('script')).toBeNull()
    expect(container.textContent).toContain('onerror')
  })
})
