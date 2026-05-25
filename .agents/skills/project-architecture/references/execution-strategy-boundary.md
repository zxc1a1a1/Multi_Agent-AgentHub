# Execution Strategy Boundary

## 策略

- `single`：一个主要任务。
- `ordered_parallel`：多个独立任务，语义上可并行，但输出必须有稳定归属和顺序。
- `sequential`：多个有依赖任务。

## parallel 兼容

Sprint 中的 `parallel` 在架构契约中归一为 `ordered_parallel`。

## 规则

- 前端不应处理 token 级多 Agent 交错作为必需能力。
- Orchestrator 必须确保每条消息有明确 sender。
- 内部可以顺序执行或并发执行，但对外输出必须稳定。
