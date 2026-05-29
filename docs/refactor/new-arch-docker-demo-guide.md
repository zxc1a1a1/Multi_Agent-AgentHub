# New Architecture Docker Demo Guide

## 1. Demo 目标
本 Demo 用于验证新架构下 `code-agent-new`、`web-agent-new`、`gateway-new`、`frontend-new` 的联通与最小验收链路。  
本轮明确不实现 Orchestrator / Planner / Executor。

## 2. 服务拓扑
```text
frontend-new
  -> gateway-new
      -> code-agent-new
      -> web-agent-new
```

约束：
- Frontend 只访问 Gateway（`/api` 反代到 `gateway-new:8080`）。
- Frontend 不直连子 Agent，不直连 A2A endpoint。
- Gateway 通过静态注册路由到 `code-agent-new` / `web-agent-new`。

## 3. Docker / Compose 预检
先执行 Doctor：

```powershell
powershell -ExecutionPolicy Bypass -File ./doctor-new-arch.ps1
```

Doctor 会检查：
- 关键文件是否存在
- Docker / compose / compose config
- 常用端口占用
- 服务可达性（可选；服务未启动只给 WARN）

你也可以手动执行：

```powershell
docker version
docker compose version
docker compose -f docker-compose.new-arch.yml config
```

说明：
- `compose config` 通过，不等于 `compose build/up` 一定通过。
- 若 Docker daemon 不可用，这是环境问题，不是代码逻辑失败。

## 4. 启动方式
```powershell
docker compose -f docker-compose.new-arch.yml up --build
```

或：

```powershell
make docker-new-arch-up
```

默认端口：
- `frontend-new`: `http://localhost:3000`
- `gateway-new`: `http://localhost:8080`
- `code-agent-new`: `http://localhost:8081`
- `web-agent-new`: `http://localhost:8082`

## 5. Smoke 验证
PowerShell：
```powershell
powershell -ExecutionPolicy Bypass -File ./smoke-new-arch.ps1
```

支持参数：
```powershell
powershell -ExecutionPolicy Bypass -File ./smoke-new-arch.ps1 `
  -GatewayURL "http://localhost:8080" `
  -CodeAgentURL "http://localhost:8081" `
  -WebAgentURL "http://localhost:8082" `
  -TimeoutSeconds 60
```

Linux/macOS（不依赖 jq）：
```bash
GATEWAY_URL=http://localhost:8080 \
CODE_AGENT_URL=http://localhost:8081 \
WEB_AGENT_URL=http://localhost:8082 \
TIMEOUT_SECONDS=60 \
sh ./smoke-new-arch.sh
```

说明：
- 当服务未启动时，smoke `exit 1` 是预期行为。
- 当 Docker daemon 不可用且未手动启动服务时，smoke 失败是预期行为，不是伪造结果。

## 6. Frontend 容器与 Nginx 说明
- `frontend/Dockerfile` 已采用多阶段构建（Node build + Nginx runtime）。
- `frontend/.dockerignore` 已排除 `.env*`、`node_modules`、`dist`。
- `frontend/nginx.conf` 的 `/api` 仅反代到 `gateway-new:8080`。
- 已开启 SSE 友好代理配置：`proxy_buffering off`、`proxy_cache off`、`proxy_http_version 1.1`。
- 边界要求：frontend 不能直连子 Agent，也不能直连 A2A endpoint。

## 7. 基本验收用例
- GET `/api/agents` 返回 `code-agent` / `web-agent`。
- POST `/api/chat` with `agentName=code-agent` 有代码类响应。
- POST `/api/chat` with `agentName=web-agent` 有网页类响应。
- POST `/api/chat` with `agentName=unknown-agent` 返回安全错误（无内部 URL / token / stack）。

## 8. 已知限制
- 当前 `code-agent` / `web-agent` 为 v0.1 mock 语义。
- 未调用真实 LLM。
- 当前无 Orchestrator / Planner / Executor。
- `/api/agents` 当前为静态注册表输出。
- Compose 不包含数据库服务。
- `web_preview` 维持 Safe Mode 路径。

## 9. Windows 额外说明
- 如遇 `npm.ps1` 执行策略问题，使用 `npm.cmd`：
  - `npm.cmd test -- --run`
  - `npm.cmd run build`
- 如本机无 `sh`，优先使用 PowerShell 版本脚本。
