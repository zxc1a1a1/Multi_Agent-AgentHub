---
name: llm-orchestration-dev
description: 用于按阶段实现 AgentHub 2.0 Orchestrator 的 LLM 计划生成、Registry Catalog、PlanVersion 校验、用户确认门、有限修复与依赖执行。
---

# AgentHub 2.0 LLM 编排开发 Skill

使用场景：开发 `services/orchestrator` 内的 LLM 编排与确认链路。

## 必须先调用或阅读

```text
/project-architecture
/planning-approval-contract
/agent-registry-contract
/context-management-contract
/a2a-agent-contract
/llm-provider-contract
/testing-review-contract
```

涉及公开 API、事件或持久化时，额外启用：

```text
/gateway-orchestrator-contract
/platform-api-contract
/agui-event-contract
/data-persistence-contract
```

读取这些 Skill 不等于自动获得修改对应模块的权限。

## 目标流水线

```text
Mode Resolver
  -> Registry eligible catalog
  -> Context bundle
  -> LLM Planner
  -> Parser
  -> Normalizer
  -> Validator
  -> Repair at most once
  -> PlanVersion awaiting confirmation
  -> Confirmation Gate
  -> Dependency Executor
```

Direct Mode 不创建多 Agent Plan。

## Phase Gate

每个 Phase 必须：

```text
只完成当前 Phase
只修改获批路径
运行当前 Phase 测试
输出 Phase Report
等待用户确认
```

不得自动进入下一 Phase。

## 默认允许路径

```text
services/orchestrator/planner/**
services/orchestrator/validator/**
services/orchestrator/plan/**
services/orchestrator/executor/**
services/orchestrator/httpapi/**
services/orchestrator/cmd/**
```

仅在任务明确授权时修改：

```text
services/orchestrator/registry/**
services/orchestrator/context/**
services/gateway/**
frontend/**
pkg/**
docs/contracts/**
```

## 禁止事项

```text
LLM 输出未经 Validator 直接执行
未确认 Plan 创建 AgentInvocation
Manual Mode 添加用户未选 Agent
Planner 使用 Registry 外 Agent
Planner 使用 disabled/unhealthy/unauthorized Agent
Planner 发明 Agent Skill
Normalizer 猜测未知 Agent
Repair 无限循环
Replan 不生成新 PlanVersion
Replan 不重新确认
静默替换确认后的 Agent
测试依赖真实 API key
跨 Conversation 注入历史
增强 RulePlanner 关键词以绕过 LLM/Validator
```

## RulePlanner 迁移

RulePlanner 是否保留由当前迁移任务决定。

允许：

- 作为明确标记的兼容 fallback；
- 生成结构化 PlanVersion；
- 经过相同 Validator；
- 经过相同用户确认门；
- 有独立指标和测试。

禁止：

- 绕过 PlanVersion；
- 绕过确认；
- 新增大量关键词规则替代 Planner 改造。

## 必测场景

```text
Direct Mode no-plan
Manual selected Agents only
Auto eligible catalog only
serial plan
parallel plan
invalid JSON repair
unknown Agent rejection
unknown Skill rejection
cycle rejection
unconfirmed execution rejection
stale PlanVersion rejection
Agent unavailable after confirmation
material Replan confirmation
repair failure/fallback
cross-conversation contamination negative
```

## 最终报告

```text
git diff --stat
git diff --name-only
修改/新增/删除文件
Phase Reports
测试命令和结果
Plan/confirmation evidence
Registry catalog evidence
repair/fallback evidence
forbidden paths confirmation
known risks
next dependency batch
```
