# CI Quality Gates

## 1. 目的

本文定义 AgentHub CI 质量门禁。

## 2. MVP 最小门禁

MVP 阶段建议至少运行：

```text
go test ./...
frontend typecheck
frontend lint
unit tests
protocol conversion tests
schema validation tests
docker compose smoke optional
secret scan
```

## 3. 正式开发门禁

正式开发阶段增加：

```text
contract tests
integration tests
Playwright E2E smoke
security regression tests
dependency vulnerability scan
coverage report
OpenAPI validation
JSON Schema validation
```

## 4. 失败策略

以下失败不得合并：

- schema validation failed。
- unit test failed。
- protocol conversion test failed。
- secret scan failed。
- typecheck failed。
- lint failed。
- security critical failed。
- E2E smoke failed for release branch。

## 5. 禁止事项

不得：

- 跳过失败测试仍允许合并。
- CI 使用真实生产 API key。
- CI 调用真实 LLM 作为必需步骤。
- CI 依赖生产数据库。
- 忽略 contract test 失败。
