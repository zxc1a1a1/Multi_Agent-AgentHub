# CI Quality Gates

## 必过 Gates

- lint / format。
- backend unit。
- frontend unit（`vitest --run`）。
- frontend typecheck + build（`tsc -b && vite build`）。
- schema validation。
- contract tests。
- security regression。
- minimal smoke。

## 集成 Gates（v3 A0-07）

- `.github/workflows/v3-integration-gates.yml` 启动 MySQL 8.4 与 Redis 7
  service，并注入：
  - `AGENTHUB_MYSQL_TEST_DSN`
  - `AGENTHUB_REDIS_TEST_ADDR`
- 运行 `go test -v -run 'TestIntegration' ./services/orchestrator/... ./services/gateway/...`。
- **SKIP 不算通过**：日志出现 `--- SKIP` 时该 job 必须失败；没有任何
  `--- PASS` 也必须失败。
- 本地缺少 DSN/Redis 时集成测试允许 SKIP，但报告必须写明 SKIP 原因，不得计入
  通过数。

## 条件必过 Gates

- docker compose smoke。
- demo E2E。
- migration check。
- coverage threshold。

## 禁止

测试失败仍合并、普通 CI 使用真实 LLM key、跳过 contract test、CI 输出 secret、
把默认 SKIP 的集成测试表述为已通过。
