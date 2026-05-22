# Gateway 架构说明

## 1. Gateway 定位

Gateway 是 AgentHub 的唯一对外后端入口，使用 Go 实现。

它负责：

- REST API
- AG-UI Server
- Auth
- Session
- Conversation / Message / Agent / Artifact API
- 转发请求给 Orchestrator

Gateway 不负责意图编排和 A2A 调度。

---

## MVP v0.1 Gateway 范围

MVP v0.1 中，Gateway 可以和 Orchestrator 合并在 `server/` 一个 Go 进程中。

Gateway 负责：

- REST API。
- AG-UI SSE endpoint。
- 固定 Token 鉴权。
- 保存用户消息到 MySQL。
- 查询会话历史。
- 调用进程内 Orchestrator 模块。

Gateway 不负责：

- 直接调用 LLM。
- 直接生成 Agent 回复。
- 直接解析 A2A Artifact 为前端组件。
- 直接实现复杂意图编排。

---

## 2. 推荐目录

```text
gateway/
  cmd/
    server/
      main.go
  internal/
    handler/
      agui_handler.go
      conversation.go
      message.go
      agent.go
      artifact.go
    middleware/
      auth.go
      cors.go
      logger.go
    service/
      session.go
      artifact.go
      orchestrator_client.go
    model/
      conversation.go
      message.go
      agent.go
      artifact.go
    config/
      config.go
  go.mod
  Dockerfile
```

---

## 3. 对外 REST API

```text
GET    /api/conversations
POST   /api/conversations
GET    /api/conversations/{id}
PATCH  /api/conversations/{id}
DELETE /api/conversations/{id}
GET    /api/conversations/{id}/messages
GET    /api/agents
GET    /api/agents/{name}/card
POST   /api/agents/custom
PATCH  /api/agents/custom/{id}
DELETE /api/agents/custom/{id}
GET    /api/artifacts/{id}
GET    /api/artifacts/{id}/preview
```

---

## 4. AG-UI Endpoint

```text
POST /api/agui/run
POST /api/agui/run/{runId}/cancel
POST /api/agui/run/{runId}/tool-result
```

---

## 5. Gateway 到 Orchestrator

Gateway 使用内部 Contract 访问 Orchestrator：

```text
POST /internal/runs
POST /internal/runs/{runId}/tool-result
POST /internal/runs/{runId}/cancel
GET  /internal/runs/{runId}/events
```

---

## 6. Gateway 禁止事项

- 禁止生成 ExecutionPlan。
- 禁止直接调用 Child Agent。
- 禁止实现 A2A Client。
- 禁止在 handler 中写复杂编排逻辑。
- 禁止返回没有进入 OpenAPI 的 REST schema。
- 禁止破坏 AG-UI event contract。
