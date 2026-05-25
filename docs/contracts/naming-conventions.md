# AgentHub 命名规范

## 1. 目的

统一项目中的文件、目录、类型、变量、字段和提交 scope 命名。

## 2. 目录

- 文档目录：kebab-case。
- Skill 目录：kebab-case。
- Go package：小写短名。
- 前端组件：PascalCase。

## 3. Go

- package：小写。
- 导出类型和函数：PascalCase。
- 非导出变量和函数：camelCase。
- 文件名：lower_snake_case.go。
- 测试文件：`*_test.go`。

## 4. TypeScript / React

- 组件：PascalCase。
- Hook：useXxx。
- Props：XxxProps。
- Store：xxxStore。
- Request / Response：XxxRequest / XxxResponse。

## 5. JSON

- 对外字段：camelCase。
- 数据库字段：可使用 snake_case。
- 不得把数据库字段名未经转换直接暴露给前端。

## 6. 文档

- Contract 文档：kebab-case.md。
- JSON Schema：kebab-case.schema.json。
- Review 清单：*-review-checklist.md。

## 7. 禁止

- 无意义缩写。
- 一个概念多个名字。
- `utils`、`common`、`helper` 滥用。
- `data`、`obj`、`tmp` 长期存在。
