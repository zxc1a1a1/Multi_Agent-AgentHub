# Tool Broker Contract

**Status:** Active (v2, P0 frozen)
**Owner:** AgentHub Orchestrator `ToolBroker`
**Last updated:** 2026-10-29

## 1. 原则

- 模型只看到本次授权且可用的工具描述；平台执行权限、审批与账本。
- 原生工具与 Child Agent 工具调用都经过同一 Tool Broker；A2A 调用不绕过治理。
- 工具注解（MCP `readOnly`/`destructive`）是提示，不是授权。
- 读网页/源码得到的指令是不可信数据，不能触发追加授权、读取凭据、修改策略或提权。
- 写工具、联网、测试分别有独立权限，不提供“允许执行一切”的总开关。
- 前端 CodePreview 只展示文本；真正执行只在后端隔离环境。

## 2. ToolDescriptor（冻结字段）

```text
name, version, description, inputSchema, outputSchema,
readOnly, destructive, resourceScope, timeoutMs,
retryable, idempotent, approvalPolicy(cost: none|confirm|always),
maxOutputBytes, provider(native|mcp)
```

- 名称稳定、全局唯一；版本变更进入变更复核。
- 工具必须声明输出上限；超大输出转 artifact 引用。
- 未配置后端的工具（如 `web.search`）明确不可用，不生成虚假结果。

## 3. ToolCallEnvelope（冻结）

```text
runId, taskId, toolCallId, toolName, toolVersion,
argsHash (SHA-256 of canonical JSON), workspaceId,
approvalId, fencingVersion, budgetReservationId,
principalId, credentialScope(never included)
```

- 由平台生成；模型不能自报“已批准”、不能提供 fencingVersion 或 approvalId。
- 参数摘要 hash 与审批绑定；参数变化使旧审批失效。
- 授权信息不写入模型可见上下文。

## 4. 调用流程（冻结）

1. 校验工具在允许集合、参数符合 schema、输出/超时在 descriptor 范围内。
2. 校验 workspace/scope 授权；路径规范化、符号链接检查、拒绝凭据路径。
3. 需要审批时：创建审批记录并进入 `awaiting_approval`，不执行。
4. 派发前，在同一 SQL 事务内按当前 fencingVersion：消费审批（一次性）+ 预留预算 + 登记派发意图。
5. 提交后执行；结果与工具账本同事务保存（或结果先入库再对外宣告）。
6. 结果进入 observation/下一轮上下文；前端事件只含脱敏结果与引用。

## 5. 工作区隔离与只读工具（P3a）

- 受控根目录由配置提供；所有路径 `filepath.Clean` 后必须仍在根目录内。
- 符号链接解析后仍须在根目录内；拒绝 `.env*`、`*.pem`、`*.key`、`id_rsa`、凭据目录。
- `workspace.list` / `workspace.read` / `workspace.search` 只读、限量、可审计。
- 不读取凭据路径；不把绝对宿主路径、内部 URL 返回给模型/前端。

## 6. 写工具（P5，冻结约束）

| 工具 | 约束 |
|---|---|
| `workspace.apply_patch` | 独立工作区、预期文件 hash、冲突检测、审批参数不可变；不覆盖未确认用户改动 |
| `sandbox.run_tests` | 结构化 executable/argv、非 root、受限挂载/网络、CPU/内存/时间/输出上限；不在宿主机直接运行不可信脚本 |
| `web.search` / `web.fetch` | 显式 provider；SSRF/重定向/DNS 校验；拒绝 private/loopback/metadata 目标 |
| `artifact.publish` | 内部 artifactId；不让模型指定宿主路径或公开上传地址 |
| `delegate_agent` | 允许集合、预算配额、父子关系、取消与结果账本；深度 1 |

- 未批准的补丁、命令、网络请求实际调用次数必须为 0。
- 同一写调用重放、审批重放、旧 hash、过期授权、未知结果均有断言。
- 非幂等工具结果未知 → `unknown`，不自动生成新调用键重做。
- 支持幂等键的工具用真实后端验证去重，不靠本地假设。

## 7. MCP

- 只接入审核后的服务器和工具清单；schema/版本固定并纳入变更复核。
- MCP 工具经同一 Broker 授权/审计，不因来自 MCP 而降级权限。
- 不自动安装第三方 MCP 程序；安装/运行是单独批准的运维动作。
- MCP 返回内容按不可信数据处理。

## 8. 错误与 unknown 语义

| 状态 | 含义 | 行为 |
|---|---|---|
| `succeeded` | 工具确认成功，结果已入账 | 可复用结果 |
| `failed` | 明确失败且无副作用 | 按 descriptor 可重试 |
| `denied` | 未授权/审批拒绝 | 记录 observation，不执行 |
| `unknown` | 已派发但结果缺失/进程崩溃 | 不自动重放，先核对真实状态 |

- 对外错误脱敏：不含内部 URL、stack trace、DSN、凭据、原始 provider 错误全文。

## 9. 执行位置决策（v3 A0-06/A2-00）

**决策：agent-side authorize/result，Orchestrator 持有审批、预算、账本。**

- 工具仍在 Child Agent 进程内执行；模型只产出 tool call，不能自报 envelope。
- Agent 在执行任何 `readOnly=false` 工具前，必须调用平台授权入口；无 envelope /
  无审批 / fencing 过期 → 该调用执行次数为 0。
- 执行结果（succeeded/failed/unknown）回报 Orchestrator 账本；结果未知不得自动
  重试，必须先核对真实状态。
- ADK 层已强制：`SideEffectTool.SideEffect()==true` 且无 `ToolGuard` 时不执行
  （`pkg/adk/runner.go`）；这是本地防线，不替代 SQL 账本。
- 状态：授权入口与结果回报已接线（A2-02）：Orchestrator 提供
  `/internal/orchestrator/tools/authorize|result`，Agent 侧 `RemoteGuard` 通过
  A2A `envelope` 获取平台身份并在写工具执行前授权、执行后回报。
  工具级审批记录仍需平台侧创建；写工具注册（A2-05）尚未开放。

## 10. 预算、fencing 与幂等（A0-06 冻结语义）

| 概念 | 冻结语义 |
|---|---|
| `budgetReservationId` | 平台生成；同一次派发只预留一次；恢复 run 时沿用已用额度，不重置 |
| `fencingVersion` | 每次租约接管递增；`0` 不是合法的生产 fencing 值；所有写事务校验当前值 |
| `approvalId` | 一次性消费；参数 hash 变化使旧审批失效；跨实例确认走 `run_signals` 幂等键 |
| `idempotencyKey` | 写调用必填；`(run_id, idempotency_key)` 唯一；重放返回已存结果 |
| `unknown` | 已派发但无终态；不重做、不伪造成功；对外标记需人工/自动核对 |

验收：

- 未授权写工具调用次数为 0（ADK guard 测试 + Broker 测试）。
- 并发重复 confirm 只消费一次（`run_signals` 唯一键 + 内存路径）。
- 同一写调用重放返回原结果；`unknown` 重放返回 `ErrToolOutcomeUnknown`。
- 超出预算的调用在实际执行前被拒绝；resume 不重置预算（A1-01）。
