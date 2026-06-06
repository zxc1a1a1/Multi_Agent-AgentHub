import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import WebPreview from './WebPreview'

describe('WebPreview', () => {
  it('renders HTML in a sandbox iframe with source view toggle', () => {
    render(
      <WebPreview
        block={{
          title: 'demo.html',
          html: '<section><h1>Demo</h1><script>alert(1)</script></section>',
        }}
      />,
    )

    // The header label is always visible.
    expect(screen.getByText('Web Preview')).toBeInTheDocument()

    // The iframe title comes from block.title.
    const iframe = screen.getByTitle('demo.html')
    expect(iframe).toBeInTheDocument()
    expect(iframe.tagName).toBe('IFRAME')
  })

  it('shows the Source tab button', () => {
    render(
      <WebPreview
        block={{
          title: 'demo.html',
          html: '<h1>Hello World</h1>',
        }}
      />,
    )

    // Source tab button is present.
    expect(screen.getByText('Source')).toBeInTheDocument()
    // Preview tab button is present and active by default.
    expect(screen.getByText('Preview')).toBeInTheDocument()
  })

  it('handles empty HTML gracefully', () => {
    render(
      <WebPreview
        block={{
          title: 'empty.html',
          html: '',
        }}
      />,
    )

    expect(screen.getByText('Web Preview')).toBeInTheDocument()
    // Shows empty state placeholder in preview tab.
  })

  it('does not include allow-scripts in iframe sandbox', () => {
    render(
      <WebPreview
        block={{
          title: 'test.html',
          html: '<h1>Safe Preview</h1>',
        }}
      />,
    )

    const iframe = screen.getByTitle('test.html') as HTMLIFrameElement
    expect(iframe).toBeInTheDocument()
    // sandbox must not include allow-scripts
    const sandboxAttr = iframe.getAttribute('sandbox')
    expect(sandboxAttr).not.toContain('allow-scripts')
    expect(sandboxAttr).toBe('')
  })

  it('injects CSP meta into srcDoc to block scripts', () => {
    render(
      <WebPreview
        block={{
          title: 'csp.html',
          html: '<html><head></head><body><p>Hello</p></body></html>',
        }}
      />,
    )

    const iframe = screen.getByTitle('csp.html') as HTMLIFrameElement
    expect(iframe).toBeInTheDocument()
    // srcDoc should contain the CSP meta tag
    const srcdocAttr = iframe.getAttribute('srcdoc')
    expect(srcdocAttr).toContain('Content-Security-Policy')
    expect(srcdocAttr).toContain("default-src 'none'")
    expect(srcdocAttr).toContain('<p>Hello</p>')
  })
})
