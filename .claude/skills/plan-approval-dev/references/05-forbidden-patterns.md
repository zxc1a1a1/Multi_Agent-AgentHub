# 05 禁止模式

## 禁止模式 1: 审批前执行

```text
错误: 在 PLAN_PROPOSAL 的 TOOL_CALL_END 之后直接调用 Agent。
正确: 必须等待 POST /api/runs/{runId}/confirm (action=approve) 到达后才执行。
```

## 禁止模式 2: Planner 越界选 Agent

```text
错误: single_chat 下 LLMPlanner 返回 task.agentName = "web-agent"（非当前 Agent）。
错误: group_chat 下 Planner 自动添加 selectedAgentNames 之外的 Agent。
正确: non-auto 模式下，task.agentName 必须在 AllowedAgents 内。Validator 必须拦截越界。
```

## 禁止模式 3: candidateParticipants 当成 selectedParticipants

```text
错误: main_agent_orchestration 中，把 candidateParticipants 直接当成 selectedParticipants 去执行。
正确: candidateParticipants 是候选，defaultSelectedParticipants 是默认建议；只有 approve 当前 proposal 后才得到最终 selectedParticipants。未确认前不得调用任何候选/default Agent。
```

## 禁止模式 4: CANCEL 走 RUN_ERROR

```text
错误: CANCEL_PLAN emit RUN_ERROR + RUN_FINISHED。
正确: STATE_UPDATE(phase=cancelled) + RUN_FINISHED(status=cancelled)。无 RUN_ERROR。
```

## 禁止模式 5: revision 不由原 planOwner 修订

```text
错误: REQUEST_PLAN_REVISION 在所有 executionPath 都调用同一个 LLMPlanner 重新生成。
正确:
  - single_chat: 调用同一个 Agent plan_only 重新生成方案。
  - group_chat: 调用 group coordinator 重新生成生产计划。
  - main_agent_orchestration: 调用 main-agent 基于 feedback/participant changes 重新生成 revision+1 PLAN_PROPOSAL。
```

## 禁止模式 6: 重新 POST /api/chat 完成 action

```text
错误: approve/revise/cancel 后重新 POST /api/chat 创建新 run。
正确: 复用已有 SSE 流。action 通过 confirm endpoint JSON 返回 + 已有 SSE 流内续流。
```

## 禁止模式 7: Gateway 生成计划或调用 Agent

```text
错误: Gateway 的 handleChat 或 confirm handler 中生成 plan 或直接调用 Agent。
正确: Gateway 只验证、透传、转发。所有编排在 Orchestrator 内完成。
```

## 禁止模式 8: 前端直调 Orchestrator 或 Agent

```text
错误: 前端 POST 到 Orchestrator URL 或 Agent URL。
正确: 前端只调 Gateway (/api/chat、/api/runs/{runId}/confirm)。
```

## 禁止模式 9: main-agent 暴露为 code-agent

```text
错误: planOwner = { type: "agent", agentName: "code-agent" } 在 auto 模式下。
正确: planOwner = { type: "main_agent", agentName: "main-agent", isMainAgent: true }。
```

## 禁止模式 10: main_agent_orchestration 与 code-agent single_chat 混淆

```text
错误: 用户选 code-agent single_chat → 返回 planOwner.type="main_agent"。
      用户选 auto → 返回 planOwner.agentName="code-agent"。
正确: executionPath 和 planOwner 必须一致：
  - single_chat: executionPath=single_chat, planOwner={type:"agent", agentName:"<用户选的>"}
  - auto: executionPath=main_agent_orchestration, planOwner={type:"main_agent", agentName:"main-agent"}
```

## 禁止模式 11: Phase 1 提前实现后续 Phase 功能

```text
Phase 1 禁止实现:
  - PlanApprovalCard 组件
  - @mention UI（MessageInput 解析 / 自动补全）
  - MainAgent 组件
  - A2A plan_only / execute 模式
  - 删除 hasExplicitAgent HITL 跳过逻辑
  - 任何用户可见行为变更
```


## 禁止模式 12: 用 APPROVE_PLAN 直接提交 participant 修改

```text
错误: main_agent_orchestration 下用户修改 Agent 勾选后，前端仍发送 action=approve，后端直接按修改后的 selectedParticipants 执行。
正确: Agent 选择改变属于 REQUEST_PLAN_REVISION。main-agent 必须基于新选择重新生成 revision+1 PLAN_PROPOSAL，用户确认新版计划后才能执行。
```
