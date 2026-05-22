# Secret Management Policy

## 1. 文档目的

本文档定义 AgentHub 中密钥和敏感配置的管理规则。

## 2. 密钥范围

密钥包括：

- LLM API key。
- JWT signing secret。
- Service token。
- Object storage credentials。
- Database password。
- OAuth secret。
- Deploy token。

## 3. 基本规则

- 密钥只能来自环境变量或安全密钥管理。
- 不得写进 Git。
- 不得写进 AgentCard。
- 不得写进 OpenAPI 示例。
- 不得写进 Contract 示例。
- 不得打印到日志。
- 不得回传给 Frontend。
- `.env.example` 只能写变量名，不写真实值。
- AI 生成代码不得硬编码密钥。

## 4. 日志脱敏

日志不得包含：

```text
Authorization
API key
service token
database password
full system prompt
raw uploaded secret file
```

日志应包含可排查 ID：

```text
requestId
traceId
runId
conversationId
messageId
agentName
a2aTaskId
errorCode
```
