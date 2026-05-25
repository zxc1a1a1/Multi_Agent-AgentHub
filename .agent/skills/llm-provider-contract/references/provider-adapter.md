# Provider Adapter

Provider Adapter 负责把不同 Provider 的请求、响应、错误和流式事件归一化。

## 适配边界

Adapter 负责：

- 构造 Provider 原始请求。
- 调用 Provider API。
- 归一化响应。
- 归一化流式事件。
- 归一化错误。
- 填充 usage / latency / providerName / modelId。

Adapter 不负责：

- 选择 Agent。
- 执行业务工具。
- 修改数据库。
- 生成前端事件。
- 直接返回 Provider 原始对象。

## 硬规则

业务层只能消费 AgentHub 的统一对象，不得依赖 Provider 原始字段。
