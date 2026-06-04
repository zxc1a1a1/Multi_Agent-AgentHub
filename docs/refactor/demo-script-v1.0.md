# Demo Script v1.0

## 环境准备

```bash
# 1. 确认 Docker 和 Docker Compose 可用
docker version
docker compose version

# 2. 克隆仓库并进入目录
cd AgentHub

# 3. 确认分支
git branch --show-current  # 应为 g
```

## 一键启动

```bash
# 启动新架构五服务（含 SQLite 持久化）
docker compose -f docker-compose.new-arch.yml up --build -d

# 等待所有服务 healthy（约 30-60 秒）
docker compose -f docker-compose.new-arch.yml ps

# 验证 health
curl http://localhost:8080/health    # gateway
curl http://localhost:8090/health    # orchestrator
curl http://localhost:8081/health    # code-agent
curl http://localhost:8082/health    # web-agent
curl http://localhost:3000/          # frontend
```

## 场景 1: Single Code-Agent

**目标**: 验证单 Agent 代码生成流程。

```bash
# 创建会话
CONV_ID=$(curl -s -X POST http://localhost:8080/api/conversations \
  -H 'Content-Type: application/json' \
  -d '{"userId":"demo-user","agentName":"code-agent"}' | \
  sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')

# 发送代码请求
curl -N -X POST http://localhost:8080/api/chat \
  -H 'Content-Type: application/json' \
  -d "{\"conversationId\":\"$CONV_ID\",\"message\":\"用 Go 写一个 HTTP API 接口\",\"agentName\":\"code-agent\"}"
```

**预期结果**:
- SSE stream 包含 `senderType: "agent"`, `senderName: "code-agent"` 事件
- 响应内容包含 Go 代码
- 无 error/run_error 事件

## 场景 2: Single Web-Agent

**目标**: 验证单 Agent 网页生成流程。

```bash
CONV_ID=$(curl -s -X POST http://localhost:8080/api/conversations \
  -H 'Content-Type: application/json' \
  -d '{"userId":"demo-user","agentName":"web-agent"}' | \
  sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')

curl -N -X POST http://localhost:8080/api/chat \
  -H 'Content-Type: application/json' \
  -d "{\"conversationId\":\"$CONV_ID\",\"message\":\"写一个 HTML 登录页面\",\"agentName\":\"web-agent\"}"
```

**预期结果**:
- SSE stream 包含 `senderType: "agent"`, `senderName: "web-agent"` 事件
- 响应内容包含 HTML section 标签
- 无 error/run_error 事件

## 场景 3: Mixed Ordered Parallel

**目标**: 验证多 Agent 有序并行编排。

```bash
CONV_ID=$(curl -s -X POST http://localhost:8080/api/conversations \
  -H 'Content-Type: application/json' \
  -d '{"userId":"demo-user","agentName":"code-agent"}' | \
  sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')

curl -N -X POST http://localhost:8080/api/chat \
  -H 'Content-Type: application/json' \
  -d "{\"conversationId\":\"$CONV_ID\",\"message\":\"写一个 HTML 登录页面，并实现 Go API 接口\"}"
```

**预期结果**:
- SSE stream 包含 3 个独立 Agent 消息：
  - web-agent（HTML 页面）
  - code-agent（Go API）
  - orchestrator（summary）
- web-agent 输出先于 code-agent 输出（ordered_parallel）
- 无 run_error/plan error 事件

## 场景 4: Page Refresh / Message Replay

**目标**: 验证页面刷新后消息恢复保持 Multi-Agent 分离。

```bash
# 使用场景 3 的 CONV_ID
curl -s http://localhost:8080/api/conversations/$CONV_ID/messages | python3 -m json.tool
```

**预期结果**:
- 返回 JSON 数组，包含 4 条消息（1 user + 3 agent）
- 每条 agent 消息有独立 `senderType`, `senderName`, `senderDisplayName`
- web-agent → `senderDisplayName: "Web Agent"`
- code-agent → `senderDisplayName: "Code Agent"`
- orchestrator → `senderDisplayName: "Orchestrator"`
- 包含 `runId`, `status` 字段
- 不合并为单一 assistant 消息

## 场景 5: Error Handling

**目标**: 验证错误安全展示。

```bash
CONV_ID=$(curl -s -X POST http://localhost:8080/api/conversations \
  -H 'Content-Type: application/json' \
  -d '{"userId":"demo-user","agentName":"code-agent"}' | \
  sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')

# 发送不包含关键词的消息（RulePlanner 无匹配规则时可能返回错误提示）
curl -N -X POST http://localhost:8080/api/chat \
  -H 'Content-Type: application/json' \
  -d "{\"conversationId\":\"$CONV_ID\",\"message\":\"hello\"}"
```

**预期结果**:
- 响应不包含 `sk-` token 或 `OPENAI_API_KEY` 等敏感信息
- error 消息为通用表述，不含内部堆栈或文件路径

## 场景 6: 数据持久化验证

**目标**: 验证 SQLite 持久化正常工作。

```bash
# 1. 确认 gateway-new 容器已启用 sqlite 模式
docker compose -f docker-compose.new-arch.yml exec gateway-new env | grep AGENTHUB

# 2. 确认 SQLite DB 文件已创建
docker compose -f docker-compose.new-arch.yml exec gateway-new ls -la /data/

# 3. 发送请求并验证 replay
CONV_ID=$(curl -s -X POST http://localhost:8080/api/conversations \
  -H 'Content-Type: application/json' \
  -d '{"userId":"demo-user","agentName":"code-agent"}' | \
  sed -n 's/.*"id"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p')

curl -s -N -X POST http://localhost:8080/api/chat \
  -H 'Content-Type: application/json' \
  -d "{\"conversationId\":\"$CONV_ID\",\"message\":\"写 Go 代码\"}"

# 验证 replay
curl -s http://localhost:8080/api/conversations/$CONV_ID/messages | \
  python3 -c "import sys,json; msgs=json.load(sys.stdin); print(f'{len(msgs)} messages'); [print(f'  {m[\"senderType\"]:>6} | {m.get(\"senderName\",\"\")}') for m in msgs]"
```

**预期输出示例**:
```
2 messages
   user | 
  agent | code-agent
```

## 一键验收

```bash
# 运行完整 smoke test（包含所有场景）
./smoke-new-arch.sh
```

预期输出: `SMOKE TEST PASSED`

## 清理

```bash
# 停止所有服务
docker compose -f docker-compose.new-arch.yml down

# 如需清理数据卷
docker compose -f docker-compose.new-arch.yml down -v
```

## 前端 UI 演示

浏览器访问 `http://localhost:3000`：

1. 创建新对话，选择 "Code Agent" 或 "Web Agent"
2. 输入请求（如 "写一个 HTML 登录页面，并实现 Go API 接口"）
3. 观察 3 个独立 Agent 气泡（Web Agent / Code Agent / Orchestrator）
4. 刷新页面 → 消息保持分离，不合并
5. 切换对话 → 对话列表正确显示

---

- Created: 2026-06-04
- Step: AgentHub v1.0 Demo Script
