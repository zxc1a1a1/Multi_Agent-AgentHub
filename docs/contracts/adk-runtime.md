# AgentHub ADK Runtime Contract

## 1. Contract 目的

本文是 AgentHub 内部 ADK Runtime 的事实源文档。

ADK Runtime 是 Child Agent 内部运行时，用于统一：

- config 加载。
- AgentCard 生成 / 校验。
- A2A Server 暴露。
- Task Handler 调用。
- Runtime Context API。
- 流式文本输出。
- Artifact 输出。
- LLMClient 生命周期。
- 工具权限。
- 错误脱敏。
- 日志追踪。

## 2. 非目标

ADK Runtime 不负责：

- Gateway API。
- Orchestrator Planner。
- AG-UI Event Schema。
- 前端渲染。
- Artifact 持久化策略。
- LLM Provider 供应商细节。

## 3. Runtime Boundary

```text
Handler → ADK Runtime → A2A Server → Orchestrator → AG-UI Converter → Frontend
```

## 4. Version Profile

当前 profile：

```text
v1.0-sprint
generic-child-agent-runtime
```

要求：

- 支持 2+ Child Agents。
- 不固定 Agent 名称。
- Agent 能力由 config.yaml / AgentCard 声明。
- Handler 不直接输出 AG-UI。
- Runtime 负责映射 A2A。

## 5. Required Runtime APIs

```go
ctx.Context() context.Context
ctx.StreamText(chunk string) error
ctx.AddArtifact(artifact adk.Artifact) error
ctx.Fail(err error) error
ctx.Metadata() map[string]string
ctx.Logger() Logger
```

## 6. Runtime Mapping

| Runtime API | A2A Event |
|---|---|
| handler start | status working |
| StreamText | text |
| AddArtifact | artifact |
| Fail / error | status failed |
| return nil | status completed |

## 7. Artifact

Artifact 类型由 `artifact-contract` 决定。

Runtime 不得用 agentName 判断 Artifact 类型。

## 8. Security

- 不得泄漏 secret。
- 不得在 AgentCard 暴露 secret。
- 不得在 `/health` 暴露 secret。
- 不得把 provider 原始敏感错误返回用户。
- 工具默认关闭。

## 9. Required Tests

- config load test。
- AgentCard generation test。
- Handler cancellation test。
- StreamText mapping test。
- AddArtifact mapping test。
- LLMClient lifecycle test。
- secret redaction test。
