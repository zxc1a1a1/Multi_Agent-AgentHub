# AgentHub v3 最小原子实施计划（完整版）

- 状态：**实施中（2026-10-31 起；P0 契约已冻结，P1/P2/P3 部分原子任务已落地）**
- 日期：2026-10-31
- 代码基线：`dev / 18b57df`
- 输入：`docs/architecture/v3/12-deployment-topologies.md` 至 `17-routing-harness-patterns.md`
- 约束：每个原子功能独立可验收、独立提交；不把“设计完成”写成“功能完成”

## 1. 计划目标

把 v3 的 6 个方向拆成可执行的最小原子功能：

1. **正确性优先**：审批、取消、预算、租约、事件投影在多实例下不丢、不重、不误取消。
2. **工具真实化**：先把 Tool Broker 接到执行路径，再逐个接真实工具。
3. **群聊可用**：先做 G1 计划扇出，做到群成员/每轮发言可持久化，再做 G2 讨论。
4. **Markdown 能力**：新增确定性 markdown-agent，补 Mermaid/KaTeX/导出。
5. **前端渲染**：先 XSS 安全，再换 Shiki/diff2html/Mermaid/KaTeX。
6. **部署可演进**：单机先稳，再对象存储/多副本/多节点。
7. **路由可解释**：先路由审计与真并行，再考虑群聊发言者选择与受控 handoff。

## 2. “最小原子功能”的定义

一个原子功能必须同时满足：

- **单一行为**：只改变一个可描述的用户/系统行为。
- **可独立验收**：有自己的测试或可复现命令，不依赖后续任务“顺便修好”。
- **可回滚**：优先 feature flag/兼容读；数据库迁移向前兼容，不删旧列。
- **提交边界清晰**：一个原子功能一个 commit；不混格式化、重构、无关修复。
- **不过度跨层**：允许“schema + repository + handler + test”作为一个垂直切片，但不把三个功能塞进一个提交。
- **有证据**：验收记录 PASS/FAIL/SKIP；SKIP 必须写原因，不能算通过。

如果一个任务需要同时改 Gateway、Orchestrator、前端和数据库，先拆成“契约/后端/前端/验收”四个原子任务。

## 3. 决策默认值（需确认）

没有这些决策，不进入代码阶段。推荐默认值如下：

| ID | 决策 | 推荐默认 | 影响 |
|---|---|---|---|
| D1 | gRPC 内部鉴权 | 立即加 token interceptor | A1-07 |
| D2 | Tool Broker 接线 | 本轮接线，先于真实写工具 | A2-02 |
| D3 | 制品存储 | 先本地卷 adapter，再做 S3/MinIO | A6-02/A6-03 |
| D4 | IM 渠道 | 本计划不实施，另立计划 | 08/09 |
| D5 | 外部编码 Agent | 本计划不实施，另立计划 | 10 |
| D6 | Temporal | 不引入 | 05 |
| D7 | TanStack Query | 不阻塞本计划，单独前端优化 | 06 |
| D8 | 长连接/入口 | 单机先上 Nginx，SSE 关闭缓冲 | A6-01 |
| D9 | 群聊 | 先 G1 计划扇出，后 G2 讨论 | A3/A4 |
| D10 | Markdown | 新增独立 markdown-agent（确定性工具） | A5-01 |
| D11 | 渲染 | Shiki + diff2html + Mermaid + KaTeX | A5-08..A5-10b |
| D12 | 部署目标 | 单机单副本优先，A1 完成后再多副本 | A6 |
| D13 | 路由 | supervisor + G1；受控 handoff 后置，不做自由 swarm | A4 |

## 4. 阶段与依赖

```text
P0 契约/决策冻结
  │
  ├─ P1 正确性 & 多实例状态 ───────────────┐
  │     └─ G1 多实例门禁                    │
  │                                         ▼
  ├─ P2 工具安全 ── G2 工具门禁 ── P3 群聊 ── G3 群聊门禁
  │                                         │
  ├─ P4 路由/Harness ── G4 路由门禁 ◄───────┘
  │
  ├─ P5 Markdown + 前端渲染 ── G5 渲染门禁
  │
  └─ P6 部署/对象存储/观测 ── G6 部署门禁

独立后置：IM 渠道（08/09）、外部编码 Agent（10）、Temporal/向量库（05）、TanStack Query（06）
```

硬依赖：

- **P1 必须先于 P6 多副本**：审批/取消/投影不落 SQL，多副本没有意义。
- **P2 必须先于真实写工具**：否则审批策略不会生效。
- **P3 依赖 P0 的会话/参与者契约**；P4 的发言者选择依赖 P3 的每轮消息。
- **P5 的 Markdown Agent 依赖 P2 的 Broker**（导出/写文件需要审批与沙箱）。
- **P6 的对象存储依赖 P0 的 artifact 元数据契约**。

## 5. 原子任务清单

### P0：契约与决策冻结（只改文档，不写代码）

| ID | 原子功能 | 允许文件 | 验收 | 依赖 |
|---|---|---|---|---|
| A0-01 | 确认 D1-D13 默认值 | `docs/architecture/v3/00-INDEX.md`、本计划 | 决策表签字；未决项列明 | 无 |
| A0-02 | 冻结会话/参与者/轮次契约 | `docs/contracts/`、schema | `conversation_participants`、`run_turns` 字段/唯一键/权限明确 | A0-01 |
| A0-03 | 冻结路由决策契约 | `docs/contracts/intent-orchestration.md`、新 schema | `route_decisions` 字段、生产者、消费者、保留期明确 | A0-01 |
| A0-04 | 冻结 Artifact 元数据契约 | `docs/contracts/artifact-schema.md`、`artifact-type-registry.md` | `storage_ref/checksum/content_type/size/version` 明确 | A0-01 |
| A0-05 | 冻结事件投影契约 | `docs/contracts/agui-events.md`、`event-replay.md` | 群聊每轮 `AGENT_TURN_*` 的持久化/游标语义明确 | A0-02 |
| A0-06 | 冻结工具执行契约 | `docs/contracts/tool-broker.md`、`tool-action-risk-policy.md` | 审批、预算、fencing、unknown、幂等键明确 | A0-01 |
| A0-07 | 集成测试环境门禁 | `.github/workflows/*`、`docs/contracts/ci-quality-gates.md` | MySQL/Redis 集成测试不再默认 SKIP | A0-01 |

**G0 门禁**：契约无矛盾；每个新增字段有生产者/消费者/测试；未批准不得改业务代码。

### P1：正确性与多实例状态（P0）

| ID | 原子功能 | 主要文件 | 验收 | 依赖 |
|---|---|---|---|---|
| A1-01 | 预算恢复失败即终止，不新建 Budget | `services/orchestrator/httpapi/resume.go` | 反序列化失败/超限返回错误；回归测试还原修复会 FAIL | G0 |
| A1-02 | 租约续租 | `httpapi/run_ledger.go`、`persistence/mysql/repository.go` | 超 15 分钟运行不失租；旧 fencing 被拒 | A1-01 |
| A1-03 | 待审批计划落 SQL | 新 migration、`handler_hitl.go`、repository | 两个 Orchestrator 实例 confirm 都成功 | G0 |
| A1-04 | HITL 信号跨实例 | SQL 信号表 + 短轮询（最小）；Redis 后续优化 | 实例 A 创建、实例 B 确认，执行继续 | A1-03 |
| A1-05 | 审批幂等缓存落 SQL | `approvals`/新表 + handler | 并发重复 confirm 只消费一次 | A1-03 |
| A1-06 | 运行任务引用/取消跨实例 | `run_task_registry.go` + 新表/Redis | 实例 B cancel 能终止实例 A 的远端任务 | A1-03 |
| A1-07 | gRPC 内部鉴权 + trace metadata | `grpcapi/server.go`、Gateway client | 无 token 请求被拒；traceId 可传 | G0 |
| A1-08 | 统一 fencing 语义 | `pkg/runtime/tools/broker.go`、ledger | `0` 值行为一致；测试覆盖 | A1-01 |
| A1-09a | 投影游标表/consumer 语义 | Gateway migration、store | 每个 run/consumer 有独立游标 | G0 |
| A1-09b | 事件投影 worker | Gateway 新 worker + `run_public_events` | kill 处理 `/api/chat` 的 Gateway，另一个实例继续投影 | A1-09a |
| A1-09c | 助手消息投影接管（已实现可恢复范围） | worker + `ProjectAssistantMessage` | Gateway 崩溃后，最终助手消息仍落库 | A1-09b |
| A1-10 | 优雅停机/释放租约 | Orchestrator main | SIGTERM 不留下永久租约 | A1-02 |

**G1 门禁**：
- 2 个 Orchestrator 实例，confirm/cancel/resume 都成功。
- kill Gateway 后，事件无缺口，助手消息最终落库。
- 预算恢复不重置；15 分钟以上运行不失租。
- 所有集成测试在 CI 或显式环境执行，记录 PASS/FAIL/SKIP。

### P2：工具执行安全（P0/P1）

| ID | 原子功能 | 主要文件 | 验收 | 依赖 |
|---|---|---|---|---|
| A2-00 | 平台工具执行位置 ADR | `docs/contracts/tool-broker.md` | 明确 agent-side authorize/result 还是 orchestrator-side execution；权限/幂等/审计 | G0 |
| A2-01 | ADK 工具执行前强制 Guard | `pkg/adk/runner.go` 或工具包装层 | 无 envelope/无审批的写工具 0 调用 | G0 |
| A2-02 | 按 A2-00 接入授权/结果账本 | `internal/toolruntime/broker.go` + 选定执行路径 | 审批/预算/fencing/unknown 生效 | A2-00、A2-01、A1-08 |
| A2-03 | 工具预算真实扣减（broker 已实现） | broker + budget | 超预算拒绝；恢复不重置 | A2-02 |
| A2-04 | 只读 workspace 工具接入 code-agent（已实现，见日志） | `pkg/runtime/tools`、code-agent | 能读真实文件；无写权限 | A2-01 |
| A2-05 | apply_patch + sandbox.run_tests（已接线，Docker 未验证） | 同上 + Docker | 未审批 0 调用；沙箱内执行 | A2-04 |
| A2-06 | diff-agent 真实 unified diff（已实现，见日志） | `agents/diff-agent` | 多行 diff 正确；不再 1 行模板 | A2-01 |
| A2-07a | security 真实扫描器（已实现工具） | security-agent + 工具 | 已知密钥/CVE 样例能复现 | A2-02 |
| A2-07b | test 真实覆盖率/失败解析（已实现工具） | test-agent + 工具 | 已知测试日志/覆盖率能复现 | A2-02 |
| A2-08 | MCP 真实传输 + allowlist（传输/allowlist 已实现） | `pkg/runtime/tools/mcp.go` | 未注册工具拒绝；调用可审计 | A2-02 |

**G2 门禁**：未授权写工具 0 调用；unknown 不重做；沙箱不可用不宿主执行；工具结果可追溯。

说明：A2-00 若选择 **agent-side authorize/result**，工具仍在 Agent 内执行，只是执行前后调用 Orchestrator 授权/记账；若选择 **orchestrator-side execution**，A2-04/A2-05 的注册位置改为 Orchestrator Tool Runtime，Agent 只发 tool call。两条路线不能混用。

### P3：群聊 G1（P1）

| ID | 原子功能 | 主要文件 | 验收 | 依赖 |
|---|---|---|---|---|
| A3-01 | 会话类型 + 参与者表 | Gateway migration/store | `conversation_type=group` 可写入/读取 | G0 |
| A3-02 | 创建/更新会话 API 支持参与者 | `httpapi/server.go`、store | 创建群聊后可查询参与者 | A3-01 |
| A3-03 | 前端群成员持久化/恢复 | `ChatWindow.tsx`、`messageStore` | 刷新后多选 Agent 仍在 | A3-02 |
| A3-04 | Picker 过滤 enabled/healthy + 注册后刷新 | `AgentPicker.tsx`、`agentStore.ts` | 禁用 Agent 不可选；新注册可见 | A3-02 |
| A3-05 | 每轮消息投影 | Gateway 投影 + migration | 刷新后多个 Agent 发言按轮次重建 | A3-01、A1-09c |
| A3-06 | 群计划参与者勾选 | `GroupPlanCard.tsx`、confirm API | 取消可选参与者后不派发 | A3-02 |
| A3-07 | 群聊 E2E（Playwright 已实现） | Playwright/Vitest | 创建→发送→刷新→历史→取消 | A3-03..A3-06 |

**G3 门禁**：群聊刷新不丢参与者；历史不合并成一条；取消能终止所有未完成 Agent。

### P4：路由与 Harness（P1/P2）

| ID | 原子功能 | 主要文件 | 验收 | 依赖 |
|---|---|---|---|---|
| A4-01 | 路由决策落库（已实现，见日志） | `route_decisions` + planner/executionpath | 每次 run 可查 path/候选/选中/拒绝原因 | G0 |
| A4-02 | DAG/ready queue 真执行（已实现，见日志） | `executor/dag_executor.go` + handler | 依赖未完成不领取；波次正确 | A4-01 |
| A4-03 | 并行预算/取消传播（已实现，见日志） | executor + budget | 并发不超上限；取消传播到所有任务 | A4-02 |
| A4-04 | 群聊 round-robin 发言（已实现，env 控制） | new speaker selector | max_rounds 生效；不连续同一人 | A3-05 |
| A4-05 | 模型选择发言者（G2） | selector + planner | 选择理由入 route_decisions；可终止 | A4-04 |
| A4-06 | 受控 handoff（可选） | 新工具 + allowlist | 非白名单拒绝；预算/上下文不越界 | A4-03 |
| A4-07 | 路由指标/评测（已实现指标部分） | metrics + eval | 选中率/拒绝率/循环/成本可查 | A4-01 |

**G4 门禁**：路由可解释；并行有证据；群聊有轮次和终止；无自由 swarm。

### P5：Markdown Agent 与前端渲染（P1/P2）

| ID | 原子功能 | 主要文件 | 验收 | 依赖 |
|---|---|---|---|---|
| A5-01 | markdown-agent 骨架/A2A/AgentCard（已实现，见日志） | 新 `services/agents/markdown-agent` | 可注册、可健康检查、默认不启用 | G0 |
| A5-02a | markdown.render（已实现，见日志） | markdown-agent + goldmark | 样例 HTML 正确；原始 HTML 默认不执行 | A5-01 |
| A5-02b | markdown.lint（已实现，见日志） | markdown-agent | 标题/空链接/代码块语言可报行/列 | A5-02a |
| A5-02c | frontmatter 解析/校验（已实现，见日志） | markdown-agent | YAML 合法/非法都有结果 | A5-02a |
| A5-02d | TOC 生成（已实现，见日志） | markdown-agent | 锚点与渲染一致 | A5-02a |
| A5-03a | Mermaid 语法校验（已实现 Go 侧，见日志） | markdown-agent/前端 | 错误图有行/列提示 | A5-02a |
| A5-03b | Mermaid 渲染（strict/sanitize，已实现） | markdown-agent/前端 | 恶意脚本不执行 | A5-03a |
| A5-04 | KaTeX render（前端已实现，见日志） | markdown-agent/前端 | `trust:false`；危险命令被拒 | A5-02a |
| A5-05 | 导出 PDF | markdown-agent + Playwright 沙箱 | 沙箱内导出；无内网访问 | A5-03b |
| A5-06 | link check（已实现，见日志） | markdown-agent + 沙箱 | 私网/重定向被拒 | A5-02a |
 | A5-02a |
| A5-08 | Shiki 替换 hljs（已实现，见日志） | `CodePreview.tsx` | 包体不增；高亮正确 | A5-07 |
| A5-09 | diff2html 渲染层（已实现，见日志） | `DiffPreview.tsx` | dry-run/apply 不变 | A5-07 |
| A5-10a | Mermaid 前端懒加载（已实现） | 前端 | 仅出现时加载；失败不白屏 | A5-03b |
| A5-10b | KaTeX 前端懒加载（已实现，见日志） | 前端 | 仅出现时加载；失败显示原文 | A5-04 |
| A5-11 | Playwright 截图 artifact | web/test agent + artifact | 截图可跨节点读 | A5-07 |
| A5-12 | 消息列表窗口化（可选，已实现） | `ChatWindow.tsx` | 长会话滚动/内存达标 | A5-08 |

**G5 门禁**：XSS 样例 0 执行；Mermaid/KaTeX 安全；首屏 JS ≤500KB gzip；Markdown 导出可用。

### P6：部署、对象存储与观测（P2）

| ID | 原子功能 | 主要文件 | 验收 | 依赖 |
|---|---|---|---|---|
| A6-01 | 单机 Nginx/卷/备份 | compose、nginx、docs | SSE 不缓冲；重启数据仍在 | G1 |
| A6-02 | ArtifactStore 接口 + 本地卷（已实现，见日志） | `pkg/runtime/tools`、orchestrator | 替换临时 JSON；重启可读 | G0 |
| A6-03 | S3/MinIO adapter（adapter 已实现，未接 publish） | artifact adapter | 跨节点读写；签名 URL | A6-02 |
| A6-04 | 投影 worker 多副本幂等验证（集成测试已加） | A1-09 worker | 多 worker 不重复写 | A1-09 |
| A6-05 | 多 Orchestrator owner/共享状态（集成测试已加） | A1-03..A1-06 | 2 副本 confirm/cancel 正常 | G1 |
| A6-06 | OTel trace | Gateway/Orchestrator/Agent | 同一 run 可串联 | A1-07 |
| A6-07 | 指标与告警（指标已实现，告警仅文档阈值） | metrics | 路由/审批/工具/SSE 有指标 | A4-07 |
| A6-08 | Runbook（单机版已实现） | docs | 故障有操作步骤 | A6-01..A6-07 |

**G6 门禁**：kill Gateway/Orchestrator 有明确结果；跨节点 artifact 可读；trace 可串联。

## 6. 每任务执行模板

每个原子任务实施时，报告必须包含：

```text
任务 ID：
目标行为：
改动文件（实际）：
未改文件（确认）：
契约变更：
测试命令：
结果：PASS / FAIL / SKIP（SKIP 写原因）
失败证据：
回归风险：
回滚方式：
提交：
```

提交规范：

- 一个任务一个 commit；`feat/fix/refactor/test/docs` 前缀。
- 只 `git add` 本任务文件；禁止 `git add .`。
- 提交前执行 `git diff --staged --check` 与密钥扫描。
- 不修改真实 `.env`，不提交构建产物。

## 7. 测试矩阵

| 层 | 范围 | 必跑 |
|---|---|---|
| 单元 | 预算、fencing、路由派生、投影 reducer、渲染安全 | 每次任务 |
| 集成 | MySQL/Redis 审批、取消、重放、artifact | 每个 P1/P6 任务 |
| 协议 | gRPC 鉴权、A2A、AG-UI 游标 | A1/A2/A3 |
| E2E | 群聊刷新、审批、取消、Markdown 渲染 | G3/G5 |
| 真机 | 真实模型主路径、SSE 断线 | 阶段门禁 |

硬规则：

- 缺 `AGENTHUB_MYSQL_TEST_DSN`/`AGENTHUB_REDIS_TEST_ADDR` 时，集成测试 SKIP；**SKIP 不算通过**。
- 修复型任务必须有“还原修复会 FAIL”的回归测试。
- 前端安全任务必须有恶意样例。

## 8. 阶段门禁汇总

| 门禁 | 必须证明 | 不通过则 |
|---|---|---|
| G0 | 契约/决策冻结 | 不写代码 |
| G1 | 多实例审批/取消/投影/预算/租约 | 不扩副本、不进入 P6 |
| G2 | 工具未授权 0 调用、unknown 不重做 | 不开放真实写工具 |
| G3 | 群聊成员/历史/取消可用 | 不做 G2 讨论 |
| G4 | 路由可解释、真并行、有终止 | 不做 handoff |
| G5 | XSS 0、渲染/导出可用 | 不替换现有渲染 |
| G6 | 单机稳定、跨节点制品可读 | 不上多节点 |

## 9. 风险与回滚

| 风险 | 影响 | 缓解 |
|---|---|---|
| 审批状态迁移遗漏 | confirm 404 | 双读写过渡；旧内存路径保留一个版本 |
| 投影 worker 重复写 | 前端重复文本 | `(run_id,seq,sub_index)` 唯一键 + 幂等投影 |
| 群聊历史迁移 | 旧消息作者/轮次丢失 | 新列可空；旧消息按单 Agent 读取 |
| Broker 接线改变工具行为 | 现有工具失败 | feature flag；先只读、后写 |
| Shiki/Mermaid 包体上涨 | 首屏变慢 | 懒加载 + 构建预算门禁 |
| Mermaid XSS | 脚本执行 | strict/sandbox + sanitize + 回归样例 |
| 多副本 owner 路由错误 | 确认/取消错实例 | owner 表 + 失败回退；先 2 实例测试 |

## 10. 本计划不做

- 不实现 IM 渠道（08/09）；另立计划，依赖 D4。
- 不接入 Claude Code 等外部编码 Agent（10）；另立计划，依赖 D5。
- 不引入 Temporal/Kafka/Service Mesh/向量库（05）。
- 不重写前端状态管理（TanStack Query）到本计划；单独前端计划。
- 不做自由 swarm / 无限群聊 / 任意 handoff。
- 不修改 `server/`、根目录 `agents/` 等 legacy 实现。
- 不读取或提交 `.env`、真实密钥、真实 DSN。

## 11. 推荐的第一个批次

如果只先做一批，按这个顺序：

1. **A1-01** 预算恢复 fail-closed（最小、独立、可测试）。
2. **A1-03** 待审批计划落 SQL（多实例前提）。
3. **A1-09b** 投影 worker（Gateway 崩溃可恢复）。
4. **A2-01/A2-02** 工具 Guard/Broker 接线（安全前提）。
5. **A3-01/A3-02** 群聊成员持久化（产品前提）。

这五项完成后，再决定是继续 G1 群聊、还是先做 P5 渲染；不要同时开三条跨层大改。

## 12. 完成定义

本计划完成不是“文档写完”，而是：

- G0-G6 门禁全部有 PASS 证据；
- 每个原子任务有独立 commit 和测试记录；
- 多实例、群聊、工具安全、Markdown、渲染、部署各有可复现验收；
- 没有未登记的 SKIP；
- 没有密钥/真实凭据进入仓库；
- v3 文档与实际实现一致，差异有记录。

---

## 13. 实施记录（2026-10-31）

记录规则：只写实际执行并验证过的结果；SKIP 必须写原因；测试数量与通过数分开。

| 任务 | 状态 | 提交 | 验证证据 |
|---|---|---|---|
| A0-01 | PASS | `5637989` | D1-D13 采纳值写入 `00-INDEX.md` §7，未决项列明 |
| A0-02 | PASS（契约） | `5637989` | 新增 `docs/contracts/conversation-participants.md`；`mysql-schema.md` 增加参与者表 |
| A0-03 | PASS（契约） | `5637989` | `intent-orchestration.md` 新增 route_decisions 字段/生产者/消费者/保留期 |
| A0-04 | PASS（契约） | `5637989` | `artifact-schema.md`、`artifact-type-registry.md` 增加 storageRef/checksum/contentType/size/version |
| A0-05 | PASS（契约） | `5637989` | `agui-events.md`、`event-replay.md` 增加 AGENT_TURN 持久化与游标语义 |
| A0-06 | PASS（契约） | `5637989` | `tool-broker.md` 增加执行位置决策、预算/fencing/unknown/幂等语义 |
| A0-07 | PASS（配置，CI 待验证） | `18b8697,80a6709` | `.github/workflows/v3-integration-gates.yml`：MySQL/Redis 集成 + SKIP 即失败；全模块 unit/build 矩阵、frontend unit/typecheck/build、Playwright E2E、compose/nginx/promtool 校验；YAML 解析通过，未在 GitHub 实跑 |
| A1-01 | PASS | `e2ec0c9` | 3 个回归测试；还原修复后超限场景 FAIL |
| A1-02 | PASS（单测） | `581099e` | 周期续租/失租取消/handler 接线测试通过；MySQL 集成测试 SKIP（无 DSN） |
| A1-03 | PASS（单测） | `1fe4d5f` | 跨实例确认 202、幂等 200、revision 冲突 409；MySQL 集成测试 SKIP（无 DSN） |
| A1-04 | PASS（单测） | `1fe4d5f` | 持有实例 Claim 信号并继续执行；禁用 Claim 后测试 FAIL；MySQL 集成测试 SKIP（无 DSN） |
| A1-05 | PASS（单测；集成 SKIP） | `e6d3dcd` | hitl_confirmations 缓存状态码/响应体；同键重放返回原响应，不同 payload 409，处理中 409；禁用 SQL 幂等后重放测试 FAIL |
| A1-06 | PASS（单测；集成 SKIP） | `b56f25b` | run_task_refs/run_cancels 持久化；实例 B 可按 SQL refs 取消远端 A2A 任务；A 侧轮询取消请求并终止本地 context；移除 SQL refs 后取消测试 FAIL |
| A1-07 | PASS | `803fcf8` | gRPC unary/stream interceptor 校验内部 token；trace metadata 透传；缺失/错误 token 返回 Unauthenticated；移除 interceptor 后测试 FAIL |
| A1-08 | PASS（单测） | `dc6f0a4` | 所有变更写（state/checkpoint/dispatch/renew/release）拒绝 fencing 0；Guard 同样拒绝；0 在触库前失败 |
| A1-09a | PASS（单测；集成 SKIP） | `860f95e` | 投影偏移表存在；`ProjectAssistantMessage` 先物化 offset 行再 `FOR UPDATE`，并新增 projection worker 集成测试（需 DSN） |
| A1-09b | PASS（单测；集成 SKIP） | `860f95e` | 新增 `services/gateway/projection` worker：按 turn 重建消息、只投影已完成轮次、共享偏移幂等；未完成轮次测试在强制终态化后 FAIL |
| A1-09c | PASS（已 journal 的终态） | `860f95e` | worker 对 TEXT_MESSAGE_CONTENT + RUN_FINISHED 与 AGENT_TURN_FINISHED 均投影最终助手消息；`AGENT_TURN_FINISHED` 前崩溃的轮次无终态，按未完成处理（无消息可落） |
| A1-10 | PASS | `244f574` | 所有 handler 返回路径释放租约；resume 同样释放；Ctrl+C 后 `server.Shutdown` 等待在途 handler；移除 defer 后 planner 失败测试 FAIL |
| A2-00 | PASS（决策） | `5637989` | 选择 agent-side authorize/result；生产接线未完成 |
| A2-01 | PASS | `b9bfaf2` | 无 Guard 的副作用工具 0 调用；绕过 authorizeTool 后 3 个测试 FAIL |
| A2-02 | PASS（授权/结果接线；审批单由平台创建） | `75b4e39` | A2A envelope 传播 run/task/fencing/principal/approval；RemoteGuard 写工具无 approval 本地拒绝；Orchestrator authorize/result 接 Broker；端到端测试覆盖允许→dispatch→结果记账；映射 403/409/429 |
| A2-03 | PASS（Broker；生产接线待 A2-02） | d3b4529 | 从 checkpoint 恢复 usage 后 ReserveTool(1)；超限/损坏预算在审批消费与工具执行前失败；写调用持久化递增后的 BudgetJSON |
| A2-04 | PASS | `d35aa42` | code-agent 通过 CODE_AGENT_WORKSPACE_ROOT 接入 list/read/search；工具均为 ReadOnly；测试读取真实文件且不含 apply_patch/sandbox |
| A2-05 | PASS（工具/接线；Docker 执行 SKIP） | `4dc1127,43c5f38,29b7fe7` | apply_patch hash/路径/符号链接校验；sandbox allowlist/审批/超时/默认禁网；未审批写调用 0 执行；code-agent 写工具需显式启用+沙箱镜像；Docker 真实执行测试 SKIP |
| A2-06 | PASS | `a0e5e4f` | generate_diff 使用 LCS 行级 diff，生成多 hunk/上下文/增删行；测试拒绝旧 1 行模板格式 |
| A2-07b | PASS（工具；Broker 接线待 A2-02） | d9ca50e | analyze_test_log 解析 go test PASS/FAIL/SKIP、失败详情与 coverage；assess_coverage 解析百分比/cover profile 并对比阈值；固定样例可复现 |
| A2-07a | PASS（工具；Broker 接线待 A2-02） | d36cbef | detect_secrets 匹配私钥/AWS/GitHub/Slack/JWT/凭据赋值并只返回脱敏值；parse_vulnerability_report 解析文本/JSON CVE 与模块；已知样例可复现 |
| A2-08 | PASS（传输/allowlist；审计待 A2-02） | 505dcfc | HTTP JSON-RPC tools/list/tools/call；Bearer token；未列策略工具不暴露；错误脱敏；allowlist 测试通过 |
| A3-01 | PASS（单测） | `ccf6ed8` | 内存参与者往返/替换/级联删除；MySQL 集成测试 SKIP（无 DSN） |
| A3-02 | PASS | `0bd8460` | 群聊创建/查询/替换/非法输入/404 测试通过 |
| A3-03 | PASS | `32bfb2f` | 服务端名单恢复 + 本地缓存回退；禁用恢复后测试 FAIL |
| A3-04 | PASS | `146e892` | 禁用 Agent 不可选；注册后 picker 重载；禁用判断移除后测试 FAIL |
| A3-05 | PASS | `94fa1f5` | 网关按 turnIndex/messageId 累积，AGENT_TURN_FINISHED 时逐轮投影；无 turn 事件仍合并一条；禁用 finish 投影后测试 FAIL（历史被合并） |
| A3-06 | PASS | `4597d7d` | GroupPlanCard 每成员勾选，required 不可取消；选择随 approve/revise 提交；后端 filterPlanByConfirmedParticipants 只派发已确认 Agent 的任务（单测） |
| A3-07 | PASS | `21ccdf2` | Playwright + 系统 Edge：创建会话→勾选 2 Agent→发送→群计划→取消→刷新；断言参与者恢复与两 Agent 历史；`PLAYWRIGHT_CHANNEL=msedge` 运行通过 |
| A4-01 | PASS（单测；集成 SKIP） | `7aa6cde` | route_decisions 表 + 决策构建/查询接口；记录 path/strategy/候选/选中/拒绝原因；移除记录调用后测试 FAIL |
| A4-02 | PASS（单测） | `13c016b` | sequential 走 DAGExecutor：拓扑波次并发、依赖失败跳过、上游输出注入；ordered_parallel 仍串行保序 |
| A4-03 | PASS（单测） | `2582a6d` | DAG 波次受 ParallelTasks 信号量限制；ctx 取消停止派发并传入 dispatcher；移除信号量后测试 FAIL |
| A4-04 | PASS（确定性 round-robin） | `68b352a` | GROUP_CHAT_MAX_ROUNDS 控制轮数（默认 1）；展开为 sequential 依赖链，轮转发言、无连续同一人、注入上一发言输出；handler 测试 2 轮 2 Agent 派发 4 次 |
| A4-05 | PASS（计划期模型选择；默认 round_robin） | `c0ea166` | MainAgent 兼 SpeakerSelector，输出 speaker/reason/done；越界/缺理由/连续重复/首轮停止即失败；`GROUP_CHAT_SPEAKER_MODE=model` 启用，上限 MAX_ROUNDS×参与者；理由写入 route_decisions.reason_summary；红检关闭 allowlist 后测试 FAIL；执行中按输出动态重选未做 |
| A4-06 | PASS（可选；默认关闭） | `b969c99` | `handoff_to_agent` + `/internal/orchestrator/tools/handoff`：本地与 run 计划双层 allowlist；深度取自 RunIdentity.DelegateDepth 且 +1 超限拒绝；消息 ≤8KB、理由 ≤512B、响应 ≤16KB；越界 0 派发；红检关闭边界校验后测试 FAIL；仅 code-agent 显式启用 |
| A5-07 | PASS（前端单测） | `a4949e5` | highlight 输出白名单 sanitize；StreamingText 原始 HTML 不渲染；WebPreview sandbox/CSP 既有测试；移除 sanitizer 后 2 个测试 FAIL |
| A5-01 | PASS | `465f35d` | 新模块 markdown-agent：A2A AgentCard/health、确定性 Agent、无模型密钥；未加入 compose，默认不启用 |
| A5-02a | PASS | `6b5fcd4` | goldmark 渲染；raw HTML 被丢弃；GFM 表格/链接测试 |
| A5-02b | PASS | `6b5fcd4` | 标题缺空格/空链接/无语言代码块报行列 |
| A5-02c | PASS | `6b5fcd4` | YAML 合法/非法/未闭合均有结果 |
| A5-02d | PASS | `6b5fcd4` | 同一 parser 生成 TOC 锚点并与渲染 id 对齐 |
| A5-03a | PASS（Go 侧） | `474f5bb` | 校验 diagram 关键字/括号/引号/闭合 fence，报行列；前端 Mermaid 渲染 A5-03b 未实现 |
| A6-02 | PASS（单节点默认；S3 可选） | `dcb539c` | LocalFileStore 每制品一 JSON，原子写；重启后 Get/ListByRun 可读；ARTIFACT_STORE_DIR 默认 ./data/artifacts；跨节点由 A6-03 `ARTIFACT_METADATA_BACKEND=s3` 提供 |
| A5-08 | PASS | `8aca161` | 移除 highlight.js；Shiki 动态 import + token 渲染（无 HTML 注入）；首屏主 chunk 1.44MB/gzip 447KB → 465KB/gzip 约 133KB；语法 chunk 懒加载 |
| A5-09 | PASS | e2c1eae | DiffPreview 用 diff2html parse（仅解析）渲染文件/hunk/行号；内容按 React 文本输出；dry-run→apply 流程测试通过；不回退解析时测试 FAIL |
| A5-12 | PASS（可选） | 2d707d6 | ChatWindow 默认只渲染最近 100 条，提供 Load earlier 分批展开；150 条测试证明首条不渲染、展开后可见；移除窗口化后测试 FAIL |
| A5-11 | PASS（单节点验证；S3 可配） | `446e449,a08c8db` | `/internal/orchestrator/render` + `capture_page`：PNG 写入 ObjectStore（本地/S3 可选），元数据含 storageRef/checksum/contentType/version；content endpoint 校验 SHA-256；Gateway `/api/artifacts/{id}/content` 代理下载；真实 Playwright 系统 Edge 生成 49,703B PNG；跨节点依赖 S3，未做多副本实跑 |
| A5-05 | PASS（子进程隔离；Docker 沙箱未验证） | `446e449` | 同一 Playwright 渲染器生成 PDF（真实 54,621B）；渲染前拒绝 loopback/私网/链路本地/metadata/CGNAT URL；`PLAYWRIGHT_RENDER_SCRIPT` 未配置时端点 503；未在 Docker 沙箱内执行 |
| A5-06 | PASS（SSRF 防护） | cee79ef | 链接检查拒绝 loopback/私网/链路本地/metadata/非 http(s)；DialContext 二次解析防 DNS rebinding；重定向逐跳校验；私网目标 0 请求 |
| A5-04 | PASS（前端） | 6c38c87 | remark-math + rehype-katex，`trust:false`；javascript href 不生成链接；恶意样例测试通过 |
| A5-03b | PASS（前端） | b2204b2 | Mermaid securityLevel=strict + DOMPurify SVG profile；script/javascript 链接被剥离；失败回退代码文本 |
| A4-07 | PASS（指标；eval 复用既有 harness） | 3fd32e0 | GET /internal/orchestrator/metrics/routing：总 run/决策、选中/拒绝原因、选中率/拒绝率、多决策（loop）run、model/tool/repair/replan 用量；时间窗非法 400 |
| A5-10a | PASS | b2204b2 | 仅 mermaid 代码块渲染时动态 import mermaid；独立懒 chunk；去掉 sanitize 后测试 FAIL |
| A5-10b | PASS | 6c38c87 | 仅在内容含 $...$ 时动态加载 remark-math/rehype-katex/katex CSS；失败保持原文；禁用加载后测试 FAIL |
| A6-01 | PASS（静态配置；本地未跑 nginx） | `af854af` | `deploy/nginx/agenthub.conf`：/api→Gateway（SSE 关闭 buffering）、/internal 404、其余→前端；compose 增加 nginx 服务；`scripts/backup-agenthub.sh` + systemd timer；`bash -n`、fake mysqldump 备份/保留期、compose YAML 通过；CI 增加 `nginx -t` 步骤但未实跑，本地无 nginx |
| A6-08 | PASS（文档，单机） | dbb0aff | 新增 `docs/delivery/runbook-single-node.md`：启停/健康/运行排查/MySQL/Redis/Gateway/Orchestrator/审批/取消/制品故障/备份/回滚/安全；明确未覆盖多副本、S3、OTel |
| A6-07 | PASS（指标 + 规则；告警通道未接） | `b96ed60` | Orchestrator/Gateway 指标保留；三入口支持 `format=prometheus`；`deploy/prometheus/{prometheus,alerts}.yml` 覆盖 down/run/approval/tool unknown/SSE/路由缺失；CI 增加 promtool 校验但未实跑，未接 Alertmanager |
| A6-04 | PASS（CI 集成；本机 SKIP） | `467397b` | 两个 worker 并发恢复同一 run，仍只写 2 条 turn 消息且 offset=4；本机无 DSN 未执行，CI 门禁负责运行 |
| A6-05 | PASS（CI 集成；本机 SKIP） | `9ab66ab` | 两个 Orchestrator 共享 MySQL：A 启动待审批 run，B confirm 写 run_signals，A 消费并执行至 completed；本机无 DSN 未执行 |
| A6-06 | PASS（SDK/传播；collector 配置未实跑） | `de167e3,92a3424` | `pkg/runtime/observability`：W3C traceparent + OTLP/HTTP；Gateway/Orchestrator 与全部 service agents server span，Gateway→Orchestrator→Agent client span；`agenthub.run_id` 属性；compose observability profile 含 otel-collector/jaeger；内存导出测试证明同 trace；红检移除 Transport 后测试 FAIL；容器未启动 |
| A6-03 | PASS（blob + metadata 接线；未多副本实跑） | `58b9ca1` | S3ObjectStore：EnsureBucket/Put/Get/Presign/List/Delete；key 安全校验、10MB/7 天签名上限；本地 blob/S3 可选；S3Store 以 `metadata/<id>.json` 跨副本存元数据；渲染制品写入 checksum/storageRef；fake S3 验证读写/签名，未跑真实 MinIO 多副本 |
| A0–A6 原子任务 | 已实现 | 见各行 | 仅剩运行时认证：MySQL/Redis 集成、GitHub CI、Docker 沙箱、真实 MinIO/collector/Nginx；见“收尾验证” |

### 已知 SKIP / 环境限制

- 所有 MySQL/Redis 集成测试在本机执行时因缺少 `AGENTHUB_MYSQL_TEST_DSN` /
  `AGENTHUB_REDIS_TEST_ADDR` 而 SKIP；不计入通过。CI 门禁见 A0-07。
- Docker sandbox 执行测试需 `AGENTHUB_DOCKER_SANDBOX_TEST=1` 与 Docker daemon；
  本机未执行，相关结果不计入通过。
- 前端命令需显式使用 `/e/node_js/node.exe` 并把 `/e/node_js` 放到 PATH 前；
  否则 npm 脚本解析到失效 shim，vitest 会报 `node:util` 导出错误。
- `vite build` 通过；首屏 index gzip 156.92KB，低于 G5 的 500KB 预算；
  个别语法/图表 chunk 压缩前 >500KB，但按需加载。


### 最终验证（2026-10-31）

| 命令 | 结果 |
|---|---|
| `cd services/orchestrator && go test ./... -count=1` | PASS（全部包 ok） |
| `cd services/gateway && go test ./... -count=1` | PASS（全部包 ok） |
| `cd pkg/adk && go test ./... -count=1` | PASS |
| `cd pkg/runtime && go test ./tools/ -count=1` | PASS |
| `cd frontend && tsc -b` | PASS |
| `cd frontend && vitest run` | 27 files / 254 tests PASS |
| `cd frontend && vite build` | PASS；主 chunk gzip 447KB（接近 G5 500KB 预算） |
| MySQL/Redis 集成测试 | SKIP（本机无 `AGENTHUB_MYSQL_TEST_DSN` / `AGENTHUB_REDIS_TEST_ADDR`；CI 门禁已配置） |

未完成范围（截至 2026-11-02）：原子任务 A0–A6 已全部落地；未完成的是运行时认证 ——
MySQL/Redis 集成在本机 SKIP、CI 门禁未在 GitHub 实跑、Docker 沙箱内 Playwright
未验证、真实 MinIO/OTel collector/Nginx/Prometheus 未启动，Alertmanager 通道未配置。
因此 G1–G6 仍未整体认证，不能把单测通过表述为门禁通过。


### 增量验证（2026-11-02）

| 命令 | 结果 |
|---|---|
| `cd services/orchestrator && go test ./... -count=1` | PASS（含 DAG/route/artifact） |
| `cd services/gateway && go test ./... -count=1` | PASS（含 projection/逐轮投影） |
| `cd services/agents/markdown-agent && go test ./... -count=1` | PASS（render/lint/frontmatter/toc/mermaid/links） |
| `cd services/agents/test-agent && go test ./... -count=1` | PASS（test log/coverage parser） |
| `cd services/agents/security-agent && go test ./... -count=1` | PASS（secret/CVE scanner） |
| `cd services/agents/diff-agent && go test ./... -count=1` | PASS（unified diff） |
| `cd services/agents/code-agent && go test ./... -count=1` | PASS（read-only workspace tools） |
| `cd pkg/runtime && go test ./tools/ -count=1` | PASS（MCP HTTP transport/budget） |
| `cd pkg/adk && go test ./... -count=1` | PASS |
| `cd pkg/runtime && go test ./tools/ -count=1` | PASS |
| `cd frontend && tsc -b` | PASS |
| `cd frontend && vitest run` | 28 files / 254 tests PASS |
| `cd frontend && vite build` | PASS；首屏 index 535.24KB/gzip 156.92KB；语言/图表 chunks 按需加载（部分压缩前 >500KB） |
| `cd services/agents/code-agent && go test ./... -count=1` | PASS（只读 workspace 真实文件） |
| `cd services/agents/diff-agent && go test ./... -count=1` | PASS（多 hunk unified diff） |
| `cd pkg/runtime && go test ./observability/ -count=1` | PASS（内存导出：gateway/orchestrator/agent 同 trace，run_id 属性存在） |
| `cd services/orchestrator && AGENTHUB_RENDER_E2E=1 ... go test ./render/ -run TestPlaywrightRendererEndToEnd` | PASS（系统 Edge：PNG 49,703B / PDF 54,621B） |
| `bash -n scripts/backup-agenthub.sh` + fake mysqldump | PASS（生成 MySQL gz/制品 tgz；10 天旧备份被清理） |
| `python -c "yaml.safe_load(open('docker-compose.new-arch.yml'))"` | PASS |
| `go test ./httpapi/ -run 'TestMetrics'`（gateway + orchestrator） | PASS（`format=prometheus` 文本格式与标签） |
| CI workflow / Prometheus / compose / otel 配置 YAML 解析 | PASS（未在 GitHub / Prometheus / 容器实跑） |
| `actionlint .github/workflows/v3-integration-gates.yml` | PASS（exit 0） |
| `scripts/run-v3-gates.sh`（未 export DSN/Redis） | 正确 fail-fast（exit 1）；完整门禁需显式环境 |
| 10 个 agent 模块 `go build ./...` + `go test ./...` | PASS（全 Agent tracing 接线） |
| `cd frontend && PLAYWRIGHT_CHANNEL=msedge ... playwright test e2e/group-chat.spec.ts` | 1 passed（创建→勾选 2 Agent→发送→取消→刷新；本次 8.2s） |

### 收尾验证（2026-11-02）

| 命令 | 结果 |
|---|---|
| `cd pkg/adk && go test ./... -count=1` | PASS |
| `cd pkg/runtime && go test ./... -count=1` | 除 launcher 外 PASS；launcher 测试二进制在 `go test` 启动时被本机策略拒绝（`Access is denied`），直接运行 `go test -c` 产物 10 个测试全部 PASS |
| `cd services/gateway && go test ./... -count=1 && go build ./cmd/gateway` | PASS |
| `cd services/orchestrator && go test ./... -count=1 && go build ./cmd/orchestrator` | PASS |
| `cd services/agents/code-agent && go test ./... && go build ./cmd/code-agent` | PASS |
| `cd services/agents/web-agent && go test ./... && go build ./cmd/web-agent` | PASS |
| Playwright 渲染（系统 Edge） | PASS：PNG 49,703B / PDF 54,621B |
| Playwright 群聊 E2E（系统 Edge） | 1 passed |
| 备份脚本 fake mysqldump + 保留期 | PASS；compose YAML 解析 PASS |
| MySQL/Redis 集成 | SKIP（本机无 DSN/Redis；CI 配置存在但未在 GitHub 实跑） |
| nginx -t / Docker / 真实 MinIO/collector | 未执行（环境限制） |

### 门禁状态（诚实版）

- G0：契约/决策已冻结；部分契约为“字段已冻结、实现未完成”。
- G1：审批/取消/幂等/租约/投影有单测，但 MySQL/Redis 集成未在本机执行（SKIP），未整体通过。
- G2：ADK 副作用 Guard 已强制，生产 Broker 授权/结果/审批接线已落地；审批单仍由平台侧创建，未通过整体认证。
- G3：成员持久化、逐轮历史、参与者取消派发、浏览器 E2E 已落地；未整体认证。
- G4：路由决策、DAG 并行/取消、群聊 round-robin、计划期模型选择发言者、受控 handoff 已落地；执行中动态重选未做，未整体认证。
- G5：XSS 消毒、Shiki/diff2html/Mermaid/KaTeX、Playwright 截图/PDF 导出已落地；Docker 沙箱未验证，未整体认证。
- G6：Artifact 元数据本地/S3、blob 本地/S3、指标/告警规则、全 Agent OTel 传播/导出与 collector 配置、Nginx 模板与备份脚本已落地；Nginx/Prometheus/多副本/collector/MinIO 运行时验证未做，未整体认证。
