# Gateway-Orchestrator 内部 API

## 必需 Endpoint

```text
GET  /health
POST /internal/orchestrator/runs/stream
POST /internal/orchestrator/runs/{runId}/cancel
```

## 鉴权

所有 `/internal/*` endpoint 必须使用服务间鉴权。

推荐：

```text
Authorization: Bearer <internal-service-token>
```

## Headers

```text
X-Request-Id
X-Trace-Id
X-Run-Id
X-Conversation-Id
X-Deadline-Ms
Authorization
```

## OrchestratorRequest 示例

```json
{
  "runId": "run_001",
  "conversationId": "conv_001",
  "userId": "user_001",
  "conversationType": "group",
  "messages": [],
  "history": [],
  "availableAgents": [],
  "selectedAgentNames": [],
  "mentions": [],
  "runtimeCapabilities": [],
  "planningMode": "auto",
  "traceId": "trace_001",
  "requestId": "req_001",
  "deadlineMs": 120000,
  "metadata": {}
}
```

## 规则

- JSON 字段使用 camelCase。
- 不传 HTTP context。
- 不传 DB handle。
- 不传 Go channel。
- 不传用户原始 Authorization token。
- 不传 LLM API key。
