# Phase 1：Schema / Parser / Prompt / Trace

## 目标

建立 LLM 编排输出的稳定合同和基础工具，但暂不接入主执行流程。

## 允许修改

```text
services/orchestrator/planner/schema.go
services/orchestrator/planner/parser.go
services/orchestrator/planner/prompt.go
services/orchestrator/planner/trace.go
services/orchestrator/planner/*_test.go
```

如果项目已有同类文件，可在已有文件中补充，但必须保持职责清楚。

## 禁止修改

```text
services/orchestrator/httpapi/**
services/orchestrator/cmd/**
RulePlanner 行为
Executor 行为
frontend/**
services/gateway/**
```

## 1. Schema 要求

LLM 输出必须遵循：

```json
{
  "intent": "brief summary of the user's goal",
  "mode": "single",
  "confidence": 0.86,
  "steps": [
    {
      "id": "step-1",
      "agent_name": "code-agent",
      "input": "Generate a Go HTTP server with middleware.",
      "depends_on": [],
      "reason": "The user requested backend code."
    }
  ],
  "user_visible_summary": "I will ask code-agent to generate the Go server."
}
```

字段标准：

```text
intent：必填，用户意图简述
mode：必填，只允许 single / parallel / sequential
confidence：可选，范围 0..1，不能作为唯一执行依据
steps：必填非空
steps[].id：可选，缺失时后续 normalizer 生成
steps[].agent_name：必填，后续必须来自 registry
steps[].input：必填，给 agent 的明确任务
steps[].depends_on：可选，nil 后续归一为空数组
steps[].reason：可选
user_visible_summary：可选，不得包含内部信息
```

## 2. Parser 要求

Parser 必须支持：

```text
纯 JSON object
```json fenced JSON
``` fenced JSON
前后空白
前后少量说明文字时提取第一个 JSON object
```

Parser 必须拒绝：

```text
空字符串
非 JSON
JSON array root
缺少 mode
缺少 steps
```

Parser 禁止：

```text
调用 RulePlanner
猜默认 plan
修改 agent_name
吞掉 parse error
```

## 3. Prompt 要求

Prompt 必须包含：

```text
AgentHub planner role
available agents
recent history
current user message
schema
rules
```

Agent 信息必须以 registry / AgentCard 等真实来源为主，不得主要依赖 `agentDefaults()`。

Prompt 禁止包含：

```text
API key
内部服务 URL
数据库 DSN
system prompt
用户不可见内部配置
```

## 4. Trace 要求

定义 PlannerTrace，至少包含：

```text
Source: llm | rule
Model
Provider
Fallback
FallbackReason
RepairCount
ParseError
ValidationErrors
Intent
Mode
TaskCount
Agents
LatencyMS
```

Trace 可用于日志和 state metadata，但禁止把 raw prompt、secret、内部 URL 暴露给前端。

## 测试要求

```bash
cd services/orchestrator
go list ./...
go test ./planner -run 'TestPlanParser|TestPrompt|TestPlannerTrace' -v
```

如 package 路径不同，用 `go list` 输出的实际路径替换。

## Phase Report 必须包含

```text
新增/修改文件
Schema 字段说明
Parser 支持/拒绝情况
Prompt agent 信息来源
Trace 字段
测试命令和结果
git diff --name-only
forbidden path 确认
```

完成后停止，等待确认。
