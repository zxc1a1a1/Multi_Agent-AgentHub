# ordered_parallel 规则

`ordered_parallel` 表示多个 task 语义上独立，可以并行或顺序执行，但对外输出必须稳定。

## 规则

- 每个 task 必须有 taskId。
- 每个 task 必须有明确 agentName。
- 输出聚合顺序必须可预测。
- 不要求 token 级并发交错。
- 如果内部并发，必须保证 messageId / taskId 隔离。
- 任一 task 失败可进入 fallback 或标记部分失败。
