# LLM Provider Contract

版本：v0.1-p1  
适用项目：AgentHub - 多 Agent 协作平台  
适用阶段：MVP 轻量接入 + P1 正式开发演进  
事实源文件：

```text
<repo-root>/docs/contracts/llm-provider.md
<repo-root>/docs/contracts/llm-provider.schema.json
```

## 1. 目的

本文定义 AgentHub LLM Provider 的项目级契约。

LLM Provider 是 AgentHub 内部统一调用不同模型服务的适配层。

核心链路：

```text
OpenAI / Anthropic / Gemini / OpenAI-compatible / Local Model
→ Provider Adapter
→ AgentHub LLMRequest
→ AgentHub LLMStreamEvent / LLMResponse
→ ADK Runtime / Orchestrator / Agent Handler
```

## 2. 官方文档优先

涉及具体 Provider 行为时，以各 Provider 官方文档为准。

优先级：

```text
Provider 官方文档 > AgentHub adapter 假设
JSON Schema 官方规范 > 手写非标准 schema
AgentHub LLMProvider Contract > 业务代码直接调用 Provider SDK
```

Provider 差异必须由 Provider Adapter 处理。

AgentHub 内部只消费统一抽象：

```text
LLMRequest
LLMResponse
LLMStreamEvent
```

## 3. Contract first 规则

任何新增、修改或删除 Provider、模型能力、streaming normalization、structured output、retry、rate limit、fallback、secret 配置前，必须先更新：

```text
<repo-root>/docs/contracts/llm-provider.md
<repo-root>/docs/contracts/llm-provider.schema.json
```

未更新 contract 的实现变更不得接受。

## 4. 阶段演进规则

### MVP 阶段

只要求：

```text
single provider
streaming text
safe error normalization
env-based secret
```

MVP 默认可以只接入一个 Provider，例如：

```text
anthropic
```

MVP 不要求：

- 多 Provider fallback。
- structured output。
- tool use。
- vision。
- token usage 精细统计。
- 成本估算。
- provider health check。
- prompt template versioning。

### P1 / 正式开发阶段

可以逐步启用：

- Provider Registry。
- Model Registry。
- 多模型能力声明。
- structured output。
- tool use 适配。
- token usage 统计。
- cost estimate。
- retry / backoff。
- rate limit。
- fallback provider。
- fallback model。
- prompt template versioning。
- provider health check。
- request tracing。

## 5. Provider Registry

Provider 注册项应包含：

```text
name
type
enabled
baseURL
defaultModel
apiKeyEnv
timeoutMs
retry
rateLimit
models
```

支持 Provider 类型：

```text
openai
anthropic
gemini
openai_compatible
local
```

配置中只能出现 env var 名称，不得出现真实 API key。

## 6. Model Registry

模型注册项应包含：

```text
id
providerModel
enabled
supportsStreaming
supportsStructuredOutput
supportsJsonSchema
supportsToolUse
supportsVision
maxInputTokens
maxOutputTokens
costClass
defaultTimeoutMs
```

不得在业务代码中硬编码模型能力。

## 7. Streaming Normalization

Provider 原始 stream event 不得直接进入 ADK Runtime、A2A 或 AG-UI。

必须转换为 AgentHub 内部事件：

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

MVP 阶段只强制：

```text
delta_text
message_end
error
```

## 8. Structured Output

如果 Provider 支持官方 structured output，应优先使用。

无论 Provider 是否支持 structured output，AgentHub 必须执行本地 schema validation。

JSON mode 不等于 schema adherence。

## 9. Secret 配置

API key 只能来自：

- environment variable。
- secret manager。
- runtime injected secret。

配置文件中只能写 env var 名称，例如：

```text
ANTHROPIC_API_KEY
OPENAI_API_KEY
GEMINI_API_KEY
```

不得将 API key 写入 config、AgentCard、ExecutionPlan、Artifact、Prompt、日志、错误信息、前端响应或数据库普通字段。

## 10. Retry / Rate Limit / Timeout

每个 Provider 请求必须有 timeout。

Retry 必须有上限。

不得无限 retry。

不得在 context cancelled 后继续请求或 retry。

Provider Adapter 必须尊重 Provider 官方 rate limit。

## 11. Prompt 模板

Prompt 模板必须可审查。

正式开发阶段建议支持 prompt version。

Prompt 不得包含 secret、token、数据库连接字符串、对象存储私有地址或用户无权访问的上下文。

## 12. Fallback

MVP 阶段不要求 fallback。

正式开发阶段 fallback 必须显式配置。

Fallback 不得降低安全等级。

Fallback 不得绕过 structured output validation。

## 13. 禁止事项

不得：

- 在业务 handler 中直接散落 Provider SDK 调用。
- 把 Provider 原始 streaming event 直接传给 ADK Runtime、A2A 或 AG-UI。
- 把 Provider 原始错误直接返回给用户。
- 把 API key 写入配置、AgentCard、Artifact、Prompt、日志或数据库普通字段。
- 让 LLM Provider 直接生成 AG-UI 事件。
- 让 LLM Provider 直接生成 AgentHub Artifact。
- 把 JSON mode 当成 schema validation。
- 跳过本地 schema validation。
- 无限 retry。
- 忽略 context cancellation。
- fallback 到未授权 Provider。
- 把 MVP 单 Provider 直连写成长期唯一架构。
