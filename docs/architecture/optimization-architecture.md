# AgentHub v2 优化架构：Redis + MySQL + LLM 编排 + Harness + Memory

- 日期：2026-10-29；源码基线：`dev / 8a44f9e`。
- 状态：**设计提案，未实施，待审核后进入代码阶段**。
- 本版替代上一版仅优化 SQLite writer 的方案；旧问题仍是回归条件，不再作为架构目标。
- 配套：[技术选型及网上项目来源](optimization-technology-selection.md)；[分阶段实施计划](../plans/agenthub-redis-sql-harness-plan.md)。

## 1. 决策摘要

1. **SQL 具体选择 MySQL**，沿用项目目标；Redis 与 MySQL 协作，而不是用 Redis 替代所有持久化。
2. **彻底取消语义规则路由**：不保留关键词评分、RulePlanner、模型失败时固定选择 code-agent 等回退。
3. **LLM 决定任务、Agent 和工具**；权限校验、预算、依赖顺序、审批与终态由确定性的 Harness 执行。这些安全控制不是语义规则路由。
4. **增加实际可执行的工具**：工作区读取/搜索、受控补丁、隔离测试、网页搜索/抓取、专家委派；不把代码模板工具或前端预览当作完整工具体系。
5. **增加 Agent Runtime Harness**：管理上下文、模型调用、工具权限、运行预算、检查点、恢复和验收证据；另设 Eval Harness 做质量评测。
6. **增加明确的 Memory 控制**：MemoryController 统一记忆读取、候选写入、确认、版本更新、过期和遗忘；用户可查看、纠正、删除及关闭记忆，模型不能自行决定永久保存或扩大作用域。
7. 保留 Go、新模块边界和现有纯 ADK，借鉴 Eino / Deep Agents / OpenAI Agents SDK，而不是同时引入多套 Agent 循环。

这是用户明确提出的新范围，不再沿用“暂缓 Redis、只改 Gateway 两个文件”的旧边界。当前 contracts 仍有不同约定，必须在实施阶段 P0 审核修订，不能把本提案当成已经生效的契约。

## 2. 当前实现与缺口

| 项目 | 当前源码事实 | v2 必须改变的内容 |
|---|---|---|
| 主编排入口 | `services/orchestrator/httpapi/handler_run_stream.go:188–193` 调用 MainAgent；nil 时创建无模型实例 | 生产启动必须获得有效模型依赖，不在请求中创建无模型 planner |
| MainAgent 回退 | `services/orchestrator/planner/main_agent.go:72–109,245–260,448–486` 在模型失败/缺失时关键词评分 | 移除关键词回退和固定 Agent 回退，失败显式停止或澄清 |
| 启动配置 | `services/orchestrator/cmd/orchestrator/main.go:237–304` 默认 rule，缺模型配置可退回 rule | 删除 rule/llm_with_rule_fallback 运行模式；生产配置错误使就绪检查失败 |
| 旧 Planner | `services/orchestrator/planner/llm_planner.go:115–119,521–556` 可禁用回退，但不控制 MainAgent | 不能用 DisableFallback 冒充全链路无规则；清理旧注入与可达回退 |
| ADK 基础 | `pkg/adk/tool.go:8–20` 已有工具接口；`pkg/adk/runner.go:12,27–45,130–168` 已有最多 25 轮、工具和 Plugin | 复用接口/循环，补全执行前后检查点和预算等能力，不再从零复制 Runner |
| 实际工具 | `services/agents/code-agent/server.go:24–34` 已接工具；`tools.go:20–25,51–71` 是确定性片段生成 | 工具不是零，但缺本方案所需的工作区、外部检索与隔离执行工具 |
| 临时状态 | `services/orchestrator/registry/run_task_registry.go:15–27` 为内存状态；`httpapi/pending_plan.go:57–79` 定义审批状态 | 内存 cancelFunc 不能持久化；审批、任务关联和运行进度需要持久化表示 |
| Gateway 存储 | `services/gateway/cmd/gateway/main.go:65–74,110–114,287–293` 默认 MemoryStore、可选 SQLite 镜像 | MySQL 成为目标启动路径的主 Store，不再依赖 MemoryStore 决定历史会话是否存在 |
| SQL 基础 | `docs/contracts/mysql-schema.md:18–63` 有 Gateway/Session schema；新模块未据此证明完整 MySQL 链路 | 建正式驱动、迁移、Repository、集成测试；legacy server 的 MySQL 依赖不算新架构已接入 |

Redis/MCP/checkpoint/run lease 的本次检索未找到完整运行接线；这不是证明仓库每个角落都没有相关文本。现有 SQLite writer 的共享状态、重复 UPDATE、空消息终态问题仍须在新投影实现中做回归测试。

## 3. 目标模块与部署

```text
Browser / Frontend
  │ REST + AG-UI SSE；携带可恢复事件游标
  ▼
Gateway
  ├─ 鉴权、会话/消息、审批交互、历史分页、事件投影
  ├─ MySQL：Gateway 所有的业务表
  └─ 内部 gRPC：启动 / 订阅 / 审批 / 取消 / 查询状态
        ▼
Orchestrator
  ├─ MainAgent：LLM 产出结构化计划与下一步动作
  ├─ Coordinator：执行动作、依赖调度、专家委派、结果汇总
  ├─ Runtime Harness：权限、预算、上下文、检查点、终态
  ├─ MemoryController：长短期记忆、召回、确认写入、更新/遗忘
  ├─ Tool Broker：授权、调用账本、审批绑定、结果归档
  ├─ MySQL：执行状态 / 事件 / checkpoint / Memory / outbox
  ├─ Redis：能力缓存 / 可失效的上下文与召回缓存 / 短期事件
  └─ A2A → Child Agents → Runtime Harness → ADK / Tools
                                      └─ 隔离工作区 / Sandbox
```

- `pkg/adk` 仍是纯执行引擎和接口，不依赖 Redis、MySQL、MCP 客户端或具体工具。
- `pkg/runtime/harness` 为拟新增位置：可复用的控制器、策略和存储接口适配，不承载用户意图路由。
- `pkg/runtime/tools` 为拟新增位置：工具描述、原生工具、Broker 适配与 MCP 适配。
- Orchestrator 是唯一的编排决策/执行协调服务；Planner 只产出结构化结果，不直接读 Gateway 数据库或调用 Child Agent。
- Gateway 仍独占浏览器 API；不让浏览器连接 Redis、MySQL、MCP Server 或 Child Agent。
- 初始部署在原有服务基础上增加 Redis；Sandbox 属于受控执行设施，不借机增加独立 Planner 微服务或 Kubernetes 平台。
- 当前服务间 HTTP/SSE 只是迁移期兼容实现；目标仍是 gRPC，公开端仍为 AG-UI SSE。

## 4. Redis 与 MySQL 的分工

### 4.1 数据所有权

可先共用一个 MySQL 实例，但不同模块使用独立逻辑 schema/账号权限；不得跨模块直接改对方表。

| 所有者 | 拟管理实体 | 权威性 |
|---|---|---|
| Gateway | conversations、messages、chat_requests、run_views、event_projection_offsets、artifact_views | 会话/用户消息为权威；运行和 Agent 消息是可重建投影 |
| Orchestrator | executions、task_executions、plan_revisions、approvals、tool_calls、checkpoints、execution_events、outbox、agent_catalog | 执行、审批、预算、调用账本的权威状态 |
| Orchestrator MemoryController | memory_entries、memory_revisions、memory_scopes、memory_mutations | 长期记忆、策略/epoch、更新/遗忘状态的唯一权威；Gateway 只代理管理入口 |
| Runtime 存储适配 | sessions、session_events、session_context_snapshots | Agent 会话与短期摘要；不是长期记忆或工作流 checkpoint 的替代品；具体存储服务只访问自己拥有的数据 |
| Redis | 带版本的能力/Memory 召回缓存、上下文快照、实例心跳、事件短期副本 | 可重建辅助数据；不能决定审批、终态或 Memory 是否仍允许使用 |

工作区/主体标识来自 Gateway 验证后的身份上下文，不信任浏览器任意传入的 userId。当前固定 Token 模式不能宣称已经提供完整多租户安全；此处要求的是执行资源的授权边界，不扩展复杂用户系统。

### 4.2 事务与幂等

- Gateway 在本地事务中写入用户消息与 `chat_requests`；使用稳定 requestId，调用 Orchestrator 超时后按相同 requestId 重试/查询，不再次创建新运行。
- Orchestrator 以 requestId 唯一约束接受运行；不能声称 Gateway 与 Orchestrator 的两次事务天然原子。
- 每次 checkpoint/状态变化，在同一个 Orchestrator 数据库事务内更新执行版本、写入 `execution_events` 和 outbox。
- outbox relay 在事务外投递 Redis。投递后未标记成功就崩溃可能重复发送；消费者按 `(runId, seq)` 去重。
- Gateway 更新消息投影与消费位置放入同一事务；至少一次投递不导致文本重复追加。
- 禁止在模型推理、等待审批或外部工具执行期间保持数据库长事务。

### 4.3 Redis 用法

建议键空间包含环境和工作区，例如 `agenthub:{env}:{workspace}:run:{runId}:events`；不放密钥、真实 DSN、原始凭据或跨工作区可共享的工具结果。

- 能力缓存：携带 SQL 中的 catalog revision；不可用时从 SQL 回源。权限撤销仍要在调用前校验权威策略，不能相信旧缓存。
- 运行事件：Redis Streams 保存短期副本，设置明确的长度/过期边界；可靠补发来自 SQL event journal。
- 浏览器订阅对应的内部广播使用独立游标的 `XREAD` 模式，不把多个订阅者放进同一个 `XREADGROUP`，否则事件会被分摊而不是各自完整收到。
- `XREADGROUP/ACK` 如用于后台工作队列，需要单独的 PEL、重领、幂等与清理设计；首版不再新增 Redis 工作队列。
- Pub/Sub 只可做允许丢失的唤醒提示，不能用于唯一的审批结果、终态或恢复日志。
- Redis 故障：跳过缓存、从 SQL 有界补读，降低新运行准入；不能重新执行写工具“补事件”。Redis 降级需压测，不能承诺同等性能。

### 4.4 运行领取与并发控制

执行所有权以 SQL 条件更新和递增 fencing version 为准；Redis 心跳/TTL 仅辅助发现，不是正确性锁。持有者续租、checkpoint、预算扣减和终态提交必须验证版本。

过期 worker 不得继续提交状态；工具 Broker 在派发前验证当前所有权。该检查不能原子覆盖远端副作用，因此对“工具已执行但结果未入账”的情况使用 `unknown`，禁止自动重试非幂等写操作。不能把 Redis 锁或 fencing 声称为任意外部工具的 exactly-once 保证。

## 5. 不使用规则路由的编排

### 5.1 决策闭环

```text
用户目标 + 有界历史 + 允许的 Agent/Tool 描述 + 已完成证据
  → MainAgent 模型
  → 结构化动作：answer / clarify / tool_call / delegate / revise_plan / finish
  → Parser → 规范化 → Schema/权限/预算/依赖校验
  → Coordinator 执行已批准动作
  → 工具结果/专家产物作为 observation
  → 模型决定下一步，或结束
```

原生 structured output/function calling 可用时优先；不支持时允许解析严格 JSON 响应，但这只是协议适配，不能改为关键词判断。

- 单聊/群聊中用户选择的 Agent 集合是权限和参与边界；`@agent` 是显式指定，不是关键词路由。
- MainAgent 根据描述选择专家，优先使用 **agents-as-tools** 模式：主协调者保留结果汇总和预算控制权，子 Agent 获得最小上下文。
- 子 Agent 不自动获得全部聊天记录、父级所有工具或继续委派权限。
- 依赖图和并行度由 Coordinator 在已验证计划上执行，不用固定“代码任务一定先 code-agent”的业务规则补全模型判断。
- 首版不追求任意多层 Agent 树；委派深度和并发都受预算限制。

### 5.2 失败语义

| 情况 | 处理 |
|---|---|
| 模型依赖缺失/不支持必需能力 | 启动或就绪失败；不退回 rule，也不把 mock 接到生产 |
| 模型暂时错误 | 预算内有限重试；失败明确返回安全错误 |
| JSON/schema 错误 | 最多一次模型修复，仍错误则失败；不猜 Agent 名或补默认任务 |
| 意图不明确 | 模型发出 clarify，等待用户，不通过关键词补路由 |
| 工具不存在/越权 | 拒绝调用，记录受控 observation；模型可在预算内调整 |
| Agent 不可用 | 模型重新规划或明确失败；不偷偷替换成默认 Agent |

必须清理 MainAgent、旧 LLMPlanner/RulePlanner、启动 default rule、Gateway 静态直连 fallback 及旧测试依赖。生产目标启动图只允许一个 LLM 决策入口；离线测试使用注入的 FakeModel，不保留面向用户的假智能模式。

## 6. 工具体系

### 6.1 原生工具清单（拟新增）

| 工具 | 用途 | 执行边界 |
|---|---|---|
| `workspace.list` / `workspace.read` / `workspace.search` | 读取和定位授权项目内容 | 受控根目录、规范化路径、符号链接检查、结果长度限制；拒绝凭据路径 |
| `web.search` / `web.fetch` | 外部证据检索与取回 | 配置明确的 provider/可信 MCP Server；SSRF、重定向/DNS 校验和网络出口限制 |
| `workspace.apply_patch` | 应用明确补丁 | 独立工作区、预期文件 hash、冲突检测、审批绑定；不覆盖未确认用户改动 |
| `sandbox.run_tests` | 构建/测试及收集证据 | 隔离运行、命令和参数约束、CPU/内存/时间/输出上限；仓库测试本身也是不可信代码 |
| `artifact.publish` | 归档工具/Agent 产物 | 内部 artifactId，不让模型随意指定宿主文件路径或公开上传地址 |
| `delegate_agent` | 委派已注册专家 | A2A 适配；允许集合、预算配额、父子关系、取消与结果账本 |
| `memory.search` / `memory.read` | 按需召回长期记忆 | Controller 强制作用域、有效期、版本和上下文预算；不能读取其他主体/工作区 |
| `memory.propose` | 提交新增或纠正候选 | 不直接激活记忆；确认、版本冲突和遗忘走可信控制面，不授权模型直接改策略 |

工具名是目标业务标识，需映射到当前 `type/set/func` Registry 路径；不能把二者混为同一个已经存在的 API。`web.search` 未配置真实检索后端时明确不可用，不生成虚假搜索结果。

### 6.2 描述与授权

ToolDescriptor 至少包含：稳定工具名、版本、输入/输出 schema、读写属性、资源范围、超时、是否可重试、幂等支持、审批策略、输出上限。模型只看到当前允许集合及任务需要的描述。

ToolCallEnvelope 包含：runId/taskId/toolCallId、参数摘要 hash、工具版本、工作区授权、审批记录引用、执行版本和预算凭据。授权信息由平台生成，模型不能自报“已批准”。

所有实际执行路径经过 Tool Broker：原生工具和 Child Agent 的工具调用均要获得受限调用许可并回报结果，不因 A2A 调用就绕过工具治理。Child Agent 不直接修改 Orchestrator 执行表；通过内部协议更新账本。

### 6.3 MCP 与安全

- 原生工具先交付，MCP 作为受控扩展适配；安装/运行第三方服务不是普通工具调用的隐式副作用。
- 仅接入审核后的服务器和工具清单，schema/工具版本固定并纳入变更复核。
- MCP 的 readOnly/destructive 等 annotations 是提示，不是安全授权；服务端工具描述和返回内容均不能覆写系统权限。
- 读网页或源码得到的指令视为不可信数据；不能触发追加授权、读取密钥、修改策略或提升权限。
- 写工具、联网和测试各自有不同权限，不设置一个宽泛的“允许执行一切”开关。
- 前端 CodePreview 仍只展示文本；真正执行发生在后端隔离环境，不能在浏览器 eval 生成代码。

## 7. Harness：不仅是一个 Prompt

### 7.1 Runtime Harness 职责

| 部件 | 职责 |
|---|---|
| ContextBuilder | 最近上下文、置顶/引用、工具 call-result 配对；大结果外置为 artifact 引用 |
| MemoryController | 统一短期窗口/长期召回、作用域授权、候选确认、CAS 更新、过期与遗忘；模型不直接操纵后端 |
| LoopController | 驱动已有 ADK/模型接口，记录观察结果；不另套第二个同级无界 Agent 循环 |
| BudgetManager | 对主模型、修复、摘要/记忆抽取、专家和工具重试使用同一运行总预算 |
| PolicyGate / ToolBroker | 校验调用、执行前审批和版本检查、隔离资源 |
| CheckpointManager | 在安全边界保存状态、pending calls、结果引用和预算 |
| RunController | 暂停、审批、取消、恢复、超时和唯一终态 |
| EvidenceCollector | 记录工具真实退出状态、验证结果和产物；不把模型自评作为完成证明 |

拟议保守初值：总模型调用 16 次、工具调用 32 次、修复 1 次、重新规划 2 次、并行任务 3 个、委派深度 1、运行执行时间 180 秒；普通工具 15 秒，隔离测试 60 秒。它们**不是当前配置或压测结果**，P0 冻结后实施，长任务通过明确的 run profile 调整。

所有子任务预算先预留再派发，总量不能因并行或重试倍增；恢复不能重置已消费预算。25 轮 ADK 默认值不等于全链路模型预算，必须在各层统一计数。无进展检测可记录同工具/同参数 hash/同结果版本的连续重复；例如阈值 3 必须在 32 次工具预算内可达，达到后让模型澄清或终止，不偷偷换规则路由。

### 7.2 运行状态

```text
accepted → planning → executing → completed
                  ↘ awaiting_approval → executing
                  ↘ awaiting_input → planning
executing → paused / failed / cancelled
paused → 恢复验证 → executing / awaiting_approval / failed
```

审批等待不占用工作 goroutine 或数据库事务；持久化后释放执行资源。180 秒指执行预算，另设审批过期时间（建议初值 15 分钟）；不能让长时间等待审批消耗全部执行时间，也不能无限延长总生命周期。

审批绑定 runId、planId/revision、toolCallId、工具版本、参数 hash、工作区、授权主体和有效期；计划/参数变化使旧审批失效。批准计划不等于批准其中所有未来工具调用，具体调用仍需依据策略取得许可。

每个审批只能被其绑定的 toolCallId 消费一次。派发前，在校验当前 fencing version 的同一 SQL 事务中消费审批、预留预算并登记派发意图；提交后才执行工具。同参但新 toolCallId 不能复用旧审批。同一 toolCallId 的重试读取已有账本，并受批准的重试范围约束，不重复消费审批或无上限派发。事务提交与外部副作用仍不是原子操作，结果未知按下一节处理。

用户明确取消应保存 cancelled 并尝试取消远端任务，保持 `STATE_UPDATE(cancelled) + RUN_FINISHED(cancelled)` 语义；普通失败使用终止性 RUN_ERROR。不能只改变内存 cancelFunc。

### 7.3 checkpoint 与副作用

保存的是可序列化执行状态，不是 Go 栈或 cancelFunc：版本化计划、执行游标、上下文/产物引用、待处理调用、已完成调用、预算、授权版本、Agent/工具版本和审批引用。

| 崩溃位置 | 恢复策略 |
|---|---|
| 只读工具派发前 | 校验授权/预算后可重新派发 |
| 只读工具结果已保存 | 复用结果，注明版本；不可假装新抓取 |
| 幂等写工具派发后 | 使用同一幂等键查询/重试，不能生成新键 |
| 非幂等工具已派发但结果缺失 | 标记 unknown，先核对真实状态/人工处理，不自动重放 |
| 等待审批 | 恢复审批状态，验证 revision/hash/有效期 |
| Agent 远端任务状态未知 | 优先按远端 taskId 查询；若当前 A2A 不支持可靠查询/续传，明确 paused/unknown，不假设自动接续 |

ADK 当前工具调用与事件落库顺序必须专项审查，`pkg/adk/runner.go:202–238` 显示工具可先执行再追加事件。新增 checkpoint 不能仅把现有 session_events 原样搬到 Redis 就宣称完成恢复。

### 7.4 Memory 分层与存储

MemoryController 是 Harness 中独立的控制面，不等于 MemoryStore，不等于上下文裁剪，也不等于执行 checkpoint。设计借鉴 DSH 的可插拔记忆接入、LangGraph 的 thread/store 分层等已检索机制；来源和采用边界见选型文档第 7.1、10.1 节。

| 层次 | 内容与范围 | 存储与调用 |
|---|---|---|
| 工作记忆 | 当前 run 的目标、工具 observation、临时计划 | 进程内；安全边界写 checkpoint。仅当前任务，不自动转长期记忆 |
| 短期会话记忆 | 同一授权 thread/Agent 分支的消息与上下文摘要 | MySQL session_events + 带来源区间/版本的 session_context_snapshots；调用前组装工作窗口 |
| 执行 checkpoint | 游标、pending calls、已消费预算、权限/记忆版本引用 | MySQL；用于恢复控制，不直接当成用户事实注入模型 |
| 长期记忆 | 已确认偏好、项目事实、带来源的有效观察 | MySQL memory_entries / memory_revisions / memory_scopes / memory_mutations；跨会话但不跨授权范围 |
| 缓存 | 已授权条目或上下文窗口的加速副本 | Redis，键含作用域与版本、短 TTL；失效不丢真值，也不能绕过删除/权限判断 |

短期和长期都可以持久化：“短期”指 thread 作用域和用途，不等于只能放 RAM/Redis。浏览器聊天记录由 Gateway 管理；Orchestrator 通过已授权请求/内部接口获得有界历史，不直接查询 Gateway 表。

保存时长与模型窗口分开控制：P0 必须冻结 sessionRetention、run/checkpointRetention、Memory 条目有效期和 backupRetention；裁剪 prompt 不触发原始历史删除，读取也不自动延长保留期。运行中或审批中的 checkpoint 不可按普通缓存 TTL 淘汰；会话摘要不能比其授权来源更长久地用于模型。Redis 缓存建议 TTL 上限 5 分钟且不超过来源剩余有效期，版本撤销仍立即作用于新授权而不是等待 TTL。到期清理采用可观察任务，遇到活跃引用须按保留策略处理，不默默生成悬空引用。

现有基础只覆盖一部分：`pkg/adk/session.go:66–75` 是 Session CRUD；`pkg/runtime/pruning/pruner.go:58–88` 的首尾窗口和工具字符限长不是长期 MemoryPolicy，也不等于真实 token 预算。必须新增明确的 Controller，不把这些已有能力换个名字宣称完成。

### 7.5 MemoryPolicy：谁能读、谁能写

由可信配置/用户控制 API 生成、版本化保存；模型不能修改。MemoryController 在 Orchestrator 托管，`pkg/runtime/memory` 为拟新增的接口/客户端和存储适配位置，Child Agent 只能通过受限许可调用。

| 模式 | 读写行为 |
|---|---|
| `off` | 不复用过往会话或长期记忆；当前请求、必需系统约束、当前 run 工具结果仍可使用 |
| `session_only`（拟默认） | 只使用当前授权 thread 历史/摘要；不跨会话召回或写长期记忆 |
| `scoped_long_term` | 在允许范围读取 active 记忆；长期新建/更新先提候选，按策略确认 |

模式控制模型上下文复用，不自动删除已有聊天、checkpoint 或安全审计；UI 必须说明这个区别。读权限、写提议权限、候选确认/修改权限、遗忘权限分别授权，不能用一个 enabled 开关包办。

- MemoryScope 至少包含部署环境、workspaceId、可信 principalId；session/agent 是进一步收窄。项目共享范围需要单独授权，默认不跨项目继承“全局用户记忆”。
- 服务端从身份上下文生成范围，不接受模型/user_id 参数自行扩大。固定 Token 无法区分真实用户时，不宣称具备个人隔离；个人记忆功能须先有可验证的主体映射。
- 用户偏好不等于系统政策；所有召回记忆作为带来源的数据，不得覆盖安全规则、审批策略、工具权限或运行预算。
- 新建与更新拟默认 `approval_required`；可显式预授权某类低风险记忆，但权限范围、有效期和写入额度必须独立记录。敏感信息、凭据和模型隐藏思维链禁止入库，不能靠用户确认绕过密钥禁令。

### 7.6 短期记忆的写入与调用

```text
Gateway 验证身份/会话 → 映射授权 thread/Agent 分支
→ 读取 Session 事件与有效摘要 → 应用 MemoryPolicy
→ 保留系统约束、当前目标、置顶/引用、完整 tool call-result 对
→ 摘要 + 最近窗口 + 所需 artifact 片段 → ContextBuilder → 模型
→ observation/消息写入 Session → 安全边界保存新 snapshot/checkpoint
```

摘要保存 sourceEventRange、来源 Memory 引用、摘要模型/模板版本和更新时间，是可重建派生物，不替代原始事件。只在预算达到阈值时整理已完成的历史片段，不拆开未完成工具调用，也不伪造真实工具结果补配对。

拟初值：为模型输出和必需指令预留空间后，输入预算达到 80% 才尝试压缩，目标降至 60%；最终组装仍必须检查完整请求不超过模型上下文限制。摘要调用计入全 run 的 16 次模型预算；预算不足时使用有界历史/artifact 引用，必需上下文仍放不下则明确报错，不静默丢掉安全信息或 pending tool call。

Redis 快照必须绑定 thread、Agent 分支、session version、Memory scope epoch；不能给并发分支复用同一个可写上下文对象。恢复时重新校验来源和权限，不能原样注入旧 checkpoint 中已撤销的 Memory。

### 7.7 长期记忆的召回与更新闭环

**调用路径**：

```text
模型调用前的有界自动召回 / 模型提出 memory.search、memory.read
→ MemoryController 身份与范围过滤
→ SQL 核验策略 epoch、active 状态、有效期与条目 revision
→ 缓存或 MySQL 检索 → 授权候选排序 → 合并去重与 token 限额
→ 带 memoryId/revision/source 的上下文数据 → 本轮模型调用
```

先授权过滤再检索/排序，不能先检索所有用户数据再让模型过滤。初版结构化字段与 MySQL 全文检索，不声称已经具备 embedding 语义搜索；中英文同义词召回由评测决定是否需要独立索引扩展。

自动预召回与显式工具召回使用同一条管线，以 `(memoryId, revision)` 去重，不重复附加。可以少量固定已确认核心偏好，其余按需检索；不把全部记忆永久塞入 system prompt。

**写入/更新路径**：

```text
用户明确请求记住 / 工具观察后的候选提议
→ memory.propose（不立即生效）
→ schema、敏感性、来源、作用域、重复/冲突检查
→ 需要时由用户确认精确内容与范围
→ 同事务校验 expectedRevision/权限与审批、幂等写入、scope epoch++、outbox
→ active 新版本 → 下一次授权召回可见
```

- 候选包含拟新增/替代内容、来源 message/artifact、时间和 expectedRevision；模型自评置信度不是确认依据。
- 条目至少记录 memoryId、scope、kind、logicalKey、content、sourceRefs、revision、status、createdAt/verifiedAt、expiresAt。
- 对同一 logicalKey 用唯一约束和 CAS 更新；冲突返回待复核，不能最后写入者直接覆盖事实。旧修订退出召回；失败/回滚不增加可见版本。
- `memory.propose` 是受 Broker 管理的写提议；模型不直接执行 approve、无审批 update/delete 或修改 MemoryPolicy。确认绑定候选、目标 revision、内容 hash、toolCallId 与范围。
- 先在请求路径内提交小规模候选/确认操作，数据库提交后才报告已记住；如后续引入后台抽取，`accepted/pending` 不等于 active 或已可检索，必须有可查询任务状态。

建议初值均待 P0 冻结：每轮最终注入最多 5 条（其中固定核心最多 2 条），最多占可用输入预算 10% 且不超过 2,048 tokens；单条正文最多 4 KiB，每个范围最多 200 条 active；候选 24 小时过期，未指定期限的长期条目默认 30 天后需复核。字节限额不冒充 token 限额；查看/命中记忆不自动续期，达到配额时提示整理而非无界追加。

### 7.8 遗忘、缓存一致性与用户控制

用户通过 Gateway 可查看记忆与来源、确认候选、纠正、删除单条/整个范围、暂停长期记忆和调整保留期；Gateway 转发至 Orchestrator MemoryController，不直接操作其表。自然语言请求只能生成待确认的操作，不把任意对话当作删除授权。

1. 遗忘事务将条目标记 revoked/tombstone，增加 scope epoch，失效相关候选并产生仅含 ID/版本的清理事件。返回 `recallDisabled=true` 与独立 purgeStatus，不把异步物理清除写成已完成。
2. 每次读记忆或取得模型上下文许可时，检查 SQL 权威 epoch；Redis 缓存命中仍要核验，SQL 不可用则省略长期记忆或明确阻止依赖它的任务，不能读旧缓存凑答案。无关的当前输入/短期会话可按策略继续。
3. 新授权以 SQL 校验/登记的事务时点为界。已准备但未获许可的旧上下文丢弃重建；已经授权或发送给外部模型的在途调用不能被数据库删除瞬间撤回，须记录在途边界并在下一轮重新组装。
4. Session 摘要/checkpoint/检索结果携带 memoryId/revision/source lineage；删除或更新后阻止旧派生片段再次注入。旧 checkpoint 恢复、重复 outbox、延迟摘要任务都必须核对 epoch，不得重新激活旧记忆。
5. 清理条目正文、旧修订、Redis 副本及可定位的派生摘要；审计保留最小操作元数据，不保留被删正文。备份按单独期限到期，恢复备份后重放删除标记再开放召回。
6. “忘记长期记忆”不同于“删除原始聊天及全部来源”：原始对话仍可能含同一事实。UI 明示两种范围；长期记忆删除应阻止从同一来源自动重新提取，用户重新明确确认才可建立新记忆。若要求删除来源和派生物，跨 Gateway/Orchestrator 的清理按任务状态追踪，不伪称单事务全部完成。

日志仅记录记忆命中 ID/版本、来源类别、字节/token 用量和操作结果，不默认记录完整记忆、prompt 或工具原文。以上控制解决记忆复用的边界，不承诺清除已发送到第三方模型或用户下载文件中的所有内容。

## 8. 流式事件、投影和恢复

1. Orchestrator 为每个 run 分配单调 `seq`，先提交 SQL 日志，再经 outbox 发布 Redis 短期副本。
2. 文本增量可按小批次持久化，试验初值为 50 ms 或 32 KiB 中先达到者；生命周期事件、工具派发和终态必须立即冲刷。未提交增量不向浏览器确认可重放。
3. 初版 relay 按 run 保序；重复投递仍可能发生。流中的 payload 保留 SQL seq，发现缺口从 SQL 补齐，不能用 Redis Stream ID 代替业务版本。
4. Gateway 经 gRPC 订阅内部事件，先幂等更新消息投影，再发送公开事件；发送完成终态前确认对应投影已提交，保证随后读取历史不会缺少已宣告完成的结果。共享投影消费位置与每个订阅的发送游标是两套状态：已投影事件仍要发送给尚未收到的订阅者。
5. 一个内部事件可能映射多个 AG-UI 事件，因此公开事件 ID 使用稳定的 `(runId, seq, subIndex)`。中段重连要重新取当前 seq 并映射，只跳过该订阅已经确认的 subIndex，不能只按 seq 去重。
6. SQL 历史切换 Redis 实时流时，先捕获 Redis 尾游标 R，再读取 SQL 已提交水位 H 并回放到 H，然后从 R 开始 XREAD，按业务 seq/公开游标过滤重复。Redis 不可用或 R 已被裁剪时直接走 SQL 补读。订阅期间定期对账 SQL 最新水位/终态，即使最后一条终态丢失且没有后继 seq，也能补齐；不是只在观察到跳号时恢复。
7. 前端记录最后实际消费游标；重连从该位置续传。固定事件映射版本；若映射版本或 SQL 日志保留窗口不兼容，则返回明确的 snapshot/resync 要求，而不是静默重启任务。历史水位 H 不是浏览器消费确认，成功写入 socket 也不等于浏览器已处理事件。
8. 断开浏览器订阅默认不取消已经接受的 durable run；显式 cancel 才取消。此行为不同于当前绑定请求 context，必须同步修改公开协议、UI 停止按钮和验收测试。
9. 慢客户端使用有上限的发送队列和写 deadline；队列满则断开该订阅、允许恢复，不让无限队列拖垮执行，也不删除已提交日志。

Redis 故障和 SQL 故障处理不同：SQL 不可用时停止接受需要执行或写工具的新任务，现有任务不得绕过账本继续产生副作用；Redis 不可用可有限度降级至 SQL 补读。

## 9. 可观测与评测

Runtime Harness 是运行控制；Eval Harness 是验证这些控制和输出质量的独立测试入口，不能用“加了一套测试”冒充实现 Harness。

建议 trace 链覆盖 requestId→runId→taskId→toolCallId→checkpoint/seq，采用 OpenTelemetry 接口，不绑定某家云平台。保存简短决策摘要与可验证操作记录，不保存/索取模型隐藏思维链；工具输入输出按敏感级别截断或只记 hash/引用。

评测至少包含：语义理解/专家选择、工具参数正确性、真实任务完成率、越权拒绝、重复副作用、审批重放、恢复正确性、上下文配对、成本预算和多订阅事件一致性。明确分开 FakeModel 功能测试、真实模型评测、实际 MySQL/Redis 故障测试、浏览器 E2E。

Memory 单列指标：授权范围内 Recall@k、注入 token/延迟、跨范围泄漏、过期/删除记忆误召回、CAS 更新冲突、摘要来源完整性、遗忘后旧任务/缓存复活。定义 R 为当前查询已标注相关且已授权、active、未过期的 memoryId 集合，Recall@k = |TopK ∩ R| / |R|；分母不是 k 或全部 active 条目，不把它称为回答正确率。R 为空时单列“应不召回”负面用例，不计入召回率均值。固定合成集覆盖中文、同义问法、矛盾更新和多 Agent 场景，安全违规以实际调用/召回次数为断言，不用平均分掩盖。

## 10. 必须先更新的契约

| 变化 | P0 需修订的事实源 |
|---|---|
| 取消所有 rule fallback | `intent-orchestration.md`、`planner-input.md`、`orchestration-plan.md`、相关编排执行指引 |
| MySQL 主存储 + Redis 辅助 | `storage-profiles.md`、`mysql-schema.md`、`data-model.md`、部署拓扑文档 |
| Orchestrator 持久化执行，不存浏览器会话 | `project-architecture.md`、`gateway-orchestrator.md`、新 Harness 契约 |
| 长短期 Memory 存储/调用/控制 | 拟新增 Memory 契约，更新 `adk-runtime.md`、`agent-config.md`、`data-model.md`、`mysql-schema.md`；冻结 scope/模式/配额/TTL/更新遗忘与在途边界 |
| 审批恢复/取消、游标重放、gRPC 扩展及 Memory 管理 | `gateway-orchestrator-api.md`、`gateway-orchestrator-events.md`、公开 API/AG-UI schema |
| 工具 Broker/MCP/隔离执行 | `adk-runtime.md`、`agent-config.md`、`security-boundaries.md`、A2A/Artifact 契约 |

本轮只重写两份方案并新增一份计划，不修改上述契约文件、依赖、数据库或代码。实际实施路径、阶段门禁和交付状态见实施计划。
