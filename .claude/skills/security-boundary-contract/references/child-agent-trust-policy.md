# Child Agent Trust Policy

## 原则

Child Agent 是能力提供者，不是全局可信主体。

## 规则

- Child Agent 只允许执行自身声明并获授权的能力。
- Child Agent 输出全部视为不可信内容。
- Child Agent 不得直接访问 Frontend。
- Child Agent 不得直接访问 Gateway 用户 API。
- Child Agent 不得决定全局编排策略。
- 用户自建 Agent 默认较低信任等级。
- 用户自建 Agent 不得默认拥有高危能力。

## 高危能力示例

- run_command
- deploy
- file_overwrite
- secret_read
- external_publish
