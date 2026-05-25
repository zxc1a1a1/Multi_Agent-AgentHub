# Platform API Error Codes

## 规则

错误码必须稳定，不依赖具体 Agent 名称。`message` 必须脱敏。HTTP status 必须合理。

## 错误域

| 错误域 | 含义 |
|---|---|
| AUTH_* | 认证或授权失败 |
| VALIDATION_* | 请求参数或语义校验失败 |
| CONVERSATION_* | 会话资源错误 |
| MESSAGE_* | 消息资源错误 |
| AGENT_* | Agent 摘要或可见性错误 |
| RUN_* | Run 创建、查询、取消错误 |
| ARTIFACT_* | Artifact 查询、预览、下载错误 |
| GATEWAY_* | Gateway 外部入口或内部下游错误 |
| INTERNAL_* | 未分类内部错误 |

## 示例

| HTTP | code | symbolicCode | message |
|---:|---:|---|---|
| 401 | 401001 | AUTH_REQUIRED | 请先登录 |
| 403 | 403001 | AUTH_FORBIDDEN | 无权访问该资源 |
| 400 | 400001 | VALIDATION_BAD_REQUEST | 请求参数无效 |
| 404 | 404001 | CONVERSATION_NOT_FOUND | 会话不存在或不可见 |
| 409 | 409301 | RUN_ALREADY_FINISHED | 运行已结束，无法取消 |
| 502 | 502001 | GATEWAY_DOWNSTREAM_UNAVAILABLE | 下游服务暂不可用 |
| 504 | 504001 | GATEWAY_DOWNSTREAM_TIMEOUT | 下游服务响应超时 |
| 500 | 500001 | INTERNAL_UNEXPECTED | 系统异常，请稍后重试 |
