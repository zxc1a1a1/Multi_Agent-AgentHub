# STATE_UPDATE Events

## 目的

`STATE_UPDATE` 用于展示运行过程状态，例如编排、分派、当前 Agent、重试、fallback。

## 推荐结构

```json
{
  "type": "STATE_UPDATE",
  "runId": "run-001",
  "state": {
    "phase": "planning",
    "message": "正在分析任务"
  }
}
```

## phase

推荐枚举：

```text
accepted
planning
dispatching
agent_streaming
tool_calling
retrying
finished
failed
```

## 兼容

旧实现可以把 JSON 字符串放在 `content` 中。

前端兼容：

```ts
const state = event.state ?? safeParseJSON(event.content)
```

## 规则

- `STATE_UPDATE` 不得替代文本消息。
- `STATE_UPDATE` 不得替代 Tool Call。
- `message` 必须可展示。
- 不得泄漏内部错误、密钥或堆栈。
