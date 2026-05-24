import { create } from 'zustand'
import type { Message, AGUIEvent, CodeBlock } from '../types'
import * as api from '../services/api'
import { runAgent, type AGUIRunRequest } from '../agui/client'
import { frontendSkills } from '../agui/skills'

interface MessageState {
  messages: Record<string, Message[]> // conversationId -> messages[]
  streamingByConversation: Record<string, boolean>
  abortControllersByConversation: Record<string, AbortController | null>

  loadMessages: (conversationId: string) => Promise<void>
  sendMessage: (conversationId: string, content: string) => void
  isStreaming: (conversationId: string) => boolean
  stopStreaming: (conversationId: string) => void
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

export const useMessageStore = create<MessageState>((set, get) => ({
  messages: {},
  streamingByConversation: {},
  abortControllersByConversation: {},

  loadMessages: async (conversationId: string) => {
    try {
      const msgs = await api.listMessages(conversationId)
      const formatted: Message[] = msgs.map((m: any) => ({
        id: m.id,
        conversationId: m.conversationId,
        senderType: m.senderType as 'user' | 'agent',
        content: m.content,
        status: 'sent' as const,
        codeBlocks: m.artifacts ? parseArtifactsToCodeBlocks(m.artifacts) : undefined,
        createdAt: m.createdAt,
      }))
      set((s) => ({
        messages: { ...s.messages, [conversationId]: formatted },
      }))
    } catch {
      // ignore load errors silently
    }
  },

  sendMessage: (conversationId: string, content: string) => {
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

    const runId = `run-${Date.now()}`
    const request: AGUIRunRequest = {
      threadId: conversationId,
      runId,
      messages: [{ role: 'user', content }],
      tools: frontendSkills.map((name) => ({ name })),
    }

    let agentMsgId = ''
    let agentContent = ''
    let codeBlocks: CodeBlock[] = []
    const toolCallArgs: Record<string, string> = {}

    const controller = runAgent(
      request,
      (event: AGUIEvent) => {
        switch (event.type) {
          case 'TEXT_MESSAGE_START':
            agentMsgId = event.messageId || `agent-${Date.now()}`
            agentContent = ''
            codeBlocks = []
            set((s) => ({
              messages: {
                ...s.messages,
                [conversationId]: [
                  ...(s.messages[conversationId] || []),
                  {
                    id: agentMsgId,
                    conversationId,
                    senderType: 'agent' as const,
                    content: '',
                    status: 'streaming' as const,
                    createdAt: new Date().toISOString(),
                  },
                ],
              },
            }))
            break

          case 'TEXT_MESSAGE_CONTENT':
            agentContent += event.content || ''
            set((s) => ({
              messages: {
                ...s.messages,
                [conversationId]: (s.messages[conversationId] || []).map((m) =>
                  m.id === agentMsgId ? { ...m, content: agentContent } : m,
                ),
              },
            }))
            break

          case 'TEXT_MESSAGE_END':
            set((s) => ({
              messages: {
                ...s.messages,
                [conversationId]: (s.messages[conversationId] || []).map((m) =>
                  m.id === agentMsgId ? { ...m, status: 'sent' as const } : m,
                ),
              },
            }))
            break

          case 'TOOL_CALL_START':
            if (event.toolCallId) {
              toolCallArgs[event.toolCallId] = ''
            }
            break

          case 'TOOL_CALL_ARGS':
            if (event.toolCallId) {
              toolCallArgs[event.toolCallId] =
                (toolCallArgs[event.toolCallId] || '') + (event.content || '')
            }
            break

          case 'TOOL_CALL_END':
            if (event.toolCallId && event.toolName === 'code_preview') {
              try {
                const args = JSON.parse(toolCallArgs[event.toolCallId] || '{}')
                codeBlocks.push({
                  code: args.code || '',
                  language: args.language || '',
                  filename: args.filename || '',
                })
                set((s) => ({
                  messages: {
                    ...s.messages,
                    [conversationId]: (s.messages[conversationId] || []).map((m) =>
                      m.id === agentMsgId ? { ...m, codeBlocks: [...codeBlocks] } : m,
                    ),
                  },
                }))
              } catch {
                // ignore parse errors
              }
            }
            break

          case 'RUN_FINISHED':
            set((s) => ({
              ...setConversationStreaming(s, conversationId, false),
              ...setConversationAbortController(s, conversationId, null),
            }))
            break

          case 'RUN_ERROR':
            set((s) => ({
              ...setConversationStreaming(s, conversationId, false),
              ...setConversationAbortController(s, conversationId, null),
              messages: {
                ...s.messages,
                [conversationId]: (s.messages[conversationId] || []).map((m) =>
                  m.id === agentMsgId
                    ? {
                        ...m,
                        status: 'failed' as const,
                        content: agentContent || event.error || 'Error',
                      }
                    : m,
                ),
              },
            }))
            break
        }
      },
      () => {
        set((s) => ({
          ...setConversationStreaming(s, conversationId, false),
          ...setConversationAbortController(s, conversationId, null),
        }))
      },
      () => {
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

function parseArtifactsToCodeBlocks(artifactsStr: string): CodeBlock[] | undefined {
  if (!artifactsStr) return undefined
  try {
    const artifacts = JSON.parse(artifactsStr)
    if (!Array.isArray(artifacts)) return undefined
    return artifacts
      .filter((a: any) => a.type === 'code')
      .map((a: any) => ({
        code: a.content || '',
        language: a.metadata?.language || '',
        filename: a.title || '',
      }))
  } catch {
    return undefined
  }
}
