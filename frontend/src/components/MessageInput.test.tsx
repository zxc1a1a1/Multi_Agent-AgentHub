import { render, screen, fireEvent, within } from '@testing-library/react'
import { describe, expect, it, vi, beforeAll } from 'vitest'
import MessageInput from './MessageInput'

beforeAll(() => {
  if (!HTMLElement.prototype.scrollIntoView) {
    HTMLElement.prototype.scrollIntoView = vi.fn() as unknown as typeof HTMLElement.prototype.scrollIntoView
  }
})

describe('MessageInput — @mention extraction', () => {
  it('extracts @agentName patterns from input', () => {
    const onSend = vi.fn()
    render(<MessageInput onSend={onSend} />)

    const textarea = screen.getByPlaceholderText(/Use @agent-name to mention/)
    fireEvent.change(textarea, { target: { value: 'Hello @code-agent please help' } })
    fireEvent.keyDown(textarea, { key: 'Enter' })

    expect(onSend).toHaveBeenCalledWith('Hello @code-agent please help', ['code-agent'])
  })

  it('extracts multiple @mentions', () => {
    const onSend = vi.fn()
    render(<MessageInput onSend={onSend} />)

    const textarea = screen.getByPlaceholderText(/Use @agent-name to mention/)
    fireEvent.change(textarea, { target: { value: '@code-agent and @web-agent collaborate' } })
    fireEvent.keyDown(textarea, { key: 'Enter' })

    expect(onSend).toHaveBeenCalledWith(
      '@code-agent and @web-agent collaborate',
      ['code-agent', 'web-agent'],
    )
  })

  it('deduplicates repeated @mentions', () => {
    const onSend = vi.fn()
    render(<MessageInput onSend={onSend} />)

    const textarea = screen.getByPlaceholderText(/Use @agent-name to mention/)
    fireEvent.change(textarea, { target: { value: '@code-agent @code-agent help' } })
    fireEvent.keyDown(textarea, { key: 'Enter' })

    expect(onSend).toHaveBeenCalledWith('@code-agent @code-agent help', ['code-agent'])
  })

  it('returns empty mentions when no @ patterns are present', () => {
    const onSend = vi.fn()
    render(<MessageInput onSend={onSend} />)

    const textarea = screen.getByPlaceholderText(/Use @agent-name to mention/)
    fireEvent.change(textarea, { target: { value: 'Just a normal message' } })
    fireEvent.keyDown(textarea, { key: 'Enter' })

    expect(onSend).toHaveBeenCalledWith('Just a normal message', [])
  })

  it('filters mentions to knownAgentNames when provided', () => {
    const onSend = vi.fn()
    render(
      <MessageInput
        onSend={onSend}
        knownAgentNames={['code-agent', 'web-agent']}
      />,
    )

    const textarea = screen.getByPlaceholderText(/Use @agent-name to mention/)
    fireEvent.change(textarea, { target: { value: '@code-agent @unknown-agent help' } })
    fireEvent.keyDown(textarea, { key: 'Enter' })

    // unknown-agent should be filtered out
    expect(onSend).toHaveBeenCalledWith(
      '@code-agent @unknown-agent help',
      ['code-agent'],
    )
  })

  it('shows mention indicator when @mentions are detected', () => {
    const onSend = vi.fn()
    render(
      <MessageInput
        onSend={onSend}
        knownAgentNames={['code-agent', 'web-agent']}
      />,
    )

    const textarea = screen.getByPlaceholderText(/Use @agent-name to mention/)
    fireEvent.change(textarea, { target: { value: 'Hello @code-agent' } })

    const indicator = screen.getByTestId('mentioning-indicator')
    expect(within(indicator).getByText('Mentioning:')).toBeInTheDocument()
    expect(within(indicator).getByText('@code-agent')).toBeInTheDocument()
  })

  it('does not submit empty message', () => {
    const onSend = vi.fn()
    render(<MessageInput onSend={onSend} />)

    const textarea = screen.getByPlaceholderText(/Use @agent-name to mention/)
    fireEvent.change(textarea, { target: { value: '   ' } })
    fireEvent.keyDown(textarea, { key: 'Enter' })

    expect(onSend).not.toHaveBeenCalled()
  })

  it('Shift+Enter does not submit', () => {
    const onSend = vi.fn()
    render(<MessageInput onSend={onSend} />)

    const textarea = screen.getByPlaceholderText(/Use @agent-name to mention/)
    fireEvent.change(textarea, { target: { value: '@code-agent help' } })
    fireEvent.keyDown(textarea, { key: 'Enter', shiftKey: true })

    expect(onSend).not.toHaveBeenCalled()
  })
})
