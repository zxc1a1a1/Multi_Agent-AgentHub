---
name: llm-orchestration-review
description: 用于审核 AgentHub services/orchestrator LLM 编排开发结果。该 skill 检查路径边界、LLMPlanner 主路径、Parser/Normalizer/Validator/Repairer、RulePlanner 是否被增强、前端兼容性、测试证据和交接报告。
---

# AgentHub LLM 编排审核 Skill

使用场景：审核 `/llm-orchestration-dev` 执行后的改动。

## 审核输入

要求提供：

```text
git diff --stat
git diff --name-only
git diff
Phase Reports
go test 输出
最终报告
```

## 必读 references

```text
references/review-checklist.md
references/rejection-rules.md
references/required-evidence.md
```

## 立即拒绝条件

出现以下情况，默认 REJECTED：

```text
frontend/** 被修改
services/gateway/** 被修改
docker-compose* 被修改
pkg/adk/** 被修改
pkg/runtime/agui/** 被修改
server/** 被修改
agents/** 被修改
RulePlanner 新增关键词
LLM 输出未经 Validator 直接执行
Repairer 可多次循环修复
测试依赖真实 API key
/api/chat 被改变
AG-UI event type 被改变
```

## 审核重点

必须检查：

```text
LLMPlanner 是否主路径
Parser 是否独立
Normalizer 是否只翻译不猜测
Validator 是否确定性且完整
Repairer 是否最多一次
RulePlanner 是否 deprecated fallback only
Prompt 是否使用 registry agent 信息
State metadata 是否可选且不破坏前端
测试是否覆盖 repair/fallback
```

## 结论格式

输出一个：

```text
APPROVED
APPROVED WITH FOLLOW-UP
CHANGES REQUESTED
REJECTED
```

并包含：

```text
Summary
Blocking issues
Non-blocking issues
Evidence checked
Required fixes
Next phase suggestion
```
