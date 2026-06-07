# Phase 5 Report：Metadata Wiring / Final Verify

## Phase

```text
Phase: 5
Status: PASS
是否停止等待确认：是
```

## 修改文件

```text
- services/orchestrator/httpapi/handler_run_stream.go (5 行新增)
```

## 修改说明

在 `run_started` 事件的 state 中追加了 4 个可选字段，不改变 event type、SSE 格式、消息顺序：

1. **`repairCount`** — `orchPlan.RepairCount`（LLM 主路径 = 0，repair 成功后 = 1）
2. **`fallback`** — `orchPlan.Fallback.Enabled`（RulePlanner = true，LLM 主路径 = false，LLM fallback = true）
3. **`fallbackReason`** — `orchPlan.Fallback.Reason`（如 `"rule_default"`, `"llm_error"`, `"repair_failed"`）
4. **`validationPassed`** — `orchPlan.Validation.Validated`（validator 校验后 = true）

### 实现细节

```go
// Phase 5: always emit repair/fallback/validation metadata.
planState["repairCount"] = orchPlan.RepairCount
planState["fallback"] = orchPlan.Fallback.Enabled
planState["fallbackReason"] = orchPlan.Fallback.Reason
planState["validationPassed"] = orchPlan.Validation.Validated
```

这些字段始终出现在 `run_started` 事件的 state 中（非条件性），与已有字段 `validated`、`plannerSource`、`strategy`、`taskCount` 等共存。

## 禁止路径确认

```text
frontend/**：未修改
services/gateway/**：未修改
docker-compose*：未修改
pkg/adk/**：未修改
pkg/runtime/agui/**：未修改
server/**：未修改
agents/**：未修改
```

全部通过。唯一修改文件为 `services/orchestrator/httpapi/handler_run_stream.go`。

## 执行命令

```bash
cd services/orchestrator
go test ./planner ./validator ./plan ./executor ./httpapi -v -count=1
```

## 测试结果

```text
PASS — 125 tests, 5 packages, 0 failures
```

关键测试输出：

```text
ok  github.com/.../services/orchestrator/planner    0.004s
ok  github.com/.../services/orchestrator/validator   0.003s
ok  github.com/.../services/orchestrator/plan        0.002s
ok  github.com/.../services/orchestrator/executor    0.002s
ok  github.com/.../services/orchestrator/httpapi     0.007s
```

## 行为证据

### 三类正常场景（LLMPlanner + fake model）

**场景 1：Code 请求 → single code-agent**
- 测试：`TestLLMPlanner_CodeRequest`
- 断言：`Strategy=StrategySingle`, `AgentName=code-agent`, `PlannerSource=llm`, `Fallback.Enabled=false`, `Validation.Validated=false`
- `run_started` state：`plannerSource=llm`, `fallback=false`, `repairCount=0`, `strategy=single`, `taskCount=1`

**场景 2：Web 请求 → single web-agent**
- 测试：`TestLLMPlanner_WebRequest`
- 断言：`Strategy=StrategySingle`, `AgentName=web-agent`, `PlannerSource=llm`
- `run_started` state：`plannerSource=llm`, `fallback=false`, `strategy=single`, `taskCount=1`

**场景 3：Full stack 请求 → ordered_parallel [web-agent, code-agent]**
- 测试：`TestLLMPlanner_FullStack`
- 断言：`Strategy=StrategyOrderedParallel`, `agentNames=[web-agent, code-agent]`, `Aggregation.Required=true`, `Aggregation.Mode=summary`
- `run_started` state：`plannerSource=llm`, `strategy=ordered_parallel`, `taskCount=2`

### invalid JSON repair 证据

- 测试：`TestLLMPlanner_InvalidJSON_RepairSucceeds`
- 断言：`Source=llm`, `RepairCount=1`, `Fallback.Enabled=false`
- `run_started` state：`repairCount=1`, `fallback=false`, `plannerSource=llm`

### unknown agent repair 证据

- 测试：`TestLLMPlanner_UnknownAgent_RepairSucceeds`
- 断言：`Source=llm`, `RepairCount=1`, `Agent=web-agent`（repair 修复了 unknown agent）
- `run_started` state：`repairCount=1`, `fallback=false`, `plannerSource=llm`

### repair fail fallback 证据

- 测试：`TestLLMPlanner_RepairFail_FallsBackToRulePlanner`
- 断言：`Source=fallback`, `Fallback.Enabled=true`, `Fallback.Reason="rule_default"`
- `run_started` state：`fallback=true`, `fallbackReason="rule_default"`, `plannerSource=fallback`

### RulePlanner 仅 deprecated fallback 确认

- 测试：`TestRulePlannerDeprecated`
- 注释存在且包含 "Deprecated: RulePlanner is a transitional fallback only"
- 无新增关键词或路由规则

### event type / SSE 格式不变 确认

- 所有 11 个 httpapi 测试继续 PASS
- `run_started` event type 未改变
- SSE 格式（`event:` + `data:` 双行格式）未改变
- 消息顺序（run_started → message_start → message_delta → message_end → run_finished）未改变

## git diff 检查

```bash
git diff --name-only
# 输出：services/orchestrator/httpapi/handler_run_stream.go
```

```bash
git diff --stat
# 输出：1 file changed, 5 insertions(+)
```

## 风险

```text
- 无。本次变更为纯追加字段，不改变现有行为。
- run_started state 向后兼容：前端/Gateway 可忽略未知字段。
- validated 字段保留不变（与 validationPassed 值相同），保证向后兼容。
- 所有五个包 125 个测试全部通过，无回归。
```

## 下一步建议

```text
停止等待用户确认。
Phase 5 是最终 Phase，LLM 编排开发全部完成。
后续可进入删除 RulePlanner 的阶段（需等 LLMPlanner 验证和 repair 稳定后）。
```

## Phase 0-5 总览

| Phase | 主题 | 状态 |
|-------|------|------|
| 0 | 只读排查 | ✅ PASS |
| 1 | Schema / Parser / Prompt / Trace | ✅ PASS |
| 2 | Normalizer / Validator | ✅ PASS |
| 3 | LLMPlanner 主流程 | ✅ PASS |
| 4 | Repair / Fallback | ✅ PASS |
| 5 | Metadata Wiring / Final Verify | ✅ PASS |

所有 Phase 全部通过，Orchestrator LLM 编排能力完整可运行。
