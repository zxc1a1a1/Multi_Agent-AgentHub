import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useMessageStore } from './messageStore'
import type { AGUIChatRequest } from '../agui/client'
import type { AGUIEvent, Message, SkillCardData, UIComponentCardData, UnknownSkillData } from '../types'
import * as api from '../services/api'

vi.mock('../services/api', () => ({
  listMessages: vi.fn().mockResolvedValue([]),
  confirmHITL: vi.fn().mockResolvedValue(undefined),
  regenerateMessage: vi.fn(),
}))

type StreamHarness = {
  request: AGUIChatRequest
  onEvent: (event: AGUIEvent) => void
  controller: AbortController
}

const streamByConversation = new Map<string, StreamHarness>()

vi.mock('../agui/client', () => ({
  loadActiveRun: vi.fn(() => null),
  saveActiveRun: vi.fn(),
  clearActiveRun: vi.fn(),
  getRunStatus: vi.fn().mockResolvedValue(null),
  recoverRun: vi.fn(),
  runAgent: vi.fn(
    (
      request: AGUIChatRequest,
      onEvent: (event: AGUIEvent) => void,
      _onError?: (error: Error) => void,
      _onComplete?: () => void,
    ) => {
      const controller = new AbortController()
      streamByConversation.set(request.conversationId, { request, onEvent, controller })
      return controller
    },
  ),
}))

function emit(conversationId: string, event: AGUIEvent) {
  const harness = streamByConversation.get(conversationId)
  if (!harness) {
    throw new Error(`missing stream harness for ${conversationId}`)
  }
  harness.onEvent(event)
}

function agentMessage(conversationId: string): Message | undefined {
  return (useMessageStore.getState().messages[conversationId] || []).find((message) => message.senderType === 'agent')
}

function cards(conversationId: string): SkillCardData[] {
  return agentMessage(conversationId)?.skillCards || []
}

function startStream(conversationId: string, messageId = 'msg-ui') {
  const { sendMessage } = useMessageStore.getState()
  sendMessage(conversationId, 'render a component', { agentName: 'code-agent' })
  emit(conversationId, { type: 'TEXT_MESSAGE_START', messageId })
  return messageId
}

function componentEvent(overrides: Record<string, unknown> = {}): AGUIEvent {
  return {
    type: 'ui.component',
    id: 'evt-1',
    runId: 'run-ui',
    messageId: 'msg-ui',
    taskId: 'task-ui',
    seq: 7,
    subIndex: 0,
    component: 'html_sandbox',
    props: { title: 'demo' },
    schemaVersion: 1,
    artifactId: 'art_a1b2c3d4e5f6a1b2c3d4e5f6',
    capabilities: ['read_only'],
    ...overrides,
  }
}

function firstCard(conversationId: string): SkillCardData {
  const list = cards(conversationId)
  expect(list.length).toBe(1)
  return list[0]
}

beforeEach(() => {
  streamByConversation.clear()
  useMessageStore.setState({
    messages: {},
    streamingByConversation: {},
    abortControllersByConversation: {},
    orchestrationByConversation: {},
    confirmationByConversation: {},
    pinnedMessageIdsByConversation: {},
  })
})

describe('ui.component reducer appends allowlisted cards', () => {
  it('appends one ui_component card for a valid event', () => {
    startStream('conv-ui-valid')
    emit('conv-ui-valid', componentEvent())

    const card = firstCard('conv-ui-valid')
    expect(card.type).toBe('ui_component')
    const componentCard = card as UIComponentCardData
    expect(componentCard.component).toBe('html_sandbox')
    expect(componentCard.schemaVersion).toBe(1)
    expect(componentCard.artifactId).toBe('art_a1b2c3d4e5f6a1b2c3d4e5f6')
    expect(componentCard.capabilities).toEqual(['read_only'])
    expect(componentCard.props).toEqual({ title: 'demo' })
    expect(componentCard.eventId).toBe('evt-1')
    expect(componentCard.seq).toBe(7)
    expect(componentCard.runId).toBe('run-ui')
    expect(agentMessage('conv-ui-valid')?.agentName).toBe('code-agent')
  })

  it('keeps the card after the message finishes', () => {
    startStream('conv-ui-finish')
    emit('conv-ui-finish', componentEvent())
    emit('conv-ui-finish', { type: 'TEXT_MESSAGE_END', messageId: 'msg-ui', runId: 'run-ui' })

    expect(agentMessage('conv-ui-finish')?.status).toBe('sent')
    expect(cards('conv-ui-finish').length).toBe(1)
  })

  it('renders distinct cards for distinct seq values', () => {
    const conversationId = 'conv-ui-distinct'
    startStream(conversationId)
    emit(conversationId, componentEvent({ id: 'evt-1', seq: 7 }))
    emit(conversationId, componentEvent({ id: 'evt-2', seq: 8, artifactId: 'art_b2c3d4e5f6a1b2c3d4e5f6a1' }))

    expect(cards(conversationId).length).toBe(2)
  })
})

describe('ui.component reducer is idempotent by eventId / seq', () => {
  it('ignores a replayed event carrying the same eventId', () => {
    const conversationId = 'conv-ui-dup-id'
    startStream(conversationId)
    const event = componentEvent({ id: 'evt-dup', seq: 7, subIndex: 0 })

    emit(conversationId, event)
    emit(conversationId, event)
    emit(conversationId, event)

    expect(cards(conversationId).length).toBe(1)
  })

  it('ignores a replayed event identified by runId + seq + subIndex when eventId is absent', () => {
    const conversationId = 'conv-ui-dup-seq'
    startStream(conversationId)
    const event = componentEvent({ id: undefined, seq: 12, subIndex: 1 })

    emit(conversationId, event)
    emit(conversationId, event)

    expect(cards(conversationId).length).toBe(1)
  })

  it('treats the same seq with a different subIndex as a different event', () => {
    const conversationId = 'conv-ui-subindex'
    startStream(conversationId)

    emit(conversationId, componentEvent({ id: undefined, seq: 12, subIndex: 0 }))
    emit(conversationId, componentEvent({ id: undefined, seq: 12, subIndex: 1 }))

    expect(cards(conversationId).length).toBe(2)
  })

  it('de-duplicates rejected events too', () => {
    const conversationId = 'conv-ui-dup-invalid'
    startStream(conversationId)
    const event = componentEvent({ id: 'evt-bad', component: 'dangerous_iframe' })

    emit(conversationId, event)
    emit(conversationId, event)

    expect(cards(conversationId).length).toBe(1)
    expect(firstCard(conversationId).type).toBe('unknown_skill')
  })

  it('still appends when the event carries no cursor at all', () => {
    // 契约规定 Gateway 不得下发无 id/seq 的 ui.component；此时前端无法去重。
    const conversationId = 'conv-ui-no-cursor'
    startStream(conversationId)

    emit(conversationId, componentEvent({ id: undefined, seq: undefined, subIndex: undefined }))

    expect(cards(conversationId).length).toBe(1)
  })
})

describe('ui.component reducer falls back to the text card', () => {
  function expectFallback(conversationId: string, reason: string, componentToken: string) {
    const card = firstCard(conversationId) as UnknownSkillData
    expect(card.type).toBe('unknown_skill')
    expect(card.toolName).toBe('ui_component')
    const args = card.args as Record<string, unknown>
    expect(args.reason).toBe(reason)
    expect(args.component).toBe(componentToken)
    expect(typeof args.artifactId).toBe('string')
    expect(args.schemaVersion).toBe('1')
  }

  it('unknown component falls through to UnknownSkillFallback data', () => {
    const conversationId = 'conv-ui-unknown'
    startStream(conversationId)
    emit(conversationId, componentEvent({ component: 'dangerous_iframe' }))

    expectFallback(conversationId, 'component_not_allowed', 'dangerous_iframe')
  })

  it('hostile component names are redacted instead of rendered', () => {
    const conversationId = 'conv-ui-hostile-name'
    startStream(conversationId)
    emit(conversationId, componentEvent({ component: '<img src=x onerror=alert(1)>' }))

    const card = firstCard(conversationId) as UnknownSkillData
    expect((card.args as Record<string, unknown>).component).toBe('<redacted>')
    expect(JSON.stringify(card.args)).not.toContain('onerror')
  })

  it('rejects unsupported schemaVersion', () => {
    const conversationId = 'conv-ui-version'
    startStream(conversationId)
    emit(conversationId, componentEvent({ schemaVersion: 2 }))

    expect(firstCard(conversationId).type).toBe('unknown_skill')
    expect((firstCard(conversationId) as UnknownSkillData).args).toMatchObject({
      reason: 'schema_version_unsupported',
      component: 'html_sandbox',
    })
  })

  it('rejects props that are not a plain object without echoing them', () => {
    const conversationId = 'conv-ui-props'
    startStream(conversationId)
    emit(conversationId, componentEvent({ props: ['<script>alert(1)</script>'] }))

    const card = firstCard(conversationId) as UnknownSkillData
    expect(card.args).toMatchObject({ reason: 'props_invalid', component: 'html_sandbox' })
    expect(JSON.stringify(card.args)).not.toContain('alert(1)')
    expect(JSON.stringify(card.args)).not.toContain('script')
  })

  it('rejects props above the 64KB limit', () => {
    const conversationId = 'conv-ui-props-size'
    startStream(conversationId)
    emit(conversationId, componentEvent({ props: { blob: 'a'.repeat(65536) } }))

    expect((firstCard(conversationId) as UnknownSkillData).args).toMatchObject({ reason: 'props_too_large' })
  })

  it('rejects artifact ids that do not match the artifact id shape', () => {
    const conversationId = 'conv-ui-artifact-id'
    startStream(conversationId)
    emit(conversationId, componentEvent({ artifactId: 'artifact_../../etc/passwd' }))

    const card = firstCard(conversationId) as UnknownSkillData
    expect(card.args).toMatchObject({ reason: 'artifact_id_invalid', artifactId: '<redacted>' })
    expect(JSON.stringify(card.args)).not.toContain('passwd')
  })

  it('rejects capabilities outside the known set', () => {
    const conversationId = 'conv-ui-capabilities'
    startStream(conversationId)
    emit(conversationId, componentEvent({ capabilities: ['read_only', 'network_access'] }))

    expect((firstCard(conversationId) as UnknownSkillData).args).toMatchObject({ reason: 'capabilities_invalid' })
  })
})

describe('ui.component does not disturb existing card paths', () => {
  it('keeps tool-call driven cards working alongside component events', () => {
    const conversationId = 'conv-ui-mixed'
    startStream(conversationId)

    emit(conversationId, {
      type: 'TOOL_CALL_START',
      id: 'tc-1',
      toolCall: { id: 'tc-1', name: 'chart_render' },
    })
    emit(conversationId, {
      type: 'TOOL_CALL_ARGS',
      id: 'tc-1',
      delta: JSON.stringify({ chartType: 'bar', data: [{ label: 'a', value: 1 }] }),
    })
    emit(conversationId, { type: 'TOOL_CALL_END', id: 'tc-1' })
    emit(conversationId, componentEvent({ id: 'evt-9', seq: 20 }))

    const list = cards(conversationId)
    expect(list.map((card) => card.type)).toEqual(['chart_render', 'ui_component'])
  })

  it('keeps unknown tool names on the existing fallback card', () => {
    const conversationId = 'conv-ui-unknown-tool'
    startStream(conversationId)

    emit(conversationId, { type: 'TOOL_CALL_START', id: 'tc-2', toolCall: { id: 'tc-2', name: 'mystery_tool' } })
    emit(conversationId, { type: 'TOOL_CALL_ARGS', id: 'tc-2', delta: '{"a":1}' })
    emit(conversationId, { type: 'TOOL_CALL_END', id: 'tc-2' })

    const card = firstCard(conversationId) as UnknownSkillData
    expect(card.type).toBe('unknown_skill')
    expect(card.toolName).toBe('mystery_tool')
  })
})
