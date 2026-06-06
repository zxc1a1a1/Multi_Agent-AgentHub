# Phase 3：LLMPlanner 主流程

## 目标

让有效 LLM 输出成为主路径：LLMPlanner 调模型、解析、归一化、校验，通过后返回 plan。

## 允许修改

```text
services/orchestrator/planner/llm_planner.go
services/orchestrator/planner/planner_llm.go
services/orchestrator/planner/model.go
services/orchestrator/planner/*_test.go
```

## 禁止修改

```text
RulePlanner 规则
httpapi event 行为
frontend/**
services/gateway/**
pkg/runtime/model/**
```

除非已有代码已经通过 `pkg/runtime/model` 注入模型，不能为了本任务大改 runtime provider。

## PlannerModel 抽象

Planner 不应强绑定具体 provider。

建议接口之一：

```go
type PlannerModel interface {
    Generate(ctx context.Context, prompt string) (string, error)
}
```

或：

```go
type PlannerModel interface {
    GeneratePlan(ctx context.Context, req PlannerModelRequest) (*PlannerModelResponse, error)
}
```

测试必须用 fake model，不依赖真实 API key。

## 主流程

必须实现：

```text
read available agents from registry
build prompt
call PlannerModel
parse raw output
normalize
validate
if valid return plan
trace Source=llm
```

有效 LLM plan 禁止调用 RulePlanner。

## 必测场景

```text
Go/backend request -> single code-agent
web/UI request -> single web-agent
full-stack request -> parallel/ordered_parallel with web-agent + code-agent
valid LLM plan -> fallback=false
prompt uses registry agents
```

## 测试命令

```bash
cd services/orchestrator
go test ./planner -run 'TestLLMPlanner_CodeRequest|TestLLMPlanner_WebRequest|TestLLMPlanner_FullStack|TestLLMPlanner_DoesNotUseRulePlanner|TestLLMPlanner_UsesRegistryAgents' -v
```

## Phase Report 必须包含

```text
LLMPlanner 主流程
PlannerModel 抽象
fake model 测试说明
测试命令和结果
确认无真实 API key 依赖
git diff --name-only
forbidden path 确认
```

完成后停止，等待确认。
