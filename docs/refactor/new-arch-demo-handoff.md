# New Architecture Demo Handoff

## 1. 当前最新能力
当前可演示能力：
- `code-agent-new`
- `web-agent-new`
- `gateway-new`
- `frontend-new`
- `GET /api/agents`
- `POST /api/chat` + `agentName` 显式路由
- Gateway 到 Frontend 的 SSE 流式返回
- WebPreview Safe Mode（不执行不可信 HTML/JS）

## 2. 当前不是哪些能力
当前明确不是：
- 不是 Orchestrator
- 不是 Planner
- 不是自动意图编排
- 不是真实 LLM
- 不是生产级 Agent Registry
- 不是完整 Artifact 系统

## 3. 拉取代码
PowerShell：
```powershell
git checkout g
git pull origin g
```

## 4. 本地静态验证
PowerShell（摘要）：
```powershell
cd services/gateway
go test ./... -v
go build ./...

cd ../agents/code-agent
go test ./... -v
go build ./...

cd ../web-agent
go test ./... -v
go build ./...

cd ../../../frontend
npm.cmd test -- --run
npm.cmd run build
```

## 5. Docker 环境预检
PowerShell：
```powershell
powershell -ExecutionPolicy Bypass -File ./doctor-new-arch.ps1
```

## 6. 启动 Demo
PowerShell：
```powershell
docker compose -f docker-compose.new-arch.yml up --build
```

或：
```powershell
make docker-new-arch-up
```

## 7. Smoke 验证
PowerShell：
```powershell
powershell -ExecutionPolicy Bypass -File ./smoke-new-arch.ps1
```

## 8. 前端访问
- `http://localhost:3000`

## 9. 常见失败
- 见：[new-arch-demo-troubleshooting.md](./new-arch-demo-troubleshooting.md)

## 10. 接手注意事项
- 不要提交 `.env`
- 不要提交 `frontend/dist`
- 不要提交 `node_modules`
- 不要删除旧 agents
- 不要把 Gateway 改成直接调 LLM
- 不要让 frontend 直连子 Agent
- 不要让 frontend 直连 A2A endpoint
