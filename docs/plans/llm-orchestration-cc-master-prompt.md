# 可直接复制给 Claude Code / CC 的主 Prompt

```text
/llm-orchestration-dev

你现在只开发 AgentHub 新架构中的 services/orchestrator LLM 编排能力。

开始前必须使用这些 skill：
/project-architecture
/intent-orchestration-contract
/a2a-agent-contract
/adk-runtime-contract
/testing-review-contract

如果当前环境不能 slash 调用 skill，请阅读：
.claude/skills/project-architecture/SKILL.md
.claude/skills/intent-orchestration-contract/SKILL.md
.claude/skills/a2a-agent-contract/SKILL.md
.claude/skills/adk-runtime-contract/SKILL.md
.claude/skills/testing-review-contract/SKILL.md

执行计划：
按 docs/plans/llm-orchestration-development-plan.md 从 Phase 0 到 Phase 5 执行。
每个 Phase 完成后必须停止并输出 Phase Report，不要自动进入下一 Phase。

硬性允许修改：
services/orchestrator/planner/**
services/orchestrator/validator/**
services/orchestrator/plan/**
services/orchestrator/httpapi/** 仅允许追加 planner metadata
services/orchestrator/cmd/** 仅允许 planner wiring

硬性禁止修改：
frontend/**
services/gateway/**
docker-compose*
pkg/adk/**
pkg/runtime/agui/**
server/**
agents/**

任务目标：
1. LLMPlanner 成为主 Planner。
2. LLM 输出必须经过 Parser、Normalizer、Validator，不能直接执行。
3. 增加 one-shot Repairer。
4. RulePlanner 只作为 deprecated transitional fallback，禁止增强关键词规则。
5. Prompt 必须使用 registry agent 信息作为主要来源。
6. LLM mode 使用 single/parallel/sequential；internal 兼容映射为 single/ordered_parallel；sequential 当前拒绝，除非 executor/httpapi 完整支持。
7. 不改变 /api/chat、Gateway、AG-UI event type、tool call 顺序、text message 顺序。
8. 只允许在 run_started/state_update state 中追加可选 planner metadata。
9. 单元测试使用 fake model，不依赖真实 API key。

每个 Phase Report 必须包含：
- 修改文件
- 测试命令和结果
- git diff --name-only
- forbidden paths 未修改确认
- 行为证据
- 风险
- 是否建议继续下一 Phase
```
