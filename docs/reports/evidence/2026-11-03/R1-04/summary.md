# R1-04 混合召回评估

- task: R1-04（`docs/plans/v3-gap-atomic-plan.md` §4.2）
- 对应缺口：GAP-13；M-17
- 环境：Windows；`go1.26.5`；离线合成数据（无真实用户数据）
- overall: **PASS（评测完成）**；结论：shadow **GO**，生产 **NO-GO**

## 1. 数据集

`services/orchestrator/internal/memory/testdata/memory-recall-eval.json`：

- 120 条合成记忆（10 个主题 × 12 条），60 条查询（每主题 3 条同义改写 + 3 条精确/混合）；
- 同义映射：deploy/release、bug/defect、database/db、memory/recall、auth/login、
  docker/container、queue/stream、chart/graph、form/schema、cache/redis；
- 相关集为该主题全部 12 条记忆（评估主题召回，而非单条排序）。

## 2. 指标（实测）

| 指标 | 关键词基线 | 合成语义 ranker | 变化 |
|---|---|---|---|
| recall@5 | 0.2292 | 0.3250 | **+41.8%** |
| MRR | 0.550 | 0.900 | +0.35 |
| 平均注入 tokens | 88.3 | 87.65 | -0.65 |
| p95 延迟 | 0.53ms | 0.53ms | ≤300ms |

门槛（recall@5 相对 +20%、p95 ≤300ms、token 预算）：**shadow 达标**。
但 ranker 是**合成同义词**，不是经评审的 embedding provider，因此
`productionGO=false`；不接入生产召回。

## 3. 实现

- `RecallRanker` 接口；`KeywordRanker` 复刻生产 SQL INSTR 语义（不扩展同义）；
  `SynonymRanker` 仅用于离线评测。
- `HybridRecallMode`：`off|shadow|on`，默认 off。控制器 `WithHybridRecall`：
  - shadow：计算候选但**不改变**注入结果（测试验证条目不增不减）；
  - on：应用 ranker（测试用 drop-all 验证）；
  - 生产 main **未接线**任何 ranker（没有真实 provider）。
- 生产关键词路径保持不变；未引入向量数据库/远程 embedding 调用。

## 4. 交互覆盖（§5.5）

| 交互 | 记录文件 | 条数 | 覆盖 |
|---|---|---|---|
| 召回评测/排序 | events.jsonl | 3 | yes（只含 ID/数量/指标） |
| 用户/API/SSE/HITL/MQ | - | 0 | not_applicable: 离线评测 |
| 记忆内容 | - | 0 | 不记录正文（数据集为合成） |

## 5. 边界 / 重新评估条件

- 未做真实 embedding/向量索引；未发送任何数据到外部 provider。
- 生产启用前需要：provider 数据出域评审、真实标注集、shadow 观察、回滚演练。
- `MEMORY_HYBRID_RECALL` 解析已就绪，但 main 不注入 ranker，on 不可达。

## 6. 回滚

- 默认 `off`；删除 ranker 接线不影响关键词路径。

## 7. 证据

- `docs/reports/evidence/2026-11-03/R1-04/`（commands.txt、events.jsonl、summary.md、logs/）
- 数据集：`services/orchestrator/internal/memory/testdata/memory-recall-eval.json`
