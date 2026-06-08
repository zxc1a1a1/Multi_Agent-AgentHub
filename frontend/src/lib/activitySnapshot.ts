import type { ActivitySnapshot } from '../types'
import type { PlanApprovalData, PlanApprovalTask, PlanApprovalParticipant } from '../types/planApproval'

export function normalizeActivityToPlanData(
  activity: ActivitySnapshot,
  runId: string,
): PlanApprovalData {
  const tasks: PlanApprovalTask[] = (activity.tasks || []).map((t) => ({
    taskId: t.taskId || '',
    agentName: t.agentName,
    content: t.content || '',
    priority: t.priority,
    riskLevel: t.riskLevel,
  }))

  const participants: PlanApprovalParticipant[] = dedupeParticipants(
    (activity.participants || []).map((p) => ({
      agentName: p.agentName,
      role: 'executor',
      required: p.required,
      selected: p.selected,
    })),
  )

  const executionPath = normalizeExecutionPath(activity.executionPath)

  return {
    runId,
    actionId: activity.planId,
    planId: activity.planId,
    revision: activity.revision,
    executionPath,
    title: activity.title || activity.summary || 'Execution Plan',
    summary: activity.summary || '',
    planOwner: activity.planOwner
      ? {
          type: activity.planOwner.type,
          agentName: activity.planOwner.agentName,
          isMainAgent: activity.planOwner.isMainAgent,
        }
      : undefined,
    tasks,
    participants,
    strategy: undefined,
    warnings: activity.warnings || [],
  }
}

function normalizeExecutionPath(
  raw: string,
): PlanApprovalData['executionPath'] {
  const valid = ['single_chat', 'group_chat', 'main_agent_orchestration']
  if (valid.includes(raw)) {
    return raw as PlanApprovalData['executionPath']
  }
  return 'unknown'
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
