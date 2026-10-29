# AgentHub v2 技术选型与优秀项目对照

- 日期：2026-10-29；状态：**提案，未实施**。
- 范围：Redis + SQL、LLM-only 编排、真实工具、Runtime/Eval Harness、长短期 Memory 存储与控制。
- 配套：[架构设计](optimization-architecture.md)；[实施计划](../plans/agenthub-redis-sql-harness-plan.md)。
- 本版明确替代旧版“不引入 Redis、仅优化 SQLite”的结论。

## 1. 最终推荐组合

| 层次 | 推荐 | 不选择/不误称 |
|---|---|---|
| 主存储 | MySQL，优先 8.4 LTS 系列中受支持的安全补丁版本 | 不继续把 MemoryStore + SQLite 镜像当生产主存储 |
| 辅助存储 | Redis：能力缓存、心跳、短期 Streams 事件副本 | Redis 不是唯一 checkpoint、审批或工具账本 |
| 语义决策 | MainAgent LLM + schema 校验 + 有限模型修复 | 无关键词路由、无规则 fallback、无隐式默认 Agent |
| Agent 执行 | 现有纯 Go ADK + 薄 Runtime Harness | 不并行维护 Eino、LangGraph、自研三套同级循环 |
| 多 Agent | 模型选择的 agents-as-tools，主管保留控制，子任务上下文隔离 | 不默认采用共享全部历史的多层 handoff |
| Memory 控制 | MemoryController + MySQL 版本化记忆；默认 session_only，长期记忆显式授权 | 不把会话历史/Redis 缓存当长期记忆，不允许模型无条件自动保存 |
| 工具 | 原生受控工具优先，MCP 作为扩展适配 | 不把工具注解当权限，也不开放宿主机任意 shell |
| 可靠状态流 | SQL event journal + transactional outbox + 幂等投影 | 不宣称 SQL/Redis 双写天然原子，不宣称任意工具 exactly-once |
| 通信 | Gateway↔Orchestrator gRPC；Orchestrator↔Agent A2A；浏览器 AG-UI SSE | 不让前端直接连接数据库或内部 Agent |
| 可观测与评测 | OpenTelemetry 接口、结构化审计、项目内 Eval Harness | 不要求先采购云观测平台，不用模型自评分替代验收 |

选择 MySQL 而非泛称“SQL”，因为 [storage-profiles.md](../contracts/storage-profiles.md) 和 [mysql-schema.md](../contracts/mysql-schema.md) 已明确该目标。Redis 是新增架构能力，需要同步修改契约和部署计划，而不是加一个缓存包就结束。

## 2. 网上优秀项目：借鉴机制，不照搬名称

以下是官方文档级研究，不是对第三方仓库的源码审计或性能测评。当前机器没有 `gh`，未安装/克隆外部项目，未运行其示例。来源详见第 10 节。

| 项目/实践 | 已核验机制 | AgentHub 借鉴点 | 不能直接照搬的部分 |
|---|---|---|---|
| **CloudWeGo Eino / ADK** [S1–S3] | Go Agent/工具执行；CheckPointStore 支撑 interrupt/resume；FAQ 当前不推荐 Supervisor/Transfer，建议 DeepAgent 或 ChatModelAgent + AgentTool | Go 接口组织、工具化专家、显式检查点/暂停恢复、子上下文隔离 | Eino 存储能力不等于 AgentHub 已有 Redis+MySQL 接线；迁入其 ADK 将改变现有 Runner/A2A/事件适配 |
| **LangChain Deep Agents** [S4] | 面向多步任务的 harness，围绕规划、文件系统上下文、子 Agent、摘要/外置上下文等能力组织 | 把上下文和工具工作区当成一等能力；进度、证据与原始聊天历史分开 | 虚拟文件系统不是 OS 安全沙箱；Python 生态不直接解决 Go 接口兼容 |
| **LangGraph** [S5] | checkpoint 保存 thread 的图状态，store 用于跨运行记忆 | 区分会话历史、执行 checkpoint、长期信息；恢复须绑定稳定运行身份 | 本轮页面提取只足以支持这些持久化概念，不据此宣称任意工具自动安全重放 |
| **OpenAI Agents SDK** [S6–S7] | 模型驱动/代码驱动编排区分；agents-as-tools 与 handoff 控制权不同 | 主 Agent 通过工具委派专家并统一汇总；模型选择能力，平台约束执行 | 官方已核验 Python/TypeScript 实现；Go API 客户端不等同原生 Go Agents SDK |
| **Anthropic 长任务 Harness 实践** [S8–S10] | 通过增量工作、进度产物与验证连接多轮任务；区分 agent 与 agent harness 的评测 | 每次运行产生可恢复进度和完成证据；将测试工具纳入工作闭环 | 属于任务实践而非通用 Go 平台；不强制所有任务都经过固定三个 Agent |

不使用 GitHub 星数、“最佳框架”营销标题或第三方未经复验的基准作为选型证据。网页内容、版本和推荐会变化，来源记录的是本次访问时的证据。

## 3. 决策 ADR-V2-01：MySQL + Redis，权威状态只放 SQL

**状态：拟采纳，待 P0 批准。**

### 对照

| 方案 | 优点 | 主要代价 | 选择 |
|---|---|---|---|
| MySQL + Redis | 符合现有 SQL 目标；事务/唯一约束承载账本，Redis 做热点缓存/短期广播 | 需要 outbox、去重、故障降级、运维和版本许可审查 | 推荐 |
| PostgreSQL + Redis | 也是可行的事务型组合 | 没有当前需求证明必须更换 SQL 方言；增加迁移验证 | 不选，不是技术能力不足 |
| SQLite + Redis | 本地部署简单 | Redis 不能解决 SQLite writer 状态归属、主 Store 接入和多实例恢复边界 | 不作为目标 |
| Redis-only | 单一访问接口、内存读取方便 | 审批、终态、工具账本依赖 Redis 持久化/复制/淘汰配置，可靠语义更难解释 | 不选 |
| SQL-only | 一致性和运维更简单 | 缺本次要求的独立缓存和事件分发能力 | 可作 Redis 故障下有限降级，不作为目标拓扑 |

### 为什么不用分布式双写

先在 SQL 一个事务中提交执行状态与 outbox，然后异步发布 Redis。发布成功但确认丢失会重复，接收方按业务事件键去重；这比“分别写 MySQL 和 Redis，假定都成功”更容易证明。

运行领取可采用 SQL 条件更新；若批量领取 outbox 选择 `FOR UPDATE SKIP LOCKED`，只用于队列式领取，不能把跳过锁行的结果当完整一致的业务查询。本轮 MySQL 原文页面抓取失败，相关索引摘要只支持该候选方向，P2 要以实际 MySQL 集成测试确认语义。

### Redis 技术取舍

- Streams 与 Pub/Sub 不等价：Pub/Sub 官方说明是 at-most-once，断连消息丢失 [S12]。
- Streams 支持游标、不同消费方式及裁剪 [S11]；consumer group 重领/ACK 仍需应用幂等，不是外部副作用的 exactly-once。
- SSE 多订阅广播选择独立读取游标，不能用同组消费分摊给多个浏览器。
- 首版不依赖新版 Streams 特定幂等扩展；以 SQL seq/唯一约束实现跨版本可解释的幂等。
- 版本在实施锁定：采用受支持补丁，核验所选 Redis 发行版许可证和部署方式；不使用漂移的 `latest` 标签，不在文档猜最新小版本。

## 4. 决策 ADR-V2-02：LLM-only，取消所有规则回退

**状态：拟采纳；需要修改当前允许 RulePlanner fallback 的契约。**

当前最重要的证据不是 `rule_planner.go` 的存在，而是 `main_agent.go:98–109` 的实际回退，以及 `cmd/orchestrator/main.go:241–277` 的默认 rule/缺配置降级。

| 方案 | 效率/可靠性特点 | 结论 |
|---|---|---|
| 关键词/规则路由 | 确定、便宜，但违背本轮要求；能力扩展依赖人工词表 | 删除目标启动路径中的实现 |
| LLM + 规则 fallback | 演示更容易持续输出，但失败时语义突然变为词表匹配 | 不保留 |
| LLM + 校验 + 有限模型修复 | 决策来自模型，错误边界明确，允许受控重试 | 推荐 |
| 任意 Agent 自由 handoff | 灵活但上下文、控制权、预算和终态更复杂 | 首版不选 |

“无规则路由”不意味着删除鉴权、JSON/schema 校验、工具允许集合、依赖调度和预算上限。模型可以提出任意文本，但不能越过这些执行约束。

模型故障时允许显式错误或模型生成澄清；不允许偷偷选择第一个 Agent。不同模型之间的重试若日后启用，也必须保留真实 provider/model 记录、相同授权和预算，不把旧规则回退换个名字。

## 5. 决策 ADR-V2-03：复用 ADK，增加薄 Harness

**状态：拟采纳。Eino 是设计参照，不是本批已决定引入的运行时依赖。**

| 方案 | 收益 | 代价 | 选择 |
|---|---|---|---|
| 现有 ADK + Runtime Harness | 复用 Tool/Plugin/Runner/A2A 和测试；符合纯引擎边界 | 需要补执行状态导出、调用前 checkpoint、恢复接口；必须证明副作用顺序 | 推荐 |
| 整体迁到 Eino ADK | 可利用成熟 Agent/HITL 抽象 | 需要一次性验证全部事件映射、工具调用、预算、A2A 和存储适配 | 本轮不直接替换；必要时用独立 ADR 重新决定 |
| 引入 Python LangGraph 服务 | 快速接入其生态 | Go/Python 两套发布、RPC、状态和调试链路，超出必要范围 | 不选 |
| 在现有 Runner 外再套一个完整 Runner | 看似可复用多个框架 | 重复工具调用、预算和终态所有权不清 | 禁止 |

薄 Harness 不是只改 prompt，也不是重写通用工作流引擎。它使用已有纯接口，在模型调用和工具副作用的边界实施控制；Redis/MySQL 细节留在 Runtime/服务的具体适配层。

P3 的关键验收是“进程重启后从安全 checkpoint 恢复，并且不重复非幂等工具”，不是“新增了 Harness 目录”。若现有接口不足，允许最小扩展 `pkg/adk` 的纯状态/存储接口；不把数据库依赖塞进 ADK。

## 6. 决策 ADR-V2-04：原生工具优先，MCP 受控扩展

**状态：拟采纳。**

- 工具语义：模型选择、Broker 授权、隔离环境执行、真实结果返回。
- 原生文件工具优先，参数和失败行为可控；从只读工具到补丁/测试逐步开放。
- 通过 MCP 扩展专业外部能力，但接入仍需服务器和工具 allowlist、版本、schema、网络/身份边界。
- `web.search` 使用显式配置的外部检索后端或经审核 MCP 工具；文档不指定一个未经验证的免费搜索服务。
- `sandbox.run_tests` 使用结构化 executable/argv 与隔离资源；测试脚本可运行任意项目代码，因此命令白名单本身不够。
- 写工具采用工作区 hash/版本检查和幂等账本。审批绑定 toolCallId 和实际参数且只能消费一次；消费审批、预留预算、登记派发意图在当前 fencing 校验下同事务提交，不能把“批准一次运行”解释为批准无限写入或部署。
- 不开放任意 SQL 执行工具、宿主机任意 shell、任意 MCP 安装、默认公网发布/部署工具。

MCP 官方明确工具 annotations 不可信，除非来自受信服务器 [S13]；即使可信也不能替代平台权限。风险分类由平台审定，不由模型或远端工具自己声明。

Sandbox 初版采用隔离进程/容器及独立工作区，配合非 root、受限挂载、资源/网络限制；不能声称普通容器配置天然抵御所有恶意代码。需要更强隔离时再评估专用 sandbox/microVM，而不是先开放宿主执行后补安全。

## 7. 决策 ADR-V2-05：工具化专家，不增加固定业务路由

**状态：拟采纳。**

参考 OpenAI agents-as-tools 的控制权和 Eino AgentTool 设计：MainAgent 根据注册能力描述提出委派动作，Coordinator 调用 A2A，结果作为 observation 回到主循环。

- 主 Agent 保留目标、预算和验收条件；子 Agent 只收到最小输入和受限工具集合。
- 不强制每个任务都“规划→编码→审查→部署”；简单问题可以直接回答，需要证据时才选择工具/专家。
- 原本 test/review/deploy 等 Agent 不等于对应能力已经安全可用；先审计是否只生成文本、是否真的调用工具。
- 独立只读任务可并行，副作用任务受工作区资源冲突约束；模型产出依赖之后由确定性调度执行，不能在调度器中重新解释用户意图。

## 7.1 决策 ADR-V2-06：长短期 Memory 的存储、调用与控制

**状态：检索后的设计提案，尚未接线或运行外部项目。** DSH 在这里指 **DeepSeek Harness**，不是助手自身的文件记忆。新增来源 M1–M17 见第 10.1 节。

### 项目对照：存哪里、什么时候读写

| 项目 | 存储与可见性 | 调用/更新机制 | 本项目采用与限制 |
|---|---|---|---|
| DSH 会话/持久化 [M1–M3] | 官方文档索引区分内存 Session 事件日志与可插拔持久化；存储接受和 durable flush 不能混为一谈 | 生命周期日志落地与会话恢复，不等于自动建立可跨会话召回的事实库 | 借鉴事件与存储边界；正文访问受限，不推断默认后端、精确写时机或崩溃保证 |
| DSH 第三方 Memory MCP [M2] | 官方指南索引列出默认关闭的接入示例，包括本地 JSONL 知识图谱参考服务 | 模型通过暴露的读/搜索/写工具调用；具体行为属于所接服务 | 借鉴“显式接入、工具化调用”，不把社区插件能力描述为 DSH 内置长期记忆 |
| DSH + Hindsight [M4] | Hindsight memory bank，作者文章按 repo 划分；Cloud 或自托管后端 | 自动 retain，后续新会话 recall；文章不足以证明所有插件都逐轮召回 | 借鉴 scope 和预注入，AgentHub 不照搬未经用户许可的自动永久保存 |
| DSH + Hologres/Mem0 插件 [M5] | Hologres 长记忆服务通过 Mem0 兼容 API 接入，不是本地 Session 数据库 | 文档描述 Agent 调用 add_memory/search_memory，可由用户请求管理；userId 是数据作用域 | 借鉴工具生命周期；ID 过滤不是认证，不因此引入 Hologres 或自动把数据上传云端 |
| LangGraph / LangChain [M6–M8] | 短期 thread state 由 checkpointer 保存；长期 namespace/key store 跨会话；示例 InMemoryStore 不代表生产持久化 | 步骤开始读状态，调用完成更新；长期工具用 runtime.store 读写；hot path 与后台整理可区分 | 借鉴分层、每轮边界和按需工具；AgentHub 使用自己的 MySQL Repository，不宣称框架已有 MySQL 适配 |
| Mem0 [M9–M13、M17] | OSS 的向量记忆与操作历史、Platform 托管记忆要分开；检索结果需应用/工具送入模型，不是天然常驻上下文 | entity ID 过滤，按 memoryId 更新/删除；Platform 批量删除返回 event_id 可表示排队；AsyncMemory 是异步调用接口 | 借鉴独立作用域与生命周期；过滤不等于鉴权，异步不等于已可见/已删除，没有通用 CAS 证据就由本项目补版本检查 |
| Letta v1-sdk legacy [M14–M16] | core blocks 跨交互持久且始终在上下文；archival 按需语义召回，不能简单等同“core=短期RAM” | 索引描述记忆工具/SDK更新、archival search/insert；共享块整体重写存在冲突风险，detach 不等于删除 | 借鉴小型核心热区+检索冷区；正文未取得，不照搬 legacy API/read_only 当认证，也不引入其数据库/向量栈 |

### 选择

- 短期：MySQL session_events / 版本化上下文摘要作为可恢复来源，进程内工作上下文为执行视图；Redis 仅作有版本/TTL 的可选加速。
- 长期：MySQL 的作用域、记忆条目、修订和变更记录为真值。先做结构化过滤与有界全文检索，明确中英文同义词召回的局限；不把 MySQL 普通查询称为语义向量检索。
- 调用：少量授权记忆在模型调用前注入；需要更多证据时由模型调用 memory.search/read。两路通过同一个 MemoryController 去重、校验版本并共享 token 预算。
- 控制：默认只用当前会话；跨会话读取、长期写入和自动整理分别授权。模型可以提出记忆候选，不能直接提升权限、永久保存或覆盖冲突事实。
- 更新/遗忘：SQL CAS 修订与 scope epoch，删除先阻止新召回，再完成缓存/派生物清理；不依赖 Redis TTL 提供删除正确性。

不直接引入 Mem0/Letta/Hindsight 服务，是为了维持 Go、MySQL+Redis 和现有数据边界；这里借鉴机制而不是声称自研语义召回已达到它们的质量。若有界全文检索在固定中英文记忆集上不满足召回门禁，再单独评估向量索引/外部 Memory 后端，其授权、删除和数据出域仍受相同 Controller 约束。

## 8. 效率设计与测量口径

### 实施效率

P0 固定接口与契约后，LLM-only 清理、SQL 存储、只读工具/Harness 骨架可分文件并行；Memory、审批恢复、Redis 重放和写工具只在前置门禁通过后整合。详细允许路径和依赖见计划，不靠同时大改所有模块缩短工期。

### 运行效率

| 手段 | 限制与验证 |
|---|---|
| 单一工具目录、按授权筛选 schema | 不把所有工具描述灌入每轮 prompt；记录输入 token/可见工具数 |
| 历史分页、上下文窗口和大结果引用 | 保留 tool call-result 配对及引用/置顶；不能简单截断全部事件 |
| 分层 Memory 召回 | 先 scope/status/expiry 过滤，再检索；固定核心与按需召回共享 token 配额，避免重复注入 |
| 能力与 Memory 缓存 | 版本化缓存，权限/记忆 epoch 从 SQL 复核；TTL 不能作为遗忘正确性保证 |
| 文本小批持久化 | 暂定 50 ms / 32 KiB，需测首字延迟和恢复一致性；终态立即提交 |
| 有限并行 | 全局预算先预留，独立任务才并行；避免嵌套 Agent 导致成本倍增 |
| 共享 HTTP/SQL/Redis 连接池 | 不每请求新建客户端；池大小通过等待/复用率实测确定 |

指标分层：平台开销、模型耗时、工具耗时、数据库等待、首字延迟、完成时间、token/成本、工具成功率、恢复成功率。相同模型、数据集、资源和并发下比较；不把 provider 变快算作存储优化收益。

不预先宣称 QPS 翻倍、零丢失、跨 Pod exactly-once 或“有 Redis 就可水平扩展”。凡涉及两个 OS 进程、共享存储和真实网络的结论，要有对应验收。

## 9. 主要代价与不做事项

- MySQL 与 Redis 增加部署、备份、升级、监控和故障演练成本；不能只交付本机内存测试。
- SQL event journal/outbox 增加写入量，应定义保留/归档策略；Redis trimming 不代表 SQL 数据可以随意删除。
- artifact 不能只落在会随容器消失的目录：首版可用持久卷支持单机部署；多节点恢复需共享持久存储，在相应门禁前不宣称可用。
- 真模型和联网工具会向外部服务发送输入；合成数据测试不等于真实项目数据不会出域。
- 不增加 Redis 向量库、通用 RAG、复杂多租户用户系统、Temporal/Kafka 或多语言微服务；它们不是本轮要求的前置条件。
- 数据库/协议迁移和新增 Harness 的决策须按 [ADR Policy](adr-policy.md) 在 P0 确认；本文件包含提案级决策记录，不能代替实施批准。

## 10. 公开来源与核验范围

访问日期均为 2026-10-29。`正文` 表示本轮通过网页工具取得官方内容；`代理正文` 表示由并行研究代理核验官方页面；`索引` 表示只取得官方搜索摘要。未核验源码、许可证全文或第三方项目运行性能。

| ID | 官方来源 | 支持的结论 | 核验 |
|---|---|---|---|
| S1 | [Eino Agent 快启](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_quickstart/) | Go Agent/Runner/工具入口 | 代理正文 |
| S2 | [Eino HITL](https://www.cloudwego.io/docs/eino/core_modules/eino_adk/agent_hitl/) | CheckPointStore、interrupt/resume、应用负责接入存储 | 正文 |
| S3 | [Eino FAQ](https://www.cloudwego.io/docs/eino/faq/) | 当前不推荐 Transfer/Supervisor，建议 DeepAgent 或 ChatModelAgent + AgentTool | 正文 |
| S4 | [Deep Agents overview](https://docs.langchain.com/oss/python/deepagents/overview) | harness、规划/上下文文件系统/子 Agent 等设计 | 正文 |
| S5 | [LangGraph persistence](https://docs.langchain.com/oss/python/langgraph/persistence) | thread checkpoint 与跨运行 store 的区别 | 正文，提取范围有限 |
| S6 | [OpenAI Agents SDK 多 Agent 编排](https://openai.github.io/openai-agents-python/multi_agent/) | agents-as-tools、handoff、模型/代码驱动编排区分 | 代理正文 |
| S7 | [OpenAI Agents SDK TypeScript](https://openai.github.io/openai-agents-js/) | 官方 TypeScript/Python 生态入口 | 代理正文 |
| S8 | [Anthropic 长任务 Harness](https://www.anthropic.com/research/effective-harnesses-for-long-running-agents) | 增量工作、进度与验证实践 | 代理正文 |
| S9 | [Anthropic 应用开发 Harness](https://www.anthropic.com/research/harness-design-long-running-apps) | 随任务/模型演进调整 Harness，不固定模式 | 代理正文 |
| S10 | [Anthropic Agent 评测](https://www.anthropic.com/research/demystifying-evals-for-ai-agents) | 区分 Agent 系统与评测 Harness | 代理正文 |
| S11 | [Redis Streams](https://redis.io/docs/latest/develop/data-types/streams/) | 独立游标、消费者组、重领与裁剪 | 正文 |
| S12 | [Redis Pub/Sub](https://redis.io/docs/latest/develop/pubsub/) | at-most-once，断连后不可补发 | 正文 |
| S13 | [MCP Tools 规范](https://modelcontextprotocol.io/specification/2026-07-28/server/tools) | 工具 schema 与不可信 annotations 边界 | 索引 |
| S14 | [MySQL 8.4 Transaction Model](https://dev.mysql.com/doc/refman/8.4/en/innodb-transaction-model.html) | InnoDB 行级锁和一致性读背景 | 索引 |

MySQL locking reads 页面正文取回失败，LangGraph durable-execution 页面未取到足够的重放/副作用细节；本方案的 SQL outbox、幂等账本和 unknown 状态是 **AgentHub 自身设计要求**，不是声称这些网页或框架自动提供完整保证。

## 10.1 Memory 专项来源与研究边界

本次补充先检索项目资料再形成方案，访问日期沿用 2026-10-29。DSH 官方网站链接到下列官方静态文档；正文访问遭本机策略拦截时，只引用已取得的搜索索引并降低结论强度，没有改用其他下载方式绕过限制。

| ID | 来源 | 证据与用途 |
|---|---|---|
| M1 | [DeepSeek Harness 官方入口](https://www.deepseek.com/harness/) | 正文；确认 DSH 名称、插件化和官方文档入口 |
| M2 | [DSH 第三方 Memory MCP 指南](https://deepseek-harness.github.io/deepseek-harness/en/guide/mcp-memory) | 官方索引；默认关闭的第三方接入示例和工具化记忆，不代表官方内置长期后端 |
| M3 | [DSH Session Persistence](https://deepseek-harness.github.io/deepseek-harness/en/reference/subsystems/persistence) | 官方索引；会话事件日志与持久化服务区分，未核验源码/具体故障保证 |
| M4 | [Hindsight 作者的 DSH 集成介绍](https://hindsight.vectorize.io/blog/2026/08/14/deepseek-harness-persistent-memory) | 正文；repo-scoped bank、retain 与新会话 recall；这是供应方插件，不是 DSH 内置能力 |
| M5 | [阿里云：DSH 接入 Hologres 长记忆](https://help.aliyun.com/zh/hologres/user-guide/integrate-hologres-long-memory-service-with-deepseek-harness) | 厂商正文；Mem0 兼容 API / 插件的 add_memory、search_memory 与作用域；未运行或安装 |
| M6 | [LangChain Memory 概念](https://docs.langchain.com/oss/python/concepts/memory) | 正文；短期 thread 与长期 namespace、请求路径与后台写入差别 |
| M7 | [LangChain Short-term Memory](https://docs.langchain.com/oss/python/langchain/short-term-memory) | 正文；步骤边界读取/更新、trim/delete/summarize、工具消息配对 |
| M8 | [LangChain Long-term Memory](https://docs.langchain.com/oss/python/langchain/long-term-memory) | 正文；namespace/key、runtime.store 访问与 InMemoryStore 的持久化限制 |
| M9 | [Mem0 Add Memory](https://docs.mem0.ai/core-concepts/memory-operations/add) | 正文；作用域/到期配置，平台和 OSS 行为不能混用；不据此证明删除或异步可见性 |
| M10 | [Mem0 Entity Scope](https://docs.mem0.ai/platform/features/entity-scoped-memory) | 代理正文；user/agent/app/run ID 是过滤关联，不是身份认证或权限继承 |
| M11 | [Mem0 工作与调用](https://docs.mem0.ai/core-concepts/how-it-works) | 代理正文；记忆操作与检索结果如何供应用使用；不是常驻模型上下文 |
| M12 | [Mem0 Delete Memories](https://docs.mem0.ai/api-reference/memory/delete-memories) | 代理正文；Platform 批量删除的过滤与 event_id/后台处理，不能当即时完成 |
| M13 | [Mem0 AsyncMemory](https://docs.mem0.ai/open-source/features/async-memory) | 代理正文；OSS 异步接口，不等同索引就绪、事务或 CAS |
| M14 | [Letta v1 Memory Blocks](https://docs.letta.com/v1-sdk/memory/memory-blocks/) | 官方索引；legacy v1 常驻上下文块，正文抓取失败，不外推为最新接口 |
| M15 | [Letta v1 Archival Memory](https://docs.letta.com/v1-sdk/memory/archival-memory/) | 官方索引；长期按需检索与工具入口，不据此选择 AgentHub 数据库 |
| M16 | [Letta v1 Shared Memory](https://docs.letta.com/v1-sdk/memory/shared-memory/) | 官方索引；共享块更新/挂载的并发与删除边界，不能把 read_only 当认证 |
| M17 | [Mem0 OSS Overview](https://docs.mem0.ai/open-source/overview) | 代理正文；OSS 与托管平台存储边界不同，未运行默认后端 |

Letta 部分仅限所列 v1-sdk legacy 文档的索引，不构成对当前 SDK 的迁移建议。供应方文章中“自动记忆”的具体时机和权限不自动成为本项目策略。

MemoryPolicy、SQL CAS/epoch、审批、召回预算和遗忘屏障是 AgentHub 的目标设计，不是这些项目已经替 AgentHub 实现的保障。未测试 DSH 插件或第三方 Memory 服务，不采用其宣传性性能/准确率数字。

## 11. 审核结论应覆盖什么

- MySQL 为主、Redis 为辅，生产移除 MemoryStore 主入口和 SQLite 镜像路径。
- 取消全部语义规则回退，模型失败显式处理；保留确定性安全校验。
- 复用 ADK，新增真正的 Runtime Harness、MemoryController 和工具 Broker，不只改 prompt。
- 明确长短期记忆存储、调用时点和读写权限；遗忘、版本冲突、过期及后台任务不能绕过控制面。
- 原生只读工具先行，写入/执行需要隔离、审批、幂等与恢复证据。
- 按实施计划分阶段推进；本轮仅文档，不安装组件、不修改代码和配置。
