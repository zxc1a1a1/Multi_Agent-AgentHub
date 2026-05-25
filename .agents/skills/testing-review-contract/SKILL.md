---
name: testing-review-contract
description: "用于定义 AgentHub v1.0 及后续演进的测试分层、契约验证、分进程集成测试、Planner/Registry/Artifact/Streaming/Frontend/E2E/Security 测试、PR Review 和 CI Quality Gate。本 Skill 不绑定具体 Agent。"
---

# testing-review-contract

## 1. Skill 目的

本 Skill 是 AgentHub 的质量门禁与测试 Review 契约。它用于约束所有涉及跨服务、跨协议、跨 Agent、LLM 编排、前端运行时产物和部署交付的变更如何被测试、审查和阻断。

本 Skill 不写业务实现代码，也不替代其他契约的字段定义。它只规定：什么必须测试、怎样分层测试、哪些场景必须进入 CI 或 smoke test、PR Review 必须检查什么、什么时候不能合并。

## 2. 独立性原则

本 Skill 必须独立可读。读者不需要先阅读其他 Skill，也能理解 AgentHub 的测试边界：Frontend 通过 Gateway 访问平台，Gateway 与 Orchestrator 分进程，Orchestrator 调度多个 Child Agent，系统通过 Registry、Health Check、Planner、Artifact、Runtime Capability 和 Streaming Event 完成协作。

本 Skill 可以说明其他边界存在，但不得复制其他 Skill 的完整规则。

## 3. 当前事实源

- `SPRINT-v1.0-Plan.md` 是 v1.0 当前目标事实源。
- MVP v0.1 已完成，只能作为 Historical Profile 和回归基线。
- Gateway Service 与 Orchestrator Service 必须分进程。
- 当前质量门禁面向 2+ Agent、单聊 + 群聊、LLM 编排、Agent Registry、健康检查、丰富产物、降级重试和 Docker Demo。

## 4. 本 Skill 负责什么

- AgentHub Test Matrix。
- Contract-first 测试策略。
- Schema validation 测试。
- Gateway ↔ Orchestrator 分进程集成测试。
- Planner / OrchestrationPlan 测试。
- Agent Registry / Health Check 测试。
- Streaming / SSE / internal stream replay 测试。
- Artifact / Runtime Capability 测试。
- Frontend component、store、event replay、E2E 测试。
- Backend unit、service integration、converter、persistence roundtrip 测试。
- failure / fallback / retry 测试。
- Security regression 测试。
- Docker smoke test 与 Demo review。
- CI Quality Gates。
- PR Review Checklist。

## 5. 本 Skill 不负责什么

- 不定义 REST API 完整字段。
- 不定义 Gateway ↔ Orchestrator 内部 API 完整 schema。
- 不定义 Child Agent 协议完整字段。
- 不定义 LLM Provider SDK 实现。
- 不定义 Artifact 完整 schema。
- 不定义数据库 DDL。
- 不定义 Docker Compose 具体实现。
- 不定义 Go / TypeScript 业务代码。
- 不替代安全边界、数据持久化、平台 API、LLM Provider、Artifact 或前端 Runtime 的专门契约。

## 6. 测试哲学

AgentHub 测试必须遵守以下原则：

1. **Contract-first**：跨边界字段先有契约，再有 mock，再有真实集成。
2. **确定性优先**：普通 CI 不依赖真实 LLM、真实外部服务、真实 API Key 或不稳定网络。
3. **测试金字塔**：大量快速单元测试和契约测试，少量关键 E2E 和 Demo smoke。
4. **失败路径等价重要**：fallback、retry、timeout、cancel、invalid schema、unhealthy Agent 必须测试。
5. **分进程优先**：Gateway 与 Orchestrator 的测试不得退化成同进程 handler 调用。
6. **通用 Agent 优先**：测试不得把 `code-agent`、`web-agent`、`doc-agent` 写成唯一合法 Agent。示例可以使用 `example-agent-a`、`example-agent-b`。
7. **安全默认开启**：错误脱敏、Secret 泄漏、iframe sandbox、high-risk action confirm 必须有回归测试。

## 7. AgentHub Test Matrix

测试矩阵按以下层级组织：

| 层级 | 目标 | 必须覆盖 |
|---|---|---|
| Schema / Contract Tests | 字段、枚举、兼容性 | OpenAPI、内部 request/result、Plan、AgentCard、Artifact、ToolCall、LLM、日志 |
| Unit Tests | 单函数和纯逻辑 | planner validation、registry filtering、converter、safe error、fallback policy |
| Component / Store Tests | 前端局部行为 | MessageBubble、WebPreview、Markdown、STATE_UPDATE、per-conversation streaming |
| Adapter / Converter Tests | 协议转换 | Agent event → Gateway event、Artifact → Runtime Capability、ToolCall assembly |
| Service Integration Tests | 单服务真实依赖边界 | Gateway handler、Orchestrator service、Registry service、Persistence roundtrip |
| Cross-process Integration Tests | 分进程通信 | Gateway → Orchestrator URL、internal stream、cancel、timeout、trace propagation |
| Streaming Replay Tests | 流式事件稳定性 | SSE block、incomplete block、malformed JSON、多 Agent messageId |
| Failure / Fallback Tests | 降级能力 | LLM timeout、invalid plan、Agent unhealthy、Agent failed、all failed |
| Security Regression Tests | 安全回归 | 401/403、redaction、sandbox、secret scan、confirm_action |
| Docker Smoke Tests | 可交付验证 | compose up、health、/api/agents、simple run、group run、artifact preview |
| Demo E2E Tests | 核心体验验证 | 单聊、群聊、@mention、代码预览、网页预览、Markdown、刷新持久化 |
| Documentation Gates | AI 协作交付 | README、架构图、contracts、skills、demo checklist |

## 8. Contract / Schema Tests

所有跨边界契约必须有 valid fixture 和 invalid fixture。

必须覆盖：

- Frontend ↔ Gateway Platform API。
- Gateway ↔ Orchestrator internal request / stream / result。
- OrchestrationPlan / ExecutionPlan。
- PlannerInput。
- AgentCard / Agent Registry summary。
- Artifact metadata / contentRef。
- Runtime Capability / ToolCall args。
- LLMRequest / LLMResponse / LLMStreamEvent。
- Observability log / error event。
- Security boundary item / risk action。

规则：

- schema 失败不得进入真实执行。
- 新增字段默认 optional，除非明确版本升级。
- 删除字段必须经过兼容性 Review。
- 枚举新增必须同步前端 fallback 渲染。
- mock、fixture、OpenAPI、schema、docs/contracts 必须保持一致。

## 9. Gateway-Orchestrator 分进程测试

必须证明 Gateway 与 Orchestrator 是独立服务：

- Gateway 通过 `ORCHESTRATOR_URL` 或等价内部地址调用 Orchestrator。
- Gateway 不得 import Orchestrator 业务包执行编排。
- Gateway → Orchestrator 请求必须携带 requestId / traceId / runId。
- Orchestrator `/health` 必须可独立验证。
- Orchestrator `/internal/**` 不得出现在公开 OpenAPI。
- Gateway internal client timeout 必须返回安全错误。
- 浏览器断连必须关闭前端流，并传播 cancel 或关闭内部请求。
- Orchestrator stream error 不得让 Gateway 崩溃。

## 10. Planner / Orchestration Tests

Planner 测试不得调用真实 LLM。必须使用 fake LLM、fixture JSON 或 deterministic planner fake。

必须覆盖：

- 合法结构化 plan 通过 schema validation。
- 非法 JSON 进入 fallback。
- 不存在 Agent 被拒绝或 fallback。
- 不存在 capability 被拒绝。
- no healthy agents 返回 safe error。
- direct / mention / manual / auto / fallback mode。
- group conversation 允许多 task。
- ordered_parallel 多 task message attribution 稳定。
- sequential dependsOn 无环。
- fallback plan 重新校验。

## 11. Agent Registry / Health Tests

必须覆盖：

- valid AgentCard accepted。
- invalid AgentCard rejected。
- missing required fields rejected。
- duplicate agentName deterministic handling。
- unhealthy Agent excluded from healthy selection。
- disabled Agent not selected。
- health timeout marks unhealthy。
- `/api/agents` 只返回前端安全摘要。
- Agent capability 不依赖具体 Agent 名称推断。

## 12. Streaming / Event Replay Tests

流式协议必须可 replay。测试 fixture 必须覆盖：

- SSE event block 以 `\n\n` 分隔。
- incomplete block 可以 buffer。
- malformed JSON 不使前端崩溃。
- message_start / message_delta / message_end 顺序稳定。
- TOOL_CALL_START / TOOL_CALL_ARGS / TOOL_CALL_END 顺序稳定。
- STATE_UPDATE 可解析，但不破坏当前 message。
- 多 Agent 输出使用不同 messageId 或明确 senderName。
- cancel 后不得继续输出普通 delta。
- stream error 必须有 errorCode / safeMessage。

## 13. Artifact / Runtime Capability Tests

v1.0 Sprint 示例包括 code、webpage、markdown，但长期规则按 artifactType / runtimeCapability 扩展。

必须覆盖：

- code artifact → code preview capability。
- webpage artifact → web preview capability。
- markdown content → markdown render path。
- unknown artifact type → safe fallback。
- missing runtime capability → 不生成前端不可消费 ToolCall。
- large artifact → contentRef，不塞入普通 message。
- Artifact 关联 runId / messageId。
- iframe sandbox 存在。
- markdown 不执行危险 HTML。

## 14. Backend Tests

后端测试必须覆盖：

- Planner validation。
- Registry discovery / filtering。
- Health check timeout。
- Orchestrator strategy selection。
- ordered_parallel message attribution。
- Gateway internal client timeout。
- Orchestrator cancellation。
- Protocol converter。
- Artifact normalization。
- Safe error mapping。
- Persistence roundtrip。
- Graceful shutdown 可验证。

禁止：

- 单元测试启动真实 Docker Compose。
- 单元测试依赖真实 LLM。
- 单元测试使用真实 API key。
- 测试 fixture 包含真实 secret。

## 15. Frontend Tests

前端测试必须覆盖用户可见行为：

- MessageBubble 显示 senderName。
- AgentAvatar 不依赖固定 Agent 名称。
- WebPreview 渲染 iframe 且 sandbox 存在。
- StreamingText 渲染 Markdown。
- ErrorBoundary 显示安全错误。
- Store 支持 per-conversation streaming。
- Store 支持 per-conversation abortControllers。
- TOOL_CALL event assembly 正确。
- STATE_UPDATE activeAgent / retrying 状态正确。
- webPreviews / codeBlocks append 正确。

## 16. E2E / Demo Smoke Tests

E2E 不追求覆盖所有分支，只覆盖关键体验路径：

- 平台加载。
- Agent 列表显示。
- 创建单聊。
- 创建群聊。
- 发送消息并看到流式回复。
- @mention 触发指定目标提示。
- 多 Agent 依次回复，消息归属清晰。
- code preview 可见。
- web preview iframe 可见。
- markdown 渲染可见。
- fallback / retrying 提示可见。
- 刷新页面后历史消息保留。

## 17. Failure / Fallback / Retry Tests

必须覆盖：

- Planner LLM timeout → fallback planner。
- Planner invalid output → fallback planner。
- selected Agent unhealthy → 选择替代或 safe error。
- selected Agent call failed → fallback candidate。
- all agents failed → RUN_ERROR safe message。
- fallback emits STATE_UPDATE。
- retry has max attempts。
- retry does not loop forever。
- partial failure does not corrupt completed message。
- frontend shows retrying / fallback hint。

## 18. Security Regression Tests

必须覆盖：

- 未鉴权 public API 返回 401。
- 无权限资源返回 403。
- 错误响应无 stack trace / token / API key。
- AgentCard 无 secret。
- logs / fixtures 无真实 key。
- iframe 有 sandbox。
- markdown 不执行危险 HTML。
- high-risk tool 需要 confirm_action。
- service-to-service token 不出现在前端响应。
- `/internal/**` 不在公开 OpenAPI。

## 19. Fixture / Mock / Fake / Golden File Policy

- fixture 必须脱敏。
- fake LLM 必须可预测。
- mock Agent 必须通用命名，不固定长期 Agent 名称。
- golden file 更新必须经过 Review。
- invalid fixture 必须覆盖缺字段、错类型、未知枚举、越权字段。
- replay fixture 必须覆盖中断、乱序、重复、缺失结束事件。

## 20. CI Quality Gates

必过门禁：

- lint / format check。
- backend unit。
- frontend unit。
- schema validation。
- contract tests。
- security regression。
- minimal smoke test。

条件必过门禁：

- docker compose smoke。
- E2E demo path。
- migration check。
- coverage threshold。

禁止：

- test failed 仍允许 merge。
- 普通 CI 使用真实 LLM key。
- 跳过 contract test。
- CI 输出 secret。
- 分进程相关测试长期关闭。

## 21. PR Review Checklist

PR Review 必须同时检查：

- Contract 是否同步。
- Schema 是否有 valid / invalid fixtures。
- Gateway / Orchestrator 是否仍分进程。
- 是否没有把具体 Agent 名称固化成长期规则。
- 是否覆盖失败路径。
- 是否有安全回归测试。
- 是否更新 smoke / demo checklist。
- 是否更新 README、架构图或 AI 协作文档。

## 22. Historical MVP Testing Profile

MVP v0.1 已完成，仅作为历史回归基线。

历史基线包括：

- 单 Agent happy path。
- 最小流式回复。
- 最小 Artifact 映射。
- 最小前端预览。
- 最小 docker compose smoke。
- 最小 secret redaction check。

这些历史测试不得继续作为 v1.0 当前质量上限。

## 23. v1.0 Sprint Testing Profile

v1.0 当前质量门禁必须覆盖：

- 2+ Agent。
- 单聊 + 群聊。
- LLM Planner。
- Agent Registry / Health Check。
- 结构化多轮消息。
- code / webpage / markdown 等丰富产物。
- fallback / retry。
- single / ordered_parallel / sequential。
- Docker Demo。
- smoke test。
- AI 协作文档完整性。

## 24. 完成定义

一个变更只有在以下条件满足时，才能被视为完成：

- 相关契约已更新。
- schema / fixture 已更新。
- 单元测试、契约测试、失败路径测试通过。
- 跨服务边界没有被破坏。
- 安全回归测试通过。
- smoke / demo 路径未破坏。
- PR Review Checklist 已完成。
