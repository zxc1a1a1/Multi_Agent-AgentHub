---
name: plan-approval-review
description: 用于审核 AgentHub 多路径计划确认方案开发结果。该 skill 检查 approval 前是否调 Agent、executionPath 边界、participant 越界、cancel 是否错误走 RUN_ERROR、revision 校验、Phase Gate 合规、禁止模式违规。
---

# AgentHub 多路径计划确认审核 Skill

## 审核输入

审核时必须要求提供：

```text
git diff --stat
git diff
Phase Report（当前 Phase 的交付物清单、测试结果）
go test 输出（services/orchestrator/...、services/gateway/...）
npm test 输出（如有前端变更）
```

## 必读 references

```text
references/01-review-checklist.md
references/02-boundary-violations.md
references/03-phase-scope.md
references/04-event-lifecycle-review.md
```

## 立即拒绝条件

出现以下任一情况，审核直接 REJECTED：

```text
1. 在用户 approval 前调用了 Agent 执行
2. Gateway 生成了 plan 或直接调用了 Agent
3. 前端直调了 Orchestrator 或 Agent
4. single_chat 调用了非选定 Agent
5. non-auto 模式下 Planner 越界选择了 Agent（task.agentName 不在 AllowedAgents 内）
6. group_chat 自动补充了 selectedAgentNames/mentions 范围外的 Agent
7. main_agent_orchestration 中 candidateParticipants/defaultSelectedParticipants 被当成最终 selectedParticipants 直接执行
8. CANCEL_PLAN emit 了 RUN_ERROR
9. REQUEST_PLAN_REVISION 不是由原 planOwner 修订（而是统一调用 Planner）
10. main-agent 在 contract 层暴露为 code-agent（planOwner.agentName="code-agent" 在 auto 模式下）
11. main_agent_orchestration 与 code-agent single_chat 在 planOwner 上混淆
12. 修改了 server/**、agents/** 等禁止路径
13. Phase 1 提前实现了 PlanApprovalCard / @mention UI / MainAgent / A2A plan_only
14. 重新 POST /api/chat 来完成 action 而非复用已有 SSE 流
15. selectedAgentNames ∩ mentions == [] 时仍生成了 plan（应返回 AGENT_SELECTION_CONFLICT）
16. 缺少 runId / planId / revision 校验
17. 缺少 idempotencyKey 校验
18. requiredParticipants 可被用户直接取消
19. 用户修改 participants 后没有重新生成 revision+1 PLAN_PROPOSAL 而直接执行（main_agent_orchestration）
20. APPROVE_PLAN 携带被修改的 participants 仍被当作批准执行，而不是返回 PARTICIPANT_CHANGE_REQUIRES_REVISION
```

## 审核重点

```text
1. 审批前执行检查：确认所有 Agent 调用的代码路径前都有 approval 检查。
2. executionPath 边界检查：确认 Validator 已拦截越界 task。
3. participant 检查：确认 requiredParticipants 不可取消，candidateParticipants/defaultSelectedParticipants 不能被误当成最终 selectedParticipants。
4. planOwner 检查：确认 planOwner 与 executionPath 一致，main-agent 未暴露为 code-agent。
5. cancel 生命周期检查：确认 CANCEL_PLAN 不 emit RUN_ERROR。
6. REQUEST_PLAN_REVISION 分发检查：确认按 executionPath 调用原 planOwner。
7. SSE 续流检查：确认 action 后在已有 SSE 连接上继续，未重新 POST /api/chat。
8. Gateway 边界检查：确认 Gateway 未生成 plan、未调 Agent、未做语义路由。
9. Phase Gate 合规检查：确认当前 Phase 没有提前实现后续 Phase 功能。
10. contract 一致性检查：确认实现与 contract 一致。
```

## 结论格式

```text
APPROVED              所有检查通过，可以进入下一 Phase
APPROVED WITH FOLLOW-UP  通过但有已知问题需下一 Phase 处理
CHANGES REQUESTED     有需修正的问题
REJECTED              违反立即拒绝条件

报告必须包含:
- Summary
- Blocking issues（如有）
- Non-blocking observations（如有）
- Phase Gate compliance
- Contract consistency check
- Test coverage assessment
```
