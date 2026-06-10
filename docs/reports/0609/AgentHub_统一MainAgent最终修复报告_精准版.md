# AgentHub 统一 MainAgent 多路径计划确认最终修复报告（精准版）

## 0. 本报告结论

本报告是新版修复方案的最终执行版。它以 **统一 MainAgent + executionPath 边界 + Validator 硬校验 + Orchestrator 串行 dispatch** 为核心。

当前代码不符合新版方案，原因是：

```text
1. single_chat 仍由当前 Agent plan_only 生成方案，没有统一进入 MainAgent。
2. group_chat 仍可能进入旧 LLMPlanner / RulePlanner pipeline。
3. MainAgent 当前只在 main_agent_orchestration 路径被调用。
4. MainAgent 当前能力不足，只是关键词匹配器，不能承担统一 planner。
5. Validator 不是 path-aware，不能强制 availableBoundary。
6. Confirm 链路仍有 selectedParticipants 丢失、502、idempotencyKey 不完整等问题。
```

旧审计文档保留，但只作为历史问题清单，不再作为新版架构蓝图。

本轮修复目标：

```text
MainAgent 统一生成计划；
executionPath 决定 MainAgent 可用 Agent 边界；
Validator 强制边界；
ActivitySnapshot 展示计划确认卡片；
用户确认后 Orchestrator 串行 dispatch；
strategy 字段保留，但不阻断执行。
```

---

## 1. MainAgent 的准确定位

### 1.1 MainAgent 是什么

MainAgent 是 **Orchestrator 内部的 LLM planner 组件**。

它不是普通 Agent，不是外部独立 Agent 服务，也不是前端可选 Agent。

推荐位置：

```text
services/orchestrator/planner/main_agent.go
```

或后续更清晰地拆成：

```text
services/orchestrator/mainagent/planner.go
services/orchestrator/mainagent/prompt.go
services/orchestrator/mainagent/schema.go
```

### 1.2 MainAgent 不是什么

MainAgent 不应：

```text
1. 不注册进 StaticAgentRegistry。
2. 不出现在前端 Agent 下拉框。
3. 不被 health checker 当作普通 Agent 探测。
4. 不通过 A2A dispatcher 作为普通子 Agent 调用。
5. 不直接调用 code-agent / web-agent / test-agent 执行。
6. 不绕过 Orchestrator 的 confirm lifecycle。
```

### 1.3 MainAgent 与 Orchestrator 的关系

```text
Orchestrator
  ├── deriveExecutionPath
  ├── buildAvailableBoundary
  ├── MainAgentPlanner        ← MainAgent 在这里
  ├── PathAwareValidator
  ├── PendingPlan / HITL
  ├── Confirm / Revision / Cancel
  └── Dispatcher
        ├── code-agent
        ├── web-agent
        ├── review-agent
        └── test-agent
```

MainAgent 负责：

```text
1. 调用大模型理解用户需求。
2. 基于 executionPath 和 availableBoundary 生成结构化计划。
3. 输出 participants / steps / reasons / warnings。
4. 根据 feedback 生成 revision + 1。
5. 可选：执行后生成汇总。
```

Orchestrator 负责：

```text
1. 推导 executionPath。
2. 构造 availableBoundary。
3. 调 MainAgent 生成计划。
4. 调 Validator 校验计划。
5. registerPendingPlan。
6. emit ActivitySnapshot。
7. 等用户 approve / revise / cancel。
8. 用户 approve 后串行 dispatch Agent。
```

---

## 2. MainAgent 必须接入大模型

### 2.1 正确理解

MainAgent 应该接入大模型。

本轮不是做“无大模型的规则规划器”，而是做：

```text
LLM + registry capability 约束的 MainAgent
```

意思是：

```text
1. MainAgent 会调用大模型。
2. 调用大模型时，把 registry 中 enabled agents 的能力列表传进去。
3. 同时传入 executionPath 和 availableBoundary。
4. 大模型只能在 availableBoundary 范围内生成计划。
5. 计划输出后由 Validator 再硬校验。
```

### 2.2 不做复杂 LLM planner 是什么意思

“不做复杂 LLM planner”不是说不接大模型。

它的意思是：本轮先不做这些复杂能力：

```text
1. 多候选计划生成和评分。
2. Planner 自我反思 / self-check 多轮修复。
3. 多模型 fallback。
4. 复杂 DAG 推理。
5. 并行调度优化。
6. 长期 planner memory。
7. 成本 / 风险 / 时延综合优化。
8. 自动 repair 多轮重试。
```

本轮只做最小可用：

```text
MainAgent 调一次大模型
输入：userMessage + executionPath + availableBoundary + enabled agents capabilities
输出：严格 JSON 计划
Validator 校验
失败则返回错误或走简单 fallback
```

---

## 3. 严格 JSON 计划与用户看到的计划卡片

### 3.1 严格 JSON 计划是什么

严格 JSON 计划是 **MainAgent 给后端和前端用的结构化数据**。

它不是用户直接看到的文本。

MainAgent 不能只输出自然语言：

```text
我建议先让 code-agent 实现接口，然后让 review-agent 检查。
```

而必须输出固定结构，例如：

```json
{
  "title": "登录接口实现与检查计划",
  "summary": "先由 code-agent 实现登录接口，再由 review-agent 检查。",
  "executionPath": "group_chat",
  "participants": [
    {
      "agentName": "code-agent",
      "required": true,
      "userSelectable": false,
      "reason": "负责实现登录接口"
    },
    {
      "agentName": "review-agent",
      "required": true,
      "userSelectable": false,
      "reason": "负责检查接口设计和安全风险"
    }
  ],
  "defaultSelectedParticipants": ["code-agent", "review-agent"],
  "requiredParticipants": ["code-agent", "review-agent"],
  "steps": [
    {
      "stepId": "step_code",
      "assignedAgent": "code-agent",
      "task": "实现 Go 登录接口、参数校验和错误返回"
    },
    {
      "stepId": "step_review",
      "assignedAgent": "review-agent",
      "task": "检查登录接口设计、错误处理和安全风险"
    }
  ],
  "warnings": [],
  "strategy": "sequential"
}
```

### 3.2 用户看到的是什么

用户看到的不是 JSON，而是前端根据 ActivitySnapshot 渲染出的计划确认卡片。

例如 group_chat 用户看到：

```text
执行前确认

登录接口实现与检查计划

MainAgent 已基于你选择 / @ 的 Agent 生成生产计划。
确认后只会按顺序调用这些 Agent。

计划摘要：
先由 code-agent 实现登录接口，再由 review-agent 检查。

参与 Agent：
✓ code-agent
  必需
  负责实现登录接口

✓ review-agent
  必需
  负责检查接口设计和安全风险

执行步骤：
1. code-agent：实现 Go 登录接口、参数校验和错误返回
2. review-agent：检查登录接口设计、错误处理和安全风险

执行方式：
确认后将按顺序执行。执行前不会调用任何 Agent。

操作：
[同意并执行] [提出修改意见] [取消]
```

### 3.3 JSON 与卡片字段对应关系

```text
JSON title
→ 卡片标题

JSON summary
→ 计划摘要

JSON participants
→ 参与 Agent 区域

JSON participants[].reason
→ 为什么需要这个 Agent

JSON participants[].required
→ 是否必需、是否可取消

JSON steps
→ 执行步骤列表

JSON warnings
→ 风险 / 提示信息

allowedActions
→ 按钮
```

---

## 4. 新版完整链路

```text
POST /api/chat
  ↓
Gateway 只透传请求字段
  ↓
Orchestrator deriveExecutionPath
  ↓
Orchestrator buildAvailableBoundary
      single_chat              → [当前选中 Agent]
      group_chat               → selectedAgentNames / mentions / activeGroupAgents
      main_agent_orchestration → all enabled agents
  ↓
MainAgent.GeneratePlan(userText, executionPath, availableBoundary)
  ↓
PathAwareValidator.Validate(plan, executionPath, availableBoundary)
  ↓
registerPendingPlan
  ↓
emit ACTIVITY_SNAPSHOT
  ↓
等待用户 approve / revise / cancel
  ↓
approve:
    校验 runId / planId / revision / selectedParticipants / idempotencyKey
    Orchestrator 串行 dispatch selectedParticipants
  ↓
Agent 独立 turn 输出
  ↓
可选 MainAgent 汇总
```

关键原则：

```text
1. MainAgent 只规划，不执行。
2. Gateway 只转发，不解释 action，不选择 Agent。
3. Orchestrator 是确认、状态、修订、幂等和执行调度的唯一权威。
4. Validator 用硬规则防止 MainAgent 越界。
5. 执行阶段先全部串行，后续再支持并行 / DAG。
```

---

## 5. 三条 executionPath 的精准语义

### 5.1 single_chat

用户明确选择一个 Agent。

```text
executionPath = single_chat
availableBoundary = [当前选中 Agent]
```

MainAgent 行为：

```text
1. MainAgent 调大模型生成计划。
2. 但只能使用 availableBoundary 中的唯一 Agent。
3. participants 只能包含当前 Agent。
4. steps[].assignedAgent 只能是当前 Agent。
5. 用户确认后只 dispatch 当前 Agent。
6. 用户反馈 revision 后仍然只能使用当前 Agent。
```

用户看到：

```text
MainAgent 已基于当前选中的 Agent 生成执行计划。
确认后只会调用该 Agent。
```

非法情况：

```text
single_chat 中 MainAgent 输出 test-agent / review-agent 等额外 Agent。
```

处理：

```text
PathAwareValidator 返回 AGENT_BOUNDARY_VIOLATION。
```

---

### 5.2 group_chat

用户选择多个 Agent 或 @ 多个 Agent。

```text
executionPath = group_chat
availableBoundary = selectedAgentNames / mentions / activeGroupAgents
```

MainAgent 行为：

```text
1. MainAgent 调大模型生成 group plan。
2. 但只能使用 availableBoundary 内的 Agent。
3. 不允许自动补 Agent。
4. 如果能力不足，只能写 warnings。
5. 用户确认后只串行 dispatch selectedParticipants 内的 Agent。
```

用户看到：

```text
MainAgent 已基于你选择 / @ 的 Agent 生成生产计划。
确认后只会按顺序调用这些 Agent。
```

非法情况：

```text
用户只选择 code-agent + review-agent，
MainAgent 输出 security-agent。
```

处理：

```text
PathAwareValidator 返回 AGENT_BOUNDARY_VIOLATION。
```

---

### 5.3 main_agent_orchestration

用户不知道找谁，选择 auto 或未指定 Agent。

```text
executionPath = main_agent_orchestration
availableBoundary = all enabled agents
```

MainAgent 行为：

```text
1. MainAgent 可以从 all enabled agents 中推荐 participants。
2. 每个 participant 必须有 reason。
3. required participant 不可直接取消。
4. optional participant 可以取消。
5. 用户确认后只串行 dispatch selectedParticipants。
```

用户看到：

```text
MainAgent 已根据任务推荐参与 Agent。
你可以调整可选 Agent，确认后才会按顺序执行。
```

非法情况：

```text
MainAgent 输出 unknown-agent 或 disabled-agent。
```

处理：

```text
PathAwareValidator 返回 UNKNOWN_AGENT / DISABLED_AGENT / AGENT_BOUNDARY_VIOLATION。
```

---

## 6. strategy 的最终处理

本轮最终决定：

```text
strategy 字段保留。
strategy 不作为架构核心。
strategy 不参与 Agent 选择。
strategy 不参与 boundary 校验。
strategy 不阻断执行。
strategy 暂不大重构。
执行阶段统一串行。
```

保留现有值：

```text
single
sequential
ordered_parallel
conversational
```

但本轮执行语义统一为：

```text
所有需要 dispatch Agent 的计划，先按 steps 顺序串行执行。
```

具体处理：

```text
single:
  串行执行唯一 Agent。

sequential:
  按 steps 顺序串行执行。

ordered_parallel:
  本轮不并行，仍按 steps 顺序串行执行。
  字段保留，后续再优化并行。

conversational:
  不需要调用子 Agent 时，由 MainAgent / Orchestrator 直接回复。
```

异常处理：

```text
1. strategy 缺失：默认按 steps 串行执行。
2. strategy 未知：记录 warning，按 steps 串行执行。
3. strategy=ordered_parallel：本轮仍按 steps 串行执行。
4. strategy=sequential：按 steps 串行执行。
5. strategy 与 dependsOn 冲突：本轮优先按 steps 串行执行。
```

禁止：

```text
1. 不因为 strategy 缺失阻断 approve。
2. 不因为 strategy 未知阻断 dispatch。
3. 不因为 ordered_parallel 没实现并行而失败。
4. 不为了 strategy 重构 contract。
```

---

## 7. Phase 1：Orchestrator 主链路收口

### 7.1 目标

将三条路径统一改为：

```text
single_chat              → MainAgent.GeneratePlan(boundary=[当前 Agent])
group_chat               → MainAgent.GeneratePlan(boundary=AllowedAgents)
main_agent_orchestration → MainAgent.GeneratePlan(boundary=all enabled agents)
```

### 7.2 对应位置

重点改：

```text
services/orchestrator/httpapi/handler_run_stream.go
```

需要调整的旧逻辑：

```text
single_chat:
  旧：handlePlanOnlySingleChat()
  新：MainAgent.GeneratePlan(executionPath=single_chat, availableBoundary=[agent])

group_chat:
  旧：fall through 到 LLMPlanner / RulePlanner
  新：MainAgent.GeneratePlan(executionPath=group_chat, availableBoundary=AllowedAgents)

auto:
  旧：MainAgent.Plan()
  新：MainAgent.GeneratePlan(executionPath=main_agent_orchestration, availableBoundary=all enabled agents)
```

相关辅助：

```text
services/orchestrator/internal/executionpath/executionpath.go
services/orchestrator/planner/main_agent.go
```

### 7.3 目标伪代码

```go
derivedPath := executionpath.DeriveExecutionPath(req)

selection, err := executionpath.ValidateAgentSelection(req, derivedPath)
if err != nil {
    return err
}

availableBoundary := selection.AllowedAgents

mainAgent := planner.NewMainAgent(registry, llmClient)

orchPlan, err := mainAgent.GeneratePlan(ctx, planner.MainAgentInput{
    UserMessage:       req.Message,
    ExecutionPath:     string(derivedPath),
    AvailableBoundary: availableBoundary,
    EnabledAgents:     enabledAgents,
    Revision:          1,
})
if err != nil {
    return err
}

if err := pathValidator.Validate(orchPlan, derivedPath, availableBoundary); err != nil {
    return err
}

registerPendingPlan(runID, orchPlan)
emitActivitySnapshot(w, orchPlan)
```

### 7.4 验收标准

```text
1. single_chat 会调用 MainAgent.GeneratePlan。
2. group_chat 会调用 MainAgent.GeneratePlan。
3. auto 会调用 MainAgent.GeneratePlan。
4. single_chat 不再依赖当前 Agent plan_only 作为主路径。
5. group_chat 不再用 RulePlanner / LLMPlanner 从全量池选 Agent。
6. 三条路径都能生成 ActivitySnapshot 计划卡片。
```

---

## 8. Phase 2：MainAgent LLM planner 最小实现

### 8.1 目标

MainAgent 必须接入大模型，但实现先保持简单。

```text
LLM + registry capability + availableBoundary
```

### 8.2 对应位置

重点改：

```text
services/orchestrator/planner/main_agent.go
```

建议后续拆分：

```text
services/orchestrator/mainagent/planner.go
services/orchestrator/mainagent/prompt.go
services/orchestrator/mainagent/schema.go
services/orchestrator/mainagent/parser.go
```

### 8.3 输入结构

```go
type MainAgentInput struct {
    UserMessage          string
    ExecutionPath        string
    AvailableBoundary    []string
    EnabledAgents        []AgentDescriptor
    PreviousPlan         *plan.OrchestrationPlan
    Feedback             string
    SelectedParticipants []string
    Revision             int
}
```

### 8.4 AgentDescriptor

```go
type AgentDescriptor struct {
    Name         string
    Description  string
    Capabilities []string
    OutputModes  []string
    Enabled      bool
}
```

### 8.5 LLM Prompt 必须包含

```text
1. 用户需求。
2. executionPath。
3. availableBoundary。
4. enabled agents 能力列表。
5. 非 auto 模式只能使用 availableBoundary。
6. 如果能力不足，只能写 warnings，不能补 Agent。
7. 必须输出严格 JSON。
8. 不允许输出 Markdown。
9. 不允许解释执行过程。
10. 不允许调用 Agent。
```

### 8.6 MainAgent 输出字段

```text
1. title
2. summary
3. executionPath
4. plannerOwner = main_agent
5. participants
6. defaultSelectedParticipants
7. requiredParticipants
8. steps
9. reasons
10. warnings
11. strategy 可选保留
```

### 8.7 验收标准

```text
1. MainAgent 调用大模型生成计划。
2. prompt 中包含 availableBoundary。
3. prompt 中包含 enabled agents capabilities。
4. MainAgent 输出严格 JSON。
5. MainAgent 不在 non-auto 模式补范围外 Agent。
6. MainAgent 输出 participants / reasons / warnings。
7. MainAgent 支持 revision。
```

---

## 9. Phase 3：PathAwareValidator 边界硬校验

### 9.1 目标

MainAgent 是 LLM 组件，不能完全信任。必须由 Validator 保证不越界。

### 9.2 对应位置

重点改：

```text
services/orchestrator/validator/validator.go
```

建议新增：

```text
services/orchestrator/validator/path_validator.go
```

调用位置：

```text
services/orchestrator/httpapi/handler_run_stream.go
services/orchestrator/httpapi/handler_hitl.go
```

### 9.3 必须校验

通用：

```text
1. executionPath 合法。
2. participants 非空。
3. participants 必须来自 availableBoundary。
4. steps 中的 assignedAgent 必须来自 availableBoundary。
5. defaultSelectedParticipants 必须来自 participants。
6. requiredParticipants 必须来自 participants。
7. requiredParticipants 必须包含在 defaultSelectedParticipants 内。
8. dependsOn 不引用不存在 step。
9. dependsOn 不成环。
```

single_chat：

```text
1. availableBoundary 只能有一个 Agent。
2. participants 只能是当前 Agent。
3. assignedAgent 只能是当前 Agent。
4. selectedParticipants 只能是当前 Agent。
```

group_chat：

```text
1. availableBoundary 只能来自 selectedAgentNames / mentions / activeGroupAgents。
2. participants 不得越界。
3. assignedAgent 不得越界。
4. selectedAgentNames 与 mentions 交集为空时返回 AGENT_SELECTION_CONFLICT。
5. MainAgent 输出范围外 Agent 时返回 AGENT_BOUNDARY_VIOLATION。
```

auto：

```text
1. participants 必须来自 enabled agents。
2. selectedParticipants 不得包含 unknown / disabled agent。
3. requiredParticipants 不能被用户直接取消。
```

Confirm：

```text
1. runId 存在。
2. planId 匹配。
3. revision 匹配。
4. pending state = waiting_user_approval。
5. selectedParticipants 来自 participants。
6. selectedParticipants 包含 requiredParticipants。
7. idempotencyKey 必填。
8. 已执行 / 已取消 / 已过期不能再次执行。
```

### 9.4 strategy 降级规则

```text
Validator 不因 strategy 未知或缺失直接阻断执行。
Validator 可以记录 warning。
执行阶段统一按 steps 串行执行。
```

### 9.5 验收标准

```text
single_chat:
  MainAgent 输出 test-agent → AGENT_BOUNDARY_VIOLATION

group_chat:
  用户只选 code-agent + review-agent
  MainAgent 输出 security-agent → AGENT_BOUNDARY_VIOLATION

auto:
  MainAgent 输出 unknown-agent → UNKNOWN_AGENT / AGENT_BOUNDARY_VIOLATION
```

---

## 10. Phase 4：PendingPlan / ActivitySnapshot / Confirm 修复

### 10.1 目标

保证计划卡片出现后，用户马上点 approve 也不会 502，也不会丢字段，也不会重复执行。

### 10.2 对应位置

Orchestrator：

```text
services/orchestrator/httpapi/handler_run_stream.go
services/orchestrator/httpapi/handler_hitl.go
```

Gateway：

```text
services/gateway/httpapi/handler_hitl.go
services/gateway/orchestratorclient/client.go
```

Frontend：

```text
frontend/src/stores/messageStore.ts
frontend/src/components/PlanApprovalCard.tsx
frontend/src/components/MainAgentPlanCard.tsx
```

### 10.3 registerPending 顺序

必须改成：

```text
registerPendingPlan
↓
emit ACTIVITY_SNAPSHOT
```

不能先 emit 再 register。

### 10.4 Gateway 透传 selectedParticipants

Gateway request struct 必须包含：

```go
SelectedParticipants []string `json:"selectedParticipants,omitempty"`
```

并完整转发给 Orchestrator。

### 10.5 Gateway 保留 Orchestrator 错误码

不能把所有 Orchestrator 错误都转成 502。

应保留：

```text
404 PLAN_NOT_FOUND
409 PLAN_REVISION_MISMATCH
409 INVALID_RUN_STATE
400 REQUIRED_PARTICIPANT_MISSING
400 AGENT_BOUNDARY_VIOLATION
```

### 10.6 idempotencyKey

前端 approve 时必须生成并发送：

```ts
idempotencyKey: crypto.randomUUID()
```

Orchestrator 必须用 idempotencyKey 防重复 dispatch。

### 10.7 PendingPlan 至少记录

```text
runId
planId
revision
executionPath
plannerOwner
availableBoundary
participants
defaultSelectedParticipants
requiredParticipants
selectedParticipants
feedbackHistory
idempotencyKeys
status
createdAt
expiresAt
```

### 10.8 ActivitySnapshot 至少包含

```text
activityId / planId
revision
executionPath
plannerOwner
participants
candidateParticipants
defaultSelectedParticipants
requiredParticipants
steps
warnings
allowedActions
```

### 10.9 验收标准

```text
1. ActivitySnapshot 出现后立即 approve 不再 502。
2. selectedParticipants 能到达 Orchestrator。
3. Orchestrator 错误码不被 Gateway 吞掉。
4. duplicate approve 不重复执行 Agent。
5. revise 后 revision + 1。
6. cancel 后不能再 approve。
```

---

## 11. Phase 5：串行 dispatch 最小收口

### 11.1 目标

Approve 后由 Orchestrator 按 selectedParticipants 和 steps 串行执行。

### 11.2 对应位置

建议新增或改造：

```text
services/orchestrator/executor/serial_executor.go
```

调用位置：

```text
services/orchestrator/httpapi/handler_run_stream.go
services/orchestrator/httpapi/handler_hitl.go
```

可复用：

```text
services/orchestrator/dispatcher/*
```

### 11.3 串行执行规则

```text
1. MainAgent 不直接调用子 Agent。
2. Orchestrator 读取 PendingPlan。
3. Orchestrator 校验 selectedParticipants。
4. Orchestrator 过滤掉未选择的 optional participants。
5. Orchestrator 按 plan.steps 数组顺序串行调用对应 Agent。
6. 每个 Agent turn 完成后再进入下一个 step。
7. 每个 Agent 输出独立 turn。
8. 可选 MainAgent 最后汇总。
```

### 11.4 step 处理规则

```text
1. step.assignedAgent 为空：
   不 dispatch 子 Agent，可作为 summary / conversational step。

2. step.assignedAgent 不在 selectedParticipants：
   跳过该 step。

3. step.assignedAgent 不在 availableBoundary：
   执行前再次拒绝，返回 AGENT_BOUNDARY_VIOLATION。

4. step.assignedAgent disabled / unknown：
   拒绝执行。

5. step 有 dependsOn：
   本轮不做 DAG，只按 steps 顺序执行。
```

### 11.5 strategy 处理

```text
1. strategy=single：仍按 steps 串行。
2. strategy=sequential：仍按 steps 串行。
3. strategy=ordered_parallel：本轮仍按 steps 串行。
4. strategy=conversational：不 dispatch 子 Agent。
5. strategy 缺失或未知：按 steps 串行。
```

### 11.6 验收标准

```text
single_chat:
  approve 后只 dispatch 当前 Agent。

group_chat:
  approve 后只 dispatch selected / mentioned agents。
  code-agent 和 review-agent 独立 turn 可见。
  严格按 steps 顺序执行。

auto:
  approve 后只 dispatch selectedParticipants。
  optional 被取消后不 dispatch。
  严格按 steps 顺序执行。
```

---

## 12. Phase 6：前端最小适配

### 12.1 目标

前端只做必要适配，不重写 UI。

### 12.2 对应位置

```text
frontend/src/stores/messageStore.ts
frontend/src/stores/activityStore.ts
frontend/src/components/ActivitySnapshotRenderer.tsx
frontend/src/components/PlanApprovalCard.tsx
frontend/src/components/MainAgentPlanCard.tsx
frontend/src/lib/activitySnapshot.ts
frontend/src/types/*
```

### 12.3 必须做

```text
1. ActivitySnapshotRenderer 继续作为计划确认主路径。
2. PlanApprovalCard 支持 plannerOwner = main_agent。
3. single_chat 下展示：MainAgent 规划，但只会执行当前 Agent。
4. group_chat 下展示：MainAgent 规划，但只会执行已选择 / 已 @ Agent。
5. auto 下展示：MainAgent 推荐参与 Agent。
6. required participant 不可取消。
7. optional participant 可取消。
8. approve 发送 selectedParticipants + idempotencyKey。
9. revise 发送 feedback + selectedParticipants。
10. cancel 发送 runId / planId / revision。
```

### 12.4 建议文案

single_chat：

```text
MainAgent 已基于当前选中的 Agent 生成执行计划。确认后只会调用该 Agent。
```

group_chat：

```text
MainAgent 已基于你选择 / @ 的 Agent 生成生产计划。确认后只会按顺序调用这些 Agent。
```

auto：

```text
MainAgent 已根据任务推荐参与 Agent。你可以调整可选 Agent，确认后才会按顺序执行。
```

### 12.5 验收标准

```text
1. 按钮防重复点击。
2. revision 更新后替换旧计划。
3. required 不可取消。
4. optional 可取消。
5. idempotencyKey 每次 approve 有值。
6. 用户看到的是计划卡片，不是 JSON。
```

---

## 13. Phase 7：测试矩阵

### 13.1 single_chat

```text
Case S1:
agentName=code-agent

期望：
executionPath=single_chat
availableBoundary=[code-agent]
MainAgent 生成 participants=[code-agent]
ActivitySnapshot 展示计划卡片
approve 后只 dispatch code-agent
```

```text
Case S2:
single_chat 下 MainAgent 输出 test-agent

期望：
Validator 返回 AGENT_BOUNDARY_VIOLATION
不生成可执行 pending plan
不执行任何 Agent
```

```text
Case S3:
single_chat revise

期望：
revision + 1
participants 仍只有 code-agent
未 approve 前不执行
```

### 13.2 group_chat

```text
Case G1:
selectedAgentNames=[code-agent, review-agent]

期望：
executionPath=group_chat
availableBoundary=[code-agent, review-agent]
MainAgent 生成 code → review 计划
ActivitySnapshot 展示计划卡片
approve 后 code-agent / review-agent 按 steps 顺序独立 turn
```

```text
Case G2:
selectedAgentNames=[code-agent]
message=@web-agent 做登录页面

期望：
AGENT_SELECTION_CONFLICT
不生成计划
不执行 Agent
```

```text
Case G3:
group_chat 下 MainAgent 输出 security-agent

期望：
AGENT_BOUNDARY_VIOLATION
不执行 security-agent
```

### 13.3 auto

```text
Case A1:
agentName=auto

期望：
executionPath=main_agent_orchestration
availableBoundary=all enabled agents
MainAgent 推荐 candidates + reasons
ActivitySnapshot 展示计划卡片
```

```text
Case A2:
取消 optional participant

期望：
approve 成功
被取消 Agent 不 dispatch
warnings 说明未执行 optional
```

```text
Case A3:
取消 required participant

期望：
REQUIRED_PARTICIPANT_MISSING
不执行
```

### 13.4 HITL

```text
Case H1:
ActivitySnapshot 出现后立即 approve

期望：
confirm 成功，不 502
```

```text
Case H2:
重复点击 approve

期望：
只 dispatch 一次 Agent
```

```text
Case H3:
revision mismatch

期望：
返回 PLAN_REVISION_MISMATCH，不变 502
```

### 13.5 串行执行

```text
Case E1:
group_chat code-agent + review-agent

期望：
code-agent 完成后，review-agent 才开始。
```

```text
Case E2:
auto code-agent + web-agent + test-agent

期望：
严格按 steps 顺序执行，不并发。
```

```text
Case E3:
strategy=ordered_parallel

期望：
本轮仍串行执行，不失败。
```

---

## 14. 禁止修复方式

本轮禁止：

```text
1. 按旧审计继续实现 GroupCoordinatorPlanOwner。
2. single_chat 继续以当前 Agent plan_only 作为主路径。
3. group_chat 继续进入 RulePlanner / LLMPlanner 全量池选 Agent。
4. MainAgent 在 single_chat 自动补其他 Agent。
5. MainAgent 在 group_chat 自动补范围外 Agent。
6. 只靠 prompt 防越界，没有 Validator。
7. Gateway 解释 action 或选择 Agent。
8. MainAgent 直接 dispatch 子 Agent。
9. 继续用 fake confirm_plan TOOL_CALL 作为新主路径。
10. 只修 sequential allowlist 就宣称完成。
11. required participant 可以被前端直接取消。
12. 不实现 idempotencyKey 就说重复点击已解决。
13. 因为 strategy 缺失或未知阻断执行。
14. 本轮实现并行 / DAG / parallel 优化。
15. 把 JSON 直接展示给用户。
```

---

## 15. 最小交付标准

完成后至少必须满足：

```text
1. 三条路径都由 MainAgent 调大模型生成计划。
2. single_chat 只能使用当前 Agent。
3. group_chat 只能使用用户选择 / @ 的 Agent。
4. auto 可以从 enabled agents 推荐 Agent。
5. Validator 能拦截所有越界 Agent。
6. ActivitySnapshot 是计划确认主路径。
7. 用户看到计划卡片，不是 JSON。
8. registerPending 在 emitActivitySnapshot 前。
9. approve 能透传 selectedParticipants。
10. approve 必须带 idempotencyKey。
11. duplicate approve 不重复执行。
12. Gateway 不吞 Orchestrator 错误码。
13. optional participant 可取消，required participant 不可直接取消。
14. 执行阶段全部串行。
15. strategy 字段保留，但不阻断执行。
```

---

## 16. 推荐执行顺序

最高效率顺序：

```text
1. handler_run_stream.go 主链路改造：
   所有路径统一调用 MainAgent.GeneratePlan。

2. MainAgent.GeneratePlan 改造：
   接入大模型，支持 executionPath + availableBoundary。

3. PathAwareValidator：
   participants / assignedAgent 必须在 availableBoundary 内。

4. Confirm 链路修复：
   registerPending 顺序、selectedParticipants、错误码、idempotencyKey。

5. PendingPlan / ActivitySnapshot 补字段：
   participants、required、default、warnings、feedbackHistory。

6. 串行 dispatch：
   approve 后按 steps 顺序调用 Agent。

7. Frontend 最小适配：
   idempotencyKey、required/optional、三种文案、计划卡片展示。

8. 回归测试：
   single_chat / group_chat / auto / HITL / duplicate approve / 串行执行。
```

不要先修：

```text
1. strategy 大重构。
2. GroupCoordinatorPlanOwner。
3. plan_only JSON。
4. fake confirm_plan legacy。
5. 并行执行。
6. DAG 调度。
```

---

## 17. 最终判断

```text
当前代码是否符合新版统一 MainAgent 多路径计划确认方案：否

是否需要修复：是

修复方式：
用 Orchestrator 内部 MainAgent 调大模型统一生成计划；
用 executionPath 生成 availableBoundary；
用 Validator 强制边界；
用 ActivitySnapshot 展示计划卡片；
用户确认后由 Orchestrator 串行 dispatch。

旧审计是否保留：是

旧审计保留方式：
作为 legacy audit / historical issue backlog。
只保留仍有效 P0/P1 问题，不作为新版架构蓝图。

strategy 是否需要大修：否

strategy 最终处理：
保留字段，不阻断执行。
本轮所有 dispatch 统一串行。
后续需要并行 / DAG / ordered_parallel 时，再单独优化。

最小完成标准：
三条路径都能生成 ActivitySnapshot 计划卡片；
用户确认前不执行；
确认后只串行 dispatch 合法 selectedParticipants；
用户看到的是计划卡片，不是 JSON。
```
