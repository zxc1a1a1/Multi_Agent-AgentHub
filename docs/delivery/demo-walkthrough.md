# AgentHub v1.0 演示说明

## 演示准备

### 环境要求

- Docker & Docker Compose（最新版本即可）
- 4 GB+ 可用内存
- **不需要**真实 LLM API key
- **不需要**MySQL / PostgreSQL
- **不需要**任何外部服务

### 启动命令

```bash
# 克隆仓库（如尚未克隆）
git clone <仓库地址>
cd AgentHub

# 一键启动新架构五服务
docker compose -f docker-compose.new-arch.yml up --build -d

# 等待约 30-60 秒，确认所有服务 healthy
docker compose -f docker-compose.new-arch.yml ps
```

### 前端入口

浏览器打开：**http://localhost:3000**

## 场景 1：Single Code-Agent

**演示内容**：用户请求代码生成，code-agent 返回 Go 代码。

**curl 命令**（可备选演示）：

```bash
CONV_ID=$(curl -s -X POST http://localhost:8080/api/conversations \
  -H 'Content-Type: application/json' \
  -d '{"userId":"demo","agentName":"code-agent"}' | \
  python3 -c "import sys,json; print(json.load(sys.stdin)['id'])")

curl -N -X POST http://localhost:8080/api/chat \
  -H 'Content-Type: application/json' \
  -d "{\"conversationId\":\"$CONV_ID\",\"message\":\"用 Go 写一个 HTTP API 接口\"}"
```

**预期结果**：
- SSE stream 包含 Go 代码片段
- sender 为 code-agent
- 无 error/run_error 事件

**前端操作**：
1. 选择 "Code Agent"
2. 输入 "用 Go 写一个 HTTP API 接口"
3. 观察 Code Agent 气泡包含 Go 代码高亮

## 场景 2：Single Web-Agent

**演示内容**：用户请求网页生成，web-agent 返回 HTML。

```bash
curl -N -X POST http://localhost:8080/api/chat \
  -H 'Content-Type: application/json' \
  -d "{\"conversationId\":\"$CONV_ID\",\"message\":\"写一个 HTML 登录页面\",\"agentName\":\"web-agent\"}"
```

**预期结果**：
- SSE stream 包含 HTML section
- sender 为 web-agent
- 前端渲染 web preview

## 场景 3：Mixed Ordered Parallel

**演示内容**：一条消息触发两个 Agent，Orchestrator 自动编排。

**关键点**：不传 `agentName`，让 RulePlanner 自动检测关键词。

```bash
curl -N -X POST http://localhost:8080/api/chat \
  -H 'Content-Type: application/json' \
  -d "{\"conversationId\":\"$CONV_ID\",\"message\":\"写一个 HTML 登录页面，并实现 Go API 接口\"}"
```

**预期结果**：
- 前端显示 **3 条独立 Agent 消息**（不合并）：
  1. Web Agent（HTML 登录页面）
  2. Code Agent（Go API 代码）
  3. Orchestrator（"All 2 task(s) completed successfully"）
- Web Agent 输出先于 Code Agent 输出（ordered_parallel 顺序）
- 每个气泡独立显示 sender 名称

## 场景 4：刷新后历史消息恢复

**演示内容**：刷新页面后消息保持 Multi-Agent 分离。

**操作**：在场景 3 之后，直接刷新浏览器（F5）。

**预期结果**：
- 用户消息 + 3 条 Agent 消息正确恢复
- 每个气泡显示正确的 senderDisplayName（Web Agent / Code Agent / Orchestrator）
- **不合并**为一个 assistant 气泡
- 包含 runId 等关联信息

**curl 验证**：

```bash
curl -s http://localhost:8080/api/conversations/$CONV_ID/messages | python3 -m json.tool
```

**预期**：返回 JSON 数组，包含 senderType/senderName/senderDisplayName/runId/status 字段。

## 场景 5：SQLite 持久化验证

**演示内容**：验证数据持久化到 SQLite。

```bash
# 查看 gateway 容器的 /data 目录
docker compose -f docker-compose.new-arch.yml exec gateway-new ls -la /data/

# 应显示 agenthub.db 文件
```

**服务重启后**：消息仍然可以 replay（因为数据在 named volume 中持久化）。

## 场景 6：错误脱敏

**演示内容**：验证错误消息不泄露敏感信息。

**curl 示例**（前端自动处理）：

```bash
# 发送不含关键词的消息（RulePlanner 匹配不到规则）
curl -N -X POST http://localhost:8080/api/chat \
  -H 'Content-Type: application/json' \
  -d "{\"conversationId\":\"$CONV_ID\",\"message\":\"hello\"}"
```

**预期结果**：
- 返回错误提示
- **不包含**：`sk-*` token、`OPENAI_API_KEY`、panic stack trace、文件路径

## 演示讲解词

> AgentHub 是一个多 Agent 协作平台。在 v1.0 版本中，我们实现了完整的前后端分离架构，包括独立的 Gateway、Orchestrator 和两个 Agent 服务。
>
> 刚刚演示的第一个场景是通过 Gateway 调用单个 code-agent 生成代码。第二个场景是 web-agent 生成 HTML 页面。
>
> 第三个场景是本项目的核心亮点：当用户说"写 HTML 页面并实现 Go API"时，Orchestrator 通过 RulePlanner 检测到 web 和 code 两个关键词，自动生成有序并行计划，先后调用 web-agent 和 code-agent 执行任务，最后返回 summary。前端显示的是三条独立的 Agent 消息，每条都有明确的发送者标签。
>
> 刷新页面后（第四场景），消息通过 SQLite 持久化恢复，仍然保持三条独立消息，不会合并为一个 assistant。这是我们做的 replay 能力，比很多同类产品的"刷新后所有 agent 消息混成一个 assistant 气泡"要精确得多。
>
> 整个系统不需要任何真实 LLM API key，所有 Agent 使用确定性 mock 响应。CI/CD 通过 GitHub Actions 全自动验证，包括单 Agent、多 Agent 并行、持久化 replay、日志安全审计。一键 `docker compose up` 即可启动全部五个服务。

## 常见问题和 Fallback 说明

**Q: 为什么 Agent 返回的是 mock 内容而不是真实 AI 生成？**

A: v1.0 使用确定性 mock 响应来验证架构完整性和编排正确性。LLMPlanner 和真实 LLM 接入是下一步（v1.1）的计划，需要 API key 管理和安全审查。

**Q: 如果没有 Docker 怎么办？**

A: 可以手动单独启动各服务。参考各服务 `cmd/` 下的 main.go 入口。Go 测试和前端测试不需要 Docker。

**Q: AGENTHUB_GATEWAY_STORE=memory 和 sqlite 有什么区别？**

A: `memory` 模式消息在进程内存中，重启丢失；`sqlite` 模式消息持久化到 SQLite 文件。compose 默认使用 `sqlite` 模式，本地开发可用 `memory`。

## 停止和清理

```bash
docker compose -f docker-compose.new-arch.yml down
# 如需清理数据卷
docker compose -f docker-compose.new-arch.yml down -v
```

---

- Created: 2026-06-05
- Step: AgentHub v1.0 Step 5 — Demo Walkthrough
