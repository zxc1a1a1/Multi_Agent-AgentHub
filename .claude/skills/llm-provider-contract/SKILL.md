---
name: llm-provider-contract
description: "用于定义 AgentHub 中所有 LLM 调用的统一 Provider Adapter 契约，包括 Provider Registry、Model Registry、LLMRequest/LLMResponse、流式归一化、结构化输出、tool use 适配边界、timeout/retry/rate limit、fallback、secret、prompt 模板、usage/cost 和错误脱敏。本 Skill 不绑定具体 Agent。"
---

# llm-provider-contract

## 1. Skill 目的

本 Skill 定义 AgentHub 中所有 LLM 调用的统一 Provider Adapter 契约。

它的目标是让 AgentHub 可以在不污染业务代码的前提下，安全、可观测、可替换地接入多个 LLM Provider 与多个模型。

本 Skill 约束：

- Provider Registry
- Model Registry
- Provider Adapter 接口
- LLMRequest / LLMResponse
- LLMStreamEvent
- 流式输出归一化
- 结构化输出与本地 schema 校验
- tool use / function calling 的适配边界
- timeout / retry / rate limit
- Provider fallback / model fallback
- usage / cost 元数据
- secret 与配置
- prompt 模板
- Provider 错误归一化
- trace / logging

一句话：

**任何模型调用都必须通过统一 Provider Adapter，不得让业务层直接绑定具体 Provider SDK 或原始响应格式。**

---

## 2. 独立性原则

本 Skill 必须独立可读。

本 Skill 不依赖其他 Skill 才能理解，也不复制其他 Skill 的详细规则。

本 Skill 不负责：

- Gateway 和 Orchestrator 的服务间 API
- Orchestrator 如何选择 Agent
- 子 Agent 如何处理任务
- 前端实时事件协议
- 前端 Runtime Capability 参数
- Artifact 完整 schema
- 数据库存储结构
- Docker Compose 服务拓扑
- 具体业务 Prompt 内容

如果其他模块需要调用 LLM，只需要遵守本文定义的 Provider Adapter 契约。

---

## 3. 当前阶段识别

当前项目设定：

```text
profile = v1-generic-llm-provider
mvpStatus = completed
processBoundary = gateway-and-orchestrator-are-separate-services
agentModel = 2-plus-child-agents
```

### 3.1 MVP v0.1 Historical Profile

MVP v0.1 已完成，仅作为历史兼容和回归测试基线。

历史基线包括：

- 单 Provider
- 单模型配置
- 流式文本输出
- API key 来自环境变量
- 基础错误脱敏
- context cancellation
- 不强制 Provider Registry
- 不强制 Model Registry
- 不强制 fallback
- 不强制 structured output

这些历史规则不得继续作为当前开发禁令。

### 3.2 v1 Generic LLM Provider Profile

当前契约必须支持：

- 多 Provider 抽象
- 多模型能力声明
- Provider Registry
- Model Registry
- LLMRequest / LLMResponse
- LLMStreamEvent
- 流式输出归一化
- 结构化输出与本地 JSON Schema 校验
- timeout / retry / rate limit
- Provider fallback / model fallback
- token usage / cost metadata
- request tracing
- secret 脱敏
- Provider error normalization

注意：本契约支持多个 Provider，并不要求当前仓库一次性实现所有 Provider。当前可以只启用一个 Provider，但架构不得写死单 Provider。

---

## 4. 通用性原则

本 Skill 不绑定任何具体 Agent 名称。

不得出现以下设计：

```text
code-agent 固定使用某 Provider
web-agent 固定使用某 Provider
doc-agent 固定使用某 Provider
```

正确设计是：

```text
调用方声明 useCase、modelPolicy、requiredCapabilities。
Provider Adapter 根据 Provider Registry、Model Registry、能力声明和运行配置选择可用模型。
```

调用方可以是：

- Orchestrator Service
- Planner
- 任意 Child Agent
- 系统后台任务
- 后续新增的授权后端运行单元

但所有调用都必须走统一 Provider Adapter。

---

## 5. Gateway / Orchestrator 分进程约束

当前设定中，Gateway 与 Orchestrator 必须分进程。

LLM 调用规则：

- Gateway Service 不得直接调用 LLM Provider。
- Gateway Service 不得直接 import Provider SDK。
- Gateway Service 不得保存 Provider API key。
- Gateway Service 不得构造完整模型 prompt。
- Orchestrator Service、Planner 或 Child Agent 如需调用 LLM，必须通过统一 Provider Adapter。
- 只有被授权的后端运行单元可以调用 Provider Adapter。

Gateway 只负责对外入口、鉴权、请求装配、流式转发和持久化边界；LLM 选择、结构化输出、fallback、错误归一化等能力属于模型调用层和被授权的后端执行单元。

---

## 6. Provider Adapter 核心原则

Provider Adapter 必须隐藏 Provider 差异。

业务层不得依赖：

- OpenAI 原始 response shape
- Anthropic 原始 response shape
- Gemini 原始 response shape
- openai-compatible 私有扩展字段
- Provider 专有 stream event 名称
- Provider 专有 tool call chunk 结构

Provider Adapter 必须输出 AgentHub 统一对象：

- `LLMResponse`
- `LLMStreamEvent`
- `SafeLLMError`
- `TokenUsage`

Provider Adapter 不得：

- 直接调用业务工具
- 直接修改数据库
- 直接生成前端事件
- 直接选择 Agent
- 直接落库原始 Provider 响应
- 直接把 Provider 原始错误返回给用户

---

## 7. Provider Registry

Provider Registry 是 Provider 配置和能力的事实源。

推荐结构：

```ts
type ProviderDefinition = {
  providerName: string
  providerType:
    | 'openai'
    | 'anthropic'
    | 'gemini'
    | 'openai_compatible'
    | 'local'
  status: 'enabled' | 'disabled' | 'experimental' | 'deprecated'
  baseURL?: string
  apiKeyEnv?: string
  defaultModel?: string
  timeoutMs: number
  retryPolicyRef?: string
  rateLimitPolicyRef?: string
  supportsStreaming: boolean
  supportsStructuredOutput: boolean
  supportsToolUse: boolean
}
```

规则：

- `providerName` 必须唯一。
- Provider 配置只能保存 env var 名称，不保存真实 key。
- `disabled` Provider 不得被请求选择。
- `disabled` Provider 不得被 fallback 选择。
- `experimental` Provider 不得作为默认生产路径，除非明确指定。
- `deprecated` Provider 只允许兼容旧请求，不推荐新请求使用。
- `openai_compatible` 不等于 OpenAI 官方完整能力。
- `local` Provider 也必须走同一 Adapter 接口。

---

## 8. Model Registry

Model Registry 是模型能力声明的事实源。

推荐结构：

```ts
type ModelDefinition = {
  modelId: string
  providerName: string
  displayName?: string
  status: 'enabled' | 'disabled' | 'deprecated'
  capabilities: {
    streaming: boolean
    structuredOutput: boolean
    jsonSchema: boolean
    toolUse: boolean
    vision: boolean
    reasoning: boolean
  }
  limits: {
    maxInputTokens?: number
    maxOutputTokens?: number
    defaultTimeoutMs?: number
  }
  costClass: 'free' | 'low' | 'medium' | 'high' | 'unknown'
  useCases?: string[]
}
```

规则：

- `modelId` 在同一 Provider 内必须唯一。
- 业务代码不得直接写死模型名。
- 模型能力必须来自 Model Registry。
- Provider 支持某能力，不代表该 Provider 下所有模型都支持该能力。
- Model `disabled` 后不得被新请求选择。
- fallback 必须选择能力兼容的 model。
- 如果 structured output 请求 fallback 到不支持 schema 的模型，必须显式拒绝或降级为安全失败。

---

## 9. Model Capability 声明

每个模型至少应声明：

```text
streaming
structuredOutput
jsonSchema
toolUse
vision
reasoning
maxInputTokens
maxOutputTokens
defaultTimeoutMs
```

能力声明用于：

- 选择模型
- 校验请求
- 判断是否允许 fallback
- 判断是否允许 structured output
- 判断是否允许 tool use
- 判断是否允许 vision 输入
- 判断是否需要更严格 timeout

禁止：

- 根据 providerName 猜测模型能力。
- 根据 modelId 字符串前缀猜测能力。
- 因为某 Provider 支持结构化输出，就假定所有模型都支持。

---

## 10. LLMRequest

所有模型请求必须归一为 `LLMRequest`。

推荐结构：

```json
{
  "requestId": "req_001",
  "traceId": "trace_001",
  "runId": "run_001",
  "caller": {
    "type": "orchestrator",
    "name": "planner"
  },
  "useCase": "planning",
  "modelPolicy": {
    "preferredProvider": "anthropic",
    "preferredModel": "model-name",
    "requiredCapabilities": ["structured_output"],
    "allowFallback": true
  },
  "messages": [],
  "systemPromptRef": "planner-v1",
  "temperature": 0.2,
  "maxOutputTokens": 1024,
  "stream": false,
  "structuredOutput": {
    "enabled": true,
    "schemaName": "orchestration-plan",
    "schema": {}
  },
  "metadata": {}
}
```

规则：

- `requestId` 必须存在。
- `traceId` 应贯穿调用链。
- `caller` 只用于审计，不用于绕过权限。
- `useCase` 用于选择模型策略。
- `messages` 必须是结构化消息数组。
- `systemPromptRef` 优先于直接内嵌长 prompt。
- `metadata` 不得包含 secret。
- `metadata` 不得包含完整 Authorization header。
- `structuredOutput.enabled = true` 时必须提供 schema 或 schema 引用。

---

## 11. LLMResponse

统一非流式响应结构：

```ts
type LLMResponse = {
  requestId: string
  traceId?: string
  providerName: string
  modelId: string
  content: string
  structured?: unknown
  usage?: TokenUsage
  finishReason?: string
  error?: SafeLLMError
}
```

规则：

- 不得把 Provider 原始 response 直接返回给业务层。
- `providerName` 和 `modelId` 必须记录实际使用值。
- fallback 后必须记录 fallback 后的实际 provider/model。
- structured output 通过本地 schema validation 后才可进入 `structured`。
- `content` 不得包含 Provider secret。
- `error` 必须是脱敏后的 `SafeLLMError`。

---

## 12. LLMStreamEvent

统一流式事件类型：

```text
message_start
delta_text
tool_call_start
tool_call_delta
tool_call_end
usage_delta
message_end
error
```

推荐结构：

```ts
type LLMStreamEvent = {
  type: string
  requestId: string
  providerName?: string
  modelId?: string
  delta?: string
  toolCallId?: string
  toolName?: string
  toolArgsDelta?: string
  usage?: Partial<TokenUsage>
  finishReason?: string
  error?: SafeLLMError
}
```

规则：

- Provider 原始 stream event 不得直接暴露给业务层。
- Provider 原始 stream event 必须先归一化。
- 业务层不得依赖 Provider 专有事件字段。
- error event 必须脱敏。
- context cancelled 后不得继续输出 delta。
- stream end 后不得继续输出普通事件。

---

## 13. Streaming Normalization

流式归一化必须解决：

- 不同 Provider 的 token delta 格式差异
- 不同 Provider 的 message start/end 差异
- 不同 Provider 的 tool call delta 差异
- usage 信息出现时机不同
- finish reason 命名不同
- 中途错误格式不同

规则：

- Adapter 层负责归一化。
- 调用方只消费 `LLMStreamEvent`。
- 流式聚合结果必须可复现为非流式 `LLMResponse`。
- 流中错误必须结束当前请求。
- 取消后必须尽快停止 Provider 请求。

---

## 14. Structured Output Policy

结构化输出不能等同于“相信 LLM”。

结构化输出必须经过三层：

```text
Provider 原生 structured output 能力
→ 本地 JSON parse
→ 本地 JSON Schema validation
```

规则：

- 如果 Provider 支持官方 structured output，应优先使用。
- JSON mode 不等于 schema validation。
- Provider 声称 schema adherence，也必须本地 schema validation。
- schema validation 失败不得执行下游动作。
- LLM 原始输出不得直接作为计划、工具参数、配置或元数据使用。
- schema 必须版本化。
- schema 变更必须记录兼容性影响。

---

## 15. Tool Use Adapter Boundary

Provider 可以支持 tool use / function calling，但 Provider Adapter 不直接执行业务工具。

规则：

- Provider tool call 必须归一化为内部 tool intent。
- Provider Adapter 不直接调用外部工具。
- Provider Adapter 不直接修改数据库。
- Provider Adapter 不直接生成前端事件。
- Tool call 参数必须 schema validation。
- tool use 支持情况必须由 model capabilities 声明。
- 不支持 tool use 的模型不得处理 tool-use 请求。

---

## 16. Timeout / Retry / Rate Limit

每个 LLM 请求必须有 timeout。

推荐结构：

```ts
type RetryPolicy = {
  maxAttempts: number
  initialBackoffMs: number
  maxBackoffMs: number
  jitter: boolean
  retryableErrorCodes: string[]
}

type RateLimitPolicy = {
  requestsPerMinute?: number
  tokensPerMinute?: number
  concurrency?: number
  queueTimeoutMs?: number
}
```

规则：

- timeout 必须小于调用方整体 deadline。
- retry 必须有最大次数。
- retry 必须区分 retryable / non-retryable error。
- context cancelled 后不得 retry。
- 用户取消后不得 fallback。
- rate limit 必须尊重 Provider 官方错误。
- retry / fallback 必须记录 trace。

---

## 17. Fallback Policy

Fallback 是 Provider / Model 选择层的降级能力，不等于 retry。

推荐结构：

```ts
type FallbackPolicy = {
  enabled: boolean
  mode:
    | 'same_provider_different_model'
    | 'different_provider_same_capability'
    | 'lower_cost_model'
    | 'fail_fast'
  maxFallbacks: number
  requiredCapabilities: string[]
}
```

规则：

- fallback 只能选择 enabled Provider / Model。
- fallback 必须满足 requiredCapabilities。
- fallback 不得跨越用户或系统禁止的 Provider。
- structured output 请求不得 fallback 到不支持 schema 的模型，除非显式降级并重新校验。
- fallback 后必须记录实际 providerName / modelId。
- fallback 不能无限循环。

---

## 18. Secret / Config Policy

API key 和敏感配置必须安全处理。

规则：

- API key 只能来自环境变量、secret manager 或等价机制。
- 配置文件只能保存 env var 名称。
- 不得把真实 API key 写进 YAML、JSON、AgentCard、Prompt、Artifact、日志、数据库普通字段。
- 不得把用户 token 当 Provider API key。
- 不得在错误信息中暴露 Authorization header。
- 不得在 trace metadata 中保存 secret。
- 不得在测试 fixture 中提交真实 Provider 响应中含有的 secret。

---

## 19. Prompt Template Policy

本 Skill 不定义具体业务 Prompt，但定义 Prompt 模板的安全边界。

推荐结构：

```ts
type PromptTemplate = {
  templateId: string
  version: string
  useCase: string
  requiredVariables: string[]
  owner?: string
  status: 'draft' | 'active' | 'deprecated'
}
```

规则：

- Prompt 模板必须版本化。
- Prompt 变量必须显式声明。
- 用户输入不得无边界拼接到 system prompt。
- Prompt 中不得包含 API key、内部 token、数据库连接串。
- Prompt 变更影响 structured output 时必须同步 schema。
- 生产路径不得依赖临时 prompt 草稿。
- Prompt 日志只能保存脱敏摘要或版本引用。

---

## 20. Usage / Cost Metadata

所有 Provider 调用应尽可能记录使用量。

```ts
type TokenUsage = {
  inputTokens?: number
  outputTokens?: number
  totalTokens?: number
  cachedInputTokens?: number
  reasoningTokens?: number
  costClass?: 'free' | 'low' | 'medium' | 'high' | 'unknown'
}
```

规则：

- usage 不得伪造。
- Provider 不返回 usage 时，应标记 unknown，而不是填 0。
- fallback 后应记录每次 attempt 的 usage 摘要。
- 成本信息用于观测和调优，不得作为唯一安全控制。

---

## 21. Error Normalization

Provider 错误必须归一化为 `SafeLLMError`。

```ts
type SafeLLMError = {
  code: string
  message: string
  retryable: boolean
  providerName?: string
  modelId?: string
  statusCode?: number
}
```

推荐错误码：

```text
LLM_BAD_REQUEST
LLM_UNAUTHORIZED
LLM_RATE_LIMITED
LLM_TIMEOUT
LLM_CONTEXT_CANCELLED
LLM_PROVIDER_UNAVAILABLE
LLM_STRUCTURED_OUTPUT_INVALID
LLM_STREAM_INTERRUPTED
LLM_INTERNAL
```

规则：

- 用户可见 message 必须脱敏。
- 内部日志可记录 provider error code，但不得记录 secret。
- Provider 原始错误不得直接返回给用户。
- 401 / 403 通常不可 retry。
- 429 / 5xx 可按策略 retry。
- schema validation 失败是本地错误，不应伪装成 Provider 成功。

---

## 22. Trace / Logging

LLM 调用必须可追踪。

建议字段：

```text
requestId
traceId
runId
caller.type
caller.name
providerName
modelId
useCase
timeoutMs
attempt
fallbackAttempt
errorCode
usage
latencyMs
```

禁止日志：

```text
API key
Authorization header
完整 system prompt
完整用户敏感输入
数据库连接串
Provider 原始响应中的敏感字段
```

---

## 23. Contract-first 规则

修改 LLM Provider 行为前，必须先更新契约。

需要先更新契约的情况：

- 新增 Provider
- 新增 Model
- 新增 model capability
- 修改 LLMRequest / LLMResponse
- 修改流式事件类型
- 启用 structured output
- 启用 tool use
- 启用 fallback
- 修改 retry / timeout / rate limit
- 修改 secret 来源
- 修改错误码

---

## 24. Contract Test 规则

至少应测试：

- Provider Registry 加载
- Model Registry 加载
- disabled Provider 不可用
- disabled Model 不可用
- missing API key 报安全错误
- timeout 生效
- context cancellation 生效
- streaming event 归一化
- structured output 本地 schema validation
- invalid structured output 不执行下游动作
- retry 上限
- fallback 能力兼容检查
- error normalization 脱敏
- usage unknown 不伪造 0

---

## 25. Review Checklist

### 通用性

- 是否没有绑定具体 Agent 名称？
- 是否没有让业务代码直接调用 Provider SDK？
- 是否所有 LLM 调用都经过 Provider Adapter？
- Gateway 是否没有直接调用 LLM Provider？

### Provider Registry

- `providerName` 是否唯一？
- Provider 状态是否明确？
- API key 是否只保存 env var 名称？
- `openai_compatible` 是否没有假装等同 OpenAI 官方完整能力？

### Model Registry

- `modelId` 是否唯一？
- model 能力是否声明？
- structuredOutput / streaming / toolUse 是否按 model 声明？
- fallback 是否选择能力兼容模型？

### Request / Response

- LLMRequest 是否有 requestId / traceId？
- messages 是否结构化？
- metadata 是否无 secret？
- LLMResponse 是否不暴露 Provider 原始对象？

### Streaming

- Provider 原始 stream 是否已归一化？
- 是否不把 Provider 原始事件传给业务层？
- context cancelled 后是否停止输出？

### Structured Output

- 是否使用 schema？
- 是否本地 JSON Schema validation？
- JSON mode 是否没有被当成 schema validation？
- validation 失败是否不执行下游动作？

### 安全

- API key 是否未进入日志 / Prompt / Artifact / 数据库普通字段？
- 错误是否脱敏？
- Prompt 模板是否不含 secret？

### 可靠性

- 是否有 timeout？
- retry 是否有上限？
- rate limit 是否处理？
- fallback 是否有授权和能力校验？

---

## 26. 完成定义

本 Skill 视为完成，当且仅当：

- `SKILL.md` 使用中文并独立可读。
- MVP v0.1 被降级为 Historical Profile。
- Skill 不绑定具体 Agent 名称。
- 定义了 Provider Registry。
- 定义了 Model Registry。
- 定义了 LLMRequest / LLMResponse / LLMStreamEvent。
- 定义了 streaming normalization。
- 定义了 structured output 本地校验规则。
- 定义了 tool use 适配边界。
- 定义了 timeout / retry / rate limit。
- 定义了 fallback policy。
- 定义了 secret / config policy。
- 定义了 prompt template policy。
- 定义了 usage / cost metadata。
- 定义了 SafeLLMError。
- 定义了 Review Checklist。

---

## References

- `references/provider-registry.md`
- `references/model-registry.md`
- `references/provider-adapter.md`
- `references/llm-request-response.md`
- `references/streaming-normalization.md`
- `references/structured-output-policy.md`
- `references/tool-use-adapter-policy.md`
- `references/retry-rate-limit-timeout-policy.md`
- `references/fallback-policy.md`
- `references/secret-config-policy.md`
- `references/prompt-template-policy.md`
- `references/usage-cost-policy.md`
- `references/error-normalization-policy.md`
- `references/llm-provider-review-checklist.md`
