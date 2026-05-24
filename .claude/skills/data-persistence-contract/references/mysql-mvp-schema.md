# MySQL MVP Schema

来源：`docs/contracts/mysql-schema.md`、`docs/contracts/data-model.md`。

## 1. MVP 事实源

- MVP v0.1 当前使用 MySQL 8。
- 核心对象：conversation、message、run、a2a_task、tool_call、artifact。

## 2. 约束

- API JSON 字段使用 camelCase。
- DB 列名可用 snake_case。
- JSON 字段不得存密钥。
