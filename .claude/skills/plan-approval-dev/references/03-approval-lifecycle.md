# 03 Plan Approval 生命周期

## 状态机

```text
planning ──> waiting_user_approval ──┬──> executing ──> completed
                 │    ↑              │
                 │    │              ├──> cancelled
                 │    └── revise ────┘
                 │                   │
                 └──> expired        └──> failed
```

## 状态说明

```text
planning                Orchestrator / Agent / Main-agent 正在生成 plan
waiting_user_approval   PLAN_PROPOSAL 已发送，等待用户操作
revising_plan           用户请求修订，正在生成新 plan
executing               用户已 approve，正在执行
completed               执行成功完成
cancelled               用户主动取消
expired                 审批窗口过期，用户未操作
failed                  执行失败（系统错误）
```

## 生命周期约束

```text
- 一个 conversation 同时只能有一个 active PendingPlan。
- revision 从 1 开始，每次 REQUEST_PLAN_REVISION 后 +1。
- status 为 completed / cancelled / expired / failed 后不可再执行。
- expired 由 expiresAt 超时触发。
```

## SSE 事件序列

### PLAN_PROPOSAL 阶段

```text
RUN_STARTED    { state: { phase: "planning", executionPath: "..." } }
STATE_UPDATE   { state: { phase: "planning", ... } }
TOOL_CALL_START   { toolCallName: "confirm_plan", toolCallId: <planId> }
TOOL_CALL_ARGS    { delta: JSON plan args }
TOOL_CALL_END     { toolCallId: <planId> }
STATE_UPDATE   { state: { phase: "waiting_user_approval", requiresConfirmation: true } }
```

### APPROVE_PLAN 后

APPROVE_PLAN 不接受 participant changes。auto 下 selectedParticipants 省略时使用 defaultSelectedParticipants；若传入值不同，返回 PARTICIPANT_CHANGE_REQUIRES_REVISION。

```text
STATE_UPDATE   { phase: "executing", requiresConfirmation: false }
(execution events...)
RUN_FINISHED   { status: "completed" }
```

### REQUEST_PLAN_REVISION 后

```text
STATE_UPDATE   { phase: "revising_plan" }
(planOwner 重新生成 plan，revision+1)
TOOL_CALL_START   { toolCallName: "confirm_plan", toolCallId: <new-planId> }
TOOL_CALL_ARGS    { delta: 新 plan args (revision+1) }
TOOL_CALL_END     { toolCallId: <new-planId> }
STATE_UPDATE   { phase: "waiting_user_approval" }
```

### CANCEL_PLAN 后

```text
STATE_UPDATE   { phase: "cancelled" }
RUN_FINISHED   { status: "cancelled" }
```

## Action 续流机制

```text
- POST /api/runs/{runId}/confirm 返回 JSON { status: "acknowledged", planId, revision }。
- action 信号通过内存 channel 通知正在阻塞的 handleRunStream goroutine。
- 原 /api/chat SSE 连接在 action 后继续输出事件。
- 不创建新的 HTTP 连接，不重新 POST /api/chat。
```

## REQUEST_PLAN_REVISION 分发

```text
- single_chat: 以 mode=plan_only 调用同一个 Agent，传入原 plan + feedback。
- group_chat: 调用 group coordinator，传入原 plan + participants + feedback。
- main_agent_orchestration: 调用 main-agent，传入原 plan + feedback + 用户请求的 participant changes，生成 revision+1 PLAN_PROPOSAL。
- 三种路径共有的: revision+1、新 planId、新 confirm_plan SSE 序列、同一个 SSE 连接。
```

## Phase 1/2 持久化说明

```text
Phase 1: plan 状态全内存（当前 HITL 行为）。不要求持久化。
Phase 2: PendingPlan 仍内存态。idempotencyKey 内存存储。
后续 Phase: 可选迁移到 SQLite/MySQL。
```
