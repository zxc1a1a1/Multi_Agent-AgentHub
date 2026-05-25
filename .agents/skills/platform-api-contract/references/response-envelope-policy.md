# 统一响应 Envelope

成功响应：

```json
{"code": 0, "data": {}, "message": "success"}
```

错误响应：

```json
{"code": 400001, "data": null, "message": "错误描述"}
```

成功时 `code = 0`。错误时 `code != 0`。错误时 `data = null`，除非 schema 明确允许错误详情。HTTP status 不能全部使用 200。
