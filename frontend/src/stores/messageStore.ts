import { create } from 'zustand'
import type { Message, AGUIEvent, CodeBlock, WebPreviewBlock } from '../types'
import type { AgentName } from '../lib/agents'
import * as api from '../services/api'
import { runAgent, type AGUIChatRequest } from '../agui/client'
import { DEFAULT_AGENT_NAME, getAgentDisplayName, isSupportedAgentName, normalizeAgentName } from '../lib/agents'

interface SendMessageOptions {
  agentName?: AgentName
}

interface MessageState {
  messages: Record<string, Message[]> // conversationId -> messages[]
  streamingByConversation: Record<string, boolean>
  abortControllersByConversation: Record<string, AbortController | null>

  loadMessages: (conversationId: string) => Promise<void>
  sendMessage: (conversationId: string, content: string, options?: SendMessageOptions) => void
  isStreaming: (conversationId: string) => boolean
  stopStreaming: (conversationId: string) => void
}

interface StoredMessage {
  id: string
  conversationId?: string
  senderType?: string
  senderName?: string
  agentName?: string
  author?: string
  role?: string
  content?: string
  text?: string
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
      return errorText
    }
  }
  if (event.error && typeof event.error === 'object') {
    const message = pickText(event.error.message)
    if (message) {
      return message
    }
  }
  const fallback = pickText(event.text, event.content)
  return fallback || 'Error'
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

function maybeExtractHTMLSnippet(content: string): string | undefined {
  const text = content.trim()
  if (text === '') {
    return undefined
  }
  if (!isLikelyHTML(text)) {
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
          senderName: pickText(raw.senderName, raw.author),
          agentName: normalizedAgentName,
          content,
          status: 'sent',
          codeBlocks: parsedArtifacts.codeBlocks,
          webPreviews: parsedArtifacts.webPreviews,
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

    const request: AGUIChatRequest = {
      conversationId,
      message: content,
    }
    if (options?.agentName) {
      request.agentName = options.agentName
    }

    let agentMsgId = ''
    let agentContent = ''
    let currentAgentName: string = selectedAgentName
    let currentSenderName = fallbackSenderName
    let codeBlocks: CodeBlock[] = []
    let webPreviews: WebPreviewBlock[] = []
    const toolCallArgs: Record<string, string> = {}
    const toolCallNames: Record<string, string> = {}

    const resolveEventAgentName = (event: AGUIEvent): string => {
      const eventAgentName = pickText(
        event.agentName,
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
      const explicitSender = pickText(event.senderName)
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
      if (webPreviews.length > 0 || !isSupportedAgentName(currentAgentName)) {
        return
      }
      if (currentAgentName !== 'web-agent') {
        return
      }
      const htmlSnippet = maybeExtractHTMLSnippet(agentContent)
      if (!htmlSnippet) {
        return
      }
      webPreviews.push({
        html: htmlSnippet,
        title: 'web-preview.html',
        agentName: 'web-agent',
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
        appendWebPreview(args)
      }
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
          case 'message.end':
            finishStreamingMessage()
            break

          case 'TOOL_CALL_START':
            if (event.toolCallId) {
              toolCallArgs[event.toolCallId] = ''
              if (event.toolName) {
                toolCallNames[event.toolCallId] = event.toolName
              }
            }
            break

          case 'TOOL_CALL_ARGS':
            if (event.toolCallId) {
              toolCallArgs[event.toolCallId] =
                (toolCallArgs[event.toolCallId] || '') + resolveContentChunk(event)
            }
            break

          case 'TOOL_CALL_END':
            if (event.toolCallId) {
              const toolName = event.toolName || toolCallNames[event.toolCallId] || ''
              handleToolPayload(toolName, toolCallArgs[event.toolCallId] || '')
            }
            break

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
            failStreamingMessage(resolveErrorText(event))
            set((s) => ({
              ...setConversationStreaming(s, conversationId, false),
              ...setConversationAbortController(s, conversationId, null),
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
}))
