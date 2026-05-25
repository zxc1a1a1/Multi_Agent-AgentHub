---
name: adk-runtime-contract
description: "用于定义 AgentHub 内部 ADK Runtime 的通用开发契约，适用于任意 Child Agent 的配置、AgentCard 生成、A2A Server 暴露、Task Handler、Runtime Context API、流式输出、Artifact 输出、LLMClient 生命周期、工具权限、安全边界与 Review 规则。"
---

# adk-runtime-contract

## 1. Skill 目的

本 Skill 定义 AgentHub 项目内部的 **ADK Runtime Contract**。

这里的 ADK 指 AgentHub 自定义的 Go 运行时抽象，不等同于 Google ADK、LangChain、AutoGen、CrewAI 或任何外部 SDK。

它约束的是：

```text
任意 Child Agent 如何基于 AgentHub ADK Runtime 实现并接入 AgentHub。
```

本 Skill 关注：

- Agent 目录结构。
- `config.yaml` 能力声明。
- AgentCard 生成与校验。
- A2A Server 暴露。
- Task Handler 输入输出边界。
- Runtime Context API。
- 流式文本输出。
- Artifact 输出。
- LLMClient 生命周期。
- 工具注册与权限声明。
- 错误处理与脱敏。
- Trace / Log / Metrics。
- Runtime Contract Test。

一句话：

**ADK Runtime 负责让任意 Child Agent 用统一方式实现任务处理、流式输出和 Artifact 输出；对外协议仍由 A2A Contract 管，前端事件仍由 AG-UI Contract 管。**

---

## 2. 当前开发阶段

当前项目阶段：

```text
profile = v1.0-sprint
mvpStatus = completed
runtimePolicy = generic-child-agent-runtime
```

MVP v0.1 已完成，MVP 中的单 Agent、单 Artifact、硬编码路由等限制只作为历史回归基线。

v1.0 Sprint 当前目标是把 AgentHub 从单 Agent 运行时升级为可承载 **2+ Child Agents** 的通用运行时。

本 Skill 不固定 Agent 名称。

以下内容不应再被视为 Post-MVP 禁止项：

```text
2+ Child Agents
AgentCard Registry
/health 健康检查
结构化多轮消息
LLM Planner 选择 Agent
single / ordered_parallel 执行策略
fallback / retry 提示
code / webpage / markdown 等多产物输出
```

`code-agent`、`web-agent` 只能作为当前 Sprint 的样例 Agent，不得成为 ADK Runtime 的硬编码约束。

---

## 3. Runtime 边界

ADK Runtime 位于 Child Agent 内部。

```text
Child Agent Handler
        ↓
AgentHub ADK Runtime
        ↓
A2A Server
        ↓
Orchestrator
        ↓
AG-UI Event Converter
        ↓
Frontend
```

ADK Runtime 负责：

- 加载 Agent 配置。
- 校验 Agent 配置。
- 生成或校验 AgentCard。
- 初始化 LLMClient / Tool Registry / Logger。
- 创建 A2A Server。
- 创建 Task Runtime Context。
- 调用 Agent Task Handler。
- 将 `ctx.StreamText()` 转成 A2A text event。
- 将 `ctx.AddArtifact()` 转成 A2A artifact event（其中 artifact 为 ArtifactDraft）。
- 将 handler error / `ctx.Fail()` 转成 A2A failed status。
- 将 handler 成功返回转成 A2A completed status。

ADK Runtime 不负责：

- 不直接返回 AG-UI Event。
- 不直接生成 `code_preview` / `web_preview` Tool Call。
- 不直接访问前端状态。
- 不决定最终 UI 渲染方式。
- 不做 Gateway 鉴权。
- 不做 Orchestrator Planner 决策。
- 不直接访问会话数据库。
- 不绕过 A2A 协议与 Orchestrator 通信。

---

## 4. 与其他 Contract 的关系

| 内容 | 归属 |
|---|---|
| Child Agent 内部 Runtime、Context、Handler | 本 Skill |
| AgentCard / A2A Task / A2A Error 对外协议 | `a2a-agent-contract` |
| Gateway ↔ Orchestrator 内部调用 | `gateway-orchestrator-contract` |
| AG-UI Event 类型和字段 | `agui-event-contract` |
| Artifact 类型、字段、存储策略 | `artifact-contract` |
| 前端 Runtime Skill 参数和渲染能力 | `frontend-runtime-skills-contract` |
| LLM Provider 抽象和密钥策略 | `llm-provider-contract` |
| 权限、沙箱、危险能力 | `security-boundary-contract` |
| 日志、trace、错误码 | `observability-debugging-contract` |
| 测试与 Review | `testing-review-contract` |

规则：

- 本 Skill 不重复定义 A2A 字段细节。
- 本 Skill 不重复定义 Artifact 最终 Schema。
- 本 Skill 不重复定义前端 Skill 参数。
- 本 Skill 只规定 ADK Runtime 必须如何把内部行为映射到外部 Contract。

---

## 5. 设计参考原则

本项目 ADK Runtime 是自定义实现，但可借鉴成熟 Agent Runtime / Agent Protocol 的高质量设计原则：

- Agent 能力必须显式声明，而不是靠服务名猜测。
- Runtime 与协议适配层应该分离。
- Handler 不应直接写 HTTP Response 或 SSE。
- 文本流与 Artifact 应分离。
- Artifact 应有类型、标题、MIME / language、metadata。
- 工具能力必须声明权限、超时、副作用和危险等级。
- Runtime 必须支持 context cancellation。
- Runtime 日志必须可追踪，但不得泄漏密钥。
- Runtime 配置必须可被测试和审查。

这些原则用于指导 AgentHub ADK Runtime，不代表项目必须直接依赖外部 SDK。

---

## 6. Version Profiles

### 6.1 MVP v0.1 Historical Profile

MVP v0.1 已完成，历史基线为：

```text
Child Agent 数量: 1
默认样例: code-agent
Agent discovery: static config / env
Artifact: code only
Frontend Skill: code_preview only
Routing: direct routing
Runtime API: ctx.StreamText + ctx.AddArtifact
```

MVP 历史规则仍用于回归测试，但不得阻止 v1.0 开发。

### 6.2 v1.0 Sprint Generic Runtime Profile

v1.0 Sprint 当前必须支持：

```text
Child Agent 数量: 2+
Agent 名称: 不固定
Agent 能力来源: config.yaml + AgentCard
Agent discovery: AgentCard Registry
Agent health: /health
Task 输入: 结构化多轮消息
Task 输出: text streaming + artifacts
Artifact 类型: 按 artifact-contract 启用
LLMClient: 启动时初始化并注入 Handler
Execution: single + ordered_parallel
Failure: fallback / retry hint
```

任何 Child Agent 只要满足以下条件，即可接入：

- 有标准目录结构。
- 有 `config.yaml`。
- 能生成或校验 AgentCard。
- 暴露 `/health`。
- 暴露 A2A Server。
- 实现 Task Handler。
- 使用 Runtime Context API 输出 text / artifact。
- 声明 inputModes / outputModes / skills / permissions。
- 遵守错误脱敏和权限边界。
- 不直接返回 AG-UI Event。

### 6.3 Post-v1.0 Runtime Profile

Post-v1.0 可扩展：

```text
dynamic user-submitted agents
agent marketplace
A2A auth / mTLS
large artifact storage
long-running task resume
task cancellation
stateful memory service
MCP tool bridge
browser automation sandbox
shell execution sandbox
multi-modal input / output
```

---

## 7. 标准 Agent 目录结构

任何 Child Agent 推荐使用以下结构：

```text
agents/{agent-name}/
  main.go
  handler.go
  config.yaml
  Dockerfile
  README.md                # 可选
  prompts/                 # 可选
  tools/                   # 可选
  fixtures/                # 可选
```

通用 Runtime 目录：

```text
agents/adk/
  config.go
  context.go
  llm.go
  server.go
  types.go
  artifact.go
  tools.go
  errors.go
  logging.go
```

规则：

- Agent 专属业务逻辑放在 `agents/{agent-name}/`。
- Runtime 通用能力放在 `agents/adk/`。
- 新增 Agent 不得复制粘贴 Runtime 基础设施。
- 新增 Agent 应复用 `adk.LoadConfig`、`adk.NewA2AServer`、`adk.Context`、`adk.Artifact`、`adk.LLMClient`。
- Agent 名称不得写死在 Runtime 通用代码中。

---

## 8. config.yaml Contract

`config.yaml` 是 Child Agent 的本地能力声明。

它用于：

- 生成或校验 AgentCard。
- 告诉 Runtime 该 Agent 是否支持 streaming / artifacts。
- 告诉 Planner 该 Agent 可执行哪些 skill。
- 告诉 Registry 该 Agent 的 inputModes / outputModes。
- 告诉安全层该 Agent 申请了哪些权限。

### 8.1 推荐结构

```yaml
name: example-agent
displayName: Example Agent
description: What this agent can do.
version: "0.1.0"
url: "http://example-agent:8080"

runtime:
  streaming: true
  artifacts: true
  llm:
    enabled: true
    providerRef: default
  tools:
    enabled: false

agentCard:
  inputModes:
    - text
  outputModes:
    - text
    - code
  skills:
    - id: example_skill
      name: Example Skill
      description: What this skill does.
      inputTypes:
        - text
      outputTypes:
        - text
        - code

permissions:
  network: false
  filesystem: false
  shell: false
  browser: false
  deploy: false
```

### 8.2 硬性规则

- `name` 必须稳定。
- `name` 必须与 AgentCard.name 一致。
- `description` 必须描述真实能力。
- `version` 必须存在。
- `agentCard.skills[].id` 必须稳定。
- `agentCard.skills[].outputTypes` 必须与 `agentCard.outputModes` 兼容。
- `outputModes` 必须只声明 Runtime 和 Artifact Contract 支持的类型。
- `permissions` 默认全部为 false。
- 任何危险权限必须经过 `security-boundary-contract`。
- `config.yaml` 不得包含 API key、token、数据库密码、用户隐私数据。

---

## 9. AgentCard 生成 / 校验

ADK Runtime 必须支持从 `config.yaml` 生成或校验 AgentCard。

映射关系：

| config.yaml | AgentCard |
|---|---|
| `name` | `name` |
| `description` | `description` |
| `url` | `url` |
| `version` | `version` |
| `runtime.streaming` | `capabilities.streaming` |
| `runtime.artifacts` | `capabilities.artifacts` |
| `agentCard.skills` | `skills` |
| `agentCard.inputModes` | `inputModes` |
| `agentCard.outputModes` | `outputModes` |

规则：

- Handler 实际能力必须与 AgentCard 一致。
- Runtime 不得生成虚假的 `skills`。
- Runtime 不得因为 Agent 名称而自动推断能力。
- AgentCard 中不得暴露 secret。
- AgentCard contract 细节由 `a2a-agent-contract` 定义。

---

## 10. A2A Server Runtime Mapping

ADK Runtime 必须把内部 Runtime API 映射为 A2A 事件。

| Runtime 行为 | A2A 输出 |
|---|---|
| Handler 开始执行 | `status: working` |
| `ctx.StreamText(chunk)` | `text` event |
| `ctx.AddArtifact(artifact)` | `artifact` event（artifact 为 ArtifactDraft） |
| `ctx.Fail(err)` | `status: failed` |
| Handler return error | `status: failed` |
| Handler return nil | `status: completed` |
| context cancelled | `status: failed` 或中断 |

规则：

- Handler 不直接构造 A2A JSON。
- Handler 不直接写 SSE。
- Runtime 负责 A2A 序列化。
- A2A 字段细节由 `a2a-agent-contract` 管。
- Runtime 不得直接构造 AG-UI Event。

---

## 11. Task Handler Contract

每个 Child Agent 必须实现一个 Task Handler。

推荐抽象：

```go
type Handler func(ctx *adk.Context, messages []Message) error
```

实际类型可以随代码演进调整，但必须满足以下语义：

- 接收结构化 messages。
- 尊重 `ctx.Context()` 的取消信号。
- 通过依赖注入使用 LLMClient。
- 可调用显式注册的 tools。
- 通过 `ctx.StreamText()` 输出文本。
- 通过 `ctx.AddArtifact()` 输出产物。
- 出错时返回 error 或调用 `ctx.Fail()`。
- 不 panic。
- 不直接写 HTTP Response。
- 不直接处理 SSE。
- 不直接生成 AG-UI Tool Call。
- 不直接访问 Gateway 会话数据库。
- 不直接调用其他 Child Agent。

Handler 可以做：

- prompt 组装。
- LLM 调用。
- 代码块解析。
- HTML 片段解析。
- Markdown 生成。
- Artifact 构造。
- 工具调用。

Handler 不可以做：

- 鉴权。
- Planner 决策。
- Agent Registry 查询。
- A2A Client 调用。
- 前端 Runtime Skill 参数构造。
- Gateway 持久化。

---

## 12. Runtime Context API

### 12.1 v1.0 必须支持

```go
ctx.Context() context.Context
ctx.StreamText(chunk string) error
ctx.AddArtifact(artifact adk.ArtifactDraft) error
ctx.Fail(err error) error
ctx.Metadata() map[string]string
ctx.Logger() Logger
```

### 12.2 可选扩展

```go
ctx.Tool(name string) (Tool, bool)
ctx.EmitProgress(state map[string]any) error
ctx.SaveState(key string, value any) error
ctx.LoadState(key string) (any, bool)
```

### 12.3 使用规则

- `StreamText` 只输出用户可读文本。
- `AddArtifact` 输出结构化产物（ArtifactDraft，非 Core Artifact）。
- `Fail` 输出可控失败，不泄漏内部细节。
- `Metadata` 只读访问 trace / run / task / agent 信息。
- `Logger` 必须自动带上 trace 字段。
- `Tool` 只能访问已注册且授权的工具。
- Context API 不得暴露 HTTP response writer。

---

## 13. LLMClient 生命周期

v1.0 必须修复请求级重复创建 LLMClient / HTTP Client 的问题。

规则：

- LLMClient 必须在 Agent 启动时创建。
- HTTP Client 必须设置 timeout。
- Handler 通过依赖注入获得 LLMClient。
- 不得在每个请求里重复 new HTTP Client。
- 不得使用无 timeout 的 `http.DefaultClient` 调 LLM。
- LLM Provider secret 只能来自环境变量或受控 secret 配置。
- LLM 错误返回给用户前必须脱敏。
- provider、model、baseURL、timeout 应集中配置。
- LLM provider 抽象细节由 `llm-provider-contract` 管。

推荐启动模式：

```go
func main() {
    cfg := adk.MustLoadConfig("config.yaml")
    llm := adk.NewLLMClientFromEnv()
    server := adk.NewA2AServer(cfg, func(ctx *adk.Context, messages []adk.Message) error {
        return handleTask(ctx, messages, llm)
    })
    server.Run()
}
```

---

## 14. Streaming Output Rules

`ctx.StreamText()` 用于流式输出自然语言解释、步骤说明、总结和提示。

规则：

- 文本流应尽快开始，提升用户感知速度。
- 文本流可以是增量 chunk。
- 文本流不得承载大型代码文件的唯一事实源。
- 文本流不得承载大型 HTML 的唯一事实源。
- 如果内容需要前端预览，应输出 Artifact。
- Markdown 文本可作为普通 text 流输出。
- 代码块可以出现在文本里，但结构化预览必须依赖 Artifact。
- Runtime 不保证文本 chunk 边界等于语义边界。
- Orchestrator / Frontend 必须能处理任意 chunk 切分。

---

## 15. Artifact Output Rules

`ctx.AddArtifact()` 输出的是 **ArtifactDraft**，不是标准 Core Artifact。

ArtifactDraft 是 Handler 能自主提供的最小产物描述。ADK Runtime 将其序列化为 A2A artifact event，之后由 Orchestrator / ArtifactRegistry 归一化为 Core Artifact。

ADK Runtime 只负责输出 ArtifactDraft，不负责生成 `artifactId`、`mimeType`、`source.*`、`links.*`、`preview.*`、`version`、`status`、`createdAt` 等平台字段。

Artifact 是否允许，由以下契约共同决定：

```text
artifact-contract
frontend-runtime-skills-contract
security-boundary-contract
```

v1.0 Runtime 推荐支持：

```text
code
webpage
document / markdown
```

但本 Skill 不固定 Agent 名称，也不规定某个 Agent 专属某种 Artifact。

### 15.1 ArtifactDraft 结构

```go
type ArtifactDraft struct {
    Type            string
    Title           string
    Content         string
    ContentRefDraft string
    Metadata        map[string]string
}
```

规则：

- `Type` 必须存在，对应 Artifact type。
- `Title` 必须存在。
- `Content` 或 `ContentRefDraft` 至少存在一个。
- `Metadata` 应尽量提供 `language`、`mimeType` 提示。
- Handler 不得在 ArtifactDraft 中设置 `artifactId`、`version`、`links.*`、`source.*`、`preview.*`、`status`、`createdAt`。
- 任意 Agent 都可以输出其 AgentCard.outputModes 声明且 Artifact Contract 支持的 ArtifactDraft。
- 不得用 agentName 判断 Artifact 类型。
- 大型产物应作为 ArtifactDraft，不应只塞进文本流。
- Handler 不直接构造 `code_preview` / `web_preview` 参数。
- Runtime 把 ArtifactDraft 转成 A2A artifact event。
- Orchestrator 再把 ArtifactDraft 归一化为 Core Artifact，然后转成 AG-UI Tool Call。

---

## 16. Tool Registration and Permissions

工具系统默认关闭。

ADK tool registration（如 `config.yaml` 中 `tools.items[].name`）**不等于** AgentHub capabilityId。capabilityId 的事实源是 `AgentCard.skills[].id`。如需在工具与 capability 之间建立关联，应通过显式映射实现，不得直接用 tool name 作为 capabilityId。

任何 Agent 启用工具前，必须在配置中声明：

```yaml
tools:
  enabled: true
  items:
    - name: example_tool
      description: What this tool does.
      timeoutMs: 10000
      dangerous: false
      requiresConfirmation: false
      permissions:
        network: false
        filesystem: false
        shell: false
        browser: false
        deploy: false
```

工具规则：

- 未声明的工具不得调用。
- 未授权的权限不得使用。
- 工具必须有 timeout。
- 工具必须有输入校验。
- 工具必须有错误脱敏。
- 有副作用的工具必须声明 `dangerous`。
- 危险工具默认需要确认。
- `shell`、`filesystem`、`browser`、`deploy` 默认禁止。
- 工具安全边界由 `security-boundary-contract` 管。

---

## 17. Error Handling

ADK Runtime 必须把内部错误转成安全、可追踪、可恢复的错误。

规则：

- Handler error 不得直接把内部堆栈返回给用户。
- LLM provider 原始错误必须脱敏。
- API key、token、secret、system prompt 不得出现在错误中。
- 用户可见错误应简短、可理解。
- 内部日志应包含 traceId、taskId、agentName、errorCode。
- 可重试错误应标记 retryable。
- 不可重试错误应清晰说明失败阶段。
- Runtime panic 必须 recover 并转成 failed status。

推荐错误码：

```text
ADK_CONFIG_INVALID
ADK_AGENTCARD_INVALID
ADK_HANDLER_PANIC
ADK_HANDLER_ERROR
ADK_LLM_TIMEOUT
ADK_LLM_ERROR
ADK_TOOL_NOT_ALLOWED
ADK_TOOL_TIMEOUT
ADK_ARTIFACT_INVALID
ADK_CONTEXT_CANCELLED
ADK_INTERNAL
```

---

## 18. Observability

Runtime 日志必须至少包含：

```text
traceId
runId
taskId
agentName
capabilityId
durationMs
artifactCount
errorCode
```

`capabilityId` 对应 `AgentCard.skills[].id`。历史代码中如保留 `skillId` 字段，它只能是 `capabilityId` 的历史别名，不得作为独立概念存在。

规则：

- 每次 task 开始、结束、失败都应记录日志。
- 每次 Artifact 输出应记录类型和大小，不记录完整敏感内容。
- 每次工具调用应记录工具名、耗时、结果状态。
- LLM 请求应记录 provider、model、耗时、token 统计，但不记录 API key。
- 日志字段命名应稳定，便于 grep 和后续接入 OpenTelemetry。

---

## 19. Contract-first Rules

修改 ADK Runtime 前，必须先检查或更新：

```text
docs/contracts/adk-runtime.md
docs/contracts/agent-config.md
.claude/skills/adk-runtime-contract/SKILL.md
.claude/skills/adk-runtime-contract/references/*.md
```

如果改动涉及 A2A 输出，必须同步检查：

```text
docs/contracts/a2a-agent-card.md
docs/contracts/a2a-task.md
docs/contracts/a2a-errors.md
```

如果改动涉及 Artifact，必须同步检查：

```text
docs/contracts/artifact-schema.md
docs/contracts/artifact.schema.json
```

如果改动涉及前端预览，必须同步检查：

```text
docs/contracts/frontend-runtime-skills.md
```

如果改动涉及权限或工具，必须同步检查：

```text
docs/contracts/security-boundaries.md
docs/contracts/sandbox-policy.md
```

---

## 20. Example Agent Profiles

以下仅为示例，不是硬约束。

### 20.1 code-like agent

适用于代码生成、解释、审查、重构、测试生成。

推荐：

```text
inputModes: text
outputModes: text, code
skills: code_generate, code_explain, code_review, test_generate
```

### 20.2 web-like agent

适用于 HTML / CSS / JS、UI、页面、组件、响应式布局。

推荐：

```text
inputModes: text
outputModes: text, code, webpage
skills: web_generation, ui_design, responsive_layout
```

### 20.3 document-like agent

适用于文档、Markdown、总结、报告、规范说明。

推荐：

```text
inputModes: text
outputModes: text, document
skills: document_generate, markdown_write, summarize
```

### 20.4 data-like agent

适用于数据分析、SQL、表格解释、轻量图表建议。

推荐：

```text
inputModes: text
outputModes: text, code, document
skills: data_analysis, sql_generate, report_write
```

---

## 21. Review Checklist

接受任何 ADK Runtime 或 Child Agent 改动前，必须检查：

### Runtime 边界

- Handler 是否没有直接写 HTTP Response？
- Handler 是否没有直接生成 AG-UI Event？
- Handler 是否没有直接调用 Gateway API？
- Handler 是否没有直接调用其他 Child Agent？
- Runtime 是否负责 A2A 映射？

### config.yaml

- 是否存在 `config.yaml`？
- 是否包含 name / description / version？
- 是否声明 inputModes / outputModes / skills？
- 是否声明 runtime.streaming / runtime.artifacts？
- 是否没有 secret？
- permissions 是否默认最小权限？

### AgentCard

- AgentCard 是否能从 config 生成或校验？
- AgentCard 是否与 Handler 实际能力一致？
- outputModes 是否没有虚假声明？
- skills[].id 是否稳定？
- `skills[].id` 是否可被 Orchestrator 作为 `capabilityIds` 引用？

### Handler

- 是否接收结构化消息？
- 是否尊重 context cancellation？
- 是否通过 ctx.StreamText 输出文本？
- 是否通过 ctx.AddArtifact 输出产物？
- 是否返回 error 而不是 panic？
- 是否通过依赖注入使用 LLMClient？

### LLMClient

- 是否启动时创建？
- HTTP Client 是否有 timeout？
- 是否没有每个请求重复 new？
- 错误是否脱敏？
- secret 是否来自环境变量或安全配置？

### Artifact

- `ctx.AddArtifact()` 输出的 ArtifactDraft 是否只包含 `type` / `title` / `content`（或 `contentRefDraft`）/ `metadata`？
- ArtifactDraft 是否没有包含 `artifactId`、`version`、`links.*`、`source.*`、`preview.*`、`status`、`createdAt` 等平台字段？
- 是否没有用 agentName 判断 Artifact 类型？
- 是否没有直接构造前端 Tool Call？
- 是否与 artifact-contract 兼容？
- 是否没有把 ArtifactDraft 当作 Core Artifact？

### 工具与权限

- 工具是否默认关闭？
- 启用工具是否显式声明权限？
- 是否有 timeout？
- 危险工具是否 requiresConfirmation？
- 是否遵守 security-boundary-contract？
- ADK tool name 是否没有被直接当作 AgentHub capabilityId？

### 观测性

- 日志是否包含 traceId / taskId / agentName？
- 是否没有打印 API key？
- 是否没有打印完整 system prompt？
- 错误是否有 errorCode？

---

## 22. 完成定义

本 Skill 视为完成，当且仅当：

```text
.claude/skills/adk-runtime-contract/SKILL.md
```

已经明确：

- ADK Runtime 是 AgentHub 内部通用 Runtime。
- 不固定具体 Agent 名称。
- v1.0 支持 2+ Child Agents。
- config.yaml 是 Agent 本地能力声明。
- AgentCard 可由 config 生成或校验。
- Task Handler 只通过 Runtime Context API 输出。
- Runtime 负责映射到 A2A Server。
- Handler 不直接输出 AG-UI Event。
- LLMClient 启动时创建并注入 Handler。
- HTTP Client 必须有 timeout。
- Artifact 输出通用化，不绑定 agentName。
- 工具系统默认关闭，启用时必须声明权限。
- 错误必须脱敏并可追踪。
- Review Checklist 完整。

---

## References

- `references/agent-config.md`
- `references/runtime-api.md`
- `references/task-handler-contract.md`
- `references/streaming-output.md`
- `references/artifact-output.md`
- `references/tool-registration.md`
- `references/a2a-runtime-mapping.md`
- `references/llm-client-lifecycle.md`
- `references/runtime-security.md`
- `references/runtime-review-checklist.md`
