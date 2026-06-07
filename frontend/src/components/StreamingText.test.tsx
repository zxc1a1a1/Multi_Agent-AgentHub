import { render, screen } from '@testing-library/react'
import { describe, expect, test } from 'vitest'
import StreamingText from './StreamingText'

function getCodeContainers() {
  return document.querySelectorAll('.rounded-lg.border.border-gray-700')
}

function getCodeText() {
  const containers = getCodeContainers()
  if (containers.length === 0) return ''
  // Get text from all code blocks concatenated
  return Array.from(containers)
    .map((c) => c.querySelector('code')?.textContent || '')
    .join('\n')
}

describe('StreamingText progressive code blocks', () => {
  test('renders plain text during streaming', () => {
    const { container } = render(
      <StreamingText content="Hello world" isStreaming={true} />,
    )
    // Should show the text content
    expect(container.textContent).toContain('Hello world')
    // Should have the pulsing cursor
    expect(container.querySelector('.animate-pulse')).toBeTruthy()
  })

  test('renders open fence as styled code block during streaming', () => {
    const content = 'Here is code:\n```ts\nconst x = 1\nconst y = 2\n'
    render(<StreamingText content={content} isStreaming={true} />)

    // Should have a code block container (the open fence was detected)
    const codeBlocks = getCodeContainers()
    expect(codeBlocks.length).toBeGreaterThanOrEqual(1)

    // The code block should contain the code content
    const codeText = getCodeText()
    expect(codeText).toContain('const x = 1')
    expect(codeText).toContain('const y = 2')

    // The text before the code fence should still be visible
    const container = document.querySelector('.message-content')
    expect(container?.textContent).toContain('Here is code')
  })

  test('renders closed fence as complete code block after streaming', () => {
    const content = 'Result:\n```python\nprint("hello")\nprint("world")\n```\nDone.'
    render(<StreamingText content={content} isStreaming={false} />)

    // After streaming (Markdown mode), code should be in a pre element
    const preElement = document.querySelector('pre')
    expect(preElement).toBeTruthy()
    expect(preElement!.textContent).toContain('print("hello")')
    expect(preElement!.textContent).toContain('print("world")')

    // Text before and after should be present
    const prose = document.querySelector('.prose')
    expect(prose?.textContent).toContain('Result:')
    expect(prose?.textContent).toContain('Done.')
  })

  test('mixed text and code: all content preserved during streaming', () => {
    const content = 'Intro text\n```js\nconst a = 1\n```\nMiddle text\n```css\nbody { color: red; }\n```\nOutro text'
    render(<StreamingText content={content} isStreaming={true} />)

    const container = document.querySelector('.message-content')
    const fullText = container?.textContent || ''

    // All text parts should be present
    expect(fullText).toContain('Intro text')
    expect(fullText).toContain('Middle text')
    expect(fullText).toContain('Outro text')

    // All code parts should be present
    const codeBlocks = getCodeContainers()
    expect(codeBlocks.length).toBeGreaterThanOrEqual(1)

    const allCodeText = Array.from(codeBlocks)
      .map((c) => c.querySelector('code')?.textContent || '')
      .join('\n')
    expect(allCodeText).toContain('const a = 1')
    expect(allCodeText).toContain('body { color: red; }')
  })

  test('incremental chunks: content accumulates without reset', () => {
    // Simulate streaming: first half of a code block arrives
    const partialContent = '```html\n<div>\n  <p>Hello'
    const { rerender, container } = render(
      <StreamingText content={partialContent} isStreaming={true} />,
    )

    let codeBlocks = getCodeContainers()
    expect(codeBlocks.length).toBeGreaterThanOrEqual(1)
    expect(container.textContent).toContain('<div>')
    expect(container.textContent).toContain('<p>Hello')

    // Second half arrives — the code block should update, not reset
    const fullContent = '```html\n<div>\n  <p>Hello</p>\n  <p>World</p>\n</div>\n```'
    rerender(<StreamingText content={fullContent} isStreaming={true} />)

    codeBlocks = getCodeContainers()
    // After closing fence in streaming mode, the block should still be present
    const allCode = Array.from(codeBlocks)
      .map((c) => c.querySelector('code')?.textContent || '')
      .join('\n')
    expect(allCode).toContain('<p>Hello</p>')
    expect(allCode).toContain('<p>World</p>')
    expect(allCode).toContain('</div>')
  })

  test('open fence without language tag renders correctly', () => {
    const content = '```\nplain text code block\nstill going\n'
    render(<StreamingText content={content} isStreaming={true} />)

    const codeBlocks = getCodeContainers()
    expect(codeBlocks.length).toBeGreaterThanOrEqual(1)

    const codeText = getCodeText()
    expect(codeText).toContain('plain text code block')
    expect(codeText).toContain('still going')
  })

  test('empty content during streaming shows thinking indicator', () => {
    render(<StreamingText content="" isStreaming={true} />)
    expect(screen.getByText(/Thinking/i)).toBeInTheDocument()
  })

  test('long code block content preserved without truncation', () => {
    const longLine = 'x'.repeat(200)
    const content = '```\n' + longLine + '\n```'
    render(<StreamingText content={content} isStreaming={false} />)

    const preElement = document.querySelector('pre')
    expect(preElement?.textContent).toContain(longLine)
  })

  test('copy button present on code block header', () => {
    const content = '```ts\nconst x = 1\n```'
    render(<StreamingText content={content} isStreaming={false} />)

    // After streaming, the Copy button should be visible
    const copyButton = screen.queryByText('Copy')
    expect(copyButton).toBeTruthy()
  })
})
