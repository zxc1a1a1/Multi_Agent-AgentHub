# 01 审核清单

## 审批前执行检查

- [ ] 所有 Agent 调用路径前是否有 approval status 检查？
- [ ] 是否在 confirm_plan TOOL_CALL_END 之后立即调用了 Agent（应等 POST /api/runs/{runId}/confirm）？
- [ ] 是否在 PLAN_PROPOSAL SSE 序列中调用了 Agent 做执行？

## executionPath 边界检查

- [ ] Validator 是否拦截了 single_chat 下非当前 Agent 的 task？
- [ ] Validator 是否拦截了 group_chat 下 AllowedAgents 之外的 task？
- [ ] deriveExecutionPath() 是否正确处理了所有推导规则？
- [ ] AGENT_SELECTION_CONFLICT 是否在交集为空时正确返回（不生成 plan）？

## participant 检查

- [ ] requiredParticipants 是否在前端和后端都禁止取消？
- [ ] main_agent_orchestration: candidateParticipants/defaultSelectedParticipants ≠ 最终 selectedParticipants？
- [ ] 用户修改 participants 后是否触发了 revision+1？

## planOwner 检查

- [ ] single_chat: planOwner.type="agent", agentName=用户选的 Agent？
- [ ] group_chat: planOwner.type="group_coordinator"？
- [ ] main_agent_orchestration: planOwner.type="main_agent", agentName="main-agent"？
- [ ] main-agent 没有被暴露为 code-agent？

## cancel 生命周期检查

- [ ] CANCEL_PLAN 是否 emit STATE_UPDATE(phase=cancelled) + RUN_FINISHED(status=cancelled)？
- [ ] CANCEL_PLAN 是否没有 emit RUN_ERROR？

## REQUEST_PLAN_REVISION 检查

- [ ] single_chat revision 是否调用了同一个 Agent plan_only？
- [ ] group_chat revision 是否调用了 group coordinator？
- [ ] main_agent_orchestration revision 是否调用了 main-agent？
- [ ] feedback 是否做了非空校验？
- [ ] revision 是否单调递增？
- [ ] revision+1 后的新 planId 是否不同于原 planId？

## SSE 续流检查

- [ ] confirm endpoint 是否返回 JSON 而非 SSE？
- [ ] action 后执行是否在已有 /api/chat SSE 流上继续？
- [ ] 是否没有重新 POST /api/chat？

## Gateway 边界检查

- [ ] Gateway 是否有任何 plan 生成逻辑？
- [ ] Gateway 是否有任何 Agent 调用？
- [ ] Gateway 是否有语义路由？
- [ ] Gateway 是否透传了 executionPath、selectedAgentNames、mentions、requestedPath？

## Phase Gate 合规检查

- [ ] 当前 Phase 是否没有提前实现后续 Phase 功能？
- [ ] 如果是 Phase 1: 是否有 PlanApprovalCard / @mention UI / MainAgent / A2A plan_only？
- [ ] 是否有 Phase Report？


## Auto approve/revise checklist

- [ ] main_agent_orchestration 下，用户未修改 Agent 且未 feedback 时，APPROVE_PLAN 使用 defaultSelectedParticipants 执行？
- [ ] main_agent_orchestration 下，用户修改 Agent 或提交 feedback 时，是否走 REQUEST_PLAN_REVISION 而不是 APPROVE_PLAN？
- [ ] APPROVE_PLAN 中 changed selectedParticipants 是否返回 PARTICIPANT_CHANGE_REQUIRES_REVISION？
- [ ] revise 是否在 feedback 为空但 participants changed 时仍允许生成新版 plan？
