# Redaction Policy

## 目的

定义日志、trace、metrics、debug dump 中的脱敏规则。

## 禁止记录

- API key
- access token / refresh token
- Authorization header
- service-to-service token
- 数据库连接串
- 对象存储签名 URL
- 完整 system prompt
- 完整 LLM raw request / response
- 完整用户隐私输入
- 本地绝对路径
- 内网拓扑
- cookie
- session id

## 允许记录

- hash 后的 userId
- 截断后的摘要
- errorCode
- provider error category
- prompt template id
- model id
- token usage 数字

## 规则

- 新增日志字段前必须判断是否敏感。
- debug dump 必须脱敏。
- 用户可见错误不得包含内部堆栈、token、密钥或连接串。
