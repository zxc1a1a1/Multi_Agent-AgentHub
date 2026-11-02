# 证据目录规范（Evidence）

本目录保存 v3 缺口修复计划（`docs/plans/v3-gap-atomic-plan.md`）中每个原子任务的**可复现证据**。
证据目录是验收材料，不是完成度声明：只有目录里真实存在的命令输出才能支撑对应结论。

## 1. 目录结构

```text
docs/reports/evidence/<YYYY-MM-DD>/<task-or-testId>/
  commands.txt      # 可复现命令；只写变量名，不写值
  summary.md        # PASS/FAIL/SKIP、环境、耗时、提交 SHA、§5.5 交互覆盖表
  events.jsonl      # 每行一个事件（测试用例结果或关键交互）
  interactions.jsonl# 测试内部写出的 §5.2 交互事件（可选，由被测用例产出）
  logs/             # 脱敏日志（可选）
  screenshots/      # 可选；不得含密钥或隐私
```

`-baseline` 后缀的目录是诊断跑（用于确认环境是否可用、哪些用例仍 SKIP），
**不作为任务通过证据**；正式证据只写在任务 ID 目录里。

## 2. 采集方式

统一通过 `scripts/run-integration-evidence.sh` 采集，本地与 CI 走同一段逻辑：

```bash
export AGENTHUB_MYSQL_TEST_DSN='<operator supplied>'
export AGENTHUB_REDIS_TEST_ADDR='<operator supplied>'
bash scripts/run-integration-evidence.sh \
  --test-id F-03 --plan-task R0-02 \
  --packages "./services/orchestrator/..." \
  --run '^TestIntegration' \
  --timeout 25m \
  --out docs/reports/evidence/$(date -u +%F)/R0-02
```

脚本行为：

- 只检查环境变量是否存在，**只打印变量名，不打印值**；不读取 `.env`，不解析凭据文件。
- 用 `go test -json` 运行，解析 `Action=run/pass/fail/skip`、`Test`、`Package`、`Elapsed`。
- 生成 `events.jsonl`（每测试一行）、`summary.md`（PASS/FAIL/SKIP 计数、提交哈希、起止时间、交互覆盖表）、`commands.txt`。
- 日志中的 DSN / Redis 地址在落盘前替换为 `<redacted-dsn>` / `<redacted-addr>`；替换在 awk 内部通过
  `ENVIRON` 完成，真实值不进入命令行参数。
- 证据目录写完后自扫描：若目录内出现任何已导出变量的值，或命中
  `sk-[A-Za-z0-9_-]{20,}` / `BEGIN ... PRIVATE KEY` / `user:pass@` 模式，立即失败且不产出结论。

退出码：

| code | 含义 |
|---|---|
| 0 | 至少一个测试 PASS 且无 SKIP、无失败 |
| 1 | 存在失败、`go test` 非零退出、证据解析失败或密钥扫描命中 |
| 2 | 必需环境变量缺失（stderr 只出现变量名） |
| 3 | 出现 SKIP（不算通过） |
| 4 | 没有任何 PASS（选择器没命中可运行用例） |

## 3. JSONL 事件约定

字段遵循计划 §5.2：`schemaVersion`、`evidenceId`、`testId`、`planTaskId`、`layer`、`actor`、
`action`、`target`、`status`、`recordedAt` 必填；`runId`/`traceId` 等关联字段可用就填，没有写空字符串。

两类行：

- `action=test_case`：门禁脚本产出的用例结果行（`status=ok|error|skipped`）。
- `action=hitl_*|tool_*|memory_*|sse_*|mq_*|artifact_*|projection_commit|lease_claim` 等：
  由被测用例在真实交互点写出。用例通过环境变量 `AGENTHUB_EVIDENCE_JSONL`（脚本自动注入）
  找到目标文件；未注入时不写任何行。

`detail` 只允许计划 §5.2 的白名单键。禁止写入 `prompt`、`content`、`text`、`headers`、`body`、
`token`、`dsn`、`password`、`cookie`、`authorization`、`blob`。

## 4. PASS 前置条件

- `summary.md` 必须包含 §5.5 交互覆盖表；未覆盖的层写 `not_applicable` 加原因，不能留空。
- 任何 `SKIP` 必须写明环境原因，且不计入通过。
- 修复型任务必须附“还原修复会 FAIL”的回归说明。
- 证据入库前完成密钥扫描。
