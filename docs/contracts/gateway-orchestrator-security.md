# Gateway-Orchestrator 安全契约

## 服务间鉴权

Gateway 调 Orchestrator 必须携带服务间凭证。

推荐：

```text
Authorization: Bearer <internal-service-token>
```

## 禁止

- Frontend 持有 service token。
- 用户 token 作为 service token。
- service token 进入日志。
- service token 进入错误响应。
- Orchestrator 公网裸露。
- Gateway 直接访问 Child Agent 绕过 Orchestrator。

## SafeError

```json
{
  "code": "ORCHESTRATOR_TIMEOUT",
  "message": "任务执行超时，请稍后重试",
  "retryable": true
}
```

SafeError 不得包含：

- API key。
- Authorization token。
- 数据库连接串。
- 内部堆栈。
- 本地路径。
- 内网拓扑。
- 完整 system prompt。
- 未脱敏 LLM 原始请求或响应。
