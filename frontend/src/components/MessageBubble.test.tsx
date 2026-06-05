import { render, screen } from '@testing-library/react'
import { describe, expect, it } from 'vitest'
import MessageBubble from './MessageBubble'
import type { Message } from '../types'

function buildMessage(overrides: Partial<Message> = {}): Message {
  return {
    id: 'msg-1',
    conversationId: 'conv-1',
    senderType: 'agent',
    senderName: 'Code Agent',
    agentName: 'code-agent',
    content: 'Hello from code agent',
    status: 'sent',
    createdAt: '2026-01-01T00:00:00Z',
    ...overrides,
  }
}

describe('MessageBubble', () => {
  it('renders code-agent sender label', () => {
    render(<MessageBubble message={buildMessage({ senderName: 'Code Agent', agentName: 'code-agent' })} />)
    expect(screen.getByText('Code Agent')).toBeInTheDocument()
    expect(screen.getByText('Hello from code agent')).toBeInTheDocument()
  })

  it('renders web-agent sender label', () => {
    render(<MessageBubble message={buildMessage({ senderName: 'Web Agent', agentName: 'web-agent' })} />)
    expect(screen.getByText('Web Agent')).toBeInTheDocument()
  })

  it('renders orchestrator sender label as independent bubble', () => {
    render(
      <MessageBubble
        message={buildMessage({
          senderName: 'Orchestrator',
          agentName: 'orchestrator',
          content: 'All tasks completed successfully.',
        })}
      />,
    )
    expect(screen.getByText('Orchestrator')).toBeInTheDocument()
    expect(screen.getByText('All tasks completed successfully.')).toBeInTheDocument()
  })

  it('renders user messages without sender label (only avatar)', () => {
    render(
      <MessageBubble
        message={buildMessage({
          senderType: 'user',
          senderName: '',
          agentName: undefined,
          content: 'Write Go code',
        })}
      />,
    )
    // User messages show content but no sender name label above bubble
    expect(screen.getByText('Write Go code')).toBeInTheDocument()
    expect(screen.queryByText('You')).not.toBeInTheDocument()
  })

  it('shows failed indicator for failed messages', () => {
    render(
      <MessageBubble
        message={buildMessage({
          status: 'failed',
          content: 'Error',
        })}
      />,
    )
    expect(screen.getByText('Failed to generate response')).toBeInTheDocument()
  })

  it('renders code preview block when codeBlocks present', () => {
    render(
      <MessageBubble
        message={buildMessage({
          codeBlocks: [{ code: 'package main', language: 'go', filename: 'main.go' }],
        })}
      />,
    )
    expect(screen.getByText('main.go')).toBeInTheDocument()
    // highlight.js splits code into spans; verify code container exists
    expect(screen.getByText('Copy')).toBeInTheDocument()
    // The code block is rendered inside a code element
    const codeEls = screen.getAllByText((_, el) => el?.textContent === 'package main')
    expect(codeEls.length).toBeGreaterThanOrEqual(1)
  })

  it('renders web preview block when webPreviews present', () => {
    render(
      <MessageBubble
        message={buildMessage({
          webPreviews: [{ html: '<h1>Demo</h1>', title: 'demo.html' }],
        })}
      />,
    )
    expect(screen.getByText('Web Preview (Safe Mode)')).toBeInTheDocument()
    expect(screen.getByText('<h1>Demo</h1>')).toBeInTheDocument()
  })

  it('sender label falls back to agentName display when senderName is missing', () => {
    render(
      <MessageBubble
        message={buildMessage({
          senderName: undefined,
          agentName: 'web-agent',
        })}
      />,
    )
    // getAgentDisplayName('web-agent') returns 'Web Agent'
    expect(screen.getByText('Web Agent')).toBeInTheDocument()
  })

  it('does not show failed indicator for streaming messages', () => {
    render(
      <MessageBubble
        message={buildMessage({
          status: 'streaming',
        })}
      />,
    )
    expect(screen.queryByText('Failed to generate response')).not.toBeInTheDocument()
  })
})
