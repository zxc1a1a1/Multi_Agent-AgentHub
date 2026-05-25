---
name: security-boundary-contract
description: "用于定义 AgentHub v1.0 及后续演进的全链路安全边界契约，包括 Frontend、Gateway、Orchestrator、Child Agent、LLM、Registry、Artifact、Tool、文件、部署、密钥、鉴权、授权、sandbox、错误脱敏、审计和安全测试。本 Skill 不绑定具体 Agent。"
---

# security-boundary-contract

## 1. Skill 目的

本 Skill 是 AgentHub 的安全边界事实源，用于在开发、评审、生成代码、生成契约、设计 Demo 或接入新能力时判断：哪些输入不可信、哪些动作必须授权、哪些数据不能泄露、哪些接口不能公开、哪些能力必须经过确认和审计。

本 Skill 面向 AgentHub v1.0 及后续扩展。当前 v1.0 目标来自 `SPRINT-v1.0-Plan.md`：多 Agent 协作、LLM 意图编排、单聊与群聊、Agent Registry、健康检查、丰富产物预览、降级重试和 Docker Demo。

## 2. 独立性原则

本 Skill 必须独立可读。阅读者不需要先阅读其他 Skill，也能理解 AgentHub 的安全边界。

可以引用以下概念，但不得展开其他 Skill 的完整规则：

- Gateway Service
- Orchestrator Service
- Child Agent Service
- Agent Registry
- Artifact
- Runtime Capability
- Tool Call
- LLM Provider
- Conversation / Message / Run

## 3. 当前事实源

- `SPRINT-v1.0-Plan.md` 是 v1.0 当前目标事实源。
- MVP v0.1 已完成，只能作为 Historical Profile 或回归基线。
- Gateway Service 与 Orchestrator Service 必须分进程。
- 当前方向支持 2+ Agent、单聊 + 群聊、LLM 编排、Registry、健康检查、丰富产物、fallback / retry。
- Sprint 中出现的具体 Agent 名称或文件路径只能作为实施示例，不能成为长期安全边界的硬编码条件。

## 4. 本 Skill 负责什么

本 Skill 负责定义：

- 信任边界模型
- Frontend 安全边界
- Gateway 公开 API 安全边界
- Gateway ↔ Orchestrator 服务间安全
- Orchestrator 编排安全
- Child Agent 与用户自建 Agent 信任边界
- Agent Registry / AgentCard / Health Check 安全
- 用户鉴权、服务鉴权、对象级授权
- LLM / Planner 安全
- Secret 管理
- Artifact / Runtime Capability 安全
- iframe sandbox / CSP / XSS 渲染安全
- Tool Action 风险等级
- confirm_action 安全门
- run_command / deploy / file overwrite 等高危动作安全
- file_upload / file_download 安全
- rate limit / quota / resource limit
- 错误脱敏
- 日志与审计
- 安全契约测试与 Review Checklist

## 5. 本 Skill 不负责什么

本 Skill 不负责：

- REST API 完整字段定义
- Gateway ↔ Orchestrator 内部 API 完整 schema
- Child Agent 协议字段
- LLM Provider SDK 适配实现
- Artifact 完整 schema
- 数据库 DDL
- Docker Compose 具体实现
- Go / TypeScript / SQL / Dockerfile 业务实现代码
- 某个具体 Agent 的业务权限细节

## 6. Trust Boundary Model

AgentHub 默认采用“不信任输入，最小权限执行，先校验后动作”的安全模型。

以下对象默认不可信：

- 用户输入
- Frontend 可提交的任何字段
- LLM 输出
- Planner 输出
- Child Agent 输出
- 用户自建 Agent
- AgentCard 声明
- Artifact 内容
- Tool Call 参数
- 上传文件
- 外部 URL
- Provider 原始错误

规则：

- LLM 输出只作为建议，不是授权依据。
- AgentCard 是能力声明，不是信任凭证。
- Artifact 是不可信内容，不得直接注入主应用 DOM。
- Tool Call 参数必须经过 schema validation、权限校验和风险判断。
- 高危动作必须经过 confirm_action。
- 所有跨用户资源访问必须做对象级授权。

## 7. Public API Security Boundary

Frontend 只能访问 Gateway Service 的公开 API。

公开 API 必须满足：

- 默认需要用户鉴权。
- token 只能放在 `Authorization` header。
- token 不得放入 query string。
- 所有 conversation、message、artifact、run、agent config 等用户资源必须做对象级授权。
- 错误响应必须脱敏。
- 返回字段必须裁剪，不得暴露内部服务 URL、service token、数据库路径、system prompt、Provider 原始响应。
- 大型下载必须使用受控接口或短期 URL。

禁止：

- 把 `/internal/**` 暴露给 Frontend。
- 把 Orchestrator Service endpoint 写进公开 API。
- 把 Child Agent endpoint 写进公开 API。
- 前端直接调用 LLM Provider。
- 公开 API 接收任意内部任务执行参数。

## 8. Gateway ↔ Orchestrator Service-to-Service Security

Gateway Service 与 Orchestrator Service 必须是两个独立进程。

服务间安全规则：

- Gateway 调 Orchestrator 必须使用 service-to-service auth。
- service token 与用户 token 必须分离。
- 用户 Authorization token 不得作为 service token 透传。
- Orchestrator 的内部 API 不得暴露给浏览器。
- 内部 API 必须有 timeout。
- 内部调用必须携带 requestId / traceId / runId。
- service token 不得进入日志、错误响应、Artifact、AgentCard、前端事件或 debug dump。
- Orchestrator 不处理浏览器登录态。
- Orchestrator 不直接写浏览器响应。

最小允许实现可以是：

- private network + `INTERNAL_SERVICE_TOKEN`
- mTLS
- service mesh identity
- 等价服务间认证机制

## 9. Orchestrator Security Boundary

Orchestrator 是内部编排服务，负责计划、校验、调度、fallback、聚合，但不得绕过安全边界。

规则：

- Planner 输出必须结构化并本地校验。
- Agent / capability / output 必须来自 Registry 或可信能力集合。
- Orchestrator 只能调用 enabled 且 healthy 的 Agent。
- fallback / retry 后的目标也必须重新校验。
- Orchestrator 不得信任 LLM 生成的权限声明。
- Orchestrator 不得把 system prompt、service token、LLM API key 传给不可信 Agent。
- Orchestrator 不得执行高危 Tool；高危动作必须进入 confirm_action 流程。

## 10. Child Agent / User Agent Trust Boundary

Child Agent 是能力提供服务，不是全局可信主体。

规则：

- Child Agent 只能执行自身声明并获授权的能力。
- Child Agent 输出必须被视为不可信内容。
- Child Agent 不得直接访问 Gateway 的用户 API。
- Child Agent 不得直接访问 Frontend。
- Child Agent 不得决定全局编排策略。
- 用户自建 Agent 默认处于更低 trustLevel，必须显式授权能力和资源范围。
- 用户自建 Agent 不得默认拥有 run_command、deploy、file_overwrite、secret_read 等高危能力。

## 11. Agent Registry / AgentCard / Health Security

Agent Registry 是 Agent 能力目录和可用性来源，但不是安全凭证本身。

规则：

- AgentCard 不得包含 API key、service token、数据库连接串、system prompt、内部文件路径、对象存储凭证。
- Registry 只能接收可信来源、白名单、签名配置或受控管理入口中的 Agent。
- Agent health check 只暴露健康状态，不暴露环境变量、堆栈、Provider key、内部网络拓扑。
- Frontend 只能看到脱敏 Agent 摘要。
- Orchestrator 只能调用 Registry 中 enabled 且 healthy 的 Agent。
- Agent 能力以 capability / inputModes / outputModes / permissions 表达，不通过 agentName 推断。

## 12. Auth / Authorization / Object-level Permission

认证解决“是谁”，授权解决“能做什么”。AgentHub 必须同时处理用户授权和服务授权。

用户授权规则：

- 用户只能访问自己有权限的 conversation、message、artifact、run。
- 群聊 participant 变更必须校验权限。
- Artifact 预览、下载、删除必须校验对象级权限。
- Agent 配置读取和修改必须校验权限。
- 失败时返回安全错误，不泄露资源是否存在的敏感细节。

服务授权规则：

- Gateway → Orchestrator 使用 service identity。
- Orchestrator → Child Agent 使用内部授权或可信网络约束。
- Orchestrator → LLM Provider 使用受控 Provider credential。
- service credential 不得被用户可见层读取。

## 13. LLM / Planner Security

LLM 与 Planner 都不可信。

规则：

- LLM 输出不得直接作为执行依据。
- Planner 输出必须 JSON parse + schema validation + capability validation。
- LLM 不得创造不存在的 Agent / capability / tool。
- LLM 不得绕过对象级授权。
- LLM 不得绕过 confirm_action。
- Prompt 中不得包含 API key、service token、数据库密码、完整内部拓扑、完整 system secret。
- 不可信 Agent 不得接收系统级 Prompt 或 Provider key。
- Prompt injection 不能改变安全策略。
- fallback plan 必须重新走同样的安全校验。
- 结构化输出失败不得执行下游动作。

## 14. Secret Management

Secret 包括但不限于：

- LLM API key
- service-to-service token
- JWT signing secret
- database password
- object storage credential
- deploy token
- OAuth secret
- webhook secret

规则：

- Secret 只能来自环境变量、secret manager 或等价机制。
- `.env.example` 只能写变量名和占位符，不写真实值。
- 文档、OpenAPI 示例、AgentCard、Artifact metadata、日志、trace、debug dump 都不得包含真实 Secret。
- Secret 不得硬编码到 Go / TypeScript / Dockerfile / Markdown 示例中。
- Secret 需要最小权限、轮换能力和访问审计。
- AI 生成代码必须经过 Secret 泄漏检查。

## 15. Artifact / Runtime Capability Security

Artifact 与 Runtime Capability 是 AgentHub 的核心体验，也是主要攻击面。

风险分级：

- 低风险：markdown_render、code_preview 只展示不执行。
- 中风险：file_download、terminal_output、generated_document。
- 高风险：web_preview、file_upload、run_command、deploy_action、file_overwrite、external_publish。

规则：

- code preview 默认只展示，不执行。
- markdown 必须防 XSS，不默认允许危险 HTML。
- web preview 必须使用 iframe sandbox。
- Artifact 内容不得直接插入主应用 DOM。
- Artifact metadata 不得包含 secret。
- 大型 Artifact 必须走 contentRef 或受控下载。
- 下载必须鉴权，URL 必须短期有效。
- 高风险 Runtime Capability 必须经过权限校验和 confirm_action。

## 16. iframe Sandbox / CSP / XSS Policy

`web_preview` 必须隔离。

规则：

- HTML 预览不得使用主应用 DOM 的 `innerHTML` 直接渲染。
- HTML 预览应使用 iframe sandbox。
- sandbox 默认最小权限，谨慎增加 `allow-*`。
- iframe 内容不得读取父页面 token、cookie、localStorage、sessionStorage。
- iframe 内容不得调用 Gateway 用户 API。
- markdown 渲染不得默认允许任意 HTML。
- 用户输入和 LLM 输出都必须按上下文编码或清洗。
- 外链应防止 tabnabbing。
- CSP 应作为额外防线，但不能替代输出编码和 sandbox。

## 17. Tool Action Risk Policy

Tool Action 必须按风险分级。

字段建议：

- actionType
- riskLevel
- requiredPermission
- requiresConfirmation
- timeoutMs
- auditRequired
- allowedScope

高风险动作包括：

- run_command
- deploy
- file_overwrite
- external_publish
- credential_update
- agent_install
- workspace_delete

规则：

- Tool 参数必须 schema validation。
- 高风险动作必须 confirm_action。
- 高风险动作必须审计。
- LLM 不能直接执行高风险动作。
- 权限不足时不得生成可执行命令。

## 18. confirm_action Policy

`confirm_action` 是高危动作的统一安全门。

规则：

- 用户确认前不得执行动作。
- 确认内容必须包含动作类型、目标资源、影响范围、风险等级。
- 确认不能只依赖 LLM 文本。
- 确认记录必须包含 userId、runId、actionId、timestamp、result。
- 确认后仍需做最终权限校验。
- 超时或取消视为拒绝。

## 19. run_command / deploy Policy

run_command / deploy 是高危能力，不得默认启用。

规则：

- 必须显式声明权限。
- 必须限制 workspace。
- 必须有 timeout。
- 必须有命令白名单或策略检查。
- 禁止读取 secret 文件。
- 禁止后台长期进程。
- 禁止无限网络访问。
- 执行输出必须脱敏。
- 必须记录审计日志。
- deploy 必须有目标环境限制和确认。

## 20. file_upload / file_download Policy

上传文件不可信。

file_upload 必须：

- 限制大小。
- 校验 MIME type。
- 校验扩展名。
- 私有存储。
- 绑定 owner / conversation / run。
- 不直接执行。
- 不直接注入 Prompt。
- 必要时安全扫描。

file_download 必须：

- 鉴权。
- 对象级授权。
- 短期 URL 或受控下载接口。
- 明确 Content-Type。
- 安全 Content-Disposition。
- 不暴露内部存储路径或签名凭证。

## 21. Rate Limit / Quota / Resource Limit

所有消耗型能力必须有限制。

规则：

- Gateway public API 必须有基础 rate limit。
- SSE / stream 必须有连接数和超时限制。
- Orchestrator run 必须有最大 task 数。
- Planner 必须有 timeout。
- LLM 请求必须有 token、timeout、retry 上限。
- fallback / retry 必须有最大次数。
- Child Agent 调用必须有 timeout。
- Artifact 大小必须有限制。
- file_upload 必须有限制。
- Docker Demo 也不得依赖无限资源。

## 22. Error Redaction

对外错误只能包含安全信息。

允许：

- code / errorCode
- safeMessage
- requestId
- runId

禁止：

- stack trace
- API key
- service token
- internal URL
- database DSN
- absolute path
- system prompt
- raw provider response
- object storage signed credential

规则：

- Provider 原始错误不得直接返回用户。
- Orchestrator 内部错误必须映射为安全错误。
- Agent 错误必须脱敏后才能进入前端事件。
- Debug dump 也必须脱敏。

## 23. Logging / Audit

安全相关事件必须可审计。

应审计：

- 登录失败
- 未授权访问
- service-to-service auth 失败
- permission denied
- Agent Registry 变更
- Agent health 异常
- LLM Provider key 更新
- high-risk tool call
- confirm_action
- file upload / download
- deploy
- run_command
- fallback / retry

日志规则：

- 使用结构化日志。
- 日志必须有 requestId / traceId / runId。
- 日志不得包含 secret。
- 原始用户隐私输入、完整 Prompt、完整 LLM 原始响应不得默认落日志。
- 审计日志要可追踪，但必须脱敏。

## 24. Security Contract Tests

安全契约必须能被测试。

最小测试项：

- 未鉴权访问公开 API 返回 401。
- 无权限访问他人资源返回 403 或安全 404。
- `/internal/**` 不可被 Frontend 访问。
- Gateway → Orchestrator 没有 service token 时失败。
- AgentCard 不包含 secret。
- Health Check 不暴露堆栈或环境变量。
- 错误响应不包含 stack trace / API key。
- web_preview iframe 有 sandbox。
- markdown 不执行危险 HTML。
- high-risk tool 未确认不得执行。
- fallback 不调用 disabled / unhealthy Agent。
- smoke test 包含关键安全检查。

## 25. Historical MVP Security Profile

MVP v0.1 已完成，仅作为历史兼容和回归测试基线。

历史基线包括：

- 固定 Token 或环境变量 Token。
- 简化用户鉴权。
- 单 Agent 最小链路。
- Docker 内网运行 Agent。
- code_preview 只展示不执行。
- LLM API key 来自环境变量。
- 错误脱敏。

这些历史规则不得继续作为 v1.0 的能力上限或当前开发禁令。

## 26. v1.0 Sprint Security Profile

v1.0 安全 Profile 必须覆盖：

- Gateway 与 Orchestrator 分进程。
- 2+ Agent。
- 单聊 + 群聊。
- LLM 意图编排。
- Agent Registry + Health Check。
- 结构化多轮消息。
- code / webpage / markdown 等丰富产物。
- fallback / retry。
- ordered_parallel。
- Docker Demo。
- smoke test。
- AI 协作文档。

Sprint 示例中的具体 Agent 名称只能作为示例，不能成为安全规则的固定分支。

## 27. Review Checklist

安全 Review 必须检查：

- 是否把 MVP v0.1 降级为 Historical Profile。
- Gateway 与 Orchestrator 是否分进程。
- Frontend 是否不能直连 Orchestrator。
- Orchestrator `/internal/**` 是否没有暴露给前端。
- Gateway → Orchestrator 是否有 service-to-service auth。
- 用户 token 是否没有被当成 service token。
- API 是否有对象级授权。
- AgentCard 是否不泄密。
- Health Check 是否不泄密。
- LLM 输出是否先校验后执行。
- 高危 Tool 是否有 confirm_action。
- Artifact 渲染是否隔离。
- Secret 是否不进入日志 / Artifact / AgentCard / Prompt。
- 错误是否脱敏。
- 安全契约测试是否存在。

## 28. 完成定义

一次安全边界更新完成，必须满足：

- 本 Skill 与 references 已更新。
- docs/contracts 中的安全契约已更新。
- 新增能力已声明 trust boundary、permission、riskLevel、redaction、audit 和 test。
- 没有引入具体 Agent 名称硬编码。
- 没有把 Sprint 示例固化为长期限制。
- 没有生成业务实现代码。
