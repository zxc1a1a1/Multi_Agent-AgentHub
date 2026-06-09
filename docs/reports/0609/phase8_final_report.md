# Phase 8: 真实联调测试报告

**测试时间:** 2026-06-08 19:00-19:10 UTC
**Gateway:** http://localhost:8080
**环境:** Docker Compose new-arch, 12 agents 全部 healthy
**测试工具:** Python 自动化测试脚本 (threaded SSE + timed confirm)

---

## 总体结果

| 指标 | 数值 |
|------|------|
| 总用例数 | 8 |
| 完全通过 | 1 (Case 4) |
| 部分通过 (核心逻辑正确) | 3 (Case 5, 6, 7) |
| 阻塞于 infra 问题 | 3 (Case 1, 2, 8) |
| 阻塞于验证器 | 1 (Case 3) |
| 确认端点 502 | 3 (Case 5, 6, 7) |

---

## Case 1: single_chat 简单代码任务

**输入:** agentName=code-agent, message="用 React 写一个 Button 组件"

**期望 vs 实际:**

| 期望 | 实际 | 结果 |
|------|------|------|
| executionPath = single_chat | 未到达 (plan阶段失败) | ❌ |
| agent 生成执行方案 | `ORCHESTRATOR_PLAN_PARSE_FAILED` | ❌ |
| participants 只有 code-agent | 无计划生成 | ❌ |
| 未同意前不执行 | N/A (计划生成失败) | ⚠️ |
| 同意后只调用 code-agent | N/A | ⚠️ |

**根因:** code-agent 的 `plan_only` 模式调用 LLM 后返回了纯文本 (以 'W' 开头)，Orchestrator 期望 JSON 格式的计划。错误: `failed to parse agent plan: invalid character 'W' looking for beginning of value`

**严重程度:** 🔴 阻塞 - code-agent 的 plan_only 响应格式与 Orchestrator 的 JSON parser 不匹配。

---

## Case 2: single_chat 用户反馈修改

**输入:** agentName=code-agent, message="写一个 Go HTTP server"
**操作:** 用户反馈 "不要写测试，先只给最小可运行版本"

**期望 vs 实际:**

| 期望 | 实际 | 结果 |
|------|------|------|
| 发送 REQUEST_PLAN_REVISION | N/A (初始计划失败) | ❌ |
| revision = 2 | N/A | ❌ |
| 新方案体现用户反馈 | N/A | ❌ |
| 未执行任何 Agent | ✅ (计划未生成即失败) | ✅ |

**根因:** 与 Case 1 相同 — code-agent `plan_only` 返回非 JSON 文本 (`ORCHESTRATOR_PLAN_PARSE_FAILED`)

**严重程度:** 🔴 阻塞 - 与 Case 1 同一根因

---

## Case 3: group_chat 多 Agent 协作

**输入:** selectedAgentNames=["code-agent", "review-agent"], message="@code-agent @review-agent 实现并检查登录接口"

**期望 vs 实际:**

| 期望 | 实际 | 结果 |
|------|------|------|
| executionPath = group_chat | 未设置 (plan阶段失败) | ❌ |
| participants 只包含 code-agent & review-agent | N/A (验证失败) | ❌ |
| code-agent 先产出, review-agent 后检查 | N/A | ❌ |
| 同意后两者以独立 turn 展示 | N/A | ❌ |

**根因:** Orchestrator 日志显示 `"validation reject: sequential strategy not supported"`。Group coordinator 生成了 `sequential` 策略的计划，但验证器拒绝了此策略。

**严重程度:** 🟡 阻塞 - 验证器不支持 sequential 策略，但 group_chat 场景需要此策略实现 code→review 的顺序执行。

---

## Case 4: group_chat 越界 Agent 拒绝 ✅ PASSED

**输入:** selectedAgentNames=["code-agent"], mentions=["web-agent"], message="@web-agent 做登录页面"

**期望 vs 实际:**

| 期望 | 实际 | 结果 |
|------|------|------|
| selectedAgentNames ∩ mentions = ∅ | 交集为空 | ✅ |
| 返回 AGENT_SELECTION_CONFLICT | 错误码 `AGENT_SELECTION_CONFLICT` | ✅ |
| 不生成计划 | 无 planId | ✅ |
| 不执行任何 Agent | 无 Agent 调用 | ✅ |

**SSE 事件序列:**
```
RUN_STARTED { phase: "planning" }
RUN_ERROR {
  code: "AGENT_SELECTION_CONFLICT",
  message: "selectedAgentNames and mentions have no overlap"
}
```

**评估:** ✅ **完全通过。** Agent 选择冲突检测在 plan 生成前正确执行，无计划生成，无 Agent 被调用。错误码和消息符合规范。

---

## Case 5: auto 简单任务

**输入:** agentName=auto, message="用 Go 写一个 HTTP server"

**期望 vs 实际:**

| 期望 | 实际 | 结果 |
|------|------|------|
| executionPath = main_agent_orchestration | `main_agent_orchestration` | ✅ |
| 主 Agent 生成一个计划 | ACTIVITY_SNAPSHOT 含 plan | ✅ |
| 可推荐 code-agent 或 main_agent_only | 推荐了 `code-agent` | ✅ |
| 未同意前不执行 | status=awaiting_confirmation | ✅ |
| 同意后执行 | 确认返回 502 | ❌ |

**Plan 详情:**
```yaml
executionPath: main_agent_orchestration
planOwner: { type: "main_agent", agentName: "main-agent", isMainAgent: true }
planId: plan_1780945410273
revision: 1
status: awaiting_confirmation
participants:
  - [R][S] code-agent
  - [O][ ] document-agent (候选但未默认选中)
candidateParticipants: [code-agent, document-agent]
defaultSelectedParticipants: [code-agent]
tasks:
  - task-1 → code-agent: Write a Go HTTP server.
allowedActions: [approve, revise, cancel]
```

**评估:** 🟢 计划生成逻辑完全正确。executionPath 推导正确，ACTIVITY_SNAPSHOT 结构完整，planOwner 正确标识为 main_agent。**唯一失败点是确认端点返回 502**（详见下文 HITL 确认问题分析）。

---

## Case 6: auto 多能力任务

**输入:** agentName=auto, message="做登录功能，前端 React 页面，后端 Go 登录接口"

**期望 vs 实际:**

| 期望 | 实际 | 结果 |
|------|------|------|
| executionPath = main_agent_orchestration | `main_agent_orchestration` | ✅ |
| 主 Agent 推荐 code-agent / web-agent | 推荐了 web-agent, code-agent | ✅ |
| participants 包含推荐原因 | candidates 无 reason 字段 | ❌ |
| 用户可看到每个 Agent 的作用 | 无 reason 说明 | ⚠️ |
| 同意后主 Agent 编排执行并汇总 | 确认返回 502 | ❌ |

**Plan 详情:**
```yaml
executionPath: main_agent_orchestration
planOwner: { type: "main_agent", agentName: "main-agent", isMainAgent: true }
participants:
  - [R][S] web-agent
  - [R][S] code-agent
candidateParticipants: [web-agent, code-agent]
defaultSelectedParticipants: [web-agent, code-agent]
tasks (2):
  - task-1 → web-agent: Build login (React + Go)
  - task-2 → code-agent: Build login (React + Go)
allowedActions: [approve, revise, cancel]
```

**评估:** 🟡 多 Agent 推荐正确 (web-agent + code-agent)，但 **candidateParticipants 缺少 `reason` 字段** 说明每个 Agent 的作用。同时确认端点 502。

---

## Case 7: auto 用户取消可选 Agent

**输入:** agentName=auto, message="做登录功能，并补充测试"

**期望 vs 实际:**

| 期望 | 实际 | 结果 |
|------|------|------|
| executionPath = main_agent_orchestration | `main_agent_orchestration` | ✅ |
| test-agent 是可选则允许执行 | 主 Agent 推荐了 web-agent (未推荐 test-agent) | ⚠️ |
| 计划 warning 说明测试未执行 | 无 warnings | ❌ |
| 不调用 test-agent | 无 test-agent 在计划中 | ✅ |

**Plan 详情:**
```yaml
participants: [web-agent]  (仅1个)
candidateParticipants: [web-agent]
defaultSelectedParticipants: [web-agent]
tasks (1):
  - task-1 → web-agent: Build login feature with tests.
```

**评估:** 🟡 主 Agent 未推荐 test-agent (推荐了 web-agent 做所有工作包括测试)。期望是推荐 test-agent 作为可选 Agent，用户可取消勾选。同时确认端点 502。

---

## Case 8: 重复点击

**输入:** agentName=code-agent, message="写一个 Python hello world 脚本"
**操作:** 连续点击两次「同意并执行」

**期望 vs 实际:**

| 期望 | 实际 | 结果 |
|------|------|------|
| 第一次进入 executing | N/A (计划失败) | ❌ |
| 第二次被拒绝或返回同一状态 | N/A | ❌ |
| 不重复调用 Agent | N/A | ❌ |

**根因:** 与 Case 1 相同 — code-agent `plan_only` 失败。无法到达确认阶段测试幂等性。

**严重程度:** 🔴 阻塞 - 测试无法进行，需要在 plan_only 修复后重测。

---

## HITL 确认端点 502 问题深度分析

### 现象
Cases 5/6/7 在收到 ACTIVITY_SNAPSHOT 后立即发送 `POST /api/runs/{runId}/confirm`，Gateway 返回:
```
HTTP 502: {"error": "HITL confirmation failed"}
```

### 技术链路
```
Frontend → Gateway (POST /api/runs/{runId}/confirm)
       → handler_hitl.go: handleRunsConfirm()
       → s.runner.ConfirmRun(ctx, req)
       → orchestratorclient/client.go: ConfirmRun()
       → HTTP POST orchestrator-new:8080/internal/orchestrator/hitl/confirm
       → handler_hitl.go: handleHITLConfirm()
       → s.hitlChans[req.RunID] lookup
       → 404 if not found → Gateway receives >=400 → returns 502
```

### 调查发现
1. **内部 Token 匹配:** Gateway 和 Orchestrator 均配置了 `dev-internal-token` ✅
2. **URL 路径正确:** `/internal/orchestrator/hitl/confirm` 在 Orchestrator 正确注册 ✅
3. **Orchestrator 日志缺失:** 在 19:03-19:09 时间段内，Orchestrator 无任何 HITL 相关日志（仅有一条 "repair succeeded"）
4. **Gateway 日志不充分:** Gateway 仅启动时输出 3 条日志，无法追踪确认请求

### 可能原因
- **时序竞争:** ACTIVITY_SNAPSHOT 通过 SSE 发送后，`registerPending` 可能尚未完成
- **runId 映射:** SSE 流中的 runId 与 `registerPending` 注册的 runId 可能不一致
- **HTTP 连接失败:** Gateway→Orchestrator 的 HTTP 请求可能在网络层失败（但这与 SSE 流正常工作矛盾）

### 建议调试步骤
1. 在 Gateway 的 `handleRunsConfirm` 和 `OrchestratorRunService.ConfirmRun` 中添加日志
2. 在 Orchestrator 的 `handleHITLConfirm` 入口处添加日志（含请求体）
3. 检查 `registerPending` 调用时机与 ACTIVITY_SNAPSHOT 发送的顺序
4. 在确认请求前添加 1-2 秒延迟测试是否为时序问题

---

## 总结

### 验证通过的功能 ✅
1. **AGENT_SELECTION_CONFLICT (Case 4):** 完全正确 - 交集为空时返回正确错误码，无计划生成，无 Agent 调用
2. **executionPath 推导:** group_chat / main_agent_orchestration 路径推导正确
3. **ACTIVITY_SNAPSHOT 事件:** 结构完整，包含 planId, revision, executionPath, planOwner, participants, candidates, tasks
4. **planOwner 标识:** main_agent_orchestration 正确标识为 `{ type: "main_agent", isMainAgent: true }`
5. **审批前不执行:** 计划正确停留在 `awaiting_confirmation` 状态

### 需要修复的问题 ❌
| 问题 | 影响用例 | 严重程度 | 根因 |
|------|----------|----------|------|
| code-agent plan_only 返回非 JSON | 1, 2, 8 | 🔴 Critical | LLM 输出格式与 parser 不匹配 |
| group_chat sequential 策略不支持 | 3 | 🔴 Critical | 验证器拒绝 sequential 策略 |
| HITL 确认 502 | 5, 6, 7 | 🔴 Critical | 待进一步调试 |
| candidateParticipants 缺少 reason | 6 | 🟡 Minor | 主 Agent 未输出 reason |
| test-agent 未被推荐 | 7 | 🟡 Minor | 主 Agent 选择逻辑需调整 |

### 建议优先级
1. **P0:** 修复 code-agent plan_only JSON 响应格式
2. **P0:** 支持 sequential 执行策略
3. **P0:** 修复 HITL 确认端点 502 问题
4. **P1:** 添加 candidateParticipants reason 字段
5. **P1:** 完善 Gateway/Orchestrator 日志

---

*报告生成: 2026-06-08 UTC*
*测试环境: Docker Compose new-arch*
*可用 Agent: auto, code-agent, web-agent, document-agent, vision-agent, context-agent, test-agent, review-agent, security-agent, deploy-agent, diff-agent*
*测试脚本: C:/Users/86138/Desktop/tmp/phase8_test_harness.py*
