# runtime-api

## 目的

本文定义 AgentHub ADK Runtime Context API。

Handler 只能通过 Context API 与 Runtime 交互，不直接操作 A2A、SSE 或 AG-UI。

## v1.0 必须支持

```go
ctx.Context() context.Context
ctx.StreamText(chunk string) error
ctx.AddArtifact(artifact adk.Artifact) error
ctx.Fail(err error) error
ctx.Metadata() map[string]string
ctx.Logger() Logger
```

## API 语义

### ctx.Context()

返回任务级 context。

规则：

- Handler 必须监听取消。
- LLM 请求必须使用该 context。
- 工具调用必须使用该 context。

### ctx.StreamText(chunk)

输出自然语言文本流。

规则：

- 只用于文本。
- 不作为大型产物的唯一事实源。
- Runtime 将其转成 A2A text event。

### ctx.AddArtifact(artifact)

输出结构化产物。

规则：

- Artifact 必须符合 artifact-contract。
- Runtime 将其转成 A2A artifact event。
- Handler 不得直接构造前端 Tool Call。

### ctx.Fail(err)

输出失败状态。

规则：

- 错误必须脱敏。
- Runtime 将其转成 A2A failed status。

### ctx.Metadata()

返回只读 metadata。

推荐包含：

```text
traceId
runId
taskId
agentName
capabilityId
```
`skillId` 如保留，只能作为 `capabilityId` 的历史别名。

### ctx.Logger()

返回带上下文字段的 logger。

日志必须自动带 traceId / taskId / agentName。

## 可选扩展

```go
ctx.Tool(name string) (Tool, bool)
ctx.EmitProgress(state map[string]any) error
ctx.SaveState(key string, value any) error
ctx.LoadState(key string) (any, bool)
```

## 禁止事项

- Context API 不得暴露 HTTP response writer。
- Context API 不得暴露前端连接对象。
- Handler 不得绕过 Context 直接写 A2A stream。
- Handler 不得直接写数据库。
