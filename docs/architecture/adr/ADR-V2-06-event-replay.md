# ADR-V2-06：SQL 事件日志 + outbox + gRPC 可恢复订阅

- 日期：2026-10-29
- 状态：已采纳（P0）
- 背景：Orchestrator 只返回流事件，运行状态在内存；浏览器断连绑定请求 context；Gateway/Orchestrator 目标协议为 gRPC 但缺 start/subscribe/control 语义。
- 决策：每个 run 分配单调 `seq`；状态/checkpoint 变化与 `execution_events`、outbox 同事务提交；outbox relay 按 run 保序投递 Redis Stream；Gateway 幂等投影后发送 AG-UI，公开事件 ID 为 `(runId, seq, subIndex)`。内部 gRPC 分离 start/subscribe/control；断开订阅不取消 durable run。
- 备选：Redis 作为唯一事件源（丢失/裁剪不可恢复）；每请求直接流式转发（无法重连/恢复）；使用框架自动 checkpoint（不满足审批/工具账本语义）。
- 后果：需要 relay 游标、SQL 水位/终态对账、投影事务、慢订阅者上限与 snapshot/resync；Redis 重启后从 SQL 恢复。
- 同步契约：`event-replay.md`、`gateway-orchestrator.md`、`gateway-orchestrator-api.md`、`gateway-orchestrator-events.md`、`agui-events.md`、`project-architecture.md`。
