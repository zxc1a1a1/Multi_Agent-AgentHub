# A2A Errors Contract

## 1. 文档目的

本文档定义 AgentHub 中 Orchestrator ↔ Child Agent 的 A2A 错误契约。

它约束：

- Child Agent 如何报告失败。
- A2A 错误如何映射为 OrchestratorError。
- A2A 错误如何最终映射为 AG-UI `RUN_ERROR`。
- 错误信息如何避免泄漏敏感数据。
- MVP v0.1 和 Post-MVP 错误处理的边界。

---

## 2. 错误边界

A2A 错误属于：

```text
Orchestrator ↔ Child Agent
```

它不属于：

```text
Frontend REST API ErrorResponse
Gateway-Orchestrator Internal Error 的完整替代品
AG-UI RUN_ERROR schema 的完整定义
```

映射关系：

```text
A2A failed
→ OrchestratorError
→ AG-UI RUN_ERROR
```

规则：

- A2A 错误不得直接暴露原始内部堆栈给 Frontend。
- A2A 错误不得包含 API key、token、内部密钥。
- Orchestrator 负责将 A2A 错误转换为用户可展示的 `RUN_ERROR`。
- Gateway handler 不直接解析 A2A 错误。

---

## 3. MVP v0.1 错误格式

MVP v0.1 允许简单字符串错误：

```json
{
  "type": "status",
  "status": "failed",
  "error": "LLM API timeout"
}
```

推荐结构化错误：

```json
{
  "type": "status",
  "status": "failed",
  "error": {
    "code": "A2A_LLM_ERROR",
    "message": "LLM API timeout",
    "retryable": false
  }
}
```

MVP v0.1 中：

- 可以先实现字符串错误。
- Contract 文档应保留结构化错误方向。
- Orchestrator 必须能把两种格式都映射为 `RUN_ERROR`。

---

## 4. 推荐错误结构

```json
{
  "code": "A2A_LLM_ERROR",
  "message": "LLM API timeout",
  "retryable": false,
  "details": {}
}
```

字段说明：

| 字段 | 类型 | 必填 | 说明 |
|---|---|---:|---|
| `code` | string | 是 | 错误码 |
| `message` | string | 是 | 可展示或可诊断消息 |
| `retryable` | boolean | 否 | 是否可重试 |
| `details` | object | 否 | 非敏感诊断信息 |

规则：

- `message` 不能包含密钥。
- `details` 不能包含 token、API key、完整 system prompt。
- `retryable` 可供 Post-MVP fallback / retry 使用。
- MVP 不强制自动 retry。

---

## 5. 推荐错误码

```text
A2A_INVALID_REQUEST
A2A_UNAUTHORIZED
A2A_AGENT_NOT_READY
A2A_TASK_NOT_FOUND
A2A_TASK_CANCELLED
A2A_LLM_ERROR
A2A_TOOL_ERROR
A2A_ARTIFACT_ERROR
A2A_STREAM_INTERRUPTED
A2A_TIMEOUT
A2A_INTERNAL
```

错误码说明：

| 错误码 | 含义 | retryable |
|---|---|---|
| `A2A_INVALID_REQUEST` | 请求格式非法 | false |
| `A2A_UNAUTHORIZED` | A2A 服务间鉴权失败 | false |
| `A2A_AGENT_NOT_READY` | Agent 未就绪 | true |
| `A2A_TASK_NOT_FOUND` | Task 不存在 | false |
| `A2A_TASK_CANCELLED` | Task 已取消 | false |
| `A2A_LLM_ERROR` | LLM 调用失败 | true |
| `A2A_TOOL_ERROR` | Agent 内部工具调用失败 | true |
| `A2A_ARTIFACT_ERROR` | Artifact 生成失败 | true |
| `A2A_STREAM_INTERRUPTED` | 流式响应中断 | true |
| `A2A_TIMEOUT` | Agent 处理超时 | true |
| `A2A_INTERNAL` | Agent 内部错误 | false |

---

## 6. A2A → OrchestratorError 映射

推荐映射：

| A2A 错误码 | Orchestrator 错误码 |
|---|---|
| `A2A_INVALID_REQUEST` | `ORCHESTRATOR_INVALID_REQUEST` |
| `A2A_AGENT_NOT_READY` | `ORCHESTRATOR_A2A_CONNECT_FAILED` |
| `A2A_LLM_ERROR` | `ORCHESTRATOR_A2A_STREAM_FAILED` |
| `A2A_STREAM_INTERRUPTED` | `ORCHESTRATOR_A2A_STREAM_FAILED` |
| `A2A_TIMEOUT` | `ORCHESTRATOR_TIMEOUT` |
| `A2A_TASK_CANCELLED` | `ORCHESTRATOR_CANCELLED` |
| `A2A_INTERNAL` | `ORCHESTRATOR_INTERNAL` |

规则：

- A2A 错误先进入 Orchestrator 边界。
- Orchestrator 再决定是否输出 AG-UI `RUN_ERROR`。
- Gateway 不直接解析 A2A 错误。
- 如果 A2A Client 无法连接 Agent，应生成 Orchestrator 层错误，而不是伪造 A2A failed event。

---

## 7. A2A → AG-UI RUN_ERROR 映射

最终错误应映射为：

```json
{
  "type": "RUN_ERROR",
  "runId": "run-001",
  "code": "ORCHESTRATOR_A2A_STREAM_FAILED",
  "message": "Agent 执行失败，请稍后重试"
}
```

规则：

- `RUN_ERROR` schema 由 `agui-event-contract` 定义。
- A2A 错误原文不应无过滤地透传给 Frontend。
- 对用户展示的 message 应简洁。
- 详细诊断写日志，但不能泄漏敏感信息。
- 如果 Orchestrator 已输出 `RUN_ERROR`，Gateway 不应重复输出冲突错误。

---

## 8. 常见错误场景

### 8.1 AgentCard 无效

触发条件：

- `/.well-known/agent.json` 不存在。
- AgentCard 缺少 `name`、`url`、`outputModes`。
- `code-agent` 不包含 `code` outputMode。

处理：

```text
不调用该 Agent
→ ORCHESTRATOR_AGENT_NOT_FOUND 或 ORCHESTRATOR_INVALID_REQUEST
→ RUN_ERROR
```

---

### 8.2 A2A 连接失败

触发条件：

- Agent 服务未启动。
- DNS / 网络不可达。
- 端口错误。
- TLS / service auth 失败。

处理：

```text
ORCHESTRATOR_A2A_CONNECT_FAILED
→ RUN_ERROR
```

---

### 8.3 LLM 调用失败

触发条件：

- LLM API timeout。
- LLM quota exceeded。
- LLM API key 无效。
- Agent 内部 LLM client 报错。

Agent 输出：

```json
{
  "type": "status",
  "status": "failed",
  "error": {
    "code": "A2A_LLM_ERROR",
    "message": "LLM 调用失败",
    "retryable": true
  }
}
```

Orchestrator 映射：

```text
ORCHESTRATOR_A2A_STREAM_FAILED
→ RUN_ERROR
```

---

### 8.4 Artifact 生成失败

触发条件：

- 代码块解析失败。
- Artifact metadata 缺少 language。
- Artifact 内容为空。
- Artifact type 不受支持。

Agent 可选择：

```text
输出文本回复 + 不输出 Artifact
```

或：

```text
status: failed + A2A_ARTIFACT_ERROR
```

MVP 建议：

- 如果文本回复已经完成，但 Artifact 解析失败，可以不输出 Artifact，并记录错误。
- 如果 Artifact 是 Demo 必须产物，则输出 `failed` 并映射为 `RUN_ERROR`。

---

### 8.5 Stream 中断

触发条件：

- Agent 进程崩溃。
- 网络断开。
- SSE 格式损坏。
- context cancelled。

处理：

```text
A2A_STREAM_INTERRUPTED
→ ORCHESTRATOR_A2A_STREAM_FAILED
→ RUN_ERROR
```

---

## 9. 安全规则

错误信息不得包含：

- API key。
- Authorization token。
- service token。
- 数据库连接串。
- 完整 system prompt。
- 内部堆栈。
- 私有路径。
- 用户隐私数据。

错误信息可以包含：

- 错误码。
- 简短 message。
- retryable。
- traceId。
- taskId。
- agentName。

---

## 10. Trace 规则

A2A 错误日志必须能关联：

```text
traceId
runId
threadId
taskId
agentName
errorCode
```

规则：

- Child Agent 日志包含 `traceId` 和 `taskId`。
- Orchestrator 日志包含 `traceId`、`runId`、`agentName`。
- `RUN_ERROR` 可以包含安全的 `code` 和 `message`。
- 用户可见错误不应展示内部 trace 细节，除非调试模式明确开启。

---

## 11. Mock Error 规则

Mock A2A Agent 必须支持模拟：

- `A2A_INVALID_REQUEST`
- `A2A_LLM_ERROR`
- `A2A_STREAM_INTERRUPTED`
- `A2A_TIMEOUT`
- `A2A_INTERNAL`

Mock 错误也必须：

- 使用合法 `status: failed` event。
- 不输出未定义错误格式。
- 能被 Orchestrator 转换为 `RUN_ERROR`。
- 不直接输出 AG-UI Event。

---

## 12. Review Checklist

- [ ] A2A failed 是否能映射为 `RUN_ERROR`？
- [ ] 错误格式是否支持字符串和结构化对象？
- [ ] 是否定义推荐错误码？
- [ ] 是否说明 retryable？
- [ ] 是否禁止泄漏 API key / token / 堆栈？
- [ ] 是否定义 A2A → OrchestratorError 映射？
- [ ] 是否定义 OrchestratorError → AG-UI `RUN_ERROR` 映射？
- [ ] 是否覆盖 AgentCard 无效？
- [ ] 是否覆盖 A2A 连接失败？
- [ ] 是否覆盖 LLM 调用失败？
- [ ] 是否覆盖 Artifact 生成失败？
- [ ] 是否覆盖 Stream 中断？
- [ ] 是否包含 traceId / runId / taskId / agentName？
- [ ] Mock Agent 是否能模拟错误？
