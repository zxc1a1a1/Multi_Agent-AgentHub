# Observability Review Checklist

## 通用性

- 是否没有写死具体 Agent 名称？
- 是否支持 2+ Agent？
- 是否支持 Gateway / Orchestrator 分进程？
- 是否不依赖其他 Skill 才能理解？

## Trace

- Gateway 是否生成或提取 trace context？
- Gateway 调 Orchestrator 是否传播 `traceparent`？
- Orchestrator 调下游是否继续传播 trace context？
- 日志是否包含 `traceId / requestId / runId`？

## 日志

- 是否结构化 JSON？
- event 命名是否稳定？
- 是否包含 `service / level / timestamp`？
- 是否没有只打自然语言日志？

## 错误

- 是否有稳定 `errorCode`？
- 是否有 `safeMessage`？
- raw error 是否脱敏？
- stack trace 是否没有暴露给用户？

## 指标

- 是否覆盖 latency / traffic / errors / saturation？
- metrics label 是否低基数？
- 是否没有把 `runId / user input / token` 放入 label？

## 脱敏

- 是否没有 API key / token / connection string？
- 是否没有完整 system prompt？
- 是否没有完整 LLM raw request / response？
- debug dump 是否可控、脱敏、可关闭？

## 分进程排障

- Gateway → Orchestrator 调用是否有开始/失败/结束日志？
- stream disconnect 是否能区分浏览器断连、Gateway 超时、Orchestrator 错误？
- service token 是否没有进入日志？
