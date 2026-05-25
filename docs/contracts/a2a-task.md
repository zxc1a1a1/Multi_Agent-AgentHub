# A2A Task Contract

## 1. 目的

本文档定义 Orchestrator 通过 A2A 调用 Child Agent 的任务请求与流式响应契约。

## 2. Endpoint

v1.0 核心 endpoint：

```text
POST /a2a/tasks/sendSubscribe
```

## 3. Request

```json
{
  "id": "task-001",
  "messages": [
    {
      "role": "user",
      "content": "用户请求"
    }
  ],
  "metadata": {
    "runId": "run-001",
    "threadId": "conversation-001",
    "traceId": "trace-001",
    "agentName": "target-agent"
  }
}
```

## 4. 字段说明

| 字段 | 必填 | 说明 |
|---|---:|---|
| `id` | 是 | A2A task id |
| `messages` | 是 | 结构化消息列表 |
| `metadata` | 否 | 追踪与上下文信息 |
| `metadata.runId` | 推荐 | AG-UI run id |
| `metadata.threadId` | 推荐 | `conversationId` 的 A2A 协议别名，不得视为独立会话 ID |
| `metadata.traceId` | 推荐 | 跨服务追踪 id |
| `metadata.agentName` | 推荐 | 目标 Agent 名称 |

## 5. Message

```json
{
  "role": "user",
  "content": "消息内容"
}
```

推荐 role：

```text
user
assistant
system
tool
```

v1.0 至少支持：

```text
user
assistant
system
```

## 6. Streaming Events

### working

```json
{"type":"status","status":"working"}
```

### text

```json
{"type":"text","content":"流式文本"}
```

### artifact

A2A streaming 中的 `event.artifact` 是 **ArtifactDraft**，不是标准 Core Artifact。

```json
{
  "type": "artifact",
  "artifact": {
    "type": "code",
    "title": "main.go",
    "content": "package main",
    "metadata": {"language": "go"}
  }
}
```

ArtifactDraft 只包含 Child Agent 能提供的字段（`type`、`title`、`content` 或 `contentRefDraft`、`metadata`）。`artifactId`、`mimeType`、`source.*`、`links.*`、`preview.*`、`version`、`status`、`createdAt` 等平台字段由 Orchestrator / ArtifactRegistry 归一化时生成。

### completed

```json
{"type":"status","status":"completed"}
```

### failed

```json
{
  "type":"status",
  "status":"failed",
  "error": {
    "code":"A2A_AGENT_ERROR",
    "message":"Agent 执行失败",
    "retryable":true
  }
}
```

## 7. 生命周期

```text
submitted → working → completed
submitted → working → failed
```

规则：

- Agent 开始处理后应输出 working。
- 可以输出多个 text。
- 可以输出多个 artifact。
- 最终必须 completed 或 failed。
- completed / failed 后不得继续输出正常内容。

## 8. Health Check 归一化

A2A `/health` endpoint 返回原始探针状态。进入 AgentHub Registry 后必须归一化：

```text
/health.status = ok       → Agent.health = healthy
/health.status = degraded → Agent.health = degraded
timeout / non-2xx / invalid response → Agent.health = unhealthy
未探测                        → Agent.health = unknown
```

`Agent.status`（生命周期/启用状态）与 `Agent.health`（健康状态）分离：
- `Agent.status`：`enabled` / `disabled` / `experimental` / `deprecated`
- `Agent.health`：`healthy` / `degraded` / `unhealthy` / `unknown`

`disabled` 属于 `Agent.status`，不属于 `Agent.health`。

## 9. 安全

metadata 不得包含：

- API key
- Authorization token
- 用户私密 token
- 数据库密码
- 完整系统 prompt
