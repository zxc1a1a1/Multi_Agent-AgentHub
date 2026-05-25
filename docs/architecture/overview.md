# AgentHub Architecture Overview

AgentHub v1.0 是 IM 聊天式多 Agent 协作平台。用户通过单聊或群聊提出需求，Gateway 负责公开入口，Orchestrator 独立负责编排，Child Agent 提供通用能力，Data Layer 承载持久化与附加资源。

核心原则：

- Gateway 与 Orchestrator 分进程。
- 不固定具体 Agent 名称。
- Conversation 是一级对象。
- Orchestrator 负责多 Agent 编排。
- Artifact 与 Runtime Capability 是丰富产物体验的核心。
- Sprint 示例进入 v1.0 Profile，不成为长期硬规则。
