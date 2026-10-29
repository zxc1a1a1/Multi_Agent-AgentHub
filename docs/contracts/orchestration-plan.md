# Orchestration Plan Contract

**Status:** Active (v2)
**Owner:** AgentHub
**Last updated:** 2026-10-29（P0：删除规则回退计划，冻结 revision/版本与拒绝语义）

## Authoritative source order

1. `docs/contracts/intent-orchestration.md`
2. `docs/contracts/participant-boundary.md`
3. `docs/contracts/runtime-harness.md`

## Shape

```json
{
  "version": "1.0",
  "planId": "plan_...",
  "runId": "run_...",
  "conversationId": "conv_...",
  "executionPath": "single_chat|group_chat|main_agent_orchestration",
  "executionOwner": {"type": "agent|group|main_agent_orchestration", "agentName": "...", "agentNames": []},
  "strategy": "single|sequential|ordered_parallel|conversational",
  "intentSummary": "brief user intent",
  "revision": 1,
  "plannerSource": "main_agent",
  "plannerModel": "provider/model",
  "tasks": [
    {
      "taskId": "task-1",
      "agentName": "code-agent",
      "taskContent": "...",
      "dependsOn": [],
      "priority": 1,
      "timeoutMs": 60000,
      "riskLevel": "low|medium|high"
    }
  ],
  "allowedAgents": ["code-agent"],
  "participants": [],
  "candidateParticipants": [],
  "defaultSelectedParticipants": [],
  "fallback": {"enabled": false}
}
```

## Frozen rules

1. `plannerSource` 生产只允许 `main_agent`；`fallback` 必须恒为 `{"enabled": false}`。任何 `fallback.enabled=true` 的计划在目标启动图视为非法。
2. `revision` 从 1 开始单调递增；计划修订产生新 revision，旧审批/旧工具许可失效。
3. `planId` 是运行内稳定标识；同一 revision 不得复用不同内容。
4. 计划必须通过依赖无环校验（Kahn）与允许集合校验后才能执行。
5. 任务数量默认上限 5（`ORCHESTRATOR_MAX_TASKS`）；并行度受 `runtime-harness.md` 预算约束。
6. 模型返回未知 Agent、未知策略或非法依赖时，整个计划被拒绝；不猜测、不补默认任务、不固定派发 `code-agent`。
7. `conversational` 仅表示当前轮不派发子任务（直接回答/澄清），不是规则路由结果。

## Sequential (DAG) semantics

- Tasks declare upstream dependencies via `dependsOn` (array of `taskId`).
- The dependency graph MUST be acyclic (validated by Kahn's algorithm).
- Upstream task output is injected into downstream `taskContent` via `{{deps.{taskId}.output}}` placeholders.
- If an upstream task fails, dependent downstream tasks are skipped with `ORCHESTRATOR_SKIPPED_DEP`.
- Independent branches continue execution regardless of failures in other branches.

## Validation

- Every `agentName` must exist in Orchestrator registry, be enabled/healthy, and be inside `AllowedAgents`.
- `dependsOn` ids must reference other taskIds in the plan (no dangling refs, no self-loops).
- `ordered_parallel` tasks must not have `dependsOn`.
- The `dependsOn` graph must contain no cycles.
- Frontend skills must support expected artifact outputs or Orchestrator must degrade to text.
- 审批改变参与者集合必须走 revision，不得在批准动作中静默改写计划。
