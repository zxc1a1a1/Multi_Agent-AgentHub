# A2A Errors Contract

## 1. 目的

本文档定义 Child Agent 通过 A2A 返回错误时的结构、错误码、fallback 规则与脱敏要求。

## 2. 标准错误事件

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

## 3. 字段说明

| 字段 | 必填 | 说明 |
|---|---:|---|
| `type` | 是 | 固定为 `status` |
| `status` | 是 | 固定为 `failed` |
| `error.code` | 是 | 稳定错误码 |
| `error.message` | 是 | 面向上层的错误说明 |
| `error.retryable` | 是 | 是否允许 fallback / retry |

## 4. 推荐错误码

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

## 5. fallback 规则

- 如果 `retryable = true`，Orchestrator 可以选择其他 healthy Agent fallback。
- fallback 前应输出 AG-UI `STATE_UPDATE`。
- fallback 不得选择 unhealthy Agent。
- 所有候选 Agent 都失败后，输出 AG-UI `RUN_ERROR`。

## 6. AG-UI 映射

```text
A2A failed + retryable=true  → STATE_UPDATE(retrying) + fallback
A2A failed + retryable=false → RUN_ERROR
全部 fallback 失败          → RUN_ERROR
```

## 7. 脱敏要求

错误信息不得包含：

- API key
- Authorization token
- 数据库密码
- 完整 system prompt
- 内部堆栈
- 容器内部敏感路径
- 私有网络拓扑细节

## 8. Review Checklist

- [ ] 是否包含稳定错误码？
- [ ] 是否包含用户可理解 message？
- [ ] retryable 是否准确？
- [ ] 是否不会泄漏 secret？
- [ ] 是否能正确映射 fallback / RUN_ERROR？
