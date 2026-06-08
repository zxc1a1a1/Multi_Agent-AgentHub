import type { PendingConfirmation } from '../stores/messageStore'
import type {
  PlanApprovalData,
  PlanApprovalTask,
  PlanApprovalParticipant,
} from '../types/planApproval'

/**
 * Normalize a PendingConfirmation (from messageStore confirm_plan parsing)
 * into PlanApprovalData for the PlanApprovalCard component.
 *
 * Applies fallbacks for all optional fields so the UI never receives
 * undefined/null values that could cause a render crash.
 */
export function normalizePlanApprovalData(
  confirmation: PendingConfirmation,
): PlanApprovalData {
  const tasks: PlanApprovalTask[] = (confirmation.tasks || []).map((t) => ({
    taskId: t.taskId || '',
    agentName: t.agentName,
    content: t.content || '',
    dependsOn: t.dependsOn,
    priority: t.priority,
    riskLevel: t.riskLevel,
  }))

  const participants = dedupeParticipants(buildParticipants(tasks, confirmation))

  const warnings: string[] = Array.isArray(confirmation.warnings)
    ? confirmation.warnings.filter((w): w is string => typeof w === 'string')
    : []

  const executionPath = normalizeExecutionPath(confirmation.executionPath)

  return {
    runId: confirmation.runId,
    actionId: confirmation.actionId,
    planId: confirmation.planId || confirmation.actionId,
    revision: confirmation.revision,
    executionPath,
    title: buildTitle(confirmation),
    summary: confirmation.intentSummary || '',
    planOwner: buildPlanOwner(confirmation),
    tasks,
    participants,
    strategy: confirmation.strategy,
    warnings,
  }
}

function normalizeExecutionPath(
  raw: unknown,
): PlanApprovalData['executionPath'] {
  if (typeof raw === 'string') {
    const valid = ['single_chat', 'group_chat', 'main_agent_orchestration']
    if (valid.includes(raw)) {
      return raw as PlanApprovalData['executionPath']
    }
  }
  return 'unknown'
}

function buildTitle(confirmation: PendingConfirmation): string {
  if (confirmation.intentSummary) {
    const truncated = confirmation.intentSummary.length > 80
      ? confirmation.intentSummary.slice(0, 80).trimEnd() + '...'
      : confirmation.intentSummary
    return truncated
  }
  return '确认执行方案'
}

function buildPlanOwner(
  confirmation: PendingConfirmation,
): PlanApprovalData['planOwner'] {
  if (
    confirmation.planOwner &&
    typeof confirmation.planOwner === 'object' &&
    !Array.isArray(confirmation.planOwner)
  ) {
    const po = confirmation.planOwner as Record<string, unknown>
    return {
      type: typeof po.type === 'string' ? po.type : undefined,
      agentName: typeof po.agentName === 'string' ? po.agentName : undefined,
      isMainAgent: typeof po.isMainAgent === 'boolean' ? po.isMainAgent : undefined,
    }
  }

  if (confirmation.agentNames.length === 1) {
    return {
      type: 'agent',
      agentName: confirmation.agentNames[0],
    }
  }

  return undefined
}

function buildParticipants(
  tasks: PlanApprovalTask[],
  confirmation: PendingConfirmation,
): PlanApprovalParticipant[] {
  if (
    Array.isArray(confirmation.participants) &&
    confirmation.participants.length > 0
  ) {
    return confirmation.participants.map((p: unknown) => {
      if (typeof p === 'object' && p !== null && !Array.isArray(p)) {
        const obj = p as Record<string, unknown>
        return {
          agentName: typeof obj.agentName === 'string' ? obj.agentName : '',
          role: typeof obj.role === 'string' ? obj.role : undefined,
          required: typeof obj.required === 'boolean' ? obj.required : undefined,
          selected: typeof obj.selected === 'boolean' ? obj.selected : undefined,
        }
      }
      return { agentName: '' }
    }).filter((p) => p.agentName !== '')
  }

  return dedupeParticipants(
    tasks.map((t) => ({
      agentName: t.agentName || '',
    })),
  )
}

function dedupeParticipants(
  participants: PlanApprovalParticipant[],
): PlanApprovalParticipant[] {
  const seen = new Set<string>()
  const result: PlanApprovalParticipant[] = []
  for (const p of participants) {
    if (!p.agentName || seen.has(p.agentName)) continue
    seen.add(p.agentName)
    result.push(p)
  }
  return result
}
