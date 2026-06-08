import { create } from 'zustand'
import type { Message, AGUIEvent, CodeBlock, WebPreviewBlock } from '../types'
import type { AgentName } from '../lib/agents'
import * as api from '../services/api'
import { runAgent, type AGUIChatRequest } from '../agui/client'
import { DEFAULT_AGENT_NAME, getAgentDisplayName, isConcreteAgentName, isSupportedAgentName, normalizeAgentName } from '../lib/agents'
import type { OrchestrationInfo } from '../components/OrchestrationCard'
import { useConversationStore } from './conversationStore'
import { persistTitle } from '../lib/conversationTitles'

interface SendMessageOptions {
  agentName?: AgentName
}

export interface PendingConfirmation {
  runId: string
  actionId: string
  agentNames: string[]
  tasks: Array<{ taskId?: string; agentName?: string; content?: string; dependsOn?: string[]; priority?: number; riskLevel?: string }>
  intentSummary?: string
  strategy?: string
  status: 'pending' | 'confirmed' | 'rejected'
  error?: string
  rejectReason?: string
}

interface MessageState {
  messages: Record<string, Message[]> // conversationId -> messages[]
  streamingByConversation: Record<string, boolean>
  abortControllersByConversation: Record<string, AbortController | null>
  orchestrationByConversation: Record<string, OrchestrationInfo>
  confirmationByConversation: Record<string, PendingConfirmation | null>

  loadMessages: (conversationId: string) => Promise<void>
  sendMessage: (conversationId: string, content: string, options?: SendMessageOptions) => void
  isStreaming: (conversationId: string) => boolean
  stopStreaming: (conversationId: string) => void
  getConfirmation: (conversationId: string) => PendingConfirmation | null
  confirmPlan: (conversationId: string, runId: string, actionId: string, confirmed: boolean, reason?: string) => Promise<void>
  clearConfirmation: (conversationId: string) => void
}

interface StoredMessage {
  id: string
  conversationId?: string
  senderType?: string
  senderName?: string
  senderDisplayName?: string
  agentName?: string
  author?: string
  role?: string
  content?: string
  text?: string
  runId?: string
  stepId?: string
  sseMessageId?: string
  status?: string
  errorCode?: string
  errorMessage?: string
  artifacts?: string
  createdAt: string
}

type ParsedArtifacts = {
  codeBlocks?: CodeBlock[]
  webPreviews?: WebPreviewBlock[]
}

function setConversationStreaming(
  state: MessageState,
  conversationId: string,
  streaming: boolean,
): Pick<MessageState, 'streamingByConversation'> {
  return {
    streamingByConversation: {
      ...state.streamingByConversation,
      [conversationId]: streaming,
    },
  }
}

function setConversationAbortController(
  state: MessageState,
  conversationId: string,
  controller: AbortController | null,
): Pick<MessageState, 'abortControllersByConversation'> {
  return {
    abortControllersByConversation: {
      ...state.abortControllersByConversation,
      [conversationId]: controller,
    },
  }
}

function pickText(...values: unknown[]): string | undefined {
  for (const value of values) {
    if (typeof value !== 'string') {
      continue
    }
    const trimmed = value.trim()
    if (trimmed !== '') {
      return trimmed
    }
  }
  return undefined
}

function resolveContentChunk(event: AGUIEvent): string {
  return pickText(event.delta, event.content, event.text) ?? ''
}

function resolveErrorText(event: AGUIEvent): string {
  if (typeof event.error === 'string') {
    const errorText = event.error.trim()
    if (errorText !== '') {
      return sanitizeErrorText(errorText)
    }
  }
  if (event.error && typeof event.error === 'object') {
    const message = pickText(event.error.message)
    if (message) {
      return sanitizeErrorText(message)
    }
  }
  const fallback = pickText(event.text, event.content)
  return sanitizeErrorText(fallback || 'Error')
}

function generateConversationTitle(text: string): string {
  const cleaned = text
    .replace(/\n/g, ' ')
    .replace(/\s+/g, ' ')
    .trim()
  // Max ~30 characters (roughly 30 ASCII chars or 15 CJK chars).
  const maxLen = 30
  if (cleaned.length <= maxLen) {
    return cleaned || 'New Conversation'
  }
  return cleaned.slice(0, maxLen).trimEnd() + '…'
}

function sanitizeErrorText(text: string): string {
  if (!text) {
    return 'Error'
  }
  let out = text
  // Strip assignment-style secrets: KEY=value
  out = out.replace(
    /\b(?:OPENAI_API_KEY|ANTHROPIC_API_KEY|AGENTHUB_API_TOKEN|DATABASE_URL|DB_PASSWORD|MYSQL_ROOT_PASSWORD)\s*=\s*(?:\S+|"[^"]*"|'[^']*')/gi,
    '[redacted]',
  )
  // Strip private-key blocks
  out = out.replace(/-----BEGIN [A-Z ]*PRIVATE KEY-----[\s\S]*?-----END [A-Z ]*PRIVATE KEY-----/gi, '[redacted]')
  // Strip sk- prefixed tokens
  out = out.replace(/\bsk-[A-Za-z0-9_-]{20,}\b/g, '[redacted]')
  // Strip Windows-style file paths
  out = out.replace(/[a-zA-Z]:\\(?:\S+\\\S*)+/g, '[path]')
  // Strip Unix-style file paths
  out = out.replace(/(?:\/(?:\S+\/)+\S+)/g, (match) => {
    // Only redact paths that look like filesystem paths, not ordinary words
    if (match.length > 8 && match.split('/').length >= 3) {
      return '[path]'
    }
    return match
  })
  // Strip internal URLs/hostnames (http://host:port, localhost, .local, .internal)
  out = out.replace(/\bhttps?:\/\/[^\s,;}\]\)"']+/gi, (match) => {
    // Redact URLs containing internal-looking hostnames or ports
    if (/:\d+/.test(match) || /localhost/i.test(match) || /\.local\b/i.test(match) ||
        /\.internal\b/i.test(match) || /\.lan\b/i.test(match) || /\.corp\b/i.test(match) ||
        /127\.0\.0\.\d+/.test(match) || /10\.\d+\.\d+\.\d+/.test(match) ||
        /192\.168\.\d+\.\d+/.test(match) || /172\.(1[6-9]|2\d|3[01])\.\d+\.\d+/.test(match)) {
      return '[url]'
    }
    return match
  })
  // If text contains stack trace / panic markers, replace entirely
  if (/\bpanic\b/i.test(out) || /\bstack\b.*\btrace\b/i.test(out) || /\bfatal\b.*\berror\b/i.test(out)) {
    return 'internal error'
  }
  return out || 'Error'
}

function normalizeToolArguments(raw: unknown): Record<string, unknown> | null {
  if (raw == null) {
    return null
  }
  if (typeof raw === 'string') {
    const text = raw.trim()
    if (text === '') {
      return null
    }
    try {
      const decoded: unknown = JSON.parse(text)
      if (decoded && typeof decoded === 'object' && !Array.isArray(decoded)) {
        return decoded as Record<string, unknown>
      }
      return null
    } catch {
      return null
    }
  }
  if (typeof raw === 'object' && !Array.isArray(raw)) {
    return raw as Record<string, unknown>
  }
  return null
}

function getStringField(record: Record<string, unknown>, key: string): string {
  const value = record[key]
  if (typeof value !== 'string') {
    return ''
  }
  return value
}

function isLikelyHTML(content: string): boolean {
  return /<[a-zA-Z][^>]*>/.test(content) && /<\/[a-zA-Z][^>]*>/.test(content)
}

/**
 * Remove markdown code fences (```...``` and ```html...```) from content
 * before checking if it contains HTML. This prevents code-agent HTML code
 * examples from being mistaken for web preview output.
 */
function stripMarkdownCodeFences(content: string): string {
  // Remove triple-backtick fenced blocks (with or without language tag).
  return content.replace(/```[\s\S]*?```/g, '')
}

function maybeExtractHTMLSnippet(content: string): string | undefined {
  const text = content.trim()
  if (text === '') {
    return undefined
  }
  // Only consider the content as HTML if it predominantly starts with an HTML tag
  // and does not contain markdown code fences wrapping HTML examples.
  if (!text.trimStart().startsWith('<')) {
    return undefined
  }
  const cleaned = stripMarkdownCodeFences(text).trim()
  if (!cleaned) {
    return undefined
  }
  if (!isLikelyHTML(cleaned)) {
    return undefined
  }
  return text
}

function parseArtifacts(artifactsStr: string): ParsedArtifacts {
  if (!artifactsStr) {
    return {}
  }

  let decoded: unknown
  try {
    decoded = JSON.parse(artifactsStr)
  } catch {
    return {}
  }

  if (!Array.isArray(decoded)) {
    return {}
  }

  const codeBlocks: CodeBlock[] = []
  const webPreviews: WebPreviewBlock[] = []

  for (const item of decoded) {
    if (!item || typeof item !== 'object' || Array.isArray(item)) {
      continue
    }
    const artifact = item as {
      type?: unknown
      title?: unknown
      content?: unknown
      metadata?: unknown
    }
    const type = typeof artifact.type === 'string' ? artifact.type : ''
    const title = typeof artifact.title === 'string' ? artifact.title : ''
    const content = typeof artifact.content === 'string' ? artifact.content : ''
    const metadata =
      artifact.metadata && typeof artifact.metadata === 'object' && !Array.isArray(artifact.metadata)
        ? (artifact.metadata as Record<string, unknown>)
        : {}
    const language = typeof metadata.language === 'string' ? metadata.language : ''

    if (type === 'code') {
      codeBlocks.push({
        code: content,
        language,
        filename: title,
      })
      continue
    }

    if (type === 'webpage' || type === 'html') {
      webPreviews.push({
        html: content,
        title: title || 'web-preview.html',
      })
    }
  }

  return {
    codeBlocks: codeBlocks.length > 0 ? codeBlocks : undefined,
    webPreviews: webPreviews.length > 0 ? webPreviews : undefined,
  }
}

export const useMessageStore = create<MessageState>((set, get) => ({
  messages: {},
  streamingByConversation: {},
  abortControllersByConversation: {},
  orchestrationByConversation: {},
  confirmationByConversation: {},

  loadMessages: async (conversationId: string) => {
    try {
      const rawMessages = (await api.listMessages(conversationId)) as unknown as StoredMessage[]
      const formatted: Message[] = rawMessages.map((raw) => {
        const senderType: Message['senderType'] =
          raw.senderType === 'user' || raw.role === 'user' || raw.author === 'user' ? 'user' : 'agent'
        const content = pickText(raw.content, raw.text) || ''
        const parsedArtifacts = parseArtifacts(raw.artifacts || '')
        const fallbackAgentName =
          senderType === 'agent'
            ? pickText(raw.agentName, raw.senderName, raw.author)
            : undefined
        const normalizedAgentName = fallbackAgentName
          ? normalizeAgentName(fallbackAgentName)
          : undefined

        return {
          id: raw.id,
          conversationId: raw.conversationId || conversationId,
          senderType,
          senderName: pickText(
            raw.senderDisplayName,
            raw.senderName,
            raw.agentName,
            senderType === 'agent' ? raw.author : undefined,
          ),
          agentName: normalizedAgentName,
          content,
          status:
            raw.status === 'failed'
              ? ('failed' as const)
              : raw.status === 'streaming'
                ? ('streaming' as const)
                : ('sent' as const),
          codeBlocks: parsedArtifacts.codeBlocks,
          webPreviews: parsedArtifacts.webPreviews,
          runId: raw.runId,
          stepId: raw.stepId,
          sseMessageId: raw.sseMessageId,
          errorCode: raw.errorCode,
          errorMessage: raw.errorMessage,
          createdAt: raw.createdAt,
        }
      })
      set((s) => ({
        messages: { ...s.messages, [conversationId]: formatted },
      }))
    } catch {
      // ignore load errors silently
    }
  },

  sendMessage: (conversationId: string, content: string, options?: SendMessageOptions) => {
    const selectedAgentName = normalizeAgentName(options?.agentName || DEFAULT_AGENT_NAME)
    const fallbackSenderName = getAgentDisplayName(selectedAgentName)

    const userMsg: Message = {
      id: `temp-${Date.now()}`,
      conversationId,
      senderType: 'user',
      content,
      status: 'sent',
      createdAt: new Date().toISOString(),
    }

    set((s) => ({
      messages: {
        ...s.messages,
        [conversationId]: [...(s.messages[conversationId] || []), userMsg],
      },
      ...setConversationStreaming(s, conversationId, true),
    }))

    // Auto-generate conversation title from first user message.
    const currentMessages = get().messages[conversationId] || []
    const isFirstMessage = currentMessages.length <= 1
    if (isFirstMessage) {
      const title = generateConversationTitle(content)
      useConversationStore.getState().updateTitle(conversationId, title)
      persistTitle(conversationId, title)
    }

    const request: AGUIChatRequest = {
      conversationId,
      message: content,
    }
    // Only pass agentName for concrete agents; "auto" lets orchestrator decide.
    if (options?.agentName && options.agentName !== 'auto') {
      request.agentName = options.agentName
    }

    let agentMsgId = ''
    let agentContent = ''
    let currentAgentName: string = selectedAgentName
    let currentSenderName = fallbackSenderName
    let codeBlocks: CodeBlock[] = []
    let webPreviews: WebPreviewBlock[] = []
    // Track whether web-related artifact/tool evidence was seen during streaming.
    // Used to gate content-based Web Preview extraction when agentName is 'auto'.
    let hasWebArtifactEvidence = false
    const toolCallArgs: Record<string, string> = {}
    const toolCallNames: Record<string, string> = {}

    const resolveEventAgentName = (event: AGUIEvent): string => {
      const eventAgentName = pickText(
        event.agentName,
        event.sender?.name,
        event.author,
        typeof event.metadata?.agentName === 'string' ? event.metadata.agentName : undefined,
        typeof event.stateDelta?.agentName === 'string' ? event.stateDelta.agentName : undefined,
      )
      if (!eventAgentName) {
        return selectedAgentName
      }
      return eventAgentName
    }

    const resolveEventSenderName = (event: AGUIEvent, eventAgentName: string): string => {
      const explicitSender = pickText(
        event.senderName,
        event.sender?.displayName,
        event.sender?.name,
      )
      if (explicitSender) {
        return explicitSender
      }
      if (!isSupportedAgentName(eventAgentName)) {
        return eventAgentName
      }
      return getAgentDisplayName(eventAgentName)
    }

    const updateAgentMessage = (updater: (message: Message) => Message) => {
      if (!agentMsgId) {
        return
      }
      set((s) => ({
        messages: {
          ...s.messages,
          [conversationId]: (s.messages[conversationId] || []).map((message) =>
            message.id === agentMsgId ? updater(message) : message,
          ),
        },
      }))
    }

    const ensureAgentMessage = (event: AGUIEvent) => {
      // If messageId changes (new agent message), create a fresh bubble.
      if (agentMsgId && event.messageId && event.messageId !== agentMsgId) {
        agentMsgId = event.messageId
        agentContent = ''
        codeBlocks = []
        webPreviews = []
      }

      if (!agentMsgId) {
        agentMsgId = event.messageId || event.id || `agent-${Date.now()}`
      }

      currentAgentName = resolveEventAgentName(event)
      currentSenderName = resolveEventSenderName(event, currentAgentName)

      set((s) => {
        const currentMessages = s.messages[conversationId] || []
        const existing = currentMessages.find((message) => message.id === agentMsgId)
        if (existing) {
          return {
            messages: {
              ...s.messages,
              [conversationId]: currentMessages.map((message) =>
                message.id === agentMsgId
                  ? {
                      ...message,
                      senderName: message.senderName || currentSenderName,
                      agentName: message.agentName || currentAgentName,
                    }
                  : message,
              ),
            },
          }
        }
        return {
          messages: {
            ...s.messages,
            [conversationId]: [
              ...currentMessages,
              {
                id: agentMsgId,
                conversationId,
                senderType: 'agent',
                senderName: currentSenderName,
                agentName: currentAgentName,
                content: '',
                status: 'streaming',
                createdAt: new Date().toISOString(),
              },
            ],
          },
        }
      })
    }

    const syncPreviewBlocks = () => {
      updateAgentMessage((message) => ({
        ...message,
        codeBlocks: codeBlocks.length > 0 ? [...codeBlocks] : undefined,
        webPreviews: webPreviews.length > 0 ? [...webPreviews] : undefined,
      }))
    }

    const appendWebPreviewFromMessageContent = () => {
      // Exit early if web previews were already created via tool calls or artifact deltas.
      if (webPreviews.length > 0) {
        return
      }

      // Determine whether we should extract HTML from raw message content.
      // Priority: 1) SSE-resolved agent name, 2) explicit UI selection, 3) artifact/tool evidence.
      const isWebAgentResolved =
        isConcreteAgentName(currentAgentName) && currentAgentName === 'web-agent'
      const isWebAgentSelected =
        isConcreteAgentName(selectedAgentName) && selectedAgentName === 'web-agent'
      // In auto mode, only extract when there's clear web artifact evidence AND
      // the content is definitively HTML (not markdown-wrapped code examples).
      const canExtract =
        isWebAgentResolved ||
        isWebAgentSelected ||
        (hasWebArtifactEvidence && currentAgentName === 'auto')

      if (!canExtract) {
        return
      }

      const htmlSnippet = maybeExtractHTMLSnippet(agentContent)
      if (!htmlSnippet) {
        return
      }
      webPreviews.push({
        html: htmlSnippet,
        title: 'web-preview.html',
        agentName: isConcreteAgentName(currentAgentName) ? currentAgentName : 'web-agent',
      })
      syncPreviewBlocks()
    }

    const appendCodePreview = (args: Record<string, unknown>) => {
      const code = getStringField(args, 'code')
      if (!code) {
        return
      }
      codeBlocks.push({
        code,
        language: getStringField(args, 'language'),
        filename: getStringField(args, 'filename'),
      })
      syncPreviewBlocks()
    }

    const appendWebPreview = (args: Record<string, unknown>) => {
      const html = pickText(args.html, args.content)
      if (!html) {
        return
      }
      const supportedAgentName = isSupportedAgentName(currentAgentName)
        ? currentAgentName
        : selectedAgentName

      webPreviews.push({
        html,
        title: pickText(args.title, args.filename) || 'web-preview.html',
        agentName: supportedAgentName,
      })
      syncPreviewBlocks()
    }

    const handleToolPayload = (toolName: string, rawArgs: unknown) => {
      const args = normalizeToolArguments(rawArgs)
      if (!args) {
        return
      }
      if (toolName === 'code_preview') {
        appendCodePreview(args)
      }
      if (toolName === 'web_preview' || toolName === 'generate_html_snippet') {
        hasWebArtifactEvidence = true
        appendWebPreview(args)
      }
    }

    // Progressive WebPreview: update preview HTML as tool args stream in.
    // This lets the Source tab show the HTML being built in real time.
    const updateProgressiveWebPreview = (toolName: string, rawArgs: string) => {
      if (toolName !== 'web_preview' && toolName !== 'generate_html_snippet') {
        return
      }
      // Try to extract HTML content from partial args.
      let html = ''
      try {
        const parsed = JSON.parse(rawArgs)
        if (parsed && typeof parsed === 'object' && !Array.isArray(parsed)) {
          html = (typeof parsed.html === 'string' ? parsed.html : '') ||
            (typeof parsed.content === 'string' ? parsed.content : '')
        }
      } catch {
        // Raw args are not yet valid JSON; try heuristic extraction.
        const htmlMatch = rawArgs.match(/"html"\s*:\s*"([^"]*(?:\\.[^"]*)*)"/)
        if (htmlMatch) {
          html = htmlMatch[1].replace(/\\"/g, '"').replace(/\\n/g, '\n')
        } else {
          const contentMatch = rawArgs.match(/"content"\s*:\s*"([^"]*(?:\\.[^"]*)*)"/)
          if (contentMatch) {
            html = contentMatch[1].replace(/\\"/g, '"').replace(/\\n/g, '\n')
          }
        }
      }

      if (!html) {
        return
      }

      hasWebArtifactEvidence = true
      const supportedAgentName = isSupportedAgentName(currentAgentName)
        ? currentAgentName
        : selectedAgentName

      // Update or create a streaming web preview block.
      const existingIdx = webPreviews.findIndex(
        (wp) => wp.title === 'web-preview.html' && wp.agentName === supportedAgentName,
      )
      if (existingIdx >= 0) {
        webPreviews[existingIdx] = {
          ...webPreviews[existingIdx],
          html,
        }
      } else {
        webPreviews.push({
          html,
          title: 'web-preview.html',
          agentName: supportedAgentName,
        })
      }
      syncPreviewBlocks()
    }

    const handleArtifactDelta = (event: AGUIEvent) => {
      if (!event.artifact) {
        return
      }
      const type = pickText(event.artifact.type) || ''
      const title = pickText(event.artifact.title) || ''
      const artifactContent = pickText(event.artifact.content) || ''
      const language = pickText(event.artifact.metadata?.language) || ''

      if (type === 'code') {
        codeBlocks.push({
          code: artifactContent,
          language,
          filename: title,
        })
        syncPreviewBlocks()
      }

      if (type === 'webpage' || type === 'html') {
        hasWebArtifactEvidence = true
        const supportedAgentName = isSupportedAgentName(currentAgentName)
          ? currentAgentName
          : selectedAgentName
        webPreviews.push({
          html: artifactContent,
          title: title || 'web-preview.html',
          agentName: supportedAgentName,
        })
        syncPreviewBlocks()
      }
    }

    const finishStreamingMessage = () => {
      appendWebPreviewFromMessageContent()
      updateAgentMessage((message) => ({ ...message, status: 'sent' }))
    }

    const failStreamingMessage = (errorText: string) => {
      updateAgentMessage((message) => ({
        ...message,
        status: 'failed',
        content: agentContent || errorText,
      }))
    }

    const controller = runAgent(
      request,
      (event: AGUIEvent) => {
        switch (event.type) {
          case 'TEXT_MESSAGE_START': {
            agentMsgId = event.messageId || event.id || `agent-${Date.now()}`
            agentContent = ''
            codeBlocks = []
            webPreviews = []
            ensureAgentMessage(event)
            break
          }

          case 'TEXT_MESSAGE_CONTENT':
          case 'message':
          case 'message.delta': {
            ensureAgentMessage(event)
            const chunk = resolveContentChunk(event)
            if (chunk) {
              if (event.type === 'message' && !event.partial) {
                if (agentContent && chunk.startsWith(agentContent)) {
                  agentContent = chunk
                } else if (!agentContent) {
                  agentContent = chunk
                } else {
                  agentContent += chunk
                }
              } else {
                agentContent += chunk
              }
              updateAgentMessage((message) => ({ ...message, content: agentContent }))
            }
            if (event.type === 'message' && event.final) {
              finishStreamingMessage()
            }
            break
          }

          case 'TEXT_MESSAGE_END':
          case 'message.end': {
            // If TEXT_MESSAGE_END carries full content, use the more complete
            // version to avoid losing content from partial deltas.
            const endDelta = resolveContentChunk(event)
            if (endDelta && endDelta.length > agentContent.length) {
              agentContent = endDelta
              updateAgentMessage((message) => ({ ...message, content: agentContent }))
            }
            finishStreamingMessage()
            break
          }

          case 'RUN_STARTED':
            // Initialize run state — store runId and phase for UI state tracking.
            // The streaming state is already set by sendMessage.
            if (event.state && typeof event.state === 'object') {
              const phase = typeof event.state.phase === 'string' ? event.state.phase : ''
              if (phase) {
                set((s) => ({
                  orchestrationByConversation: {
                    ...s.orchestrationByConversation,
                    [conversationId]: {
                      ...(s.orchestrationByConversation[conversationId] || {}),
                      phase,
                    },
                  },
                }))
              }
            }
            break

          case 'STATE_UPDATE':
          case 'state.delta': {
            // Persist state updates for orchestration progress display.
            const state = (event.state ?? event.stateDelta) as Record<string, unknown> | undefined
            if (state && typeof state === 'object') {
              const phaseInfo = pickText(
                typeof state.phase === 'string' ? state.phase : undefined,
                typeof state.status === 'string' ? state.status : undefined,
              )
              // Track the current execution phase so the UI can show thinking/planning/executing states.
              if (phaseInfo) {
                // eslint-disable-next-line no-console
                console.debug(`[orchestrator] phase: ${phaseInfo}`, state)
                set((s) => ({
                  orchestrationByConversation: {
                    ...s.orchestrationByConversation,
                    [conversationId]: {
                      ...(s.orchestrationByConversation[conversationId] || {}),
                      phase: phaseInfo,
                    },
                  },
                }))
              }
              // If a messageId is present in the state, track it for multi-agent separation.
              if (typeof state.messageId === 'string' && state.messageId) {
                if (!agentMsgId) {
                  agentMsgId = state.messageId
                  ensureAgentMessage(event)
                }
              }
              // Populate orchestration info for display.
              const orchInfo: OrchestrationInfo = {}
              let hasOrchInfo = false
              const intentValue = pickText(
                typeof state.intentSummary === 'string' ? state.intentSummary : undefined,
                typeof state.intent === 'string' ? state.intent : undefined,
              )
              if (intentValue) {
                orchInfo.intent = intentValue
                hasOrchInfo = true
              }
              if (typeof state.reasoning === 'string' && state.reasoning) {
                orchInfo.reasoning = state.reasoning as string
                hasOrchInfo = true
              }
              if (typeof state.strategy === 'string' && state.strategy) {
                orchInfo.strategy = state.strategy as string
                hasOrchInfo = true
              }
              if (typeof state.taskCount === 'number') {
                orchInfo.taskCount = state.taskCount as number
                hasOrchInfo = true
              }
              if (typeof state.plannerSource === 'string' && state.plannerSource) {
                orchInfo.plannerSource = state.plannerSource as string
                hasOrchInfo = true
              }
              if (typeof state.plannerSourceLabel === 'string' && state.plannerSourceLabel) {
                orchInfo.plannerSourceLabel = state.plannerSourceLabel as string
                hasOrchInfo = true
              }
              if (typeof state.plannerModel === 'string' && state.plannerModel) {
                orchInfo.plannerModel = state.plannerModel as string
                hasOrchInfo = true
              }
              // Capture actual agent selection from orchestrator state.
              if (typeof state.selectedAgentName === 'string' && state.selectedAgentName) {
                orchInfo.selectedAgentName = state.selectedAgentName as string
                hasOrchInfo = true
              }
              if (typeof state.selectedAgentDisplayName === 'string' && state.selectedAgentDisplayName) {
                orchInfo.selectedAgentDisplayName = state.selectedAgentDisplayName as string
                hasOrchInfo = true
              }
              if (typeof state.taskAgentNames === 'string' && state.taskAgentNames) {
                orchInfo.taskAgentNames = state.taskAgentNames as string
                hasOrchInfo = true
              }
              // HITL plan confirmation: detect requiresConfirmation flag and planned agents/tasks.
              if (state.requiresConfirmation === true) {
                orchInfo.requiresConfirmation = true
                hasOrchInfo = true
              }
              if (typeof state.confirmationActionId === 'string' && state.confirmationActionId) {
                orchInfo.confirmationActionId = state.confirmationActionId as string
                hasOrchInfo = true
              }
              if (Array.isArray(state.plannedAgents)) {
                orchInfo.plannedAgents = state.plannedAgents as string[]
                hasOrchInfo = true
              }
              if (Array.isArray(state.tasks)) {
                orchInfo.plannedTasks = state.tasks as OrchestrationInfo['plannedTasks']
                hasOrchInfo = true
              }
              if (hasOrchInfo) {
                set((s) => ({
                  orchestrationByConversation: {
                    ...s.orchestrationByConversation,
                    [conversationId]: {
                      ...s.orchestrationByConversation[conversationId],
                      ...orchInfo,
                    },
                  },
                }))
              }
            }
            break
          }

          case 'TOOL_CALL_START':
            if (event.id) {
              toolCallArgs[event.id] = ''
              if (event.toolCall?.name) {
                toolCallNames[event.id] = event.toolCall.name
              }
              if (event.toolName) {
                toolCallNames[event.id] = event.toolName
              }
            } else if (event.toolCallId) {
              toolCallArgs[event.toolCallId] = ''
              if (event.toolName) {
                toolCallNames[event.toolCallId] = event.toolName
              }
            }
            break

          case 'TOOL_CALL_ARGS':
            if (event.id) {
              toolCallArgs[event.id] =
                (toolCallArgs[event.id] || '') + resolveContentChunk(event)
              const tn = event.toolCall?.name || toolCallNames[event.id] || ''
              updateProgressiveWebPreview(tn, toolCallArgs[event.id])
            } else if (event.toolCallId) {
              toolCallArgs[event.toolCallId] =
                (toolCallArgs[event.toolCallId] || '') + resolveContentChunk(event)
              const tn = event.toolName || toolCallNames[event.toolCallId] || ''
              updateProgressiveWebPreview(tn, toolCallArgs[event.toolCallId])
            }
            break

          case 'TOOL_CALL_END': {
            let toolName = ''
            let toolArgs = ''
            let toolId = ''
            if (event.id) {
              toolName = event.toolCall?.name || toolCallNames[event.id] || ''
              toolArgs = toolCallArgs[event.id] || ''
              toolId = event.id
            } else if (event.toolCallId) {
              toolName = event.toolName || toolCallNames[event.toolCallId] || ''
              toolArgs = toolCallArgs[event.toolCallId] || ''
              toolId = event.toolCallId
            }
            // Handle confirm_plan: set up pending HITL confirmation state.
            if (toolName === 'confirm_plan') {
              const parsed = normalizeToolArguments(toolArgs)
              if (parsed) {
                const confirmRunId = (typeof parsed.runId === 'string' ? parsed.runId : '') || ''
                const confirmPlanId = (typeof parsed.planId === 'string' ? parsed.planId : '') || toolId
                const confirmAgents: string[] = Array.isArray(parsed.plannedAgents)
                  ? (parsed.plannedAgents as string[])
                  : []
                const confirmTasks = Array.isArray(parsed.tasks)
                  ? (parsed.tasks as PendingConfirmation['tasks'])
                  : []
                const confirmSummary = typeof parsed.intentSummary === 'string' ? parsed.intentSummary : ''
                const confirmStrategy = typeof parsed.strategy === 'string' ? parsed.strategy : ''
                set((s) => ({
                  confirmationByConversation: {
                    ...s.confirmationByConversation,
                    [conversationId]: {
                      runId: confirmRunId,
                      actionId: confirmPlanId,
                      agentNames: confirmAgents,
                      tasks: confirmTasks,
                      intentSummary: confirmSummary,
                      strategy: confirmStrategy,
                      status: 'pending',
                    },
                  },
                }))
              }
            } else {
              handleToolPayload(toolName, toolArgs)
            }
            break
          }

          case 'tool.call': {
            ensureAgentMessage(event)
            const toolName = pickText(event.toolCall?.name, event.toolName) || ''
            const toolArgs = event.toolCall?.arguments ?? event.content
            handleToolPayload(toolName, toolArgs)
            break
          }

          case 'artifact.delta':
            ensureAgentMessage(event)
            handleArtifactDelta(event)
            break

          case 'RUN_FINISHED':
            finishStreamingMessage()
            set((s) => ({
              ...setConversationStreaming(s, conversationId, false),
              ...setConversationAbortController(s, conversationId, null),
            }))
            break

          case 'RUN_ERROR':
          case 'error': {
            ensureAgentMessage(event)
            const errorText = resolveErrorText(event)
            // Extract error code and details for structured error display.
            // Always pass through sanitizeErrorText to strip secrets/paths/traces.
            let errorCode = ''
            let errorMessage = ''
            if (event.error && typeof event.error === 'object') {
              errorCode = sanitizeErrorText((event.error as Record<string, unknown>).code as string || '')
              errorMessage = sanitizeErrorText((event.error as Record<string, unknown>).message as string || errorText)
            } else {
              errorMessage = errorText // already sanitized by resolveErrorText
            }
            // Determine the phase from orchestration state for context.
            const currentOrch = get().orchestrationByConversation[conversationId]
            const errorPhase = currentOrch?.phase || ''
            failStreamingMessage(errorMessage)
            updateAgentMessage((message) => ({
              ...message,
              errorCode: errorCode || 'RUN_ERROR',
              errorMessage,
            }))
            set((s) => ({
              ...setConversationStreaming(s, conversationId, false),
              ...setConversationAbortController(s, conversationId, null),
              orchestrationByConversation: {
                ...s.orchestrationByConversation,
                [conversationId]: {
                  ...(s.orchestrationByConversation[conversationId] || {}),
                  phase: 'error',
                  errorCode: errorCode || 'RUN_ERROR',
                  errorPhase: errorPhase || 'unknown',
                },
              },
            }))
            break
          }
        }
      },
      (error: Error) => {
        if (agentMsgId) {
          failStreamingMessage(error.message || 'Error')
        }
        set((s) => ({
          ...setConversationStreaming(s, conversationId, false),
          ...setConversationAbortController(s, conversationId, null),
        }))
      },
      () => {
        finishStreamingMessage()
        set((s) => ({
          ...setConversationStreaming(s, conversationId, false),
          ...setConversationAbortController(s, conversationId, null),
        }))
      },
    )

    set((s) => ({
      ...setConversationAbortController(s, conversationId, controller),
    }))
  },

  isStreaming: (conversationId: string) => {
    const { streamingByConversation } = get()
    return Boolean(streamingByConversation[conversationId])
  },

  stopStreaming: (conversationId: string) => {
    const { abortControllersByConversation } = get()
    const abortController = abortControllersByConversation[conversationId]
    if (abortController) {
      abortController.abort()
      set((s) => ({
        ...setConversationStreaming(s, conversationId, false),
        ...setConversationAbortController(s, conversationId, null),
      }))
    }
  },

  getConfirmation: (conversationId: string) => {
    const { confirmationByConversation } = get()
    return confirmationByConversation[conversationId] || null
  },

  confirmPlan: async (conversationId: string, runId: string, actionId: string, confirmed: boolean, reason?: string) => {
    const current = get().confirmationByConversation[conversationId]
    if (!current || current.status !== 'pending') {
      return
    }
    try {
      await api.confirmHITL({
        runId: runId || current.runId,
        actionId: actionId || current.actionId,
        confirmed,
        rejectReason: reason || '',
      })
      set((s) => ({
        confirmationByConversation: {
          ...s.confirmationByConversation,
          [conversationId]: {
            ...current,
            status: confirmed ? 'confirmed' : 'rejected',
            rejectReason: reason || '',
          },
        },
      }))
    } catch (err) {
      const errorMessage = err instanceof Error ? err.message : 'Confirmation failed'
      // Sanitize error — no internal URLs, tokens, or stack traces
      const safeMessage = sanitizeErrorText(errorMessage)
      set((s) => ({
        confirmationByConversation: {
          ...s.confirmationByConversation,
          [conversationId]: {
            ...current,
            status: 'pending',
            error: safeMessage,
          },
        },
      }))
      throw new Error(safeMessage)
    }
  },

  clearConfirmation: (conversationId: string) => {
    set((s) => ({
      confirmationByConversation: {
        ...s.confirmationByConversation,
        [conversationId]: null,
      },
    }))
  },
}))
