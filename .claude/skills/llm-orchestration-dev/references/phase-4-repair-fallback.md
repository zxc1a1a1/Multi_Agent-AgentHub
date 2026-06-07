# Phase 4：One-shot Repair + RulePlanner 过渡兜底

## 目标

模型输出非法时，先让 LLM 修复一次，而不是直接走 RulePlanner。

## 允许修改

```text
services/orchestrator/planner/repair.go
services/orchestrator/planner/llm_planner.go
services/orchestrator/planner/rule_planner.go
services/orchestrator/planner/*_test.go
```

## Repair 触发条件

必须触发 repair：

```text
parse failed
normalizer failed
validator failed
unknown agent
invalid mode
empty steps
empty input
parallel has depends_on
unsupported sequential
```

不触发 repair，直接 fallback/error：

```text
LLM API call failed
context canceled
timeout
no available agents
registry empty
```

## Repair 输入

repair prompt 必须包含：

```text
original raw output
parse error or validation errors
available agents
schema
instruction: return corrected JSON only
```

## Repair 次数

最多一次。

禁止：

```text
while 循环不断 repair
repair 失败后再次 repair
repair 失败后静默执行非法 plan
```

## Repair 成功行为

```text
repairCount=1
fallback=false
Source=llm
执行 repaired validated plan
```

## Repair 失败行为

当前阶段：

```text
fallback=true
fallbackReason=repair_failed
Source=rule
调用 RulePlanner transitional fallback
```

后续删除 RulePlanner 时再改为 PlanningError。

## RulePlanner 要求

必须加注释：

```go
// Deprecated: RulePlanner is a transitional fallback only.
// Do not add new routing rules. It will be removed after LLMPlanner
// validation and repair are stable.
```

禁止新增关键词或路由规则。

## 测试命令

```bash
cd services/orchestrator
go test ./planner -run 'TestLLMPlanner_InvalidJSON|TestLLMPlanner_UnknownAgent|TestLLMPlanner_RepairFail|TestPlanRepairer|TestRulePlannerDeprecated' -v
```

## Phase Report 必须包含

```text
Repair 触发条件
Repair prompt 输入
Repair max once 证据
Repair success 行为
Repair fail fallback 行为
RulePlanner Deprecated 注释
确认没有新增 RulePlanner keyword
测试命令和结果
git diff --name-only
forbidden path 确认
```

完成后停止，等待确认。
