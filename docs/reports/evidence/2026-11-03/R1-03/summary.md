# R1-03 受控自动记忆提取

- task: R1-03（`docs/plans/v3-gap-atomic-plan.md` §4.2）
- 对应缺口：GAP-12；M-15
- 环境：Windows；`go1.26.5`
- overall: **PASS**（4/4，0 SKIP）

## 1. 行为

- 开关：`MEMORY_AUTO_EXTRACT=1`，且必须 `MEMORY_MODE=scoped_long_term` + `MEMORY_ALLOW_PROPOSE=true`；
  否则 main 只打印 WARN，不构造 extractor（0 次模型调用）。
- 触发：run 终态为 `completed` 后，**异步**（detached goroutine，30s 超时）调用 extractor；
  用户响应不等待提取。输入为截断到 8KB 的用户文本，不发送完整 prompt/密钥。
- 模型输出严格 JSON `{"candidates":[...]}`；解析失败整批丢弃并记 `parse_error`。
- 校验：
  - content 1..4096 bytes；拒绝疑似密钥/密码/Bearer/私钥内容；
  - logicalKey 必须 `^[a-z0-9_.:-]{1,128}$`；
  - confidence ≥0.7；
  - sourceRefs 非空、格式合法且必须包含 `run:<runId>`；
  - 每 run 最多 3 条；同 scope+logicalKey 冲突计 `duplicate`。
- 写入只调用 `Controller.Propose`：**永不自动 confirm**，条目保持 candidate，等待用户确认。
- 日志只记录 attempted/proposed/rejected 数量与原因，不记录内容或 prompt。

## 2. 测试

| 测试 | 结果 |
|---|---|
| 合法/低置信/缺 refs/密钥/非法 key 混合 | PASS（仅 1 条 candidate） |
| 每 run 上限 3 + 重复 logicalKey | PASS（propose 3，limit=1，duplicate=1） |
| 非 JSON / 模型失败 | PASS（parse_error / 上报错误） |
| session-only 策略关闭 | PASS（0 次模型调用） |

回归：`internal/memory`、`httpapi` 全部通过。

## 3. 交互覆盖（§5.5）

| 交互 | 记录文件 | 条数 | 覆盖 |
|---|---|---|---|
| 提取/校验/propose | events.jsonl | 4 | yes（仅数量/原因） |
| 用户/API/SSE/HITL/MQ | - | 0 | not_applicable: 后台提取 |
| 记忆内容 | - | 0 | 不记录正文 |

## 4. 边界

- 未实现用户显式“记住”触发（仅 run completed）；MVP 摘要为用户文本，不是完整对话/工具结果。
- 前端 MemoryPanel 未新增 source/confidence 展示；candidate 仍通过既有列表/确认 API 处理。
- 每 scope 每小时配额未单独实现（每 run 3 条 + 既有 scope active 配额生效）。

## 5. 回滚

- `MEMORY_AUTO_EXTRACT=0`（默认）；已生成 candidate 可保持未确认或批量 revoke。

## 6. 证据

- `docs/reports/evidence/2026-11-03/R1-03/`（commands.txt、events.jsonl、summary.md、logs/go-test.jsonl）
