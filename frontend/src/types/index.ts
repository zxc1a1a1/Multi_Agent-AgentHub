export interface AGUIEvent {
  type: string
  messageId?: string
  runId?: string
  content?: string
  toolCallId?: string
  toolName?: string
  error?: string
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
  content: string
  status: 'sending' | 'streaming' | 'sent' | 'failed'
  codeBlocks?: CodeBlock[]
  createdAt: string
}

export interface CodeBlock {
  code: string
  language: string
  filename: string
}

export interface Agent {
  name: string
  description: string
}
