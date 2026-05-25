# LLM Provider Security

## Secret 规则

- API key 只能来自环境变量、secret manager 或等价机制。
- 配置文件只能保存 env var 名称。
- 不得把真实 API key 写进仓库。
- 不得把用户 token 当 Provider API key。

## 日志规则

不得记录：

- API key
- Authorization header
- 完整 system prompt
- 数据库连接串
- Provider 原始敏感响应

## Gateway / Orchestrator 分进程规则

- Gateway Service 不直接调用 LLM Provider。
- Gateway Service 不保存 Provider secret。
- LLM 调用必须发生在被授权的后端运行单元中，并通过 Provider Adapter。
