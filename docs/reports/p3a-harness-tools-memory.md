# P3a 阶段报告：Harness 骨架与只读工作区工具

- 日期：2026-10-29
- 基线：`dev / ac53163`（P2 部分完成后）
- 状态：**P3a 完成（接口/只读工具层）；P3b/P3c 未开始**。

## 1. 交付

| 交付 | 路径 | 说明 |
|---|---|---|
| BudgetManager | `pkg/runtime/harness/budget.go` | 总模型/工具/修复/重规划/并行预算，先预留后派发，checkpoint 恢复不归零 |
| RunState | `pkg/runtime/harness/state.go` | 冻结 accepted→…→terminal 状态机，终态不可逆 |
| ContextBuilder | `pkg/runtime/harness/context.go` | 有界窗口、system/pinned 保留、tool call-result 配对、孤儿结果丢弃 |
| LoopController 接口 | `pkg/runtime/harness/loop.go` | 单一循环所有权；语义动作仍由模型产生 |
| Tool Descriptor/Broker | `pkg/runtime/tools/descriptor.go`、`broker.go` | schema/读写/审批/超时/输出上限；写工具必须审批；identity 校验 |
| 只读工具 | `pkg/runtime/tools/workspace.go` | `workspace.list/read/search`，根目录限制、符号链接检查、凭据拒绝、输出截断 |
| 工具事件 | `pkg/runtime/tools/events.go` | 有界输出 + SHA-256 全量 hash，避免事件流存大结果 |
| MemoryPolicy/接口 | `pkg/runtime/memory/policy.go`、`controller.go` | off/session_only/scoped_long_term，默认 session_only；配额/TTL 与接口 |

## 2. 验证

```text
$ go test ./pkg/adk/... ./pkg/runtime/... -count=1
PASS（全部包 ok）

覆盖用例：
- 预算耗尽、并行预留、restore 不重置
- 状态迁移合法/非法、终态唯一
- 上下文保留 system/pinned、完整 tool 对、丢弃孤儿结果、必需上下文超限报错
- 凭据文件、路径穿越、越界 symlink、超大文件、二进制、搜索上限
- descriptor 写工具强制审批、broker identity/审批校验
- memory 模式默认值、权限组合、scope key 隔离
```

## 3. 未完成/边界

1. 尚未接入真实模型与 Orchestrator 服务入口；当前是库级/单元级验证，不是端到端工具调用。
2. 未接入前端工具事件（P4/P6 浏览器验收）。
3. Memory 只有策略与接口，无 MySQL 存储/召回/CAS/遗忘实现（P3c）。
4. P3b 的 checkpoint/审批/预算持久化、SQL fencing、取消/恢复未开始。
5. P2 的真实 MySQL 集成仍 SKIP，P3 依赖的持久化门禁尚未通过。

## 4. 结论

P3a 的“接口与只读工具骨架”可声明完成；不能声明 P3 整体完成，也不能声明模型已能正确选择工具。
