# a2a-runtime-mapping

## 目的

本文定义 ADK Runtime 行为如何映射到 A2A Server 输出。

A2A 字段细节由 `a2a-agent-contract` 管；本文只定义 Runtime 映射关系。

## 映射表

| Runtime 行为 | A2A 输出 |
|---|---|
| Handler 被调用 | status: working |
| ctx.StreamText(chunk) | text event |
| ctx.AddArtifact(artifact) | artifact event |
| ctx.Fail(err) | status: failed |
| Handler return error | status: failed |
| Handler return nil | status: completed |
| context cancelled | failed 或 stream interrupted |

## Runtime 职责

- 统一序列化 A2A event。
- 统一处理 completed / failed 生命周期。
- 统一处理 panic recover。
- 统一处理 context cancellation。
- 统一处理 Artifact validate。

## Handler 禁止事项

- 不直接构造 A2A JSON。
- 不直接写 SSE。
- 不直接控制 HTTP flush。
- 不直接输出 AG-UI Event。

## Orchestrator 职责

- 调用 A2A Server。
- 读取 A2A stream。
- 将 A2A event 交给 ProtocolConverter。
- 处理 fallback。
- 输出 AG-UI Event。

## Review Checklist

- [ ] Runtime 是否统一映射 A2A？
- [ ] Handler 是否没有绕过 Runtime？
- [ ] Artifact 是否经过 validate？
- [ ] failed / completed 生命周期是否明确？
