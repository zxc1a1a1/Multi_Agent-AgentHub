# OpenAPI Style Guide

## 版本

使用：

```yaml
openapi: 3.1.0
```

## operationId

每个 operation 必须有稳定 `operationId`。

推荐格式：`listConversations`、`createConversation`、`getConversation`、`listConversationMessages`、`listAgents`、`getAgent`、`createRun`、`getRun`、`cancelRun`、`getArtifact`。

## Schema

公共对象放在 `components.schemas`。成功响应使用统一 envelope。错误响应使用统一 error envelope。JSON 字段使用 `camelCase`。时间使用 ISO 8601 字符串。

## Lifecycle

公开 API 可以标记：

```yaml
x-agenthub-lifecycle: implemented | planned | deprecated
```

## 禁止

- 把 `/internal/**` 放进公开 OpenAPI。
- 把 Orchestrator endpoint 放进公开 OpenAPI。
- 把 Child Agent endpoint 放进公开 OpenAPI。
- 在 schema 中出现 API key、service token、内部 URL、system prompt。
