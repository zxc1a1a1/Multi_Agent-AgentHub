# JSON 字段规则

## 允许使用 JSON 的场景

```text
agents.agent_card
agents.skills
agents.input_modes
agents.output_modes
runs.plan_json
tool_calls.args
artifacts.metadata
messages.content_json
```

## 必须满足

- 有结构说明。
- 有兼容策略。
- 有脱敏规则。
- 有大小控制。
- 有测试样例。

## 不适合只放 JSON 的字段

- 经常 where 的字段。
- 经常 order by 的字段。
- 经常 join 的字段。
- 权限控制字段。
- 状态字段。
- 外键关联字段。

## 禁止

- 把 JSON 当成垃圾桶。
- 把 secret 放 JSON。
- 把大文件放 JSON。
- 没有 schema 的任意结构。
- 让前端依赖未文档化 JSON 字段。
