# Code Style Contract

## 1. 文档目的

本文档定义 AgentHub 全项目代码风格兜底规则，用于约束 TypeScript、React、Go、JSON、OpenAPI、JSON Schema、Contract 文档、目录命名和提交说明。

本文件不替代外部 React / Go 最佳实践 Skill，而是 AgentHub 的统一风格基线。

## 2. TypeScript / React

- TypeScript 必须启用 `strict: true`。
- 禁止无理由使用 `any`。
- 禁止长期保留 `// @ts-ignore`。
- React 组件使用 PascalCase。
- React hooks 使用 `useXxx`。
- API response 类型必须来自 `docs/contracts/openapi.yaml` 的生成类型。
- AG-UI Event 类型必须来自 AG-UI Event Contract。
- Frontend Runtime Skill 参数类型必须来自 `frontend-runtime-skills.schema.json`。
- 组件内不得直接散落后端 endpoint 字符串。
- 前端不得引用 A2A / Gateway-Orchestrator internal 类型作为 REST API response 类型。

## 3. Go

- Go 代码必须通过 `gofmt`。
- Go 包名使用小写。
- 文件名使用小写加下划线。
- 导出类型使用 PascalCase。
- 非导出类型使用 camelCase。
- 必须显式处理 error。
- 必须向下传递 `context.Context`。
- 不得随意 `panic`。
- 不得把复杂业务逻辑写进 handler。
- 正式代码必须通过 `go test`。
- 正式代码应通过 `golangci-lint`。

## 4. JSON 命名

所有对外 JSON 使用 camelCase：

```json
{
  "runId": "run-001",
  "threadId": "conv-001",
  "messageId": "msg-001",
  "toolCallId": "tc-001",
  "pageSize": 20
}
```

数据库列名可使用 snake_case，但不得直接暴露为 API JSON 字段。

## 5. OpenAPI / JSON Schema

- OpenAPI 文件固定为 `docs/contracts/openapi.yaml`。
- OpenAPI `operationId` 必须稳定。
- Component schema 使用 PascalCase。
- JSON 字段使用 camelCase。
- JSON Schema 必须明确 `required`。
- enum 必须显式声明。
- 不得让 Markdown Contract 和 JSON Schema 脱节。

## 6. Contract 文档

Contract 文档统一放在：

```text
docs/contracts/
```

文件使用 kebab-case：

```text
agui-events.md
a2a-task.md
artifact-schema.md
security-boundaries.md
```

每份 Contract 建议包含：

```text
文档目的
协议边界
MVP v0.1 范围
Post-MVP 扩展方向
字段 / 事件 / endpoint 定义
禁止事项
Review Checklist
```

## 7. MVP v0.1 特别规则

MVP 可以简化功能，但不能放松风格：

- 仍然必须保持模块边界。
- 仍然必须遵守 Contract。
- 仍然必须使用 camelCase JSON。
- 仍然必须处理 error。
- 仍然必须传递 context。
- 仍然必须避免 token / API key 泄漏。
- 仍然必须保持可测试。

## 8. 禁止事项

- 禁止前端手写 API response 类型。
- 禁止 Go handler 堆叠复杂编排逻辑。
- 禁止 Gateway handler 直接调用 A2A endpoint。
- 禁止把大 Artifact 塞进文本流。
- 禁止跳过 lint / typecheck / go test 后直接宣称完成。
- 禁止生成未格式化代码。
