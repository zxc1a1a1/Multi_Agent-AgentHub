export interface PlanApprovalTask {
  taskId?: string
  agentName?: string
  content: string
  dependsOn?: string[]
  priority?: number
  riskLevel?: string
}

export interface PlanApprovalParticipant {
  agentName: string
  role?: string
  required?: boolean
  selected?: boolean
}

export interface PlanApprovalData {
  runId: string
  actionId?: string
  planId?: string
  revision?: number

  executionPath?: 'single_chat' | 'group_chat' | 'main_agent_orchestration' | 'legacy' | 'unknown'

  title: string
  summary?: string

  planOwner?: {
    type?: string
    agentName?: string
    isMainAgent?: boolean
  }

  tasks: PlanApprovalTask[]

  participants: PlanApprovalParticipant[]

  strategy?: string

  warnings: string[]
}
