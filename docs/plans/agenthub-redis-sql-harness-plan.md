# AgentHub v2 实施计划：Redis / MySQL / LLM-only / Tools / Harness / Memory

- 日期：2026-10-29；源码基线：`dev / 8a44f9e`。
- 状态：**P0、P1 已完成；P2 存储层与真实 MySQL 门禁通过（跨服务 accept 待 P3b）；P3a 已完成；P3c 部分完成；P3b、P4–P7 未开始（2026-10-29）**。
- 输入：[架构方案](../architecture/optimization-architecture.md)、[技术选型与来源](../architecture/optimization-technology-selection.md)。
- 本轮只交付这三份文档；不运行迁移、不安装组件、不修改依赖/配置/业务代码/测试、不提交或推送。

## 1. 目标与完成定义

目标不是“把 SQLite 换一个连接字符串”，而是建立：

```text
MySQL 权威状态 + Redis 辅助缓存/事件
+ 无语义规则回退的 LLM 编排
+ 实际工具 + Runtime Harness
+ MemoryController：作用域、读写授权、更新、过期与遗忘
+ 审批/恢复/取消/预算/副作用账本
+ 可重放的前端事件与独立 Eval Harness
```

完成必须同时满足：目标启动图不使用 rule/关键词 fallback；会话重启后可继续；工具确实执行并返回证据；未授权工具不执行；长期记忆可查看、纠正、关闭与遗忘，不能跨作用域泄漏或在恢复后复活；恢复不重置预算或重放未知写操作；两个独立进程的故障/并发测试通过；浏览器主路径与边界场景验证通过。

“配置了 Redis”“有 Harness 目录”“注入了工具名称”“mock 测试通过”均不能单独作为完成证据。

## 2. 实施纪律与依赖

- 每阶段开始前核查 git status，确认允许文件与契约；工作区已有两份方案是本轮文档，不能当作需要清除的杂项。
- P0 先确认契约；后续每个阶段验收后再进入依赖阶段，不一次性重写全仓。
- 不修改 `server/`、根目录 `agents/`、AGENTS.md、用户权限配置或真实 `.env`。
- 不读取真实凭据；需要真实模型或外部检索评测时，由用户在受控环境自行注入，报告只记录配置引用/非敏感元信息。
- 增加 Redis/MySQL/Docker/网络依赖是后续实施范围，不构成本轮立即启动外部服务的授权。
- 不通过新增隐藏 flag 保留 rule fallback；离线验证使用依赖注入的 FakeModel。

```text
P0 契约与接口冻结
 ├─ P1 LLM-only 入口与失败语义
 ├─ P2 MySQL Repository / 执行账本
 └─ P3a Harness / Memory 接口与原生只读工具（可使用 fake store）
P1 + P2 + P3a → P3b checkpoint / 审批 / 预算整合
P2 + P3b → P3c 长短期 Memory 的存储 / 召回 / 更新 / 遗忘
P3c → P4 Redis + gRPC + 事件重放/投影 + Memory 缓存与管理交互
P3c → P5 隔离写工具 + MCP
P4 + P5 → P6 故障/Memory评测/性能/浏览器验收
P6 → P7 部署与数据切换
```

并行条件：先定义接口；共享文件分配单一负责人。P1 的 `planner/`、P2 的持久化实现、P3a 的新 Harness/Tools 可独立推进；对 `main.go`、`server.go`、公共 schema 和 go.mod 的合并统一在整合阶段进行，避免多个代理覆盖同一文件。

## 3. P0：契约与接口冻结

### 目标

把用户新增范围变为明确契约，解决旧文档“允许规则回退、Orchestrator 只返回流、不含 Redis”的冲突。

### 允许范围

- `docs/contracts/intent-orchestration.md`、`planner-input.md`、`orchestration-plan.md`。
- `docs/contracts/project-architecture.md`、`new-architecture-source-of-truth.md`。
- `docs/contracts/data-model.md`、`mysql-schema.md`、`storage-profiles.md`。
- `docs/contracts/gateway-orchestrator*.md` 及对应 schema；公开 API/AG-UI/A2A/Artifact 契约按字段变化逐项确认。
- `docs/contracts/adk-runtime.md`、`agent-config.md`、`security-boundaries.md`。
- `docs/architecture/` 的 ADR/部署拓扑文档；拟新增 Harness、Memory 与工具契约文件。
- 编排实施指引中与新契约冲突的部分，只在路径核查后定点更新；不批量修改所有 Skills。

### 必须冻结

1. SQL 模块表所有权、Redis 非权威职责、幂等键和 run seq。
2. MainAgent 动作 schema、合法 Agent/Tool 集合、显式选人边界、模型失败语义。
3. ToolDescriptor/调用许可、审批绑定、工作区隔离和 unknown 语义。
4. checkpoint 版本、预算计数、run lease/fencing、终态与 artifact 持久性边界。
5. 内部 gRPC 操作、事件 1:N 映射、公开游标以及断连不取消 durable run 的变化。
6. 初始运行预算及超时，包括审批等待与执行时间分离。
7. MySQL/Redis 版本、镜像/依赖锁定和许可证复核责任。
8. MemoryScope/MemoryPolicy、session/长期条目所有权、召回/写入工具与用户管理 API；固定 Token 下的主体隔离限制。
9. Memory 模式默认值、token/条数/字节配额、TTL、候选确认、CAS 冲突、逻辑遗忘与物理清除状态，以及来源/派生物和备份保留边界。
10. 对 DSH、LangGraph、Mem0、Letta 的采用点及已读正文/索引/legacy 证据，不能直接把第三方自动记忆行为当项目默认。

### 门禁

三份设计和 contracts 无矛盾；所有新增字段有生产者、消费者、权限和测试方案。未批准不得进入代码阶段。禁止在此阶段把现有 contract 标记为“已完成实现”。

## 4. P1：清除规则路由，统一 LLM 决策入口

### 允许范围

- `services/orchestrator/planner/`：MainAgent、Parser/Normalizer/Validator/Repairer、旧 Planner 及相关测试。
- `services/orchestrator/cmd/orchestrator/main.go`、`httpapi/server.go`、`httpapi/handler_run_stream.go` 及启动/handler 测试。
- `services/gateway/cmd/gateway/main.go`、Gateway 静态路由启动接线及相关测试。
- 必要模型 capability 接口与契约在 P0 指定后才扩展；不在本阶段改存储/前端。

### 实施顺序

1. 建覆盖默认配置、显式 llm、模型缺失、模型错误、JSON 错误的入口矩阵。
2. 移除 `main_agent.go` 的 fallbackKeywordPlan/scoreAgents/固定 Agent fallback。
3. 删除生产 `rule` / `llm_with_rule_fallback` 模式及缺配置自动退回规则的分支。
4. 统一 MainAgent 的结构化输出和有限模型修复；未知 Agent/策略拒绝，不猜测填默认值。
5. 清理旧 RulePlanner/LLMPlanner 的运行注入、依赖和仅服务旧回退的辅助代码；保留的通用 parser 等不得依赖词表。
6. Gateway 目标启动必须配置 Orchestrator，不通过静态直连 Child Agent 绕过统一编排。
7. 用显式 FakeModel 替换规则型离线测试；Fake 只由测试注入，生产不可因缺模型而启用。

### 门禁

- 真实 handler 入口上：模型失败/无模型时，Child Agent 派发调用次数为 0。
- 代码中仍出现 `RulePlanner` 字样必须解释是历史文档还是遗漏；以启动图和调用断言为准，不能只做字符串计数。
- 模型返回的 Agent/Tool 只来自允许集合；非法输出不能被默认任务补救。
- 所有语义选择来自模型，确定性安全校验保留；不声称仅 `DisableFallback` 就已满足要求。

## 5. P2：MySQL 主存储与执行账本

### 允许范围

- `services/gateway/store/`、`internal/persistence/`、相关 bootstrap/handler 适配和测试。
- `services/orchestrator/` 下拟新增 `internal/persistence/`、执行 Repository、迁移和测试。
- `pkg/runtime/session/` 的 MySQL 接入/测试；不得静默改变完整 SessionService.Get 语义。
- 对应模块 go.mod/go.sum 与仅含占位引用的配置示例；版本变更必须审查。

### 实施顺序

1. 使用 MySQL Repository 实现 Gateway Store，统一会话/用户消息读写，取消 MemoryStore 主入口与 SQLite 镜像的双轨目标设计。
2. 建执行、审批、计划版本、工具账本、预算、checkpoint、event journal/outbox 的最小表集合。
3. 明确唯一约束：requestId、(runId,seq)、(runId,toolCallId)、审批幂等键；考虑工作区作用域。
4. 实现 Gateway request 记录到 Orchestrator accept 的超时重试/查询，两个服务各自事务，不假装跨服务原子。
5. 持久化执行状态与 outbox 同事务；不跨模型/工具调用保持事务。
6. 用 SQL 条件更新和执行版本处理领取/续租；先验证唯一有效提交者，再接 Redis 心跳。
7. 编写合成 SQLite→MySQL 数据转换测试和字段映射；实际用户数据库导入留到 P7，经明确授权执行。
8. 建 session_context_snapshots、memory_entries/revisions/scopes/mutations 及必要唯一约束/索引，明确 active/candidate/revoked 状态、CAS revision、scope epoch、expiresAt 和来源/派生物引用。
9. 提供按作用域/状态/有效期的有界读取、字面/全文检索和候选原子提交接口；此阶段验证存储，不提前声称模型已经完成记忆调用。

### 门禁

- 使用实际隔离 MySQL 实例验证事务/唯一约束/并发；sqlmock 不能替代这一层。
- 建会话→重启服务→读历史→继续聊天，证明不依赖旧 MemoryStore。
- 同 run 并发更新不能分配重复 seq；旧执行版本无法更新终态或预算。
- duplicate requestId 不重复保存用户消息或创建新运行。
- 失败事务不能留下“状态已完成但没有事件”的半结果。
- 修复原 SQLite 路径暴露的跨 run 状态污染、重复内容追加和空消息终态问题；新实现不能重新引入全局 writer。

## 6. P3：Runtime Harness、工具与长短期 Memory

### P3a：可并行的骨架

允许：拟新增 `pkg/runtime/harness/`、`pkg/runtime/tools/`、`pkg/runtime/memory/`，`pkg/runtime/registry/tool.go`、纯 ADK 接口/Runner/Plugin 的必要扩展及对应测试；服务入口暂以 fake store 接线验证。

交付：ContextBuilder、MemoryController/MemoryPolicy、LoopController、BudgetManager、ToolBroker 接口；`workspace.list/read/search`；工具 schema/授权、输出限制和事件转换。

要求：工具不读取凭据路径；父子上下文分离；同一模型/工具循环只有一个执行所有者；优先复用 `adk.Tool/Plugin/Runner`，不复制通用框架。

### P3b：与 P1/P2 整合

允许补充 `services/orchestrator/` Coordinator、审批/取消实现；`services/agents/*` 的统一 Harness 接线及 A2A 元信息适配，仅修改本阶段迁移的 Agent。

交付：执行前 checkpoint、已完成调用结果复用、审批版本、预算持久化、取消、暂停/恢复、SQL fencing 和工具 Broker 远程许可。

### P3c：Memory 存储与调用闭环（必做，不推迟为未来能力）

允许：`pkg/runtime/memory/`、`pruning/`、`session/` 的相关接口/适配；Orchestrator MemoryController、Repository 和管理操作；Gateway 鉴权代理和对应内部 API/契约测试。P4 再完成 gRPC 整合、Redis 缓存与前端交互，不能在 P3c 只建立表就宣布 Memory 完成。

1. 短期：授权 thread/Agent 分支映射、Session 读取/写入、摘要与来源区间、窗口/工具配对、token 预算和恢复再校验。
2. 长期：按 scope/status/expiry 过滤的 MySQL 读取与有界全文检索；模型前自动预召回 + memory.search/read 按需调用，合并去重并共享预算。
3. 新建/更新：memory.propose→来源/敏感性/重复冲突校验→确认→expectedRevision CAS；与审批、幂等 mutation 和 epoch/outbox 同事务生效。
4. 模式：off/session_only/scoped_long_term；读写/管理权限独立，默认 session_only；当前请求和必要运行安全数据不因 off 而丢失。
5. 遗忘：SQL tombstone/epoch→新授权拒绝旧条目→缓存与派生物清理；分别报告 recallDisabled 和 purgeStatus，不把第三方模型或备份清除当作即时保证。
6. 暂不让后台模型自动激活长期记忆；如加入异步整理，必须区分 pending 与 active，并重验 scope epoch、来源删除和审批。

### 门禁

- schema 注册→模型收到 schema→模型提出调用→Broker 授权→工具执行→结果进入下一轮→前端工具事件，不能只测 Registry。
- 只读工具对越界路径、符号链接、超大输出和凭据文件均有负面测试。
- MainAgent/修复/子 Agent/工具重试共用总预算，恢复不归零，并行预留不超额。
- SQL 不可用时不派发需要记账的新副作用；Redis 此时尚非必需。
- 等待审批保存后释放工作资源；重启恢复能校验 revision/hash/过期时间。
- 审批绑定具体 toolCallId 且一次性消费；两个并发派发请求中，只有一个能在当前 fencing 下原子完成审批消费、预算预留和派发意图登记。同参数新调用不能复用旧审批。
- 同一个 toolCallId 的成功结果可复用；缺结果的未知外部执行不得自动视为未执行。
- FakeModel 确定性测试通过与真实模型效果分开报告。

### Memory 专项门禁

- 同 thread 短期记忆可恢复；不同主体、工作区、Agent 私有分支不能串读；相同关键词不能绕过 scope 过滤。
- 两个会话间：session_only 不召回；scoped_long_term 只召回同授权范围已激活记忆；off 不复用旧记忆但保留本次必要工具上下文。
- 候选/accepted 状态不能被 search 或下一轮模型读到；确认成功后的 SQL 真值可立即读取，不要求等待 Redis TTL。
- 两个并发更新使用同一个 expectedRevision 时只有一个成功；另一个得到冲突，不覆盖新事实。
- 遗忘与读取/模型上下文授权交错：以 SQL 事务授权时点断言新读被拒绝，已授权在途调用单独记录；不得把对外已发送内容说成可撤回。
- 过期、权限收窄、删除后旧 Redis 值、旧摘要、旧 checkpoint、延迟候选和重复事件都不能重新激活记忆。
- 摘要/抽取计入总模型预算；固定核心、预召回、memory.read 合并后仍满足条数和 token 限制，工具 call-result 配对保持完整。
- 删除长期记忆不冒充删除聊天/第三方/备份；操作结果区分停止召回、清理中和清理完成。

## 7. P4：Redis、gRPC、事件重放与 Gateway 投影

### 允许范围

- Orchestrator outbox relay、Redis 能力/Memory 缓存与事件适配、内部 gRPC 服务。
- Gateway Orchestrator client、持久化投影、SSE writer、恢复入口及 Memory 管理代理。
- `services/gateway/proto/`、`services/orchestrator/proto/` 的契约/生成产物路径按 P0 核查确定；不能让两套 proto 各自漂移。
- 前端 API client、流事件 reducer、断线恢复与停止按钮；新增范围限定的 Memory 控制界面及测试。
- 模块依赖与协议 schema；仅新增所需 Memory 管理交互，不重写视觉布局或前端总体状态管理。

### 实施顺序

1. SQL commit→outbox→Redis Stream，内部订阅者独立游标；以 SQL seq 作为真值。
2. relay 按 run 保序；先记录 Redis 尾游标 R，再读取 SQL 水位 H 回放，之后从 R 续读；缺号与定期 SQL 水位/终态核验共同负责补齐，包括没有后续事件的尾部缺失。
3. Gateway 投影和消费位置同事务；共享投影去重不抑制独立订阅发送。内部事件到多个公开事件有稳定 subIndex，中段重连重新映射当前 seq、按该订阅游标过滤。
4. 确认消息投影后发送完成事件，保证完成后读历史的一致性；回放水位不充当浏览器消费确认。
5. gRPC 分离 start/subscribe/control；Gateway 不再直连 Child Agent。
6. 前端恢复游标不创建新任务；用户停止按钮发明确 cancel；普通断连只关闭订阅。
7. Redis TTL/trimming、有界队列、写 deadline、SQL 补读和日志过期 snapshot/resync。
8. Memory 缓存键含 scope/epoch/revision，SQL 提交后失效事件只含 ID/版本；即使失效消息丢失，模型调用前仍核验 SQL 权威状态。
9. Memory 管理 UI 展示模式/范围、来源/修订/到期时间、候选确认、纠正、单条/范围遗忘及 purgeStatus；明确区分忘记记忆和删除来源聊天，错误不得显示成成功。

### 门禁

- 两个 Gateway OS 进程订阅同一个 run，各自收到完整事件；即使共享投影已更新，也不跳过另一个订阅的发送。
- relay 在 publish 后确认前崩溃，重试不重复追加文本；序号缺口可恢复。
- 在 SQL 回放切换 Redis 的窗口只产生一条终态、不再产生后续事件，仍可通过交接水位和定期对账取得终态。
- Redis 重启/被清空后，SQL 仍能恢复审批、终态和允许窗口内的事件。
- Gateway 在投影提交前/后崩溃都可恢复；在同 seq 的中间 subIndex 断开，重连不会漏掉剩余片段，也不会重复应用消息投影。
- 慢订阅者不导致无界内存或阻塞其他订阅；断开后游标恢复不重新执行工具。
- 浏览器实际验证断连重连、手动取消、审批、历史回放和 Agent 归属，检查控制台/网络异常。
- 浏览器验证关闭长期记忆、候选确认、并发编辑冲突、单条/范围遗忘、清理失败和来源保留提示；通过模型请求捕获或 fake recorder 证明行为生效，不只检查 UI 文案。
- Redis 缓存旧记忆且删除失效消息被丢弃时，下一次新授权仍不能注入旧版本；SQL 不可用时不得使用无法复核的 Memory 缓存。

## 8. P5：隔离写工具、外部检索与 MCP

### 允许范围

- `pkg/runtime/tools/`、工具 Broker/Harness 策略和相关测试。
- 迁移 Agent 的工具配置和 A2A 适配；对应 Artifact 输出/前端工具结果消费。
- 隔离执行设施及 Docker 配置列入单独批准的本阶段变更清单；不得自动安装第三方 MCP 程序。

### 交付

- `workspace.apply_patch`：独立工作区、文件 hash、冲突检测、批准参数不可变。
- `sandbox.run_tests`：可审计 executable/argv、资源/输出/时间限制、受控网络与挂载。
- `web.search/fetch`：真实后端接线、来源/时间、SSRF/重定向限制；无服务时报告不可用。
- `artifact.publish`：内部归档及引用；单机持久卷和多节点共享存储能力分开声明。
- MCP adapter：注册服务器/工具清单、schema/版本校验、授权与审计；与原生工具经过相同 Broker。

### 门禁

- 未批准的补丁、命令、网络请求实际调用次数为 0。
- 读取工具提示注入不导致写入、读取凭据或修改运行策略。
- 拒绝 private/loopback/metadata 网络目标，并验证重定向与解析变化；不是只检查原始 URL 字符串。
- 同一写调用重放、审批重放、旧 hash、过期授权、未知结果均有断言。
- 崩溃后非幂等工具状态 unknown，不自动生成新调用键重做；支持幂等键的工具单独验证真实去重。
- 在隔离环境验证测试代码超时、异常退出、恶意读取和超大输出；不在宿主机直接运行不可信项目脚本。

## 9. P6：Eval Harness、故障与性能验收

### 允许范围

拟新增项目内评测目录/入口，复用模块现有测试布局；`pkg/runtime` 的观测接口、Gateway/Orchestrator 事件审计和所需测试；前端只改本方案新增交互的缺陷。若新增独立工具目录，先在阶段计划中列出实际路径。

### 测试矩阵

| 层次 | 测什么 | 不能替代什么 |
|---|---|---|
| FakeModel 单元/契约 | 参数、调用计数、预算、权限、终态、映射 | 不能证明真实模型会正确使用工具 |
| Memory 存储/调用/控制 | 同会话恢复、跨会话召回、scope隔离、候选确认、CAS、过期/遗忘、摘要lineage、注入预算 | Memory CRUD通过不等于模型正确召回；相关内容命中不等于已授权 |
| 真模型合成集 | 任务完成、工具/Agent 选择、记忆使用与冲突处理、参数正确性、成本 | 不能证明分布式一致性或无信息泄漏 |
| 实际 MySQL/Redis | 事务、重放、故障、Memory缓存失效、并发领取/更新 | 不能由 SQLite/sqlmock/miniredis 单独替代 |
| 独立进程故障测试 | worker/relay/Gateway 崩溃、接管、旧版本拒绝 | 两个对象实例不等于两个 OS 进程 |
| Race | 内存数据竞争 | 不能证明没有逻辑串流、重复副作用 |
| 浏览器 E2E | 对话/审批/工具/归属/恢复/历史/取消 | 不能由 TypeScript build 替代 |
| 基准与负载 | 首字延迟、队列、DB等待、吞吐、成本 | 不把不同模型或不同资源结果混算 |

固定合成评测集和工具环境；任务结果以实际产物/测试断言判定，模型评审仅补充。真模型/联网评测必须明确数据出域、预算和用户授权，绝不读取 `.env` 获取密钥。

建议负载矩阵：1/10/50 个并发运行、100/1,000/10,000 条历史消息，明确这些是待测规模而非容量承诺。模型/工具耗时固定或分别归因。无进展阈值、预算、队列上限应在实际配置下能触发。

## 10. P7：部署、切换与回滚

### 允许范围

`docker-compose.new-arch.yml`、相关新架构 Dockerfile/启动/健康检查/smoke 脚本、仅含空值或占位符的配置示例、数据迁移工具和交付文档。不顺手清理 legacy 文件或重写 CI。

### 切换前

1. 锁定 Go/依赖/MySQL/Redis/镜像版本，复核许可证和支持状态。
2. 健康检查区分 MySQL 必需依赖、Redis 降级状态、模型配置/能力；不能每次探针发付费模型请求。
3. 在合成/脱敏副本演练导入、主键/归属/终态映射和事件偏移校验。
4. 用户明确授权后才备份/迁移真实数据；不自动读取本地 data 或数据库内容。
5. 停止旧入口接受新写，等待或显式暂停旧 run，迁移并校验后切换；首版不采用无证明的在线双写。
6. 确认 session/checkpoint/Memory/备份的独立保留策略和清理作业；从旧备份恢复后先补齐 Memory 删除标记与最新策略，再开放召回，不能使已遗忘条目复活。
7. 导入聊天历史不自动生成长期记忆；任何候选迁移均保留来源、状态和待确认范围，不能把用户数据升级为永久共享记忆。

### 回滚边界

代码回滚不等于数据库回滚。新 schema 已接受新写后，禁止直接启动旧 SQLite/规则模式继续工作；应暂停入口，使用经验证的向前修复或导出恢复流程。破坏性清表、回退 migration、覆盖备份均需单独确认。

交付验证包括目标启动配置无 rule/SQLite 主路径、MySQL+Redis 可观测、冷启动/恢复/审批/工具/取消/回放 smoke。没有通过跨节点 artifact 存储测试时，只标注单机恢复能力。

## 11. 验证命令与证据纪律

以下模块命令可作为后续实施的基线；依赖变更后离线缓存可能不足，届时报告真实失败并另行批准安装，不绕过验证。

```bash
go test ./pkg/adk/... ./pkg/runtime/... ./services/gateway/... ./services/orchestrator/... -count=1
go test -race ./pkg/adk/... ./pkg/runtime/... ./services/gateway/... ./services/orchestrator/... -count=1
go build ./services/gateway/... ./services/orchestrator/...
npm --prefix frontend test -- --run
npm --prefix frontend run build
git diff --check
git status --short
```

实际 MySQL/Redis、真模型、浏览器和故障测试的脚本尚未编写，不能把上述命令解释为已经包含这些验收。新增专项测试/基准命令前先确认名称存在、确实执行到预期测试；`go test -run` 没匹配到测试但退出 0 不算通过。

每阶段报告分开记录：

1. 用例/断言 PASS、FAIL、SKIP 口径，顶层/子测试不混加。
2. 原始命令及退出码。
3. 环境限制和绕行复验，不能覆盖原命令失败。
4. race、真实服务、独立进程和浏览器证据分别列出。
5. 修复类回归先在缺陷实现失败，再在修复后通过；未咬住的问题如实说明。

只有用户要求时才 commit/push；提交前按仓库要求审查 staged 文件、diff 和 `.env` 跟踪状态，不提交凭据或构建产物。

## 12. 本轮实际交付与下一步

| 项目 | 本轮状态 |
|---|---|
| 现有实现调查 | 已只读核验关键入口、工具、ADK、审批取消和存储边界 |
| 网上优秀项目 | 已查询官方文档；来源和取回限制见技术选型第 10 节 |
| 文档 | 两份重写方案 + 本计划 + P0 契约冻结（含 4 份新契约与 6 份 ADR） |
| P0 | 已完成：规则回退/存储/事件/Memory/Tool 契约与 ADR 已冻结 |
| P1 | 已完成：删除 RulePlanner/LLMPlanner 与 rule 模式；MainAgent LLM-only；Gateway 必须配置 Orchestrator |
| P2 | 存储层完成：MySQL Store/账本/Session/转换测试在真实 MySQL 通过；跨服务 accept 与运行接线待 P3b |
| P3a | 已完成：Harness 预算/状态/上下文、Tool Descriptor/Broker、只读 workspace 工具、MemoryPolicy 接口 |
| P3c | 部分完成：Memory MySQL 存储/候选/确认/CAS/遗忘与 Controller；短期记忆、工具、管理 API/UI、Redis 未完成 |
| 业务代码/测试/配置/依赖 | P0 未修改；P1 已按阶段修改并验证 |
| Redis/MySQL/真模型/浏览器/性能验收 | 未执行，不宣称通过 |
| 上一轮临时回归测试 | 已撤回，不是现存测试套件；不能用其结果替本版架构验收 |

下一步先审核 P0 的契约变更与选型；确认后再按依赖分批实施，不从添加 Redis 依赖或改数据库配置直接开始。
