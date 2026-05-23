---
name: adk-runtime-contract
description: 当定义、实现、修改或审查 AgentHub 子 Agent Runtime、Agent 配置、AgentCard 生成、A2A Server 暴露、Task handler、流式输出、Artifact 输出或工具权限时，使用本 Skill。
---

# adk-runtime-contract

## 1. 目的

本 Skill 定义 AgentHub 子 Agent 的 Runtime 开发契约。

本 Skill 中的 ADK Runtime 指：

```text
AgentHub ADK Runtime
```

它是 AgentHub 项目内部用于开发子 Agent 的 Runtime 抽象，不等同于 Google Agent Development Kit。

本 Skill 约束的边界是：

```text
AgentHub ADK Runtime
→ A2A Server
→ AgentCard
→ Orchestrator
```

本 Skill 的目标是确保 Coding Agent 在开发子 Agent 时：

- 不把 AgentHub ADK Runtime 和外部 Google ADK 混淆。
- 不绕过 A2A 对外暴露私有协议。
- 不让子 Agent 直接返回 AG-UI 事件。
- 不让子 Agent 直接调用 React Frontend Runtime Skill。
- 不让 AgentCard 声明和实际 handler 能力不一致。
- 不把 secret、内部路径、内部服务地址暴露到 AgentCard、流式输出或错误信息中。
- 不把文本流输出当作 Artifact 事实源。

## 2. 官方约束优先级

涉及 A2A 协议时，官方 A2A Specification 优先于本项目文档。

官方 A2A 约束包括但不限于：

- Agent discovery。
- AgentCard。
- Task。
- Message。
- Artifact。
- Streaming。
- Protocol versioning。
- Security。
- Error semantics。

如果本项目 MVP 端点、字段或命名与官方 A2A 最新规范不一致：

```text
官方 A2A 规范 = 长期兼容目标
项目 MVP 端点 = 当前落地兼容层
```

不得把 MVP 兼容端点描述为官方最新标准。

Google ADK 官方文档只作为命名冲突和外部生态参考。AgentHub ADK Runtime 的事实源是本项目 contract。

## 3. 适用场景

当进行以下工作时，启用本 Skill：

- 新建子 Agent，例如 `code-agent`、`web-agent`、`doc-agent`。
- 修改 `agents/adk` 公共 Runtime。
- 修改子 Agent 的 `config.yaml`。
- 修改 AgentCard 生成逻辑。
- 修改 A2A Server 暴露方式。
- 修改 `/health`。
- 修改 A2A task send / streaming / get / cancel 逻辑。
- 修改 Task handler 输入输出。
- 修改 `ctx.StreamText`。
- 修改 `ctx.AddArtifact`。
- 修改工具注册和工具权限。
- 审查 AgentCard 是否泄漏敏感信息。
- 审查子 Agent 是否违反 A2A 语义。
- 审查子 Agent 是否绕过 Artifact contract。

## 4. 长期契约基线

长期架构中，每个子 Agent 必须通过 AgentHub ADK Runtime 标准接入 AgentHub。

标准子 Agent 必须具备：

- 标准目录结构。
- `config.yaml`。
- AgentCard。
- A2A Server。
- `/health`。
- Task handler。
- 流式输出能力。
- Artifact 输出能力。
- 工具注册机制。
- 工具权限边界。
- 错误处理机制。
- 安全输出边界。

标准子 Agent 目录：

```text
agents/{name}/
  main.go
  handler.go
  tools/
  config.yaml
  Dockerfile
```

本契约的事实源文件是：

项目 contract 文件位于项目根目录

```text
<repo-root>/docs/contracts/adk-runtime.md
<repo-root>/docs/contracts/agent-config.md
```

本 Skill 的辅助参考文件位于当前 Skill 目录：

```text
<current-skill-dir>/references/agent-config.md
<current-skill-dir>/references/runtime-api.md
<current-skill-dir>/references/task-handler-contract.md
<current-skill-dir>/references/streaming-output.md
<current-skill-dir>/references/artifact-output.md
<current-skill-dir>/references/tool-registration.md
<current-skill-dir>/references/a2a-compatibility.md
```

## 5. MVP 约束

MVP 阶段只实现一个子 Agent：

```text
code-agent
```

MVP 阶段允许的 Runtime 范围：

```text
agents/adk/
agents/code-agent/
```

MVP 阶段 `code-agent` 必须支持：

- 读取 `config.yaml`。
- 暴露 AgentCard。
- 暴露 A2A Server。
- 暴露 `/health`。
- 支持流式任务调用。
- 通过 `ctx.StreamText(chunk)` 输出流式文本。
- 通过 `ctx.AddArtifact(...)` 添加 `code` Artifact。
- 将代码块解析为 `artifact.type = code`。
- 不执行生成的代码。
- 不暴露 secret、stack trace、内部路径或内部服务地址。

MVP 阶段只强制以下 Runtime API：

```text
ctx.StreamText(chunk)
ctx.AddArtifact(artifact)
```

MVP 阶段只允许输出以下 Artifact：

```text
artifact.type = code
```

MVP 阶段兼容端点：

```text
GET    /health
GET    /.well-known/agent.json
POST   /a2a/tasks/send
POST   /a2a/tasks/sendSubscribe
GET    /a2a/tasks/:id
DELETE /a2a/tasks/:id/cancel
```

以上端点是 AgentHub MVP 的兼容端点，不应描述为官方 A2A 最新标准。

## 6. 阶段演进规则

### 6.1 MVP 阶段

MVP 阶段只允许实现：

```text
code-agent
```

MVP 阶段不得提前实现：

- `web-agent`。
- `doc-agent`。
- `deploy-agent`。
- 复杂工具权限系统。
- 动态 Agent Registry。
- 完整官方 A2A 全部操作。
- Object Storage Artifact 输出。
- 多 Agent 协作。
- 复杂沙箱执行。

MVP 阶段不得把 `code-agent` 的实现规则扩展成所有 Agent 的通用规则。

### 6.2 正式开发阶段

MVP 完成后，可以按需求逐步新增子 Agent。

新增任何子 Agent 前，必须先更新：

```text
docs/contracts/adk-runtime.md
docs/contracts/agent-config.md
```

新增 Agent 必须明确：

- Agent 名称。
- Agent 描述。
- AgentCard。
- `skills` 声明。
- inputModes。
- outputModes。
- 支持的 Artifact 类型。
- 是否支持 streaming。
- 是否使用 LLM。
- 是否使用工具。
- 工具权限边界。
- 安全限制。
- 错误处理策略。
- 与 Orchestrator 的调用方式。

新增 Agent 的 `AgentCard.skills` 必须和 handler 实际能力一致。

新增 Agent 的 Artifact 输出必须遵守：

```text
artifact-contract
```

新增 Agent 的工具权限必须遵守：

```text
security-boundary-contract
```

### 6.3 A2A 兼容阶段

正式开发阶段应逐步补齐官方 A2A 兼容能力。

A2A 兼容目标包括：

- 支持官方 well-known AgentCard 路径：

```text
/.well-known/agent-card.json
```

- 保留 MVP 兼容路径：

```text
/.well-known/agent.json
```

- 支持官方 A2A 版本协商或等价机制。
- 支持官方 A2A 错误语义。
- 支持官方 message send / stream / task get / task cancel 的操作映射。
- 确保 AgentCard 不包含 secret 或内部实现细节。
- 确保 Artifact 和 Message 语义分离。

## 7. 本 Skill 负责

本 Skill 负责：

- AgentHub ADK Runtime 命名边界。
- 标准子 Agent 目录结构。
- `config.yaml` 规则。
- AgentCard 生成规则。
- A2A Server 暴露规则。
- MVP A2A 兼容端点。
- 官方 A2A 兼容目标。
- Task handler 输入输出规则。
- Runtime Context API 规则。
- `ctx.StreamText` 规则。
- `ctx.AddArtifact` 规则。
- 工具注册规则。
- 工具权限边界。
- Runtime 错误处理规则。
- 子 Agent 安全输出边界。
- MVP `code-agent` 规则。
- 正式开发阶段新增子 Agent 的规则。

## 8. 本 Skill 不负责

本 Skill 不负责：

- Orchestrator 如何选择 Agent。
- 意图如何生成 ExecutionPlan。
- AG-UI 事件名称。
- AG-UI 事件字段结构。
- 前端 Runtime Skill 注册。
- React Component 实现。
- Artifact 完整 schema。
- Artifact 长期持久化策略。
- 数据库完整 DDL。
- LLM provider 抽象。
- Docker Compose 交付规则。
- 通用 Go 代码风格。
- 通用 TypeScript 代码风格。

涉及以上内容时，只引用相关 contract，不在本 Skill 中重新定义。

## 9. Contract first 规则

任何新增或修改 AgentHub ADK Runtime 行为前，必须先更新 contract。

必须优先更新：

```text
docs/contracts/adk-runtime.md
docs/contracts/agent-config.md
```

未更新 contract 的实现变更不得接受。

不得先改 Runtime 实现，再补 contract。

如果修改 AgentCard、A2A 端点、Task handler、Runtime API 或工具权限，还必须同步更新对应 reference 文件。

## 10. 必须更新的文件

修改 ADK Runtime 契约时，必须优先更新：

```text
docs/contracts/adk-runtime.md
docs/contracts/agent-config.md
```

如果涉及 Artifact 输出，还必须检查：

```text
docs/contracts/artifact-schema.md
docs/contracts/artifact.schema.json
```

如果涉及 AgentCard / A2A，还必须检查：

```text
docs/contracts/a2a-agent-card.md
docs/contracts/a2a-task.md
docs/contracts/a2a-errors.md
```

如果涉及安全或工具权限，还必须检查：

```text
docs/contracts/security-boundaries.md
docs/contracts/sandbox-policy.md
```

如果涉及 MVP 实现，可能影响：

```text
agents/adk/agent.go
agents/adk/context.go
agents/adk/server.go
agents/adk/types.go
agents/adk/llm.go
agents/code-agent/main.go
agents/code-agent/handler.go
agents/code-agent/config.yaml
agents/code-agent/Dockerfile
```

## 11. 标准 Agent 目录规则

长期标准目录：

```text
agents/{name}/
  main.go
  handler.go
  tools/
  config.yaml
  Dockerfile
```

MVP 阶段：

```text
agents/adk/
agents/code-agent/
  main.go
  handler.go
  config.yaml
  Dockerfile
```

MVP 阶段可以暂不要求复杂 `tools/` 目录。

正式开发阶段，如果 Agent 使用工具，必须建立 `tools/` 目录或等价清晰模块边界。

## 12. config.yaml 规则

`config.yaml` 是 Agent 身份、AgentCard 生成和 Runtime 行为的重要输入。

`config.yaml` 必须至少定义：

- `name`
- `displayName`
- `description`
- `version`
- `agentCard`
- `inputModes`
- `outputModes`
- `skills`
- `runtime`
- `permissions`

MVP 阶段 `code-agent` 至少声明：

```yaml
name: code-agent
displayName: Code Agent
description: Generate and explain code.
version: 0.1.0

agentCard:
  inputModes:
    - text/plain
  outputModes:
    - text/plain
  skills:
    - id: code_generate
      name: Code Generate
      description: Generate code from user instructions.

runtime:
  streaming: true
  artifacts:
    - code
  tools:
    enabled: false
  permissions:
    network: false
    filesystem: false
    shell: false
```

详细规则见：

```text
references/agent-config.md
```

## 13. AgentCard 生成规则

AgentCard 应由 `config.yaml` 和 Runtime capability 生成或校验。

AgentCard 必须准确声明：

- Agent 身份。
- Agent 描述。
- 支持的输入模式。
- 支持的输出模式。
- 支持的 skills。
- 支持的协议 / endpoint。
- 鉴权要求。
- 流式能力。
- 版本信息。

`AgentCard.skills` 必须和实际 handler 能力一致。

AgentCard 不得包含：

- API key。
- access token。
- refresh token。
- 内部服务地址。
- 内部文件路径。
- workspace 绝对路径。
- system prompt secret。
- LLM provider secret。
- 数据库连接字符串。
- 对象存储私有地址。
- 未公开工具实现细节。

## 14. A2A Server 暴露规则

AgentHub ADK Runtime 必须将子 Agent 暴露为 A2A Server。

MVP 兼容端点：

```text
GET    /health
GET    /.well-known/agent.json
POST   /a2a/tasks/send
POST   /a2a/tasks/sendSubscribe
GET    /a2a/tasks/:id
DELETE /a2a/tasks/:id/cancel
```

正式开发阶段 A2A 兼容目标：

```text
GET  /.well-known/agent-card.json
POST /message:send
POST /message:stream
GET  /tasks/{id}
POST /tasks/{id}:cancel
```

具体官方兼容策略见：

```text
references/a2a-compatibility.md
```

## 15. Task handler 规则

每个子 Agent 必须有明确 Task handler。

推荐抽象：

```text
HandleTask(ctx, task) error
```

Task handler 负责：

- 读取用户输入。
- 读取必要上下文。
- 调用 LLM 或工具。
- 通过 `ctx.StreamText` 输出文本。
- 通过 `ctx.AddArtifact` 添加 Artifact。
- 返回错误。
- 尊重取消信号。
- 不泄漏 secret。

Task handler 不得：

- 直接返回 AG-UI 事件。
- 直接调用 React Component。
- 直接决定 Orchestrator 编排策略。
- 直接写入前端 Runtime Skill registry。
- 把大 Artifact 塞进文本流。

详细规则见：

```text
references/task-handler-contract.md
```

## 16. Runtime API 规则

AgentHub ADK Runtime 可以提供以下 Context API：

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

Runtime API 必须隐藏 A2A streaming 细节，让 handler 专注任务逻辑。

Runtime API 不得隐藏安全边界。

详细规则见：

```text
references/runtime-api.md
```

## 17. StreamText 规则

`ctx.StreamText(chunk)` 用于输出流式文本。

它可以表示：

- Agent 回复文本。
- 任务进度说明。
- 解释性内容。
- 用户可读状态。

`ctx.StreamText` 不得作为 Artifact 事实源。

不得通过 `ctx.StreamText` 输出：

- 大代码包。
- 图片。
- zip。
- 大日志。
- 私有文件内容。
- API key。
- token。
- stack trace。
- 内部路径。
- 对象存储私有地址。

详细规则见：

```text
references/streaming-output.md
```

## 18. AddArtifact 规则

`ctx.AddArtifact(artifact)` 用于添加任务产物。

MVP 阶段只允许：

```text
artifact.type = code
```

MVP 示例：

```go
ctx.AddArtifact(adk.Artifact{
    Type: "code",
    Title: "main.go",
    Content: code,
    Metadata: map[string]string{
        "language": "go",
    },
})
```

Artifact schema、存储策略和预览映射由：

```text
artifact-contract
```

负责。

详细规则见：

```text
references/artifact-output.md
```

## 19. 工具注册和权限规则

子 Agent 使用工具前，必须先声明工具和权限边界。

工具必须声明：

- name
- description
- input schema
- output schema
- permissions
- timeout
- side effects
- dangerous
- requiresConfirmation

MVP 阶段 `code-agent` 默认：

```text
tools.enabled = false
```

涉及文件系统、shell、网络、部署、上传、下载、删除等能力时，必须遵守：

```text
security-boundary-contract
```

详细规则见：

```text
references/tool-registration.md
```

## 20. 错误处理规则

Runtime 错误必须用户安全。

错误信息不得包含：

- stack trace。
- API key。
- access token。
- refresh token。
- system prompt。
- 内部文件路径。
- workspace 绝对路径。
- 数据库连接字符串。
- 对象存储私有地址。
- LLM provider 原始错误详情。
- 内部服务地址。

A2A 错误语义应逐步对齐官方 A2A 规范。

MVP 阶段可以使用简化错误结构，但不得泄漏敏感信息。

## 21. 禁止事项

Coding Agent 不得：

- 把 AgentHub ADK Runtime 当成 Google ADK。
- 把 Google ADK API 当成本项目 Runtime API。
- 让子 Agent 直接返回 AG-UI 事件。
- 让子 Agent 直接调用前端 Runtime Skill。
- 让子 Agent 绕过 A2A 返回私有格式。
- 让 AgentCard.skills 和 handler 实际能力不一致。
- 在 AgentCard 中暴露 secret、内部路径或内部服务地址。
- 把 `ctx.StreamText` 当成 Artifact 事实源。
- 把大型 Artifact 塞进流式文本。
- 在 MVP 阶段启用非 `code-agent` 的完整子 Agent。
- 在 MVP 阶段启用复杂工具权限系统。
- 在没有安全边界的情况下启用 shell、filesystem、network、deploy 工具。
- 把 MVP 端点描述成官方 A2A 最新标准。
- 忽略官方 A2A versioning、AgentCard、安全和错误语义的长期兼容要求。

## 22. 相关契约

本 Skill 只引用以下契约，不重新定义它们：

- `a2a-agent-contract`：负责 A2A AgentCard、Task、Streaming、错误码和官方协议映射。
- `artifact-contract`：负责 Artifact schema、类型、生命周期、存储和预览映射。
- `intent-orchestration-contract`：负责 Orchestrator 如何选择 Agent、skill 和执行策略。
- `gateway-orchestrator-contract`：负责 Gateway 与 Orchestrator 的内部 run 协议。
- `frontend-runtime-skills-contract`：负责前端 Runtime Skill 注册、参数校验、组件映射和 ToolResult。
- `security-boundary-contract`：负责工具权限、沙箱、文件访问、命令执行、部署确认等安全边界。
- `llm-provider-contract`：负责 LLM provider、prompt 模板、API key、重试和降级。
- `data-persistence-contract`：负责数据库、Redis、对象存储和迁移策略。
- `observability-debugging-contract`：负责 traceId、runId、a2aTaskId、日志和调试规则。
