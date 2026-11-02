# A0-07-pass R0-01 integration evidence

- task: R0-01
- testId: A0-07-pass
- commit: 7193a89f493be9c46d3d52083d738c7ece1d4b1d (dev)
- go: go version go1.26.5 windows/amd64
- started: 2026-11-02T08:59:15Z
- ended: 2026-11-02T08:59:23Z
- duration: 9s
- packages: ./services/gateway/store/...
- run pattern: ^TestIntegration
- go test exit: 0

## results

| metric | count |
|---|---|
| total tests | 8 |
| PASS | 8 |
| FAIL | 0 |
| SKIP | 0 |
| incomplete | 0 |
| interaction rows | 0 |
| overall | PASS |

## per test

| test | status | elapsed(s) |
|---|---|---|
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store#TestIntegrationSQLiteToMySQLFieldMapping | ok | 0.310 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store#TestIntegrationMySQLProjectionOffsetIsIdempotent | ok | 0.680 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store#TestIntegrationMySQLStoreCRUD | ok | 0.680 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store#TestIntegrationRecordChatRequestIdempotent | ok | 0.390 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store#TestIntegrationMySQLStoreSurvivesReopen | ok | 0.580 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store#TestIntegrationConversationIsolation | ok | 0.680 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store#TestIntegrationPublicEventJournal | ok | 0.530 |
| github.com/zxc1a1a1/Multi_Agent-AgentHub/services/gateway/store#TestIntegrationMySQLConversationParticipants | ok | 0.770 |

## skipped tests (must be explained)

- none

## failed tests

- none

## interaction coverage (plan §5.5)

| interaction | record file | rows | coverage |
|---|---|---|---|
| test case result | events.jsonl | 8 | yes |
| middleware interactions | interactions.jsonl | 0 | not_applicable: no test in this selection emitted interaction rows |
| user actions | - | 0 | not_applicable: gate script runs no browser code |
| SSE / AG-UI events | - | 0 | not_applicable: no SSE client in this selection |

## redaction

- DSN and Redis address values are replaced with `<redacted-dsn>` / `<redacted-addr>` in all logs.
- Environment values are read only through `ENVIRON` inside awk, so they never reach argv or evidence files.

## R0-01 acceptance checks

任务 ID：R0-01
对应缺口：GAP-01/GAP-02 前置、A0-07

| 验收项 | 命令 | 期望 | 实际 | 结果 |
|---|---|---|---|---|
| 未 export 环境 | `env -u AGENTHUB_MYSQL_TEST_DSN -u AGENTHUB_REDIS_TEST_ADDR bash scripts/run-integration-evidence.sh --out /tmp/r001-c1` | 退出码 2，stderr 只出现变量名 | 退出码 2，stderr 仅两行变量名，未创建目录 | PASS |
| 占位 DSN | `AGENTHUB_MYSQL_TEST_DSN=<placeholder> ... --out logs/placeholder-run` | 连接失败但证据不含 DSN 值 | 退出码 1；`logs/placeholder-run` 内 DSN 值 0 次出现，全部为 `<redacted-dsn>` | PASS |
| 有环境且至少一个 PASS | 本目录 `events.jsonl`（`./services/gateway/store/...`） | 退出码 0 | 退出码 0，6 PASS / 0 SKIP / 0 FAIL | PASS |
| 出现 SKIP | `--require-env AGENTHUB_REDIS_TEST_ADDR`，DSN 保持未设置 | 退出码 3 | 退出码 3，`logs/skip-run/summary.md` 记录 SKIP 原因 | PASS |
| 证据目录密钥扫描 | `grep -R` 变量值 + `sk-…{20,}` / `BEGIN …PRIVATE KEY` / `user:pass@` | 0 命中 | 0 命中（脚本内置扫描，命中即退出 1） | PASS |

原始 transcript 保存在 `logs/check1-missing-env.txt`、`logs/check2-placeholder.txt`、
`logs/check3-pass.txt`、`logs/check4-skip.txt`；占位与 SKIP 两次运行各自的完整证据目录在
`logs/placeholder-run/`、`logs/skip-run/`。

- 未修改文件（确认）：`services/**`、`pkg/**`、`frontend/**`、`docker-compose.new-arch.yml`、任何 `.env`
- 行为变化：无（新增只读门禁脚本；CI integration job 改为调用同一脚本，判定条件不变：SKIP 即失败、无 PASS 即失败）
- 回滚：删除 `scripts/run-integration-evidence.sh`，还原 workflow integration job 的两步 `go test` + grep 检查
