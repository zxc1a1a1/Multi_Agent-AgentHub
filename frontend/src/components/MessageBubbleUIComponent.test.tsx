import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import MessageBubble from './MessageBubble'
import { fetchArtifactContent } from '../services/api'
import type { Message, SkillCardData, UIComponentCardData, UnknownSkillData } from '../types'

vi.mock('../services/api', () => ({
  fetchArtifactContent: vi.fn(),
}))

const mockedFetch = vi.mocked(fetchArtifactContent)

const SAFE_HTML = '<!doctype html><html><head></head><body><h1>sandbox body</h1><script>parent.document.title="pwn"</script></body></html>'
const HOSTILE_PROPS = { heading: '<img src=x onerror=alert(1)>', url: 'https://internal.example/evil.js' }

function componentCard(overrides: Partial<UIComponentCardData> = {}): UIComponentCardData {
  return {
    type: 'ui_component',
    component: 'html_sandbox',
    props: HOSTILE_PROPS,
    schemaVersion: 1,
    artifactId: 'art_a1b2c3d4e5f6a1b2c3d4e5f6',
    capabilities: ['read_only'],
    runId: 'run-ui',
    eventId: 'evt-1',
    seq: 7,
    ...overrides,
  }
}

function fallbackCard(reason: string): UnknownSkillData {
  return {
    type: 'unknown_skill',
    toolName: 'ui_component',
    args: { reason, component: '<redacted>', artifactId: 'art_a1b2c3d4e5f6a1b2c3d4e5f6', schemaVersion: '1' },
  }
}

function buildMessage(skillCards: SkillCardData[]): Message {
  return {
    id: 'msg-ui-1',
    conversationId: 'conv-ui-1',
    senderType: 'agent',
    senderName: 'Code Agent',
    agentName: 'code-agent',
    content: 'see card below',
    status: 'sent',
    skillCards,
    createdAt: '2026-11-02T00:00:00Z',
  }
}

/** 文本降级卡片的根节点。 */
function fallbackRoot(): HTMLElement {
  const title = screen.getByText('Tool: ui_component')
  const card = title.closest('div')?.parentElement
  expect(card).toBeInstanceOf(HTMLElement)
  return card as HTMLElement
}

function assertNoExecutableFallback(container: HTMLElement) {
  expect(container.querySelector('iframe')).toBeNull()
  expect(container.querySelector('script')).toBeNull()
  expect(container.querySelector('img')).toBeNull()
  expect(container.innerHTML).not.toContain('onerror')
  expect(container.innerHTML).not.toContain('alert(1)')
  expect(container.innerHTML).not.toContain('internal.example')
}

beforeEach(() => {
  mockedFetch.mockReset()
  mockedFetch.mockResolvedValue({ size: SAFE_HTML.length, type: 'text/html', text: async () => SAFE_HTML } as unknown as Blob)
})

afterEach(() => {
  vi.restoreAllMocks()
})

describe('MessageBubble renders ui.component cards', () => {
  it('renders html_sandbox in a scripts-only iframe with a strict CSP', async () => {
    const evalSpy = vi.spyOn(window, 'eval').mockImplementation(() => undefined)
    const { container } = render(<MessageBubble message={buildMessage([componentCard()])} />)

    const frame = (await screen.findByTitle('html_sandbox artifact')) as HTMLIFrameElement
    expect(frame.tagName).toBe('IFRAME')
    expect(frame.getAttribute('sandbox')).toBe('allow-scripts')
    expect(frame.getAttribute('sandbox')).not.toContain('allow-same-origin')
    expect(frame.getAttribute('referrerpolicy')).toBe('no-referrer')
    const srcdoc = frame.getAttribute('srcdoc') ?? ''
    expect(srcdoc).toContain('<meta http-equiv="Content-Security-Policy"')
    expect(srcdoc).toContain("connect-src 'none'")
    expect(srcdoc).toContain("default-src 'none'")
    expect(srcdoc).toContain('<h1>sandbox body</h1>')
    // props 不进入 DOM/iframe 内容。
    expect(srcdoc).not.toContain(HOSTILE_PROPS.heading)
    expect(container.innerHTML).not.toContain('internal.example')
    expect(evalSpy).not.toHaveBeenCalled()
  })

  it('keeps non-html components as text fallbacks', () => {
    const { container } = render(
      <MessageBubble message={buildMessage([componentCard({ component: 'json_form' })])} />,
    )
    expect(screen.getByText('Tool: ui_component')).toBeInTheDocument()
    expect(screen.getByText(/renderer_not_implemented/)).toBeInTheDocument()
    assertNoExecutableFallback(container)
  })

  it('shows the rejection reason for an unknown component without echoing props', () => {
    const { container } = render(<MessageBubble message={buildMessage([fallbackCard('component_not_allowed')])} />)

    expect(screen.getByText('Tool: ui_component')).toBeInTheDocument()
    expect(screen.getByText(/component_not_allowed/)).toBeInTheDocument()
    expect(screen.getByText(/<redacted>/)).toBeInTheDocument()
    assertNoExecutableFallback(container)
  })

  it('stays inert through click and keyboard for fallback cards', async () => {
    const user = userEvent.setup()
    const evalSpy = vi.spyOn(window, 'eval').mockImplementation(() => undefined)
    const { container } = render(
      <MessageBubble message={buildMessage([componentCard({ component: 'json_form' })])} />,
    )

    const card = fallbackRoot()
    expect(card.querySelector('button, a, input, textarea, select, iframe, script')).toBeNull()

    await user.click(screen.getByText('Tool: ui_component'))
    await user.tab()
    expect(card.contains(document.activeElement as Node)).toBe(false)
    await user.keyboard('{Enter}')
    await user.keyboard('{Escape}')

    expect(screen.getAllByText('Tool: ui_component')).toHaveLength(1)
    assertNoExecutableFallback(container)
    expect(evalSpy).not.toHaveBeenCalled()
  })

  it('renders a single sandbox iframe for a replayed card', async () => {
    const { container } = render(<MessageBubble message={buildMessage([componentCard()])} />)
    await waitFor(() => expect(container.querySelector('iframe')).not.toBeNull())
    expect(container.querySelectorAll('iframe')).toHaveLength(1)
    expect(container.querySelectorAll('pre')).toHaveLength(0)
    expect(mockedFetch).toHaveBeenCalledTimes(1)
  })
})
