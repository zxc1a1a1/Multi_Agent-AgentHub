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
  pinned?: boolean
  pinnedAt?: string
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
  skillCards?: SkillCardData[]
  orchestrationSummary?: OrchestrationSummaryData
  replyTo?: ReplyTo
  quote?: Quote
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

/**
 * Skill card data produced by tool calls during streaming.
 * Each tool maps to a specific card type for lightweight display.
 */
export interface TerminalOutputData {
  type: 'terminal_output'
  command?: string
  stdout?: string
  stderr?: string
  exitCode?: number
}

export interface DiffPreviewData {
  type: 'diff_preview'
  diffText: string
  filename?: string
}

export interface DeployStatusData {
  type: 'deploy_status'
  environment?: string
  version?: string
  status?: string
  timestamp?: string
}

export interface ChartRenderData {
  type: 'chart_render'
  chartType?: 'bar' | 'line' | 'pie'
  data?: Array<{ label?: string; value: number }>
}

export interface FileDownloadData {
  type: 'file_download'
  filename?: string
  size?: number | string
  mimeType?: string
  createdAt?: string
}

export interface ImagePreviewData {
  type: 'image_preview'
  url?: string
  alt?: string
}

export interface UnknownSkillData {
  type: 'unknown_skill'
  toolName?: string
  args?: unknown
}

export type SkillCardData =
  | TerminalOutputData
  | DiffPreviewData
  | DeployStatusData
  | ChartRenderData
  | FileDownloadData
  | ImagePreviewData
  | UnknownSkillData

/**
 * Post-run orchestration summary data.
 */
export interface OrchestrationSummaryData {
  agents?: string[]
  tasks?: Array<{
    agentName: string
    taskId?: string
    status: 'completed' | 'failed' | 'running' | 'pending' | 'skipped'
    content?: string
    duration?: string
  }>
  runStatus?: string
  duration?: string
  artifactCount?: number
}

/**
 * Reply-to reference for threaded/IM-style replies.
 */
export interface ReplyTo {
  id: string
  author: string
  senderType: 'user' | 'agent' | 'system'
  contentPreview: string // first ~100 chars of replied-to message
}

/**
 * Text quote selection from a message.
 */
export interface Quote {
  messageId: string
  author: string
  text: string // the selected/quoted text
  startOffset?: number // character offset in source message
  endOffset?: number
}

/**
 * AgentManagement extends Agent with orchestration-level metadata
 * returned by the agent management API (GET /api/agents).
 * Used by the Agent Directory page.
 */
export interface AgentManagement {
  name: string
  displayName?: string
  description?: string
  outputModes?: string[]
  source: 'static' | 'dynamic'
  enabled: boolean
  healthy: boolean
  capabilities?: string[]
  lastError?: string
  baseURL?: string
  createdAt?: string
  updatedAt?: string
}

/**
 * Request body for registering a new dynamic agent.
 */
export interface RegisterAgentRequest {
  name: string
  url: string
  replace?: boolean
}
