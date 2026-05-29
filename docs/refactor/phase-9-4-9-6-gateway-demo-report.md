# Phase 9.4~9.6 交付报告（Gateway 新架构 Demo /api/agents / 多 Agent SSE Smoke）

## 1. 本轮目标
1. Phase 9.4：补齐 `services/gateway` 的本地可运行入口。  
2. Phase 9.5：新增 Gateway `GET /api/agents`，前端接入动态列表并保留 fallback。  
3. Phase 9.6：补联调文档与 smoke 测试入口。  

## 2. 代码改动清单
1. `services/gateway/cmd/gateway/main.go`（新增）  
2. `services/gateway/cmd/gateway/main_test.go`（新增）  
3. `services/gateway/httpapi/server.go`（新增 `/api/agents` + 可注入 Agent 摘要）  
4. `services/gateway/httpapi/server_test.go`（新增 `/api/agents` 测试）  
5. `frontend/src/lib/agents.ts`（重构为动态摘要映射 + fallback）  
6. `frontend/src/services/api.ts`（响应归一化、`/api/agents` 支持、会话创建兼容）  
7. `frontend/src/stores/agentStore.ts`（新增）  
8. `frontend/src/stores/agentStore.test.ts`（新增）  
9. `frontend/src/components/ConversationList.tsx`（接入动态 Agent 列表）  
10. `frontend/src/components/ChatWindow.tsx`（接入动态 Agent 选择器）  
11. `frontend/src/stores/messageStore.ts`（保留未知 agentName，不再仅限 code/web）  
12. `smoke-multi-agent-sse.sh`（新增，Go smoke）  
13. `smoke-multi-agent-sse.ps1`（新增，PowerShell smoke）  
14. `Makefile`（新增新架构启动与 smoke 命令）  

## 3. Phase 9.4：Gateway 新架构本地 Demo 启动入口
### 3.1 启动命令
```bash
go run ./services/gateway/cmd/gateway
```

### 3.2 新增行为
1. 从环境变量读取 Gateway 地址、CORS、鉴权开关。  
2. 从 `AGENT_CODE_URL`/`AGENT_WEB_URL` 组装静态 registry。  
3. 注入 `RoutingRunService`，Gateway 仅负责 API/SSE 转发与持久化入口。  
4. 支持优雅退出。  

### 3.3 关键环境变量
1. `GATEWAY_ADDR`（默认 `:8080`）  
2. `AGENT_CODE_URL`（默认 `http://127.0.0.1:8081`）  
3. `AGENT_WEB_URL`（默认 `http://127.0.0.1:8082`）  
4. `GATEWAY_DEFAULT_AGENT_NAME`（默认 `code-agent`）  
5. `AGENTHUB_API_TOKEN` + `GATEWAY_ENABLE_AUTH`（可选）  

## 4. Phase 9.5：`/api/agents` + 前端动态 fallback
### 4.1 Gateway 侧
1. 新增 `GET /api/agents`。  
2. 支持 `WithAgents(...)` 注入 Agent 摘要：`name/displayName/description/outputModes`。  
3. 未注入时返回默认 `code-agent` + `web-agent`。  

### 4.2 Frontend 侧
1. 新增 `agentStore`：优先请求 `/api/agents`。  
2. 请求失败或空列表时回退到静态 `AGENT_OPTIONS`。  
3. `ConversationList` 与 `ChatWindow` 从同一来源读取 Agent 列表。  
4. `api.ts` 兼容数组与 envelope（`{data:[...]}`）返回。  

## 5. Phase 9.6：多 Agent SSE 联调文档与 smoke
### 5.1 Smoke 命令
Linux/macOS（或具备 bash 的环境）：
```bash
bash ./smoke-multi-agent-sse.sh
```

Windows PowerShell：
```powershell
powershell -ExecutionPolicy Bypass -File ./smoke-multi-agent-sse.ps1
```

### 5.2 Smoke 覆盖项
1. Gateway 多 Agent SSE 路由核心测试：
`TestGateway_StaticAgentRegistry_MultiAgentChatSSE`。  
2. Gateway `/api/agents` API 测试：
`TestListAgentsDefault` + `TestListAgentsWithOverride`。  
3. （PowerShell 版本）前端动态 fallback 关键测试：
`agentStore/messageStore/agui-client/WebPreview`。  

### 5.3 Makefile 新入口
1. `make dev-gateway-new`  
2. `make dev-code-agent-new`  
3. `make dev-web-agent-new`  
4. `make dev-new-arch`  
5. `make smoke-multi-agent`  
6. `make smoke-multi-agent-ps`  

## 6. 本轮验证命令
1. `cd services/gateway && go test ./...`  
2. `cd frontend && npm test -- --run src/stores/agentStore.test.ts src/stores/messageStore.test.ts src/agui/client.test.ts src/components/WebPreview.test.tsx`  
3. `powershell -ExecutionPolicy Bypass -File ./smoke-multi-agent-sse.ps1`  

## 7. 边界与安全
1. 未写入真实密钥，未修改 `.env`。  
2. 未改 Orchestrator 主链路。  
3. `/api/agents` 只返回展示摘要，不暴露内部 agent URL。  
