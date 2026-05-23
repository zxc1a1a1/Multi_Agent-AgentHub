# Tool Call 消费规则

## 1. 目的

本文定义前端如何消费 AG-UI Tool Call 生命周期。

本文件不定义 AG-UI 事件结构。

AG-UI 事件结构由 `agui-event-contract` 负责。

## 2. 生命周期

前端消费生命周期：

```text
TOOL_CALL_START
TOOL_CALL_ARGS
TOOL_CALL_END
```

或等价：

```text
ToolCallStart
ToolCallArgs
ToolCallEnd
```

## 3. 消费流程

```text
1. TOOL_CALL_START:
   创建 pending tool call。

2. TOOL_CALL_ARGS:
   按 toolCallId 追加 args chunk。

3. TOOL_CALL_END:
   标记 args 完成。

4. parse JSON:
   只在 END 后解析完整 args。

5. schema validation:
   根据 toolName 找到 parametersSchema 校验。

6. registry validation:
   检查 implemented / disabled / dangerous / confirmation。

7. render or execute:
   选择 React Component。

8. result:
   如果 blocking / interactive，返回 ToolResult。
```

## 4. Pending 状态

前端状态必须按 `toolCallId` 管理 pending calls。

不得假设同一时间只有一个 Tool Call。

推荐结构：

```ts
type PendingToolCall = {
  toolCallId: string;
  toolName: string;
  argsBuffer: string;
  status: "pending" | "completed" | "failed";
};
```

## 5. 容错规则

必须安全处理：

- unknown toolCallId。
- duplicate TOOL_CALL_END。
- malformed args JSON。
- missing TOOL_CALL_END。
- args too large。
- unknown toolName。
- implemented=false。
- component render failure。

## 6. 禁止事项

不得：

- 在 TOOL_CALL_ARGS 分片中途解析完整 JSON。
- 用单一 currentToolCall 假设永远只有一个 Tool Call。
- 重复 TOOL_CALL_END 导致重复执行。
- 参数校验失败后继续渲染。
- Tool Call 解析错误导致聊天 UI 崩溃。
