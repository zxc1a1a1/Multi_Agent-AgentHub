import type { Conversation, Message, Agent } from '../types'

const API_BASE = '/api'

export async function listConversations(): Promise<Conversation[]> {
  const res = await fetch(`${API_BASE}/conversations`)
  if (!res.ok) throw new Error('Failed to load conversations')
  return res.json()
}

export async function createConversation(agentName: string, title?: string): Promise<Conversation> {
  const res = await fetch(`${API_BASE}/conversations`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ agentName, title: title || 'New Conversation' }),
  })
  if (!res.ok) throw new Error('Failed to create conversation')
  return res.json()
}

export async function listMessages(conversationId: string): Promise<Message[]> {
  const res = await fetch(`${API_BASE}/conversations/${conversationId}/messages`)
  if (!res.ok) throw new Error('Failed to load messages')
  return res.json()
}

export async function listAgents(): Promise<Agent[]> {
  const res = await fetch(`${API_BASE}/agents`)
  if (!res.ok) throw new Error('Failed to load agents')
  return res.json()
}
