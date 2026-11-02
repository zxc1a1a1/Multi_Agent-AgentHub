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

## 工作流实现（2026-11-02）

`.github/workflows/v3-integration-gates.yml` 包含 5 个 job：

- `integration`：MySQL 8.4 + Redis 7，`TestIntegration` 无 SKIP 才通过。
- `unit`：Go 模块矩阵（pkg/adk、pkg/runtime、gateway、orchestrator、全部
  services/agents/*），逐个 `go test ./...`；有 cmd 的服务同时 build。
- `frontend`：`npm ci`、`vitest --run`、`npm run build`（含 tsc）。
- `e2e`：安装 Chromium，跑 `e2e/group-chat.spec.ts`（API 为浏览器内 mock，
  不依赖真实后端）；失败上传 Playwright report。
- `compose-config`：`docker compose config --quiet` 校验
  `docker-compose.new-arch.yml`（仅静态解析，不启动容器）。
- 状态：工作流文件通过 YAML 解析与模块本地测试；未在 GitHub 实跑，因此不能
  宣称这些 job 已通过。

## 条件必过 Gates

- docker compose smoke。
- demo E2E。
- migration check。
- coverage threshold。

## 禁止

测试失败仍合并、普通 CI 使用真实 LLM key、跳过 contract test、CI 输出 secret、
把默认 SKIP 的集成测试表述为已通过。
