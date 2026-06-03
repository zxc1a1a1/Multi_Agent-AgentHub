# AgentHub v1.0 Productization Stage 开发指导文档

## 1. 阶段定位

AgentHub 当前已经完成了三个基础阶段：

1. 模块解耦与 Runtime 基础设施阶段
2. 子 Agent 能力池与多模态契约阶段
3. 新架构 Orchestrator vertical slice 阶段

当前后端新架构主链路已经形成：

```text
Frontend / Client
  → Gateway
  → Orchestrator
  → RulePlanner
  → OrchestrationPlan
  → PlanValidator
  → SingleExecutor / OrderedParallelExecutor
  → code-agent / web-agent
  → Orchestrator summary
  → Gateway SSE
```

因此，下一阶段不再继续沿用 Phase 8 编号，而是进入新的开发阶段：

```text
AgentHub v1.0 Productization Stage
```

中文可称为：

```text
AgentHub v1.0 产品化与能力池接入阶段
```

本阶段目标：

```text
把已经跑通的新架构链路，
整理成可演示、可持久化、可扩展、可逐步接入更多 Agent 的平台基础。
```

本阶段重点不是继续证明链路能跑通，而是解决：

1. 项目事实源是否统一
2. 前端是否能正确展示多 Agent 输出
3. Conversation / Run / Message / Artifact 是否能持久化
4. Artifact 是否能形成结构化预览
5. 旧 Agent 能力池如何按优先级接入新架构
6. AgentCard / Registry 如何从静态配置走向长期能力系统
7. LLMPlanner 如何在不破坏稳定 smoke 的前提下受控接入

---

## 2. 当前项目结构理解

当前项目不是单一架构，而是过渡态结构：

```text
旧能力池：
  agents/

legacy 主链路：
  server/

新架构主路径：
  services/gateway
  services/orchestrator
  services/agents/code-agent
  services/agents/web-agent

Runtime 基础设施：
  pkg/adk
  pkg/adk/a2a
  pkg/runtime

前端：
  frontend/

架构与契约：
  docs/architecture
  docs/contracts
  docs/integration
  docs/refactor
  docs/reports

开发约束：
  .claude/skills
  .agents/skills
```

### 2.1 `services/*` 是当前新架构主路径

后续所有新架构开发默认进入：

```text
services/gateway
services/orchestrator
services/agents/*
```

其中：

```text
services/gateway:
  对外 HTTP API、SSE、Conversation API、Agent API、Gateway → Orchestrator client。

services/orchestrator:
  Planner、Plan、Validator、Executor、Dispatcher、Registry、内部 stream API。

services/agents/code-agent:
  当前已服务化的代码类 Agent，v0.1 mock response。

services/agents/web-agent:
  当前已服务化的 Web/UI 类 Agent，v0.1 mock response。
```

### 2.2 `agents/` 是旧 Agent 能力池

`agents/` 下仍保留大量旧 Agent，例如：

```text
vision-agent
file-agent
document-agent
ppt-agent
security-agent
test-agent
review-agent
deploy-agent
web-research-agent
artifact-agent
context-agent
```

这些不是当前新架构主运行链路的一部分。

当前定位应是：

```text
agents/ 是旧能力池与未来服务化候选池；
services/agents/ 是已经完成服务化、可被 Orchestrator 调用的 Agent。
```

后续不能一次性把所有旧 Agent 迁移到 `services/agents/`，必须按能力价值和契约成熟度分批服务化。

### 2.3 `server/` 是 legacy 路径

`server/` 代表旧主链路或历史兼容路径。

后续不建议继续扩展 `server/` 作为主路径。新功能默认进入：

```text
services/gateway
services/orchestrator
services/agents/*
```

除非明确是在做 legacy 兼容或迁移清理，否则不要在 `server/` 中新增主功能。

### 2.4 `docker-compose.new-arch.yml` 是新架构主 compose

当前新架构 demo 主路径应是：

```text
docker-compose.new-arch.yml
smoke-new-arch.sh
.github/workflows/new-arch-smoke.yml
```

旧 `docker-compose.yml` 应标注为 legacy 或历史路径，避免后续开发者误用。

---

## 3. 后续开发总原则

### 3.1 不回退 Gateway 直连 Agent

必须保持：

```text
Frontend → Gateway → Orchestrator → Agent
```

禁止回退为：

```text
Gateway → code-agent / web-agent
Gateway → 新增 Agent
Frontend → Orchestrator
Frontend → Agent
```

Gateway 的职责是：

```text
公开 API
鉴权
CORS
Conversation 入口
SSE 转换
Gateway → Orchestrator 内部调用
```

Gateway 不负责最终多 Agent 编排决策。

### 3.2 Orchestrator 是唯一编排层

多 Agent 调度必须进入 Orchestrator：

```text
RulePlanner / LLMPlanner
OrchestrationPlan
PlanValidator
Executor
Dispatcher
Registry
```

禁止为了快速 demo 绕过：

```text
PlanValidator
Executor
Registry
A2A Dispatcher
```

### 3.3 Conversation 是一级对象

后续所有单聊、群聊、跨轮任务、Agent 上下文记忆、Artifact 历史，都要围绕统一模型：

```text
Conversation
ConversationParticipant
Run
RunStep
Message
Artifact
```

不要用临时 context 拼接跨轮记忆。

### 3.4 Agent 能力不能长期硬编码

当前 `code-agent` 和 `web-agent` 是 demo profile 的两个默认 Agent。

长期应该从：

```text
agentName
```

升级到：

```text
agentId
AgentCard
capabilityIds
inputModes
outputTypes
runtimeCapabilities
health
status
```

### 3.5 CI 必须保持确定性

CI 默认不依赖：

```text
真实 LLM key
真实 OCR
外部文件解析服务
外部模型响应稳定性
```

CI 默认使用：

```text
RulePlanner
mock / deterministic Agent response
docker compose runtime smoke
```

LLMPlanner、真实 OCR、真实文件处理必须通过 feature flag 开启。

---

## 4. Skill 使用总则

后续开发必须按任务域选择 skill，而不是一次性全开所有 skill。

推荐模式：

```text
1 个主业务 skill
1 个边界 skill
1 个测试 skill
必要时加 security / observability / code-style
```

### Claude Code 调用方式

```text
/project-architecture
/gateway-orchestrator-contract
/testing-review-contract
```

### Codex 调用方式

```text
$project-architecture
$gateway-orchestrator-contract
$testing-review-contract
```

---

## 5. 每个 Skill 的理解与使用场景

### 5.1 `project-architecture`

作用：

```text
项目级总架构事实源。
```

负责判断：

```text
Gateway / Orchestrator / Agent / Data Layer 边界
服务拓扑
新旧架构职责
Conversation / Artifact / Registry / Runtime Capability 所在位置
Docker Demo 交付边界
架构 Review
```

使用场景：

```text
阶段规划
README 重写
current architecture 文档
legacy boundary 文档
重大架构判断
```

典型搭配：

```text
/project-architecture
/testing-review-contract
/code-style-and-conventions
```

---

### 5.2 `ai-collaboration-workflow`

作用：

```text
规范人类开发者与 AI 编程代理的协作方式。
```

负责：

```text
任务输入
范围控制
串行处理
先计划后修改
交接报告
修改清单
审计输出
```

使用场景：

```text
让 Claude / Codex 执行阶段任务
让 AI 生成文档
让 AI 修改代码
要求 AI 不扩范围
要求 AI 输出自查结论
```

---

### 5.3 `code-style-and-conventions`

作用：

```text
统一代码、文档、YAML、JSON 风格。
```

负责：

```text
Go 风格
TypeScript / React 风格
Markdown 风格
YAML workflow 风格
错误处理
日志格式
测试风格
```

使用场景：

```text
新增 Go package
修改前端组件
清理 workflow
新增 docs
更新 README
```

---

### 5.4 `testing-review-contract`

作用：

```text
全项目测试与 Review 质量门禁。
```

负责：

```text
unit test
contract test
service integration test
cross-process integration test
frontend component test
E2E smoke
security regression
CI quality gate
PR review checklist
```

使用场景：

```text
所有跨服务、Agent、SSE、Artifact、持久化、前端展示任务都必须使用。
```

后续高频搭配：

```text
/testing-review-contract
/security-boundary-contract
/observability-debugging-contract
```

---

### 5.5 `security-boundary-contract`

作用：

```text
全链路安全边界事实源。
```

负责：

```text
Frontend 安全
Gateway API 安全
Gateway ↔ Orchestrator 服务间安全
Child Agent 信任边界
Artifact 渲染安全
LLM / Tool / 文件 / Secret 安全
错误脱敏
审计日志
```

使用场景：

```text
web_preview sandbox
markdown raw HTML 禁用
Artifact contentRef
LLM key 接入
AgentCard 动态发现
run_error 脱敏
```

---

### 5.6 `commit-security-review`

作用：

```text
提交阶段安全审查。
```

负责：

```text
secret 泄露检测
.env / key / token / pem 检查
commit message 安全
PR 安全门禁
```

使用场景：

```text
引入环境变量
引入 LLM provider key
新增对象存储配置
修改 CI secrets
提交前检查
```

---

### 5.7 `docker-compose-delivery`

作用：

```text
本地开发、Demo、Compose、Smoke 的交付契约。
```

负责：

```text
compose 文件
服务拓扑
healthcheck
环境变量
profiles
Dockerfile
smoke test
doctor script
failure logs
cleanup
Demo checklist
```

使用场景：

```text
修改 docker-compose.new-arch.yml
新增 vision-agent-new 服务
修改 Dockerfile
清理 new-arch-smoke workflow
新增 runtime smoke
```

---

### 5.8 `platform-api-contract`

作用：

```text
Frontend ↔ Gateway 公开 API 契约。
```

负责：

```text
/api/**
Conversation API
Message API
Agent API
Run API
Artifact API
统一错误响应
分页
鉴权
trace header
Frontend API client
```

使用场景：

```text
修改 Gateway 对外 API
新增 /api/runs
新增 /api/artifacts
新增 /api/messages
修改前端 API client
```

禁止：

```text
Frontend 直连 Orchestrator
Frontend 直连 Agent
公开 /internal/** API
```

---

### 5.9 `gateway-orchestrator-contract`

作用：

```text
Gateway Service ↔ Orchestrator Service 内部通信契约。
```

负责：

```text
分进程边界
内部 API
streaming event
OrchestrationPlan 传递
fallback / retry
cancel / timeout
服务间鉴权
trace propagation
错误脱敏
Artifact delivery
attachment routing
```

使用场景：

```text
修改 Gateway 调 Orchestrator
修改 Orchestrator stream
增加 cancel / timeout
增加 traceId 透传
Attachment 转发
Artifact event 转换
```

---

### 5.10 `agui-event-contract`

作用：

```text
Frontend ↔ Gateway 的 AG-UI / SSE 实时事件契约。
```

负责：

```text
Run 生命周期事件
Text message events
State update events
Tool call events
Artifact events
Attachment events
multi-agent author / sender
错误事件
SSE wire format
前端聚合规则
```

使用场景：

```text
前端 multi-agent SSE 展示
message.delta 聚合
state.delta 展示
run_error 显示
ordered_parallel UI timeline
```

---

### 5.11 `frontend-runtime-skills-contract`

作用：

```text
前端 Runtime Capability 注册与渲染契约。
```

负责：

```text
toolName + schema + component binding
code_preview
web_preview
markdown_render
vision_analysis_card
file_summary_card
failureMode
sandbox
schema validation
riskLevel
```

使用场景：

```text
WebPreview Safe Mode
CodePreview
Markdown rendering
vision_analysis preview
file_summary preview
Artifact → component mapping
```

禁止：

```text
不要通过 agentName 决定前端组件。
```

应该通过：

```text
previewType
toolName
artifact.type
runtime capability registry
```

---

### 5.12 `artifact-contract`

作用：

```text
AgentHub 所有产物 Artifact 的标准事实源。
```

负责：

```text
artifactId
type
title
mimeType
content
contentRef
source
links
status
version
previewType
storage
安全边界
生命周期
```

使用场景：

```text
code-agent 输出 code artifact
web-agent 输出 webpage artifact
vision-agent 输出 vision_analysis artifact
Artifact 持久化
Artifact preview mapping
```

---

### 5.13 `data-persistence-contract`

作用：

```text
数据持久化事实源。
```

负责：

```text
Conversation
ConversationParticipant
Message
Agent
AgentHealthCheck
Run
RunStep
AgentTask
ToolCall
Artifact
migration
index
soft delete
JSON 字段
数据安全
```

使用场景：

```text
Conversation 持久化
Run / RunStep 持久化
Message 持久化
Artifact metadata 持久化
Agent registry persistence
```

---

### 5.14 `intent-orchestration-contract`

作用：

```text
用户意图 → 结构化计划 的编排契约。
```

负责：

```text
PlannerInput
PlanningMode
OrchestrationPlan
TaskPlan
capability validation
single / ordered_parallel / sequential
@mention
fallback / retry
LLM structured output
plan tracking
safe error
```

使用场景：

```text
RulePlanner
LLMPlanner
Plan schema
Validator
执行策略
vision-agent routing
file-agent routing
群聊 @mention routing
```

---

### 5.15 `a2a-agent-contract`

作用：

```text
Orchestrator ↔ Child Agent 的 A2A 协议契约。
```

负责：

```text
AgentCard
Agent Registry
/health
A2A Streaming Task
Artifact output
错误处理
Registry discovery
Agent lifecycle
multimodal parts
```

使用场景：

```text
新增任何 Child Agent 服务
服务化 vision-agent
服务化 file-agent
修改 code-agent AgentCard
修改 web-agent outputModes
新增 Agent healthcheck
```

---

### 5.16 `adk-runtime-contract`

作用：

```text
Child Agent 内部 ADK Runtime 实现契约。
```

负责：

```text
Agent 目录结构
config
AgentCard 生成
A2A Server 暴露
Task Handler
Runtime Context API
streaming output
Artifact output
LLMClient 生命周期
tool registration
runtime security
```

使用场景：

```text
vision-agent v0.1 服务化
file-agent 服务化
code-agent 从 mock 升级
web-agent 从 mock 升级
给 Agent 添加 tool / artifact output
```

区别：

```text
a2a-agent-contract 管对外协议。
adk-runtime-contract 管 Agent 内部实现。
```

---

### 5.17 `llm-provider-contract`

作用：

```text
统一 LLM Provider Adapter 契约。
```

负责：

```text
Provider Registry
Model Registry
LLMRequest / LLMResponse
stream normalization
structured output
tool use adapter boundary
timeout / retry / rate limit
fallback
secret
prompt template
usage / cost
error sanitization
```

使用场景：

```text
LLMPlanner feature flag
OpenAI / Anthropic / Gemini provider adapter
structured plan generation
Agent mock → real LLM
usage / cost tracking
```

禁止：

```text
CI 默认依赖真实 LLM
Provider SDK 细节散落到各 Agent
API key 写进代码或 compose
```

---

### 5.18 `observability-debugging-contract`

作用：

```text
全链路可观测性与排障契约。
```

负责：

```text
traceId
requestId
runId
stepId
agentTaskId
structured logs
span naming
metrics
error taxonomy
debug dump
Gateway / Orchestrator 分进程排障
Agent 调度排障
LLM 排障
敏感信息脱敏
```

使用场景：

```text
traceId 贯穿 Gateway → Orchestrator → Agent
RunStep 日志关联
debug dump failure-only
错误码标准化
Agent dispatch 失败定位
```

---

## 6. 下一阶段 Milestones

## Milestone 1：项目事实源收口

### 目标

统一当前项目事实，避免新旧路径混乱。

### 使用 skill

```text
/project-architecture
/ai-collaboration-workflow
/code-style-and-conventions
/testing-review-contract
```

### 任务

```text
1. 新增 current-architecture-state.md。
2. 新增 legacy-boundary.md。
3. 更新 README。
4. 明确 services/* 是主路径。
5. 明确 server/ 是 legacy。
6. 明确 agents/ 是旧能力池与未来服务化候选池。
```

### 验收

```text
README 能准确描述五服务新架构。
文档明确 Gateway 不再直连 Agent。
文档明确 Orchestrator 是独立进程。
文档明确 code-agent/web-agent 是 demo profile。
文档明确旧 agents/ 是能力池，不是已服务化 Agent 列表。
```

---

## Milestone 2：CI / Smoke 正式化

### 目标

把排障型 CI 变成正式质量门禁。

### 使用 skill

```text
/docker-compose-delivery
/testing-review-contract
/observability-debugging-contract
/security-boundary-contract
```

### 任务

```text
1. 精简 new-arch-smoke workflow。
2. 保留 smoke-new-arch.sh。
3. 保留 failure logs。
4. raw gateway/orchestrator debug 改为 failure-only 或移除。
5. 保持 single code / single web / mixed ordered_parallel smoke。
```

### 验收

```text
CI 成功时输出简洁。
CI 失败时能定位。
SMOKE TEST PASSED。
无 run_error。
无 panic/fatal。
无 API key 泄漏。
```

---

## Milestone 3：前端 Multi-Agent SSE 展示

### 目标

前端正确展示多 Agent 输出，而不是只在后端 smoke 中通过。

### 使用 skill

```text
/agui-event-contract
/frontend-runtime-skills-contract
/platform-api-contract
/testing-review-contract
/security-boundary-contract
```

### 任务

```text
1. 审计前端 SSE parser。
2. 审计 message store。
3. 按 author/sender 分组展示。
4. mixed 场景显示 web-agent、code-agent、orchestrator summary。
5. run_error 安全展示。
6. 补前端测试或 E2E。
```

### 验收

```text
single code 显示 code-agent。
single web 显示 web-agent。
mixed 显示 web-agent → code-agent → orchestrator summary。
不同 sender 不混成一个气泡。
错误不暴露内部细节。
```

---

## Milestone 4：Conversation / Run / Message 持久化

### 目标

把运行时消息变成可查询、可恢复、可审计的数据。

### 使用 skill

```text
/data-persistence-contract
/platform-api-contract
/gateway-orchestrator-contract
/testing-review-contract
/security-boundary-contract
```

### 任务

```text
1. 设计 Conversation / Participant / Run / RunStep / Message / Artifact schema。
2. 增加 migration。
3. Gateway store 从 memory 逐步迁移到持久化实现。
4. Orchestrator 执行过程写入 Run / RunStep。
5. SSE 输出与持久化消息建立对应关系。
```

### 验收

```text
刷新页面后历史消息不丢。
mixed run 能查到 web-agent、code-agent、orchestrator summary。
失败 run 能查到失败 step。
Message 保留 senderName / senderId。
```

---

## Milestone 5：Artifact 与 Runtime Preview 落地

### 目标

从纯文本 Agent 输出升级为结构化产物输出。

### 使用 skill

```text
/artifact-contract
/frontend-runtime-skills-contract
/agui-event-contract
/data-persistence-contract
/security-boundary-contract
/testing-review-contract
```

### 任务

```text
1. code-agent 输出 code artifact。
2. web-agent 输出 webpage artifact。
3. orchestrator summary 支持 markdown。
4. Artifact 进入持久化。
5. 前端 Runtime Capability 根据 previewType / toolName 渲染。
```

### 验收

```text
code artifact 能 code_preview。
webpage artifact 能 web_preview。
markdown 能 markdown_render。
前端不通过 agentName 决定组件。
web_preview sandbox。
Markdown raw HTML 默认禁用。
```

---

## Milestone 6：vision-agent 服务化与多模态入口

### 目标

把旧能力池中的 vision-agent v0.1 接入新架构，但不做真实 OCR。

### 使用 skill

```text
/a2a-agent-contract
/adk-runtime-contract
/artifact-contract
/intent-orchestration-contract
/frontend-runtime-skills-contract
/testing-review-contract
/security-boundary-contract
```

### 任务

```text
1. 新增 services/agents/vision-agent。
2. 实现 /health、AgentCard、A2A Server、Dockerfile。
3. 支持 image_ref / extracted_text / description 输入。
4. 输出 vision_analysis artifact。
5. Orchestrator RulePlanner 能把图像类输入路由给 vision-agent。
6. vision_analysis 可被 web-agent / code-agent 消费。
```

### 验收

```text
vision-agent 不读真实图片也可以。
vision-agent 不做真实 OCR 也可以。
必须能输出结构化 vision_analysis。
mixed multimodal demo 能出现 vision-agent → web-agent 或 vision-agent → code-agent。
```

---

## Milestone 7：AgentCard / Registry 契约化

### 目标

从 demo 静态 registry 走向通用 Agent 能力系统。

### 使用 skill

```text
/a2a-agent-contract
/platform-api-contract
/intent-orchestration-contract
/data-persistence-contract
/testing-review-contract
/security-boundary-contract
```

### 任务

```text
1. 统一 AgentCard 字段。
2. capabilityIds 来自 AgentCard.skills[].id。
3. inputModes / outputTypes 与 docs 契约对齐。
4. /api/agents 返回 AgentSummary。
5. 区分 agent status 与 health。
6. Validator 基于 Registry / AgentCard 校验。
```

### 验收

```text
新增 Agent 不需要改 handler。
Validator 不硬编码 agentName。
Frontend 不依赖固定 code-agent/web-agent。
/api/agents 能返回通用 AgentSummary。
```

---

## Milestone 8：LLMPlanner 受控接入

### 目标

在 RulePlanner 稳定的基础上引入 LLMPlanner。

### 使用 skill

```text
/intent-orchestration-contract
/llm-provider-contract
/security-boundary-contract
/testing-review-contract
/observability-debugging-contract
```

### 任务

```text
1. 增加 ORCHESTRATOR_PLANNER_MODE。
2. 支持 rule / llm / llm_with_rule_fallback。
3. LLMPlanner 输出 OrchestrationPlan。
4. 所有 LLM plan 必须经过 PlanValidator。
5. LLMPlanner 失败 fallback RulePlanner。
6. CI 默认仍使用 RulePlanner。
```

### 验收

```text
无 LLM key 时 demo 仍能跑。
llm mode 可手动开启。
非法 LLM plan 被 Validator 拒绝。
LLM 失败不会破坏 smoke。
usage / cost / error 可观测。
```

---

## 7. 后续 Agent 服务化顺序

不要一次性服务化所有旧 agents。

推荐顺序：

```text
1. vision-agent
2. file-agent
3. document-agent
4. test-agent
5. security-agent
6. ppt-agent
7. review-agent
8. deploy-agent
```

原因：

```text
vision-agent / file-agent 是多模态与文件入口。
document-agent / ppt-agent 是 Artifact 输出入口。
test-agent / security-agent 是质量与安全入口。
review-agent / deploy-agent 是后续协作和交付入口。
```

每个 Agent 服务化都必须同时满足：

```text
/a2a-agent-contract
/adk-runtime-contract
/testing-review-contract
/security-boundary-contract
```

如果 Agent 输出 Artifact，还必须加：

```text
/artifact-contract
/frontend-runtime-skills-contract
```

---

## 8. 后续开发禁止事项

### 8.1 不要扩大 Gateway 直连 Agent

禁止：

```text
Gateway 根据 agentName 直接调新 Agent
Frontend 绕过 Gateway
Frontend 直连 Orchestrator
Frontend 直连 Agent
```

### 8.2 不要一次性迁移全部 Agent

禁止：

```text
一次性把 agents/ 下所有 Agent 全搬进 services/agents/
```

必须按能力价值和契约成熟度逐步服务化。

### 8.3 不要提前引入不稳定外部依赖

禁止：

```text
CI 依赖真实 LLM key
CI 依赖真实 OCR
CI 依赖真实外部文件服务
CI 依赖外部模型响应稳定性
```

### 8.4 不要绕过 Validator

禁止：

```text
Planner 直接设置 validation.validated=true
Executor 接受未验证 plan
Handler 为了 smoke 跳过 Validator
```

### 8.5 不要通过 agentName 决定前端组件

禁止：

```text
if agentName == "web-agent" render WebPreview
```

应使用：

```text
artifact.type
previewType
toolName
runtime capability registry
```

---

## 9. 后续审计方式

每个 milestone 完成后，必须输出：

```text
1. 修改文件清单
2. 触发了哪些 skill
3. 核心设计说明
4. 禁止项自查
5. 测试命令与结果
6. smoke 或 E2E 结果
7. 剩余风险
8. 是否允许进入下一 milestone
```

审计重点：

```text
是否违反 Gateway / Orchestrator / Agent 边界
是否绕过 Validator
是否硬编码具体 Agent
是否破坏 deterministic CI
是否泄露内部错误或 secret
是否有真实 runtime 证据
```

---

## 10. 第一批推荐任务

下一步不要直接写 LLMPlanner，也不要马上服务化 vision-agent。

第一批建议：

```text
Task 1: 新增 productization-stage-guide.md
Task 2: 新增 current-architecture-state.md
Task 3: 更新 README
Task 4: 清理 new-arch-smoke workflow debug
Task 5: 审计 frontend multi-agent SSE 展示
```

这五个任务完成后，再进入：

```text
Conversation / Run / Message 持久化
Artifact 与 Runtime Preview
vision-agent 服务化
```

---

## 11. 第一批任务推荐 Prompt

### Claude Code Prompt

```text
/project-architecture
/ai-collaboration-workflow
/code-style-and-conventions
/testing-review-contract

当前进入 AgentHub v1.0 Productization Stage，不再沿用 Phase 8 编号。

请先读取：
- docs/architecture/**
- docs/contracts/**
- docs/refactor/**
- README 或 readme.md
- docker-compose.new-arch.yml
- smoke-new-arch.sh
- .github/workflows/new-arch-smoke.yml
- services/gateway/**
- services/orchestrator/**
- services/agents/code-agent/**
- services/agents/web-agent/**
- agents/** 目录结构

目标：
1. 新增 docs/refactor/productization-stage-guide.md。
2. 新增 docs/refactor/current-architecture-state.md。
3. 明确 services/* 是当前新架构主路径。
4. 明确 server/ 是 legacy 路径。
5. 明确 agents/ 是旧能力池与未来服务化候选池。
6. 明确 docker-compose.new-arch.yml 是新架构主 compose。
7. 明确当前已完成 Orchestrator vertical slice。
8. 明确下一阶段开发主线：CI 收口、前端 multi-agent SSE、持久化、Artifact、vision-agent、AgentCard、LLMPlanner feature flag。
9. 不修改业务代码。
10. 不生成新的 Agent。
11. 不改 Orchestrator / Gateway 逻辑。

完成后输出：
- 新增文档清单
- 当前架构判断
- 后续 milestone 列表
- 禁止项自查
```

### Codex Prompt

```text
使用 $project-architecture、$ai-collaboration-workflow、$code-style-and-conventions、$testing-review-contract。

任务：为 AgentHub 下一开发阶段新增指导文档和当前架构状态文档。

当前状态：
- 模块解耦已完成 pkg/adk、pkg/adk/a2a、pkg/runtime、services/gateway。
- 子 Agent 能力池仍在 agents/。
- services/agents 下已服务化 code-agent 和 web-agent。
- services/orchestrator 已完成 vertical slice。
- docker-compose.new-arch.yml 已形成新架构主路径。
- 后续进入 Productization Stage，不再沿用 Phase 8 编号。

只允许修改 docs/refactor 和 README/readme.md。
不要修改业务代码。

目标：
1. 新增 productization-stage-guide.md。
2. 新增 current-architecture-state.md。
3. 说明 new architecture 主路径。
4. 说明 legacy 路径。
5. 说明每个 skill 后续如何使用。
6. 说明后续主线：CI、Frontend SSE、Persistence、Artifact、Vision Agent、Registry、LLMPlanner。
7. 输出审计检查清单。

禁止：
- 不修改 services/**
- 不修改 agents/**
- 不修改 docker-compose
- 不新增 Agent
- 不引入 LLMPlanner
- 不删除 legacy 代码

完成后输出：
- diff 摘要
- 文档结构
- 后续开发建议
- 风险点
```

---

## 12. 阶段总结

当前项目已经不是“缺 Orchestrator”的状态。

当前真实状态是：

```text
Orchestrator vertical slice 已完成；
下一阶段要把它产品化，并把旧能力池按契约逐步接入。
```

后续最重要的不是继续堆 Agent，而是先解决：

```text
项目事实源统一
前端是否能承接多 Agent 输出
数据是否能持久化
Artifact 是否能结构化预览
Agent 能力是否能通过 AgentCard 管理
LLMPlanner 是否能安全、可回退地接入
```

只有这些完成后，AgentHub 才会从“新架构 demo”升级为“v1.0 多 Agent 协作平台基础版”。
