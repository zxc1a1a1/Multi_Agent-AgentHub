import type { Conversation, Message, Agent } from '../types'

const API_BASE = '/api'

const token = () => import.meta.env.VITE_AGENTHUB_API_TOKEN as string

function authHeaders(extra?: Record<string, string>): Record<string, string> {
  const h: Record<string, string> = { ...extra }
  const t = token()
  if (t) h['Authorization'] = `Bearer ${t}`
  return h
}

export async function listConversations(): Promise<Conversation[]> {
  const res = await fetch(`${API_BASE}/conversations`, { headers: authHeaders() })
  if (!res.ok) throw new Error('Failed to load conversations')
  return res.json()
}

export async function createConversation(agentName: string, title?: string): Promise<Conversation> {
  const res = await fetch(`${API_BASE}/conversations`, {
    method: 'POST',
    headers: authHeaders({ 'Content-Type': 'application/json' }),
    body: JSON.stringify({ agentName, title: title || 'New Conversation' }),
  })
  if (!res.ok) throw new Error('Failed to create conversation')
  return res.json()
}

export async function listMessages(conversationId: string): Promise<Message[]> {
  const res = await fetch(`${API_BASE}/conversations/${conversationId}/messages`, {
    headers: authHeaders(),
  })
  if (!res.ok) throw new Error('Failed to load messages')
  return res.json()
}

export async function listAgents(): Promise<Agent[]> {
  const res = await fetch(`${API_BASE}/agents`, { headers: authHeaders() })
  if (!res.ok) throw new Error('Failed to load agents')
  return res.json()
}
