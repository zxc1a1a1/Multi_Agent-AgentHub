# OrchestratorStreamEvent 规则

`OrchestratorStreamEvent` 是 Orchestrator 发送给 Gateway 的内部流式事件。

## 事件类型

```text
run_started
state_update
message_start
message_delta
message_end
tool_call_start
tool_call_args
tool_call_end
run_finished
run_error
```

## 通用字段

```text
type
runId
messageId
sender
delta
state
toolCall
error
```

## 规则

- 所有事件必须包含 `runId`。
- message 类事件必须包含 `messageId`。
- 多 Agent 输出不得复用同一个 `messageId`。
- Gateway 不得接收 Child Agent 原始流。
- Gateway 不得改变事件业务语义。
- Gateway 只负责转换为对外传输格式。
- 错误事件必须使用 SafeError。

## 有序并行

ordered_parallel 场景中可以有多个 message 流。

规则：

- 每个 Agent 输出有独立 messageId。
- 事件可按 message 粒度顺序输出。
- 不要求不同 Agent token 级交错。
- 如果内部并发，Orchestrator 负责排序或隔离。
