# API 设计说明

本项目后端采用 REST-like API + 协议接口混合设计：

- 普通业务资源遵循 RESTful 规则。
- AG-UI 使用 SSE 事件流。
- A2A 使用 Agent Card 和后续 A2A SDK 暴露的协议接口。
- A2UI 作为 Agent 输出的声明式 UI schema，跟随 Agent 运行结果返回。

## 当前 API

### Health Check

```http
GET /healthz
```

响应：

```json
{
  "status": "ok"
}
```

### A2A Agent Card

```http
GET /.well-known/agent-card.json
```

响应：

```json
{
  "name": "go-react-root-agent",
  "description": "Root agent for React + Go multi-agent framework starter.",
  "url": "http://localhost:8080/api/v1/a2a",
  "version": "0.1.0",
  "skills": []
}
```

### Create Agent Run

```http
POST /api/v1/agent-runs
Content-Type: application/json
```

请求：

```json
{
  "message": "帮我规划多 Agent 开发任务"
}
```

响应：

```json
{
  "answer": "...",
  "surface": {
    "id": "starter-surface",
    "title": "Agent 输出的 A2UI 示例",
    "components": []
  }
}
```

### Stream Agent Run

```http
POST /api/v1/agent-runs:stream
Content-Type: application/json
Accept: text/event-stream
```

请求：

```json
{
  "message": "帮我规划多 Agent 开发任务"
}
```

响应为 SSE，每个 frame 的 `data` 是 AG-UI 事件：

```text
data: {"type":"run_started","runId":"run_..."}

data: {"type":"text_message_content","delta":"..."}

data: {"type":"a2ui_surface","surface":{...}}

data: {"type":"run_finished"}
```

## 后续建议资源

随着项目发展，可以逐步补充这些资源：

```http
GET    /api/v1/agents
GET    /api/v1/agents/{agentId}
PATCH  /api/v1/agents/{agentId}
GET    /api/v1/agent-runs
GET    /api/v1/agent-runs/{runId}
POST   /api/v1/agent-runs/{runId}:cancel
GET    /api/v1/tools
GET    /api/v1/tools/{toolId}
```
