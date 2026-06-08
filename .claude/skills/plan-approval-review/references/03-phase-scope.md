# 03 Phase 范围检查

## Phase 判定

审核时必须先确认当前变更属于哪个 Phase，再按对应范围检查。

```text
Phase 1: executionPath + Participant Boundary
  允许: ChatRequest 字段扩展、deriveExecutionPath()、validateAgentSelection()、AGENT_SELECTION_CONFLICT
  禁止: PlanApprovalCard、@mention UI、MainAgent、A2A plan_only、删除 HITL 跳过、用户可见行为变更

Phase 2: single_chat 计划确认最小闭环
  允许: A2A Mode 字段、CodeAgent plan_only/execute 分支、confirm_plan 字段扩展、HITL 跳过删除
  禁止: PlanApprovalCard 组件、revision 功能、group_chat、main_agent_orchestration

Phase 3: PlanApprovalCard 最小版
  允许: PlanApprovalCard 组件、approve/cancel 按钮、feedback 输入框(不触发revision)、ChatWindow 集成
  禁止: revision 功能、group_chat 多 turn、main_agent_orchestration participant 勾选

Phase 4: REQUEST_PLAN_REVISION 闭环
  允许: action=revise、revision 单调递增、idempotencyKey 存储、feedback history
  禁止: group_chat multi-turn、main_agent_orchestration participants 调整

Phase 5: group_chat
  允许: AgentMultiSelect、@mention、conversation_participants、group coordinator、AGENT_TURN 事件
  禁止: main_agent_orchestration

Phase 6: auto / main_agent_orchestration
  允许: MainAgent 内部组件、candidateParticipants、defaultSelectedParticipants、participant 修改触发 revision、按最终确认 plan 执行
  禁止: MainAgent 远程化

Phase 7: 集成测试 + 合约定稿
  允许: E2E 测试、contract 更新、smoke test
```

## Phase Gate 检查模板

```text
当前 Phase: ____
变更内容: ____

逐项检查:
1. 是否实现了当前 Phase 的所有交付物? □
2. 是否提前实现了后续 Phase 功能? □
3. 是否遗漏了当前 Phase 的禁止修改路径? □
4. 是否有 Phase Report? □
```

## 常见 Phase 违规

```text
1. Phase 1 就写了 PlanApprovalCard 组件 → 违规
2. Phase 2 就实现了 revision 功能 → 违规
3. Phase 3 feedback 输入框触发了 revision → 违规
4. Phase 1 删除了 hasExplicitAgent HITL 跳过逻辑 → 违规（应在 Phase 2）
5. Phase 1 做了用户可见行为变更 → 违规
6. Phase 2 没有扩展 confirm_plan 字段 → 遗漏
```
