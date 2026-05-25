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

规则：

- 不触发 LLM。
- 不执行昂贵工具。
- 不泄漏环境变量。
- unhealthy Agent 不进入 Planner。

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
