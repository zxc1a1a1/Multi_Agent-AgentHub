# A2A Error Policy

## 1. 错误结构

推荐：

```json
{
  "type": "status",
  "status": "failed",
  "error": {
    "code": "A2A_AGENT_ERROR",
    "message": "Agent 执行失败",
    "retryable": true
  }
}
```

## 2. 推荐错误码

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
A2A_INTERNAL
```

## 3. fallback 规则

- retryable = true 时，Orchestrator 可以尝试 fallback。
- fallback 只能选择 healthy Agent。
- fallback 应输出 AG-UI STATE_UPDATE。
- 所有候选 Agent 都失败后，输出 AG-UI RUN_ERROR。

## 4. 脱敏规则

错误信息不得包含：

- API key
- Authorization token
- 数据库密码
- 完整 system prompt
- 内部堆栈
- 私有网络拓扑细节

## 5. Review 要点

- 是否有明确 code。
- message 是否用户可理解。
- retryable 是否准确。
- 是否不泄漏敏感信息。
- 是否能映射到 fallback / RUN_ERROR。
