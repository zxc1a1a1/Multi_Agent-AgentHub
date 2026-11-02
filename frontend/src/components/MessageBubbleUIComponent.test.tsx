import { render, screen } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { afterEach, describe, expect, it, vi } from 'vitest'
import MessageBubble from './MessageBubble'
import type { Message, SkillCardData, UIComponentCardData, UnknownSkillData } from '../types'

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

/** 卡片根节点：标题 span 的祖父 div（UnknownSkillFallback 的最外层容器）。 */
function uiCardRoot(): HTMLElement {
  const title = screen.getByText('Tool: ui_component')
  const card = title.closest('div')?.parentElement
  expect(card).toBeInstanceOf(HTMLElement)
  return card as HTMLElement
}

function assertInert(container: HTMLElement) {
  expect(container.querySelector('iframe')).toBeNull()
  expect(container.querySelector('script')).toBeNull()
  expect(container.querySelector('img')).toBeNull()
  expect(container.innerHTML).not.toContain('onerror')
  expect(container.innerHTML).not.toContain('alert(1)')
  expect(container.innerHTML).not.toContain('internal.example')
}

afterEach(() => {
  vi.restoreAllMocks()
})

describe('MessageBubble renders ui.component cards as inert text cards', () => {
  it('shows the text fallback for an allowlisted component with no renderer yet', () => {
    const evalSpy = vi.spyOn(window, 'eval').mockImplementation(() => undefined)
    const { container } = render(<MessageBubble message={buildMessage([componentCard()])} />)

    // R3-01 未注册渲染器：卡片必须走 UnknownSkillFallback 文本路径。
    expect(screen.getByText('Tool: ui_component')).toBeInTheDocument()
    expect(screen.getByText('raw output')).toBeInTheDocument()
    assertInert(container)
    expect(evalSpy).not.toHaveBeenCalled()
  })

  it('shows the rejection reason for an unknown component without echoing props', () => {
    const { container } = render(<MessageBubble message={buildMessage([fallbackCard('component_not_allowed')])} />)

    expect(screen.getByText('Tool: ui_component')).toBeInTheDocument()
    expect(screen.getByText(/component_not_allowed/)).toBeInTheDocument()
    expect(screen.getByText(/<redacted>/)).toBeInTheDocument()
    assertInert(container)
  })

  it('stays inert through click and keyboard interaction', async () => {
    const user = userEvent.setup()
    const evalSpy = vi.spyOn(window, 'eval').mockImplementation(() => undefined)
    const { container } = render(<MessageBubble message={buildMessage([componentCard()])} />)

    const card = uiCardRoot()
    // 卡片内部不含任何可执行或可交互节点。
    expect(card.querySelector('button, a, input, textarea, select, iframe, script')).toBeNull()

    await user.click(screen.getByText('Tool: ui_component'))
    await user.click(screen.getByText('raw output'))

    // 键盘遍历：焦点不会落到卡片内部，Enter/Escape 不触发任何行为。
    await user.tab()
    expect(card.contains(document.activeElement as Node)).toBe(false)
    await user.keyboard('{Enter}')
    await user.keyboard('{Escape}')

    expect(screen.getAllByText('Tool: ui_component')).toHaveLength(1)
    assertInert(container)
    expect(evalSpy).not.toHaveBeenCalled()
  })

  it('renders a single card so a replayed event cannot double the UI', () => {
    // reducer 已按 eventId/seq 去重，因此这里只会收到一张卡片。
    const { container } = render(<MessageBubble message={buildMessage([componentCard()])} />)

    expect(screen.getAllByText('Tool: ui_component')).toHaveLength(1)
    // 合法但未实现渲染器的卡片不展示 props 原文，因此没有 args 文本块。
    expect(container.querySelectorAll('pre')).toHaveLength(0)
    assertInert(container)
  })
})
