# 数据安全契约

## 不得入库

数据库不得保存：

- 明文 API key。
- 明文用户 token。
- 明文服务间 token。
- 数据库连接串。
- 完整敏感 system prompt。
- 未脱敏 LLM 原始请求 / 响应。
- 私有文件绝对路径。
- 永久公开下载 URL。
- 内网服务拓扑。
- 未脱敏用户隐私。

## 错误字段

`error_message` 应面向展示或排查做脱敏。

推荐：

- 用户可见错误：简短、安全、可理解。
- 内部日志错误：包含 trace_id，但不包含 secret。
- 数据库存储错误：只保存 code、摘要、trace。

## Token 与密码

- 密码不得明文保存。
- token 不得明文保存。
- token 可保存 hash、摘要、过期时间、撤销状态。
- secret 应来自环境变量或 secret manager。

## content_ref

- 不得是永久公开 URL。
- 不得暴露内部绝对路径。
- 不得携带长期签名 token。
- 访问必须可校验、可审计。
