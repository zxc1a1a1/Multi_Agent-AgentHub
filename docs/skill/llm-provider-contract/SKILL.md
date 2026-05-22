---
name: llm-provider-contract
description: 当定义、实现、修改或审查 AgentHub 中 LLM Provider 适配、模型注册表、流式归一化、结构化输出、Prompt 模板、API Key 安全、重试限流或 Provider fallback 时，使用本 Skill。
---

# llm-provider-contract

## 1. 目的

本 Skill 定义 AgentHub 的 LLM Provider 开发契约。

LLM Provider 是 AgentHub 内部统一调用不同模型服务的适配层。

核心链路：

```text
OpenAI / Anthropic / Gemini / OpenAI-compatible / Local Model
→ Provider Adapter
→ AgentHub LLMRequest
→ AgentHub LLMStreamEvent / LLMResponse
→ ADK Runtime / Orchestrator / Agent Handler
```

目标：

- 不把业务逻辑绑定到某一家 Provider SDK。
- 不把 Provider 原始 streaming event 直接传入 ADK Runtime、A2A 或 AG-UI。
- 不把 API key 写入配置、AgentCard、日志、Artifact 或 Prompt。
- 不把 Provider 原始错误直接暴露给用户。
- 不让 LLM Provider 直接生成 AgentHub Artifact 或 AG-UI 事件。
- 不让结构化输出绕过 JSON Schema validation。
- 不让 retry、rate limit、fallback 变成隐式行为。

## 2. 官方文档优先级

涉及具体 Provider 行为时，以各 Provider 官方文档为准。

官方文档优先范围包括：

- OpenAI API：Structured Outputs、Responses / Chat API、tool calling、streaming、error、rate limit。
- Anthropic Claude API：Messages API、streaming、tool use、error、rate limit、SDK 行为。
- Google Gemini API：generateContent、structured output、streaming、tool use、error、rate limit。
- JSON Schema 官方规范：结构化输出和本项目 schema 的基础规范。
- OpenAI-compatible Provider：以其兼容声明和真实行为为准，不默认等同 OpenAI 官方完整能力。

优先级：

```text
Provider 官方文档 > AgentHub adapter 假设
JSON Schema 官方规范 > 手写非标准 schema
AgentHub LLMProvider Contract > 业务代码直接调用 Provider SDK
```

如果 Provider 官方能力与本项目统一抽象不一致：

```text
Provider Adapter 负责差异转换
AgentHub 内部只消费统一 LLMRequest / LLMResponse / LLMStreamEvent
```

## 3. 文件位置说明

### 项目级 Contract 文件

以下文件属于 AgentHub 项目仓库，是项目级事实源：

```text
<repo-root>/docs/contracts/llm-provider.md
<repo-root>/docs/contracts/llm-provider.schema.json
```

### 当前 Skill 的参考文件

以下文件属于当前 Coding Agent Skill：

```text
<current-skill-dir>/references/provider-adapter.md
<current-skill-dir>/references/model-registry.md
<current-skill-dir>/references/streaming-normalization.md
<current-skill-dir>/references/structured-output.md
<current-skill-dir>/references/retry-rate-limit-policy.md
<current-skill-dir>/references/secret-config.md
<current-skill-dir>/references/prompt-template-policy.md
<current-skill-dir>/references/fallback-policy.md
```

如果当前 Skill 安装在 Claude Code 项目目录中，则 `<current-skill-dir>` 通常是：

```text
<repo-root>/.claude/skills/llm-provider-contract
```

## 4. 适用场景

当进行以下工作时，启用本 Skill：

- 新增 LLM Provider。
- 修改 Provider Adapter。
- 修改模型注册表。
- 修改模型能力声明。
- 修改流式输出归一化。
- 修改结构化输出能力。
- 修改 prompt 模板规则。
- 修改 API key / secret 配置。
- 修改 retry / rate limit / timeout。
- 修改 Provider fallback。
- 修改 token usage / cost 统计。
- 修改 LLM error normalization。
- 修改 code-agent、planner 或任意 Agent 的 LLM 调用方式。

## 5. 长期契约基线

长期架构中，AgentHub 必须通过统一 Provider Adapter 调用模型。

业务代码不得直接散落调用 Provider SDK。

长期 contract 必须定义：

- Provider Registry。
- Model Registry。
- 模型能力声明。
- 统一 `LLMRequest`。
- 统一 `LLMResponse`。
- 统一 `LLMStreamEvent`。
- streaming normalization。
- structured output 策略。
- tool use 适配边界。
- token usage 统计。
- retry / backoff。
- rate limit。
- timeout。
- fallback。
- secret 配置。
- prompt 模板安全策略。
- provider error normalization。
- request trace 规则。

长期支持的 Provider 类型：

```text
openai
anthropic
gemini
openai_compatible
local
```

模型能力声明至少包括：

```text
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

## 6. MVP 约束

MVP 阶段只要求：

```text
code-agent 能调用一个 LLM Provider
```

MVP 默认可以只接入：

```text
anthropic
```

或项目负责人指定的单一 Provider。

MVP 阶段最小能力：

- 支持 streaming text。
- 支持 context cancellation。
- API key 从环境变量读取。
- Provider 原始 stream chunk 归一化后再交给 ADK Runtime。
- Provider 原始错误转换为用户安全错误。
- 不要求多 Provider registry UI。
- 不要求 Provider fallback。
- 不要求复杂成本统计。
- 不要求 prompt 模板版本系统。
- 不要求 tool use。
- 不要求 vision。
- 不要求完整 structured output。

## 7. 阶段演进规则

### MVP 阶段

只实现：

```text
single provider
streaming text
safe error normalization
env-based secret
```

不得把单 Provider 直连写成长期唯一实现。

### P1 / 正式开发阶段

逐步启用：

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

启用前必须先更新：

```text
<repo-root>/docs/contracts/llm-provider.md
<repo-root>/docs/contracts/llm-provider.schema.json
```

并同步检查当前 Skill 的 references 文件。

## 8. 本 Skill 负责

本 Skill 负责：

- LLM Provider 抽象。
- Provider Adapter 规则。
- Model Registry 规则。
- Provider / model 能力声明。
- LLMRequest / LLMResponse / LLMStreamEvent 规则。
- Streaming normalization。
- Structured output 规则。
- Tool use 适配边界。
- Prompt 模板安全策略。
- API key / secret 配置规则。
- Retry / timeout / rate limit 策略。
- Provider fallback 策略。
- Token usage / cost metadata 规则。
- Provider error normalization。
- MVP 单 Provider 接入规则。
- P1 多 Provider 扩展规则。

## 9. 本 Skill 不负责

本 Skill 不负责：

- Orchestrator 如何选择 Agent。
- ExecutionPlan schema。
- A2A Task 协议。
- AG-UI 事件结构。
- Frontend Runtime Skill。
- React Component。
- Artifact schema。
- Artifact 持久化策略。
- ADK Runtime Context API。
- 子 Agent handler 的业务逻辑。
- 数据库完整 DDL。
- Docker Compose 交付规则。
- 通用 Go / TypeScript 代码风格。

## 10. Contract first 规则

任何新增或修改 LLM Provider 行为前，必须先更新：

```text
<repo-root>/docs/contracts/llm-provider.md
<repo-root>/docs/contracts/llm-provider.schema.json
```

未更新 contract 的实现变更不得接受。

如果修改 structured output，还必须检查：

```text
<repo-root>/docs/contracts/execution-plan.schema.json
```

如果修改 Agent Runtime 使用 LLM 的方式，还必须检查：

```text
<repo-root>/docs/contracts/adk-runtime.md
```

如果修改安全边界，还必须检查：

```text
<repo-root>/docs/contracts/security-boundaries.md
```

## 11. 核心规则

### Provider Adapter

业务代码必须通过统一 Provider Adapter 调用 LLM。

推荐接口：

```text
Generate(ctx, request) -> LLMResponse
Stream(ctx, request) -> Iterator<LLMStreamEvent>
CountTokens(ctx, request) -> TokenUsage
```

### Streaming Normalization

Provider 原始 stream event 不得直接进入 ADK Runtime、A2A 或 AG-UI。

必须先转换为 AgentHub 内部事件：

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

### Structured Output

如果 Provider 支持官方 structured output，应优先使用。

无论 Provider 是否声称支持 structured output，AgentHub 必须执行本地 schema validation。

JSON mode 不等于 schema adherence。

### Secret

API key 只能从环境变量、secret manager 或等价安全机制读取。

配置中只能出现 env var 名称：

```text
ANTHROPIC_API_KEY
OPENAI_API_KEY
GEMINI_API_KEY
```

不得出现真实 key。

### Retry / Rate Limit / Timeout

每个 Provider 请求必须有 timeout。

Retry 必须有上限。

不得无限 retry。

不得在 context cancelled 后继续请求或 retry。

Provider Adapter 必须尊重 Provider 官方 rate limit。

## 12. 禁止事项

Coding Agent 不得：

- 在业务 handler 中直接到处调用 Provider SDK。
- 把 Provider 原始 streaming event 直接传给 ADK Runtime、A2A 或 AG-UI。
- 把 Provider 原始错误直接返回给用户。
- 把 API key 写入配置、AgentCard、Artifact、Prompt、日志或数据库普通字段。
- 让 LLM Provider 直接生成 AG-UI 事件。
- 让 LLM Provider 直接生成 AgentHub Artifact。
- 把 JSON mode 当成 schema validation。
- 跳过本地 schema validation。
- 在 Provider 不支持 structured output 时假装支持。
- 无限 retry。
- 忽略 context cancellation。
- fallback 到未授权 Provider。
- 在没有 contract 的情况下新增 Provider。
- 把 MVP 单 Provider 直连写成长期唯一架构。

## 13. 相关契约

本 Skill 只引用以下契约，不重新定义它们：

- `adk-runtime-contract`：负责子 Agent Runtime、Task handler、`ctx.StreamText`、`ctx.AddArtifact`。
- `intent-orchestration-contract`：负责 ExecutionPlan、Agent 路由、planner validation。
- `artifact-contract`：负责 Artifact schema、类型、生命周期和存储策略。
- `a2a-agent-contract`：负责 A2A Task、AgentCard、Streaming 和错误语义。
- `agui-event-contract`：负责 AG-UI 事件名称和事件结构。
- `security-boundary-contract`：负责 secret、sandbox、工具权限和敏感信息保护。
- `observability-debugging-contract`：负责 traceId、runId、日志、指标和调试。
- `data-persistence-contract`：负责数据库、Redis、对象存储和迁移策略。
