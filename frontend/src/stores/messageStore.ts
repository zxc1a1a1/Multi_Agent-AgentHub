import { create } from 'zustand'
import type { Message, AGUIEvent, CodeBlock } from '../types'
import * as api from '../services/api'
import { runAgent, type AGUIRunRequest } from '../agui/client'
import { frontendSkills } from '../agui/skills'

interface MessageState {
  messages: Record<string, Message[]> // conversationId → messages[]
  streaming: boolean
  abortController: AbortController | null

  loadMessages: (conversationId: string) => Promise<void>
  sendMessage: (conversationId: string, content: string) => void
  stopStreaming: () => void
}

export const useMessageStore = create<MessageState>((set, get) => ({
  messages: {},
  streaming: false,
  abortController: null,

  loadMessages: async (conversationId: string) => {
    try {
      const msgs = await api.listMessages(conversationId)
      // Transform DB messages to frontend format
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
    // 1. Optimistically add user message
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
      streaming: true,
    }))

    // 2. Prepare AG-UI run request
    const runId = `run-${Date.now()}`
    const request: AGUIRunRequest = {
      threadId: conversationId,
      runId,
      messages: [{ role: 'user', content }],
      tools: frontendSkills.map((name) => ({ name })),
    }

    // 3. Track streaming state
    let agentMsgId = ''
    let agentContent = ''
    let codeBlocks: CodeBlock[] = []
    const toolCallArgs: Record<string, string> = {} // toolCallId → accumulated args JSON

    // 4. Start the SSE stream
    const controller = runAgent(
      request,
      // onEvent handler
      (event: AGUIEvent) => {
        switch (event.type) {
          case 'TEXT_MESSAGE_START':
            agentMsgId = event.messageId || `agent-${Date.now()}`
            agentContent = ''
            codeBlocks = []
            // Add empty agent message placeholder
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
            // Update the streaming message content
            set((s) => ({
              messages: {
                ...s.messages,
                [conversationId]: (s.messages[conversationId] || []).map((m) =>
                  m.id === agentMsgId ? { ...m, content: agentContent } : m
                ),
              },
            }))
            break

          case 'TEXT_MESSAGE_END':
            // Mark message as complete
            set((s) => ({
              messages: {
                ...s.messages,
                [conversationId]: (s.messages[conversationId] || []).map((m) =>
                  m.id === agentMsgId ? { ...m, status: 'sent' as const } : m
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
                // Update message with code blocks
                set((s) => ({
                  messages: {
                    ...s.messages,
                    [conversationId]: (s.messages[conversationId] || []).map((m) =>
                      m.id === agentMsgId
                        ? { ...m, codeBlocks: [...codeBlocks] }
                        : m
                    ),
                  },
                }))
              } catch {
                // ignore parse errors
              }
            }
            break

          case 'RUN_FINISHED':
            set({ streaming: false, abortController: null })
            break

          case 'RUN_ERROR':
            set((s) => ({
              streaming: false,
              abortController: null,
              messages: {
                ...s.messages,
                [conversationId]: (s.messages[conversationId] || []).map((m) =>
                  m.id === agentMsgId
                    ? { ...m, status: 'failed' as const, content: agentContent || event.error || 'Error' }
                    : m
                ),
              },
            }))
            break
        }
      },
      // onError handler
      () => {
        set({ streaming: false, abortController: null })
      },
      // onComplete handler
      () => {
        set({ streaming: false, abortController: null })
      },
    )

    set({ abortController: controller })
  },

  stopStreaming: () => {
    const { abortController } = get()
    if (abortController) {
      abortController.abort()
      set({ streaming: false, abortController: null })
    }
  },
}))

// Helper to parse artifacts JSON string from DB into CodeBlock array
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
