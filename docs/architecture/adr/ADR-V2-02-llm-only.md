# ADR-V2-02：LLM-only 编排，删除全部规则回退

- 日期：2026-10-29
- 状态：已采纳（P0）
- 背景：`main_agent.go` 在模型失败/缺失时使用关键词评分；启动默认 `rule`，缺 API key 或模型创建失败会退回规则模式。旧契约允许 RulePlanner fallback。
- 决策：生产启动图只有 MainAgent 一个语义入口。删除关键词评分、固定 Agent fallback、`rule`/`llm_with_rule_fallback` 模式与缺配置降级。模型失败显式报错；schema 错误最多一次模型修复；意图不明使用 `clarify`。确定性鉴权/校验/预算/调度保留。
- 备选：关键词路由（能力扩展依赖词表）；LLM + 规则 fallback（失败时语义突变）；任意 Agent 自由 handoff（上下文/预算/终态复杂，首版不做）。
- 后果：离线测试必须注入 FakeModel；生产缺模型配置启动/就绪失败；`RulePlanner` 进入移除清单而非运行注入。
- 同步契约：`intent-orchestration.md`、`planner-input.md`、`orchestration-plan.md`、`chat-execution-path.md`、`agent-config.md`。
