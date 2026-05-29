import type { Conversation, Message, Agent } from '../types'
import type { AgentName } from '../lib/agents'
import { DEFAULT_AGENT_NAME, normalizeAgentName } from '../lib/agents'

const API_BASE = '/api'
const DEFAULT_USER_ID = 'demo-user'

const token = () => import.meta.env.VITE_AGENTHUB_API_TOKEN as string

function authHeaders(extra?: Record<string, string>): Record<string, string> {
  const h: Record<string, string> = { ...extra }
  const t = token()
  if (t) h['Authorization'] = `Bearer ${t}`
  return h
}

function isObject(value: unknown): value is Record<string, unknown> {
  return !!value && typeof value === 'object' && !Array.isArray(value)
}

function pickString(value: unknown): string {
  if (typeof value !== 'string') {
    return ''
  }
  return value.trim()
}

function extractArray(payload: unknown): unknown[] {
  if (Array.isArray(payload)) {
    return payload
  }
  if (isObject(payload) && Array.isArray(payload.data)) {
    return payload.data
  }
  return []
}

function normalizeConversation(raw: unknown): Conversation {
  const source = isObject(raw) ? raw : {}
  const now = new Date().toISOString()
  const id = pickString(source.id)
  return {
    id: id || `conv-${Date.now()}`,
    title: pickString(source.title) || 'New Conversation',
    agentName: normalizeAgentName(pickString(source.agentName) || DEFAULT_AGENT_NAME),
    createdAt: pickString(source.createdAt) || now,
    updatedAt: pickString(source.updatedAt) || now,
  }
}

function normalizeAgent(raw: unknown): Agent | null {
  const source = isObject(raw) ? raw : {}
  const name = pickString(source.name)
  if (!name) {
    return null
  }
  return {
    name,
    displayName: pickString(source.displayName),
    description: pickString(source.description),
    outputModes: Array.isArray(source.outputModes)
      ? source.outputModes
          .map((mode) => pickString(mode))
          .filter((mode) => mode !== '')
      : [],
  }
}

export async function listConversations(): Promise<Conversation[]> {
  const res = await fetch(`${API_BASE}/conversations`, { headers: authHeaders() })
  if (!res.ok) throw new Error('Failed to load conversations')
  const payload = await res.json()
  return extractArray(payload).map(normalizeConversation)
}

export async function createConversation(agentName: AgentName): Promise<Conversation> {
  const res = await fetch(`${API_BASE}/conversations`, {
    method: 'POST',
    headers: authHeaders({ 'Content-Type': 'application/json' }),
    body: JSON.stringify({
      userId: DEFAULT_USER_ID,
      agentName: normalizeAgentName(agentName),
    }),
  })
  if (!res.ok) throw new Error('Failed to create conversation')
  return normalizeConversation(await res.json())
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
  const payload = await res.json()
  const agents = extractArray(payload).map(normalizeAgent).filter((item): item is Agent => item !== null)
  return agents
}
