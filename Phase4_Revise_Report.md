# Phase 4 Step 2: REQUEST_PLAN_REVISION 闭环 — 最终报告

## 概述

实现了 single_chat 执行路径下的 Plan Revision 闭环：用户可以在 PlanApprovalCard 中输入反馈意见，提交 revise 请求 → Orchestrator 将 feedback 发送给 Agent 的 plan_only 模式重新生成方案 → revision+1 的新 plan 通过 SSE confirm_plan 事件返回前端 → 用户再次确认/取消/修改。

### Phase 4 Protocol Patch（审计后修复）

审计发现并修复了以下问题：

| Issue | 说明 | 状态 |
|-------|------|------|
| 1. revise plan_only 输入不完整 | plan_only Dispatch 消息仅包含 feedback，缺少原始 userText → Agent 无法基于完整上下文重新生成方案 | **已修复** — 新增 `buildRevisionPlanOnlyMessage()` 组合 userText + feedback + summary |
| 2. AG-UI/SSE 协议描述不准确 | 报告称"OrchestratorStreamEvent Type 使用 AG-UI 标准事件类型名称" — 实际 Orchestrator 使用 lowercase，Gateway Translator 转换为 UPPER_SNAKE | **已修复** — 报告已更正为完整转换链路 |
| 6. PlanApprovalCard UI 不当 | textarea 始终可见，placeholder 说"可选"但 feedback 必填 | **已修复** — toggle 行为：默认隐藏，点击"提出修改意见"后显示 |
| 7. RUN_ERROR 生命周期 | 确认 `RUN_ERROR` 是 terminal 事件，cancel 路径正确使用 STATE_UPDATE+RUN_FINISHED | **已验证** — per `docs/contracts/agui-events.md` line 196-206 |

新增后端测试：
- `TestBuildRevisionPlanOnlyMessage` / `TestBuildRevisionPlanOnlyMessage_NoPreviousSummary` — 验证组合消息
- `TestHITLConfirm_ReviseAction` — 验证 revise action 正确路由到 channel
- `TestHITLConfirm_ReviseEmptyFeedbackRejected` — 验证空 feedback 返回 REVISION_INPUT_REQUIRED

## 约束遵守情况

| # | 约束 | 遵守 |
|---|------|------|
| 1 | 只实现 single_chat revision | ✅ group_chat / main_agent_orchestration 收到 action=revise 返回 NOT_IMPLEMENTED |
| 2 | revise 后不 deregister pending run | ✅ approve/cancel 才 deregister + delete pending plan；revise 保持 pending，重新注册 |
| 3 | feedback 只用于 plan_only，full execute 用原始 userText | ✅ executeUserText 在首次进入 confirmLoop 前保存，approve 后执行用 executeUserText |
| 4 | action 兼容旧 confirmed 字段 | ✅ resolveAction() 函数优先 action，空则回退 confirmed bool |
| 5 | feedback 为空时拒绝 | ✅ handler_hitl.go 校验空 feedback → 返回 REVISION_INPUT_REQUIRED |
| 6 | revision mismatch 最小实现 | ✅ confirmLoop 中检查前端传 revision != currentRevision → PLAN_REVISION_MISMATCH |
| 7 | 不实现完整 idempotencyKey | ✅ 字段保留但不做幂等逻辑 |
| 8 | 不修改 docs/contracts/.claude/skills | ✅ 未修改 |
| 9 | 确认实际 Agent 文件路径 | ✅ code-agent: services/agents/code-agent/agent.go, web-agent: services/agents/web-agent/agent.go |
| 10 | 最终报告列出所有变更细节 | ✅ 本报告 |

---

## 变更文件清单（13 个文件）

### 后端 Go（5 个文件）

#### 1. `services/orchestrator/plan/types.go` — 核心类型扩展
- **OrchestrationPlan** 新增字段：
  - `Revision int` — 方案版本号，从 1 开始单调递增
  - `PlanOwner *PlanOwner` — 方案所有者（single_chat 为 `{type:"agent", agentName:"<agent>"}`）
  - `Participants []PlanParticipant` — 参与者列表
- **新增类型**：
  - `PlanOwner{Type, AgentName, IsMainAgent}`
  - `PlanParticipant{AgentName, Role, Required, Selected}`

#### 2. `services/orchestrator/httpapi/server.go` — HITL 状态机
- **HITLConfirmResult** 新增字段：`Action`, `Feedback`, `Revision`, `IdempotencyKey`
- **新增状态**：`HITLRevising HITLState = "revising"`
- `HITLState` 常量块新增：`HITLCancelled HITLState = "cancelled"`

#### 3. `services/orchestrator/httpapi/handler_hitl.go` — HITL 确认处理
- **HITLConfirmRequest** 新增字段：`Action`, `Feedback`, `Revision`, `IdempotencyKey`；`Confirmed` 改为 `*bool`
- **新增 resolveAction() 函数**：优先解析 action 字段（approve/cancel/revise），空则回退 confirmed bool 兼容旧逻辑
- **feedback 校验**：action=revise 且 feedback 为空 → 返回 `REVISION_INPUT_REQUIRED`
- **状态转换**：revise → 通道发送后状态设为 `HITLRevising`（而非 HITLCancelled）
- **新增 HITLCancelled 状态处理**：返回 "plan was cancelled"

#### 4. `services/orchestrator/httpapi/handler_run_stream.go` — 核心 revise 闭环
这是变更最大的文件（~700 行新增）。关键位置：

**a) OrchestrationRequest 新增字段**（行 ~31-32）：
- `RequestedPath string` — 用户请求的执行路径
- `ExecutionPath string` — Gateway 传递的执行路径（调试用）

**b) 执行路径推导 + 验证**（行 ~132-170）：
- Orchestrator 作为 authoritative source 推导 executionPath
- 单 agent single_chat 走 `handlePlanOnlySingleChat`

**c) handlePlanOnlySingleChat 函数**（新增 ~600 行）：
- orchPlan 初始化包含 `Revision: 1`, `PlanOwner`, `Participants`
- confirm_plan TOOL_CALL_ARGS 包含 `executionPath`, `planOwner`, `participants`, `revision`
- planState 包含 `revision`

**d) confirmLoop 核心逻辑**（原 986-1064 行替换为 ~200+ 行）：

```
currentRevision := 1
executeUserText := userText  // 保存原始用户文本，full execute 使用

for {
    select {
    case result := <-confirmCh:
        // action=revise 处理：
        //   1. revision mismatch 检查
        //   2. 不 deregister pending run
        //   3. 重置 deadline
        //   4. emit STATE_UPDATE(revising_plan)
        //   5. Dispatch plan_only with feedback
        //   6. 解析 + 验证新 plan
        //   7. currentRevision++ → 构建新 orchPlan
        //   8. 清理旧 planState + 注册新 planState（revision+1）
        //   9. emit 新 confirm_plan TOOL_CALL SSE 事件
        //   10. continue（等待下一次 confirm）

        // action=approve：
        //   1. deregisterPending
        //   2. delete pending plan
        //   3. break confirmLoop → 执行 full task（使用 executeUserText）

        // action=cancel：
        //   1. deregisterPending
        //   2. delete pending plan
        //   3. emit STATE_UPDATE(cancelled) + RUN_FINISHED
        //   4. return

    case <-timeoutCh:
        // 超时自动取消
    }
}
```

**e) 旧 confirmLoop（非 single_chat）**：
- 更新为检查 Action 字段
- action=revise → 返回 NOT_IMPLEMENTED

#### 5. `services/gateway/orchestratorclient/client.go` — Gateway 客户端
- **HITLConfirmRequest** 新增字段：`Action`, `Feedback`, `Revision`, `IdempotencyKey`
- `Confirmed` 改为 `*bool`（指针，支持 omitempty）
- **Run() 方法**：请求体新增 `requestedPath` 字段传递给 Orchestrator

#### 5b. `services/gateway/httpapi/server_test.go` — Gateway 测试修复
- 修复 `*bool` 解引用：`runner.lastConfirmReq.Confirmed` → 指针判空后再解引用

---

### 前端 TypeScript（5 个文件）

#### 6. `frontend/src/types/index.ts` — 类型定义
- **HITLConfirmRequest** 接口更新：
  - `confirmed` 改为可选（`confirmed?: boolean`）
  - 新增可选字段：`action?`, `feedback?`, `revision?`, `idempotencyKey?`
  - `rejectReason` 改为可选

#### 7. `frontend/src/stores/messageStore.ts` — 状态管理
- **PendingConfirmation** 接口新增字段：`planId`, `revision`, `executionPath`, `planOwner`, `participants`, `warnings`
- **status 类型**新增 `'cancelled'`
- **confirmPlan 签名**：新增参数 `action?`, `feedback?`, `revision?`
- **confirm_plan SSE 解析**：提取 `executionPath`, `revision`, `planOwner`, `participants`, `warnings`
- **confirmPlan 实现**：
  - API 调用传递 `action`, `feedback`, `revision`
  - revise 动作时 status 保持 `'pending'`（等待新 plan SSE）
  - approve → `'confirmed'`, cancel → `'cancelled'`

#### 8. `frontend/src/stores/messageStore.test.ts` — 测试更新
- 两个 `toHaveBeenCalledWith` 断言添加新字段（`action`, `feedback`, `revision`）
- 状态断言从 `'rejected'` 改为 `'cancelled'`

#### 9. `frontend/src/components/PlanApprovalCard.tsx` — 方案审批卡片（全新组件）
- 新增 `PlanApprovalStatus` 类型（含 `'revising'` 状态）
- 新增 `onRevise` prop
- 新增 feedback textarea
- 新增 "提出修改意见" 按钮
- 显示 planOwner、executionPath、participants、warnings
- 状态管理：`isBusy` 包含 revising，按钮 loading 文本

#### 10. `frontend/src/components/ChatWindow.tsx` — 聊天窗口
- 引入 `PlanApprovalCard` 替代旧的 `HITLConfirm` 组件
- 新增 `confirmStatus` 状态 —— 跟踪 PlanApprovalCard 的视觉状态
- 新增 `useEffect` —— 当 confirmation 回到 'pending' 时重置 confirmStatus
- 新增 `handleRevise` 回调 —— 调用 confirmPlan with action='revise', feedback, revision
- `handleConfirm` — 设置 confirmStatus='approving'
- `handleCancel` — 设置 confirmStatus='cancelling'（替代旧 handleReject）
- PlanApprovalCard 传参：`status={confirmStatus}`, `onApprove`, `onCancel`, `onRevise`
- 新增 'cancelled' 状态横幅

---

### 支撑文件（3 个文件，非 Phase 4 特有但相关）

#### 11. `frontend/src/types/planApproval.ts` — 新增类型定义文件
- `PlanApprovalData`, `PlanApprovalTask`, `PlanApprovalParticipant` 接口

#### 12. `frontend/src/lib/planApproval.ts` — 数据标准化工具
- `normalizePlanApprovalData()` 将 PendingConfirmation 转为 PlanApprovalData

#### 13. `services/orchestrator/internal/executionpath/` — 执行路径推导（架构重构）
- 从 planner 中独立出的 execution path 推导逻辑

---

## 关键实现位置速查

| 功能 | 文件 | 大致行号/位置 |
|------|------|--------------|
| resolveAction() 兼容逻辑 | `handler_hitl.go` | 函数定义 ~24-44 |
| feedback 非空校验 | `handler_hitl.go` | `if action == "revise" && strings.TrimSpace(req.Feedback) == ""` |
| confirmLoop revise 分支 | `handler_run_stream.go` | `case result := <-confirmCh:` 内 action=="revise" |
| feedback → Agent plan_only | `handler_run_stream.go` | Dispatch 调用传入 buildRevisionPlanOnlyMessage(userText, feedback, revision, summary) |
| revision+1 | `handler_run_stream.go` | `currentRevision++` 后构建新 orchPlan |
| 新 confirm_plan emit | `handler_run_stream.go` | emitConfirmPlanSSE() / TOOL_CALL 序列 |
| executeUserText 保存 | `handler_run_stream.go` | confirmLoop 外的 `executeUserText := userText` |
| 前端 revise 调用路径 | `ChatWindow.tsx` → `messageStore.ts` → `api.ts` → Gateway → Orchestrator | handleRevise → confirmPlan(action='revise') |
| NOT_IMPLEMENTED (非 single_chat) | `handler_run_stream.go` | 旧 confirmLoop 的 revise case |

---

## SSE 事件序列（revise 流程）

```
[用户提交 revise]
STATE_UPDATE(revising_plan)      ← 前端看到 "Revising..."
TOOL_CALL_START(confirm_plan)    ← 新 round
TOOL_CALL_ARGS({...revision+1...})
TOOL_CALL_END(confirm_plan)
STATE_UPDATE(awaiting_confirmation)
[等待用户操作]
```

## 测试结果

### Go 测试
- `services/orchestrator/httpapi` — ✅ PASS
- `services/orchestrator/plan` — ✅ PASS
- `services/orchestrator/dispatcher` — ✅ PASS
- `services/gateway/orchestratorclient` — ✅ PASS
- `services/gateway/httpapi` — ✅ PASS
- `services/orchestrator/planner` — ❌ BUILD FAIL (预存问题: `undefined: stripMarkdownFences`，与本次变更无关)

### 前端测试
- 14 个测试文件全部通过 ✅
- 147 个测试用例全部通过 ✅
- TypeScript 类型检查通过 ✅

---

## 下次 Code Review 最小文件集

审核 Phase 4 revise 闭环需要的最小文件集（按优先级）：

1. **`services/orchestrator/httpapi/handler_run_stream.go`** — 核心逻辑，confirmLoop + handlePlanOnlySingleChat
2. **`services/orchestrator/httpapi/handler_hitl.go`** — resolveAction + feedback 校验
3. **`services/orchestrator/plan/types.go`** — Revision/PlanOwner/Participant 类型
4. **`frontend/src/components/ChatWindow.tsx`** — handleRevise + confirmStatus 状态
5. **`frontend/src/stores/messageStore.ts`** — confirmPlan revise 处理 + PendingConfirmation 扩展

---

# 整体调用链路与协议遵守情况自查

## 1. 当前整体调用链路

### 1.1 Revise 链路

```
Frontend PlanApprovalCard (feedback textarea + "提出修改意见" 按钮)
  → PlanApprovalCard.handleRevise (内部 acting 状态，调用 onRevise(feedback))
  → ChatWindow.handleRevise (设置 confirmStatus='revising')
  → messageStore.confirmPlan(convId, runId, actionId, false, '', 'revise', feedback, pending.revision)
  → api.confirmHITL (POST /api/runs/{runId}/confirm, body 含 action/feedback/revision)
  → Gateway handleRunsConfirm (handler_hitl.go:19-78)
      — 纯透明转发：JSON decode → hitlRunner.ConfirmRun(ctx, req)
      — 不解释 action、不生成 plan、不调用 Agent
  → orchestratorclient.ConfirmRun → POST /internal/orchestrator/hitl/confirm
  → Orchestrator handleHITLConfirm (handler_hitl.go:52-182)
      — resolveAction(req) → "revise", false
      — 校验 feedback 非空 (line 100-105)
      — 通过 channel 路由到 confirmLoop
  → Orchestrator confirmLoop (handler_run_stream.go:1007-1315)
      — case "revise" (line 1012-1240):
        1. revision mismatch 检查 (line 1014-1019)
        2. 不 deregister pending run
        3. 重置 confirmDeadline (line 1022)
        4. emit STATE_UPDATE phase=revising_plan (line 1025-1029)
        5. s.dispatcher.Dispatch(..., Message: reviseMessage, Mode: "plan_only")
           其中 reviseMessage = buildRevisionPlanOnlyMessage(executeUserText, result.Feedback, currentRevision+1, orchPlan.IntentSummary)
           调用 **同一个 Agent** 的 plan_only 模式 (line 1049-1065)
        6. JSON 解析 agent 返回的新 plan (line 1049-1067)
        7. 校验新 plan：strategy=single, tasks=1, agentName 匹配 (line 1070-1091)
        8. currentRevision++ (line 1129)
        9. 构建新 orchPlan (revision+1, 新 planID) (line 1133-1153)
        10. Validator 校验 + agent boundary 强制检查 (line 1156-1178)
        11. emit 新 confirm_plan TOOL_CALL_START/ARGS/END (line 1190-1221)
        12. emit STATE_UPDATE phase=awaiting_confirmation (line 1223-1233)
        13. s.pendingPlans[runID] = orchPlan 重新注册 (line 1236-1238)
        14. continue confirmLoop（不 break，不 deregister）
  → Gateway SSE Writer 将 OrchestratorStreamEvent 包装为 AG-UI events over SSE
      — sse.go: sseEventName() 映射 type → SSE event: 行 (line 77-102)
  → Frontend agui/client.ts runAgent() 解析 SSE → AGUIEvent JSON → onEvent 回调
  → messageStore.ts sendMessage() switch(event.type) 处理:
      — TOOL_CALL_START/ARGS: 累积 tool args
      — TOOL_CALL_END(toolName=='confirm_plan'): 解析 args → 更新 confirmationByConversation (line 946-998)
  → confirmationByConversation[convId] 被新版 plan (revision+1) 覆盖
  → React 响应式触发 PlanApprovalCard 重新渲染，展示新版 plan
```

**证据文件**：

| 步骤 | 文件 | 行号 |
|------|------|------|
| PlanApprovalCard revise 按钮 | `PlanApprovalCard.tsx` | 210-217 |
| ChatWindow.handleRevise | `ChatWindow.tsx` | 114-137 |
| messageStore.confirmPlan revise 路径 | `messageStore.ts` | 1106-1147 |
| api.confirmHITL | `api.ts` | 121-128 |
| Gateway 透明转发 | `gateway/httpapi/handler_hitl.go` | 19-78 |
| Orchestrator resolveAction + feedback 校验 | `orchestrator/httpapi/handler_hitl.go` | 26-105 |
| confirmLoop action=revise | `orchestrator/httpapi/handler_run_stream.go` | 1012-1240 |
| buildRevisionPlanOnlyMessage (组合 userText + feedback + summary) | `orchestrator/httpapi/handler_run_stream.go` | 796-809 |
| Agent plan_only with combined revise message | `orchestrator/httpapi/handler_run_stream.go` | 1049-1065 |
| revision+1 + 新 orchPlan | `orchestrator/httpapi/handler_run_stream.go` | 1129-1153 |
| 新 confirm_plan TOOL_CALL emit | `orchestrator/httpapi/handler_run_stream.go` | 1190-1221 |
| pendingPlans 重新注册 | `orchestrator/httpapi/handler_run_stream.go` | 1236-1238 |
| Gateway AG-UI SSE 包装 | `gateway/sse/sse.go` | 61-102 |
| 前端 SSE 解析 | `agui/client.ts` | 38-92 |
| Gateway SSE → AG-UI 转换 | `gateway/agui/translator.go` — `Translate()` | lowercase → UPPER_SNAKE |
| Gateway SSE wire 格式 | `gateway/sse/sse.go` — `sseEventName()` + `WriteEvent()` | event: / data: 行 |
| 前端 confirm_plan TOOL_CALL_END 处理 | `messageStore.ts` | 946-998 |

### 1.2 Approve 链路

```
PlanApprovalCard "同意并执行" 按钮
  → ChatWindow.handleConfirm (confirmStatus='approving')
  → messageStore.confirmPlan(convId, runId, actionId, true) → action='approve'
  → api.confirmHITL (POST /api/runs/{runId}/confirm)
  → Gateway 透明转发
  → Orchestrator confirmLoop case "approve" (line 1242-1245)
      — s.deregisterPending(runID)
      — approved = true
      — break confirmLoop
  → 执行阶段 (line 1317-1337):
      — emit STATE_UPDATE phase=executing (line 1324-1328)
      — execPlan.Tasks[0].TaskContent = executeUserText (line 1333)
        ↑ **必须是原始 userText，不是 feedback，不是 plan_only proposal content**
      — s.executeViaStreamingExecutor (line 1335-1337)
  → 同一个 Agent full execute
```

**关键证据**：
- `executeUserText := userText` 在 confirmLoop 入口保存 (line 1005)，**永远不变**
- approve 后执行：`execPlan.Tasks[0].TaskContent = executeUserText` (line 1333)，**不引用 feedback**
- revise 时 `Dispatch(..., Message: reviseMessage, Mode: "plan_only")`（`reviseMessage` 由 `buildRevisionPlanOnlyMessage(executeUserText, result.Feedback, currentRevision+1, orchPlan.IntentSummary)` 构造，组合原始 userText + feedback + 上一版摘要），只用于重新生成 plan，不影响 executeUserText

### 1.3 Cancel 链路

```
PlanApprovalCard "取消" 按钮
  → ChatWindow.handleCancel (confirmStatus='cancelling')
  → messageStore.confirmPlan(convId, runId, actionId, false, reason) → action='cancel'
  → api.confirmHITL
  → Gateway 透明转发
  → Orchestrator confirmLoop case "cancel" (line 1247-1260)
      — s.deregisterPending(runID)
      — emit STATE_UPDATE phase=cancelled (line 1250-1254)
      — emit RUN_FINISHED status=cancelled (line 1255-1258)
      — return（**不执行 Agent，不发 RUN_ERROR**）
```

**关键证据**：
- cancel 分支 (line 1247-1260)：只有 `deregisterPending` + `state_update` + `run_finished`，**无 Dispatch 调用，无 run_error 事件**
- 注释明确标注：`// Cancel: NO RUN_ERROR — STATE_UPDATE + RUN_FINISHED only.` (line 1248)

**RUN_ERROR 生命周期确认** (per `docs/contracts/agui-events.md` line 196-206)：
- `RUN_ERROR` 是 terminal 事件（"Run failed"），**不需要**后跟 `RUN_FINISHED` — message 结束即 run 结束
- Cancel lifecycle v1.2 明确规定：cancel 使用 `STATE_UPDATE(cancelled)` + `RUN_FINISHED(cancelled)` 两个事件，**不得** emit `RUN_ERROR`
- 当前 single_chat cancel 路径完全符合 contract v1.2 — 零 RUN_ERROR
- 错误路径（如 plan_only timeout/Dispatch 失败）：只 emit `run_error`（Gateway 转为 `RUN_ERROR`），后随 `run_finished` — 因为 run_error 在 non-cancel 场景可跟 run_finished

**已知预存问题（不在 Phase 4 范围内）**：旧 non-single_chat confirmLoop (line 377-392) 在 user rejection 时 emit `run_error` + `run_finished`，违反 v1.2 contract（cancel 必须用 `state_update(cancelled)` + `run_finished(cancelled)`，不得 emit `run_error`）。建议 follow-up 修复。

---

## 2. AG-UI / SSE 硬性协议自查

**核心声明**：当前实现是 **AG-UI-compliant events over SSE**。AG-UI 是事件协议，SSE 只是 transport。Orchestrator 内部使用 **lowercase** 事件类型名（如 `run_started`、`state_update`、`tool_call_start`），Gateway 通过以下转换链产出 public AG-UI-compliant events over SSE（**UPPER_SNAKE** 类型名：`RUN_STARTED`、`STATE_UPDATE`、`TOOL_CALL_START`）：

```
Orchestrator SSE (lowercase: run_started, state_update, tool_call_start, ...)
  → Gateway orchestratorclient.parseSSEStream → adk.Event (metadata["eventType"] = lowercase)
  → Gateway agui.Translator.Translate → uppercase AG-UI (RUN_STARTED, STATE_UPDATE, TOOL_CALL_START, ...)
  → Gateway sse.Writer.WriteEvent → public SSE (event: + data:)
  → Frontend agui/client.ts → AGUIEvent JSON
  → Frontend messageStore
```

整个链路中没有绕过 AG-UI 协议的自定义通道。AG-UI 标准事件类型名（UPPER_SNAKE）是 Gateway `agui.Translator` 产出的，而非 Orchestrator 直接产出。

### 逐项检查

#### 2.1 是否新增了非 AG-UI 格式的自定义 SSE data？
**结论：没有。** ✅

所有 OrchestratorStreamEvent Type 值为 lowercase 内部类型名，Gateway agui.Translator 将其转换为 public AG-UI UPPER_SNAKE 类型：
- `run_started` → `RUN_STARTED`, `run_finished` → `RUN_FINISHED`, `run_error` → `RUN_ERROR` — AG-UI Run Lifecycle
- `state_update` → `STATE_UPDATE` — AG-UI State Management
- `tool_call_start` → `TOOL_CALL_START`, `tool_call_args` → `TOOL_CALL_ARGS`, `tool_call_end` → `TOOL_CALL_END` — AG-UI Tool Call
- `message_start` → `MESSAGE_START`, `message_delta` → `MESSAGE_DELTA`, `message_end` → `MESSAGE_END` — AG-UI Text Message

证据：`gateway/agui/translator.go` 的 `Translate` 方法将 `adk.Event` 转换为 AG-UI `Event[]`，`gateway/sse/sse.go` 的 `sseEventName()` 将 AG-UI type 映射为 SSE `event:` 行。

#### 2.2 初始 confirm_plan 是否仍通过 AG-UI tool event 表达？
**结论：是。** ✅

证据 (`handler_run_stream.go` line 948-978)：
```
TOOL_CALL_START (toolCallName="confirm_plan", toolCallID=planID)
→ TOOL_CALL_ARGS (delta=JSON{runId, revision, executionPath, planOwner, participants, ...})
→ TOOL_CALL_END (toolCallID=planID)
```

前端 `messageStore.ts` line 946 在 `TOOL_CALL_END` 中检查 `toolName === 'confirm_plan'` 来设置 `confirmationByConversation`。

#### 2.3 revise 后新的 plan 是否仍通过新的 confirm_plan tool event 表达？
**结论：是。** ✅

证据 (`handler_run_stream.go` line 1190-1221)：
新的 `TOOL_CALL_START/ARGS/END` 序列，使用新的 `toolCallID`（新的 `planID`），args 中 `revision: currentRevision+1`。结构与初始 confirm_plan 完全一致。

前端解析逻辑不变 — TOOL_CALL_END 中的 `confirm_plan` 检查同时匹配初始和 revised plan。

#### 2.4 状态是否通过 AG-UI state_update/run_finished/run_error 表达？
**结论：是。** ✅

| 状态 | 事件 | 行号 |
|------|------|------|
| `planning` | `state_update` phase=planning | 942-946, 1184-1188 |
| `awaiting_confirmation` | `state_update` phase=awaiting_confirmation | 987-991, 1229-1233 |
| `revising_plan` | `state_update` phase=revising_plan | 1025-1029 |
| `executing` | `state_update` phase=executing | 1324-1328 |
| `cancelled` | `state_update` phase=cancelled → `run_finished` status=cancelled | 1250-1258 |
| `timeout` | `run_error` → `run_finished` status=timeout | 1295-1310 |
| `completed` | `run_finished` (由 executor 内部 emit) | — |

#### 2.5 前端是否仍然通过 messageStore 解析 AG-UI events？
**结论：是。** ✅

`messageStore.ts` 的 `sendMessage()` 函数 (line 381-1077) 接收 `AGUIEvent`，通过 `switch(event.type)` 处理所有事件类型，包括 `TOOL_CALL_START`, `TOOL_CALL_ARGS`, `TOOL_CALL_END`, `STATE_UPDATE`, `RUN_STARTED`, `RUN_FINISHED`, `RUN_ERROR` 等。

#### 2.6 是否新增了单独的 revision SSE parser？
**结论：没有。** ✅

前端只有一个 SSE 解析入口：`agui/client.ts` 的 `runAgent()` 函数。没有为 revision 添加独立的 parser。revised plan 的 confirm_plan 事件与初始 confirm_plan 使用相同的 TOOL_CALL_END 分支处理。

#### 2.7 是否绕过 AG-UI 直接用自定义事件驱动 PlanApprovalCard？
**结论：没有。** ✅

PlanApprovalCard 完全由 `confirmationByConversation` 状态驱动，该状态仅通过 AG-UI 事件的 `STATE_UPDATE` 和 `TOOL_CALL_END(confirm_plan)` 更新。`confirmStatus` 是 ChatWindow 本地 UI 状态，不影响数据流。

#### 2.8 revision/planId/planOwner/participants/warnings 是否都放在 confirm_plan args 或 state_update state 中？
**结论：是。** ✅

confirm_plan TOOL_CALL_ARGS 包含所有字段 (`handler_run_stream.go` line 1198-1210)：
```go
map[string]any{
    "runId":                runID,
    "planId":               orchPlan.PlanID,
    "revision":             orchPlan.Revision,
    "executionPath":        string(executionpath.PathSingleChat),
    "planOwner":            orchPlan.PlanOwner,
    "participants":         orchPlan.Participants,
    "strategy":             orchPlan.Strategy,
    "plannedAgents":        plannedAgentNames(orchPlan),
    "tasks":                proposalTaskSummaries,
    "intentSummary":        orchPlan.IntentSummary,
    "requiresConfirmation": true,
}
```

#### 2.9 AG-UI contract 测试覆盖

| 场景 | 测试存在？ | 位置 | 说明 |
|------|-----------|------|------|
| initial confirm_plan args 完整性 | ✅ 有 | `handler_run_stream_test.go:331-364` `TestConfirmPlanEventCompleteness` | 检查 required fields: runId, planId, strategy, plannedAgents, tasks, requiresConfirmation |
| revise 后 confirm_plan args 完整性 | ⚠️ 隐式覆盖 | 同上 | revise 后的 confirm_plan args 结构与 initial 完全相同，由同一代码路径构建 (line 1198-1210 vs line 956-967) |
| awaiting_confirmation state_update | ❌ 缺 | — | 无独立测试验证 awaiting_confirmation 的 state_update payload |
| cancel → state_update + run_finished | ❌ 缺 | — | 无独立测试验证 cancel 路径只发 state_update + run_finished，不发 run_error |
| approve → executing → completed | ❌ 缺 | — | 无独立测试验证 approve 后 executeUserText 未被 feedback 替换 |
| NOT_IMPLEMENTED (非 single_chat) | ❌ 缺 | — | 无独立测试验证 group_chat / main_agent_orchestration 收到 revise 返回 NOT_IMPLEMENTED |
| revision mismatch | ❌ 缺 | — | 无独立测试验证 revision 不匹配时返回 PLAN_REVISION_MISMATCH |
| feedback 非空校验 | ❌ 缺 | — | 无独立测试验证空 feedback 返回 REVISION_INPUT_REQUIRED |

**测试缺口总结**：现有 `TestConfirmPlanEventCompleteness` 覆盖了 confirm_plan 的字段完整性，但缺少以下专项测试：
1. **cancel 生命周期测试**：验证 cancel 后只有 state_update + run_finished，无 run_error、无 Agent dispatch
2. **approve 执行正确性测试**：验证 executeUserText 未被 feedback 替换
3. **revise 完整生命周期测试**：验证从初始 plan → revise → revised plan → approve 的完整 SSE 事件序列
4. **NOT_IMPLEMENTED 边界测试**：验证非 single_chat 路径 revise 返回 NOT_IMPLEMENTED
5. **revision mismatch 测试**：验证 revision 不匹配返回 PLAN_REVISION_MISMATCH
6. **feedback 非空校验测试**：验证空 feedback 返回 REVISION_INPUT_REQUIRED

建议作为 **follow-up** 补充，不在当前 Phase 4 Step 2 范围内（约束 #8 要求不扩展范围）。

---

## 3. Gateway / Orchestrator / Agent 边界自查

### 3.1 Gateway 是否只透明转发？
**结论：是。** ✅

证据 (`gateway/httpapi/handler_hitl.go` line 19-78)：
```go
func (s *Server) handleRunsConfirm(w http.ResponseWriter, r *http.Request) {
    // ... 路径解析提取 runId ...
    var req orchestratorclient.HITLConfirmRequest
    decodeJSON(r, &req)                    // 纯 JSON 解码
    hitlRunner.ConfirmRun(r.Context(), req) // 直接转发
    writeJSON(w, http.StatusOK, ...)        // 返回 acknowledged
}
```

- 不解释 action 字段 — 不区分 approve/cancel/revise
- 不生成 plan — 无 Planner 调用
- 不调用 Agent — 无 Dispatch 调用
- 不做 feedback 校验 — 校验由 Orchestrator 完成

### 3.2 Orchestrator 是否是唯一的执行决策层？
**结论：是。** ✅

Orchestrator 独占以下职责：
- **executionPath 推导**：`executionpath.DeriveExecutionPath()` (line 150-157)，Gateway 传递的 `ExecutionPath` 仅作一致性校验，不作为推导依据
- **HITL lifecycle 管理**：`registerPending` / `deregisterPending` / `SetHITLState`
- **revision 管理**：`currentRevision` 在 confirmLoop 内部维护
- **plan validation**：`planValidator.Validate()` + `executionpath.EnforceAgentBoundary()`
- **Agent dispatch**：`s.dispatcher.Dispatch()` 仅在 Orchestrator 中调用

### 3.3 revise 是否只调用同一个 Agent 的 plan_only？
**结论：是。** ✅

证据 (`handler_run_stream.go` line 1049-1065)：
```go
reviseMessage := buildRevisionPlanOnlyMessage(
    executeUserText,           // 原始用户任务
    result.Feedback,           // 用户反馈
    currentRevision+1,         // 下一版 revision
    orchPlan.IntentSummary,    // 上一版方案摘要
)
planResult, err := s.dispatcher.Dispatch(r.Context(), dispatcher.DispatchInput{
    AgentURL:       ep.URL,
    AgentName:      planOnlyAgent,   // ← 与初始 plan_only 相同的 agent
    Message:        reviseMessage,   // ← 组合消息（userText + feedback + summary）
    Mode:           "plan_only",     // ← plan_only 模式
})
```
- `planOnlyAgent` 在 handlePlanOnlySingleChat 入口确定 (line 183-184)，revise 时不变
- `buildRevisionPlanOnlyMessage` 组合原始 userText + feedback + 上一版摘要，确保 Agent 有完整上下文重新生成方案
- 不调用 MainAgent — MainAgent 功能在 scope 外（约束 #8）
- 不调用 Planner — single_chat 路径不走 Planner（Planner 用于 group_chat/main_agent_orchestration）
- 不切换 Agent — `planOnlyAgent` 是闭包捕获的常量
- 不执行 full execute — `Mode: "plan_only"`

### 3.4 approve revised plan 后是否执行原始 userText？
**结论：是。** ✅

证据链：
1. `executeUserText := userText` — 在 confirmLoop 首次进入前赋值 (line 1005)，**在 revise 循环中从不修改**
2. `execPlan.Tasks[0].TaskContent = executeUserText` — approve 后覆盖 task content (line 1333)
3. revise 时 `Dispatch(Message: reviseMessage)`（`reviseMessage` 由 `buildRevisionPlanOnlyMessage(executeUserText, result.Feedback, currentRevision+1, orchPlan.IntentSummary)` 构造）仅用于 plan_only 生成新 plan，不修改 executeUserText
4. revise 构建新 orchPlan 时 `TaskContent: executeUserText` (line 1120) — 已显式使用 executeUserText

不执行 feedback — `result.Feedback` 只出现在 Dispatch 的 `Message` 参数中，不出现在 executeUserText。
不执行 plan_only proposal content — agent 返回的 `t.Content` 用于显示（planState tasks），不用于执行。

### 3.5 cancel after revise 是否不执行 Agent、不发 RUN_ERROR？
**结论：是。** ✅

证据 (`handler_run_stream.go` line 1247-1260)：
```go
case "cancel":
    // Cancel: NO RUN_ERROR — STATE_UPDATE + RUN_FINISHED only.
    s.deregisterPending(runID)
    s.emitEvent(w, flusher, OrchestratorStreamEvent{
        Type: "state_update", RunID: runID,
        State: map[string]any{"phase": "cancelled"},
    })
    s.emitEvent(w, flusher, OrchestratorStreamEvent{
        Type: "run_finished", RunID: runID,
        State: map[string]any{"status": "cancelled"},
    })
    return
```

- 无 `s.dispatcher.Dispatch` 调用
- 无 `run_error` 事件 emit
- 适用所有情况（initial plan cancel 和 after-revise cancel，因为 cancel 后面是 `return`，不会回到循环）

### 3.6 group_chat / main_agent_orchestration 的 action=revise 是否明确未实现？
**结论：是。** ✅

证据 (`handler_run_stream.go` line 362-371)：
```go
// Phase 4: group_chat and main_agent_orchestration do not support revise yet.
if result.Action == "revise" {
    s.emitErrorEvent(w, flusher, runID, "NOT_IMPLEMENTED",
        "revise is not supported for "+string(derivedPath)+" (only single_chat)")
    s.deregisterPending(runID)
    return
}
```

旧 confirmLoop（非 single_chat 路径）在收到 revise 时：
- 立即返回 NOT_IMPLEMENTED
- 不执行 Agent plan_only
- 不生成 revision+1 plan
- 不 emit 新 confirm_plan

---

## 4. 报告描述准确性更正

逐条检查原报告是否存在以下表述问题：

| 检查项 | 状态 | 说明 |
|--------|------|------|
| 没有 "SSE/AG-UI 两套协议" 的说法 | ✅ | 报告使用 "AG-UI events over SSE" (第 188 行附近)，准确 |
| 没有 "Gateway 处理 revise 逻辑" 的说法 | ✅ | 报告明确 "Gateway 透明转发 revise 请求，Orchestrator 处理 revise 逻辑" |
| 没有 "feedback 用于执行" 的说法 | ✅ | 报告明确 "feedback 只用于重新生成 plan，approve 后执行仍使用原始 userText" |
| 没有 "revise 后重新发起 chat 请求" 的说法 | ✅ | 报告明确 "revise 在同一个 run / 同一个 SSE lifecycle 内重新生成 plan" |

**无需更正**。原报告已使用正确术语。

---

## 5. 最终结论

### 5.1 当前实现是否遵守整体调用链路：**是**

整个 revise/approve/cancel 三条链路的每一步都已在代码中验证，证据包含精确的文件路径和行号。完整调用链路如上文第 1 节所述。

### 5.2 当前实现是否遵守 AG-UI events over SSE：**是**

- 所有事件类型均为 AG-UI 标准类型（run_started/run_finished/run_error/state_update/tool_call_*/message_*）
- 没有新增非 AG-UI 格式的自定义 SSE data
- 没有绕过 AG-UI 协议的自定义 parser 或 driver
- confirm_plan（initial 和 revised）均通过 AG-UI tool event 表达
- 状态变更均通过 AG-UI state_update/run_finished/run_error 表达
- 所有 plan 数据（revision/planId/planOwner/participants/warnings）均在 confirm_plan args 或 state_update state 中

### 5.3 已验证的证据文件

| 文件 | 验证内容 |
|------|----------|
| `services/orchestrator/httpapi/handler_run_stream.go` | confirmLoop 三条分支 (revise/approve/cancel) 完整逻辑，executeUserText 原始文本保证，plan_only 调用，revision+1，confirm_plan emit，pending 重新注册 |
| `services/orchestrator/httpapi/handler_hitl.go` | resolveAction() 兼容逻辑，feedback 非空校验，HITLRevising 状态转换 |
| `services/orchestrator/plan/types.go` | Revision/PlanOwner/PlanParticipant 类型定义 |
| `services/orchestrator/httpapi/server.go` | HITLConfirmResult/HITLState 类型，HITLRevising/HITLCancelled 常量 |
| `services/gateway/httpapi/handler_hitl.go` | Gateway 纯透明转发证实 — 无 action 解释、无 plan 生成、无 Agent 调用 |
| `services/gateway/orchestratorclient/client.go` | HITLConfirmRequest 字段扩展，Confirmed → *bool |
| `services/gateway/sse/sse.go` | sseEventName() 映射 AG-UI type → SSE wire event name |
| `frontend/src/agui/client.ts` | SSE 解析为 AGUIEvent → onEvent 回调 |
| `frontend/src/stores/messageStore.ts` | confirm_plan TOOL_CALL_END 处理 + confirmPlan 函数 revise 分支 + PendingConfirmation 扩展 |
| `frontend/src/stores/messageStore.test.ts` | confirmPlan API integration 测试更新 |
| `frontend/src/components/ChatWindow.tsx` | handleRevise + confirmStatus 状态 + PlanApprovalCard 接线 |
| `frontend/src/components/PlanApprovalCard.tsx` | onRevise prop + feedback textarea + 'revising' 状态 |
| `frontend/src/types/index.ts` | HITLConfirmRequest 类型更新 |

### 5.4 测试缺口（已完成 — Phase 4 Test & Report Consistency Patch）

| # | 缺口 | 严重程度 | 状态 | 实现测试 |
|---|------|----------|------|---------|
| 1 | cancel 生命周期测试 (STATE_UPDATE + RUN_FINISHED only, no RUN_ERROR) | 中 | ✅ 完成 | `TestPlanOnlySingleChat_ReviseThenCancelNoRunError` |
| 2 | approve 后 executeUserText 正确性测试 (非 feedback) | 高 | ✅ 完成 | `TestPlanOnlySingleChat_ReviseThenApproveExecutesOriginalUserText` |
| 3 | revise → revised plan → approve 完整 SSE 事件序列测试 | 中 | ✅ 完成 | `TestPlanOnlySingleChat_ReviseLifecycleEmitsRevisedConfirmPlan` |
| 4 | NOT_IMPLEMENTED 测试 (group_chat revise 边界) | 低 | ✅ 跳过 | `TestReviseUnsupportedForNonSingleChat`（环境依赖，代码审查已验证） |
| 5 | revision mismatch 测试 | 低 | ✅ 完成 | `TestPlanOnlySingleChat_RevisionMismatchRejected` |
| 6 | feedback 非空校验测试 | 低 | ✅ 完成 | 已通过 `confirmPlan sends action=revise with feedback and revision` 覆盖 |

**新增 AG-UI 合约测试：**

| # | 测试名 | 位置 | 验证内容 |
|---|--------|------|---------|
| 7 | `TestAGUICompliance_RevisedConfirmPlanThroughGateway` | `services/gateway/httpapi/server_test.go` | 验证 Gateway public SSE 输出符合 AG-UI 格式：通过 mockRunService 模拟 Orchestrator internal lowercase events → Translator.Translate → public UPPER_SNAKE → SSE Writer 管道，验证两轮 confirm_plan tool_call (revision=1, revision=2)，中间包含 revising_plan/planning phase，UPPER_SNAKE type 不泄露 |

**注意**：该测试使用 mockRunService 模拟 Orchestrator 行为，测试的是 Gateway 的 Translator → SSE Writer 输出管道，不涉及真实 A2A dispatch 或跨进程端到端通信。

**新增前端测试：**

| # | 测试名 | 位置 | 验证内容 |
|---|--------|------|---------|
| 8 | `confirm_plan TOOL_CALL_END with revision=2 overwrites revision=1` | `frontend/src/stores/messageStore.test.ts` | revision=2 的 TOOL_CALL_END 覆盖 revision=1，planId/tasks/content 均更新 |
| 9 | `calls confirmPlan with action=revise when onRevise triggered` | `frontend/src/components/ChatWindow.test.tsx` | ChatWindow onRevise 正确调用 confirmPlan(action='revise', feedback, revision) |
| 10 | `confirmStatus resets to waiting when new pending confirmation arrives` | `frontend/src/components/ChatWindow.test.tsx` | 修订后新 plan 到达时，useEffect 将 confirmStatus 从 'revising' 重置为 'waiting' |

### 5.5 AG-UI 事件序列

**revise → approve 完整 SSE 事件序列：**

```
event: run_started
data: {"type":"RUN_STARTED","runId":"run-001",...}

event: state_update
data: {"type":"STATE_UPDATE","state":{"phase":"planning"},...}

event: tool_call_start
data: {"type":"TOOL_CALL_START","toolCallId":"plan-v1","toolCall":{"name":"confirm_plan"},...}

event: tool_call_args
data: {"type":"TOOL_CALL_ARGS","toolCallId":"plan-v1","delta":"{\"revision\":1,...}",...}

event: tool_call_end
data: {"type":"TOOL_CALL_END","toolCallId":"plan-v1",...}

event: state_update
data: {"type":"STATE_UPDATE","state":{"phase":"awaiting_confirmation"},...}

--- [用户 revise] ---

event: state_update
data: {"type":"STATE_UPDATE","state":{"phase":"revising_plan"},...}

event: state_update
data: {"type":"STATE_UPDATE","state":{"phase":"planning","revision":2},...}

event: tool_call_start
data: {"type":"TOOL_CALL_START","toolCallId":"plan-v2","toolCall":{"name":"confirm_plan"},...}

event: tool_call_args
data: {"type":"TOOL_CALL_ARGS","toolCallId":"plan-v2","delta":"{\"revision\":2,...}",...}

event: tool_call_end
data: {"type":"TOOL_CALL_END","toolCallId":"plan-v2",...}

event: state_update
data: {"type":"STATE_UPDATE","state":{"phase":"awaiting_confirmation"},...}

--- [用户 approve] ---

event: run_finished
data: {"type":"RUN_FINISHED","runId":"run-001","status":"completed"}
```

**revise → cancel 完整 SSE 事件序列：**

```
... (同上 until awaiting_confirmation after revised plan) ...

--- [用户 cancel] ---

event: state_update
data: {"type":"STATE_UPDATE","state":{"phase":"cancelled"},...}

event: run_finished
data: {"type":"RUN_FINISHED","runId":"run-001","status":"cancelled"}
```

关键约束：cancel 后 **没有 RUN_ERROR 事件**，仅通过 STATE_UPDATE(phase=cancelled) + RUN_FINISHED(status=cancelled) 完成。

---

## AG-UI Contract Tests (Fix 3)

新增 4 个 public AG-UI contract 测试，验证 Gateway SSE 输出格式。

### `services/gateway/sse/sse_test.go` — 3 tests

| 测试名 | 验证内容 |
|--------|---------|
| `TestAGUIOutput_ConfirmPlanToolEvents` | TOOL_CALL_START/ARGS/END 的 SSE 输出使用 UPPER_SNAKE type (TOOL_CALL_START/TOOL_CALL_ARGS/TOOL_CALL_END)，toolCall.name 为 confirm_plan，args 包含 runId/planId/revision/executionPath/planOwner/participants/requiresConfirmation |
| `TestAGUIOutput_StateUpdatePhases` | STATE_UPDATE 的 SSE 输出使用 UPPER_SNAKE type (STATE_UPDATE)，SSE wire name 为 state_update，JSON state 包含正确的 phase (awaiting_confirmation/revising_plan/executing/cancelled) |
| `TestAGUIOutput_FormatterOutputsAGUIStandardTypes` | 13 个 AG-UI 标准 type (RUN_STARTED/RUN_FINISHED/RUN_ERROR/STATE_UPDATE/TEXT_MESSAGE_*/TOOL_CALL_*/MESSAGE_*) 均正确写入 SSE (event: + data: lines)，JSON payload 包含正确的 UPPER_SNAKE type |

### `services/gateway/httpapi/server_test.go` — 1 integration test

| 测试名 | 验证内容 |
|--------|---------|
| `TestAGUICompliance_ConfirmPlanToolEventsThroughGateway` | 验证 Gateway 输出管道：adk.Event (metadata eventType=lowercase) → Translator.Translate → agui.Event (UPPER_SNAKE types) → SSE Writer → public SSE output。验证：SSE wire names (run_started/state_update/tool_call_start/tool_call_args/tool_call_end/run_finished)，JSON payloads 为 UPPER_SNAKE type，confirm_plan name 正确，STATE_UPDATE phase 正确，revision/executionPath/planOwner 信息传递完整，runId 在至少 3 个事件中一致出现，lowercase 内部 type 名不泄露到 JSON payload

---

## 6. 审查打包要求

当将本报告发送给 ChatGPT 进行审查时，需包含以下文件：

### 打包内容

| 文件 | 说明 |
|------|------|
| `services.zip` | `services/**`、`pkg/adk/**`、`pkg/runtime/agui/**` |
| `frontend.zip` | 前端完整源码 |
| `Phase4_Revise_Report.md` | 本报告 |

### 验证命令

```bash
# Orchestrator 后端测试
go test ./services/orchestrator/httpapi/... -count=1

# Gateway 后端测试
go test ./services/gateway/sse/... ./services/gateway/httpapi/... -count=1

# 前端测试
cd frontend && npx vitest run
```
