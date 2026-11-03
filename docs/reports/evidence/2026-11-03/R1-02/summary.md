# R1-02 短期上下文边界与运行入口硬上限门禁

- task: R1-02（`docs/plans/v3-gap-atomic-plan.md` §4.2）
- 对应缺口：M-04、P-02；M-01~M-03/M-05/M-06/M-09/M-12 回归
- commit baseline: `a083e4df78886e68370c7719e607b1d3bf3a4e9d`（dev）
- 环境：Windows 10.0.19045 / `go version go1.26.5 windows/amd64`
- 执行时间：2026-11-03T03:01Z – 03:03Z（UTC）
- overall: **PASS**

## 1. 结果

| 套件 | 顶层测试 | PASS | FAIL | SKIP |
|---|---|---|---|---|
| `pkg/runtime/pruning` + `pkg/runtime/harness` | 17 | 17 | 0 | 0 |
| `services/orchestrator/httpapi`（R1-02 范围） | 7 | 7 | 0 | 0 |
| `services/orchestrator/httpapi` 全量回归 | 全包 | PASS | 0 | 0 |

- `events.jsonl`：22 行；17 行 `ok`，5 行 `error` 均为**预期的拒绝路径**（`ErrContextTooLarge` 2 条、`pinned_context_exceeds_hard_limit` 1 条、`characterization_additive_window` 1 条、边界拒绝 1 条），不是失败。
- 未读取 `.env`，未连接外部 LLM。

## 2. 发现并修复的生产缺陷（M-04）

原特征化用例证明：`handleRunStream` 在 pinned 组装上下文超过 120,000 字符时仍调用 planner，缺口 M-04 真实存在。

修复（`services/orchestrator/httpapi/handler_run_stream.go`）：

- 在 `RUN_STARTED` 之后、planner 之前检查 `harness.DefaultContextLimits().MaxChars`；
- 超限时返回稳定错误 `ORCHESTRATOR_CONTEXT_TOO_LARGE`，不调用模型；
- 不回显被拒绝的上下文内容。

回归咬合证明（变异测试）：临时移除该门禁后
`TestR102RunStreamGatesOversizedPinnedBeforePlanner` FAIL（`oversized context reached the planner 1 time(s)`）；恢复后 PASS。测试不是空转。

## 3. 覆盖范围

- pinned 总字符 > `MaxChars` → `ErrContextTooLarge`，planner 0 调用；
- 边界：119,999 PASS、120,000 PASS、120,001 FAIL；
- system/pinned 永不静默截断；孤立 tool result 丢弃、tool call/result 成对保留；
- head/tail 窗口（20/4）确定性、裁剪停在上限内；
- 前端 100 条显示窗口不改变服务端收到的历史（`itemsIn=150, displayWindow=100`）；
- 门禁不误伤：100,000 字上下文仍调用 planner。

## 4. 交互覆盖（§5.5）

| 交互 | 记录文件 | 条数 | 覆盖 |
|---|---|---|---|
| 用户操作 | - | 0 | not_applicable: API 级测试，无浏览器 |
| API 请求/响应 | events.jsonl | 2 | yes（run 入口门禁/放行） |
| SSE 连接/事件 | - | 0 | not_applicable: 断言 handler 响应体，不测浏览器订阅 |
| HITL 请求/信号/Claim | - | 0 | not_applicable: 本用例无审批 |
| 工具授权/结果 | events.jsonl | 2 | yes（pairing/孤立 result 计数） |
| 记忆召回 | - | 0 | not_applicable: 本任务无记忆 |
| Artifact | - | 0 | not_applicable: 无制品 |
| 多实例 | - | 0 | not_applicable: 单实例 |
| 上下文裁剪（扩展 action） | events.jsonl | 18 | yes（数量/字符/token，无正文） |

## 5. 回滚

- 生产门禁按 commit revert；测试文件可回退；默认窗口参数未改。

## 6. 证据

- `docs/reports/evidence/2026-11-03/R1-02/`（commands.txt、events.jsonl、summary.md、logs/）
