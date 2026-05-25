# Secret Management Policy

## Secret 类型

- LLM API key
- service-to-service token
- JWT signing secret
- database password
- object storage credential
- deploy token
- OAuth secret
- webhook secret

## 规则

- Secret 只能来自环境变量、secret manager 或等价机制。
- `.env.example` 只能写占位符。
- Secret 不得硬编码进代码、文档示例、OpenAPI 示例、AgentCard、Artifact metadata。
- 日志、trace、debug dump 不得包含 Secret。
- Secret 需要最小权限、轮换能力和访问审计。
- AI 生成代码必须经过 Secret 泄漏检查。
