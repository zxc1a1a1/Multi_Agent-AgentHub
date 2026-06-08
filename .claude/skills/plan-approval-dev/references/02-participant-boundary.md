# 02 Participant 边界规则

## 核心定义

```text
AvailableBoundary     该 executionPath 可用的最大 Agent 集合
AllowedAgents         Planner/Executor 必须遵守的具体 Agent 列表
participants          计划中出现的所有 Agent
candidateParticipants  (auto only) main-agent 推荐的候选 Agent。≠ 最终 selectedParticipants
defaultSelectedParticipants (auto only) main-agent 推荐的默认选择
selectedParticipants   用户最终确认的执行者
requiredParticipants   required=true 的参与者。用户不可取消。
```

## 各 executionPath 边界

| 规则 | single_chat | group_chat | main_agent_orchestration |
|------|-----------|-----------|------------------------|
| AvailableBoundary | [当前Agent] | selectedAgentNames ∩ mentions | all enabled agents |
| AllowedAgents | [当前Agent] | 同上 | all enabled agents |
| candidateParticipants | N/A | N/A | main-agent 推荐的候选 |
| defaultSelectedParticipants | N/A | N/A | main-agent 推荐的默认选择 |
| main-agent 可列候选? | 否 | 否 | 是（仅作为 candidateParticipants） |
| 用户可取消 Agent? | 否（required） | 否（均为 required） | 可取消 optional；不可取消 required |

## group_chat 交集规则

```text
- mentions 为空: AllowedAgents = selectedAgentNames
- mentions 非空: AllowedAgents = selectedAgentNames ∩ mentions
- 交集为空: ERROR AGENT_SELECTION_CONFLICT
  - 不生成 PLAN_PROPOSAL
  - 不执行任何 Agent
  - 返回错误详情含 selectedAgentNames、mentions、intersection
```

## main_agent_orchestration 参与者规则

```text
- 用户不修改 participants + 无 feedback:
  selectedParticipants = defaultSelectedParticipants → APPROVE_PLAN 按当前计划执行。

- 用户修改 participants 或 提交 feedback:
  main-agent 重新生成 revision+1 的 PLAN_PROPOSAL。
  新的 candidateParticipants 和 defaultSelectedParticipants。
  selectedParticipants 被丢弃。
  新的 PLAN_PROPOSAL 进入 waiting_user_approval。

- 用户移除 required participant → 尝试 approve:
  400 REQUIRED_PARTICIPANT_MISSING。
  plan 留在 waiting_user_approval。
```

## Main-agent identity

```text
- planOwner.type = "main_agent"
- planOwner.agentName = "main-agent"  
- 第一版为 Orchestrator 内部组件，不注册为 A2A Agent
- 不暴露为 code-agent
- 不进入 dispatchable agent registry
```

## 错误码

```text
AGENT_SELECTION_CONFLICT       selectedAgentNames ∩ mentions == []
AGENT_BOUNDARY_VIOLATION       task.agentName 不在 AllowedAgents 内
REQUIRED_PARTICIPANT_MISSING   用户取消了 required participant
PLAN_REVISION_MISMATCH         revision 不匹配
INVALID_RUN_STATE              plan 状态不允许当前 action
```


## Approve vs revise for auto participant changes

```text
action=approve:
  只能接受当前 proposal。selectedParticipants 省略时使用 defaultSelectedParticipants；
  如果传入的 selectedParticipants 与 defaultSelectedParticipants 不一致，返回 PARTICIPANT_CHANGE_REQUIRES_REVISION。

action=revise:
  用于用户修改 Agent 选择或提交 feedback。main-agent 基于修改重新生成 revision+1 PLAN_PROPOSAL。
```
