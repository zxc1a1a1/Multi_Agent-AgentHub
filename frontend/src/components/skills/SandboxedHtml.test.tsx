import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import SandboxedHtml, {
  injectSandboxCSP,
  SANDBOX_HTML_CSP,
  SANDBOX_HTML_MAX_BYTES,
} from './SandboxedHtml'
import { fetchArtifactContent } from '../../services/api'

vi.mock('../../services/api', () => ({
  fetchArtifactContent: vi.fn(),
}))

const mockedFetch = vi.mocked(fetchArtifactContent)

// jsdom's Blob.text() is not available in this environment; the component only
// needs size/type/text, so a spec-shaped fake keeps the test on real logic.
function blobOf(html: string, type = 'text/html'): Blob {
  return { size: html.length, type, text: async () => html } as unknown as Blob
}

beforeEach(() => {
  mockedFetch.mockReset()
  mockedFetch.mockResolvedValue(blobOf('<html><head></head><body><p>ok</p></body></html>'))
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('injectSandboxCSP', () => {
  it('inserts the CSP before any existing head content', () => {
    const out = injectSandboxCSP('<html><head><script>1</script></head><body></body></html>')
    expect(out.indexOf('Content-Security-Policy')).toBeLessThan(out.indexOf('<script>'))
  })

  it('wraps fragments and documents without head', () => {
    expect(injectSandboxCSP('<p>x</p>')).toContain(`content="${SANDBOX_HTML_CSP}"`)
    expect(injectSandboxCSP('<html><body>x</body></html>')).toContain('<head><meta')
  })
})

describe('SandboxedHtml', () => {
  it('renders only allow-scripts and a no-referrer iframe', async () => {
    const { container } = render(<SandboxedHtml artifactId="art_test" title="probe" />)
    const frame = (await screen.findByTitle('probe')) as HTMLIFrameElement
    expect(frame.getAttribute('sandbox')).toBe('allow-scripts')
    expect(frame.getAttribute('sandbox')).not.toContain('allow-same-origin')
    expect(frame.getAttribute('referrerpolicy')).toBe('no-referrer')
    expect(frame.getAttribute('srcdoc')).toContain("connect-src 'none'")
    expect(container.querySelector('script')).toBeNull()
  })

  it('rejects an artifact above 256KB and does not create an iframe', async () => {
    mockedFetch.mockResolvedValue({
      size: SANDBOX_HTML_MAX_BYTES + 1,
      type: 'text/html',
      text: async () => '',
    } as unknown as Blob)
    const { container } = render(<SandboxedHtml artifactId="art_big" />)
    expect(await screen.findByRole('alert')).toHaveTextContent('artifact_too_large')
    expect(container.querySelector('iframe')).toBeNull()
  })

  it('rejects a non-HTML content type', async () => {
    mockedFetch.mockResolvedValue(blobOf('{"x":1}', 'application/json'))
    const { container } = render(<SandboxedHtml artifactId="art_json" />)
    expect(await screen.findByRole('alert')).toHaveTextContent('unsupported_content_type')
    expect(container.querySelector('iframe')).toBeNull()
  })

  it('shows a retryable error and recovers on retry', async () => {
    mockedFetch.mockRejectedValueOnce(new Error('network down'))
    const user = userEvent.setup()
    render(<SandboxedHtml artifactId="art_retry" />)
    expect(await screen.findByRole('alert')).toHaveTextContent('fetch_failed')
    await user.click(screen.getByRole('button', { name: 'Retry' }))
    expect(await screen.findByTitle(/art_retry/)).toBeInTheDocument()
  })

  it('accepts messages only from its own frame and ignores oversized payloads', async () => {
    const onMessage = vi.fn()
    render(<SandboxedHtml artifactId="art_msg" onMessage={onMessage} />)
    const frame = (await screen.findByTitle(/art_msg/)) as HTMLIFrameElement
    await waitFor(() => expect(frame.contentWindow).not.toBeNull())

    window.dispatchEvent(new MessageEvent('message', { data: { ok: true }, source: window }))
    expect(onMessage).not.toHaveBeenCalled()

    window.dispatchEvent(new MessageEvent('message', { data: { ok: true }, source: frame.contentWindow }))
    expect(onMessage).toHaveBeenCalledTimes(1)

    window.dispatchEvent(
      new MessageEvent('message', {
        data: { padding: 'x'.repeat(17 * 1024) },
        source: frame.contentWindow,
      }),
    )
    expect(onMessage).toHaveBeenCalledTimes(1)
  })
})
