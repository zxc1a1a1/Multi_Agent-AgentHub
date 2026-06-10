import type { Conversation, Message, Agent, AgentManagement, RegisterAgentRequest, HITLConfirmRequest } from '../types'
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
    pinned: typeof source.pinned === 'boolean' ? source.pinned : undefined,
    pinnedAt: pickString(source.pinnedAt) || undefined,
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

function normalizeAgentManagement(raw: unknown): AgentManagement | null {
  const source = isObject(raw) ? raw : {}
  const name = pickString(source.name)
  if (!name) {
    return null
  }
  const sourceField = pickString(source.source)
  return {
    name,
    displayName: pickString(source.displayName),
    description: pickString(source.description),
    outputModes: Array.isArray(source.outputModes)
      ? source.outputModes
          .map((mode) => pickString(mode))
          .filter((mode) => mode !== '')
      : [],
    source: sourceField === 'dynamic' ? 'dynamic' : 'static',
    enabled: typeof source.enabled === 'boolean' ? source.enabled : true,
    healthy: typeof source.healthy === 'boolean' ? source.healthy : false,
    capabilities: Array.isArray(source.capabilities)
      ? source.capabilities.map((c: unknown) => pickString(c)).filter((c: string) => c !== '')
      : [],
    lastError: pickString(source.lastError) || undefined,
    baseURL: pickString(source.baseURL) || pickString(source.url) || undefined,
    createdAt: pickString(source.createdAt) || undefined,
    updatedAt: pickString(source.updatedAt) || undefined,
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

export async function deleteConversation(conversationId: string): Promise<void> {
  const res = await fetch(`${API_BASE}/conversations/${conversationId}`, {
    method: 'DELETE',
    headers: authHeaders(),
  })
  if (!res.ok) {
    if (res.status === 404) {
      throw new Error('Conversation not found')
    }
    throw new Error('Failed to delete conversation')
  }
}

/**
 * Pin or unpin a conversation.
 */
export async function pinConversation(id: string, pinned: boolean): Promise<Conversation> {
  const res = await fetch(`${API_BASE}/conversations/${id}/pin`, {
    method: 'PATCH',
    headers: authHeaders({ 'Content-Type': 'application/json' }),
    body: JSON.stringify({ pinned }),
  })
  if (!res.ok) {
    if (res.status === 404) throw new Error('Conversation not found')
    throw new Error(`Failed to ${pinned ? 'pin' : 'unpin'} conversation`)
  }
  return normalizeConversation(await res.json())
}

/**
 * Cancel an active run via the Gateway.
 */
export async function cancelRun(runId: string): Promise<void> {
  const res = await fetch(`${API_BASE}/runs/${runId}/cancel`, {
    method: 'POST',
    headers: authHeaders(),
  })
  if (!res.ok) {
    throw new Error(`Failed to cancel run: HTTP ${res.status}`)
  }
}

/**
 * Send a tool result back to the Gateway/Orchestrator for an active tool call
 * that requires human input (confirmation, form, file selection, etc.).
 */
export async function sendToolResult(request: {
  runId: string
  taskId?: string
  toolCallId: string
  status: 'success' | 'cancelled' | 'failed'
  contentType?: string
  data?: unknown
  error?: { code: string; message: string }
}): Promise<void> {
  const res = await fetch(`${API_BASE}/runs/${request.runId}/tool-result`, {
    method: 'POST',
    headers: authHeaders({ 'Content-Type': 'application/json' }),
    body: JSON.stringify(request),
  })
  if (!res.ok) {
    let message = `Failed to send tool result: HTTP ${res.status}`
    try {
      const body = await res.json()
      if (body.error) message = body.error
    } catch { /* fall through */ }
    throw new Error(message)
  }
}

/**
 * Send a HITL confirmation response to the Gateway.
 */
export async function confirmHITL(request: HITLConfirmRequest): Promise<void> {
  const res = await fetch(`${API_BASE}/runs/${request.runId}/confirm`, {
    method: 'POST',
    headers: authHeaders({ 'Content-Type': 'application/json' }),
    body: JSON.stringify(request),
  })
  if (!res.ok) {
    let errorCode = `HTTP ${res.status}`
    try {
      const body = await res.json()
      if (body.error) {
        errorCode = body.error
      } else if (body.code) {
        errorCode = body.code
      }
    } catch {
      // If JSON parsing fails, fall back to HTTP status code.
    }
    throw new Error(errorCode)
  }
}

// ---------------------------------------------------------------------------
// Agent Management API
// ---------------------------------------------------------------------------

/**
 * List all agents with management metadata (source, enabled, healthy, etc.).
 * Returns richer AgentManagement objects compared to listAgents().
 */
export async function listAgentsManagement(): Promise<AgentManagement[]> {
  const res = await fetch(`${API_BASE}/agents`, { headers: authHeaders() })
  if (!res.ok) throw new Error('Failed to load agents')
  const payload = await res.json()
  return extractArray(payload)
    .map(normalizeAgentManagement)
    .filter((item): item is AgentManagement => item !== null)
}

/**
 * Get a single agent by name with full management metadata.
 */
export async function getAgent(name: string): Promise<AgentManagement> {
  const res = await fetch(`${API_BASE}/agents/${encodeURIComponent(name)}`, {
    headers: authHeaders(),
  })
  if (!res.ok) {
    if (res.status === 404) throw new Error('Agent not found')
    throw new Error(`Failed to get agent: HTTP ${res.status}`)
  }
  const agent = normalizeAgentManagement(await res.json())
  if (!agent) throw new Error('Invalid agent response')
  return agent
}

/**
 * Register a new dynamic agent.
 */
export async function registerAgent(req: RegisterAgentRequest): Promise<AgentManagement> {
  const res = await fetch(`${API_BASE}/agents`, {
    method: 'POST',
    headers: authHeaders({ 'Content-Type': 'application/json' }),
    body: JSON.stringify(req),
  })
  if (!res.ok) {
    let message = `Failed to register agent: HTTP ${res.status}`
    try {
      const body = await res.json()
      if (body.error) message = body.error
    } catch { /* fall through */ }
    throw new Error(message)
  }
  const agent = normalizeAgentManagement(await res.json())
  if (!agent) throw new Error('Invalid agent response')
  return agent
}

/**
 * Update a dynamic agent's display name or URL.
 */
export async function updateAgent(
  name: string,
  patch: { displayName?: string; url?: string },
): Promise<AgentManagement> {
  const res = await fetch(`${API_BASE}/agents/${encodeURIComponent(name)}`, {
    method: 'PATCH',
    headers: authHeaders({ 'Content-Type': 'application/json' }),
    body: JSON.stringify(patch),
  })
  if (!res.ok) {
    let message = `Failed to update agent: HTTP ${res.status}`
    try {
      const body = await res.json()
      if (body.error) message = body.error
    } catch { /* fall through */ }
    throw new Error(message)
  }
  const agent = normalizeAgentManagement(await res.json())
  if (!agent) throw new Error('Invalid agent response')
  return agent
}

/**
 * Delete (unregister) a dynamic agent.
 */
export async function deleteAgent(name: string): Promise<void> {
  const res = await fetch(`${API_BASE}/agents/${encodeURIComponent(name)}`, {
    method: 'DELETE',
    headers: authHeaders(),
  })
  if (!res.ok) {
    let message = `Failed to delete agent: HTTP ${res.status}`
    try {
      const body = await res.json()
      if (body.error) message = body.error
    } catch { /* fall through */ }
    throw new Error(message)
  }
}

/**
 * Enable or disable a dynamic agent.
 */
export async function setAgentEnabled(name: string, enabled: boolean): Promise<AgentManagement> {
  const action = enabled ? 'enable' : 'disable'
  const res = await fetch(`${API_BASE}/agents/${encodeURIComponent(name)}/${action}`, {
    method: 'POST',
    headers: authHeaders(),
  })
  if (!res.ok) {
    let message = `Failed to ${action} agent: HTTP ${res.status}`
    try {
      const body = await res.json()
      if (body.error) message = body.error
    } catch { /* fall through */ }
    throw new Error(message)
  }
  const agent = normalizeAgentManagement(await res.json())
  if (!agent) throw new Error('Invalid agent response')
  return agent
}

/**
 * Refresh a dynamic agent's card from its upstream URL.
 */
export async function refreshAgent(name: string): Promise<AgentManagement> {
  const res = await fetch(`${API_BASE}/agents/${encodeURIComponent(name)}/refresh`, {
    method: 'POST',
    headers: authHeaders(),
  })
  if (!res.ok) {
    let message = `Failed to refresh agent: HTTP ${res.status}`
    try {
      const body = await res.json()
      if (body.error) message = body.error
    } catch { /* fall through */ }
    throw new Error(message)
  }
  const agent = normalizeAgentManagement(await res.json())
  if (!agent) throw new Error('Invalid agent response')
  return agent
}

/**
 * Run a health check on an agent.
 */
export async function checkAgent(name: string): Promise<AgentManagement> {
  const res = await fetch(`${API_BASE}/agents/${encodeURIComponent(name)}/check`, {
    method: 'POST',
    headers: authHeaders(),
  })
  if (!res.ok) {
    let message = `Failed to check agent: HTTP ${res.status}`
    try {
      const body = await res.json()
      if (body.error) message = body.error
    } catch { /* fall through */ }
    throw new Error(message)
  }
  const agent = normalizeAgentManagement(await res.json())
  if (!agent) throw new Error('Invalid agent response')
  return agent
}
