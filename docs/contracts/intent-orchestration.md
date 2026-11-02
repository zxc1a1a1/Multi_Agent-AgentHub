# Intent Orchestration Contract

**Status:** Active (v2)
**Owner:** AgentHub
**Primary source:** Module Separation & Runtime Redesign；v2 优化见 `docs/architecture/optimization-architecture.md`
**Last updated:** 2026-10-29（P0：取消全部语义规则回退，冻结 MainAgent 动作 schema 与失败语义）

## Authoritative source order

1. `docs/architecture/optimization-architecture.md`（v2 决策）
2. `docs/contracts/chat-execution-path.md`
3. `docs/contracts/participant-boundary.md`
4. `docs/contracts/planner-input.md`
5. `docs/contracts/orchestration-plan.md`
6. 旧 `docs/superpowers/specs/*` 仅作历史设计参考；与 v2 冲突时以 v2 为准
7. PDR/Sprint/UML 只作产品参考

旧 `server/` 与根目录 `agents/` 是 legacy 参考实现，不是目标路径。

## Scope

规划与执行只发生在 `services/orchestrator`。Planner 只产出结构化计划/动作，不直接读 Gateway 数据库、不调用 Child Agent、不执行工具。

## Decision entry（唯一语义入口）

```text
用户目标 + 有界历史 + 允许的 Agent/Tool 描述 + 已完成 observation
  → MainAgent（LLM）
  → 结构化动作
  → Parser → Normalizer → Validator（确定性）→ Coordinator 执行
```

- 生产启动图**只有一个 LLM 决策入口**：`MainAgent`。
- **禁止**关键词评分、`RulePlanner`、模型失败时固定选择 `code-agent`、`llm_with_rule_fallback`、缺模型自动退回规则等任何语义回退。
- 保留的确定性逻辑（鉴权、schema 校验、允许集合、预算、依赖调度、审批、终态）不是语义规则路由。
- 离线测试只允许注入 FakeModel，FakeModel 不得因缺少配置在生产启用。

## MainAgent 动作 schema（冻结）

```json
{
  "action": "answer | clarify | tool_call | delegate | revise_plan | finish",
  "reason": "short model rationale (not chain-of-thought)",
  "answer": {"content": "..."},
  "clarify": {"question": "..."},
  "toolCall": {"toolName": "...", "arguments": {}},
  "delegate": {"agentName": "...", "task": "...", "dependsOn": []},
  "plan": {"strategy": "single|sequential|ordered_parallel", "tasks": []}
}
```

规则：

1. `action` 必填且只能取上列枚举；未知动作拒绝。
2. `toolName` / `agentName` 只能来自本次请求的允许集合；越权或未知名称拒绝，不猜测、不填默认值。
3. `delegate.agentName` 不得超出 `AllowedAgents`；`single_chat` / `group_chat` 下不得新增或替换 Agent。
4. `plan.tasks[].agentName` 同样受允许集合约束；依赖必须引用已知 taskId 且无环。
5. 模型输出必须通过 Parser → Normalizer → Validator；修复最多一次，且修复仍由模型执行，不换关键词规则。
6. `reason` 只保存简短决策摘要，不保存模型隐藏思维链。

## 失败语义（冻结）

| 情况 | 处理 |
|---|---|
| 模型依赖缺失/不支持必需能力 | 启动或就绪检查失败；不退回 rule，不把 mock 接到生产 |
| 模型暂时错误 | 预算内有限重试；仍失败返回脱敏的终止性错误 |
| JSON/schema 错误 | 最多一次模型修复；仍错误则失败；不猜 Agent 名、不补默认任务 |
| 意图不明确 | 返回 `clarify` 动作等待用户，不用关键词补路由 |
| 工具不存在/越权 | 拒绝调用，记录受控 observation；模型可在预算内调整 |
| Agent 不可用 | 模型重新规划或明确失败；不偷偷替换成默认 Agent |
| 模型返回空/非法 Agent 集合 | 终止当前 run 并返回安全错误，派发次数必须为 0 |

## ChatExecutionPath 与 Planner 约束

| executionPath | Planner 角色 | AllowedAgents |
|---|---|---|
| `single_chat` | 路由到指定 Agent；不替代 Agent 自身规划 | `[currentAgent]` |
| `group_chat` | 协调预选 Agent 的顺序/依赖；不新增 Agent | `selectedAgentNames ∩ mentions` |
| `main_agent_orchestration` | 从完整 Agent 池推荐/选择 | 全部启用 Agent |

- 非自动路径中，Planner **MUST NOT** 选择、添加或替换 `AllowedAgents` 之外的 Agent。
- Validator **MUST reject** 任何 `task.agentName` 不在 `AllowedAgents` 的计划。
- 用户显式 `@agent` 是权限/参与边界，不是关键词路由。

## Planning mode

| Mode | Meaning |
|---|---|
| `single` | One Child Agent handles the task. |
| `sequential` | Ordered dependent steps. |
| `parallel` | Independent steps may run concurrently. |
| `ordered_parallel` | Independent intent, deterministic ordered output. |

`planningMode`（`auto`/`direct`/`manual`/`mention`）已被 `executionPath` 取代。

## Participant boundary

- `single_chat`: `AllowedAgents = [currentAgent]`.
- `group_chat`: `AllowedAgents = selectedAgentNames ∩ mentions`；交集为空 → `AGENT_SELECTION_CONFLICT`。
- `main_agent_orchestration`: `AvailableBoundary = all enabled agents`。MainAgent 提出 `candidateParticipants` / `defaultSelectedParticipants`，它们是提案；最终 `selectedParticipants` 由原样批准产生。

## Auto participant-change rule

`main_agent_orchestration` 中，用户修改候选集合或提供反馈时动作为 `REQUEST_PLAN_REVISION`，不是批准。带修改参与者的 `APPROVE_PLAN` 必须以 `PARTICIPANT_CHANGE_REQUIRES_REVISION` 拒绝。

## 证据要求

- 门禁以真实 handler 入口的派发调用计数为准：模型失败/无模型时 Child Agent 派发次数为 0。
- 代码中残留 `RulePlanner` 字样必须能解释为历史文档或迁移遗留；启动图与调用断言优先于字符串计数。
- FakeModel 测试通过不得表述为真实模型效果通过。

## 路由决策契约（v3 A0-03 冻结）

每次 run 必须能回答：走了哪条路径、考虑过哪些 Agent、为什么选中、为什么拒绝。
状态等级：`已实现`（A4-01 落库；A4-05 追加模型选发言者决策）。

### 表 `route_decisions`

| 字段 | 类型 | 说明 |
|---|---|---|
| `run_id` | `VARCHAR(64)` | 主键组成部分 |
| `decision_seq` | `INT` | 同一 run 内单调递增，从 0 开始 |
| `execution_path` | `VARCHAR(32)` | `single_chat` / `group_chat` / `main_agent_orchestration` |
| `strategy` | `VARCHAR(32)` | `single` / `sequential` / `ordered_parallel` / `conversational` |
| `candidates_json` | `JSON` | 候选 Agent 名称数组（确定性排序） |
| `selected_json` | `JSON` | 实际派发 Agent 名称数组 |
| `rejected_json` | `JSON` | `[{agentName, reasonCode}]`；reasonCode 稳定枚举 |
| `planner_source` | `VARCHAR(32)` | `main_agent`（生产唯一入口） |
| `reason_summary` | `VARCHAR(512)` | 模型给出的简短理由，不含思维链 |
| `created_at` | `DATETIME(3)` | 服务端 UTC |

唯一键：`PRIMARY KEY (run_id, decision_seq)`。保留期：与 `execution_events` 同级；MVP 不做自动裁剪。

### 生产者 / 消费者

- 生产者：`services/orchestrator` 的路径派生与 `MainAgent` planner 结果落库（A4-01）。
- 消费者：run 排查接口、路由指标（A4-07）、评测 harness。
- 禁止：把候选/拒绝原因直接透传浏览器；对外只暴露脱敏摘要。

### 验收

- 每次 run 至少一条决策；`selected_json` 必须等于实际派发的 Agent 集合。
- 拒绝原因使用稳定 code，不写自由文本堆栈。
- 无 `route_decisions` 时指标必须显示“缺失”，不能默认为 100% 命中。

## 群聊发言者选择（v3 A4-05）

- 默认模式 `GROUP_CHAT_SPEAKER_MODE=round_robin`，行为与 A4-04 一致。
- `GROUP_CHAT_SPEAKER_MODE=model` 时，`MainAgent` 兼任 `SpeakerSelector`，逐位选择发言者：
  输出 `{"speaker":"<name>","reason":"...","done":false}` 或 `{"done":true,"reason":"..."}`。
- 候选边界 = 当前计划中已勾选/必需的参与者；模型输出越界、缺少理由、连续重复上一发言者、
  或首轮即终止，都直接使 run 失败（`ORCHESTRATOR_SPEAKER_SELECTION_FAILED`），不静默改选。
- 终止条件：模型 `done=true`，或达到
  `GROUP_CHAT_MAX_ROUNDS × 参与者数` 的发言上限。
- 每次选择写入该 run 的 `route_decisions`（与初始计划同一行），`reason_summary` 追加
  `speaker selection: turn N <agent>: <reason>; stop: <reason>`，总长截断到 512；
  `selected_json` 仍是实际派发的 Agent 集合，不新增 decision_seq，避免路由指标把一次
  群聊误计为 re-plan 循环。
- 选择发生在计划展开阶段，依据用户消息、参与者角色与既有选择理由；尚未实现执行中依据
  Agent 输出动态重选（后续任务）。
