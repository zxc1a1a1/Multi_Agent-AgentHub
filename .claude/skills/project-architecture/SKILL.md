---
name: project-architecture
description: "用于定义 AgentHub v1.0 及后续演进的项目级总架构契约，包括 IM 聊天式多 Agent 协作、Gateway 与 Orchestrator 分进程、Frontend/Gateway/Orchestrator/Child Agent/Data Layer 边界、Conversation、Registry、Artifact、执行策略、fallback/retry、Docker Demo、AI 协作文档和架构 Review。本 Skill 不绑定具体 Agent。"
---

# project-architecture

## 1. Skill 目的

本 Skill 是 AgentHub 的项目级总架构事实源，用于判断系统边界、服务拓扑、进程职责、通信方向、交付约束与架构 Review 是否一致。

本 Skill 面向 v1.0 与后续扩展。它不生成业务代码，不替代具体协议、数据库、前端组件或部署实现契约。

## 2. 当前事实源

当前项目事实源以 `SPRINT-v1.0-Plan.md` 为 v1.0 目标来源：v1.0 方向是多 Agent 协作、LLM 意图编排、单聊与群聊、Agent Registry 与健康检查、结构化多轮消息、丰富产物预览、降级重试和一键 Demo。

MVP v0.1 已完成，只能作为历史兼容与回归测试基线。旧的 MVP 限制不得继续作为当前开发禁令。

Gateway Service 与 Orchestrator Service 必须是两个独立进程，这是当前架构硬约束。

## 3. 独立性原则

本 Skill 必须独立可读。读者只看本文件和本 Skill 的 references，就应理解 AgentHub 的项目级架构边界。

允许说明与其他契约的边界，但不得复制其他契约的完整字段或实现规则。

## 4. 本 Skill 负责什么

本 Skill 负责：

- AgentHub 系统上下文。
- v1.0 Sprint Profile 与长期架构边界。
- Frontend、Gateway Service、Orchestrator Service、Child Agent Services、Data Layer 的职责划分。
- Gateway 与 Orchestrator 分进程硬规则。
- Frontend 只能访问 Gateway 的公开 API 与公开 stream。
- Gateway 公开 API 与内部服务 API 隔离。
- Orchestrator 作为 LLM 编排中心。
- 2+ Agent、单聊、群聊、@mention、多消息归属的架构位置。
- Agent Registry 与 Health Check 的架构位置。
- Artifact 与 Runtime Capability 的架构位置。
- fallback、retry、ordered_parallel 的架构位置。
- Docker Demo、smoke test、README、架构图、AI 协作文档的交付边界。
- 架构 Review Checklist。

## 5. 本 Skill 不负责什么

本 Skill 不负责：

- REST API 的完整 request / response 字段。
- Gateway 与 Orchestrator 内部 API 的完整 schema。
- 子 Agent 通信协议字段。
- LLM Provider SDK 适配细节。
- Artifact 完整 schema。
- 数据库 DDL。
- 前端组件实现。
- Docker Compose 实现。
- Go / TypeScript / SQL / Dockerfile 业务实现代码。

## 6. AgentHub v1.0 System Context

AgentHub 是 IM 聊天式多 Agent 协作平台。用户通过对话与 Agent 协作，系统通过 Orchestrator 理解意图、生成计划、选择 Agent、调度任务并聚合结果。

v1.0 架构必须支持：

- 2+ Child Agent。
- 单聊与群聊。
- LLM 意图编排。
- Agent Registry 与健康检查。
- 结构化多轮上下文。
- 多 Agent 消息归属。
- code、webpage、markdown 等丰富产物类型的通用扩展能力。
- fallback 与 retry 提示。
- 可运行 Demo、smoke test 和 AI 协作文档。

## 7. Target Service Topology

目标服务拓扑：

```text
Frontend App
  ↓ Public REST / Public SSE or stream
Gateway Service
  ↓ Protected Internal API / Internal stream
Orchestrator Service
  ↓ Agent protocol / Registry / LLM Provider / Artifact normalization
Child Agent Services
  ↓ Attached Resources
Data Layer / Object Storage / Cache / LLM Providers
```

说明：

- Frontend 只访问 Gateway。
- Gateway 是对外入口，不承载 Agent 编排。
- Orchestrator 是独立内部编排服务。
- Child Agent 是通用能力提供服务。
- Data Layer、Object Storage、Cache、LLM Provider 都属于 attached resources，不应被单进程内存状态替代。

## 8. Process Boundary Hard Rules

硬规则：

- Gateway Service 与 Orchestrator Service 必须是两个独立进程。
- Frontend 不得直接访问 Orchestrator Service。
- Frontend 不得直接访问 Child Agent Services。
- Gateway 不得 import Orchestrator 业务包并在 handler 中执行编排。
- Gateway 不得直接调用 Child Agent。
- Gateway 不得直接调用 LLM Provider。
- Orchestrator 不得直接暴露给浏览器。
- Child Agent 不得反向调用 Gateway 用户 API。

如果仓库当前存在同进程历史实现，只能视为迁移期 legacy 形态；新增功能不得继续加深同进程耦合。

## 9. Frontend Boundary

Frontend App 负责：

- IM 聊天 UI。
- 对话列表。
- 单聊与群聊 UI。
- @Agent 提及交互。
- 多 Agent 消息渲染。
- senderName、Agent 头像、Agent 状态展示。
- SSE / stream 消费。
- 产物预览。
- 编排状态可视化。
- 用户确认和交互。

Frontend App 禁止：

- 直接访问 Orchestrator。
- 直接访问 Child Agent。
- 实现 Agent 编排策略。
- 实现 Agent fallback。
- 直接调用 LLM Provider。
- 手写与公开 API 契约不一致的 API 类型。

## 10. Gateway Service Boundary

Gateway Service 负责：

- 对外 REST API。
- 对外 SSE / stream endpoint。
- 用户鉴权与权限边界。
- CORS、requestId、traceId。
- 会话、消息、Agent 摘要、Artifact 元数据查询入口。
- 接收前端 run 请求。
- 调用 Orchestrator Service 内部 API。
- 将 Orchestrator 内部事件转成前端可消费事件。
- 持久化用户消息和结果入口。
- 将内部错误转换为前端安全错误。

Gateway Service 禁止：

- 实现意图编排。
- 调 LLM 生成计划。
- 根据用户意图选择 Agent。
- 直接调用 Child Agent。
- 执行 fallback / retry 策略。
- 合并多个 Agent 的业务结果。
- 把 Orchestrator 业务逻辑写进 handler。

## 11. Orchestrator Service Boundary

Orchestrator Service 负责：

- 意图理解。
- LLM Planner / rule fallback。
- OrchestrationPlan / ExecutionPlan。
- Agent Registry 查询。
- Health Check 过滤。
- Agent 选择。
- single / ordered_parallel / sequential 执行策略。
- 子任务调度。
- fallback / retry。
- 多 Agent 结果聚合。
- 状态更新。
- Artifact / ToolCall 引用归一。

Orchestrator Service 禁止：

- 直接暴露给 Frontend。
- 处理用户登录态。
- 返回 React 组件。
- 管理浏览器连接。
- 定义公开 REST API response envelope。
- 依赖 Gateway handler 内部类型作为业务输入。

## 12. Child Agent Service Boundary

Child Agent Service 是通用能力提供服务，不以具体 Agent 名称定义项目架构。

Child Agent Service 负责：

- 声明身份。
- 声明能力。
- 暴露健康检查。
- 接收任务。
- 流式返回文本。
- 返回 Artifact、ToolCall、status 或等价结果引用。

Child Agent Service 禁止：

- 直接访问 Frontend。
- 直接访问 Gateway 用户 API。
- 自己决定全局编排。
- 控制前端组件。
- 返回未声明能力的产物。

示例：代码类 Agent、网页类 Agent、文档类 Agent可以作为 Sprint 或 Demo 示例，但不得成为长期架构固定枚举。

## 13. Conversation Architecture

Conversation 是一级架构对象。

规则：

- Conversation 必须支持 `single` 与 `group`。
- Group conversation 可以包含多个 Agent participant。
- 用户输入可以通过 `@mention` 提示目标 Agent。
- `@mention` 是路由提示，不是绕过 Orchestrator 的直接调用。
- 一个 run 可以产生多条 Agent message。
- 每条 Agent message 必须能标识 senderName / senderId。
- 前端展示 Agent 头像和名称只依赖 message sender 字段，不依赖写死 Agent 名称。

## 14. Agent Registry / Health Check Boundary

Agent Registry 是 Orchestrator 选择 Agent 的能力目录来源。Health Check 是 Agent 可用性判断来源。

Agent 发现、Agent URL、AgentCard、Health 信息的权威调用方是 Registry / Orchestrator。

规则：

- Gateway 可以读取脱敏 Agent 摘要用于前端展示。
- Gateway 不得直接持有 Child Agent 调用地址（Agent URL / service name）。
- Gateway 不得基于 Agent URL 执行调度，不得直接调用 Child Agent。
- Orchestrator 读取 Agent 能力与健康状态用于编排。
- Orchestrator 通过 Registry 获取 AgentCard、能力、健康状态和调用地址。
- Child Agent 暴露健康检查和能力声明。
- Frontend 不直接调用 Child Agent 的 AgentCard endpoint。
- Agent URL / service name 属于 Registry 或 Orchestrator 配置，不属于 Gateway 业务编排配置。
- Registry 可以由配置、数据库、服务发现或专用注册表实现。
- 本 Skill 不固定 Registry 的具体存储或发现机制。

## 15. Intent Orchestration Architecture

意图编排属于 Orchestrator Service。

规则：

- Gateway 不生成编排计划。
- Gateway 不根据关键词选择 Agent。
- Orchestrator 根据用户输入、上下文、Registry、Health、能力声明、运行约束生成计划。
- LLM Planner 输出必须可校验。
- LLM 不得直接驱动执行；计划必须经过本地校验。
- 计划执行结果必须能回溯 runId、planId、taskId、agentName。

## 16. Artifact / Runtime Capability Architecture

Artifact / Runtime Capability 是 AgentHub 核心体验之一，不是附属模块。

规则：

- 非纯文本产物必须通过 Artifact 或等价产物引用表达。
- 大产物不得塞入普通文本消息。
- Artifact 通过 type、contentRef、metadata、messageId、runId 关联上下文。
- Frontend Runtime Capability 根据 Artifact 类型或 ToolCall 渲染。
- 新增产物类型不应要求修改项目总架构。

v1.0 Sprint 示例可以包含 code、webpage、markdown；长期架构不得把这些写成唯一全集。

## 17. Execution Strategy Architecture

架构层统一使用以下策略：

- `single`：一个任务或一个主要 Agent。
- `ordered_parallel`：多个独立任务，语义上可并行，但事件输出必须保持稳定顺序和消息归属。
- `sequential`：多个有依赖任务，必须按依赖关系执行。

Sprint 中的 `parallel` 在架构契约中归一为 `ordered_parallel`。实现层可以顺序执行，也可以内部并发执行，但不得让前端消息归属混乱。

## 18. Fallback / Retry Architecture

fallback / retry 是 v1.0 架构能力，不再是 Post-MVP 禁令。

规则：

- Agent 不健康时不得被主计划选择。
- Agent 调用失败时可以尝试能力兼容的替代 Agent。
- retry 必须有上限。
- fallback 必须有上限。
- fallback 后必须记录实际执行 Agent。
- 用户侧必须收到安全、可理解的降级或失败提示。
- fallback 不得隐藏主失败原因，只能脱敏展示。

## 19. Data Layer Boundary

本 Skill 只定义数据层职责，不固定具体数据库产品。

Primary Relational Database 可承载：

- users。
- conversations。
- participants。
- messages。
- agents。
- runs。
- artifacts。
- tool calls。

Cache / Ephemeral Store 可承载：

- run transient state。
- health cache。
- online state。
- short-lived stream state。

Object Storage 可承载：

- large artifacts。
- files。
- generated packages。
- images。
- documents。

具体 MySQL / PostgreSQL / Redis / Object Storage 选择属于数据持久化或部署契约，不在本 Skill 中固化。

## 20. Deployment / Demo Architecture

v1.0 架构必须支持可运行 Demo。

交付边界：

- docker compose 一键启动。
- frontend、gateway、orchestrator、若干 child agents、database 等服务可按部署契约组合。
- 所有服务应有健康检查或等价启动验证。
- smoke test 覆盖 Gateway、Orchestrator、Agent health、Registry、核心 run 流程。
- Demo 脚本应覆盖单聊、群聊、产物预览、历史消息、错误降级。

本 Skill 只规定交付边界，不提供 Dockerfile 或 compose 实现。

## 21. AI Collaboration Documentation Architecture

AI 协作文档是 v1.0 交付的一部分。

要求：

- Skill / Contract 文档必须与当前架构一致。
- README 必须说明目标架构、启动方式和 Demo 场景。
- 架构图必须体现 Gateway 与 Orchestrator 分进程。
- 关键跨服务变更必须先更新契约，再实现 Mock，再接真实服务。
- Sprint 示例应放入 v1.0 Profile，不得污染长期架构硬规则。

## 22. Architecture Decision Record Policy

重大架构变化必须写 ADR 或等价决策记录。

触发条件：

- 改变 Gateway / Orchestrator / Agent 进程边界。
- 新增跨服务协议。
- 替换主数据库。
- 引入消息队列。
- 改变 Artifact 存储方式。
- 改变公开 API 版本策略。
- 引入新的 Provider 类型。

ADR 至少记录：背景、决策、备选方案、影响、状态、日期。

## 23. Historical MVP Profile

MVP v0.1 已完成，仅作为历史兼容与回归测试基线。

历史基线包括：

- 单 Agent 最小闭环。
- 最小流式回复。
- 最小 Artifact 预览。
- 静态 Agent 配置。
- 简化鉴权。
- 简化数据层。
- 简化错误处理。

这些历史规则不得继续作为 v1.0 当前开发禁令。

## 24. v1.0 Sprint Implementation Profile

v1.0 Sprint Profile 用于承接当前 Sprint 的示例目标，不代表长期固定枚举。

允许作为 Sprint 示例的内容：

- 至少 2 个 Agent。
- 代码类 Agent 与网页类 Agent。
- code preview、web preview、markdown render。
- 单聊与群聊 Demo。
- AgentCard 注册表与健康检查。
- 结构化多轮消息。
- fallback / retry 提示。
- Docker 一键启动与 smoke test。
- README、架构图、AI 协作文档。

长期规则：新增 Agent、产物类型、Provider、存储后端或部署形态时，不应要求改写项目总架构，只应扩展对应契约和 registry / capability 描述。

## 25. Contract-first / Mock-first Workflow

项目级开发流程：

1. Architecture boundary。
2. Contract。
3. Mock。
4. Real integration。
5. Observability。
6. Review。

规则：

- 涉及公开 HTTP API，先更新 OpenAPI。
- 涉及内部服务通信，先更新内部 contract。
- 涉及 Agent 能力，先更新能力声明 / Registry 描述。
- 涉及 Artifact，先更新 Artifact / Runtime Capability 约束。
- 涉及架构边界，先更新 project-architecture。

## 26. Architecture Review Checklist

每次架构变更必须检查：

- 是否参考 v1.0 Sprint Profile。
- 是否把 MVP v0.1 放入 Historical Profile。
- 是否没有把 Sprint 示例固化为长期限制。
- Gateway 与 Orchestrator 是否是两个独立进程。
- Frontend 是否只访问 Gateway。
- Orchestrator 是否不暴露给 Frontend。
- Gateway 是否没有 import Orchestrator 业务包。
- Gateway 是否没有直接调用 Child Agent。
- Gateway 是否没有直接调用 LLM Provider。
- 是否没有写死具体 Agent 名称。
- Conversation 是否支持 single / group。
- 多 Agent message 是否有 senderName / senderId。
- Orchestrator 是否负责编排、Registry、Health、fallback、aggregation。
- 非文本产物是否通过 Artifact 或等价引用。
- Docker Demo、smoke test、README、架构图、AI 协作文档是否有交付边界。

## 27. 完成定义

本 Skill 完成时，应满足：

- 读者能清楚理解 AgentHub 总架构。
- Gateway / Orchestrator 分进程硬规则明确。
- v1.0 Sprint 目标被吸收为当前 Profile。
- MVP v0.1 被降级为历史基线。
- 架构不绑定具体 Agent 名称。
- 架构不包含业务实现代码。
- references 与 docs/contracts 可独立支持 Review。
