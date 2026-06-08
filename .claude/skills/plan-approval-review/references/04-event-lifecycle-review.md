# 04 事件生命周期审核

## SSE 事件序列审核检查

### PLAN_PROPOSAL 序列

```text
必须出现的 SSE 事件（按顺序）:
  RUN_STARTED    { state: { phase: "planning", executionPath: "..." } }
  STATE_UPDATE   { state: { phase: "planning", ... } }
  TOOL_CALL_START   { toolCallName: "confirm_plan", toolCallId: <planId> }
  TOOL_CALL_ARGS    { delta: JSON plan args }
  TOOL_CALL_END     { toolCallId: <planId> }
  STATE_UPDATE   { state: { phase: "waiting_user_approval", requiresConfirmation: true } }

禁止出现的:
  RUN_ERROR      在 PLAN_PROPOSAL 阶段
  RUN_FINISHED   在用户操作之前
  Agent 内容的 TOOL_CALL_START/ARGS/END  （plan 内容不应与 Agent 执行内容混在一起）
```

### APPROVE_PLAN 序列

```text
必须出现的 SSE 事件（按顺序）:
  STATE_UPDATE   { phase: "executing", requiresConfirmation: false }
  (Agent 执行事件，与原有的 TOOL_CALL_START/CONTENT/END 一致)
  RUN_FINISHED   { status: "completed" }

禁止出现的:
  RUN_ERROR      除非执行真正失败
  STATE_UPDATE   phase != "executing"
  新的 RUN_STARTED
```

### REQUEST_PLAN_REVISION 序列

```text
必须出现的 SSE 事件（按顺序）:
  STATE_UPDATE   { phase: "revising_plan" }
  (planOwner 重新生成 plan)
  TOOL_CALL_START   { toolCallName: "confirm_plan", toolCallId: <new-planId> }
  TOOL_CALL_ARGS    { delta: 新 plan (revision+1) }
  TOOL_CALL_END     { toolCallId: <new-planId> }
  STATE_UPDATE   { phase: "waiting_user_approval" }

审核要点:
  - new-planId != 原 planId
  - revision = 原 revision + 1（单调递增）
  - 没有重复的 RUN_STARTED
  - 在同一个 SSE 连接上
```

### CANCEL_PLAN 序列

```text
必须出现的 SSE 事件（按顺序）:
  STATE_UPDATE   { phase: "cancelled" }
  RUN_FINISHED   { status: "cancelled" }

禁止出现的:
  RUN_ERROR      取消不是错误，不得 emit RUN_ERROR
  STATE_UPDATE   phase != "cancelled"
```

## confirm endpoint 审核

```text
POST /api/runs/{runId}/confirm

审核要点:
  - 返回 Content-Type: application/json（不是 text/event-stream）
  - 返回 { status: "acknowledged", planId, revision }
  - action 通过内存 channel 通知 SSE goroutine，不创建新 HTTP 连接
  - 复用已有 SSE 流继续输出
```

## idempotencyKey 审核

```text
审核要点:
  - Phase 2: 内存存储
  - 相同 key + 相同 payload → 返回第一次的结果（幂等）
  - 相同 key + 不同 payload → 409 Conflict
  - 不同 key + 无效状态 → 409 Conflict（如重复 approve）
  - key 与 runId 关联
```

## runId / planId / revision 校验

```text
审核要点:
  - runId 必须与当前 active run 匹配
  - planId 必须与当前 active PendingPlan 匹配
  - revision 必须与当前 active PendingPlan 匹配（approve 时）
  - 不匹配 → 409 Conflict
```

## AGENT_TURN 事件审核（group_chat，Phase 5）

```text
必须出现的事件:
  AGENT_TURN_STARTED   { agentName, turnIndex }
  (Agent 执行事件)
  AGENT_TURN_FINISHED  { agentName, turnIndex }

审核要点:
  - turnIndex 单调递增
  - 同一时间只有一个 Agent 的 turn 活跃
  - CONENT 改为 CONTENT（修正拼写错误）
```


## Auto participant-change lifecycle

```text
User changes Agent selection in main_agent_orchestration
  → POST /api/runs/{runId}/confirm action=revise selectedParticipants=[...] feedback?
  → STATE_UPDATE { phase: "revising_plan" }
  → new confirm_plan sequence with revision+1
  → STATE_UPDATE { phase: "waiting_user_approval" }

APPROVE_PLAN with changed selectedParticipants
  → 409 PARTICIPANT_CHANGE_REQUIRES_REVISION
  → no Agent dispatch
```
