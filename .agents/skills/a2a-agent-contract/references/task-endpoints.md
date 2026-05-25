# A2A Task Endpoints

## 1. v1.0 必需 endpoint

每个 Child Agent 必须暴露：

```text
GET /.well-known/agent.json
GET /health
POST /a2a/tasks/sendSubscribe
```

## 2. AgentCard endpoint

```text
GET /.well-known/agent.json
```

返回 AgentCard。

规则：

- 由 Registry 拉取。
- 不暴露敏感信息。
- AgentCard 无效时，该 Agent 不参与编排。

## 3. Health endpoint

```text
GET /health
```

最小响应：

```json
{
  "status": "ok",
  "agent": "agent-name",
  "version": "0.1.0"
}
```

`/health.status` 是 A2A 原始探针状态，进入 Registry 后归一化为 `Agent.health`：

```text
/health.status = ok       → Agent.health = healthy
/health.status = degraded → Agent.health = degraded
timeout / non-2xx / invalid response → Agent.health = unhealthy
未探测                        → Agent.health = unknown
```

规则：

- 不触发 LLM。
- 不执行昂贵工具。
- 不泄漏环境变量。
- unhealthy / degraded Agent 不进入 Planner。
- `disabled` 属于 `Agent.status`（生命周期），不属于 `Agent.health`。

## 4. sendSubscribe endpoint

```text
POST /a2a/tasks/sendSubscribe
```

用于提交任务并返回流式结果。

推荐 request：

```json
{
  "id": "task-001",
  "messages": [
    {"role": "user", "content": "用户请求"}
  ],
  "metadata": {
    "runId": "run-001",
    "threadId": "conversation-001",
    "traceId": "trace-001",
    "agentName": "target-agent"
  }
}
```

`metadata.threadId` 是 `conversationId` 的 A2A 协议别名，不得视为独立会话 ID。

## 5. v1.0 暂不强制

```text
POST /a2a/tasks/send
GET /a2a/tasks/{id}
POST /a2a/tasks/{id}/cancel
GET /a2a/tasks/{id}/history
```

## 6. 边界

- Frontend 不得直接调用这些 endpoint。
- Gateway Handler 不得直接调用这些 endpoint。
- 只能由 Registry / Orchestrator / A2A Client 调用。
