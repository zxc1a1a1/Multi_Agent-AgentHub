# A2A 兼容规则

## 1. 目的

本文定义 AgentHub ADK Runtime 与 A2A 官方协议的兼容规则。

涉及 A2A 协议时，官方 A2A Specification 优先。

项目 MVP 端点是当前落地兼容层，不代表官方最新标准。

## 2. 官方优先原则

当项目文档与官方 A2A 规范冲突时：

```text
官方 A2A 规范优先
```

但 MVP 阶段可以保留项目兼容端点，以保证最小链路跑通。

## 3. AgentCard discovery

MVP 阶段必须支持：

```text
/.well-known/agent.json
```

正式开发阶段应支持官方 well-known 路径：

```text
/.well-known/agent-card.json
```

并可继续保留：

```text
/.well-known/agent.json
```

作为兼容路径。

## 4. MVP 兼容端点

MVP 阶段使用：

```text
GET    /health
GET    /.well-known/agent.json
POST   /a2a/tasks/send
POST   /a2a/tasks/sendSubscribe
GET    /a2a/tasks/:id
DELETE /a2a/tasks/:id/cancel
```

这些端点服务于 AgentHub MVP。

## 5. 正式开发兼容目标

正式开发阶段应逐步支持或映射到官方 A2A 操作：

```text
GET  /.well-known/agent-card.json
POST /message:send
POST /message:stream
GET  /tasks/{id}
POST /tasks/{id}:cancel
```

## 6. Versioning

正式开发阶段应支持 A2A 版本协商或等价机制。

客户端和服务端应能明确协议版本。

不支持的版本应返回可识别错误。

## 7. Message 与 Artifact 分离

A2A 语义中，Message 和 Artifact 有不同职责。

Message 用于交互、澄清、状态更新和任务输入。

Artifact 用于表达任务产物。

AgentHub Runtime 中：

```text
ctx.StreamText → Message / status-oriented text
ctx.AddArtifact → Artifact
```

不得把任务产物事实源塞进文本消息。

## 8. AgentCard 安全

AgentCard 可包含公开能力信息，但不得包含：

- API key。
- token。
- 内部路径。
- 内部服务地址。
- system prompt secret。
- 工具内部实现细节。
- 数据库连接字符串。
- 对象存储私有地址。

## 9. 禁止事项

不得：

- 把 MVP endpoint 描述为官方最新标准。
- 忽略官方 AgentCard discovery 的长期兼容要求。
- 忽略 A2A versioning。
- 让 AgentCard 泄漏 secret。
- 把 Message 和 Artifact 语义混用。
- 绕过 A2A 返回私有协议。
