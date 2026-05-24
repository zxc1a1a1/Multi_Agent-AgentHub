# TypeScript React Style

来源：`docs/contracts/code-style.md`、`docs/contracts/conventions.md`。

- TypeScript `strict: true`。
- 组件 PascalCase，hooks `useXxx`。
- 避免长期 `any` 与 `@ts-ignore`。
- 前端 API response 类型来自 OpenAPI 生成。
- SSE 解析与 API 调用放在基础层，不散落到页面组件。
