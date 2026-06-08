---
name: plan-approval-dev
description: 用于实现 AgentHub 多路径计划确认方案。该 skill 覆盖 frontend/Gateway/Orchestrator/ADK 的跨模块 plan approval 开发，强制执行先计划后执行、executionPath 边界、participant 校验、Phase Gate 和 contract-first 约束。
---

# AgentHub 多路径计划确认开发 Skill

## 本 Skill 的唯一目标

实现三条 executionPath 的计划确认闭环：

1. **single_chat**：用户选 Agent → 该 Agent 以 plan_only 模式生成方案 → 用户确认 → 该 Agent 执行。
2. **group_chat**：用户选多 Agent / @mention → 群聊协调器生成生产计划 → 用户确认 → 多 Agent 依次产出。
3. **main_agent_orchestration**：用户只发需求 / 选 auto → 主 Agent 给出计划并列出推荐/候选 Agent → 用户不修改且不反馈则 approve 后按当前计划执行；用户修改 Agent 或反馈则必须 revision+1 后再确认。

## 必须先使用的项目 Skill

在改代码之前，必须先执行以下 skill 获取项目上下文：

- `/project-architecture` — 项目架构边界
- `/platform-api-contract` — Gateway 公共 API
- `/gateway-orchestrator-contract` — Gateway-Orchestrator 服务边界
- `/agui-event-contract` — AG-UI 事件流
- `/frontend-runtime-skills-contract` — 前端 skills 渲染
- `/intent-orchestration-contract` — Planner / Executor 边界
- `/a2a-agent-contract` — A2A 子 Agent 协议
- `/security-boundary-contract` — 安全边界

## 必读 references

所有 reference 文档必须按顺序阅读：

```text
references/00-core-constraints.md
references/01-execution-path.md
references/02-participant-boundary.md
references/03-approval-lifecycle.md
references/04-phase-gates.md
references/05-forbidden-patterns.md
```

## Phase Gate 强制规则

1. 每个 Phase 必须完成当前 Phase 的全部交付物，才能进入下一 Phase。
2. 每个 Phase 结束后必须输出 Phase Report（含 git diff --stat、测试结果、变更文件清单）。
3. 禁止跨 Phase 提前实现。例如 Phase 1 禁止实现 PlanApprovalCard、@mention UI、MainAgent、A2A plan_only。
4. 如果当前 Phase 遇到需要后续 Phase 才能完成的能力，必须记录为已知限制，不能跳过 Phase Gate。
5. 禁止自动进入下一 Phase。必须等待人工确认 Phase Report 后才能继续。

## 允许修改路径

```text
frontend/src/**
services/gateway/**
services/orchestrator/**
pkg/adk/**
docs/contracts/**
.claude/skills/plan-approval-dev/**
```

## 禁止修改路径

```text
server/**
agents/**
docker-compose* (除非 Phase 需要新的 service 定义)
pkg/runtime/agui/  (除非修改 event 翻译规则，需在报告中说明)
```

## 最重要的禁止事项

```text
禁止在用户 approval 前调用任何 Agent 执行
禁止 single_chat 调用非选定 Agent
禁止 non-auto 模式下 Planner 越界选择 Agent
禁止 group_chat 从全量 Agent 池自动补 Agent
禁止 main_agent_orchestration 在用户确认前调用候选 Agent 或 defaultSelectedParticipants
禁止 candidateParticipants/defaultSelectedParticipants 被当成最终 selectedParticipants
禁止 APPROVE_PLAN 携带被修改的 participants 直接执行；必须 REQUEST_PLAN_REVISION
禁止用户取消 requiredParticipants
禁止 CANCEL_PLAN emit RUN_ERROR
禁止 REQUEST_PLAN_REVISION 不由原 planOwner 修订
禁止 Gateway 生成计划 / 调用 Agent / 做语义路由
禁止前端直调 Orchestrator 或 Agent
禁止重新 POST /api/chat 来完成 action — 必须复用已有 SSE 流
禁止 main-agent 在 contract 层暴露为 code-agent
禁止 main_agent_orchestration 与 code-agent single_chat 混淆
```

## 最终报告要求

每个 Phase 完成后必须输出：

```bash
git diff --stat
```

并列出：
- 变更文件清单
- 新增 / 修改的测试用例数量
- 测试运行结果（pass / fail / skip）
- 已知限制和待后续 Phase 处理的问题
- 与 contract 的一致性确认
