# 脱敏规则

## 1. 目的

本文定义日志、错误、debug dump、metrics 中的敏感信息脱敏规则。

## 2. 禁止出现内容

以下内容不得出现在日志、用户错误、debug dump、metrics label 中：

```text
API key
access token
refresh token
system prompt
完整 LLM raw request
完整 LLM raw response
完整用户隐私输入
数据库连接串
对象存储签名 URL
内部绝对路径
workspace 绝对路径
private network URL
cookie
session secret
```

## 3. 允许记录方式

可以记录：

```text
present / missing
length
hash
redacted preview
errorCode
safeMessage
traceId
```

## 4. stack trace

stack trace 只能进入内部 debug 日志，并且必须受环境和权限控制。

用户响应不得包含 stack trace。

## 5. 禁止事项

不得：

- 在日志中记录真实 secret。
- 在 debug dump 中包含完整 prompt。
- 在错误信息中返回 provider raw error。
- 在前端展示内部路径。
- 在 metrics label 中写敏感字段。
