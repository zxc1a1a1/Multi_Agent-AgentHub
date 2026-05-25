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

## 规则

- 所有 `/internal/*` endpoint 只允许内部服务访问。
- 必须有服务间鉴权。
- 必须有 timeout。
- 必须传递 trace headers。
- 请求和响应必须 JSON 可序列化。
- 不得传递 HTTP context、Go channel、DB handle 或函数指针。
- Orchestrator 不得把内部实现类型暴露为 API 字段。

## 推荐 Headers

```text
X-Request-Id
X-Trace-Id
X-Run-Id
X-Conversation-Id
X-Deadline-Ms
Authorization: Bearer <internal-service-token>
```
