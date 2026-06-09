import type { AgentName } from '../lib/agents'

// ActivitySnapshot types for ACTIVITY_SNAPSHOT AG-UI events (v1.3).
export interface ActivityPlanOwner {
  type: string // "agent" | "group_coordinator" | "main_agent"
  agentName: string
  isMainAgent?: boolean
}

export interface ActivityExecutionOwner {
  type: string // "agent" | "group" | "main_agent_orchestration"
  agentName?: string
  agentNames?: string[]
  selectedParticipants?: string[]
}

export interface ActivityParticipant {
  agentName: string
  required?: boolean
  selected?: boolean
}

export interface ActivityTask {
  taskId: string
  agentName: string
  content?: string
  priority?: number
  riskLevel?: string
}

export interface ActivitySnapshot {
  activityId: string
  activityType: string // "plan_approval" | "agent_turn" | "orchestration_progress"
  status: string // "awaiting_confirmation" | "revising_plan" | "executing" | "completed" | "cancelled" | "failed"
  executionPath: string // "single_chat" | "group_chat" | "main_agent_orchestration"
  planId: string
  revision: number
  planOwner?: ActivityPlanOwner
  executionOwner?: ActivityExecutionOwner
  participants: ActivityParticipant[]
  candidateParticipants?: ActivityParticipant[]
  defaultSelectedParticipants?: ActivityParticipant[]
  requiredParticipants?: string[]
  title?: string
  summary?: string
  tasks: ActivityTask[]
  allowedActions: string[]
  warnings?: string[]
}

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
  turnIndex?: number
  stepId?: string
  status?: string
  summary?: string
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
  // HITL confirmation fields
  actionId?: string
  actionName?: string
  riskLevel?: 'low' | 'medium' | 'high'
  description?: string
  parameters?: Record<string, unknown>
  timeoutMs?: number
  confirmed?: boolean
  rejectReason?: string
  // Snapshot rendering
  snapshotType?: string
  payload?: Record<string, unknown>
  // ACTIVITY_SNAPSHOT v1.3
  activity?: ActivitySnapshot
}

/**
 * HITL confirmation request sent from Frontend to Gateway.
 */
export interface HITLConfirmRequest {
  runId: string
  actionId: string
  planId?: string
  confirmed?: boolean
  action?: string
  feedback?: string
  revision?: number
  rejectReason?: string
  idempotencyKey?: string
  selectedParticipants?: string[]
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
  turnIndex?: number
  summary?: string
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
