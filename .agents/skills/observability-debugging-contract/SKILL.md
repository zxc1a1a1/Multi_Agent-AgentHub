---
name: observability-debugging-contract
description: "用于定义 AgentHub 全链路可观测性与排障契约，包括 Trace Context、关联 ID、结构化日志、span 命名、metrics、错误码、脱敏、debug dump、Gateway/Orchestrator 分进程排障、多 Agent 排障和 Review Checklist。本 Skill 不绑定具体 Agent。"
---

# observability-debugging-contract

## 1. Skill 目的

本 Skill 定义 AgentHub 的可观测性与排障契约。

它用于保证：

- Gateway Service、Orchestrator Service、Child Agent、LLM Provider 调用、Artifact 生成、Tool Call 消费、Frontend 流式接收等环节可以被统一追踪。
- 线上或 Demo 出现问题时，可以通过稳定 ID、结构化日志、span、metrics、错误码和 debug dump 快速定位。
- 可观测数据足够排障，但不会泄漏 API key、token、完整 system prompt、数据库连接串、对象存储签名 URL、用户隐私和内部敏感拓扑。

一句话：

**任何一次 AgentHub Run 都必须能通过 `traceId + runId + requestId` 串起前端入口、Gateway、Orchestrator、Agent、LLM、Artifact、ToolCall 和持久化结果。**

---

## 2. 独立性原则

本 Skill 独立定义 AgentHub 的可观测性与排障规则。

本 Skill 不要求读者先阅读其他 Skill。

可以出现的外部概念仅作为可观测对象名称，例如：

- Gateway Service
- Orchestrator Service
- Child Agent
- Artifact
- Tool Call
- LLM Provider
- Frontend Stream

但本 Skill 不展开这些对象的业务协议、字段 schema 或实现细节。

---

## 3. 当前阶段识别

### 3.1 MVP v0.1 Historical Profile

MVP v0.1 已完成，仅作为历史兼容和回归测试基线。

历史基线包括：

- lightweight IDs
- 结构化 JSON 日志
- 安全错误码
- 基础脱敏
- `requestId`
- `runId`
- `conversationId`
- `messageId`
- 单 Agent 链路可排障

这些历史基线不得继续限制 v1.0 及后续的多服务、多 Agent、fallback 和分布式追踪设计。

### 3.2 v1 Generic Observability Profile

当前可观测性契约必须支持：

- Gateway 与 Orchestrator 分进程
- 2+ Child Agent
- 单聊与群聊
- LLM Planner / structured plan
- Registry / health check
- `single` / `ordered_parallel` / `sequential`
- fallback / retry
- 多 assistant message
- 多 Artifact
- 多 Tool Call
- 多 LLM request
- 统一 `traceId / requestId / runId / stepId / agentTaskId`
- 结构化日志
- 错误码
- 指标
- 脱敏
- debug dump
- 排障手册

本 Profile 不要求当前必须引入完整 Grafana、Jaeger、OpenTelemetry Collector 或 Prometheus，但字段、命名和传播规则必须为分布式可观测性准备好。

---

## 4. 本 Skill 负责什么

本 Skill 负责：

- Trace Context 传播
- AgentHub 内部关联 ID 规则
- 服务级 structured logging
- span 命名和属性规则
- metrics 命名和维度规则
- error taxonomy
- safe error / user-visible error
- redaction / debug dump 脱敏
- Gateway ↔ Orchestrator 分进程排障
- 多 Agent / Run / Task / Artifact / ToolCall / LLM 请求关联
- 前端、Gateway、Orchestrator、Agent、LLM、Artifact、ToolCall 的排障路径

---

## 5. 本 Skill 不负责什么

本 Skill 不负责：

- Gateway ↔ Orchestrator 的内部 API 字段完整定义
- Agent 编排计划 schema
- 子 Agent 通信协议
- 前端事件 schema
- 前端 Runtime Skill 参数
- Artifact 完整 schema
- LLM Provider Adapter 实现
- 数据库 DDL
- Docker Compose 实现
- Go / TypeScript 代码风格

如果其他文档定义了具体业务字段，本 Skill 只要求这些字段在日志、trace、metrics 和 debug dump 中使用稳定 ID 进行关联。

---

## 6. 官方标准优先级

AgentHub 可观测性设计应优先参考以下标准与行业实践：

1. **W3C Trace Context**：跨服务传播 `traceparent` / `tracestate`。
2. **OpenTelemetry**：统一 traces、metrics、logs 的上下文传播和信号模型。
3. **OpenTelemetry Semantic Conventions**：HTTP、server、client、error 等通用属性命名。
4. **SRE 四个黄金信号**：latency、traffic、errors、saturation。
5. **OWASP Logging 安全原则**：日志足够排障，但不得泄漏敏感信息。

AgentHub 自定义字段应使用稳定命名，并避免和标准语义字段冲突。

---

## 7. Observability Signals

AgentHub 至少维护以下可观测信号：

| 信号 | 目的 | 当前要求 |
|---|---|---|
| Traces | 跨服务追踪一次 run 的路径 | 必须设计字段和传播规则 |
| Logs | 记录结构化事件、错误和关键状态 | 必须输出 JSON 结构化日志 |
| Metrics | 统计延迟、流量、错误、饱和度 | 必须定义指标命名和低基数维度 |
| Safe Errors | 向用户或上游暴露安全错误 | 必须有稳定 errorCode 和 safeMessage |
| Debug Dump | 排障快照 | 必须可控、脱敏、可关闭 |

---

## 8. ID Correlation Model

统一关联 ID：

| 字段 | 含义 |
|---|---|
| `traceId` | 跨服务的一次因果链 |
| `spanId` | 当前 span 标识 |
| `requestId` | Gateway 接收的一次外部请求 |
| `runId` | 一次 AgentHub run |
| `conversationId` | 会话标识 |
| `messageId` | 消息标识 |
| `planId` | 编排计划标识 |
| `stepId` | 编排步骤标识 |
| `agentTaskId` | AgentHub 内部子任务标识 |
| `externalTaskId` | 外部 Agent / Provider 返回的任务 ID |
| `agentName` | 实际执行的 Agent 名称，不作为能力硬编码依据 |
| `capabilityId` | 能力标识 |
| `artifactId` | 产物标识 |
| `toolCallId` | Tool Call 标识 |
| `llmRequestId` | AgentHub 内部 LLM 请求 ID |
| `providerRequestId` | Provider 返回的请求 ID |
| `errorCode` | 稳定错误码 |

规则：

- `traceId` 串联跨进程、跨服务、跨 Provider 的一次因果链。
- `requestId` 标识一次 Gateway 外部请求。
- `runId` 标识一次用户发起的 AgentHub run。
- `planId` 标识一次编排计划。
- `stepId` 标识计划中的步骤。
- `agentTaskId` 标识 AgentHub 内部子任务。
- `externalTaskId` 标识外部系统返回的任务 ID。
- `messageId` 标识前端和持久化可见的消息。
- `artifactId / toolCallId / llmRequestId` 必须能回溯到 `runId`。
- 不使用具体协议名称作为唯一通用字段，例如不要把 `a2aTaskId` 作为所有 Agent 任务的唯一字段。

---

## 9. Trace Context Policy

Trace Context 分两层：

### 9.1 标准传播层

跨 HTTP 服务边界优先传播：

- `traceparent`
- `tracestate`

规则：

- 如果外部请求没有 `traceparent`，Gateway 必须创建新的 trace。
- Gateway 调 Orchestrator 时必须传播 trace context。
- Orchestrator 调 Agent、LLM Provider、Registry、存储服务时必须继续传播或记录 trace context。
- `traceId` 不得被当作用户安全凭证。
- 不得在 `tracestate` 中塞 secret、token、完整 prompt 或用户隐私内容。

### 9.2 AgentHub 业务关联层

业务关联字段：

- `requestId`
- `runId`
- `conversationId`
- `messageId`
- `planId`
- `stepId`
- `agentTaskId`
- `artifactId`
- `toolCallId`
- `llmRequestId`

这些字段可以出现在日志、span attributes、debug dump 或内部事件中，但不得替代 `traceparent` 的标准传播职责。

---

## 10. Structured Logging Policy

所有后端服务日志必须优先使用 JSON 结构化日志。

基础字段：

```json
{
  "timestamp": "2026-05-25T00:00:00Z",
  "level": "info",
  "service": "orchestrator",
  "environment": "dev",
  "event": "run.started",
  "traceId": "trace_001",
  "spanId": "span_001",
  "requestId": "req_001",
  "runId": "run_001",
  "conversationId": "conv_001",
  "messageId": "msg_001",
  "planId": "plan_001",
  "stepId": "step_001",
  "agentTaskId": "task_001",
  "agentName": "some-agent",
  "capabilityId": "some_capability",
  "artifactId": "art_001",
  "toolCallId": "tc_001",
  "llmRequestId": "llm_001",
  "errorCode": null,
  "safeMessage": null,
  "durationMs": 123
}
```

规则：

- `event` 使用 dot.case，例如 `run.started`、`agent_task.failed`。
- `service` 必须明确，例如 `gateway`、`orchestrator`、`agent`、`frontend`、`worker`。
- `level` 使用 `debug / info / warn / error`。
- `durationMs` 使用数字。
- `errorCode` 必须稳定。
- `safeMessage` 可以面向用户展示，但不得包含敏感细节。
- 原始错误只能在受控 debug 日志中出现，且必须脱敏。
- 禁止只输出自然语言日志而缺失可关联字段。

---

## 11. Event Naming Policy

事件命名使用 dot.case，按领域分组。

推荐事件：

```text
service.started
service.health_check.failed

gateway.request.accepted
gateway.stream.opened
gateway.stream.closed
gateway.orchestrator.call_started
gateway.orchestrator.call_failed
gateway.orchestrator.stream_connected
gateway.orchestrator.stream_disconnected

orchestrator.run.accepted
run.started
plan.created
plan.validation_failed
run.strategy_selected
agent_task.started
agent_task.completed
agent_task.failed
fallback.started
fallback.completed
fallback.failed
run.completed
run.failed
run.cancelled

llm.request.started
llm.stream.started
llm.structured_output.invalid
llm.request.failed

artifact.created
artifact.failed
tool_call.started
tool_call.failed
```

规则：

- 事件名不包含具体 Agent 名称。
- 具体 Agent 名称写入 `agentName` 字段。
- 事件名必须稳定，不随文案变化。
- 同一事件在不同服务中含义必须一致。

---

## 12. Span Naming Policy

推荐 span 命名：

```text
HTTP POST /api/runs
HTTP POST /internal/orchestrator/runs/stream

gateway.call_orchestrator
gateway.forward_stream_event
gateway.persist_result

orchestrator.plan
orchestrator.validate_plan
orchestrator.execute_run
orchestrator.execute_task
orchestrator.fallback

agent.call
llm.generate
llm.stream
artifact.normalize
```

自定义属性使用 `agenthub.*` 前缀：

```text
agenthub.run_id
agenthub.conversation_id
agenthub.message_id
agenthub.plan_id
agenthub.step_id
agenthub.agent_task_id
agenthub.agent_name
agenthub.capability_id
agenthub.tool_call_id
agenthub.artifact_id
agenthub.llm_request_id
agenthub.strategy
agenthub.planning_mode
agenthub.error_code
```

规则：

- 通用 HTTP 属性优先使用 OpenTelemetry HTTP semantic conventions。
- AgentHub 自定义属性统一使用 `agenthub.*` 前缀。
- 不把用户原始输入、prompt、token 放进 span attributes。
- 高基数字段谨慎进入 metrics label，可以进入 logs 或 span attributes。

---

## 13. Metrics Policy

指标优先覆盖四类问题：

- latency
- traffic
- errors
- saturation

推荐指标：

```text
gateway_http_requests_total
gateway_http_request_duration_ms
gateway_stream_disconnects_total

orchestrator_runs_total
orchestrator_run_duration_ms
orchestrator_plan_validation_failures_total
orchestrator_agent_tasks_total
orchestrator_agent_task_duration_ms
orchestrator_fallback_attempts_total

llm_requests_total
llm_request_duration_ms
llm_stream_errors_total
llm_tokens_input_total
llm_tokens_output_total

artifact_created_total
tool_calls_total
registry_health_check_failures_total
```

允许低基数标签：

```text
service
environment
route
status
strategy
planningMode
provider
model
errorCode
agentType
capabilityCategory
```

禁止高基数或敏感标签：

```text
userId
messageId
runId
traceId
prompt
raw user input
API key
full URL with token
```

---

## 14. Error Taxonomy

错误码按领域分组：

```text
GATEWAY_*
ORCHESTRATOR_*
PLAN_*
AGENT_*
LLM_*
ARTIFACT_*
TOOL_CALL_*
STREAM_*
REGISTRY_*
PERSISTENCE_*
SECURITY_*
INTERNAL_*
```

推荐错误码：

```text
GATEWAY_BAD_REQUEST
GATEWAY_ORCHESTRATOR_UNAVAILABLE
ORCHESTRATOR_PLAN_INVALID
ORCHESTRATOR_AGENT_UNAVAILABLE
PLAN_SCHEMA_INVALID
PLAN_CAPABILITY_UNSUPPORTED
AGENT_TASK_FAILED
LLM_TIMEOUT
LLM_STRUCTURED_OUTPUT_INVALID
ARTIFACT_NORMALIZATION_FAILED
TOOL_CALL_ARGS_INVALID
STREAM_CLIENT_DISCONNECTED
REGISTRY_HEALTH_CHECK_FAILED
PERSISTENCE_WRITE_FAILED
SECURITY_REDACTION_REQUIRED
INTERNAL_UNEXPECTED
```

规则：

- 错误码稳定，不随错误文案改变。
- 用户可见错误只使用 `safeMessage`。
- 内部日志可记录 normalized cause，但必须脱敏。
- Provider raw error 不直接给用户。
- stack trace 不进入前端响应。
- `errorCode` 必须出现在日志、trace 和用户可见失败事件中。

---

## 15. Redaction Policy

禁止进入日志、trace attribute、metric label、debug dump：

```text
API key
access token / refresh token
Authorization header
service-to-service token
数据库连接串
对象存储签名 URL
完整 system prompt
完整 LLM raw request / response
完整用户隐私输入
本地绝对路径
内网拓扑
cookie
session id
```

允许保存：

```text
hash 后的 userId
截断后的摘要
errorCode
provider error category
prompt template id
model id
token usage 数字
```

规则：

- 日志必须足够排障，但不得记录敏感信息。
- 用户可见错误必须脱敏。
- debug dump 必须可控、脱敏、可关闭。
- 任何新增字段进入日志前必须经过敏感性检查。

---

## 16. Debug Dump Policy

Debug dump 是用于排障的结构化快照。

它不是：

- 完整数据库导出
- 完整 prompt dump
- Provider 原始响应归档
- 用户隐私数据导出

允许字段：

```text
traceId
requestId
runId
conversationId
planId
stepId
agentTaskId
strategy
planningMode
selectedAgentNames
event timeline
errorCode
safeMessage
durationMs
provider/model 摘要
artifact/toolCall 引用
```

禁止字段：

```text
完整 prompt
完整用户隐私输入
API key
token
原始 Provider 响应
数据库连接串
签名 URL
```

规则：

- debug dump 必须默认关闭或受控开启。
- debug dump 必须脱敏。
- debug dump 不得作为长期事实源。
- debug dump 中的引用字段必须能回溯正式日志或持久化记录。

---

## 17. Gateway-Orchestrator Process Boundary Debugging

Gateway 与 Orchestrator 是两个独立进程，因此必须具备跨进程排障能力。

关键日志事件：

```text
gateway.orchestrator.call_started
gateway.orchestrator.call_failed
gateway.orchestrator.stream_connected
gateway.orchestrator.stream_disconnected
orchestrator.run.accepted
orchestrator.stream.event_sent
orchestrator.run.cancelled
```

必须字段：

```text
traceId
requestId
runId
orchestratorUrl
durationMs
statusCode
errorCode
safeMessage
```

规则：

- Gateway 调 Orchestrator 失败必须有 `errorCode`。
- Orchestrator 内部 run 失败必须返回可关联 `runId`。
- Gateway 超时和 Orchestrator 超时要能区分。
- 浏览器断连和 Orchestrator 异常要能区分。
- service token 不得打印。
- Gateway 不应只记录“stream failed”，必须记录失败阶段。

---

## 18. Multi-Agent / Group Debugging

多 Agent 和群聊排障必须支持：

- 一个 run 多个 agent task
- 一个 run 多条 assistant message
- 一个 run 多个 artifact
- 一个 run 多个 tool call
- fallback 后实际执行 Agent 可追踪

必须字段：

```text
runId
planId
stepId
agentTaskId
agentName
capabilityId
messageId
artifactId
toolCallId
strategy
planningMode
fallbackAttempt
```

规则：

- 不同 Agent 的 message 不得只靠日志顺序区分。
- fallback 前后的 agentName 必须能区分。
- ordered_parallel 必须能排查每个 task 的开始、结束、失败和跳过原因。
- group conversation 中每条 Agent 回复必须能关联 `senderName` 或等价字段。

---

## 19. LLM / Provider Debugging

LLM 调用排障必须记录：

```text
llmRequestId
providerName
modelId
useCase
requestTimeoutMs
durationMs
tokenInput
tokenOutput
finishReason
errorCode
retryAttempt
fallbackAttempt
```

禁止记录：

```text
完整 prompt
完整用户输入
完整 Provider raw response
API key
Authorization header
```

规则：

- structured output validation 失败必须有独立错误码。
- Provider timeout 与本地 context cancelled 必须区分。
- retry 与 fallback 必须记录 attempt。
- token usage 可以用于成本统计，但不得包含敏感内容。

---

## 20. Artifact / ToolCall Debugging

Artifact 必须能通过以下字段回溯：

```text
artifactId
runId
messageId
agentName
type
status
errorCode
```

Tool Call 必须能通过以下字段回溯：

```text
toolCallId
runId
messageId
toolName
status
errorCode
```

规则：

- 大内容不进入日志。
- 只记录 type、title、mimeType、size、status、引用 ID。
- Tool Call 参数非法必须有 `TOOL_CALL_ARGS_INVALID` 或等价错误码。
- 渲染失败和生成失败必须能区分。

---

## 21. Frontend Debugging

Frontend debug log 可记录：

```text
requestId
runId
conversationId
messageId
toolCallId
eventType
streamState
errorCode
```

不得记录：

```text
API token
完整用户隐私输入
完整 HTML artifact
完整 LLM prompt
```

排查前端流式问题时至少检查：

- 是否创建 requestId / runId？
- SSE 是否打开？
- 最后收到的 eventType 是什么？
- messageId 是否稳定？
- toolCallId 是否完整？
- RUN_ERROR 是否含 errorCode？
- 浏览器断连是否在 Gateway 中被记录？

---

## 22. Contract-first 规则

修改可观测性相关实现前，应先更新：

```text
docs/contracts/observability-debugging.md
docs/contracts/error-taxonomy.md
docs/contracts/trace-context-policy.md
```

涉及日志字段、错误码、指标、debug dump 或 trace context 的变更，必须同步更新 Review Checklist。

---

## 23. Contract Test 规则

至少应覆盖：

- 日志是否 JSON 结构化。
- 日志是否包含 `traceId / requestId / runId`。
- Gateway 调 Orchestrator 时是否传播 trace context。
- Orchestrator 失败是否返回稳定 `errorCode`。
- 脱敏规则是否阻止 token / API key / connection string 出现在日志中。
- metrics label 是否不包含高基数字段。
- debug dump 是否脱敏。
- 多 Agent run 是否能关联 `agentTaskId / messageId / agentName`。

---

## 24. Review Checklist

### 通用性

- 是否没有写死具体 Agent 名称？
- 是否支持 2+ Agent？
- 是否支持 Gateway / Orchestrator 分进程？
- 是否不依赖其他 Skill 才能理解？

### Trace

- Gateway 是否生成或提取 trace context？
- Gateway 调 Orchestrator 是否传播 `traceparent`？
- Orchestrator 调下游是否继续传播 trace context？
- 日志是否包含 `traceId / requestId / runId`？

### ID

- `runId` 是否贯穿所有主链路日志？
- 多 Agent task 是否有 `agentTaskId / stepId`？
- Artifact / ToolCall / LLM 请求是否能回溯 `runId`？
- 是否没有把某个外部协议 task id 当成唯一通用字段？

### 日志

- 是否结构化 JSON？
- `event` 命名是否稳定？
- 是否包含 `service / level / timestamp`？
- 是否没有只打自然语言日志？

### 错误

- 是否有稳定 `errorCode`？
- 是否有 `safeMessage`？
- raw error 是否脱敏？
- stack trace 是否没有暴露给用户？

### 指标

- 是否覆盖 latency / traffic / errors / saturation？
- metrics label 是否低基数？
- 是否没有把 `runId / user input / token` 放入 label？

### 脱敏

- 是否没有 API key / token / connection string？
- 是否没有完整 system prompt？
- 是否没有完整 LLM raw request / response？
- debug dump 是否可控、脱敏、可关闭？

### 分进程排障

- Gateway → Orchestrator 调用是否有开始/失败/结束日志？
- stream disconnect 是否能区分浏览器断连、Gateway 超时、Orchestrator 错误？
- service token 是否没有进入日志？

---

## 25. 完成定义

本 Skill 视为完成，当且仅当：

- `SKILL.md` 为中文。
- Skill 独立可读。
- MVP v0.1 已降级为 Historical Profile。
- Gateway / Orchestrator 分进程可观测性已明确。
- 不绑定任何具体 Agent 名称。
- ID correlation model 明确。
- Trace Context 传播规则明确。
- 结构化日志字段明确。
- span 命名规则明确。
- metrics 命名和维度规则明确。
- error taxonomy 明确。
- redaction 规则明确。
- debug dump 规则明确。
- 多 Agent / 群聊 / fallback 排障规则明确。
- docs/contracts 已同步。
- Review Checklist 明确。
