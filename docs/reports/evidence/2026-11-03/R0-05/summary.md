# R0-05 GitHub CI 实跑认证

- task: R0-05
- 对应缺口：GAP-04
- run: https://github.com/zxc1a1a1/Multi_Agent-AgentHub/actions/runs/41588624458
- SHA: `e7460ebd332ef731544ac90b6c0653cc4364613a`（dev）
- 结果：**PASS**（19/19 jobs success，0 failed）

## 1. Job 结果

| Job | 结果 |
|---|---|
| integration（MySQL 8.4 + Redis 7，run-integration-evidence.sh） | success（exit 0，0 SKIP） |
| unit matrix（pkg/adk、pkg/runtime、gateway、orchestrator、10 agents） | 15/15 success |
| frontend（Vitest + build） | success |
| e2e（Playwright group-chat） | success |
| config-validation（compose/Dockerfile/promtool/nginx） | success |

原始数据：`logs/run.json`、`logs/jobs.json`。

## 2. 修复迭代

| Run | 失败项 | 根因 | 修复 |
|---|---|---|---|
| 41585615281 | integration / config-validation / gateway unit | S3 `TestIntegration...` 缺可选 env 被门禁记为 SKIP；Dockerfile 无 `ca-certificates` 字面量；Linux 连接错误未脱敏 | 见下 |
| 41587479018 | config-validation | promtool 步骤缺少 `--entrypoint promtool` | `ci: 修复 promtool 规则校验入口` |
| 41588624458 | 无 | - | **PASS** |

修复提交：
- `fix(ci): 修复 v3 门禁在 Linux/CI 的失败`（sanitizeStreamError；ca-certificates；S3 测试更名）
- `ci: 修复 promtool 规则校验入口`

## 3. 交互覆盖（§5.5）

| 交互 | 记录文件 | 条数 | 覆盖 |
|---|---|---|---|
| CI run/job | events.jsonl | 19 | yes |
| 本地/CI 代码 | logs/*.json | 2 | yes |
| 用户/SSE/HITL/记忆/MQ | - | 0 | not_applicable: CI 门禁 |

## 4. 边界

- integration 证据 artifact 由 workflow 上传（GitHub artifact），本地仅保存 run/job JSON。
- 未使用真实密钥；CI 使用固定测试 DSN/评论用凭据。
- GitHub 日志需仓库 admin 权限，未下载；以 job conclusion 为准。

## 5. 证据

- `docs/reports/evidence/2026-11-03/R0-05/`（commands.txt、events.jsonl、summary.md、logs/run.json、logs/jobs.json）
