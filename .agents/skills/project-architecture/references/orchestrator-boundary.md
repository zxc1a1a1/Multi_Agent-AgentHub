# Orchestrator Boundary

Orchestrator Service 是内部编排服务。

## 负责

- 意图理解。
- LLM Planner 与规则 fallback。
- OrchestrationPlan。
- Registry 查询。
- Health 过滤。
- Agent 选择。
- single / ordered_parallel / sequential。
- 子任务调度。
- fallback / retry。
- 多 Agent 结果聚合。
- 状态更新。
- Artifact 与 ToolCall 引用归一。

## 禁止

- 暴露给 Frontend。
- 处理用户登录态。
- 管理浏览器连接。
- 定义公开 REST response envelope。
- 返回前端组件实现。

## 输出要求

Orchestrator 输出必须可由 Gateway 转发、可追踪、可持久化，并能关联 runId、planId、taskId、agentName。
