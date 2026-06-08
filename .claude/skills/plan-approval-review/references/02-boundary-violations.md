# 02 边界违规检查

## 违规 1: 审批前执行

```text
检测方法: 在所有 Agent 调用路径前 grep approval/confirm 检查。
识别特征: TOOL_CALL_END 之后紧接 Agent SDK/Runner 调用，没有等待 POST /api/runs/{runId}/confirm。
违规判定: 任何从 confirm_plan TOOL_CALL_END 直接进入 execute 分支的代码路径。
```

## 违规 2: Planner 越界选 Agent

```text
检测方法: 比对 Planner 输出的 task.agentName 与 AllowedAgents 列表。
single_chat: AllowedAgents = [用户选的 Agent]，任何 task.agentName != 用户选的 Agent 即违规。
group_chat: AllowedAgents = selectedAgentNames，任何 task.agentName 不在其中即违规。
non-auto 模式: AllowedAgents 不可为空，Planner 不可自行扩展。
违规判定: Validator 未拦截越界 task，或 Planner 直接输出了越界 agentName。
```

## 违规 3: candidateParticipants/defaultSelectedParticipants 当最终 selectedParticipants

```text
检测方法: 搜索 candidateParticipants 被直接用于 Agent 调用的代码路径。
main_agent_orchestration: candidateParticipants 仅是候选展示；defaultSelectedParticipants 仅是默认建议。只有 approve 当前 proposal 后才可形成最终 selectedParticipants。
违规判定: 任何在 confirm 前用 candidateParticipants 做了 Agent dispatch 的代码。
```

## 违规 4: CANCEL 走 RUN_ERROR

```text
检测方法: grep CANCEL_PLAN 附近的代码，查找 RUN_ERROR emit。
识别特征: STATE_UPDATE(phase=cancelled) 之前或同时 emit RUN_ERROR。
违规判定: CANCEL_PLAN 路径中出现了 RUN_ERROR。
正确行为: 仅 STATE_UPDATE(phase=cancelled) + RUN_FINISHED(status=cancelled)。
```

## 违规 5: revision 不由原 planOwner 修订

```text
检测方法: 搜索 REQUEST_PLAN_REVISION 处理逻辑，比对 planOwner 与 revision 调用的 Agent。
识别特征: 所有 executionPath 都调用同一个 Planner 重新生成。
违规判定: revision 时调用了与 planOwner 不匹配的 Agent/Planner。
正确: single_chat→同一Agent plan_only, group_chat→group coordinator, main_agent_orchestration→main-agent。
```

## 违规 6: 重新 POST /api/chat

```text
检测方法: grep confirm/approve/cancel 后的 HTTP 调用。
识别特征: action 处理后创建了新的 /api/chat POST 请求。
违规判定: confirm 返回后前端重新 POST /api/chat。
正确: confirm 返回 JSON，执行在已有 SSE 流上继续。
```

## 违规 7: Gateway 越界

```text
检测方法: 在 Gateway 代码中 grep plan/Planner/Agent/orchestrat。
识别特征: Gateway 文件中有任何 plan 生成、Agent 调用、语义路由逻辑。
违规判定: Gateway 中存在 orchestration 逻辑。
正确: Gateway 仅验证、透传、转发。
```

## 违规 8: 前端直调 Orchestrator/Agent

```text
检测方法: 前端代码中 grep orchestrator/agent URL。
识别特征: fetch/post 目标不是 Gateway (/api/*)。
违规判定: 前端直接请求 Orchestrator 或 Agent 的 HTTP 端点。
```

## 违规 9: main-agent 暴露为 code-agent

```text
检测方法: 搜索 planOwner 赋值代码，检查 agentName。
识别特征: planOwner = { type: "agent", agentName: "code-agent" } 出现于 auto 模式。
违规判定: auto 模式下 planOwner.agentName 为 "code-agent" 或 type 不为 "main_agent"。
```

## 违规 10: executionPath/planOwner 混淆

```text
检测方法: 交叉比对 executionPath 与 planOwner 的组合。
合法组合:
  executionPath=single_chat → planOwner={type:"agent", agentName:"<用户选>"}
  executionPath=group_chat → planOwner={type:"group_coordinator"}
  executionPath=main_agent_orchestration → planOwner={type:"main_agent", agentName:"main-agent", isMainAgent:true}
其他所有组合均为违规。
```

## 违规 11: AGENT_SELECTION_CONFLICT 不返回

```text
检测方法: 当 selectedAgentNames ∩ mentions = ∅ 时，检查是否仍生成了 plan。
违规判定: 交集为空时没有返回 AGENT_SELECTION_CONFLICT 错误，而是继续生成了 plan 或 fallback 到默认 Agent。
正确: 立即返回 AGENT_SELECTION_CONFLICT error，不生成 plan，不调用 Agent。
```

## 违规 12: Phase Gate 提前实现

```text
检测方法: 对比当前 Phase 范围与代码变更内容。
识别特征: Phase 1 实现了 Phase 3-6 的功能（PlanApprovalCard、@mention UI、MainAgent 等）。
违规判定: 代码变更超出当前 Phase 范围。
```

## 违规 13: requiredParticipants 可取消

```text
检测方法: 前端 participant 勾选组件中，requiredParticipants 的 disabled/checked 状态。
违规判定: requiredParticipants 的 checkbox 可被用户取消勾选。
正确: requiredParticipants 始终 checked + disabled。
```

## 违规 14: 用户改 participants 后无 revision

```text
检测方法: main_agent_orchestration 中用户修改 selectedParticipants 后的流程。
违规判定: 修改后直接执行而没有触发 revision+1 生成新 plan。
正确: 用户修改 participants → REQUEST_PLAN_REVISION → main-agent 以 participant changes 重新生成 revision+1 PLAN_PROPOSAL → 用户确认新版 plan。
```


## 违规 15: APPROVE_PLAN 被用于提交 participant 修改

检测方法: 检查 confirm/action handler 中 action=approve 且 selectedParticipants 与当前 proposal 默认/固定集合不一致的分支。
识别特征: 代码直接进入 executing，而不是返回 PARTICIPANT_CHANGE_REQUIRES_REVISION。
违规判定: approve 携带 participant changes 并导致执行。
正确: approve 只能接受当前 proposal；participant changes 必须走 revise。
