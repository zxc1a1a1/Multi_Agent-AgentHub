# AgentHub 统一 MainAgent 多路径计划确认方案

> 本文是对原《AgentHub 多路径计划确认方案：Single Chat、Group Chat 与 Auto 主 Agent 编排》的新版修订。
>
> 核心变化：**保留 `single_chat` / `group_chat` / `main_agent_orchestration` 三种用户模式，但计划生成统一由 Orchestrator 内部 MainAgent 完成。**
>
> 这不是把所有模式都变成 auto，而是：
>
> ```text
> MainAgent 统一生成计划；
> executionPath 决定 MainAgent 的可用 Agent 边界；
> Validator 强制边界；
> Orchestrator 负责确认、修订、取消、幂等和 dispatch。
> ```

---

## 0. 本次调整结论

原方案中三条路径分别有不同计划生成者：

```text
single_chat              → 当前 Agent plan_only 生成方案
group_chat               → GroupCoordinator 生成生产计划
main_agent_orchestration → MainAgent 生成执行计划
```

这种设计语义清晰，但工程实现容易分叉：

```text
1. single_chat 依赖每个 Agent 稳定输出 plan_only JSON。
2. group_chat 需要额外 GroupCoordinator。
3. auto 需要 MainAgent。
4. Planner / Validator / Executor / Frontend 容易各认一套 strategy。
5. 旧 RulePlanner / LLMPlanner 容易混入非 auto 路径，造成越界选 Agent。
```

新版方案改为：

```text
所有路径都由 MainAgent 生成一个计划；
但 MainAgent 在不同 executionPath 下拥有不同的 Agent 可用边界。
```

三条路径仍然保留：

```text
1. single_chat
   用户知道要找一个 Agent。
   MainAgent 只能围绕当前选中 Agent 生成计划。
   用户确认后，Orchestrator 只 dispatch 当前 Agent。

2. group_chat
   用户知道一组 Agent 要协作。
   MainAgent 只能围绕 selectedAgentNames / mentions / activeGroupAgents 生成生产计划。
   不允许自动补充范围外 Agent。
   用户确认后，Orchestrator 按计划 dispatch 多个 Agent 独立 turn。

3. auto / main_agent_orchestration
   用户不知道该找谁，或选择 auto。
   MainAgent 可以从 all enabled agents 中推荐 participants。
   用户确认后，Orchestrator 按 selectedParticipants dispatch 子 Agent，并可由 MainAgent 汇总。
```

关键原则：

```text
1. 计划生成统一：MainAgent 是唯一计划生成者。
2. 路径语义保留：single_chat / group_chat / auto 仍然代表不同用户意图。
3. 非 auto 不允许越界推荐 Agent。
4. auto 才允许 MainAgent 从全量 enabled agents 中推荐参与者。
5. Validator 是硬边界，不能只靠 prompt 约束 MainAgent。
6. Orchestrator 是唯一状态与执行权威层。
7. Gateway 只转发，不做语义路由，不生成计划，不调用 Agent。
8. 新计划确认主路径使用 AG-UI `ACTIVITY_SNAPSHOT`，不再使用 fake `confirm_plan` TOOL_CALL。
```

---

## 1. 核心架构

### 1.1 新主链路

```text
POST /api/chat
  ↓
Gateway 透传 agentName / selectedAgentNames / mentions / requestedPath
  ↓
Orchestrator deriveExecutionPath
  ↓
Orchestrator buildAvailableBoundary
  ↓
MainAgent.GeneratePlan(userText, executionPath, availableBoundary)
  ↓
PathAwarePlanValidator.Validate(plan, executionPath, availableBoundary)
  ↓
registerPendingPlan / registerHITL
  ↓
emit AG-UI ACTIVITY_SNAPSHOT
  ↓
等待 approve / revise / cancel
  ↓
approve:
    校验 runId / planId / revision / selectedParticipants / idempotencyKey
    Orchestrator dispatch selectedParticipants
  ↓
Agent 独立 turn 输出
  ↓
可选 MainAgent 汇总
```

### 1.2 组件职责

| 组件 | 职责 | 禁止事项 |
|---|---|---|
| Frontend | 选择 Agent、提交消息、展示 ActivitySnapshot、发送 approve/revise/cancel | 不直接调用 Orchestrator，不自行解释私有 SSE |
| Gateway | 鉴权、基础字段校验、请求透传、AG-UI 协议转换 | 不生成计划、不调用 Agent、不做语义路由、不改写 participants |
| Orchestrator | executionPath 推导、边界构建、MainAgent 调用、Validator、PendingPlan、HITL、dispatch | 不把非 auto 请求交给旧 RulePlanner / LLMPlanner 选择 Agent |
| MainAgent | 统一生成计划、修订计划、可选最终汇总 | 非 auto 模式不得扩展 participants |
| Validator | 强制检查路径边界、participants、selectedParticipants、revision、idempotency | 不参与语义选 Agent |
| Dispatcher / Executor | approve 后按确认计划调度 Agent | 不自行添加计划外 Agent |

---

## 2. 顶层执行路径

### 2.1 ChatExecutionPath

```ts
export type ChatExecutionPath =
  | "single_chat"
  | "group_chat"
  | "main_agent_orchestration";
```

### 2.2 三条路径的新含义

| executionPath | 用户意图 | 谁生成计划 | MainAgent 可用边界 | 谁执行 |
|---|---|---|---|---|
| `single_chat` | 用户明确知道要找一个 Agent | MainAgent | 当前选中 Agent | 当前选中 Agent |
| `group_chat` | 用户明确知道一组 Agent 协作 | MainAgent | selectedAgentNames / mentions / activeGroupAgents | 已确认多个 Agent |
| `main_agent_orchestration` | 用户不知道找谁，选择 auto 或未指定 Agent | MainAgent | all enabled agents | Orchestrator dispatch selectedParticipants，MainAgent 可汇总 |

注意：

```text
所有路径都经过 MainAgent 生成计划；
但只有 main_agent_orchestration 允许 MainAgent 推荐新的 Agent。
```

---

## 3. 请求字段与 executionPath 推导

### 3.1 ChatRequest

```ts
export type ChatRequest = {
  conversationId?: string;
  message: string;

  agentName?: string;              // legacy 单选 Agent
  selectedAgentNames?: string[];   // UI 手动选择 Agent
  mentions?: string[];             // 用户消息中的 @Agent
  requestedPath?: ChatExecutionPath | "auto";
};
```

### 3.2 推导规则

```text
requestedPath == "auto"                         → main_agent_orchestration
agentName == "auto"                             → main_agent_orchestration
selectedAgentNames.length == 0 && mentions == 0 → main_agent_orchestration

selectedAgentNames.length == 1 && mentions == 0 → single_chat
agentName 非空且非 auto                         → single_chat

selectedAgentNames.length > 1                   → group_chat
mentions.length > 1                              → group_chat
selectedAgentNames.length > 0 && mentions.length > 0 → group_chat，且 mentions 只能缩小 selectedAgentNames 范围

mentions.length == 1 且无 selectedAgentNames      → 默认 single_chat；
                                                   如果当前 conversation 已是 group_chat，可按 group_chat 处理。
```

### 3.3 AvailableBoundary 构建规则

`availableBoundary` 是 MainAgent 在本次请求中允许使用的 Agent 集合。

```ts
export type AvailableBoundary = {
  executionPath: ChatExecutionPath;
  agentNames: string[];
  source:
    | "single_selected"
    | "group_selected"
    | "mention"
    | "active_group"
    | "main_agent_all_enabled";
};
```

边界规则：

```text
single_chat:
  availableBoundary = [当前选中 Agent]

group_chat:
  如果 selectedAgentNames 和 mentions 同时存在：
    availableBoundary = intersection(selectedAgentNames, mentions)
  否则：
    availableBoundary = selectedAgentNames || mentions || activeGroupAgents

main_agent_orchestration:
  availableBoundary = all enabled agents
```

冲突规则：

```text
selectedAgentNames 与 mentions 同时存在且交集为空：
  返回 AGENT_SELECTION_CONFLICT
  不调用 MainAgent
  不生成计划
  不执行任何 Agent
```

---

## 4. MainAgent 统一计划生成

### 4.1 MainAgent 不是普通 Agent

MainAgent 是 Orchestrator 内部组件：

```text
1. 不注册为普通 A2A Agent。
2. 不进入 StaticAgentRegistry。
3. 不出现在前端 Agent 下拉框。
4. 不被 health checker 探测。
5. 不通过普通 dispatcher URL 调用。
6. 不直接执行用户任务。
```

MainAgent 的职责：

```text
1. 根据 userText、executionPath、availableBoundary 生成一个计划。
2. 为 participants 生成 role、reason、required、userSelectable。
3. 为 plan.steps 分配 assignedAgent。
4. 根据 feedback 生成 revision + 1 的新版计划。
5. 在执行完成后，可选生成最终汇总。
```

### 4.2 MainAgent 输入

```ts
export type MainAgentPlanInput = {
  runId: string;
  conversationId?: string;
  userText: string;

  executionPath: ChatExecutionPath;
  availableBoundary: AvailableBoundary;

  enabledAgents: AgentDescriptor[];

  previousPlan?: ExecutionPlan;
  feedback?: string;
  selectedParticipants?: string[];
  revision: number;
};
```

### 4.3 MainAgent 输出

```ts
export type MainAgentPlanOutput = {
  planId: string;
  revision: number;

  executionPath: ChatExecutionPath;

  plannerOwner: PlannerOwner;
  executionOwner: ExecutionOwner;

  title: string;
  summary: string;

  plan: ExecutionPlan;

  participants: PlanParticipant[];
  defaultSelectedParticipants: string[];
  requiredParticipants: string[];

  warnings?: string[];
};
```

---

## 5. plannerOwner 与 executionOwner

原文档只有 `planOwner`，新版建议拆成两个概念：

```text
plannerOwner：谁生成计划
executionOwner：谁实际执行或编排执行
```

原因：

```text
新版所有路径都由 MainAgent 生成计划；
但 single_chat 的执行者仍然是当前 Agent；
group_chat 的执行者仍然是用户选定的一组 Agent；
auto 的执行者是 confirmed selectedParticipants，MainAgent 可负责汇总。
```

### 5.1 类型定义

```ts
export type PlannerOwner = {
  type: "main_agent";
  agentName: "main-agent";
  internal: true;
};

export type ExecutionOwner =
  | {
      type: "agent";
      agentName: string;
    }
  | {
      type: "group";
      agentNames: string[];
    }
  | {
      type: "main_agent_orchestration";
      agentName: "main-agent";
      selectedParticipants: string[];
    };
```

### 5.2 路径映射

```text
single_chat:
  plannerOwner = main-agent
  executionOwner = { type: "agent", agentName: 当前选中 Agent }

group_chat:
  plannerOwner = main-agent
  executionOwner = { type: "group", agentNames: allowedParticipants }

main_agent_orchestration:
  plannerOwner = main-agent
  executionOwner = { type: "main_agent_orchestration", selectedParticipants }
```

兼容建议：

```text
如果现有代码短期内仍使用 planOwner，可以先让 planOwner = plannerOwner。
但前端展示时必须明确区分：
- “计划由 MainAgent 生成”
- “执行将由哪些 Agent 完成”
```

---

## 6. 三条路径的交互设计

## 6.1 single_chat：单 Agent 边界内的 MainAgent 计划

适用场景：

```text
用户知道要找谁，例如：
- 选择 code-agent：写一个 Go HTTP server。
- 选择 web-agent：做一个 React 登录页面。
- 选择 Claude Code：写一个 React Button 组件。
```

流程：

```text
用户选择一个 Agent
  ↓
Orchestrator 判定 executionPath = single_chat
  ↓
availableBoundary = [当前 Agent]
  ↓
MainAgent 在该边界内生成计划
  ↓
Validator 校验 participants / steps 不越界
  ↓
ACTIVITY_SNAPSHOT 展示计划
  ↓
用户同意 / 反馈修改 / 取消
  ↓
同意后，Orchestrator 只 dispatch 当前 Agent
```

行为要求：

```text
1. participants 只能有当前 Agent。
2. requiredParticipants = [当前 Agent]。
3. userSelectable = false。
4. MainAgent 不允许推荐其他 Agent。
5. 如果任务看起来需要其他 Agent，只能写 warnings，不能自动添加。
6. 用户反馈时，MainAgent 仍在 [当前 Agent] 边界内生成 revision + 1。
7. 执行阶段只调用当前 Agent。
```

single_chat 示例：

```json
{
  "executionPath": "single_chat",
  "plannerOwner": {
    "type": "main_agent",
    "agentName": "main-agent",
    "internal": true
  },
  "executionOwner": {
    "type": "agent",
    "agentName": "code-agent"
  },
  "participants": [
    {
      "agentName": "code-agent",
      "defaultSelected": true,
      "required": true,
      "userSelectable": false,
      "role": "executor",
      "source": "user_selected",
      "reason": "用户明确选择该 Agent。"
    }
  ],
  "defaultSelectedParticipants": ["code-agent"],
  "requiredParticipants": ["code-agent"],
  "plan": {
    "strategy": "agent_self_execute",
    "steps": [
      {
        "stepId": "step-1",
        "title": "实现最小 HTTP server",
        "task": "使用 Go net/http 实现一个最小可运行 HTTP server。",
        "assignedAgent": "code-agent",
        "outputType": "code"
      }
    ],
    "reasons": [
      "用户明确选择 code-agent。",
      "任务可由单 Agent 完成。"
    ]
  }
}
```

---

## 6.2 group_chat：多 Agent 边界内的 MainAgent 生产计划

适用场景：

```text
用户知道需要一组 Agent 协作，例如：
- @code-agent @review-agent 实现并检查登录接口。
- 选择 code-agent + web-agent 做前后端登录功能。
- 群聊中让多个 Agent 依次给意见或产出。
```

流程：

```text
用户选择多个 Agent 或 @ 多个 Agent
  ↓
Orchestrator 判定 executionPath = group_chat
  ↓
availableBoundary = selectedAgentNames / mentions / activeGroupAgents
  ↓
MainAgent 只在 availableBoundary 内生成生产计划
  ↓
Validator 校验 participants / steps 不越界
  ↓
ACTIVITY_SNAPSHOT 展示计划
  ↓
用户同意 / 反馈修改 / 取消
  ↓
同意后，Orchestrator 按 steps dispatch 多 Agent 独立 turn
```

行为要求：

```text
1. participants 只能来自 availableBoundary。
2. steps[].assignedAgent 只能来自 availableBoundary。
3. 不允许从 all enabled agents 中自动补充 Agent。
4. 如果能力不足，只能写 warnings。
5. 用户反馈时，MainAgent 基于原 participants + feedback 生成 revision + 1。
6. 同意后，各 Agent 以独立 message / independent turn 形式输出。
7. MainAgent 可选最终汇总，但不能吞掉各 Agent 的独立输出。
```

group_chat 示例：

```json
{
  "executionPath": "group_chat",
  "availableBoundary": ["code-agent", "review-agent"],
  "plannerOwner": {
    "type": "main_agent",
    "agentName": "main-agent",
    "internal": true
  },
  "executionOwner": {
    "type": "group",
    "agentNames": ["code-agent", "review-agent"]
  },
  "participants": [
    {
      "agentName": "code-agent",
      "defaultSelected": true,
      "required": true,
      "userSelectable": false,
      "role": "executor",
      "source": "mention",
      "reason": "负责实现登录接口。"
    },
    {
      "agentName": "review-agent",
      "defaultSelected": true,
      "required": false,
      "userSelectable": true,
      "role": "reviewer",
      "source": "mention",
      "reason": "负责检查实现质量。"
    }
  ],
  "plan": {
    "strategy": "group_sequential",
    "steps": [
      {
        "stepId": "step-code",
        "title": "实现登录接口",
        "task": "实现 Go 登录接口、参数校验和基础错误返回。",
        "assignedAgent": "code-agent",
        "outputType": "code"
      },
      {
        "stepId": "step-review",
        "title": "检查实现",
        "task": "检查接口设计、错误处理和安全风险。",
        "assignedAgent": "review-agent",
        "dependsOn": ["step-code"],
        "outputType": "review"
      }
    ],
    "reasons": [
      "用户明确 @ 了 code-agent 和 review-agent。",
      "先实现再检查的顺序适合该任务。"
    ]
  }
}
```

---

## 6.3 auto / main_agent_orchestration：全量 enabled agents 边界内的 MainAgent 编排

适用场景：

```text
用户不知道该找谁，例如：
- 做一个登录功能，前端 React，后端 Go，并补充测试。
- 帮我把项目跑起来并修问题。
- 我想实现一个 AgentHub 计划确认机制。
```

流程：

```text
用户未指定 Agent，或选择 auto
  ↓
Orchestrator 判定 executionPath = main_agent_orchestration
  ↓
availableBoundary = all enabled agents
  ↓
MainAgent 推荐 participants / candidateParticipants 并生成计划
  ↓
Validator 校验 participants 来自 enabled agents
  ↓
ACTIVITY_SNAPSHOT 展示计划和候选 Agent
  ↓
用户同意 / 反馈修改 / 调整可选参与者 / 取消
  ↓
同意后，Orchestrator 按 selectedParticipants dispatch 子 Agent
  ↓
可选 MainAgent 汇总
```

行为要求：

```text
1. MainAgent 可以推荐 / 选择子 Agent。
2. MainAgent 只生成一个计划，不生成多个 options。
3. 计划中必须展示推荐原因。
4. 不应为了凑多 Agent 而拆分简单任务。
5. 用户同意前不调用任何子 Agent。
6. 用户可以取消 optional participants。
7. requiredParticipants 不能直接取消；如不同意，提交 feedback 让 MainAgent 修订计划。
8. selectedParticipants 不得包含 unknown / disabled agent。
9. 同意后，Orchestrator 按 confirmed selectedParticipants 调度子 Agent。
```

auto 示例：

```json
{
  "executionPath": "main_agent_orchestration",
  "availableBoundary": ["code-agent", "web-agent", "test-agent", "review-agent"],
  "plannerOwner": {
    "type": "main_agent",
    "agentName": "main-agent",
    "internal": true
  },
  "executionOwner": {
    "type": "main_agent_orchestration",
    "agentName": "main-agent",
    "selectedParticipants": ["code-agent", "web-agent"]
  },
  "participants": [
    {
      "agentName": "code-agent",
      "defaultSelected": true,
      "required": true,
      "userSelectable": false,
      "role": "executor",
      "source": "main_agent_recommended",
      "reason": "适合实现 Go 登录接口。"
    },
    {
      "agentName": "web-agent",
      "defaultSelected": true,
      "required": true,
      "userSelectable": false,
      "role": "executor",
      "source": "main_agent_recommended",
      "reason": "适合实现 React 登录页面。"
    },
    {
      "agentName": "test-agent",
      "defaultSelected": false,
      "required": false,
      "userSelectable": true,
      "role": "test",
      "source": "main_agent_recommended",
      "reason": "可选，用于补充登录流程测试。"
    }
  ],
  "defaultSelectedParticipants": ["code-agent", "web-agent"],
  "requiredParticipants": ["code-agent", "web-agent"],
  "plan": {
    "strategy": "sequential_sub_agents",
    "steps": [
      {
        "stepId": "step-backend",
        "title": "实现后端登录接口",
        "task": "实现 Go 登录接口、请求参数校验和基础错误返回。",
        "assignedAgent": "code-agent",
        "outputType": "code"
      },
      {
        "stepId": "step-frontend",
        "title": "实现登录页面",
        "task": "实现 React 登录页面并对接后端接口。",
        "assignedAgent": "web-agent",
        "dependsOn": ["step-backend"],
        "outputType": "code"
      }
    ],
    "reasons": [
      "任务包含后端接口和前端页面。",
      "用户未指定 Agent，因此由 MainAgent 推荐参与者。"
    ]
  }
}
```

---

## 7. 关于“不使用规则路由”的新版边界

新版允许所有路径由 MainAgent 生成计划，但仍然禁止规则路由式越界分派。

关键区分：

```text
MainAgent 生成计划 ≠ MainAgent 在所有模式下自由推荐 Agent。
```

非 auto 模式：

```text
1. 用户已经知道找谁。
2. MainAgent 只能在用户指定边界内写计划。
3. 不允许根据语义自动补 Agent。
4. 不允许用 RulePlanner / LLMPlanner 从全量 Agent 池重新选 Agent。
5. 如果边界内 Agent 能力不足，只能写 warnings。
```

auto 模式：

```text
1. 用户显式或隐式表示“不知道该找谁”。
2. MainAgent 可以从 all enabled agents 中推荐子 Agent。
3. 仍然必须等待用户确认后才执行。
```

示例：

```text
用户选择 code-agent：
  MainAgent 不能自动加 test-agent。
  如果任务提到测试，只能 warning：“如需测试，请选择 auto 或添加 test-agent。”

用户 @web-agent：
  MainAgent 不能自动补 code-agent。
  如果任务需要后端，只能 warning：“当前参与者不包含后端 Agent。”

用户选择 auto：
  MainAgent 可以推荐 code-agent / web-agent / test-agent。
```

---

## 8. 统一计划确认模型

三条路径共享同一个计划生命周期。

### 8.1 标准流程

```text
用户输入任务
  ↓
Gateway 透传请求字段
  ↓
Orchestrator 推导 executionPath
  ↓
Orchestrator 构建 availableBoundary
  ↓
MainAgent 生成一个计划
  ↓
PathAwarePlanValidator 校验计划
  ↓
registerPendingPlan
  ↓
服务端发送 ACTIVITY_SNAPSHOT
  ↓
前端展示计划确认卡片
  ↓
用户选择：
  A. 同意并执行
  B. 修改可选参与者后同意并执行
  C. 不同意，提交反馈修改
  D. 取消
```

### 8.2 用户同意并执行

```text
用户点击“同意并执行”
  ↓
前端发送 approve
  ↓
Gateway 透传 confirm request
  ↓
Orchestrator 校验 runId / planId / revision / selectedParticipants / idempotencyKey
  ↓
run.status = executing
  ↓
按 executionPath 执行：
    single_chat：dispatch 当前 Agent
    group_chat：按 steps dispatch 多个 Agent 独立 turn
    main_agent_orchestration：dispatch confirmed selectedParticipants，可选 MainAgent 汇总
  ↓
前端流式展示结果
```

### 8.3 用户不同意并提交反馈

```text
用户点击“不同意，提出修改意见”
  ↓
前端发送 revise + feedback
  ↓
Orchestrator 校验 runId / planId / revision
  ↓
保存 feedback history
  ↓
run.status = revising_plan
  ↓
MainAgent 基于原计划 + feedback + availableBoundary 生成新版计划
  ↓
revision + 1
  ↓
Validator 再次校验
  ↓
registerPendingPlan
  ↓
再次发送 ACTIVITY_SNAPSHOT
  ↓
继续等待用户确认
```

### 8.4 用户取消

```text
用户点击取消
  ↓
前端发送 cancel
  ↓
Orchestrator 校验 runId / planId / revision
  ↓
run.status = cancelled
  ↓
释放 pending 状态
  ↓
不 emit RUN_ERROR
  ↓
不执行任何 Agent
```

---

## 9. AG-UI 事件与 ActivitySnapshot

### 9.1 主路径

新计划确认主路径使用：

```text
AG-UI event.type = ACTIVITY_SNAPSHOT
```

不再使用：

```text
TOOL_CALL_START confirm_plan
TOOL_CALL_ARGS confirm_plan
TOOL_CALL_END confirm_plan
```

`TOOL_CALL_*` 只能用于真实工具调用，例如：

```text
code_preview
web_preview
shell
file operation
real external tool call
```

### 9.2 ActivitySnapshot payload

```ts
export type PlanApprovalActivitySnapshot = {
  type: "ACTIVITY_SNAPSHOT";

  runId: string;
  messageId: string;
  conversationId?: string;

  activity: {
    activityId: string;      // 建议等于 planId 或可映射到 planId
    kind: "plan_approval";

    phase: "awaiting_confirmation" | "revising_plan" | "executing" | "completed" | "cancelled" | "failed";
    status: "waiting_user_approval" | "revising_plan" | "executing" | "completed" | "cancelled" | "failed";

    planId: string;
    revision: number;

    executionPath: ChatExecutionPath;

    plannerOwner: PlannerOwner;
    executionOwner: ExecutionOwner;

    title: string;
    summary: string;

    plan: ExecutionPlan;

    availableBoundary: string[];
    participants: PlanParticipant[];
    candidateParticipants?: PlanParticipant[];
    defaultSelectedParticipants: string[];
    requiredParticipants: string[];

    selectedParticipants?: string[];

    warnings?: string[];
    allowedActions: Array<"approve" | "revise" | "cancel">;

    createdAt: string;
    expiresAt?: string;
  };
};
```

兼容建议：

```text
如果当前前端仍使用 PlanApprovalCard / MainAgentPlanCard，
可以通过 normalizeActivityToPlanData 将 ACTIVITY_SNAPSHOT.activity 转换为卡片数据。
但新主路径必须是 ACTIVITY_SNAPSHOT，不是 fake confirm_plan TOOL_CALL。
```

---

## 10. Action / Confirm Contract

### 10.1 Approve

```ts
export type ApprovePlanAction = {
  action: "approve";
  runId: string;
  planId: string;
  revision: number;
  selectedParticipants: string[];
  idempotencyKey: string;
};
```

### 10.2 Revise

```ts
export type RequestPlanRevisionAction = {
  action: "revise";
  runId: string;
  planId: string;
  revision: number;
  feedback: string;
  selectedParticipants?: string[];
  idempotencyKey?: string;
};
```

### 10.3 Cancel

```ts
export type CancelPlanAction = {
  action: "cancel";
  runId: string;
  planId: string;
  revision: number;
  idempotencyKey?: string;
};
```

### 10.4 Gateway 职责

Gateway 必须：

```text
1. 透传 action / planId / revision / selectedParticipants / idempotencyKey / feedback。
2. 保留 Orchestrator 返回的错误码。
3. 不把 404 / 409 / 422 全部吞成 502。
4. 不改写 selectedParticipants。
5. 不自行解释 approve / revise / cancel。
```

---

## 11. ExecutionPlan 与 Strategy 收口

### 11.1 Canonical strategy

```ts
export type ExecutionStrategy =
  | "agent_self_execute"
  | "group_sequential"
  | "group_rounds"
  | "main_agent_only"
  | "single_sub_agent"
  | "sequential_sub_agents"
  | "ordered_parallel_sub_agents";
```

### 11.2 路径对应

```text
single_chat:
  agent_self_execute

group_chat:
  group_sequential
  group_rounds

main_agent_orchestration:
  main_agent_only
  single_sub_agent
  sequential_sub_agents
  ordered_parallel_sub_agents
```

### 11.3 旧 strategy 映射

如果现有代码仍使用旧策略：

```text
single           → agent_self_execute 或 single_sub_agent，取决于 executionPath
sequential       → group_sequential 或 sequential_sub_agents，取决于 executionPath
ordered_parallel → ordered_parallel_sub_agents
conversational   → group_rounds 或 main_agent_only，取决于 executionPath
```

原则：

```text
1. Validator / Executor / Frontend 必须认同一套 canonical strategy。
2. strategy 只描述执行方式，不参与 Agent 选择。
3. 不允许用 ordered_parallel 伪装 code-agent → review-agent 的顺序执行。
4. 如果使用 dependsOn，Executor 必须按依赖阻塞执行。
5. 旧 strategy 只允许作为兼容映射，不作为新 contract 主体。
```

---

## 12. PathAwarePlanValidator

### 12.1 通用校验

```text
1. runId 非空。
2. planId 非空。
3. revision >= 1。
4. executionPath 必须合法。
5. plannerOwner 必须是 internal main-agent。
6. executionOwner 必须和 executionPath 匹配。
7. plan.steps >= 1。
8. 每个 step.task 非空。
9. dependencies 不允许引用不存在 step。
10. dependencies 不允许成环。
11. participants 必须全部存在且 enabled。
12. defaultSelectedParticipants 必须来自 participants。
13. requiredParticipants 必须来自 participants。
14. requiredParticipants 必须包含在 defaultSelectedParticipants 内。
15. selectedParticipants 如果存在，必须来自 participants。
16. selectedParticipants 不得包含 unknown / disabled agent。
17. assignedAgent 如果存在，必须在 participants 和 availableBoundary 内。
```

### 12.2 single_chat 特有校验

```text
1. availableBoundary 必须只有一个 Agent。
2. participants 必须只有一个 Agent。
3. participants[0].agentName 必须等于 availableBoundary[0]。
4. requiredParticipants 必须等于 [availableBoundary[0]]。
5. defaultSelectedParticipants 必须等于 [availableBoundary[0]]。
6. steps[].assignedAgent 必须等于 availableBoundary[0]。
7. selectedParticipants 必须只能包含该 Agent。
8. executionOwner.type 必须是 agent。
```

### 12.3 group_chat 特有校验

```text
1. availableBoundary = selectedAgentNames / mentions / activeGroupAgents。
2. availableBoundary 不能为空。
3. selectedAgentNames 与 mentions 同时存在且交集为空时返回 AGENT_SELECTION_CONFLICT。
4. participants 不得越过 availableBoundary。
5. steps[].assignedAgent 不得越过 availableBoundary。
6. 不允许自动补 Agent。
7. 如果计划需要范围外能力，只能放入 warnings。
8. executionOwner.type 必须是 group。
```

### 12.4 main_agent_orchestration 特有校验

```text
1. availableBoundary = all enabled agents。
2. participants 必须来自 enabled agents。
3. requiredParticipants 不能被用户直接取消。
4. selectedParticipants 必须来自 participants。
5. selectedParticipants 必须包含 requiredParticipants。
6. selectedParticipants 不得包含 unknown / disabled agent。
7. executionOwner.type 必须是 main_agent_orchestration。
```

### 12.5 confirm action 校验

```text
1. runId 存在。
2. planId 属于当前 run。
3. revision 匹配当前 active plan。
4. pending state = awaiting_confirmation。
5. selectedParticipants 必须来自 participants。
6. selectedParticipants 必须包含 requiredParticipants。
7. selectedParticipants 不得越过 availableBoundary。
8. idempotencyKey 必填。
9. 相同 idempotencyKey + 相同 action 返回同一结果。
10. 相同 idempotencyKey + 不同 action 返回 409。
11. 已执行、已取消、已过期的 run 不允许再次执行。
```

---

## 13. PendingPlan 状态结构

```go
type PendingPlan struct {
    RunID          string
    PlanID         string
    Revision       int
    MessageID      string
    ConversationID string

    ExecutionPath string

    PlannerOwner  PlannerOwner
    ExecutionOwner ExecutionOwner

    AvailableBoundary []string

    Plan         ExecutionPlan
    Participants []PlanParticipant

    CandidateParticipants        []PlanParticipant
    DefaultSelectedParticipants  []string
    RequiredParticipants         []string
    SelectedParticipants         []string

    Warnings []string

    UserFeedbackHistory []PlanFeedback

    Status    string
    CreatedAt time.Time
    UpdatedAt time.Time
    ExpiresAt time.Time
    ApprovedAt *time.Time

    IdempotencyKeys map[string]IdempotencyRecord
}

type PlanFeedback struct {
    Revision             int
    Feedback             string
    SelectedParticipants []string
    CreatedAt            time.Time
}

type IdempotencyRecord struct {
    Key       string
    Action    string
    Result    string
    CreatedAt time.Time
}
```

存储要求：

```text
1. planId 必须和 runId 绑定。
2. revision 必须单调递增。
3. 同一个 run 只能有一个 active pending plan。
4. 每次修订保留 feedback history。
5. 已执行、已取消、已过期的 plan 不允许再次执行。
6. 三条路径复用 PendingPlan。
7. 第一版可内存态；生产建议持久化。
```

---

## 14. 执行语义

### 14.1 single_chat 执行

```text
1. Orchestrator 读取 PendingPlan。
2. 校验 selectedParticipants = [当前 Agent]。
3. dispatch 当前 Agent。
4. 前端以该 Agent 的消息流展示输出。
5. run.status = completed。
```

特点：

```text
1. MainAgent 只负责计划，不执行任务。
2. 不调用范围外 Agent。
3. 反馈修订仍由 MainAgent 在单 Agent 边界内完成。
```

### 14.2 group_chat 执行

```text
1. Orchestrator 读取 PendingPlan。
2. 校验 selectedParticipants 在 availableBoundary 内。
3. 按 plan.steps / dependsOn 依次触发对应 Agent。
4. 每个 Agent 输出独立 turn。
5. 可选 MainAgent 输出最终摘要。
6. run.status = completed。
```

特点：

```text
1. 各 Agent 的产出在 UI 中独立可见。
2. 不应只展示一个最终汇总。
3. 不允许临时引入未选择 / 未 @ 的 Agent。
```

### 14.3 main_agent_orchestration 执行

```text
1. Orchestrator 读取 PendingPlan。
2. 校验 selectedParticipants。
3. 按 confirmed selectedParticipants 和 plan.steps 调用子 Agent。
4. 子 Agent 输出独立 turn。
5. MainAgent 可汇总最终结果。
6. run.status = completed。
```

特点：

```text
1. 只在 auto / 未指定 Agent 场景下允许推荐 Agent。
2. 用户同意前不执行任何子 Agent。
3. 用户可调整 optional participants。
4. requiredParticipants 不可直接取消，只能通过 feedback 修改计划。
```

---

## 15. 前端实现

### 15.1 主渲染入口

建议保留或新增：

```text
frontend/src/components/ActivitySnapshotRenderer.tsx
```

职责：

```text
1. 接收 AG-UI ACTIVITY_SNAPSHOT。
2. 判断 activity.kind = plan_approval。
3. normalizeActivityToPlanData。
4. 根据 executionPath 渲染对应确认卡。
```

### 15.2 通用 PlanApprovalCard

```text
frontend/src/components/PlanApprovalCard.tsx
```

职责：

```text
1. 展示 executionPath。
2. 展示 plannerOwner：MainAgent。
3. 展示 executionOwner：实际执行 Agent / group / selectedParticipants。
4. 展示计划标题、摘要、steps。
5. 展示 participants。
6. 支持勾选 userSelectable=true 的 optional participants。
7. required=true 的参与者不可直接取消。
8. 支持同意并执行。
9. 支持不同意并提交 feedback。
10. 支持取消。
11. 防重复点击。
12. revision 更新后展示新版计划。
```

### 15.3 UI 文案

single_chat：

```text
MainAgent 已基于你选择的 Agent 生成执行方案。
确认后只会调用该 Agent。
```

group_chat：

```text
MainAgent 已基于当前选定的多个 Agent 生成生产计划。
确认后，各 Agent 将按计划独立产出。
```

auto：

```text
MainAgent 已推荐参与 Agent 并生成执行计划。
确认后才会调用子 Agent。
```

按钮：

```text
同意并执行
按当前选择执行
不同意，提出修改意见
提交修改意见
取消
```

提示：

```text
执行前不会调用任何 Agent。
非 auto 模式下不会自动添加未选择的 Agent。
你可以调整可选参与者，或提交修改意见重新生成计划。
```

---

## 16. 错误处理

### 16.1 Agent 选择冲突

```json
{
  "type": "ERROR",
  "code": "AGENT_SELECTION_CONFLICT",
  "message": "Selected agents and mentioned agents do not overlap. Please adjust your selection or mentions."
}
```

### 16.2 越过 Agent 范围

```json
{
  "type": "ERROR",
  "code": "AGENT_BOUNDARY_VIOLATION",
  "message": "The plan includes agents outside the allowed participant boundary."
}
```

### 16.3 缺少必需参与者

```json
{
  "type": "ERROR",
  "code": "REQUIRED_PARTICIPANT_MISSING",
  "message": "Required participants are missing. Please revise the plan instead of removing them."
}
```

### 16.4 revision 不匹配

```json
{
  "type": "ERROR",
  "code": "PLAN_REVISION_MISMATCH",
  "message": "The plan has been updated. Please review the latest version."
}
```

### 16.5 幂等冲突

```json
{
  "type": "ERROR",
  "code": "IDEMPOTENCY_CONFLICT",
  "message": "The same idempotency key was used for a different action."
}
```

### 16.6 状态错误

```json
{
  "type": "ERROR",
  "code": "INVALID_RUN_STATE",
  "message": "This run is not waiting for plan approval."
}
```

---

## 17. 旧 Planner 处理原则

### 17.1 RulePlanner

```text
1. 不得作为 single_chat / group_chat 的主路径。
2. 不得在非 auto 模式中按关键词从全量 Agent 池选 Agent。
3. 如保留，只能作为测试 helper、fallback 示例或 MainAgent 内部非权威参考。
4. 任何 RulePlanner 输出都必须经过 PathAwarePlanValidator。
```

### 17.2 LLMPlanner

```text
1. 不得作为 single_chat / group_chat 的主路径。
2. 不得在非 auto 模式中决定 Agent 范围。
3. 如保留，只能作为 MainAgent 内部能力、plan text normalizer 或 legacy fallback。
4. 不允许 LLMPlanner 内部 validator 拦截 group_chat 的合法 strategy。
5. 统一校验必须交给 PathAwarePlanValidator。
```

### 17.3 禁止补丁

```text
1. 只在 LLMPlanner 内部 validator 加 sequential allowlist，然后宣称修好了。
2. 让 group_chat 继续走 RulePlanner，只是加 AllowedAgents 过滤。
3. 让 group_chat 继续走 LLMPlanner，只是在 prompt 里要求不要越界。
4. 用 ordered_parallel 伪装 code-agent → review-agent 的顺序执行。
5. 让 MainAgent 继续只靠关键词匹配，却宣称是主 Agent。
6. 让 Gateway 解释 action 或选择 Agent。
7. 继续用 fake confirm_plan TOOL_CALL 作为新计划主路径。
8. 在非 auto 模式下根据用户语义自动补 Agent。
9. 把 executionPath 全部抹平成 auto。
```

---

## 18. 迁移计划

### Phase A：冻结旧 Planner 在非 auto 路径中的使用

目标：

```text
single_chat / group_chat 不再进入 RulePlanner / LLMPlanner 的 Agent 选择路径。
```

验收：

```text
1. single_chat 不调用 RulePlanner.Plan / LLMPlanner.Plan。
2. group_chat 不调用 RulePlanner.matchedAgents。
3. group_chat 不从全量 Agent 池补 Agent。
```

### Phase B：新增 MainAgent 统一计划入口

目标：

```text
所有 executionPath 都调用 MainAgent.GeneratePlan。
```

验收：

```text
1. MainAgent 接收 executionPath + availableBoundary。
2. MainAgent 生成 participants / steps / reasons / warnings。
3. MainAgent 不直接执行 Agent。
```

### Phase C：新增 PathAwarePlanValidator

目标：

```text
用统一 Validator 替代分散 validator。
```

验收：

```text
1. single_chat 越界被拒绝。
2. group_chat 越界被拒绝。
3. auto unknown / disabled agent 被拒绝。
4. dependencies 不合法被拒绝。
```

### Phase D：统一 ActivitySnapshot / PendingPlan / Confirm

目标：

```text
计划确认链路统一。
```

验收：

```text
1. registerPending 在 emit ACTIVITY_SNAPSHOT 之前。
2. confirm 透传 selectedParticipants / idempotencyKey。
3. Gateway 不吞错误码。
4. approve / revise / cancel 状态正确。
```

### Phase E：统一 strategy

目标：

```text
Planner / Validator / Executor / Frontend 使用同一套 canonical strategy。
```

验收：

```text
1. group_sequential 能表达 code-agent → review-agent。
2. Executor 按 dependsOn 或 steps 顺序执行。
3. Frontend 正确展示顺序计划。
```

### Phase F：MainAgent 能力升级

目标：

```text
MainAgent 从关键词 matcher 升级为真正内部规划组件。
```

验收：

```text
1. 读取 enabled agents / capabilities / descriptions。
2. auto 能推荐 code-agent / web-agent / test-agent / review-agent 等所有 enabled agents。
3. participants 包含 reason。
4. 简单任务不强行拆多 Agent。
```

### Phase G：测试矩阵

目标：

```text
覆盖 single_chat / group_chat / auto / HITL / AG-UI / 幂等。
```

验收见第 19 节。

---

## 19. 测试矩阵

### 19.1 single_chat

```text
1. 用户选择 code-agent → executionPath = single_chat。
2. MainAgent 生成计划，但 participants 只有 code-agent。
3. MainAgent 不推荐 test-agent / review-agent。
4. feedback revise 后 revision + 1。
5. approve 后只 dispatch code-agent。
6. selectedParticipants 包含其他 Agent 时返回 AGENT_BOUNDARY_VIOLATION。
```

### 19.2 group_chat

```text
1. selectedAgentNames = [code-agent, review-agent] → executionPath = group_chat。
2. @code-agent @review-agent → executionPath = group_chat。
3. selectedAgentNames=[code-agent] + @web-agent → AGENT_SELECTION_CONFLICT。
4. MainAgent 只能在 [code-agent, review-agent] 内生成计划。
5. 如果计划包含 security-agent → AGENT_BOUNDARY_VIOLATION。
6. code-agent → review-agent 按顺序独立 turn 输出。
7. group_chat 不调用 RulePlanner.matchedAgents。
```

### 19.3 auto / main_agent_orchestration

```text
1. agentName=auto → executionPath = main_agent_orchestration。
2. 未指定 Agent → executionPath = main_agent_orchestration。
3. MainAgent 可以从 all enabled agents 推荐 participants。
4. participants 必须包含 reason。
5. requiredParticipants 不能直接取消。
6. optional test-agent 被取消后不 dispatch test-agent。
7. warning 中说明测试未执行。
8. MainAgent 可最终汇总。
```

### 19.4 AG-UI / HITL

```text
1. 新计划通过 ACTIVITY_SNAPSHOT 发送。
2. 不发送 fake confirm_plan TOOL_CALL。
3. registerPending 发生在 emit ACTIVITY_SNAPSHOT 前。
4. approve 校验 runId / planId / revision / selectedParticipants / idempotencyKey。
5. revise 生成 revision + 1。
6. cancel 不 emit RUN_ERROR。
7. duplicate approve 不重复 dispatch Agent。
8. Gateway 保留 Orchestrator 错误码。
```

---

## 20. 最终验收标准

交互验收：

```text
1. single_chat：用户选择一个 Agent，MainAgent 只基于该 Agent 生成计划。
2. single_chat：同意后只调用该 Agent。
3. group_chat：用户选择多个 Agent 或 @ 多个 Agent，MainAgent 只基于这组 Agent 生成生产计划。
4. group_chat：同意后多个 Agent 依次产出，且各自输出可见。
5. auto：用户未指定 Agent 或选择 auto，MainAgent 可以推荐 Agent。
6. 三条路径都只生成一个计划。
7. 三条路径都必须先确认后执行。
8. 用户可以同意、反馈修改、取消。
9. feedback 生成 revision + 1。
10. requiredParticipants 不能被直接取消，只能通过 feedback 修改计划。
```

Contract 验收：

```text
1. ACTIVITY_SNAPSHOT 是计划确认主路径。
2. activity 包含 runId / planId / revision。
3. activity 包含 executionPath。
4. activity 包含 plannerOwner / executionOwner。
5. activity 包含 participants / selectedParticipants / requiredParticipants。
6. approve 包含 selectedParticipants / idempotencyKey。
7. revise 包含 feedback。
8. planId 必须和 runId 绑定。
9. revision 必须匹配。
10. 已执行 / 已取消 / 已过期的 plan 不能再次执行。
```

路径验收：

```text
single_chat:
1. MainAgent 生成计划。
2. availableBoundary 只有当前 Agent。
3. 不调用范围外 Agent。

group_chat:
1. availableBoundary 来自 selectedAgentNames / mentions / activeGroupAgents。
2. 不从全量 Agent 池自动补充 Agent。
3. 多 Agent 输出独立 turn。

auto / main_agent_orchestration:
1. availableBoundary = all enabled agents。
2. MainAgent 推荐 participants。
3. MainAgent 记录 feedback history。
4. MainAgent 可生成 revision + 1。
5. Orchestrator 按用户确认后的 selectedParticipants 执行。
6. MainAgent 可汇总子 Agent 输出。
```

---

# 给 Claude Code 的总提示词

```text
请将 AgentHub 的多路径计划确认架构调整为“统一 MainAgent 计划生成 + executionPath 边界控制”方案。

核心原则：

1. 保留三条 executionPath：
   - single_chat
   - group_chat
   - main_agent_orchestration

2. 所有路径都由 Orchestrator 内部 MainAgent 生成计划：
   - single_chat：MainAgent 只能在当前选中 Agent 边界内生成计划。
   - group_chat：MainAgent 只能在 selectedAgentNames / mentions / activeGroupAgents 边界内生成计划。
   - auto：MainAgent 可以从 all enabled agents 中推荐 participants。

3. 这不是把所有模式都变成 auto：
   - executionPath 仍然表示用户意图。
   - executionPath 决定 availableBoundary。
   - 非 auto 模式不得自动补 Agent。

4. Validator 必须强制边界：
   - single_chat participants 只能有当前 Agent。
   - group_chat participants / steps.assignedAgent 只能来自 availableBoundary。
   - auto participants 必须来自 enabled agents。
   - selectedParticipants 必须包含 requiredParticipants。
   - requiredParticipants 不能被直接取消。
   - idempotencyKey 必填。

5. Orchestrator 是唯一权威层：
   - deriveExecutionPath
   - buildAvailableBoundary
   - call MainAgent.GeneratePlan
   - PathAwarePlanValidator
   - registerPending before emit ACTIVITY_SNAPSHOT
   - approve / revise / cancel
   - dispatch selectedParticipants
   - duplicate approve 幂等控制

6. Gateway 只能转发：
   - 不生成计划
   - 不解释 action
   - 不调用 Agent
   - 不做 participant 选择
   - 不吞 Orchestrator 错误码
   - 不丢 selectedParticipants

7. AG-UI：
   - 新计划确认主路径必须是 ACTIVITY_SNAPSHOT。
   - TOOL_CALL_* 只能用于真实工具调用。
   - 不再使用 fake confirm_plan TOOL_CALL 承载计划确认。

8. 旧 Planner：
   - RulePlanner / LLMPlanner 不得作为 single_chat / group_chat 的 Agent 选择主路径。
   - 如保留，只能作为 MainAgent 内部参考、normalizer 或 legacy fallback。
   - 不允许只给 LLMPlanner validator 加 sequential allowlist 就宣称完成。

请按以下阶段执行：

Phase A：冻结旧 Planner 在非 auto 路径中的使用。
Phase B：新增 MainAgent 统一计划入口。
Phase C：新增 PathAwarePlanValidator。
Phase D：统一 ActivitySnapshot / PendingPlan / Confirm。
Phase E：统一 ExecutionPlan strategy。
Phase F：升级 MainAgent 能力。
Phase G：补齐测试矩阵。

每个 Phase 完成后输出：
- 修改文件
- 核心逻辑
- 请求 / 响应示例
- ACTIVITY_SNAPSHOT 样本
- 测试结果
- 遗留问题
```
