# Phase 3 Report：LLMPlanner 主流程

## Phase

```text
Phase: 3 — LLMPlanner Main Flow
Status: PASS
是否停止等待确认：是
```

## 修改文件

```text
- services/orchestrator/cmd/orchestrator/main.go       (+26 lines)
- services/orchestrator/planner/llm_planner.go         (rewritten Plan(), +interface)
- services/orchestrator/planner/llm_planner_test.go    (+566 lines, 8 new tests)
```

## 禁止路径确认

逐项确认：

```text
frontend/**          ：未修改
services/gateway/**  ：未修改
docker-compose*      ：未修改
pkg/adk/**           ：未修改
pkg/runtime/agui/**  ：未修改
server/**            ：未修改
agents/**            ：未修改
```

## 本 Phase 实现内容

### 1. PlannerModel 抽象接口

```go
type PlannerModel interface {
    Generate(ctx context.Context, systemPrompt, userPrompt string) (string, error)
}
```

- `PlannerLLM` 已自动满足此接口（签名完全匹配）
- 测试使用 `fakeModel`，零依赖真实 API key
- 接口不与具体 provider 绑定

### 2. LLMPlanner 新主流程

```
PromptBuilder(registry agents)
  → PlannerModel.Generate()
  → PlanParser.Parse()
  → PlanNormalizer.Normalize()
  → Stamp PlannerSource="llm"
  → Return plan
```

- 任何失败（model error, parse error, normalize error）→ fallback to RulePlanner
- `PlannerSource="llm"` 标记 LLM 规划成功
- `Fallback.Enabled=false` 标记主路径完成

### 3. 不做 Repairer

Phase 3 明确不实现 Repairer。Parse/Normalize 失败直接回退 RulePlanner。

### 4. 旧代码标记为 legacy

- `parsePlanResponse` — `// Deprecated: new pipeline uses PlanParser → PlanSchema`
- `convertToPlan` — `// Deprecated: new pipeline uses PlanNormalizer`
- `fuzzyMatchAgent` — `// Deprecated: new pipeline preserves unknown agent names as-is`
- `defaultAgent` — `// Deprecated: new pipeline preserves unknown agent names as-is`
- `buildSystemPrompt` / `buildUserPrompt` — `// Deprecated: new pipeline uses PromptBuilder`
- 旧代码不删除，保持向后兼容

### 5. cmd/orchestrator/main.go 最小改动

新增 `registryAgentLister` adapter（15 行），将 `StaticAgentRegistry` 适配为 `planner.AgentLister`：

```go
type registryAgentLister struct {
    reg *registry.StaticAgentRegistry
}
func (a *registryAgentLister) List() []planner.AgentInfoLite { ... }
```

构造 LLMPlanner 调用变更：
```go
// Before:
llmPlanner := planner.NewLLMPlanner(llmClient, agentRegistry.Names())
// After:
adapter := &registryAgentLister{reg: agentRegistry}
llmPlanner := planner.NewLLMPlanner(llmClient, llmCfg.Model, adapter)
```

## 执行命令

```bash
# Run all orchestrator tests
cd services/orchestrator && go test ./... -count=1 -v

# Run Phase 3 specific tests
go test ./planner -run 'TestLLMPlanner_CodeRequest|TestLLMPlanner_WebRequest|TestLLMPlanner_FullStack|TestLLMPlanner_DoesNotUseRulePlanner|TestLLMPlanner_UsesRegistryAgents|TestLLMPlanner_UnknownAgentNotFuzzyMatched|TestLLMPlanner_ModelErrorFallsBackToRulePlanner|TestLLMPlanner_ParseErrorFallsBack' -v
```

## 测试结果

```text
PASS — 所有测试通过
```

完整 orchestrator 测试套件：

```text
ok  github.com/.../orchestrator/planner       0.003s (94 tests PASS)
ok  github.com/.../orchestrator/validator     0.003s
ok  github.com/.../orchestrator/httpapi       0.007s
ok  github.com/.../orchestrator/plan          0.002s
ok  github.com/.../orchestrator/registry      0.002s
ok  github.com/.../orchestrator/executor      0.002s
ok  github.com/.../orchestrator/dispatcher    30.034s
```

### Phase 3 8个新测试全部通过

| 测试 | 结果 | 说明 |
|------|------|------|
| TestLLMPlanner_CodeRequest | PASS | Go/backend → single code-agent |
| TestLLMPlanner_WebRequest | PASS | web/UI → single web-agent |
| TestLLMPlanner_FullStack | PASS | full-stack → parallel web-agent + code-agent |
| TestLLMPlanner_DoesNotUseRulePlanner | PASS | PlannerSource=llm, Fallback.Enabled=false |
| TestLLMPlanner_UsesRegistryAgents | PASS | prompt 使用 registry 信息而非 agentDefaults |
| TestLLMPlanner_UnknownAgentNotFuzzyMatched | PASS | 未知 agent 保留原样 |
| TestLLMPlanner_ModelErrorFallsBackToRulePlanner | PASS | API 故障回退到 RulePlanner |
| TestLLMPlanner_ParseErrorFallsBack | PASS | 无效 JSON 回退到 RulePlanner |

### 零真实 API key 依赖确认

所有 LLM 测试使用 `fakeModel`（内存中返回预设 JSON），不依赖任何环境变量或网络调用。

## 行为证据

### Parser：

- PlanParser 正确解析有效 JSON（plain、markdown-fenced、leading/trailing whitespace）
- 拒绝空输入、非JSON、数组根、残缺JSON、缺少mode/steps
- 所有 Parser 测试通过（22 tests）

### Normalizer：

- LLM mode=single → plan.StrategySingle
- LLM mode=parallel → plan.StrategyOrderedParallel (compat mapping)
- LLM mode=sequential → plan.StrategySequential (validator rejects)
- Unknown agent 保留原样，不做 fuzzyMatch/defaultAgent
- Confidence [0,1] 通过，超出范围拒绝
- 所有 Normalizer 测试通过（18 tests）

### Validator：

- 由 httpapi 层在 Plan() 返回后调用
- PlanValidator 检查 registry agent 存在性、capability 匹配、output type 匹配
- sequential strategy 被显式拒绝
- 所有 Validator 测试通过

### LLMPlanner：

- 新 `Plan()` 使用 PromptBuilder → PlannerModel → PlanParser → PlanNormalizer 管线
- 成功路径：PlannerSource="llm", Fallback.Enabled=false
- 失败路径：自动回退 RulePlanner，PlannerSource="fallback"

### Repairer：

- **不实现**（按 Phase 3 要求）

### RulePlanner fallback：

- LLM API 调用失败 → fallbackPlan("llm_error")
- Parse 失败（无效JSON、非JSON、解析错误）→ fallbackPlan("parse_error")
- Normalize 失败（未知 mode、confidence 超范围）→ fallbackPlan("normalize_error")
- FallbackPlan → RulePlanner.Plan() → 如果 RulePlanner 也失败 → ultimate fallback (code-agent)

### State metadata：

- PlannerSource 在 Plan() 中设置为 "llm"
- PlannerModel 从构造时传入的 modelName 获取
- PlannerReasoning 使用 schema.Intent 截取
- Fallback.Enabled 在成功路径设为 false

## 证据 1：valid LLM plan 不触发 RulePlanner

**测试：** `TestLLMPlanner_DoesNotUseRulePlanner`

```
=== RUN   TestLLMPlanner_DoesNotUseRulePlanner
    ✓ Valid LLM plan does not trigger RulePlanner:
      Source=llm, Fallback.Enabled=false, Model=test-model
--- PASS
```

证据链：
1. `PlannerSource == "llm"` — 确认 LLM 管线产出，非 rule/fallback
2. `Fallback.Enabled == false` — 确认未触发 RulePlanner fallback
3. `PlannerModel == "test-model"` — 确认 LLM metadata 写入
4. 结果中没有 `PlannerSource="rule"` 或 `"fallback"`

## 证据 2：unknown agent 不再走 fuzzyMatchAgent/defaultAgent

**测试：** `TestLLMPlanner_UnknownAgentNotFuzzyMatched`

```
=== RUN   TestLLMPlanner_UnknownAgentNotFuzzyMatched
    ✓ Unknown agent 'gibberish-agent-xyz' preserved as-is
      (no fuzzyMatchAgent/defaultAgent)
--- PASS
```

证据链：
1. LLM 返回 `"agent_name": "gibberish-agent-xyz"`（不在 registry 中）
2. Normalizer 保留原样输出 `AgentName = "gibberish-agent-xyz"`
3. 不是 `"code-agent"`（fuzzyMatchAgent 没触发）
4. 不是 `"web-agent"`（defaultAgent 没触发）
5. PlannerSource 仍是 "llm"（主路径成功，agent 验证留给 Validator）

对比旧路径（`convertToPlan` 的 `TestLLMPlanner_ConvertToPlan_UnknownAgent`）：
```go
// 旧路径：web-agent unknown → defaultAgent → code-agent
orchPlan.Tasks[0].AgentName != "code-agent" // FAILS test (correctly)
```

## git diff 检查

```bash
git diff --name-only
```

输出：

```text
services/orchestrator/cmd/orchestrator/main.go
services/orchestrator/planner/llm_planner.go
services/orchestrator/planner/llm_planner_test.go
```

```bash
git diff --stat
```

输出：

```text
services/orchestrator/cmd/orchestrator/main.go    |  26 +-
services/orchestrator/planner/llm_planner.go      | 128 +++--
services/orchestrator/planner/llm_planner_test.go | 566 ++++++++++++++++-
3 files changed, 671 insertions(+), 49 deletions(-)
```

**仅修改 3 个文件，全部在允许路径内。**

## 风险

- `cmd/orchestrator/main.go` 新增 adapter 类型，生产环境需确保 `StaticAgentRegistry` 正确注入
- RulePlanner 仍然是 fallback，后续 Phase 需要删除（Phase 5）
- Validator 对 unknown agent 的拒绝行为依赖 httpapi 层的 PlanValidator 调用
- FakeModel 仅覆盖单次调用场景，多轮对话/streaming 不在 Phase 3 范围

## 下一步建议

```text
停止等待审核 — Phase 3 完成，等待用户确认后进入 Phase 4（Repairer + Fallback）
```

用户额外注意事项：
- Phase 4 做 Repairer（但用户说"不做 Repairer"，因此 Phase 4 可能需要跳过）
- Phase 5 做 Metadata wiring + final verify
