# Phase 2：Normalizer / Validator

## 目标

把 LLM 输出转为内部 OrchestrationPlan，并用确定性 Validator 拦截不可执行计划。

## 允许修改

```text
services/orchestrator/planner/normalizer.go
services/orchestrator/validator/**
services/orchestrator/plan/**
services/orchestrator/planner/*_test.go
services/orchestrator/validator/*_test.go
```

## 禁止修改

```text
Executor 大改
httpapi event 行为
RulePlanner 规则
frontend/**
services/gateway/**
docker-compose*
```

## Normalizer 标准

Normalizer 负责“翻译”，不是“猜测”。

必须做：

```text
trim string fields
step id 缺失时生成 step-1、step-2
depends_on nil 归一为空数组
confidence 缺失设为 0
LLM mode=single 映射 internal single
LLM mode=parallel 映射 internal ordered_parallel
LLM mode=sequential 标记为 unsupported 或交给 validator 拒绝
```

禁止做：

```text
未知 agent 自动改成 code-agent
根据关键词改 agent
删除非法 step
生成默认兜底 plan
调用 RulePlanner
```

## Validator 标准

Validator 必须是确定性代码，不调用 LLM。

必须校验：

```text
mode/strategy legal
steps non-empty
single exactly one step
parallel at least two steps
parallel no depends_on
sequential rejected until executor/httpapi fully supports it
step id unique
depends_on references existing step
depends_on no self-reference
agent_name exists and is available in registry
input non-empty
confidence in [0,1]
no internal URL
no API key/token-like secret
no DB DSN
no system prompt leakage
```

建议结构化错误：

```go
type ValidationError struct {
    Field   string
    Code    string
    Message string
}
```

## 测试要求

```bash
cd services/orchestrator
go test ./planner ./validator ./plan -run 'TestNormalize|TestPlanValidator' -v
```

必须覆盖：

```text
unknown agent rejected
empty steps rejected
single with multiple steps rejected
parallel with depends_on rejected
sequential rejected until enabled
empty input rejected
confidence out of range rejected
internal URL rejected
secret-like token rejected
```

## Phase Report 必须包含

```text
Normalizer 映射规则
sequential 当前处理方式
Validator 校验规则
测试命令和结果
git diff --name-only
forbidden path 确认
```

完成后停止，等待确认。
