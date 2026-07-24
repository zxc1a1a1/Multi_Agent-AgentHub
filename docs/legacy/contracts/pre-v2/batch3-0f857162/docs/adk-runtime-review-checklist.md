# ADK Runtime Review Checklist

## Agent 接入

- [ ] 新增 Child Agent 是否复用 `agents/adk/`？
- [ ] 是否没有复制 Runtime 基础设施？
- [ ] 是否没有把 agentName 写进 Runtime 通用逻辑？

## 配置

- [ ] 是否有合法 `config.yaml`？
- [ ] 是否能生成或校验 AgentCard？
- [ ] skills / outputModes 是否真实？
- [ ] permissions 是否最小化？

## Handler

- [ ] 是否接收结构化消息？
- [ ] 是否尊重 context cancellation？
- [ ] 是否通过 Context API 输出？
- [ ] 是否不直接生成 AG-UI？

## LLM

- [ ] LLMClient 是否启动时创建？
- [ ] HTTP Client 是否设置 timeout？
- [ ] 是否通过依赖注入进入 Handler？
- [ ] 错误是否脱敏？

## Artifact

- [ ] Artifact 是否符合 artifact-contract？
- [ ] Artifact 类型是否由 outputModes 决定？
- [ ] 是否没有硬编码 agentName？

## 安全

- [ ] AgentCard 无 secret。
- [ ] /health 无 secret。
- [ ] 日志无 secret。
- [ ] 工具默认关闭。

## 测试

- [ ] config load test。
- [ ] AgentCard test。
- [ ] Runtime API mapping test。
- [ ] Handler error test。
- [ ] secret redaction test。
