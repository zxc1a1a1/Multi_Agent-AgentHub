# Registry / Health Boundary

Agent Registry 是 Agent 能力目录。Health Check 是 Agent 可用性来源。

## 规则

- Orchestrator 读取 Registry 与 Health 来选择 Agent。
- Gateway 可以读取脱敏 Agent 摘要用于展示。
- Frontend 不直接调用 Child Agent 的 AgentCard 或 health endpoint。
- Child Agent 必须能声明能力与健康状态。
- 不健康 Agent 不得被主计划选择。

## 实现中立

Registry 可以由配置、数据库、服务发现、文件或专用服务实现。本 Skill 不固定具体方式。
