# Tool Call Events

## 事件

```text
TOOL_CALL_START
TOOL_CALL_ARGS
TOOL_CALL_END
```

## 作用

Tool Call 用于让前端渲染某类产物或交互控件。

本文件只定义传输过程，不定义具体工具参数 Schema。

## TOOL_CALL_START

必须包含：

- `toolCallId`
- `toolName`
- `messageId`

## TOOL_CALL_ARGS

新实现优先使用 `delta`，兼容旧字段 `content`。

前端必须按 `toolCallId` 拼接参数片段。

拼接完成后必须是合法 JSON。

## TOOL_CALL_END

表示参数传输结束。

前端只能在收到 `TOOL_CALL_END` 后解析并执行。

## 规则

- 未知 `toolName` 不得导致前端崩溃。
- malformed args 不得导致前端崩溃。
- `toolCallId` 不得复用。
- Tool Call 应归属到某条 `messageId`。
