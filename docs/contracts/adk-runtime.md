# AgentHub ADK Runtime Contract

版本：v0.1-mvp  
适用项目：AgentHub - 多 Agent 协作平台  
适用阶段：MVP + 后续正式开发演进  
事实源文件：

```text
docs/contracts/adk-runtime.md
docs/contracts/agent-config.md
```

## 1. 目的

本文定义 AgentHub ADK Runtime 的项目级契约。

本文中的 ADK Runtime 指：

```text
AgentHub ADK Runtime
```

它是 AgentHub 项目内部用于开发子 Agent 的 Runtime 抽象，不等同于 Google Agent Development Kit。

本文约束的边界是：

```text
AgentHub ADK Runtime
→ A2A Server
→ AgentCard
→ Orchestrator
```

## 2. 官方约束优先

涉及 A2A 协议时，官方 A2A Specification 优先。

本项目 MVP 端点是当前落地兼容层，不代表官方最新标准。

长期目标是对齐官方 A2A 的 AgentCard、Task、Message、Artifact、Streaming、Versioning、安全和错误语义。

## 3. Contract first 规则

任何新增、修改或删除 Runtime API、AgentCard、A2A Server、Task handler、Artifact 输出或工具权限前，必须先更新：

```text
docs/contracts/adk-runtime.md
docs/contracts/agent-config.md
```

未更新 contract 的实现变更不得接受。

## 4. 阶段演进规则

### 4.1 MVP 阶段

MVP 阶段只实现：

```text
code-agent
```

MVP 阶段只强制：

```text
ctx.StreamText(chunk)
ctx.AddArtifact(artifact)
```

MVP 阶段只输出：

```text
artifact.type = code
```

MVP 阶段使用项目兼容端点：

```text
GET    /health
GET    /.well-known/agent.json
POST   /a2a/tasks/send
POST   /a2a/tasks/sendSubscribe
GET    /a2a/tasks/:id
DELETE /a2a/tasks/:id/cancel
```

### 4.2 正式开发阶段

新增子 Agent 前，必须更新本文和 `agent-config.md`。

新增 Agent 必须声明：

- Agent 身份。
- AgentCard。
- skills。
- inputModes。
- outputModes。
- Artifact 类型。
- streaming 能力。
- 工具能力。
- 权限边界。
- 错误处理策略。

### 4.3 A2A 兼容阶段

正式开发阶段应逐步支持：

```text
/.well-known/agent-card.json
```

并可继续保留：

```text
/.well-known/agent.json
```

作为兼容路径。

同时应逐步支持或映射官方 A2A message send、message stream、task get、task cancel 等操作。

## 5. 标准 Agent 目录

长期标准目录：

```text
agents/{name}/
  main.go
  handler.go
  tools/
  config.yaml
  Dockerfile
```

MVP 目录：

```text
agents/adk/
agents/code-agent/
  main.go
  handler.go
  config.yaml
  Dockerfile
```

## 6. Runtime API

长期 Runtime API 可以包括：

```text
ctx.StreamText(chunk)
ctx.AddArtifact(artifact)
ctx.Fail(error)
ctx.Metadata()
ctx.Tools()
ctx.Logger()
```

MVP 阶段只强制：

```text
ctx.StreamText(chunk)
ctx.AddArtifact(artifact)
```

## 7. Task handler

推荐抽象：

```text
HandleTask(ctx, task) error
```

handler 负责：

- 读取用户输入。
- 读取上下文。
- 调用 LLM 或工具。
- 流式输出文本。
- 添加 Artifact。
- 尊重取消信号。
- 返回用户安全错误。

handler 不得：

- 直接返回 AG-UI 事件。
- 直接调用 React Component。
- 绕过 A2A 返回私有格式。
- 把大 Artifact 塞进文本流。

## 8. StreamText

`ctx.StreamText` 用于输出用户可读文本。

不得用于输出：

- 大型 Artifact。
- secret。
- stack trace。
- 内部路径。
- 私有文件。
- 对象存储私有地址。

## 9. AddArtifact

`ctx.AddArtifact` 用于添加任务产物。

MVP 阶段只允许：

```text
artifact.type = code
```

MVP code Artifact：

```json
{
  "type": "code",
  "title": "main.go",
  "content": "...",
  "metadata": {
    "language": "go"
  }
}
```

Artifact schema 和存储策略由 `artifact-contract` 负责。

## 10. 工具权限

MVP 阶段 `code-agent` 默认：

```text
tools.enabled = false
```

正式开发阶段启用工具前，必须定义：

- input schema。
- output schema。
- permissions。
- timeout。
- side effects。
- dangerous。
- requiresConfirmation。

危险工具必须遵守 `security-boundary-contract`。

## 11. 禁止事项

不得：

- 把 AgentHub ADK Runtime 当成 Google ADK。
- 把 Google ADK API 当成本项目 Runtime API。
- 让子 Agent 直接返回 AG-UI 事件。
- 让子 Agent 直接调用前端 Runtime Skill。
- 让子 Agent 绕过 A2A 返回私有格式。
- 在 AgentCard 中暴露 secret、内部路径或内部服务地址。
- 把 `ctx.StreamText` 当成 Artifact 事实源。
- 在 MVP 阶段启用非 `code-agent` 的完整子 Agent。
- 把 MVP 端点描述成官方 A2A 最新标准。
