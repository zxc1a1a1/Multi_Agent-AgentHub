# 04 Phase Gates

## Phase 0.5：Contract + Skills 对齐（当前阶段）

```text
允许修改: docs/contracts/**、.claude/skills/plan-approval-*/**
禁止修改: frontend/**、services/**、pkg/**、任何业务代码
交付物:
  - docs/contracts/plan-approval.md
  - docs/contracts/chat-execution-path.md
  - docs/contracts/participant-boundary.md
  - 已有 contract 修订（agui-events.md、platform-api.md、openapi.yaml、gateway-orchestrator.md、intent-orchestration.md、a2a-task.md）
  - plan-approval-dev skill + references
  - plan-approval-review skill + references
```

## Phase 1：executionPath + Participant Boundary

```text
目标: 打通 agentName / selectedAgentNames / mentions / requestedPath / executionPath 传递链。
      non-auto 模式增加 Agent 边界校验。

前端:
  - AGUIChatRequest 增加 selectedAgentNames、mentions、requestedPath 字段
Gateway:
  - chatRequest 增加 requestedPath 字段
  - 新增 deriveExecutionPath() 函数
  - 透传 executionPath 到 Orchestrator
Orchestrator:
  - OrchestratorRequest 增加 ExecutionPath 字段
  - PlannerInput 增加 ExecutionPath、AllowedAgents 字段
  - 新增 validateAgentSelection() 函数
  - 新增 AGENT_SELECTION_CONFLICT 错误

不做:
  - PlanApprovalCard 组件
  - 前端 @mention UI
  - MainAgent 组件
  - A2A plan_only 协议
  - 删除 HITL 跳过逻辑
  - 任何用户可见行为变更

校验规则:
  - group_chat + selectedAgentNames ∩ mentions == [] → AGENT_SELECTION_CONFLICT
  - non-auto 模式 AllowedAgents 必须非空
```

## Phase 2：single_chat 计划确认最小闭环

```text
目标: selectedAgentNames=["code-agent"] 时先返回 PLAN_PROPOSAL，用户 approve 后仅 code-agent 执行。

ADK/A2A:
  - adk.GenerateRequest 增加 Mode 字段: "full" | "plan_only" | "execute"
  - a2a.RunRequest 增加 Mode 字段
  - dispatcher.DispatchInput 增加 Mode 字段
  - CodeAgent / WebAgent 增加 plan_only 和 execute 分支

Orchestrator:
  - single_chat 路径下以 plan_only 模式调用 Agent
  - 删除 hasExplicitAgent HITL 跳过逻辑
  - confirm_plan 增加 executionPath、planOwner、participants、revision 字段

前端:
  - 现有 HITLConfirm 复用（暂不建 PlanApprovalCard）

不做:
  - PlanApprovalCard 组件
  - revision 功能（只有 approve/cancel）
  - group_chat
  - main_agent_orchestration
```

## Phase 3：PlanApprovalCard 最小版

```text
目标: 前端展示 plan、支持 approve/cancel、feedback 输入 UI 接好。

前端:
  - 新增 PlanApprovalCard 组件
  - 展示: executionPath、planOwner、steps、participants
  - approve 按钮 → POST /api/runs/{runId}/confirm
  - cancel 按钮 → POST /api/runs/{runId}/confirm
  - feedback 输入框（UI 渲染，暂不触发 revision）
  - 按钮防重复（loading + disabled）
  - ChatWindow 集成

不做:
  - revision 功能（feedback 输入框可见但不触发 revision）
  - group_chat 多 turn 展示
  - main_agent_orchestration participant 勾选
```

## Phase 4：REQUEST_PLAN_REVISION 闭环

```text
目标: 用户输入 feedback → 原 planOwner 生成 revision+1 plan → 重新展示。

Orchestrator:
  - confirm endpoint 增加 action="revise" + feedback 处理
  - revise 分支按 executionPath 调用原 planOwner 重新生成 plan
  - revision 单调递增校验
  - revision mismatch 校验
  - idempotencyKey 校验存储

前端:
  - PlanApprovalCard revise 按钮启用
  - revision 更新后替换旧 plan
  - feedback history 展示

不做:
  - group_chat multi-turn
  - main_agent_orchestration participants 调整
```

## Phase 5：group_chat

```text
目标: 多 Agent 选择、@mention、participants 持久化、群聊生产计划、多 Agent 独立 turn。

前端:
  - AgentMultiSelect 组件
  - MessageInput @mention 解析 + 自动补全
  - PlanApprovalCard 多 Agent turn 展示

Gateway:
  - ConversationType 透传（不再硬编码 "single"）
  - conversation_participants Go 模型 + CRUD

Orchestrator:
  - group_chat plan 生成（group coordinator）
  - AGENT_TURN_STARTED/CONTENT/FINISHED 事件
  - Agent 边界校验

不做:
  - main_agent_orchestration
```

## Phase 6：auto / main_agent_orchestration

```text
目标: 主 Agent 给出计划和候选 participants；无修改则 approve 后执行；有修改/feedback 则 revision+1 后再确认。

Orchestrator:
  - MainAgent 组件（内部实现，不注册为 A2A Agent）
  - planOwner = { type: "main_agent", agentName: "main-agent" }
  - candidateParticipants + defaultSelectedParticipants 推荐
  - 用户修改 participants 或提交 feedback 后重新生成 revision+1 PLAN_PROPOSAL
  - 按最终确认 plan 执行；如计划包含汇总步骤，可由主 Agent/协调器汇总

前端:
  - PlanApprovalCard participant 勾选
  - required participant 不可取消
  - candidateParticipants 与 selectedParticipants 区分展示

不做:
  - MainAgent 远程化
```

## Phase 7：集成测试 + 合约定稿

```text
目标: 全链路 E2E 测试、contract 文档定稿。

- 8 个验证用例 E2E 测试
- Contract 文档最终更新
- Smoke test
- Phase Reports 汇总
```
