# runtime-review-checklist

## ADK Runtime Review Checklist

### 1. Agent 结构

- [ ] Agent 使用标准目录结构。
- [ ] Agent 复用 `agents/adk/` Runtime。
- [ ] Agent 名称没有写死在 Runtime 通用代码中。

### 2. config.yaml

- [ ] 包含 name / description / version。
- [ ] 包含 inputModes / outputModes / skills。
- [ ] permissions 默认最小权限。
- [ ] 不包含 secret。

### 3. AgentCard

- [ ] 可由 config 生成或校验。
- [ ] 与 Handler 实际能力一致。
- [ ] skills[].id 稳定。
- [ ] outputModes 真实。

### 4. Handler

- [ ] 接收结构化消息。
- [ ] 尊重 context cancellation。
- [ ] 使用 `ctx.StreamText` 输出文本。
- [ ] 使用 `ctx.AddArtifact` 输出产物。
- [ ] 不直接写 HTTP / SSE。
- [ ] 不直接输出 AG-UI。
- [ ] 错误可控且脱敏。

### 5. LLMClient

- [ ] 启动时创建。
- [ ] HTTP Client 有 timeout。
- [ ] 通过依赖注入进入 Handler。
- [ ] secret 来自环境变量或受控 secret。

### 6. Artifact

- [ ] ArtifactDraft 类型由 outputModes 和 artifact-contract 决定。
- [ ] 不用 agentName 判断 ArtifactDraft 类型。
- [ ] ArtifactDraft 字段只包含 `type`、`title`、`content`（或 `contentRefDraft`）、`metadata`。
- [ ] ArtifactDraft 没有包含 `artifactId`、`version`、`links.*`、`source.*`、`preview.*`、`status`、`createdAt` 等平台字段。
- [ ] Handler 不直接构造前端 Tool Call。
- [ ] 没有把 ArtifactDraft 当作 Core Artifact。

### 7. 工具

- [ ] 工具默认关闭。
- [ ] 启用工具有权限声明。
- [ ] 有 timeout。
- [ ] 危险工具需要确认。

### 8. 安全

- [ ] AgentCard 无 secret。
- [ ] /health 无 secret。
- [ ] 日志无 API key。
- [ ] 错误无内部堆栈和 system prompt。

### 9. A2A 映射

- [ ] Runtime 统一映射 A2A event。
- [ ] Handler 不构造 A2A JSON。
- [ ] completed / failed 生命周期明确。

### 10. v1.0 通用性

- [ ] 新增 Agent 不需要修改 Runtime 核心逻辑。
- [ ] 新增 Agent 不需要硬编码 agentName。
- [ ] Runtime 支持 2+ Child Agents。
