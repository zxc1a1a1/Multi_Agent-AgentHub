# Gateway ↔ Orchestrator Events

**Status:** Active (v2)
**Owner:** AgentHub
**Last updated:** 2026-10-29（P0：seq/subIndex、重连与终态规则）

## Authoritative source order

1. `docs/contracts/event-replay.md`
2. `docs/contracts/agui-events.md`
3. `docs/contracts/runtime-harness.md`

## Public event identity

```text
publicEventId = (runId, seq, subIndex)
```

- `seq` 是 SQL 权威业务序号；`subIndex` 是同一内部事件映射到多个公开事件时的稳定下标。
- 重连只跳过该订阅已确认的 `(seq, subIndex)`，不能只按 seq 去重。
- 映射版本随事件返回；不兼容时返回 snapshot/resync，不静默重启任务。

## Event sequence

典型单 Agent：

```text
RUN_STARTED
STATE_UPDATE {phase:"planning"}
STATE_UPDATE {activeAgent:"code-agent"}
TEXT_MESSAGE_START
TEXT_MESSAGE_CONTENT ...
TEXT_MESSAGE_END
TOOL_CALL_START / TOOL_CALL_ARGS / TOOL_CALL_END
RUN_FINISHED
```

多 Agent：

```text
RUN_STARTED
STATE_UPDATE {phase:"planning", mode:"parallel|sequential"}
STATE_UPDATE {activeAgent:"web-agent"}
... web-agent events ...
STATE_UPDATE {activeAgent:"code-agent"}
... code-agent events ...
RUN_FINISHED
```

审批：

```text
STATE_UPDATE {phase:"awaiting_approval", approvalId, toolCallId}
→ 恢复后 STATE_UPDATE {phase:"executing"}
```

## Ordering rules

- Every `TEXT_MESSAGE_CONTENT` must follow a `TEXT_MESSAGE_START` for the same message id.
- Every `TOOL_CALL_ARGS` / `TOOL_CALL_END` must reference an active `TOOL_CALL_START`.
- `RUN_FINISHED` 精确一次；用户取消为 `RUN_FINISHED(status=cancelled)` + `STATE_UPDATE(cancelled)`，不是 `RUN_ERROR`。
- `RUN_ERROR` 终止运行，除非 metadata 明确 retryable。
- 终态发送前，Gateway 必须确认对应消息投影已提交，保证随后读历史不缺失。
- 断连不取消运行；重连从最后实际消费游标续传。
- 慢客户端有界队列 + 写 deadline；队列满断开该订阅，不阻塞其他订阅。
