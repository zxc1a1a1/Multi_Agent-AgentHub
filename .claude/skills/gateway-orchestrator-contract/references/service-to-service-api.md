# 服务间 API

Gateway 调用 Orchestrator 必须使用内部 API。

## 必需 Endpoint

```text
GET  /health
POST /internal/orchestrator/runs/stream
POST /internal/orchestrator/runs/{runId}/cancel
```

## 可选 Endpoint

```text
GET  /internal/orchestrator/runs/{runId}
```

## 事件映射边界

Gateway 调用 Orchestrator stream endpoint 时，Orchestrator 返回 `OrchestratorStreamEvent`（内部事件，snake_case）。Gateway / ProtocolConverter 负责将其映射为 AG-UI Event（前端 SSE 事件，UPPER_SNAKE_CASE）。

映射表：

| OrchestratorStreamEvent | AG-UI Event (SSE) |
|---|---|
| `run_started` | `RUN_STARTED` |
| `state_update` | `STATE_UPDATE` |
| `message_start` | `TEXT_MESSAGE_START` |
| `message_delta` | `TEXT_MESSAGE_CONTENT` |
| `message_end` | `TEXT_MESSAGE_END` |
| `tool_call_start` | `TOOL_CALL_START` |
| `tool_call_args` | `TOOL_CALL_ARGS` |
| `tool_call_end` | `TOOL_CALL_END` |
| `run_finished` | `RUN_FINISHED` |
| `run_error` | `RUN_ERROR` |

Child Agent A2A event 不得绕过 Orchestrator 直接透传给 Gateway 或 Frontend。

## 规则

- 所有 `/internal/*` endpoint 只允许内部服务访问。
- 必须有服务间鉴权。
- 必须有 timeout。
- 必须传递 trace headers。
- 请求和响应必须 JSON 可序列化。
- 不得传递 HTTP context、Go channel、DB handle 或函数指针。
- Orchestrator 不得把内部实现类型暴露为 API 字段。
- Orchestrator stream endpoint 只输出 `OrchestratorStreamEvent`，不直接输出 AG-UI Event 名称。。

## 推荐 Headers

```text
X-Request-Id
X-Trace-Id
X-Run-Id
X-Conversation-Id
X-Deadline-Ms
Authorization: Bearer <internal-service-token>
```
