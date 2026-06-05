import type { AgentName } from '../lib/agents'

export interface AGUIEvent {
  type: string
  id?: string
  messageId?: string
  runId?: string
  threadId?: string
  taskId?: string
  role?: string
  delta?: string
  text?: string
  content?: string
  author?: string
  senderName?: string
  agentName?: string
  // AG-UI v1.0 sender object
  sender?: {
    type?: string
    name?: string
    displayName?: string
  }
  // AG-UI v1.0 state object (replaces string-encoded stateDelta)
  state?: Record<string, unknown>
  toolCallId?: string
  toolName?: string
  toolCall?: {
    id?: string
    name?: string
    arguments?: unknown
  }
  artifact?: {
    type?: string
    title?: string
    content?: string
    metadata?: Record<string, string>
  }
  stateDelta?: Record<string, unknown>
  metadata?: Record<string, unknown>
  error?:
    | string
    | {
        code?: string
        message?: string
      }
  final?: boolean
  partial?: boolean
  timestamp?: string
  traceId?: string
}

export interface Conversation {
  id: string
  title: string
  agentName: string
  createdAt: string
  updatedAt: string
}

export interface Message {
  id: string
  conversationId: string
  senderType: 'user' | 'agent'
  senderName?: string
  agentName?: string
  content: string
  status: 'sending' | 'streaming' | 'sent' | 'failed'
  codeBlocks?: CodeBlock[]
  webPreviews?: WebPreviewBlock[]
  runId?: string
  stepId?: string
  sseMessageId?: string
  errorCode?: string
  errorMessage?: string
  createdAt: string
}

export interface CodeBlock {
  code: string
  language: string
  filename: string
}

export interface WebPreviewBlock {
  html: string
  title: string
  agentName?: AgentName
}

export interface Agent {
  name: string
  displayName?: string
  description?: string
  outputModes?: string[]
}
