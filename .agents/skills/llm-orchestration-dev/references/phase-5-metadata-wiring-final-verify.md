# Phase 5：Metadata / Wiring / Final Verify

## 目标

把 Planner 接入 Orchestrator 初始化，并只追加可选 planner metadata，不破坏前端和 Gateway。

## 允许修改

```text
services/orchestrator/httpapi/**
services/orchestrator/cmd/**
services/orchestrator/planner/** 仅限收尾修复
```

## httpapi 限制

只允许在 `run_started` 或 `state_update` 的 state 中追加可选字段：

```json
{
  "plannerSource": "llm",
  "plannerModel": "claude-sonnet",
  "strategy": "ordered_parallel",
  "taskCount": 2,
  "intent": "build login page and Go API",
  "repairCount": 0,
  "fallback": false,
  "fallbackReason": "",
  "validationPassed": true
}
```

禁止：

```text
改变 event type
改变 SSE 格式
删除已有 state 字段
改变 tool call event 顺序
改变 text message event 顺序
改变 /api/chat
修改 services/gateway
```

## cmd 限制

只允许：

```text
planner wiring
配置 LLMPlanner primary + RulePlanner deprecated fallback
必要 env/config 读取
```

禁止：

```text
改服务通信协议
改 gRPC/HTTP 拓扑
改 Docker
```

## 最终测试

```bash
cd services/orchestrator
go list ./...
go test ./planner ./validator ./plan ./executor ./httpapi -v
```

如果 package 路径不同，以 `go list ./...` 为准。

## 手动验收场景

使用 fake model 或本地测试 harness 验证：

```text
输入：用 Go 写一个 HTTP server，带 logging middleware
期望：plannerSource=llm, fallback=false, strategy=single, agent=code-agent

输入：写一个现代登录页面
期望：plannerSource=llm, fallback=false, strategy=single, agent=web-agent

输入：做登录功能，前端 React 页面，后端 Go 登录接口
期望：plannerSource=llm, fallback=false, strategy=ordered_parallel, agents=[web-agent, code-agent]

输入：非法 JSON
期望：repairCount=1, fallback=false 或 repair fail 时 fallback=true

输入：unknown agent
期望：repair 修复为可用 agent
```

## Final Report 必须包含

```text
git diff --stat
git diff --name-only
所有修改文件说明
所有测试命令和结果
三类正常场景证据
invalid JSON repair 证据
unknown agent repair 证据
repair fail fallback 证据
forbidden paths 未修改确认
RulePlanner 仅 deprecated fallback 确认
风险和后续建议
```
