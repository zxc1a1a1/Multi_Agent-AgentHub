# AgentHub v1.0 Skills 使用报告

> 目的：说明当前仓库 `.claude/skills` 下 17 个 Skill 的定位、边界、调用方式，以及开发阶段如何使用它们完成 AgentHub v1.0 的架构、编排、Agent、前端、数据、安全、测试和交付工作。

---

## 1. 报告背景

当前 AgentHub 的 Skill 体系已经从 MVP 说明文档升级为 v1.0 开发契约体系。MVP v0.1 已完成，旧的单 Agent、硬编码路由、单聊、最小 `code_preview`、基础错误处理等限制，只保留为 Historical Profile / 回归基线。

当前 v1.0 开发目标来自 `SPRINT-v1.0-Plan.md`：多 Agent 协作、LLM 意图编排、群聊模式、AgentCard Registry、健康检查、结构化多轮消息、丰富产物预览、降级重试、Docker Demo、AI 协作文档和 smoke test。

本报告重点不是补丁说明，而是回答：

```text
开发时如何调用这些 Skill？
什么时候用哪个 Skill？
Claude Code 和 Codex 分别怎么调用？
如何避免破坏 Gateway / Orchestrator 分进程、通用 Agent、Contract-first、安全和测试边界？
```

---

## 2. 调用方式总览

### 2.1 Claude Code 调用方式

当前仓库使用 Claude Code 的 Skill 目录结构：

```text
.claude/skills/{skill-name}/SKILL.md
```

在 Claude Code 中，Skill 可以像 slash command 一样直接调用：

```text
/project-architecture 检查这个需求是否破坏 Gateway / Orchestrator 分进程边界
```

规则：

```text
- 以 /skill-name 开头。
- skill-name 通常就是 .claude/skills 下的目录名。
- 后面的文本作为任务参数。
- 也可以不显式调用，Claude 根据任务描述自动选择相关 Skill；但涉及架构、安全、测试、契约时，建议显式调用。
```

### 2.2 Codex 调用方式

Codex 的 Skill 调用方式和 Claude Code 不完全一样。

Codex 官方文档中，Skill 的仓库级目录是：

```text
.agents/skills/{skill-name}/SKILL.md
```

Codex 显式调用 Skill 有两种常用方式：

```text
/skills
```

打开 Skill 选择 / 浏览界面，然后选择要使用的 Skill。

或者在输入中用 `$` 明确提及某个 Skill：

```text
$project-architecture 检查这个需求是否破坏 Gateway / Orchestrator 分进程边界
```

规则：

```text
- Codex 的普通 /xxx 是 CLI slash command，用于控制会话，例如 /plan、/review、/model、/status、/skills。
- Codex 中不要默认写 /project-architecture 来调用仓库 Skill。
- Codex 推荐用 /skills 浏览并选择 Skill，或用 $skill-name 明确提及 Skill。
- Codex 也支持隐式调用：当任务与 Skill description 匹配时，Codex 可以自动选择 Skill。
```

### 2.3 当前仓库若要给 Codex 使用

当前仓库主要文件在：

```text
.claude/skills
```

如果要让 Codex 直接识别这些 Skill，建议同步或软链接到：

```text
.agents/skills
```

可选做法：

```bash
mkdir -p .agents
ln -s ../.claude/skills .agents/skills
```

或者复制一份：

```bash
mkdir -p .agents
cp -R .claude/skills .agents/skills
```

建议团队只维护一个事实源，另一个目录用生成脚本或软链接同步，避免 Claude 与 Codex 看到不同版本的 Skill。

---

## 3. 统一使用约束

所有 Skill 的使用都遵守以下约束：

```text
1. 一个任务优先选择一个主 Skill，不要主动扩展到无关 Skill。
2. 问“怎么改 / 如何设计 / 如何检查”时，只输出思路，不生成文件包。
3. 明确说“生成 / 输出文件包”时，才生成 zip 或文件。
4. Skill 文档必须中文。
5. Skill 必须独立可读，不能依赖其他 Skill 才能理解。
6. Skill 必须通用，不绑定 code-agent / web-agent / doc-agent。
7. Gateway 和 Orchestrator 必须分进程。
8. MVP v0.1 只作为 Historical Profile / 回归基线。
9. 当前开发面向 v1.0 和后续扩展：2+ Agent、群聊、LLM 编排、Registry、健康检查、丰富产物、降级重试。
10. 默认不包含业务实现代码，除非用户明确要求。
```

---

## 4. 17 个 Skill 定位与调用示例

### 4.1 `ai-collaboration-workflow`

**定位**：人类开发者与 AI 编程代理的协作流程契约。它不定义业务协议，而是定义 AI 如何接收任务、锁定范围、串行处理 Skill、生成文件包、验证、Review 和交接。

**什么时候用**：

```text
- 需要 AI 分析一个 Skill 怎么改。
- 需要控制“先思考，再生成”。
- 需要生成 zip 包并保持固定包结构。
- 需要检查 AI 是否越界扩展到其他 Skill。
```

**不负责**：具体 API、Agent 协议、数据库、前端事件、业务实现。

**Claude 调用**：

```text
/ai-collaboration-workflow 按当前规则检查这次 Skill 更新流程是否合规：只处理 testing-review-contract，不要生成文件包。
```

**Codex 调用**：

```text
$ai-collaboration-workflow 按当前规则检查这次 Skill 更新流程是否合规：只处理 testing-review-contract，不要生成文件包。
```

---

### 4.2 `project-architecture`

**定位**：AgentHub 项目级总架构事实源。它定义 v1.0 系统上下文、服务拓扑、Frontend / Gateway / Orchestrator / Child Agent / Data Layer 边界，并明确 Gateway 与 Orchestrator 必须分进程。

**什么时候用**：

```text
- 需求涉及系统架构或服务边界。
- 新增 Agent 类型、Conversation 模式、Artifact 类型。
- 检查 Gateway 是否越权做了编排。
- 检查 Sprint 示例是否被固化为长期架构。
```

**不负责**：具体 API 字段、内部 API schema、Agent 协议字段、数据库 DDL、业务代码。

**Claude 调用**：

```text
/project-architecture 检查“Gateway handler 中直接选择 Agent 并调用 Child Agent”是否违反项目架构边界。
```

**Codex 调用**：

```text
$project-architecture 检查“Gateway handler 中直接选择 Agent 并调用 Child Agent”是否违反项目架构边界。
```

---

### 4.3 `platform-api-contract`

**定位**：Frontend ↔ Gateway 公开 Platform API 契约。它以 OpenAPI 为公开 HTTP API 事实源，定义 `/api/**` 的资源边界、响应格式、错误码、鉴权、分页、Conversation / Message / Agent / Artifact / Run API。

**什么时候用**：

```text
- 新增或修改 Frontend 调 Gateway 的 REST API。
- 更新 docs/contracts/openapi.yaml。
- 设计 /api/agents、/api/conversations、/api/runs 等公开接口。
```

**不负责**：Gateway ↔ Orchestrator 的 `/internal/**`，子 Agent endpoint，实时事件细节，编排计划 schema。

**Claude 调用**：

```text
/platform-api-contract 设计 /api/agents 的公开响应，只返回前端可见 Agent 摘要，不暴露内部 URL、token、system prompt。
```

**Codex 调用**：

```text
$platform-api-contract 设计 /api/agents 的公开响应，只返回前端可见 Agent 摘要，不暴露内部 URL、token、system prompt。
```

---

### 4.4 `gateway-orchestrator-contract`

**定位**：Gateway Service ↔ Orchestrator Service 的分进程内部通信契约。它定义内部 API、流式事件、OrchestratorRequest、OrchestrationPlan、OrchestratorResult、cancel / timeout、service-to-service auth、trace 和 SafeError。

**什么时候用**：

```text
- Gateway 调 Orchestrator。
- 设计内部 stream。
- 设计 run cancel / timeout。
- 检查 Gateway 是否把编排逻辑写进 handler。
```

**不负责**：公开 Platform API、前端事件完整 schema、子 Agent 协议、LLM Provider SDK。

**Claude 调用**：

```text
/gateway-orchestrator-contract 评审这个 OrchestratorRequest 是否适合分进程调用，重点检查 JSON 可序列化、traceId、runId、timeout 和 service-to-service auth。
```

**Codex 调用**：

```text
$gateway-orchestrator-contract 评审这个 OrchestratorRequest 是否适合分进程调用，重点检查 JSON 可序列化、traceId、runId、timeout 和 service-to-service auth。
```

---

### 4.5 `intent-orchestration-contract`

**定位**：用户意图 → 结构化 OrchestrationPlan 的编排契约。它定义 PlannerInput、PlanningMode、OrchestrationPlan、TaskPlan、AgentCapabilitySet、Plan Validation、`single / ordered_parallel / sequential`、fallback / retry、群聊 @mention 和 PlanTrace。

**什么时候用**：

```text
- 设计 LLM Planner。
- 决定用户请求应该调用哪个 Agent。
- 设计多 Agent 拆任务。
- 校验 LLM 输出 plan。
- 设计 fallback / retry。
```

**不负责**：Gateway API、内部服务 API、LLM SDK、前端事件、数据库 DDL。

**Claude 调用**：

```text
/intent-orchestration-contract 为“做一个计数器应用，要前端页面和后端 API”设计 OrchestrationPlan。不要写死 code-agent/web-agent，只基于 availableAgents 的 capabilityId 和 outputTypes 选择任务。
```

**Codex 调用**：

```text
$intent-orchestration-contract 为“做一个计数器应用，要前端页面和后端 API”设计 OrchestrationPlan。不要写死 code-agent/web-agent，只基于 availableAgents 的 capabilityId 和 outputTypes 选择任务。
```

---

### 4.6 `llm-provider-contract`

**定位**：统一 LLM Provider Adapter 契约。它定义 Provider Registry、Model Registry、LLMRequest / LLMResponse、LLMStreamEvent、结构化输出、本地 schema 校验、tool use 边界、timeout / retry / rate limit、fallback、usage / cost、secret 和错误归一化。

**什么时候用**：

```text
- 接入 OpenAI / Anthropic / Gemini / OpenAI-compatible / local model。
- 修改 Planner 或 Agent 的模型调用方式。
- 设计 structured output。
- 设计 Provider fallback。
```

**不负责**：业务 Prompt 内容、Agent 编排策略、Gateway API、数据库、前端渲染。

**Claude 调用**：

```text
/llm-provider-contract 检查 Planner LLM 调用是否符合 structured output + 本地 JSON Schema validation 规则，失败时必须进入 fallback。
```

**Codex 调用**：

```text
$llm-provider-contract 检查 Planner LLM 调用是否符合 structured output + 本地 JSON Schema validation 规则，失败时必须进入 fallback。
```

---

### 4.7 `a2a-agent-contract`

**定位**：Orchestrator ↔ Child Agent 的通用 A2A 接入契约。它定义 AgentCard、Agent Registry、健康检查、A2A Streaming Task、ArtifactDraft 输出、错误归一、Registry discovery 和 Child Agent 生命周期。

**什么时候用**：

```text
- 新增 Child Agent。
- 修改 AgentCard。
- 修改 /health。
- 修改 Orchestrator 调 Agent。
- 修改 A2A Streaming Task。
```

**不负责**：Agent 内部 Runtime 实现、前端事件、Gateway 公开 API。

**Claude 调用**：

```text
/a2a-agent-contract 设计一个“图表生成类 Agent”的 AgentCard 和 A2A 输出边界，要求能力声明通用，不写死具体 Agent 名称。
```

**Codex 调用**：

```text
$a2a-agent-contract 设计一个“图表生成类 Agent”的 AgentCard 和 A2A 输出边界，要求能力声明通用，不写死具体 Agent 名称。
```

---

### 4.8 `adk-runtime-contract`

**定位**：AgentHub 自定义 ADK Runtime 契约。它定义任意 Child Agent 内部如何基于 `config.yaml`、AgentCard、A2A Server、Task Handler、Runtime Context API、LLMClient、ArtifactDraft、Tool 权限和错误处理实现。

**什么时候用**：

```text
- 写 agents/adk。
- 新增 Agent 目录。
- 改 Agent handler。
- 改 ctx.StreamText / ctx.AddArtifact。
- 改 LLMClient 生命周期。
- 设计 Agent 内部工具权限。
```

**不负责**：前端事件、Gateway API、全局编排计划。

**Claude 调用**：

```text
/adk-runtime-contract 检查这个 Agent handler 是否符合 Runtime Context、LLMClient 单例、ArtifactDraft 和安全错误规则。
```

**Codex 调用**：

```text
$adk-runtime-contract 检查这个 Agent handler 是否符合 Runtime Context、LLMClient 单例、ArtifactDraft 和安全错误规则。
```

---

### 4.9 `agui-event-contract`

**定位**：Frontend ↔ Gateway 的 AG-UI / SSE 实时事件流契约。它定义 Run 生命周期事件、文本消息事件、ToolCall 事件、STATE_UPDATE、多 Agent 消息归属、错误脱敏、SSE wire format 和前端聚合规则。

**什么时候用**：

```text
- 修改 SSE 事件。
- 修改 messageStore。
- 修改 Gateway event 输出。
- 修改 ToolCall 分片。
- 修改 STATE_UPDATE。
- 处理多 Agent 消息归属。
```

**不负责**：内部 OrchestratorStreamEvent、A2A 原始事件、Artifact schema。

**Claude 调用**：

```text
/agui-event-contract 检查这组多 Agent SSE 事件是否能稳定表达 ordered_parallel 输出，每个 Agent 是否有独立 messageId。
```

**Codex 调用**：

```text
$agui-event-contract 检查这组多 Agent SSE 事件是否能稳定表达 ordered_parallel 输出，每个 Agent 是否有独立 messageId。
```

---

### 4.10 `frontend-runtime-skills-contract`

**定位**：前端 Runtime Capability 注册与执行契约。它定义前端收到完整 Tool Call 后，如何按 `toolName + schema + component + behavior + riskLevel + failureMode` 执行渲染或降级，不绑定 Agent 名称。

**什么时候用**：

```text
- 新增或修改 code_preview、web_preview、markdown_render。
- 新增图表、文件、下载、确认动作等前端能力。
- 评审 ToolCall args schema。
- 检查 web_preview iframe 安全。
```

**不负责**：服务端如何生成 ToolCall、Artifact 持久化、Agent 协议。

**Claude 调用**：

```text
/frontend-runtime-skills-contract 设计 chart_preview 的前端 Runtime Capability，包含 toolName、参数 schema、组件绑定、failureMode 和安全等级。
```

**Codex 调用**：

```text
$frontend-runtime-skills-contract 设计 chart_preview 的前端 Runtime Capability，包含 toolName、参数 schema、组件绑定、failureMode 和安全等级。
```

---

### 4.11 `artifact-contract`

**定位**：AgentHub 产物事实源契约。它定义 ArtifactDraft 与 Core Artifact、Artifact 类型注册、`content / contentRef`、生命周期、版本、source / links、previewType、schema 校验和安全边界。

**什么时候用**：

```text
- Agent 输出代码、网页、Markdown、文档、文件引用。
- 设计 Artifact 类型。
- 判断大内容是否走 contentRef。
- 设计产物版本和消息关联。
```

**不负责**：前端组件实现、数据库 DDL、Agent 具体 handler。

**Claude 调用**：

```text
/artifact-contract 设计 webpage artifact 的元数据和 contentRef 规则，不把完整大 HTML 永久塞进 message.content。
```

**Codex 调用**：

```text
$artifact-contract 设计 webpage artifact 的元数据和 contentRef 规则，不把完整大 HTML 永久塞进 message.content。
```

---

### 4.12 `data-persistence-contract`

**定位**：数据持久化事实源契约。它定义 Conversation、Message、Run、Agent、Artifact、ToolCall 的关系，支持群聊、多 Agent、运行记录、健康状态、迁移、JSON 字段、索引、软删除和数据安全。

**什么时候用**：

```text
- 改数据库模型。
- 改消息历史。
- 设计 participants。
- 设计 runs / run_steps。
- 设计 artifacts / tool_calls。
- 设计 Agent 状态和健康缓存。
```

**不负责**：API response envelope、前端组件、Agent 协议完整字段。

**Claude 调用**：

```text
/data-persistence-contract 检查群聊消息模型是否能支持一个 run 产生多条 Agent 回复，并通过 senderName 区分来源。
```

**Codex 调用**：

```text
$data-persistence-contract 检查群聊消息模型是否能支持一个 run 产生多条 Agent 回复，并通过 senderName 区分来源。
```

---

### 4.13 `docker-compose-delivery`

**定位**：本地开发、Demo 和验收的一键启动交付契约。它定义 Compose 文件、服务拓扑、多 Child Agent 服务、环境变量、secret、healthcheck、profiles、Makefile、smoke test、volume / network、日志和 Demo checklist。

**什么时候用**：

```text
- 改 docker-compose.yml / compose.yaml。
- 新增服务。
- 改 .env.example。
- 改 healthcheck。
- 改 smoke-test。
- 改 Makefile。
- 准备 Demo。
```

**不负责**：生产部署、安全策略完整定义、业务实现代码。

**Claude 调用**：

```text
/docker-compose-delivery 检查当前 compose 是否满足 v1.0 Demo：Gateway/Orchestrator 分进程、2+ Agent、health check、smoke test。
```

**Codex 调用**：

```text
$docker-compose-delivery 检查当前 compose 是否满足 v1.0 Demo：Gateway/Orchestrator 分进程、2+ Agent、health check、smoke test。
```

---

### 4.14 `security-boundary-contract`

**定位**：全链路安全边界契约。它覆盖 Frontend、Gateway、Orchestrator、Child Agent、LLM、Registry、Artifact、Tool、文件、部署、密钥、鉴权、授权、sandbox、错误脱敏、审计和安全测试。

**什么时候用**：

```text
- 任何涉及 token、权限、LLM 输出执行、Artifact 渲染、iframe、file_upload、run_command、deploy、Registry、AgentCard、internal API 的改动。
```

**不负责**：完整 API schema、数据库 DDL、具体实现代码。

**Claude 调用**：

```text
/security-boundary-contract 审查这个 PR 是否存在用户 token 透传给 Orchestrator、AgentCard 泄密、LLM 绕过 confirm_action 的问题。
```

**Codex 调用**：

```text
$security-boundary-contract 审查这个 PR 是否存在用户 token 透传给 Orchestrator、AgentCard 泄密、LLM 绕过 confirm_action 的问题。
```

---

### 4.15 `observability-debugging-contract`

**定位**：全链路可观测性与排障契约。它定义 trace context、关联 ID、结构化日志、span、metrics、错误码、redaction、debug dump、Gateway / Orchestrator 分进程排障、多 Agent 排障。

**什么时候用**：

```text
- 增加日志、trace、metrics、errorCode。
- 设计 debug dump。
- 排查 fallback。
- 排查 stream disconnect。
- 串联 Agent / LLM / Artifact / ToolCall。
```

**不负责**：业务协议完整字段、具体日志库实现、监控系统部署。

**Claude 调用**：

```text
/observability-debugging-contract 给 Gateway → Orchestrator → Agent → LLM 的一次 run 设计最小日志字段，要求能定位 fallback 和 stream disconnect。
```

**Codex 调用**：

```text
$observability-debugging-contract 给 Gateway → Orchestrator → Agent → LLM 的一次 run 设计最小日志字段，要求能定位 fallback 和 stream disconnect。
```

---

### 4.16 `testing-review-contract`

**定位**：质量门禁与测试 Review 契约。它定义测试分层、Contract / Schema 测试、分进程集成测试、Planner / Registry / Artifact / Streaming / Frontend / E2E / Security 测试、PR Review 和 CI Gate。

**什么时候用**：

```text
- 补测试。
- 设计 CI。
- Review PR。
- 写 smoke test。
- 验证 Demo 路径。
- 判断是否能合并。
```

**不负责**：具体业务实现、测试框架细节、完整安全策略。

**Claude 调用**：

```text
/testing-review-contract 为“群聊多 Agent + web_preview + fallback”补测试矩阵，包括 unit、contract、stream replay、E2E smoke 和安全回归。
```

**Codex 调用**：

```text
$testing-review-contract 为“群聊多 Agent + web_preview + fallback”补测试矩阵，包括 unit、contract、stream replay、E2E smoke 和安全回归。
```

---

### 4.17 `code-style-and-conventions`

**定位**：全项目代码风格与工程质量契约。它覆盖 Go、TypeScript、React、JSON、YAML、Markdown、Contract 文档、测试、错误处理、日志、提交信息和 AI 生成代码 Review。

**什么时候用**：

```text
- 写代码。
- 重构。
- 改测试。
- 改文档。
- 生成 AI 代码。
- 写 commit message。
- 做风格 Review。
```

**不负责**：业务协议、架构边界、数据库事实源。

**Claude 调用**：

```text
/code-style-and-conventions 检查这次 Go 和 React 改动是否符合项目代码风格、错误处理、日志和测试约定。
```

**Codex 调用**：

```text
$code-style-and-conventions 检查这次 Go 和 React 改动是否符合项目代码风格、错误处理、日志和测试约定。
```

---

## 5. 开发阶段组合用法

### 5.1 需求分析 / 架构判断

主 Skill：

```text
/project-architecture
$project-architecture
```

辅助 Skill：

```text
/ai-collaboration-workflow
$ai-collaboration-workflow
```

常用 Prompt：

```text
/project-architecture 检查“新增一个图表生成 Agent”是否需要修改项目总架构。要求不写死 agentName，只基于 capability、Registry 和 Artifact 类型判断。
```

Codex：

```text
$project-architecture 检查“新增一个图表生成 Agent”是否需要修改项目总架构。要求不写死 agentName，只基于 capability、Registry 和 Artifact 类型判断。
```

---

### 5.2 API 与服务边界

公开 API：

```text
/platform-api-contract
$platform-api-contract
```

内部 API：

```text
/gateway-orchestrator-contract
$gateway-orchestrator-contract
```

常用 Prompt：

```text
/platform-api-contract 设计创建群聊 Conversation 的公开 API，要求支持 participants，但不暴露 Orchestrator 内部字段。
```

Codex：

```text
$platform-api-contract 设计创建群聊 Conversation 的公开 API，要求支持 participants，但不暴露 Orchestrator 内部字段。
```

---

### 5.3 编排与 LLM

主 Skill：

```text
/intent-orchestration-contract
$intent-orchestration-contract
```

辅助 Skill：

```text
/llm-provider-contract
$llm-provider-contract
```

常用 Prompt：

```text
/intent-orchestration-contract 评审这个 Planner 输出：检查 agentName 是否来自 Registry，capabilityId 是否存在，strategy 是否应为 ordered_parallel。
```

Codex：

```text
$intent-orchestration-contract 评审这个 Planner 输出：检查 agentName 是否来自 Registry，capabilityId 是否存在，strategy 是否应为 ordered_parallel。
```

---

### 5.4 新增 Child Agent

主 Skill：

```text
/adk-runtime-contract
$a2a-agent-contract
```

Claude 常用：

```text
/adk-runtime-contract 设计一个新 Agent 的 Runtime Context 使用规则，包括 LLMClient、StreamText、AddArtifact、timeout 和 safe error。
```

Codex 常用：

```text
$adk-runtime-contract 设计一个新 Agent 的 Runtime Context 使用规则，包括 LLMClient、StreamText、AddArtifact、timeout 和 safe error。
```

---

### 5.5 前端事件与产物

主 Skill：

```text
/agui-event-contract
/frontend-runtime-skills-contract
/artifact-contract
```

Codex：

```text
$agui-event-contract
$frontend-runtime-skills-contract
$artifact-contract
```

常用 Prompt：

```text
/frontend-runtime-skills-contract 设计 markdown_render 的失败降级规则，要求危险 HTML 不执行，渲染失败时保留纯文本。
```

Codex：

```text
$frontend-runtime-skills-contract 设计 markdown_render 的失败降级规则，要求危险 HTML 不执行，渲染失败时保留纯文本。
```

---

### 5.6 数据、历史消息与持久化

主 Skill：

```text
/data-persistence-contract
$ data-persistence-contract  # 注意：Codex 实际输入不要加空格，应写 $data-persistence-contract
```

正确 Codex 写法：

```text
$data-persistence-contract 检查 messages / runs / artifacts 是否能支持群聊中一个 run 产生多条 Agent 回复。
```

---

### 5.7 安全、观测、测试

主 Skill：

```text
/security-boundary-contract
/observability-debugging-contract
/testing-review-contract
```

Codex：

```text
$security-boundary-contract
$observability-debugging-contract
$testing-review-contract
```

常用 Prompt：

```text
/testing-review-contract 为“Planner LLM 返回非法 JSON → fallback → STATE_UPDATE → 前端提示”设计测试矩阵。
```

Codex：

```text
$testing-review-contract 为“Planner LLM 返回非法 JSON → fallback → STATE_UPDATE → 前端提示”设计测试矩阵。
```

---

### 5.8 Docker Demo / 交付

主 Skill：

```text
/docker-compose-delivery
$docker-compose-delivery
```

常用 Prompt：

```text
/docker-compose-delivery 检查 docker compose 是否满足 v1.0 Demo：frontend、gateway、orchestrator、2+ child agents、database、health check、smoke test。
```

Codex：

```text
$docker-compose-delivery 检查 docker compose 是否满足 v1.0 Demo：frontend、gateway、orchestrator、2+ child agents、database、health check、smoke test。
```

---

## 6. Skill 分层关系

```text
A. 协作与总架构
- ai-collaboration-workflow
- project-architecture

B. API 与服务边界
- platform-api-contract
- gateway-orchestrator-contract

C. 编排、模型与 Agent 接入
- intent-orchestration-contract
- llm-provider-contract
- a2a-agent-contract
- adk-runtime-contract

D. 前端事件、前端能力与产物
- agui-event-contract
- frontend-runtime-skills-contract
- artifact-contract

E. 数据与交付
- data-persistence-contract
- docker-compose-delivery

F. 安全、观测、测试、代码质量
- security-boundary-contract
- observability-debugging-contract
- testing-review-contract
- code-style-and-conventions
```

---

## 7. 红线规则

开发、Review、调用 Skill 时都要优先检查这些红线：

```text
- Frontend 不得直接访问 Orchestrator。
- Frontend 不得直接访问 Child Agent。
- Gateway 不得做 LLM 编排。
- Gateway 不得直接调用 Child Agent。
- Gateway 不得直接调用 LLM Provider。
- Orchestrator 不得直接暴露给 Frontend。
- Skill 不得写死 code-agent / web-agent / doc-agent。
- Sprint 示例不得固化为长期架构限制。
- LLM 输出不得直接执行，必须 schema validation。
- AgentCard 不得包含 secret、system prompt、内部 URL。
- Artifact / ToolCall / Runtime Capability 必须经过安全边界。
- 改 contract 必须补测试和 Review Checklist。
```

---

## 8. PR Review 推荐调用

Claude：

```text
/testing-review-contract 按 v1.0 质量门禁 Review 当前 PR：检查 contract、schema、Gateway/Orchestrator 分进程、Planner、Registry、Artifact、Streaming、fallback、安全回归和 smoke test。
```

Codex：

```text
$testing-review-contract 按 v1.0 质量门禁 Review 当前 PR：检查 contract、schema、Gateway/Orchestrator 分进程、Planner、Registry、Artifact、Streaming、fallback、安全回归和 smoke test。
```

再结合：

```text
$security-boundary-contract 检查当前 PR 是否泄露 secret、是否让 LLM 绕过 confirm_action、是否把 service token 暴露给前端。
```

```text
$observability-debugging-contract 检查当前 PR 是否保留 traceId、requestId、runId、messageId、agentTaskId、llmRequestId 的可追踪链路。
```

---

## 9. Claude 与 Codex 调用差异速查

| 场景 | Claude Code | Codex |
|---|---|---|
| 查看/选择 Skill | 输入 `/` 后选择 Skill，或直接 `/skill-name` | 输入 `/skills` 打开 Skill 选择，或用 `$skill-name` 提及 |
| 显式调用某个 Skill | `/project-architecture 任务描述` | `$project-architecture 任务描述` |
| 隐式调用 | 可以由 Claude 根据描述自动触发 | 可以由 Codex 根据 Skill description 自动触发 |
| Skill 目录 | `.claude/skills/{name}/SKILL.md` | `.agents/skills/{name}/SKILL.md` |
| 当前仓库直接可用性 | 当前仓库已使用 `.claude/skills` | 建议同步/软链到 `.agents/skills` |
| `/xxx` 的含义 | Skills 和 commands 已统一，通常可以 `/skill-name` | `/xxx` 主要是 Codex CLI slash command；Skill 推荐 `$skill-name` 或 `/skills` |

---

## 10. 总结

这 17 个 Skill 的作用不是替代开发，而是把 AgentHub v1.0 的开发过程变成可约束、可审查、可回归的流程。

使用时应遵循：

```text
1. 先确定任务属于哪个边界。
2. 显式调用对应 Skill。
3. 先产出契约 / 设计 / Review 思路。
4. 再实现代码。
5. 最后用 testing-review、security-boundary、observability-debugging 做合并前检查。
```

Claude Code 中使用 `/skill-name`；Codex 中使用 `/skills` 选择或 `$skill-name` 明确提及。两者调用方式不同，但目标一致：让 AI 在正确的契约边界内工作。

---

## 参考资料

- Claude Code Skills 文档：https://code.claude.com/docs/en/skills
- Codex Agent Skills 文档：https://developers.openai.com/codex/skills
- Codex CLI Slash Commands 文档：https://developers.openai.com/codex/cli/slash-commands
- AgentHub Sprint 事实源：`SPRINT-v1.0-Plan.md`
