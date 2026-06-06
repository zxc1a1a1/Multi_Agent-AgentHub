# Phase 3 Fix Report：LLMPlanner 内嵌 Validator

## Phase

```text
Phase: 3 Fix — LLMPlanner 主路径嵌入 PlanValidator.Validate
Status: PASS
是否停止等待确认：是
```

## 问题

Phase 3 初版 `LLMPlanner.Plan()` 在 normalize 后直接 stamp metadata 并返回，**未调用 PlanValidator.Validate**。unknown agent 只有在 httpapi 层的 Validator 才能被拒绝，LLMPlanner 内部不做校验。

## 修复内容

在 `LLMPlanner.Plan()` 主路径 normalize 之后、metadata stamp 之前插入 **Step 5: Validate**：

```text
PromptBuilder → PlannerModel → PlanParser → PlanNormalizer
  → PlanValidator → (pass) → stamp metadata → return
                  → (fail) → RulePlanner fallback
```

### 1. PlanValidator 接口（planner 包内）

```go
type PlanValidator interface {
    Validate(p *plan.OrchestrationPlan) bool
}
```

- 定义在 `planner` 包内，避免对外部 validator 包产生依赖
- 返回 bool 表示 pass/fail，LLMPlanner 只需要知道是否通过

### 2. listerPlanValidator 实现

```go
type listerPlanValidator struct {
    lister AgentLister
}
```

验证规则：
- 每个 task 的 agentName 必须在 registry（lister）中存在
- agentName 不能为空
- tasks 不能为空
- 拒绝 sequential strategy

**不做的：**
- 不调用 `fuzzyMatchAgent`
- 不调用 `defaultAgent`
- 不做 Repairer（按用户要求）
- 不检查 capability/output 匹配（留给 httpapi Validator）

### 3. 自动注入

`NewLLMPlanner` 内部从 `AgentLister` 自动创建 `listerPlanValidator`：

```go
return &LLMPlanner{
    ...
    validator: newListerPlanValidator(lister),
    ...
}
```

**构造函数签名不变**，cmd/orchestrator/main.go **不需要任何修改**。

### 4. 主路径新增验证步骤

```go
// 5. Validate the normalized plan against registry agents.
if p.validator != nil && !p.validator.Validate(orchPlan) {
    log.Printf("llm_planner: validation failed, falling back to RulePlanner")
    return p.fallbackPlan(input, "validation_error", p.modelName), nil
}
```

验证失败 → `fallbackPlan("validation_error")` → RulePlanner。

## 修改文件

```text
- services/orchestrator/planner/llm_planner.go      (+83 lines: interface, impl, validate step)
- services/orchestrator/planner/llm_planner_test.go  (+70 lines: fix + new test)
- services/orchestrator/cmd/orchestrator/main.go     (unchanged since Phase 3)
```

## 未修改确认

```text
cmd/orchestrator/main.go   ：无需修改（validator 自动注入）
httpapi/                   ：未修改
frontend/**                ：未修改
services/gateway/**        ：未修改
docker-compose*            ：未修改
pkg/adk/**                 ：未修改
pkg/runtime/agui/**        ：未修改
server/**                  ：未修改
agents/**                  ：未修改
```

## 测试命令和结果

```bash
cd services/orchestrator && go test ./... -count=1
```

```text
ok  planner       0.003s  (95 tests PASS)
ok  validator     0.003s
ok  httpapi       0.006s
ok  plan          0.002s
ok  registry      0.002s
ok  executor      0.003s
ok  dispatcher    30.034s
```

### Phase 3 Fix 关键测试结果

#### 证据 1：valid LLM plan 通过 parse → normalize → validate，不触发 RulePlanner

```
TestLLMPlanner_DoesNotUseRulePlanner  PASS
  ✓ Valid LLM plan does not trigger RulePlanner:
    Source=llm, Fallback.Enabled=false, Model=test-model

TestLLMPlanner_CodeRequest           PASS  (single code-agent)
TestLLMPlanner_WebRequest            PASS  (single web-agent)
TestLLMPlanner_FullStack             PASS  (parallel web+code)
```

#### 证据 2：unknown agent 被 Validator 拒绝（不做 fuzzy/default），fallback RulePlanner

```
TestLLMPlanner_UnknownAgentNotFuzzyMatched  PASS
  llm_planner: validation reject: agent "gibberish-agent-xyz" not in registry
  llm_planner: validation failed, falling back to RulePlanner
  ✓ Unknown agent rejected by Validator → RulePlanner fallback:
    Source=fallback, Agent=code-agent
```

关键：gibberish-agent-xyz **从未被 fuzzyMatchAgent/defaultAgent 改写**。
Normalizer 保留了它 → Validator 拒绝了它 → RulePlanner fallback 用了已知 agent。

#### 证据 3：validation fail → fallback（新测试）

```
TestLLMPlanner_ValidationFailFallsBack  PASS
  llm_planner: validation reject: agent "nonexistent-agent" not in registry
  llm_planner: validation failed, falling back to RulePlanner
  ✓ Validation fail → RulePlanner fallback: Source=fallback, Agent=code-agent
```

#### 证据 4：其他 fail path 不变

```
TestLLMPlanner_ModelErrorFallsBackToRulePlanner  PASS
  ✓ Model error correctly triggers deprecated RulePlanner fallback

TestLLMPlanner_ParseErrorFallsBack               PASS
  ✓ Parse error correctly triggers deprecated RulePlanner fallback
```

## 完整 pipeline 流程图

```
LLMPlanner.Plan()
  │
  ├─[1] PromptBuilder.BuildSystemPrompt / BuildUserPrompt
  │      └─ 来自 AgentLister（registry）的 agent info
  │
  ├─[2] PlannerModel.Generate()
  │      └─ 失败 → fallbackPlan("llm_error") → RulePlanner
  │
  ├─[3] PlanParser.Parse()
  │      └─ 失败 → fallbackPlan("parse_error") → RulePlanner
  │
  ├─[4] PlanNormalizer.Normalize()
  │      └─ 保留 unknown agent（不做 fuzzy/default）
  │      └─ 失败 → fallbackPlan("normalize_error") → RulePlanner
  │
  ├─[5] PlanValidator.Validate()          ← NEW（本次修复）
  │      └─ 检查 agent 名称在 registry 中存在
  │      └─ 拒绝 unknown agent（不 fuzzy/default）
  │      └─ 失败 → fallbackPlan("validation_error") → RulePlanner
  │
  └─[6] Stamp PlannerSource="llm", Fallback.Enabled=false → return
```

## 行为证据

| 行为 | 旧版（Phase 3 初版） | 新版（Phase 3 Fix） |
|------|----------------------|---------------------|
| valid agent plan | return llm plan | return llm plan ✓ |
| unknown agent | return llm plan (agent preserved) | fallback RulePlanner ✓ |
| fuzzyMatchAgent | 不调用 ✓ | 不调用 ✓ |
| defaultAgent | 不调用 ✓ | 不调用 ✓ |
| Repairer | 不做 ✓ | 不做 ✓ |
| RulePlanner fallback | 仅 model/parse/normalize fail | + validation fail ✓ |

## 不修改的确认

- `cmd/orchestrator/main.go` — `NewLLMPlanner` 签名不变，validator 自动注入
- `httpapi/` — 不受影响，httpapi 层 Validator 仍做二次校验
- RulePlanner 关键词 — 未新增任何关键词
- 旧 `parsePlanResponse` / `convertToPlan` / `fuzzyMatchAgent` / `defaultAgent` — 全部保留，标为 legacy

## 风险

- `listerPlanValidator` 只做 agent name existence 和基本结构检查，不做 capability/output 匹配（留给 httpapi Validator）
- 如果 lister 为 nil，validator 也为 nil，validation 步骤被跳过（保持向后兼容）
- httpapi 层仍会二次调用 `validator.PlanValidator.Validate`，形成双重校验（通过 LLMPlanner 内部 validator 的 plan 也会通过 httpapi 的 validator）

## 下一步建议

```text
停止等待确认 — Phase 3 Fix 完成，等待用户确认。
不做 Phase 4（Repairer），可考虑直接进入 Phase 5（Metadata wiring final verify）。
```
