# 00 核心约束

## 先计划后执行

所有 executionPath 的通用约束：

```text
1. 用户发送消息后，Orchestrator 或 Agent 先生成 PLAN_PROPOSAL。
2. PLAN_PROPOSAL 通过已有 /api/chat SSE 连接发送到前端。
3. 用户必须通过 POST /api/runs/{runId}/confirm 明确 APPROVE_PLAN。
4. 用户 approval 前，不得调用任何 Agent 做执行。
5. 用户 approval 后，在同一个 SSE 连接上继续输出执行结果。
6. 不重新 POST /api/chat 来完成 approval 后续流程。
```

## runId / planId / revision 绑定

```text
- 每个 run 绑定一个 runId。
- 每个 PLAN_PROPOSAL 有唯一 planId。
- revision 从 1 开始，每次 REQUEST_PLAN_REVISION 后单调 +1。
- APPROVE_PLAN 或 REQUEST_PLAN_REVISION 必须校验 planId 和 revision 匹配。
```

## idempotencyKey 必须校验

```text
- 前端为每个 action 请求生成 UUID 作为 idempotencyKey。
- 相同 idempotencyKey + 相同 payload：返回第一次结果，不重复执行。
- 相同 idempotencyKey + 不同 payload：409 IDEMPOTENCY_KEY_CONFLICT。
- plan 已 executing/completed/cancelled/expired + 不同 idempotencyKey：409 INVALID_RUN_STATE。
```

## selectedParticipants 边界校验

```text
- requiredParticipants 不可被用户直接取消。
- selectedParticipants 必须来自 participants（或 candidateParticipants for auto）。
- selectedParticipants 不能包含 AllowedAgents 之外的 Agent。
- main_agent_orchestration 中 candidateParticipants/defaultSelectedParticipants ≠ 最终 selectedParticipants。
```

## 非 auto 不得越界

```text
- single_chat: AllowedAgents = [当前 Agent]。
- group_chat: AllowedAgents = selectedAgentNames ∩ mentions（mentions 非空时取交集）。
- 非 auto 的 Planner（LLMPlanner / RulePlanner）不得选择 AllowedAgents 之外的 Agent。
- Validator 必须在执行前拦截越界 task。
```

## Gateway 约束

```text
- Gateway 只做：验证请求、透传字段、转发 SSE、持久化。
- Gateway 不做：生成 plan、调用 Agent、语义路由、解释 action。
- Gateway 通过 deriveExecutionPath() 推导 executionPath，但不做编排决策。
```

## 前端约束

```text
- 前端只调 Gateway（/api/chat、/api/runs/{runId}/confirm）。
- 不直调 Orchestrator 或 Agent。
```

## Cancel 约束

```text
- CANCEL_PLAN 是正常生命周期，不作为系统错误。
- SSE 序列：STATE_UPDATE(phase=cancelled) → RUN_FINISHED(status=cancelled)。
- 不 emit RUN_ERROR。
```

## REQUEST_PLAN_REVISION 约束

```text
- 必须由原 planOwner 修订：
  - single_chat → 原 Agent plan_only 重新生成方案。
  - group_chat → group coordinator 基于原 participants + feedback 重新生成生产计划。
  - main_agent_orchestration → main-agent 基于原 plan + feedback + 用户请求的 participant changes 重新生成 revision+1 的 PLAN_PROPOSAL。
- feedback 非空校验。
- revision+1 后生成新 planId。
- 修订后的新 PLAN_PROPOSAL 在同一个 SSE 流上发送。
```

## Main-agent identity

```text
- planOwner.type = "main_agent"。
- planOwner.agentName = "main-agent"。
- 不得在 contract 层暴露为 code-agent。
- 第一版作为 Orchestrator 内部组件，不注册为 A2A Agent。
- 不与 code-agent single_chat 混淆。
```


## Auto participant change rule

```text
APPROVE_PLAN 表示接受当前 proposal 原样执行。
main_agent_orchestration 下，如果用户修改 Agent 选择或提交反馈，前端/后端必须走 REQUEST_PLAN_REVISION。
APPROVE_PLAN 携带与 defaultSelectedParticipants 不一致的 selectedParticipants 时，后端必须返回 PARTICIPANT_CHANGE_REQUIRES_REVISION。
```
