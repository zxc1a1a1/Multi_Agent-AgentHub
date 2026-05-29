# Phase 10.5~10.7 Demo 稳定化完成报告

## 1. 本轮目标
本轮完成了以下稳定化工作：
- 增强 `smoke-new-arch.ps1` / `smoke-new-arch.sh` 的参数化、readiness wait 与可诊断性。
- 增加 Docker daemon / compose 预检提示与常见故障指引。
- 稳定化 `frontend/nginx` `/api` 代理与 SSE 相关配置。
- 稳定化 `docker-compose.new-arch.yml`（四服务一致命名、healthcheck）。
- 补充 Demo Acceptance Checklist 与 Troubleshooting 文档。

## 2. 新增/修改文件
新增：
- `docs/refactor/new-arch-demo-acceptance-checklist.md`
- `docs/refactor/new-arch-demo-troubleshooting.md`
- `docs/refactor/phase-10-demo-stabilization-report.md`

修改：
- `smoke-new-arch.ps1`
- `smoke-new-arch.sh`
- `docker-compose.new-arch.yml`
- `Makefile`
- `frontend/nginx.conf`
- `docs/refactor/new-arch-docker-demo-guide.md`
- `docs/refactor/phase-10-docker-new-arch-report.md`

说明：`frontend/Dockerfile` 已存在且满足最小安全容器化要求，本轮未做大改。

## 3. Smoke 脚本增强
已完成：
- readiness wait：新增 HTTP 就绪轮询（2 秒间隔，超时可配置）。
- 参数化 URL：
  - PowerShell：`GatewayURL`、`CodeAgentURL`、`WebAgentURL`、`TimeoutSeconds`、`SkipAgentHealth`
  - sh：支持 `GATEWAY_URL`、`CODE_AGENT_URL`、`WEB_AGENT_URL`、`TIMEOUT_SECONDS` 环境变量覆盖。
- API 覆盖：
  - `/health`（gateway/code/web）
  - `/api/agents`
  - `/api/conversations`
  - `/api/chat`（code-agent / web-agent / unknown-agent）
- unknown-agent 安全断言：校验错误事件不泄露 token / stack / panic / `code-agent-new:8080` / `web-agent-new:8080`。
- 预检与排障提示：输出 compose 启动命令、Docker daemon/compose/config/端口检查提示。

## 4. Frontend 容器化与 nginx
- `frontend/Dockerfile`：已存在（多阶段构建 + nginx runtime），本轮仅验收，不大改。
- `frontend/nginx.conf`：
  - `/api` 反代目标明确为 `gateway-new:8080`。
  - 保持 frontend 不直连子 Agent。
  - 新增 SSE 友好配置：`proxy_buffering off`、`proxy_cache off`、`proxy_http_version 1.1`、`proxy_read_timeout 3600s`。
- 未提交 `dist` / `node_modules`，未复制 `.env`。

## 5. Compose 稳定化
- 服务命名统一并保留：
  - `code-agent-new`
  - `web-agent-new`
  - `gateway-new`
  - `frontend-new`
- `gateway-new` 环境变量：
  - `AGENT_CODE_URL=http://code-agent-new:8080`
  - `AGENT_WEB_URL=http://web-agent-new:8080`
  - 兼容补充：`CODE_AGENT_URL` / `WEB_AGENT_URL`（当前 Gateway 代码读取 `AGENT_*`）。
- healthcheck：四个服务均配置基于 `wget` 的健康检查。
- 真实执行结果：
  - `docker compose config` 通过。
  - `docker compose build/up/down` 因本机 Docker daemon 不可用失败。

## 6. Makefile targets
- 保留了所有旧 target。
- 新增/修正了新架构相关 target：
  - `docker-new-arch-up`
  - `docker-new-arch-down`
  - `docker-new-arch-logs`
  - `docker-new-arch-config`
  - `smoke-new-arch`
  - `smoke-new-arch-sh`

## 7. 验证结果
已执行并记录：

- Frontend
  - `npm test -- --run`：首次因 PowerShell 执行策略拦截 `npm.ps1` 失败。
  - `npm.cmd test -- --run`：通过（5 files, 22 tests passed）。
  - `npm.cmd run build`：通过（有 chunk size warning，不阻塞）。

- Gateway
  - `cd services/gateway && go test ./... -v`：通过。
  - `cd services/gateway && go build ./...`：通过。

- Agents
  - `cd services/agents/code-agent && go test ./... -v`：通过。
  - `cd services/agents/code-agent && go build ./...`：通过。
  - `cd services/agents/web-agent && go test ./... -v`：通过。
  - `cd services/agents/web-agent && go build ./...`：通过。

- Compose
  - `docker compose -f docker-compose.new-arch.yml config`：通过。
  - `docker version`：失败（`dockerDesktopLinuxEngine` pipe missing，daemon 不可用）。
  - `docker compose -f docker-compose.new-arch.yml build`：失败（daemon 不可用）。
  - `docker compose -f docker-compose.new-arch.yml up -d`：失败（daemon 不可用）。
  - `powershell -ExecutionPolicy Bypass -File ./smoke-new-arch.ps1`：执行完成并 `exit 1`，按预期输出 readiness 超时与排障提示，无伪造通过。
  - `docker compose -f docker-compose.new-arch.yml down`：失败（daemon 不可用）。

- Git 与差异
  - `git status --short`：显示本轮目标文件改动 + 已存在未跟踪 `.claude/settings.local.json`（未修改）。
  - `git diff --stat`：
    - `Makefile`（32 行变更统计的一部分）
    - `frontend/nginx.conf`（32 行变更统计的一部分）
    - 其余新文件由 `git status --short` 显示为 `??`。

- 安全扫描
  - `git diff | Select-String sensitive-pattern`：未命中本轮 diff 中的真实密钥。
  - 全目录扫描命中项主要来自历史测试 mock、规则常量和文档说明文本（如 `OPENAI_API_KEY`、`DATABASE_URL`、`sk-THIS_IS_A_MOCK_SECRET_TOKEN_12345`），未发现真实凭据。

- dist/node_modules/exe 检查
  - `git status --short | Select-String "frontend/dist|node_modules|\.env|\.exe|\.claude/settings\.local\.json"`：仅命中 `.claude/settings.local.json`（未跟踪，未修改）。
  - `Get-ChildItem . -Recurse -Filter "*.exe" ...`：无新增可疑 exe 结果。

## 8. 保持不变范围
已确认：
- 未修改旧 `agents/code-agent`。
- 未修改旧 `agents/web-agent`。
- 未删除任何旧 Agent。
- 未修改旧 `server` 业务逻辑。
- 未破坏旧 `docker-compose.yml`。
- 未删除 Makefile 旧 target。
- 未修改根 `go.mod` / `go.sum`。
- 未读取/修改 `.env`。
- 未提交。
- 未使用 `git add`。
- 未调用真实 LLM。
- 未提交 `frontend/dist`。
- 未提交 `node_modules`。

## 9. 风险与注意事项
- 本机 Docker daemon 不可用时，只能完成 config/static validation 与 smoke 失败路径验证。
- 当前 Demo 仍是显式 `agentName` 路由，不是 Orchestrator。
- 未接入真实 LLM。
- `/api/agents` 仍是静态注册表。
- web preview 仍按 Safe Mode 路径处理。

## 10. 补充说明（Phase 10.8~10.10 之前）
- 建议先执行 doctor，再执行 smoke。
- `compose config` 通过不等于 `build/up` 通过。
- Docker daemon 不可用属于环境问题，不是代码逻辑失败。
- Windows 下若 `npm.ps1` 被策略阻止，请使用 `npm.cmd`。
- 若本机无 `sh`，可直接使用 PowerShell 版本脚本。

## 11. 下一步建议
下一步可以提交 Phase 10 Docker Demo 稳定化成果；提交后可在 Docker 可用环境执行完整 compose build/up/smoke，再决定是否进入最小 Orchestrator。
