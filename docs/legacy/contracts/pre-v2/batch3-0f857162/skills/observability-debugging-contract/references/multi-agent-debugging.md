# Multi-Agent Debugging

## 目的

定义 2+ Agent、群聊、fallback、ordered_parallel 下的排障字段。

## 必须字段

- `runId`
- `planId`
- `stepId`
- `agentTaskId`
- `agentName`
- `capabilityId`
- `messageId`
- `artifactId`
- `toolCallId`
- `strategy`
- `planningMode`
- `fallbackAttempt`

## 规则

- 一个 run 中不同 Agent 的消息必须有不同 `messageId`。
- 一个 run 中不同 Agent task 必须有不同 `agentTaskId`。
- fallback 前后的实际执行 Agent 必须可追踪。
- ordered_parallel 必须能定位每个 task 的开始、结束、失败和跳过原因。
- 群聊消息归属错误必须能通过 `senderName / agentName / messageId` 排查。
