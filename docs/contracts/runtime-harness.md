# Runtime Harness Contract

**Status:** Active (v2, P0 frozen)
**Owner:** AgentHub Orchestrator / `pkg/runtime/harness`
**Last updated:** 2026-10-29

## 1. 目的与边界

Runtime Harness 是执行控制面，不是第二套 Agent 框架，也不是 Eval Harness。

- 复用 `pkg/adk` 的 `Agent` / `Model` / `Tool` / `Plugin` / `Runner` / `SessionService` 接口；同一模型/工具循环只有一个执行所有者。
- 不承载用户意图路由；语义决策只来自 MainAgent 模型。
- 不依赖 Redis 作为审批、终态、预算或 Memory 真值；这些以 SQL 为准。

```text
pkg/runtime/harness  ContextBuilder / BudgetManager / LoopController / PolicyGate
                     CheckpointManager / RunController / EvidenceCollector
pkg/runtime/tools    ToolDescriptor / ToolBroker / 原生只读工具（P3a）
pkg/runtime/memory   MemoryController / MemoryPolicy / 存储适配
```

## 2. 运行状态机（冻结）

```text
accepted → planning → executing → completed
                  ↘ awaiting_approval → executing
                  ↘ awaiting_input → planning
executing → paused / failed / cancelled
paused → 恢复校验 → executing / awaiting_approval / failed
```

- 终态集合：`completed | failed | cancelled`。终态唯一，不可逆；重复提交终态必须被 fencing 拒绝。
- 用户显式取消：保存 `cancelled`，尝试取消远端任务，输出 `STATE_UPDATE(cancelled)` + `RUN_FINISHED(cancelled)`，不是 `RUN_ERROR`。
- 普通失败：终止性 `RUN_ERROR`（脱敏）。
- 审批等待不占用工作 goroutine 或数据库长事务；持久化后释放执行资源。
- 断开浏览器订阅**不取消**已接受的 durable run；只有显式 cancel 才取消。

## 3. 预算（冻结初值）

| 项目 | 初值 | 口径 |
|---|---|---|
| 总模型调用 | 16 次/run | 主模型、修复、摘要/记忆抽取、子 Agent 共用 |
| 工具调用 | 32 次/run | 含重试与子 Agent 工具 |
| 模型修复 | 1 次 | 每个计划/输出节点 |
| 重新规划 | 2 次 | 模型主动 revise_plan |
| 并行任务 | 3 个 | 全局并发上限 |
| 委派深度 | 1 层 | 子 Agent 不得再委派 |
| 运行执行时间 | 180 秒 | 不含审批等待 |
| 普通工具超时 | 15 秒 | 单次 |
| 沙箱测试超时 | 60 秒 | 单次 |
| 审批有效期 | 15 分钟 | 从创建开始 |
| 无进展阈值 | 3 次连续相同工具+参数 hash+结果版本 | 达到后要求 clarify 或终止 |

- 预算必须先预留再派发；并行/重试不得使总量倍增。
- 恢复**不得**重置已消费预算；checkpoint 记录已用计数。
- ADK 默认 25 轮不等于全链路模型预算；各层统一计数。
- 预算/超时必须是可实际触发的配置，不允许文档承诺而配置不可达。

## 4. Checkpoint（冻结）

- 版本字段 `checkpointVersion`（整数，初值 1），内容至少包含：runId、planId/revision、执行游标、上下文/产物引用、pending calls、已完成调用与结果引用、预算已消费、授权版本、agent/tool 版本、审批引用、workspace、fencingVersion。
- 保存的是可序列化状态，不是 Go 栈或 cancelFunc。
- 只在安全边界保存：模型调用前后、工具调用完成之后、审批状态变化、终态前。
- 不在模型推理或外部工具执行期间保持数据库长事务。
- `session_events` 不得原样搬到 Redis 冒充 checkpoint。

### 4.1 崩溃恢复矩阵

| 崩溃位置 | 恢复策略 |
|---|---|
| 只读工具派发前 | 校验授权/预算后可重新派发 |
| 只读工具结果已保存 | 复用结果，注明版本；不假装新抓取 |
| 幂等写工具派发后 | 使用同一幂等键查询/重试，不生成新键 |
| 非幂等工具已派发但结果缺失 | 标记 `unknown`，先核对真实状态/人工处理，不自动重放 |
| 等待审批 | 恢复审批状态，验证 revision/hash/有效期 |
| Agent 远端任务状态未知 | 按远端 taskId 查询；不支持时明确 paused/unknown，不假设自动接续 |

- 同一 `toolCallId` 的成功结果可复用；缺结果的未知外部执行不得视为未执行。
- 工具结果与事件落库顺序必须保证：先持久化结果/账本，再对外宣告完成。

## 5. 执行所有权与 fencing（冻结）

- 执行所有权以 SQL 条件更新 + 递增 `fencingVersion` 为准；Redis 心跳/TTL 只辅助发现。
- 持有者续租、checkpoint、预算扣减、终态提交必须带当前 `fencingVersion`；旧版本更新必须失败。
- 过期 worker 不得继续提交状态；工具 Broker 派发前验证当前所有权。
- 远端副作用不可原子覆盖，因此“已执行但结果未入账”使用 `unknown`，不自动重试非幂等写操作。
- 不宣称 Redis 锁/fencing 提供任意外部工具的 exactly-once。

## 6. 审批（冻结）

审批绑定：`runId`、`planId`/`revision`、`toolCallId`、工具版本、参数 hash、workspace、授权主体、有效期。

1. 每个审批只能被其绑定的 `toolCallId` 消费一次。
2. 派发前，在校验当前 fencingVersion 的同一 SQL 事务中：消费审批 + 预留预算 + 登记派发意图；提交后才执行工具。
3. 同参数但新 `toolCallId` 不能复用旧审批。
4. 计划/参数/版本/工作区变化使旧审批失效。
5. 批准计划不等于批准其中所有未来工具调用；具体调用仍需策略许可。
6. 等待审批保存后释放工作资源；重启恢复必须校验 revision/hash/过期时间。

## 7. Artifact 与证据

- EvidenceCollector 记录工具真实退出状态、验证结果、产物引用；模型自评不是完成证明。
- Artifact 必须有稳定 `artifactId` 和持久化位置；单机持久卷与多节点共享存储能力分开声明，未通过跨节点测试只标注单机恢复。
- 大结果外置为 artifact 引用，上下文只保留摘要与引用。
- 日志/事件不包含密钥、DSN、原始凭据或完整敏感 prompt。

## 8. 与存储的边界

- SQL 不可用时：停止接受需要记账的新副作用（写工具、新委派、新运行），现有任务不得绕过账本继续产生副作用。
- Redis 不可用时：可跳过缓存、从 SQL 有界补读；不得使用无法复核的缓存做授权/召回。
- 不允许使用全局 writer 单点串行所有 run；每次写入使用独立连接/事务与条件更新。

## 9. 测试证据

- FakeModel 单测证明参数、调用计数、预算、权限、终态与映射。
- 实际 MySQL 集成测试证明事务/唯一约束/并发/fencing；sqlmock 不能替代。
- 两个独立 OS 进程故障测试证明接管与旧版本拒绝；两个对象实例不算。
- 浏览器 E2E 证明对话/审批/工具/恢复/取消；构建通过不算。
- 各类证据分开报告，不互相替代。
