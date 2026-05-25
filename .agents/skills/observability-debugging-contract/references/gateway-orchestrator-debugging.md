# Gateway-Orchestrator Debugging

## 目的

定义 Gateway Service 与 Orchestrator Service 分进程后的排障规则。

## 必须事件

- `gateway.orchestrator.call_started`
- `gateway.orchestrator.call_failed`
- `gateway.orchestrator.stream_connected`
- `gateway.orchestrator.stream_disconnected`
- `orchestrator.run.accepted`
- `orchestrator.stream.event_sent`
- `orchestrator.run.cancelled`

## 必须字段

- `traceId`
- `requestId`
- `runId`
- `orchestratorUrl`
- `durationMs`
- `statusCode`
- `errorCode`
- `safeMessage`

## 排障重点

- 区分浏览器断连、Gateway 超时、Orchestrator 错误。
- 区分 Gateway 无法连接 Orchestrator 与 Orchestrator 内部 run 失败。
- service token 不得进入日志。
- Gateway 不应只记录“stream failed”，必须记录失败阶段。
