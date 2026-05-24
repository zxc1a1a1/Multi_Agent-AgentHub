# Error Response Policy

来源：`docs/contracts/openapi.yaml`、`docs/contracts/security-boundaries.md`、`docs/contracts/api-error-codes.md`。

## 1. 统一错误响应

```json
{
  "code": 400001,
  "data": null,
  "message": "错误描述"
}
```

## 2. 安全脱敏

错误消息不得包含：

- stack trace
- API key / token
- 数据库连接串
- 内部服务地址

## 3. 语义

- HTTP 状态码表达传输层语义。
- `code` 表达业务层语义。
- 禁止“所有错误都返回 200”。
