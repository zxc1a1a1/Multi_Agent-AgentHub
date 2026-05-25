# Project Architecture Contract

## 1. 目的

本契约定义 AgentHub v1.0 与后续演进的项目级架构边界。它是服务拆分、职责判断、协议归属、文档同步和架构 Review 的事实源。

## 2. 当前硬约束

- Gateway Service 与 Orchestrator Service 必须分进程。
- Frontend 只能访问 Gateway。
- Orchestrator 不直接暴露给 Frontend。
- Child Agent 不直接暴露给 Frontend。
- Gateway 不做 Agent 编排。
- Orchestrator 负责 LLM 意图编排、多 Agent 调度、fallback / retry 和结果聚合。

## 3. v1.0 Profile

v1.0 当前目标包括：

- 2+ Agent。
- 单聊与群聊。
- LLM 编排。
- Agent Registry 与健康检查。
- 结构化多轮消息。
- 丰富产物预览。
- fallback / retry。
- single / ordered_parallel / sequential。
- docker compose Demo。
- smoke test。
- README、架构图、AI 协作文档。

## 4. 服务职责

### Frontend

负责 UI、单聊/群聊、消息展示、流消费、产物预览、编排状态展示。

### Gateway

负责公开 REST、公开 stream、鉴权、持久化入口、调用 Orchestrator、错误映射。

### Orchestrator

负责编排计划、Agent 选择、任务调度、fallback / retry、结果聚合。

### Child Agent

负责声明能力、健康检查、执行任务、返回文本与产物引用。

### Data Layer

负责持久化、缓存、对象存储等 attached resources。

## 5. 长期通用性

项目架构不得绑定具体 Agent 名称。代码类 Agent、网页类 Agent、文档类 Agent 只能作为示例。

新增 Agent 应通过能力声明、Registry、Health、Orchestrator 计划和 Artifact / Runtime Capability 扩展，而不是修改总架构。

## 6. Historical Profile

MVP v0.1 已完成，只作为历史兼容和回归测试基线。旧 MVP 限制不得继续作为 v1.0 当前禁令。
