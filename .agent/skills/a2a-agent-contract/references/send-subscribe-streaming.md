# sendSubscribe Streaming Contract

## 1. 事件语义

v1.0 必须支持：

```text
status: working
text
artifact
status: completed
status: failed
```

## 2. working

```json
{"type":"status","status":"working"}
```

表示 Agent 已接收任务并开始处理。

## 3. text

```json
{"type":"text","content":"流式文本片段"}
```

规则：

- 可以出现多次。
- 应保持顺序。
- 不应塞入大型结构化产物。

## 4. artifact

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

规则：

- Artifact 必须是结构化对象。
- Artifact 类型必须与 AgentCard.outputModes 兼容。
- Artifact 由 ProtocolConverter 转成 AG-UI Tool Call。

## 5. completed

```json
{"type":"status","status":"completed"}
```

规则：

- completed 后不应继续输出 text / artifact。
- Orchestrator 收到 completed 后 flush artifacts。

## 6. failed

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

规则：

- failed 后不应继续输出正常结果。
- retryable = true 时 Orchestrator 可以 fallback。
- 错误信息不得泄漏 secret。

## 7. 转换关系

```text
A2A working     → AG-UI TEXT_MESSAGE_START
A2A text        → AG-UI TEXT_MESSAGE_CONTENT
A2A artifact    → AG-UI TOOL_CALL_*
A2A completed   → AG-UI TEXT_MESSAGE_END
A2A failed      → fallback 或 AG-UI RUN_ERROR
```
