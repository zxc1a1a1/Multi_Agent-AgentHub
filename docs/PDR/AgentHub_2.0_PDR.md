# AgentHub 2.0 PDR

## 多会话 IM、可注册 Agent、LLM 可确认编排与版本化 Artifact 预览

**文档版本：** v0.4  
**文档状态：** AgentHub 2.0 产品与架构基线  
**项目定位：** 面向多 Agent 协作的即时通信系统  
**内置参考 Agent：** CodeAgent、WebAgent  
**Agent 扩展方式：** Built-in、Config、Remote Registration  
**执行方式：** 按依赖关系与退出门槛推进，不按时间划分  
**核心技术栈：** Go、React、TypeScript、SQLite、SSE/AG-UI、A2A、MCP、Sandpack  
**目标岗位：** Agent 开发、Agent 平台、Agent Harness、Agent 应用算法  

---

# 0. v0.4 更新摘要

v0.4 保留 v0.3 的产品定位和主体方案，只对已经确认的边界进行收口。

主要更新如下：

1. CodeAgent 和 WebAgent 改称“内置稳定参考 Agent”，不再表述为平台只能调用的两个 Agent。
2. 动态 Agent 注册成为 AgentHub 2.0 的正式能力。
3. Direct、Manual Multi-Agent、Auto 三种模式统一使用 Agent Registry 的可用性与授权结果。
4. Auto Mode 的候选集合由 Registry 动态生成，不再固定为 CodeAgent 和 WebAgent。
5. PlanStep 增加 `agent_version` 和可选 `skill_id`，确认和执行绑定准确的 PlanVersion 与 AgentVersion。
6. 新增 RegisteredAgent、AgentVersion、AgentHealthCheck、AgentCredential 和 RunAgentSnapshot 等领域对象。
7. 明确 Agent Registry、A2A、Planner、Context Manager 的职责边界。
8. AgentCard 刷新、健康检查、启用、禁用、注销和历史快照进入正式范围。
9. Artifact 明确为持久化、不可变版本资源；Preview 绑定准确 ArtifactVersion。
10. AG-UI 使用标准事件及 AgentHub 命名空间扩展，不再使用伪 Tool Call 表示计划确认或预览。
11. Compose 默认仍只包含 Frontend、Gateway、Orchestrator、CodeAgent、WebAgent；动态远程 Agent 不要求写入 Compose。
12. Planner、Synthesizer、Context Manager、摘要器、标题生成器和 Preview Renderer 均保持为内部模块，不注册为 Agent。

本次更新不扩大 2.0 的主要产品范围，也不把 AgentHub 改造成工作流编辑器、通用后端执行平台或无限自治系统。

---

# 1. 文档目标

本 PDR 用于指导 `Multi_Agent-AgentHub` 2.0 的重构、实现和验收。

AgentHub 2.0 不再定位为“多个专业 Agent 的能力展示集合”，也不收敛为“代码仓库自动审查平台”，而是定位为：

> **支持多会话管理、单 Agent 对话、用户选择多 Agent、系统自主选择 Agent、LLM 生成可编辑计划、用户确认后执行、动态注册符合协议的 Agent，以及代码产物在线预览的多 Agent IM 系统。**

2.0 的稳定能力范围包括：

- 完整的多会话 IM；
- Conversation、Message、Run、Plan、AgentInvocation 和 Artifact 持久化；
- 内置 CodeAgent；
- 内置 WebAgent；
- 动态注册符合 A2A 约束的 Agent；
- 单 Agent 指定调用；
- 用户指定多个 Agent；
- 系统从可用 Registry Catalog 中自主选择 Agent；
- LLM 生成结构化执行计划；
- 用户编辑、确认、拒绝或重新生成计划；
- 多 Agent 串行、并行和混合依赖执行；
- 分别展示或统一汇总 Agent 结果；
- HTML、CSS、JavaScript、React 项目预览；
- 对话级上下文管理；
- 长对话压缩和相关历史检索；
- 流式输出、断线恢复、取消和失败处理；
- Agent 注册、版本、健康、权限和历史快照；
- 可复现的 Mock、测试和可观测性链路。

---

# 2. 产品边界与非目标

## 2.1 AgentHub 是 IM 系统

用户直接面对的核心对象是：

- Conversation；
- Message；
- Agent；
- Plan；
- Run；
- AgentInvocation；
- Attachment；
- Artifact；
- Preview。

系统主界面保持聊天产品形态。

DAG、依赖图和调度波次是 Orchestrator 的内部执行表示，不作为产品主界面或产品定位。

## 2.2 2.0 不是以下系统

AgentHub 2.0 不定位为：

- 可视化工作流编辑器；
- 通用任务调度后台；
- 代码仓库自动审查专用平台；
- 固定十个 Agent 的展示系统；
- 无确认门槛的自治 Agent 循环；
- 任意本地命令和服务器后端执行平台；
- 通用云 IDE；
- 生产部署平台；
- 自动收集敏感长期用户记忆的系统。

## 2.3 来源优先级

实现和文档冲突时使用：

```text
当前用户明确决定
→ 已批准的 AgentHub 2.0 PDR
→ project-architecture
→ 对应 Active Domain Contract
→ JSON Schema / OpenAPI / ADR
→ 当前实现和测试
→ Legacy / Sprint / UML / 旧 Redesign
```

旧架构文档只作为历史参考，不能覆盖本 PDR 和 Active Contract。

---

# 3. 核心设计决策

## 3.1 内置 Agent 与可注册 Agent

AgentHub 2.0 默认提供两个内置稳定参考 Agent：

```text
CodeAgent
WebAgent
```

它们用于：

- 提供开箱即用的核心能力；
- 验证 Agent Runtime、A2A、Context、Artifact 和流式事件链路；
- 作为第三方 Agent 接入的参考实现；
- 支撑无外部 Agent 时的默认 Demo。

它们不是平台能够调用的全部 Agent。

符合平台协议、授权和安全要求的 Agent 可以通过以下来源进入 Registry：

```text
builtin
config
remote
```

未来 AgentHub 托管型 Agent 可使用单独的 `managed` 资源，但不与远程 A2A 注册混为一谈。

## 3.2 非 Agent 组件

以下组件属于 Orchestrator 或平台内部能力，不注册为 Agent：

```text
LLM Planner
Plan Parser
Plan Normalizer
Plan Validator
Plan Repairer
Confirmation Gate
Dependency Executor
Context Manager
Result Synthesizer
Conversation Summarizer
Auto Title Generator
Preview Renderer
```

不新增：

```text
PlannerAgent
ManagerAgent
AggregatorAgent
SummaryAgent
PreviewAgent
RepositoryAgent
SecurityAgent
TestAgent
DiffAgent
ReviewAgent
```

代码审查、测试建议、重构和前端项目生成是 CodeAgent 的 Skill。

搜索、页面读取、官方文档研究和来源整理是 WebAgent 的 Skill。

复杂的“检索后改代码”等过程是 Plan 或 Workflow，不是新的 Agent 类型。

## 3.3 多 Agent 编排必须由 LLM 生成 Plan

只要一次请求需要多个 Agent，就必须经过：

```text
用户请求
→ Eligible Registry Catalog
→ Context Bundle
→ LLM Planner
→ Parser
→ Normalizer
→ Deterministic Validator
→ 有限 Repair
→ PlanVersion
→ 用户确认
→ Executor
```

LLM 输出不能直接驱动执行。

## 3.4 用户确认是执行门槛

未经确认的多 Agent Plan 不得创建 AgentInvocation。

用户可以：

- 确认；
- 编辑 Goal；
- 编辑 Step；
- 调整依赖和顺序；
- 更换 Agent；
- 禁用步骤；
- 修改响应方式；
- 重新生成；
- 拒绝；
- 取消。

实质性 Replan 必须创建新的 PlanVersion 并重新确认。

实质性变化包括：

- 新增或删除 Step；
- 更换 Agent 或 Skill；
- 修改目标；
- 改变依赖结构；
- 改变预期输出类型；
- 增加新的权限或高风险操作。

## 3.5 网页预览是 Artifact 能力

代码项目预览流程：

```text
CodeAgent 或其他授权 Agent
→ WebProjectArtifact
→ ArtifactVersion
→ Gateway 授权读取
→ Sandpack Preview
```

Preview：

- 不是 Agent；
- 不是 Tool；
- 不注册 AgentCard；
- 不由 Orchestrator 伪装为 Tool Call；
- 不在主应用 DOM 中直接执行生成代码；
- 始终绑定准确 ArtifactVersion。

## 3.6 完整存储不等于完整上下文

```text
完整原始历史持久化
≠
每次把完整历史发送给模型或远程 Agent
```

Context Manager 根据：

- Use Case；
- Agent；
- Skill；
- PlanStep；
- Token Budget；
- Artifact；
- 用户授权；
- Conversation Policy；

生成受控的 ContextSnapshot。

---

# 4. 用户调用模式

## 4.1 Direct Mode

用户选择一个可调用 Agent：

```text
用户
→ 一个 Eligible Registered Agent
→ Agent 回复或 Artifact
```

可选择：

- CodeAgent；
- WebAgent；
- 动态注册且当前可用的 Agent。

特点：

- 不生成多 Agent Plan；
- 不需要多 Agent Plan 确认；
- 创建 Run 和一个 AgentInvocation；
- 支持流式 Message；
- 支持 Tool 和 Artifact；
- 可通过 Agent Selector 或 `@Agent` 临时覆盖默认 Agent；
- 调用前仍需进行授权、启用状态和健康检查。

Direct Mode 是直接调用，不属于多 Agent 编排。

## 4.2 Manual Multi-Agent Mode

用户显式选择两个或更多可用 Agent。

流程：

```text
用户选择 Agent 集合
→ 输入请求
→ Planner 仅从该集合分配任务
→ Plan 校验
→ 用户确认
→ 执行
→ 分别回复或统一汇总
```

Planner 不得偷偷增加用户未选择的 Agent。

用户修改 Agent 集合后，需要生成新的 PlanVersion，而不是把修改后的集合视为对旧计划的确认。

## 4.3 Auto Mode

用户选择自动模式。

流程：

```text
用户请求
→ Registry 生成 Eligible Agent Catalog
→ Planner 选择 Agent 和 Skill
→ 生成 PlanVersion
→ 用户确认
→ 执行
```

候选集合不是固定列表，而是：

```text
已注册
+ 已启用
+ 健康状态可接受
+ 当前用户和 Conversation 有权限
+ AgentVersion 有效
+ 至少有一个适用 Skill/Capability
```

CodeAgent 和 WebAgent 是默认候选，但 Registry 中其他符合条件的 Agent 也可以进入 Auto Mode。

## 4.4 响应模式

### Separate

聊天时间线分别显示每个 Agent 的结果。

### Synthesize

原始 Agent 输出保留在 Run 详情和 Artifact 中，Orchestrator 内部 Synthesizer 生成最终统一回复。

Synthesizer 不作为 ConversationMember。

---

# 5. 总体架构

```text
Frontend
├── Conversation Sidebar
├── Chat Timeline
├── Agent Selector
├── Agent Registration UI
├── Plan Confirmation Card
├── Agent Invocation Timeline
├── Attachment Panel
├── Artifact Workspace
└── Sandpack Web Preview
        │
        │ Public API + SSE / AG-UI
        ▼
Gateway
├── Authentication / Authorization
├── Conversation API
├── Message API
├── Run API
├── Plan API
├── Agent / Registration API
├── Attachment API
├── Artifact API
├── Search API
└── Event Replay / Stream
        │
        │ Private Internal API
        ▼
Orchestrator
├── Mode Resolver
├── LLM Planner
├── Parser / Normalizer / Validator / Repair
├── Plan Version Store
├── Confirmation Gate
├── Dependency Executor
├── Context Manager
├── Agent Registry
├── A2A Dispatcher
├── Result Synthesizer
├── Run / Event Store
└── Artifact Service
        │
        │ A2A
        ▼
Registered Agents
├── Built-in CodeAgent
├── Built-in WebAgent
├── Configured Agent
└── Remote Registered Agent
        │
        ├── LLM Provider
        ├── Local Tool
        └── MCP Tool
```

## 5.1 服务边界

### Frontend

负责：

- IM 页面；
- Agent 选择；
- 注册状态展示；
- Plan 确认；
- Artifact 编辑与预览；
- AG-UI Event Reducer。

Frontend 只调用 Gateway。

### Gateway

负责：

- 公共 API；
- 用户认证和对象级授权；
- Message 幂等；
- Conversation 和用户可见资源；
- Plan 确认入口；
- SSE 重放；
- 内部事件到 AG-UI 的映射。

Gateway 不调用 Agent 或 LLM。

### Orchestrator

负责：

- 模式解析；
- Registry Catalog；
- LLM 计划；
- Plan 校验和确认门；
- Context；
- Agent 调度；
- Replan；
- 结果汇总；
- Registry 生命周期。

### Registered Agent

负责：

- 声明 AgentCard 和 Skill；
- 处理 A2A Task/Message；
- 使用自身获批的模型和 Tool；
- 返回 Message、结构化数据或 Artifact。

Agent 不获取不相关 Conversation 历史，也不因注册自动获得平台 Tool 权限。

---

# 6. 核心领域模型

## 6.1 User

```text
User
├── id
├── display_name
├── avatar
├── preferences
├── created_at
└── updated_at
```

即使 2.0 初期只支持单用户，也保留 `user_id`。

## 6.2 Conversation

```text
Conversation
├── id
├── user_id
├── title
├── mode
├── response_mode
├── status
├── context_policy
├── version
├── last_message_at
├── pinned_at
├── archived_at
├── deleted_at
├── created_at
└── updated_at
```

### mode

```text
direct
manual_multi
auto
```

### response_mode

```text
separate
synthesize
```

## 6.3 ConversationMember

```text
ConversationMember
├── conversation_id
├── member_type
├── member_id
├── role
├── joined_at
└── left_at
```

`member_type`：

```text
user
agent
```

Auto Mode 不把全部 Registry Agent 永久加入 Conversation。

## 6.4 Message

```text
Message
├── id
├── client_message_id
├── conversation_id
├── run_id
├── sender_type
├── sender_id
├── message_type
├── content_text
├── content_json
├── reply_to_message_id
├── status
├── sequence
├── created_at
├── updated_at
└── deleted_at
```

Message 只表示用户可见时间线。

Tool Call、流式 Delta 和内部状态属于 Event。

## 6.5 Run

```text
Run
├── id
├── conversation_id
├── trigger_message_id
├── mode
├── status
├── plan_id
├── confirmed_plan_version
├── context_snapshot_id
├── started_at
├── finished_at
├── error_code
├── metadata
├── created_at
└── updated_at
```

状态：

```text
pending
planning
awaiting_confirmation
executing
synthesizing
completed
partial_failure
failed
canceled
```

## 6.6 Plan 与 PlanVersion

```text
Plan
├── id
├── run_id
├── current_version
├── status
├── created_at
└── updated_at
```

```text
PlanVersion
├── plan_id
├── version
├── goal
├── mode
├── selected_agent_ids
├── steps
├── constraints
├── response_mode
├── planner_metadata
├── validation_result
├── content_hash
├── created_at
├── confirmed_at
└── confirmed_by
```

确认后的 PlanVersion 不可修改。

## 6.7 PlanStep

```text
PlanStep
├── step_id
├── agent_id
├── agent_version
├── skill_id
├── title
├── instruction
├── depends_on
├── input_refs
├── expected_output
└── risk_metadata
```

Direct Mode 可以不指定 `skill_id`，由 Agent 内部选择默认 Skill。

Auto Mode 应在可能时显式指定 `skill_id`。

## 6.8 AgentInvocation

```text
AgentInvocation
├── id
├── run_id
├── plan_step_id
├── agent_id
├── agent_version
├── skill_id
├── status
├── input_context_snapshot_id
├── output_message_id
├── remote_task_id
├── remote_context_id
├── started_at
├── finished_at
├── token_usage
├── provider_request_id
├── error_code
└── error
```

A2A `context_id` 不等同于 AgentHub `conversation_id`。

## 6.9 Event

```text
Event
├── id
├── conversation_id
├── run_id
├── invocation_id
├── event_type
├── payload
├── sequence
└── created_at
```

单个 Run 内 `sequence` 单调递增，用于断线重放。

## 6.10 Attachment

```text
Attachment
├── id
├── conversation_id
├── message_id
├── filename
├── mime_type
├── size_bytes
├── storage_uri
├── content_hash
├── parse_status
├── extracted_text_uri
├── created_at
└── deleted_at
```

## 6.11 Artifact 与 ArtifactVersion

```text
Artifact
├── id
├── conversation_id
├── run_id
├── source_invocation_id
├── type
├── name
├── status
├── current_version
├── created_at
├── updated_at
└── deleted_at
```

```text
ArtifactVersion
├── artifact_id
├── version
├── base_version
├── content
├── content_uri
├── content_hash
├── media_type
├── change_summary
├── created_by_type
├── created_by_id
├── source_message_id
├── source_invocation_id
└── created_at
```

ArtifactVersion 不可变。

恢复旧版本会创建新版本，不直接覆盖历史版本。

## 6.12 ArtifactPatch

```text
ArtifactPatch
├── artifact_id
├── base_version
├── operations
└── change_summary
```

`base_version` 冲突必须明确返回，不能静默覆盖用户编辑。

## 6.13 ConversationSummary

```text
ConversationSummary
├── id
├── conversation_id
├── version
├── covered_from_sequence
├── covered_to_sequence
├── summary
├── open_tasks
├── key_decisions
├── active_artifact_refs
├── model_metadata
└── created_at
```

摘要不删除原始 Message。

## 6.14 ContextSnapshot

```text
ContextSnapshot
├── id
├── conversation_id
├── run_id
├── invocation_id
├── agent_id
├── agent_version
├── skill_id
├── policy_version
├── system_context_hash
├── message_ids
├── summary_ids
├── artifact_refs
├── retrieved_chunk_ids
├── estimated_tokens
└── created_at
```

ContextSnapshot 记录上下文来源，不保存模型私有推理过程。

## 6.15 RegisteredAgent

```text
RegisteredAgent
├── id
├── name
├── description
├── owner_id
├── registration_source
├── status
├── visibility
├── trust_level
├── current_version
├── created_at
├── updated_at
└── removed_at
```

### registration_source

```text
builtin
config
remote
managed_future
```

### status

```text
pending
validating
active
unhealthy
disabled
rejected
removed
```

## 6.16 AgentVersion

```text
AgentVersion
├── agent_id
├── version
├── card_json
├── card_hash
├── endpoint
├── protocol
├── skills
├── input_modes
├── output_modes
├── streaming
├── auth_type
├── created_at
├── activated_at
└── deprecated_at
```

## 6.17 AgentHealthCheck

```text
AgentHealthCheck
├── id
├── agent_id
├── agent_version
├── status
├── latency_ms
├── error_code
├── error_message
└── checked_at
```

健康状态与 Agent 生命周期状态分离。

## 6.18 AgentCredential

```text
AgentCredential
├── id
├── agent_id
├── credential_type
├── encrypted_payload
├── key_version
├── created_at
├── updated_at
└── rotated_at
```

凭据只写、加密存储，不进入 Planner Catalog、Event、ContextSnapshot 或公共读取接口。

## 6.19 RunAgentSnapshot

```text
RunAgentSnapshot
├── run_id
├── agent_id
├── agent_version
├── card_hash
├── endpoint_snapshot
├── skills_snapshot
├── capabilities_snapshot
└── created_at
```

Agent 刷新、禁用或注销后，历史 Run 仍可解释和展示。

---

# 7. Agent Registry

## 7.1 职责

Agent Registry 负责：

```text
谁可以被调用
```

A2A 负责：

```text
如何调用
```

Planner 负责：

```text
为什么调用以及调用顺序
```

Context Manager 负责：

```text
向该 Agent 提供什么输入
```

## 7.2 注册流程

```text
提交 AgentCard URL 或 Agent Base URL
→ 用户和请求校验
→ 网络目标安全校验
→ 获取 AgentCard
→ 协议和 Schema 校验
→ 规范化本地 Registry 元数据
→ Agent ID / Version 冲突校验
→ 初始健康检查
→ 保存 RegisteredAgent 和 AgentVersion
→ 激活
→ 进入 Agent Selector 和 Planner Catalog
```

远程动态注册必须失败关闭。

AgentCard 无效或不可达时，不得使用虚构的本地默认 Card 将其激活。

内置或配置型 Agent 可以在明确的受信任配置下使用本地定义。

## 7.3 可用性判定

```text
registered
+ active / enabled
+ health acceptable
+ user authorized
+ Conversation policy allowed
+ valid AgentVersion
+ applicable Skill
```

Agent Selector、Manual Mode 和 Auto Planner 必须使用一致的判定结果。

## 7.4 AgentCard 与 Skill

AgentHub 使用官方 A2A AgentCard 语义。

本地 Registry 可以：

- 隐藏 Skill；
- 禁用 Skill；
- 标记风险；
- 限制 Auto Mode；
- 设置可见性；
- 增加 AgentHub 扩展元数据。

Registry 不能替远程 Agent 发明不存在的 Skill。

AgentHub 扩展元数据单独存储，例如：

```text
accepted_artifact_types
context_tags
max_context_tokens
supports_conversation_history
sensitive_data_policy
```

不把这些字段伪装成官方 AgentCard 标准字段。

## 7.5 刷新和版本

AgentCard 刷新：

```text
获取 Card
→ 校验
→ 计算 Hash
→ 未变化：更新时间
→ 已变化：创建 AgentVersion
→ 健康检查
→ 按策略激活
```

Skill 语义、输入输出模式、Artifact 兼容性或安全要求变化时，应形成新 AgentVersion。

## 7.6 注销和历史

Agent 被禁用或注销后：

- 不进入新 Run；
- 不进入 Planner Catalog；
- 历史 Message 和 Run 保留；
- 历史 AgentVersion 和 RunAgentSnapshot 保留；
- Conversation 中的历史 Agent 身份仍可显示。

## 7.7 注册安全

远程注册必须处理：

- HTTPS 默认要求；
- 开发环境 HTTP 例外；
- Loopback、Link-local、Multicast、Unspecified 和云元数据地址；
- DNS 解析结果检查；
- Redirect 目标重新检查；
- Redirect 次数限制；
- DNS Rebinding；
- 连接、读取和总超时；
- AgentCard 响应大小和嵌套深度限制；
- Credential 加密和脱敏；
- 注册、刷新、启用、禁用和注销审计。

注册 Agent 不代表授权其调用平台 Tool。

---

# 8. 对话存储与持久化

## 8.1 存储原则

1. AgentHub 数据库是对话事实源。
2. Provider Conversation ID 只是调用优化字段。
3. 原始 Message 不被 Summary 覆盖。
4. Message、Event、Plan、Invocation 和 Artifact 分开存储。
5. 确认后的 PlanVersion 不可变。
6. ArtifactVersion 不可变。
7. AgentVersion 和 RunAgentSnapshot 保留历史。
8. 删除默认软删除。
9. 列表接口必须分页。
10. 所有写入有明确事务边界。

## 8.2 默认存储 Profile

```text
SQLite
WAL
sqlc
goose
FTS5
```

当前目标是单机与 Docker Compose。

PostgreSQL 可作为未来 Profile，通过稳定 Repository 接口替换。

MySQL 不是 2.0 默认目标。

## 8.3 主要数据表

```text
users
conversations
conversation_members
messages
runs
plans
plan_versions
agent_invocations
events
attachments
artifacts
artifact_versions
conversation_summaries
context_snapshots
context_chunks
registered_agents
agent_versions
agent_health_checks
agent_credentials
run_agent_snapshots
```

显式用户记忆可后续增加，但不是默认自动提取能力。

## 8.4 写入所有权

逻辑所有权：

```text
Gateway:
  User
  Conversation
  ConversationMember
  Message
  Attachment
  公共对象授权

Orchestrator:
  Run
  Plan
  PlanVersion
  AgentInvocation
  Event
  ContextSnapshot

Registry:
  RegisteredAgent
  AgentVersion
  AgentHealthCheck
  AgentCredential
  RunAgentSnapshot

Artifact Service:
  Artifact
  ArtifactVersion
```

单机 SQLite Profile 可以共享数据库文件和统一 Repository 包，但不得由多个模块散落执行无边界 SQL。

## 8.5 并发规则

默认每个 Conversation 只允许一个活动 Run。

新消息到达时必须显式选择：

```text
queue
cancel_current
human_input
```

使用：

- `client_message_id` 做消息幂等；
- `Conversation.version` 做乐观锁；
- `base_version` 做 Artifact 冲突控制；
- `(run_id, sequence)` 做 Event 顺序和重放；
- WAL、短事务、busy timeout 和有限重试处理 SQLite 并发。

模型、网络和 Tool 调用不能放在数据库事务内。

---

# 9. 多对话管理

## 9.1 Conversation Sidebar

至少支持：

- 新建；
- 最近；
- 置顶；
- 重命名；
- 归档；
- 恢复；
- 软删除；
- 搜索；
- 按 Agent 筛选；
- 按模式筛选；
- 按更新时间排序；
- 分页加载。

## 9.2 自动标题

首条有效用户 Message 后，可由规则或轻量模型生成标题。

Auto Title：

- 不是 Agent；
- 失败不影响 Run；
- 用户修改后不再自动覆盖；
- 内容长度受限并经过清理。

## 9.3 Conversation 切换

切换时恢复：

- Message 历史；
- 当前模式；
- Direct 默认 Agent 或 Manual Agent 集合；
- 当前或等待确认的 Run；
- Plan 卡片和准确版本；
- Artifact 列表与版本；
- Preview 当前版本；
- 草稿和 UI 状态。

切换不能复用另一个 Conversation 的 Summary、检索结果或 Artifact。

## 9.4 搜索

2.0 使用 SQLite FTS5 搜索：

- Conversation 标题；
- 用户 Message；
- Agent 最终 Message；
- ConversationSummary；
- Artifact 名称和可检索文本。

搜索结果必须先应用用户和 Conversation 授权。

## 9.5 归档、删除和彻底删除

### Archive

保留全部数据，可恢复。

### Soft Delete

默认查询隐藏，在保留期内可恢复。

### Purge

显式清理：

- Attachment 文件；
- Artifact 内容；
- FTS 索引；
- 凭据；
- 受保留策略控制的关联记录。

---

# 10. Context Manager

## 10.1 上下文层次

### 当前轮

- 当前 Message；
- 当前 Attachment；
- 当前选择的 Agent；
- 当前 PlanVersion；
- 用户对 Plan 的修改；
- 当前 ArtifactVersion。

### 短期对话上下文

- 近期 Message；
- 当前 Run；
- 未解决问题；
- 最近 Agent 结果；
- 当前 Artifact。

### 长期 ConversationSummary

- 对话目标；
- 用户约束；
- 已确认事实；
- 关键决定；
- 完成事项；
- 待办；
- 来源；
- Artifact 引用。

### 显式跨对话记忆

仅包括用户明确保存或批准的内容。

默认不自动提取敏感长期个人事实。

## 10.2 Context Builder

```text
System Policy
→ Use-case Policy
→ Registered Agent Profile
→ Agent Skill
→ Conversation Settings
→ Confirmed PlanStep
→ Current User Message
→ Recent Messages
→ ConversationSummary
→ Pinned Context
→ Relevant History
→ Attachment Chunks
→ Active ArtifactVersion
→ Upstream Agent Structured Output
→ Token Budget
→ ContextSnapshot
```

## 10.3 Agent 专属投影

### CodeAgent

优先：

- 当前代码和 WebProjectArtifact；
- 最新 ArtifactVersion；
- 修改目标；
- 相关代码讨论；
- WebAgent 的结构化事实和来源；
- 当前 PlanStep。

### WebAgent

优先：

- 检索问题；
- 时间和来源约束；
- 需要验证的事实；
- 相关近期 Message；
- 当前 PlanStep。

### 动态注册 Agent

基于：

- AgentCard Skill；
- 输入输出模式；
- Registry 扩展元数据；
- Agent 信任等级；
- 当前 PlanStep；
- 预期 Artifact 类型；

构建通用投影。

## 10.4 Token Budget

优先级：

```text
P0 系统和安全约束
P0 当前用户输入
P0 已确认 PlanStep
P1 当前 Artifact 与直接上游输出
P1 Pinned Context
P2 近期 Message
P2 ConversationSummary
P3 检索到的旧历史
P3 已被替代的 Artifact 摘要
```

必须保留模型输出、Tool 输出和协议开销空间。

## 10.5 长对话压缩

```text
选择稳定的早期范围
→ 与旧 Summary 一起增量摘要
→ 校验关键字段
→ 保存新 Summary 和覆盖范围
→ 保留原始 Message
```

Summary 不得把未经确认的 Agent 输出变成用户已确认事实。

## 10.6 检索

默认：

```text
SQLite FTS5
+ Source / Type Filter
+ 时间衰减
+ Conversation 内范围
```

Cross-conversation Retrieval 默认关闭。

只有用户明确请求或启用命名策略时才使用，并展示来源 Conversation。

## 10.7 Provider 状态

允许保存：

```text
provider_conversation_id
previous_response_id
provider_cache_key
```

Provider 状态丢失或切换后，AgentHub 仍可从本地事实重新构建上下文。

---

# 11. LLM 计划与确认

## 11.1 触发规则

| 模式 | 是否需要多 Agent Plan |
|---|---|
| Direct | 否 |
| Manual Multi-Agent | 是 |
| Auto | 是 |
| Material Replan | 是，新版本 |

## 11.2 Planner 输入

```text
user_id
conversation_id
run_id
mode
current_user_message
selected_agent_ids
eligible_agent_catalog
conversation_summary
relevant_history
attachment_summaries
active_artifact_refs
context_budget
response_mode
policy_constraints
```

Planner 不接收 AgentCredential。

## 11.3 Plan 示例

```json
{
  "plan_id": "plan_001",
  "version": 1,
  "run_id": "run_001",
  "goal": "检索最新官方 API，并更新当前网页项目",
  "mode": "auto",
  "selected_agent_ids": [
    "web-agent",
    "code-agent"
  ],
  "response_mode": "synthesize",
  "status": "awaiting_confirmation",
  "steps": [
    {
      "step_id": "step_1",
      "agent_id": "web-agent",
      "agent_version": "2.0.0",
      "skill_id": "docs_research",
      "title": "检索官方资料",
      "instruction": "检索目标 API 的最新官方行为并提供来源",
      "depends_on": [],
      "input_refs": [],
      "expected_output": {
        "artifact_type": "research"
      }
    },
    {
      "step_id": "step_2",
      "agent_id": "code-agent",
      "agent_version": "2.0.0",
      "skill_id": "frontend_project",
      "title": "更新网页项目",
      "instruction": "根据步骤 1 的结果更新当前 WebProjectArtifact",
      "depends_on": [
        "step_1"
      ],
      "input_refs": [
        "artifact:step_1"
      ],
      "expected_output": {
        "artifact_type": "web_project"
      }
    }
  ]
}
```

## 11.4 Validator

确定性检查：

- Agent 是否注册；
- AgentVersion 是否存在；
- Agent 是否启用；
- 健康结果是否可接受；
- 用户和 Conversation 是否有权限；
- Manual Mode 是否只使用用户选择的 Agent；
- Skill 是否由该 AgentVersion 声明并被本地策略允许；
- Step ID 是否唯一；
- 依赖是否存在；
- 是否有环；
- 输入引用是否有效；
- Artifact 类型是否兼容；
- Step 数量和深度是否超限；
- 是否隐藏高风险操作。

## 11.5 Normalizer

可以：

- 规范 ID；
- 转换合同允许的别名；
- 填写安全默认值；
- 排序稳定字段。

不能：

- 猜测未知 Agent；
- 发明 Skill；
- 静默替换不可用 Agent；
- 删除确认要求；
- 根据名称猜测依赖。

## 11.6 Repair

最多执行一次。

Repair 仍然经过 Parser、Normalizer 和 Validator。

Repair 失败后返回安全错误，或进入明确配置且仍需确认的兼容 Fallback。

## 11.7 Agent 可用性变化

每次 AgentInvocation 前重新检查：

- Enabled；
- Authorization；
- Health Freshness；
- Endpoint Availability。

确认后的 Agent 不可用时：

- 暂停或失败；
- 发出结构化错误；
- 可提出 Replan；
- 不得静默换成另一个 Agent；
- 新 PlanVersion 必须重新确认。

---

# 12. Agent Runtime 与内置 Agent

## 12.1 Agent Runtime

通用 Runtime 负责：

```text
A2A Server Adapter
Handler Lifecycle
LLM Adapter
MCP / Local Tool Adapter
Streaming
Cancellation
Structured Message
Artifact Candidate
Mock
Telemetry
```

Runtime 不负责：

- Conversation；
- Planner；
- Registry 生命周期；
- AG-UI；
- 公共 API；
- 全局 Artifact 持久化。

不建立第二套不兼容的 A2A 或 MCP Wire Model。

## 12.2 CodeAgent

定位：

> 理解、生成、修改、分析代码，并维护当前 Conversation 中的代码 Artifact；不负责主动获取互联网最新事实。

建议 AgentCard Skill：

```text
code_generate
code_analyze
code_transform
frontend_project
```

其中：

- `code_generate`：代码生成；
- `code_analyze`：解释、调试、审查、测试分析；
- `code_transform`：修改、重构、迁移、Patch；
- `frontend_project`：创建或更新 WebProjectArtifact。

输入：

- 用户 Message；
- 代码和 Attachment；
- ArtifactRef；
- PlanStep；
- WebAgent 结构化研究结果；
- ContextSnapshot。

输出：

- Message；
- CodeArtifact；
- Unified Diff；
- ArtifactPatch；
- WebProjectArtifact；
- 错误和验证信息。

## 12.3 WebAgent

定位：

> 搜索互联网、读取页面、研究官方文档、验证事实并输出带来源的结构化结果；不负责最终代码修改。

建议 AgentCard Skill：

```text
web_search
web_read
web_research
docs_research
```

结构化输出：

```json
{
  "summary": "结论摘要",
  "facts": [
    {
      "claim": "事实内容",
      "source_ids": ["source_1"]
    }
  ],
  "sources": [
    {
      "id": "source_1",
      "title": "页面标题",
      "url": "https://...",
      "publisher": "发布方",
      "published_at": null,
      "retrieved_at": "检索时间",
      "snippet": "支持该事实的摘要"
    }
  ],
  "uncertainties": []
}
```

下游 Agent 默认接收结构化事实和必要引用，不接收全部网页原文。

## 12.4 动态 Agent

动态 Agent：

- 使用相同 A2A Dispatcher；
- 使用 Registry AgentVersion；
- 使用通用 Context Projection；
- 可以产生已注册类型的 Artifact；
- 未知输出类型安全降级；
- 不自动获得平台 Tool；
- 不因注销而破坏历史 Run。

---

# 13. A2A、MCP 与 Tool 边界

## 13.1 A2A

优先复用官方 A2A Go SDK。

标准 AgentCard 发现路径：

```text
/.well-known/agent-card.json
```

旧路径仅作为迁移兼容：

```text
/.well-known/agent.json
```

A2A 用于：

- AgentCard；
- Message；
- Task；
- Task Status；
- Artifact；
- Streaming；
- Cancellation；
- Input Required。

Orchestrator 将 A2A 事件转换为内部 Event，Gateway 再转换为 AG-UI。

前端不直接接收原始 A2A Event。

## 13.2 Retry

可在用户可见输出开始前进行有限重试。

可见流式内容开始后，不默认重新执行整个调用，避免重复 Message 或 Artifact。

优先使用协议支持的恢复和重新订阅。

## 13.3 MCP 与 Local Tool

复用官方 MCP Go SDK。

Tool 必须：

- 有明确 Schema；
- 受 Agent 和 Skill Policy 限制；
- 校验参数；
- 支持超时和取消；
- 限制输出大小；
- 记录审计；
- 高风险 Tool 使用独立确认。

Plan 确认不自动等于高风险 Tool 确认。

---

# 14. Artifact 与网页预览

## 14.1 核心 Artifact 类型

```text
text
markdown
code
research
json
file
web_project
```

新增 Artifact 类型必须定义：

- 唯一 Type ID；
- Schema；
- 大小限制；
- 输入输出兼容性；
- Renderer Policy；
- 安全 Fallback；
- 测试。

## 14.2 未知输出

未知 Agent 输出：

- 可以安全显示为 JSON 或文本；
- 可以作为授权下载文件；
- 不自动执行；
- 不自动选择自定义 Renderer；
- 不注入主应用 DOM。

## 14.3 WebProjectArtifact

```json
{
  "template": "react",
  "entry_file": "/src/App.jsx",
  "files": {
    "/package.json": "{...}",
    "/src/main.jsx": "...",
    "/src/App.jsx": "...",
    "/src/styles.css": "..."
  },
  "dependencies": {
    "react": "^19.0.0",
    "react-dom": "^19.0.0"
  },
  "preview": {
    "renderer": "sandpack",
    "enabled": true,
    "network_policy": "allowlisted"
  }
}
```

Artifact ID 和 Version 由 Artifact Envelope 提供。

## 14.4 预览范围

稳定支持：

- Static HTML；
- Vanilla JavaScript；
- React；
- 多文件项目；
- 公共前端 npm 依赖；
- 编辑器；
- Preview；
- Console；
- Runtime Error；
- 用户编辑；
- Agent Patch；
- 历史版本查看。

不执行：

- Go/Python 后端；
- 数据库；
- Docker；
- 任意 Shell；
- 私有包凭据；
- 服务端 Secret；
- 生产部署。

## 14.5 版本冲突

用户保存和 Agent Patch 均携带 `base_version`。

冲突时：

- 不覆盖；
- 返回当前版本；
- 保留本地编辑；
- 提供重新应用、比较或重新生成入口。

## 14.6 Sandbox

Preview 必须：

- 与主应用 Token 和 Storage 隔离；
- 禁止 Top Navigation；
- 限制 Popup 和 Download；
- 有明确 CSP / Sandbox；
- 有网络策略；
- 能停止和重置；
- 不将 Secret 注入浏览器项目。

---

# 15. 流式事件与 API

## 15.1 AG-UI

Gateway 输出标准 AG-UI Event，例如：

```text
RUN_STARTED
RUN_FINISHED
RUN_ERROR
STEP_STARTED
STEP_FINISHED
TEXT_MESSAGE_START
TEXT_MESSAGE_CONTENT
TEXT_MESSAGE_END
TOOL_CALL_START
TOOL_CALL_ARGS
TOOL_CALL_END
TOOL_CALL_RESULT
STATE_SNAPSHOT
STATE_DELTA
ACTIVITY_SNAPSHOT
ACTIVITY_DELTA
CUSTOM
```

AgentHub 自定义事件使用命名空间：

```text
agenthub.plan.*
agenthub.artifact.*
agenthub.preview.*
agenthub.context.*
```

计划确认和 Preview 不使用伪 Tool Call。

## 15.2 Event Replay

SSE `id` 使用持久化 Run Sequence。

重连：

```text
Last-Event-ID
→ 重放缺失 Event
→ 接入实时流
```

Frontend Reducer 按 Event ID 幂等处理。

## 15.3 Public API

主要资源：

```text
/api/conversations
/api/conversations/{id}/messages
/api/runs/{id}
/api/runs/{id}/events
/api/plans/{id}
/api/agents
/api/agent-registrations
/api/attachments
/api/artifacts
```

Agent 生命周期操作包括：

```text
refresh
check
enable
disable
remove
```

未来 AgentHub 托管 Agent 使用：

```text
/api/managed-agents
```

不与 `/api/agent-registrations` 混用。

## 15.4 Internal API

Gateway 到 Orchestrator 的核心操作：

```text
ExecuteDirectRun
CreatePlan
ExecuteConfirmedPlan
CreateReplan
CancelRun
GetRun
ListEligibleAgents
RegisterAgent
RefreshAgent
CheckAgent
EnableAgent
DisableAgent
RemoveAgent
```

当前可继续使用 Private HTTP/Streaming。

gRPC 不是隐含的 2.0 强制目标。

---

# 16. 安全

## 16.1 信任模型

以下内容均视为不可信：

```text
Frontend 输入
用户输入
LLM 输出
远程 Agent
AgentCard
网页内容
Attachment
Artifact
Tool Result
Preview Code
```

## 16.2 关键规则

- Frontend 只访问 Gateway；
- Orchestrator 和 Agent Endpoint 不直接暴露给浏览器；
- 所有子资源执行对象级授权；
- Agent Credential 只写、加密、脱敏；
- Browser Bearer Token 不转发给 Agent；
- Planner 只使用 Eligible Catalog；
- 多 Agent Plan 必须确认；
- 高风险 Tool 独立确认；
- Cross-conversation Context 默认关闭；
- Attachment 和 Artifact 有类型、大小和路径限制；
- Preview 运行在 Sandbox；
- Secret 不进入 Git、AgentCard、Plan、Event、ContextSnapshot 或公共错误；
- 外部网页和 Agent 输出作为数据，不作为高优先级系统指令。

---

# 17. 可观测性、测试与交付

## 17.1 Correlation

关键标识：

```text
request_id
trace_id
conversation_id
message_id
run_id
plan_id
plan_version
step_id
invocation_id
agent_id
agent_version
skill_id
context_snapshot_id
artifact_id
artifact_version
tool_call_id
error_code
```

高基数 ID 用于 Trace 和 Log，不作为不受控 Metric Label。

## 17.2 指标

至少覆盖：

- Run 成功率和延迟；
- Plan 生成和校验；
- 确认等待时间；
- Agent 选择；
- Context Token 和压缩；
- Agent/A2A 调用；
- Tool 调用；
- Model Token 和费用；
- Event Replay；
- Artifact Version 和 Conflict；
- Preview 启动和错误；
- Registry Health。

## 17.3 测试原则

测试不得依赖：

- 生产 Secret；
- 不受控公共 Agent；
- 不受控公共网站；
- 非确定性真实模型作为唯一路径。

需要：

- Mock Planner；
- Mock LLM；
- Mock Agent；
- Local Mock A2A Server；
- Mock MCP/Tool；
- 固定 Web 和 Artifact Fixture；
- 临时 SQLite。

## 17.4 默认 Compose

```text
frontend
gateway
orchestrator
code-agent
web-agent
```

默认 Profile：

- SQLite 持久化；
- Mock 可运行；
- 无真实 API Key；
- 动态远程 Agent 在运行时注册；
- 可选 Observability Profile。

---

# 18. 复用边界

| 模块 | 复用 | AgentHub 自研边界 |
|---|---|---|
| Agent 通信 | A2A 官方规范与 Go SDK | Registry、鉴权、Context、业务映射 |
| Tool 协议 | MCP 官方 Go SDK | Tool 权限、审计和 Adapter |
| 前端事件 | AG-UI 标准事件 | Internal Event 转换、重放和 Product Custom Event |
| 计划确认 UX | Magentic-UI 等交互参考 | Plan Schema、Validator、版本确认 |
| 网页预览 | Sandpack | WebProjectArtifact 映射和版本同步 |
| 数据访问 | sqlc | Schema、Query 和 Repository |
| 数据迁移 | goose | Migration 文件和所有权 |
| 搜索 | SQLite FTS5 | 索引、权限和排序策略 |
| 可观测性 | OpenTelemetry | AgentHub 业务属性和指标 |
| Provider 状态 | Provider 官方 API | 本地 Conversation 事实源和 Context Builder |

不自行实现：

- A2A/MCP 底层协议；
- 浏览器 JavaScript 编译器；
- npm 包管理器；
- HMR；
- 通用代码编辑器；
- 数据库迁移框架；
- SQL 到 Go 代码生成器；
- 全文检索引擎；
- Trace 协议。

需要自行实现：

- Conversation/Message/Run/Plan；
- Agent Registry；
- Planner Prompt 和 Validator；
- 用户确认状态机；
- Context Projection；
- AgentHub Agent Skill；
- Artifact Version；
- IM 与执行过程映射；
- 评测和产品指标。

---

# 19. 实施顺序

前一步未达到退出门槛时，不进入下一步。

## Step 0：收口事实源与 Contract

动作：

1. 以本 PDR 和 Active Contract 为事实源。
2. 将旧 Contract 和旧 Skill 移入 Legacy。
3. 收口 Project Architecture、Conversation、Planning、Registry、Context、API、A2A、Artifact 和 Security Contract。
4. 修正文档中的 MySQL、强制 gRPC、固定 Agent 数量和旧 ToolCall 语义。

退出门槛：

- Active Contract 无旧架构冲突；
- CodeAgent/WebAgent 表述为内置参考 Agent；
- 动态注册是正式能力；
- Contract、Schema 和 OpenAPI 可校验。

## Step 1：领域模型和 SQLite 持久化

动作：

1. SQLite/WAL。
2. goose Migration。
3. sqlc Query。
4. 核心表和 Repository。
5. FTS5。
6. 幂等、乐观锁和 Event Sequence。

退出门槛：

- 空库可迁移；
- 重启后数据存在；
- Message 幂等；
- PlanVersion 和 ArtifactVersion 不可变；
- AgentVersion 和 Snapshot 可读取；
- Event 可重放。

## Step 2：多会话 IM

动作：

1. Conversation CRUD。
2. Message 分页。
3. Sidebar。
4. 切换、置顶、归档、搜索和软删除。
5. 单 Conversation 单活动 Run。
6. 授权。

退出门槛：

- 两个 Conversation 独立工作；
- 切换不串状态；
- 搜索只返回授权内容；
- 历史 Message 和 Artifact 可恢复。

## Step 3：Direct Mode

动作：

1. Agent Selector。
2. Eligible Agent API。
3. Direct Run。
4. A2A Dispatcher。
5. Message Streaming。
6. Cancel 和 Replay。
7. CodeAgent/WebAgent Mock。
8. 动态 Agent Mock。

退出门槛：

- 内置 Agent 可直接聊天；
- 动态注册 Mock Agent 可直接聊天；
- 不需要真实 API Key；
- 取消和重连可用；
- 无跨 Conversation Context 泄漏。

## Step 4：Agent Registry

动作：

1. RegisteredAgent 和 AgentVersion。
2. AgentCard Fetch/Validation。
3. 注册、刷新、检查、启用、禁用和注销。
4. 健康检查。
5. 可见性和权限。
6. Planner Catalog。
7. RunAgentSnapshot。
8. SSRF 和 Credential。

退出门槛：

- 无效 Card 不激活；
- Unhealthy/Disabled Agent 不进入 Catalog；
- 注销后历史可读；
- 注册不提升 Tool 权限；
- Secret 不泄漏。

## Step 5：LLM Planner 与确认

动作：

1. Planner Input。
2. Structured Output。
3. Parser。
4. Normalizer。
5. Validator。
6. Repair Once。
7. PlanVersion。
8. Plan Card。
9. Confirm/Reject/Edit/Regenerate。

退出门槛：

- Direct Mode 不生成多 Agent Plan；
- Manual Mode 不加入未选 Agent；
- Auto Mode 使用 Registry Catalog；
- 未确认不执行；
- Stale Version 被拒绝；
- Replan 重新确认。

## Step 6：依赖执行与汇总

动作：

1. 串行、并行和混合依赖。
2. Step 状态。
3. 上游 ArtifactRef。
4. Partial Failure。
5. Synthesizer。
6. Agent 不可用时 Replan。

退出门槛：

- 依赖和波次正确；
- 失败不会静默替换 Agent；
- Separate/Synthesize 均可用；
- 原始 Agent 输出可追踪。

## Step 7：Context Manager

动作：

1. Agent 专属投影。
2. Recent Message。
3. ConversationSummary。
4. Pinned Context。
5. FTS5 Retrieval。
6. Token Budget。
7. ContextSnapshot。
8. Cross-conversation 显式策略。

退出门槛：

- 不发送完整历史；
- 不同 Agent 获得不同 Context；
- 长对话不溢出；
- 原始 Message 保留；
- 默认不跨 Conversation。

## Step 8：Artifact 与 Web Preview

动作：

1. Artifact/Version/Patch。
2. Type Registry。
3. WebProjectArtifact。
4. Sandpack。
5. 用户编辑。
6. Agent Patch。
7. Conflict。
8. History/Restore。
9. Sandbox 和 Network Policy。

退出门槛：

- HTML/React 可预览；
- 用户和 Agent 编辑不互相覆盖；
- 历史版本可查看；
- 未知内容不执行；
- Preview 无法读取主应用 Secret。

## Step 9：协议、运行时和恢复

动作：

1. 官方 A2A SDK Adapter。
2. 官方 MCP SDK Adapter。
3. Agent Runtime。
4. Standard AG-UI Mapping。
5. Cancellation。
6. Retry/Resume。
7. Error Mapping。
8. Compatibility Adapter。

退出门槛：

- 不维护重复底层协议；
- 内置和动态 Agent 使用统一 Dispatcher；
- 流式重试不重复输出；
- Cancel 可以贯穿调用链。

## Step 10：可观测性、测试和交付

动作：

1. OpenTelemetry。
2. Token/Cost/Latency。
3. Security Negative Tests。
4. Race Test。
5. Migration Test。
6. Mock Integration。
7. Compose Smoke。
8. Evaluation Dataset。
9. README、Status 和操作文档。

退出门槛：

- Mock Compose 无 Secret 启动；
- 核心流程有确定性测试；
- Run 可追踪到 Plan、Context、Agent 和 Artifact；
- 日志和错误无 Secret；
- 文档和实际命令一致。

---

# 20. 验收标准

## 20.1 产品验收

- 可以创建、切换、搜索、归档和删除多个 Conversation。
- 可以选择任意 Eligible Agent 进行 Direct Chat。
- 可以选择多个 Agent 并生成 Plan。
- Auto Mode 可以从动态 Registry Catalog 中选择 Agent。
- 用户能编辑、确认、拒绝或重新生成 Plan。
- 未确认 Plan 不产生 AgentInvocation。
- CodeAgent、WebAgent 和一个动态 Mock Agent 均可被调用。
- Agent 失效时不静默替换。
- Agent 注销后历史 Message 和 Run 仍可展示。
- Artifact 可以创建、修改、查看历史和恢复。
- React/HTML WebProjectArtifact 可以预览。
- Conversation 切换不会串 Context 或 Artifact。
- 断线后可以重放 Event。
- 取消可以传递到 Orchestrator 和 Agent。

## 20.2 技术验收

- SQLite 空库 Migration 成功。
- `client_message_id` 幂等。
- 单 Conversation 单活动 Run。
- PlanVersion、AgentVersion、ArtifactVersion 有历史。
- ContextSnapshot 可复现输入来源。
- Registry 过滤 Disabled、Unhealthy 和 Unauthorized Agent。
- Agent Credential 不出现在 API、Event、Plan、Log 或 ContextSnapshot。
- AG-UI 标准事件和 AgentHub Custom Event 可校验。
- A2A Task/Streaming/Cancel 有本地 Mock 测试。
- Artifact 和 Preview 有路径、大小、类型和 Sandbox 测试。
- 核心 CI 不需要生产 API Key 或公共 Agent。
- `git diff --check`、Schema/OpenAPI、Go Test 和 Frontend Test 可执行。

## 20.3 评测指标

至少记录：

```text
Plan JSON Valid Rate
Plan Validation Pass Rate
Agent Selection Accuracy
Manual Agent Set Violation Rate
Unconfirmed Execution Violation Rate
Dependency Accuracy
Replan Reconfirmation Rate
Cross-conversation Contamination Rate
Context Constraint Retention
Artifact Conflict Detection Rate
Preview Startup Success
Event Replay Correctness
Agent Invocation Success
Run Latency
Input / Output Token
Estimated Cost
```

其中以下指标目标必须为零违规：

```text
Manual Agent Set Violation
Unconfirmed Plan Execution
Cross-conversation Context Contamination
Credential Exposure
Unknown Artifact Automatic Execution
```

---

# 21. 主要风险

## 21.1 动态 Agent 安全风险

风险：

- SSRF；
- 恶意 AgentCard；
- Credential 泄漏；
- 不可信 Artifact；
- Prompt Injection；
- 不稳定流式协议。

应对：

- 失败关闭；
- 网络目标校验；
- Card 大小和 Schema 限制；
- Credential 隔离；
- Trust Level；
- 最小 Context；
- 未知输出安全降级；
- 审计和健康检查。

## 21.2 LLM 计划不稳定

应对：

- Structured Output；
- Deterministic Validator；
- Repair Once；
- PlanVersion；
- 用户确认；
- Mock 和评测数据集。

## 21.3 长对话质量下降

应对：

- Summary；
- FTS5；
- Pinned Context；
- Agent Projection；
- ContextSnapshot；
- 约束保留评测。

## 21.4 Artifact 编辑冲突

应对：

- 不可变版本；
- `base_version`；
- 明确 Conflict；
- 历史恢复；
- Agent Patch 而非无条件覆盖。

## 21.5 协议和现有代码迁移风险

应对：

- 官方 SDK Adapter；
- Legacy Compatibility Layer；
- Contract Test；
- 分批迁移；
- 不在同一批次同时重写全部服务。

---

# 22. 最终结论

AgentHub 2.0 的核心不是“Agent 数量”，而是建立完整、可控和可扩展的多 Agent IM 执行闭环：

```text
Conversation
→ Message
→ Mode
→ Eligible Agent Catalog
→ Direct Invocation 或 Confirmed PlanVersion
→ ContextSnapshot
→ AgentVersion / Skill
→ A2A Execution
→ Message / ArtifactVersion
→ AG-UI Event
→ Persistence / Replay / Audit
```

CodeAgent 和 WebAgent 提供稳定的内置参考能力。

Agent Registry 使平台能够接入其他符合要求的 Agent。

LLM Planner 提供动态编排，但所有多 Agent 计划都必须经过确定性校验和用户确认。

Context Manager 控制每个 Agent 实际接收的信息。

ArtifactVersion 和 Sandpack 提供可追踪、可编辑且安全隔离的代码产物体验。

该方案在保持 2.0 范围可控的同时，保留了 AgentHub 作为可扩展 Agent 平台的核心价值。
