# Gateway to AG-UI Mapping

来源：`docs/contracts/gateway-orchestrator-events.md`、`docs/contracts/agui-events.md`。

- Gateway 不改写事件业务语义，只负责传输层包装。
- Orchestrator 负责 A2A → AG-UI 语义转换。
- Artifact flush 顺序必须保持：

```text
TEXT_MESSAGE_END → TOOL_CALL_* → RUN_FINISHED
```

- 不向前端暴露 Gateway-Orchestrator 内部协议对象。
