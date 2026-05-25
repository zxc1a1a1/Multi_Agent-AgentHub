# Tool Call Consumption

## 标准状态机

```text
TOOL_CALL_START
  → 创建 pending buffer

TOOL_CALL_ARGS
  → 按 toolCallId 追加 delta/content

TOOL_CALL_END
  → 拼接完整 args
  → JSON.parse
  → schema validation
  → registry lookup
  → safety check
  → component render / fallback
```

## 聚合规则

- 必须按 `toolCallId` 聚合。
- 不得使用单个 `currentToolCall` 假设只有一个工具调用。
- `TOOL_CALL_ARGS` 可以出现多次。
- 兼容 `delta` 与 `content`。
- END 前不得执行 capability。

## 错误处理

- 未知 `toolCallId` 安全忽略或记录错误。
- malformed JSON 不得导致页面崩溃。
- 重复 END 不得导致重复执行。
- 未知 `toolName` 必须降级。
- 不得把 `artifact.type` 直接当 `toolName`。
- 不得使用 `download` 作为 toolName 查找（已废弃，统一使用 `file_download`）。
