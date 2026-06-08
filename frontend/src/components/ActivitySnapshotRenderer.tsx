import { useMemo } from 'react'
import { useActivityStore } from '../stores/activityStore'
import PlanApprovalCard, { type PlanApprovalStatus } from './PlanApprovalCard'
import GroupPlanCard from './GroupPlanCard'
import MainAgentPlanCard from './MainAgentPlanCard'
import { normalizeActivityToPlanData } from '../lib/activitySnapshot'

export type { PlanApprovalStatus } from './PlanApprovalCard'

export interface ActivitySnapshotRendererProps {
  conversationId: string
  status?: PlanApprovalStatus
  onApprove: (selectedParticipants?: string[]) => void | Promise<void>
  onCancel: () => void | Promise<void>
  onRevise?: (feedback: string, selectedParticipants?: string[]) => void | Promise<void>
  onTimeout?: (actionId: string) => void
}

export default function ActivitySnapshotRenderer({
  conversationId,
  status = 'waiting',
  onApprove,
  onCancel,
  onRevise,
}: ActivitySnapshotRendererProps) {
  const activity = useActivityStore(
    (s) => s.pendingActivityByConversation[conversationId],
  )

  const planData = useMemo(() => {
    if (!activity) return null
    return normalizeActivityToPlanData(activity, '')
  }, [activity])

  if (!activity || !planData || activity.activityType !== 'plan_approval') {
    return null
  }

  const executionPath = activity.executionPath || 'unknown'

  switch (executionPath) {
    case 'main_agent_orchestration': {
      return (
        <MainAgentPlanCard
          data={planData}
          candidateParticipants={activity.candidateParticipants || []}
          defaultSelectedParticipants={activity.defaultSelectedParticipants || []}
          requiredParticipants={activity.requiredParticipants || []}
          status={status}
          onApprove={(selected) => onApprove(selected)}
          onCancel={onCancel}
          onRevise={
            onRevise
              ? (feedback, selected) => onRevise(feedback, selected)
              : undefined
          }
        />
      )
    }

    case 'group_chat': {
      return (
        <GroupPlanCard
          data={planData}
          status={status}
          onApprove={() => onApprove()}
          onCancel={onCancel}
          onRevise={
            onRevise
              ? (feedback) => onRevise(feedback)
              : undefined
          }
        />
      )
    }

    case 'single_chat':
    default: {
      return (
        <PlanApprovalCard
          data={planData}
          status={status}
          onApprove={() => onApprove()}
          onCancel={onCancel}
          onRevise={
            onRevise
              ? (feedback) => onRevise(feedback)
              : undefined
          }
        />
      )
    }
  }
}
