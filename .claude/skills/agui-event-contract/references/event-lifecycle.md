# Event Lifecycle

来源：`docs/contracts/agui-events.md`、`docs/contracts/agui-event-review-checklist.md`。

MVP v0.1 成功链路：

```text
RUN_STARTED
TEXT_MESSAGE_START
TEXT_MESSAGE_CONTENT*
TEXT_MESSAGE_END
TOOL_CALL_START?
TOOL_CALL_ARGS*
TOOL_CALL_END?
RUN_FINISHED
```

失败链路：

```text
RUN_STARTED
...
RUN_ERROR
```

规则：

- `RUN_ERROR` 后不应继续输出正常完成事件。
- 无 Artifact 时可以省略 `TOOL_CALL_*`。
