# R1-07 core entries 与 recheck 策略接线

- task: R1-07（`docs/plans/v3-gap-atomic-plan.md` §4.2）
- 对应缺口：GAP-21
- 环境：Windows；MySQL 8.4.13（真实测试库）
- overall: **PASS**（新用例 1/1；memory/persistence/httpapi 回归全部 PASS）

## 1. 结果

- `MaxCoreEntries` 生效：召回时 core 条目先占槽位（最多 N 条），其余按关键词/时间填充；
  token 预算统一计算。
- core 条目不再被关键词过滤掉：SQL 条件为 `((关键词匹配) OR e.core = 1)`，排序 `core DESC`。
- `DefaultRecheckAfter` 生效：`Propose` 未显式给 `recheckAfter` 时写入 `now + 默认值`；
  召回时 `Stale=true` 表示已过复核时间；不自动删除/修改内容。
- 迁移 `009_memory_core_recheck.sql`：`memory_entries.core`、`recheck_after`（仅加列/索引，向前兼容）。
- Propose API 接受 `core`/`recheckAfter`；Entry 增加 `core/recheckAfter/stale` 字段。

## 2. 测试

`TestIntegrationMemoryCoreAndRecheck`（真实 MySQL）：
- core 条目的关键词不匹配查询，仍被召回且排在第一位；
- `MaxCoreEntries=1` 下 core 只占一个优先槽位；
- 过去 `recheck_after` → `Stale=true`；新条目默认 `recheckAfter≈now+1h` 且非 stale；
- DB 列 `core/recheck_after` 持久化。

回归：`internal/memory`、`internal/persistence/mysql`、`httpapi` 全部通过。

## 3. 交互覆盖（§5.5）

| 交互 | 记录文件 | 条数 | 覆盖 |
|---|---|---|---|
| 记忆 propose/confirm/recall | events.jsonl | 1 | yes（ID/数量/状态） |
| 用户/API/SSE/HITL/MQ | - | 0 | not_applicable: 记忆层测试 |
| 记忆内容 | - | 0 | 不记录正文 |

## 4. 边界

- `recheck_after` 在 Propose 时写入；Confirm/Update 不刷新复核时间（后续可按策略调整）。
- `MaxCoreEntries=0` 时不优先 core，但 core 仍可作为普通候选被填充。
- 未实现自动复核/提示 UI；仅返回 `stale` 标记供上层使用。

## 5. 回滚

- 迁移只加列；可将 `core` 置 false、清空 `recheck_after`，旧召回顺序不变。

## 6. 证据

- `docs/reports/evidence/2026-11-03/R1-07/`（commands.txt、events.jsonl、summary.md、logs/go-test.jsonl）
