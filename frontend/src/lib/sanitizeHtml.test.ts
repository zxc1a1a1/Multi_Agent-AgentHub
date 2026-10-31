import { describe, expect, it } from 'vitest'
import { sanitizeHighlightHtml } from './sanitizeHtml'

describe('sanitizeHighlightHtml', () => {
  it('keeps hljs spans and strips every other element and attribute', () => {
    const dirty = '<span class="hljs-keyword" onclick="alert(1)">const</span><script>alert(1)</script><img src=x onerror="alert(2)">'
    const clean = sanitizeHighlightHtml(dirty)
    expect(clean).toContain('hljs-keyword')
    expect(clean).toContain('const')
    expect(clean).not.toContain('<script')
    expect(clean).not.toContain('<img')
    expect(clean).not.toContain('onclick')
    expect(clean).not.toContain('onerror')
  })

  it('escapes disallowed element content as text instead of markup', () => {
    const clean = sanitizeHighlightHtml('<iframe srcdoc="<script>x</script>">fallback</iframe>')
    expect(clean).not.toContain('<iframe')
    expect(clean).toContain('fallback')
  })

  it('returns an empty string for empty input', () => {
    expect(sanitizeHighlightHtml('')).toBe('')
  })
})
