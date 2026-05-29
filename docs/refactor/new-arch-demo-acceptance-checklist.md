# New Architecture Demo Acceptance Checklist

## 1. 前置条件
- Docker Desktop 已启动
- docker compose 可用
- 端口 8080/8081/8082/3000 未占用
- 当前分支 g
- 最新提交基线说明：`0e28b00 feat(frontend): 接入多 Agent 选择与新 Gateway SSE Demo`
- 无 .env 依赖
- 无真实 LLM key 需求

## 2. 构建验收
- docker compose config 通过
- docker compose build 通过
- code-agent-new image 构建成功
- web-agent-new image 构建成功
- gateway-new image 构建成功
- frontend-new image 构建成功（本轮包含 frontend）

## 3. 启动验收
- docker compose up 成功
- code-agent-new health pass
- web-agent-new health pass
- gateway-new health pass
- frontend-new 可访问

## 4. API 验收
- GET /api/agents 返回 code-agent/web-agent
- /api/chat code-agent 返回 code 内容
- /api/chat web-agent 返回 HTML/web 内容
- unknown agentName 返回安全错误

## 5. 前端验收
- Agent 列表可加载
- 默认 code-agent
- 可切换 web-agent
- Code Agent 消息展示正确
- Web Agent 消息展示正确
- WebPreview Safe Mode 不执行 HTML

## 6. 安全验收
- 不需要 .env
- 不需要真实 API key
- 不泄露内部 URL/token/stack
- frontend 不直连子 Agent
- HTML 不执行 script

## 7. 回滚
- docker compose down
- 不影响旧 docker-compose.yml
- 不影响旧 server/agents
