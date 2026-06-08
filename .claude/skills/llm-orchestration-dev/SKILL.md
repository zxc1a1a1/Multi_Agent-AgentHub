---
name: llm-orchestration-dev
description: 用于在 Claude Code 中按阶段实现 AgentHub services/orchestrator 的 LLM 编排开发。该 skill 强制执行范围边界、Phase Gate、Parser/Normalizer/Validator/Repairer 设计、禁止路径检查和阶段报告，防止 agent 乱改前端、Gateway、Docker、ADK 或 Runtime AG-UI。
---

# AgentHub LLM 编排开发 Skill

使用场景：只开发 `services/orchestrator` 的 LLM 编排能力。

## 必须先使用的项目 Skill

开始改代码前，必须先调用或阅读：

```text
/project-architecture
/intent-orchestration-contract
/a2a-agent-contract
/adk-runtime-contract
/testing-review-contract
```

如果需要确认前端事件兼容性，只能额外阅读：

```text
/agui-event-contract
/frontend-runtime-skills-contract
```

这两个只用于确认“不能破坏前端协议”，不是允许修改前端。

## 本 Skill 的唯一目标

把 Orchestrator 编排升级为：

```text
LLMPlanner
  -> Parser
  -> Normalizer
  -> Validator
  -> Repairer once
  -> Deprecated RulePlanner transitional fallback
  -> Executor
```

后续目标是删除 RulePlanner，因此本阶段禁止增强 RulePlanner 关键词规则。

## 必读 references

按顺序阅读并执行：

```text
references/00-execution-protocol.md
references/01-scope-boundary.md
references/02-design-standard.md
references/phase-0-readonly-recon.md
references/phase-1-schema-parser-prompt-trace.md
references/phase-2-normalizer-validator.md
references/phase-3-llm-planner-main-flow.md
references/phase-4-repair-fallback.md
references/phase-5-metadata-wiring-final-verify.md
references/report-template.md
```

## Phase Gate 强制规则

每个 Phase 必须：

```text
只做本 Phase 的任务
只改本 Phase 允许的路径
跑本 Phase 指定测试
输出 Phase Report
停下来等待用户确认
```

禁止自动进入下一 Phase。

## 允许修改路径

```text
services/orchestrator/planner/**
services/orchestrator/validator/**
services/orchestrator/plan/**
services/orchestrator/httpapi/**
services/orchestrator/cmd/**
```

## 禁止修改路径

```text
frontend/**
services/gateway/**
docker-compose*
pkg/adk/**
pkg/runtime/agui/**
server/**
agents/**
```

## 与 plan-approval-dev 的关系

```text
当任务同时涉及 llm-orchestration-dev 和 plan-approval-dev 时，plan-approval-dev 的边界规则优先。
non-auto executionPath 下，LLMPlanner 的 task.agentName 必须在 AllowedAgents 内，不得自行扩展。
main_agent_orchestration 下，LLM 可列出 candidateParticipants 推荐，但不得将 candidateParticipants/defaultSelectedParticipants 当作最终 selectedParticipants 直接执行。
```

## 最重要的禁止事项

```text
不要修改前端
不要修改 Gateway
不要修改 Docker
不要改 /api/chat
不要改 AG-UI event type
不要让 LLM 输出直接执行
不要新增 RulePlanner 关键词
不要让 Repairer 无限重试
不要让测试依赖真实 API key
```

## 最终报告要求

最终必须输出：

```text
git diff --stat
git diff --name-only
修改文件清单
测试命令和结果
三类正常场景证据
invalid JSON repair 证据
unknown agent repair 证据
repair fail fallback 证据
forbidden paths 未修改确认
RulePlanner 只作为 deprecated fallback 的确认
风险与后续建议
```
