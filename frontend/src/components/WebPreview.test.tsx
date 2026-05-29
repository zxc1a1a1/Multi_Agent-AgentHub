import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import WebPreview from './WebPreview'

describe('WebPreview', () => {
  it('renders html as plain text without iframe execution path', () => {
    render(
      <WebPreview
        block={{
          title: 'demo.html',
          html: '<section><h1>Demo</h1><script>alert(1)</script></section>',
        }}
      />,
    )

    expect(screen.getByText('Web Preview (Safe Mode)')).toBeInTheDocument()
    expect(screen.getByText(/<script>alert\(1\)<\/script>/)).toBeInTheDocument()
    expect(screen.queryByRole('iframe')).not.toBeInTheDocument()
  })
})
