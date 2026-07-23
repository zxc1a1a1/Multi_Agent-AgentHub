---
name: llm-orchestration-review
description: 用于审核 AgentHub 2.0 LLM 编排改动，检查 Registry Catalog、PlanVersion、Validator、用户确认门、Replan、上下文隔离、测试证据和路径边界。
---

# AgentHub 2.0 LLM 编排审核 Skill

使用场景：审核 `/llm-orchestration-dev` 产生的改动。

## 必读 Skill

```text
/project-architecture
/planning-approval-contract
/agent-registry-contract
/context-management-contract
/testing-review-contract
```

## 审核输入

```text
git diff --stat
git diff --name-only
git diff
Phase Reports
测试输出
最终报告
```

## 立即拒绝条件

出现以下任一情况，默认 `REJECTED`：

```text
LLM 输出未经 Validator 直接执行
未确认 Plan 创建 AgentInvocation
Manual Mode 使用用户未选择 Agent
Planner 使用未注册 Agent
Planner 使用 disabled/unhealthy/unauthorized Agent
Planner 使用不存在的 skillId
执行缺少 PlanVersion 或 AgentSnapshot
确认后静默更换 Agent
Material Replan 未重新确认
Repair 可无限循环
Normalizer 猜测未知 Agent
测试依赖真实 API key
跨 Conversation 混入历史或 Artifact
RulePlanner 绕过 Validator/confirmation
未授权修改 Frontend/Gateway/Docker/legacy paths
```

## 审核重点

```text
Direct Mode 是否不生成多 Agent Plan
Planner Catalog 是否来自 Registry eligibility
Parser/Normalizer/Validator/Repair 是否分离
Validator 是否确定性
Repair 是否最多一次
PlanVersion 是否不可变
确认是否绑定准确版本
Executor 是否只执行已确认版本
依赖图是否校验无环
Agent 调用前是否重新检查可用性
Replan 是否生成新版本
ContextSnapshot 是否可追溯
上游输出是否结构化且有界
Rule fallback 是否仍经过 PlanVersion/Validator/confirmation
```

## 必要证据

- Manual Mode 越权 Agent 被拒绝；
- Auto Mode unhealthy Agent 被过滤；
- invalid JSON repair；
- unknown Agent/Skill rejection；
- unconfirmed execution rejection；
- stale version rejection；
- Agent unavailable after confirmation；
- material Replan confirmation；
- cross-conversation contamination negative；
- mock-only deterministic test path。

## 结论

输出一个：

```text
APPROVED
APPROVED WITH FOLLOW-UP
CHANGES REQUESTED
REJECTED
```

并包含：

```text
Summary
Blocking issues
Non-blocking issues
Evidence checked
Required fixes
Contract mismatches
Next phase suggestion
```
