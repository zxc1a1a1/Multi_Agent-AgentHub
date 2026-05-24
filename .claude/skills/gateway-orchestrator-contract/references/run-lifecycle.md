# Run Lifecycle

来源：`docs/contracts/gateway-orchestrator.md`、`docs/contracts/gateway-orchestrator-events.md`。

MVP v0.1 流程：

```text
Gateway 接收 run
→ Orchestrator 产出 RUN_STARTED
→ A2A 调用与转换
→ 文本结束
→ Tool Call flush（可选）
→ RUN_FINISHED / RUN_ERROR
```

要求：

- `runId` 贯穿请求到事件。
- 失败统一映射为 `RUN_ERROR`。
