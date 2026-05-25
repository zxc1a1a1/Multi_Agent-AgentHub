# 服务间鉴权规则

Gateway 调 Orchestrator 必须有服务间鉴权。

## 最小方案

```text
Authorization: Bearer <internal-service-token>
```

## 环境变量

Gateway：

```text
ORCHESTRATOR_INTERNAL_TOKEN
```

Orchestrator：

```text
INTERNAL_SERVICE_TOKEN
```

## 规则

- Orchestrator 必须校验 token。
- Frontend 不得持有 service token。
- 用户 token 不得当作 service token。
- service token 不得进入日志。
- service token 不得进入错误响应。
- service token 不得写进 Dockerfile。
- Orchestrator 不得公网裸露。

## 长期演进

可升级为：

- mTLS。
- service identity。
- short-lived internal token。
