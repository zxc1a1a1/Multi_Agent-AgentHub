# 01 范围边界：允许和禁止修改

## 允许修改

只允许修改：

```text
services/orchestrator/planner/**
services/orchestrator/validator/**
services/orchestrator/plan/**
services/orchestrator/httpapi/**
services/orchestrator/cmd/**
```

### 允许修改说明

`services/orchestrator/planner/**`

```text
实现 LLMPlanner、Parser、Prompt、Schema、Normalizer、Repairer、Trace。
```

`services/orchestrator/validator/**`

```text
实现或扩展确定性 plan 校验。
```

`services/orchestrator/plan/**`

```text
仅允许补充 plan metadata、strategy 常量、结构体字段兼容。
禁止大改 executor 协议。
```

`services/orchestrator/httpapi/**`

```text
只允许追加 planner metadata 到 run_started/state_update。
禁止改变 event type、SSE 结构、消息顺序。
```

`services/orchestrator/cmd/**`

```text
只允许调整 planner wiring、配置读取、初始化。
禁止改服务通信协议。
```

## 禁止修改

严禁修改：

```text
frontend/**
services/gateway/**
docker-compose*
pkg/adk/**
pkg/runtime/agui/**
server/**
agents/**
```

默认也不改：

```text
pkg/runtime/**
pkg/runtime/session/**
pkg/runtime/model/**
```

除非当前 orchestrator 包本身无法编译，且必须在报告中说明原因、文件、最小变更。

## 硬性失败条件

出现以下情况，审核默认不通过：

```text
改了 frontend/**
改了 services/gateway/**
改了 docker-compose*
改了 pkg/adk/**
改了 pkg/runtime/agui/**
改了旧 server/**
改了旧 agents/**
增强 RulePlanner 关键词规则
改变 /api/chat
改变 AG-UI event type
让 LLM 输出绕过 Validator 直接执行
Repairer 无限循环或多次自动重试
测试依赖真实 API key
```

## 每个 Phase 必跑检查

```bash
git diff --name-only
```

如输出包含 forbidden path，必须立即停止并说明。
