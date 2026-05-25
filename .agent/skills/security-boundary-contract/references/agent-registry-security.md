# Agent Registry Security

## 目的

保护 Agent 发现、能力声明和健康检查不成为攻击入口。

## 规则

- AgentCard 是能力声明，不是信任凭证。
- AgentCard 不得包含 API key、service token、数据库连接串、system prompt、内部路径。
- Registry 只接受可信来源、白名单、签名配置或受控管理入口中的 Agent。
- Orchestrator 只能调用 enabled 且 healthy 的 Agent。
- Health Check 只暴露健康状态，不暴露环境变量、堆栈、内部网络拓扑。
- Frontend 只能看到脱敏 Agent 摘要。

## 通用字段

- agentId
- displayName
- capabilities
- inputModes
- outputModes
- trustLevel
- healthStatus
- version
