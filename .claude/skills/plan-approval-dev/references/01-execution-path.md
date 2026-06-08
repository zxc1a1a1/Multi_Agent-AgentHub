# 01 执行路径规则

## ChatExecutionPath 枚举

```text
single_chat              用户选一个 Agent，该 Agent 出方案并执行
group_chat               用户选多个 Agent / @mention，群聊协调器生成计划，多 Agent 依次产出
main_agent_orchestration  用户只发需求 / 选 auto，主 Agent 给出计划和候选 Agent；无修改则确认执行，有修改则 revision 后再确认
```

## 推导规则

Gateway 的 `deriveExecutionPath()` 按以下优先级推导：

```text
1. 如果 requestedPath 显式设置：
   - 校验 requestedPath 与 agentName / selectedAgentNames 兼容性。
   - 不兼容则返回错误（如 requestedPath=single_chat 但 selectedAgentNames 有多个）。
   - 兼容则使用 requestedPath。

2. 如果 requestedPath 为空 — 从 agent 选择字段推导：
   - agentName == "auto" 或 "" + selectedAgentNames == [] → main_agent_orchestration
   - agentName 是具体 Agent + selectedAgentNames == [] → single_chat
   - selectedAgentNames 非空 → group_chat（selectedAgentNames 优先于 agentName）
```

## 各 executionPath 行为

### single_chat

```text
planOwner:    { type: "agent", agentName: "<用户选的Agent>" }
plan 生成:    当前 Agent 以 plan_only 模式生成方案。LLMPlanner 不能替代 Agent 生成 plan。
participants: [{ agentName: "<用户选的Agent>", required: true, selected: true }]
AllowedAgents: [当前 Agent]
执行:         用户 approve 后，同一 Agent 以 execute 模式执行。
禁止:         调用任何其他 Agent。
禁止:         Planner 替代 Agent 生成 plan。
```

### group_chat

```text
planOwner:    { type: "group_coordinator" }
plan 生成:    群聊协调器基于 AllowedAgents 生成生产计划。
participants: AllowedAgents 中的 Agent 均为 required: true。
AllowedAgents: selectedAgentNames ∩ mentions（mentions 非空时取交集）
              selectedAgentNames（mentions 为空时）
执行:         用户 approve 后，多 Agent 按计划依次产出，每个 Agent 独立 turn。
禁止:         从全量 Agent 池自动补充 Agent。
禁止:         selectedAgentNames ∩ mentions == [] 时生成 plan（应返回 AGENT_SELECTION_CONFLICT）。
```

### main_agent_orchestration

```text
planOwner:    { type: "main_agent", agentName: "main-agent", isMainAgent: true }
plan 生成:    main-agent 生成一个计划，并从 all enabled agents 中列出相关候选 Agent。
AvailableBoundary: all enabled available agents
candidateParticipants: main-agent 推荐的候选 Agent 列表
defaultSelectedParticipants: main-agent 推荐的默认选中项
approve:       仅在用户不修改 Agent 且不提交 feedback 时有效；否则必须 revise
participants: 最终确认后的执行者列表
执行:         用户 approve 当前 proposal 后，按最终确认 selectedParticipants 执行；approve 不能携带 participant 修改。
禁止:         在用户确认前调用任何候选 Agent。
禁止:         把 candidateParticipants/defaultSelectedParticipants 当成最终 selectedParticipants。
禁止:         用户修改 participants 或提交 feedback 后直接执行；必须 revision+1 重新出 PLAN_PROPOSAL。
禁止:         main-agent 在 contract 层暴露为 code-agent。
```

## Phase 1 范围

```text
Phase 1 只做：
- requestedPath / selectedAgentNames / mentions 字段透传（前端→Gateway→Orchestrator）
- deriveExecutionPath() 实现（Gateway）
- validateAgentSelection() 实现（Orchestrator）
- AGENT_SELECTION_CONFLICT 返回（group_chat 交集为空时）

Phase 1 不做：
- 前端 @mention UI（MessageInput 解析 / 自动补全）→ Phase 5
- PlanApprovalCard 组件 → Phase 3
- MainAgent 组件实现 → Phase 6
- A2A plan_only / execute 模式 → Phase 2
- 删除 hasExplicitAgent HITL 跳过逻辑 → Phase 2
- 任何用户可见行为变更
```
