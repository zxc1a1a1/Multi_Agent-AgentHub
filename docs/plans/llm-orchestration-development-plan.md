# AgentHub LLM 编排开发详细执行计划书 v2

> 给 Claude Code / CC 直接执行。  
> 目标：按阶段实现 `services/orchestrator` 的 LLM 编排主路径，同时防止 agent 乱改前端、Gateway、Docker、ADK、Runtime AG-UI 或旧目录。

## 1. 总目标

把 Orchestrator 编排升级为：

```text
LLMPlanner
  → Parser
  → Normalizer
  → Validator
  → Repairer once
  → Deprecated RulePlanner transitional fallback
  → Executor
```

本阶段不删除 RulePlanner，但必须把它降级为 deprecated transitional fallback。  
禁止增强 RulePlanner 关键词规则。

## 2. 执行方式

在 Claude Code 中：

```text
/llm-orchestration-dev

按 docs/plans/llm-orchestration-development-plan.md 执行。
从 Phase 0 开始。
每个 Phase 完成后停止并输出报告，不要自动进入下一 Phase。
```

## 3. 必须先使用的 Skill

```text
/project-architecture
/intent-orchestration-contract
/a2a-agent-contract
/adk-runtime-contract
/testing-review-contract
```

只在确认事件兼容时阅读：

```text
/agui-event-contract
/frontend-runtime-skills-contract
```

## 4. 允许修改路径

```text
services/orchestrator/planner/**
services/orchestrator/validator/**
services/orchestrator/plan/**
services/orchestrator/httpapi/**
services/orchestrator/cmd/**
```

## 5. 禁止修改路径

```text
frontend/**
services/gateway/**
docker-compose*
pkg/adk/**
pkg/runtime/agui/**
server/**
agents/**
```

## 6. Phase Gate

每个 Phase 必须：

```text
读本 Phase 指令
只改允许路径
跑指定测试
输出 Phase Report
停下来等待确认
```

禁止自动进入下一 Phase。

## 7. Phase 0：只读侦察

详见：

```text
.claude/skills/llm-orchestration-dev/references/phase-0-readonly-recon.md
```

目标：

```text
确认现有 planner、plan、validator、httpapi、strategy、event 输出点。
不改代码。
```

## 8. Phase 1：Schema / Parser / Prompt / Trace

详见：

```text
.claude/skills/llm-orchestration-dev/references/phase-1-schema-parser-prompt-trace.md
```

目标：

```text
定义 LLM plan schema
实现 parser
实现 prompt builder
实现 planner trace
暂不接主流程
```

## 9. Phase 2：Normalizer / Validator

详见：

```text
.claude/skills/llm-orchestration-dev/references/phase-2-normalizer-validator.md
```

目标：

```text
LLM mode=single -> internal single
LLM mode=parallel -> internal ordered_parallel
LLM mode=sequential -> 当前拒绝
Validator 确定性校验所有 plan
```

## 10. Phase 3：LLMPlanner 主路径

详见：

```text
.claude/skills/llm-orchestration-dev/references/phase-3-llm-planner-main-flow.md
```

目标：

```text
LLMPlanner 调 fake/real model abstraction
parse
normalize
validate
valid plan 直接返回，不走 RulePlanner
```

## 11. Phase 4：Repair once + fallback

详见：

```text
.claude/skills/llm-orchestration-dev/references/phase-4-repair-fallback.md
```

目标：

```text
parse/validate fail -> repair once
repair success -> 执行 repaired plan
repair fail -> 当前阶段 fallback RulePlanner
RulePlanner 标 deprecated，禁止新增关键词
```

## 12. Phase 5：Metadata / Wiring / Final Verify

详见：

```text
.claude/skills/llm-orchestration-dev/references/phase-5-metadata-wiring-final-verify.md
```

目标：

```text
planner wiring
只追加可选 state metadata
不改 /api/chat
不改 event type
不改 Gateway/frontend
```

## 13. 最终交付

必须提供：

```text
git diff --stat
git diff --name-only
Phase Reports
go test 输出
行为证据
风险清单
```

审核时使用：

```text
/llm-orchestration-review
```
