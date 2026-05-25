# 命名规范

## 通用原则

命名应表达意图，不表达偶然实现。

好的命名：

- 稳定。
- 可搜索。
- 可读。
- 不依赖上下文猜测。

避免：

- `data`、`info`、`obj`、`tmp` 长期存在。
- 无意义缩写。
- 同一概念多个名字。
- 一个名字表示多个概念。

## 目录命名

- 文档目录使用 kebab-case。
- Skill 目录使用 kebab-case。
- Contract 文档使用 kebab-case。
- Go package 目录使用短小写名。
- 前端组件目录使用符合项目现有风格的命名。

## Go 命名

- package：小写短名。
- 导出类型 / 函数：PascalCase。
- 非导出类型 / 函数 / 变量：camelCase。
- 测试函数：`TestXxx`。
- benchmark：`BenchmarkXxx`。

## TypeScript / React 命名

- 组件：PascalCase。
- Hook：useXxx。
- props：XxxProps。
- store：xxxStore。
- service 函数：动词开头，例如 `listAgents`、`createConversation`。

## JSON 字段

- 对外 JSON 字段使用 camelCase。
- 内部数据库字段可以使用 snake_case。
- 不得把数据库字段名未经转换直接暴露到前端 API。
- 布尔字段使用 `is`、`has`、`can`、`should` 等前缀。

## 文档命名

- Markdown 使用 kebab-case.md。
- JSON Schema 使用 kebab-case.schema.json。
- Review checklist 使用 `*-review-checklist.md`。
- 版本包命名包含 skill 名、版本方向和语言，例如 `code-style-and-conventions-v1-independent-cn.zip`。
