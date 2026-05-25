# 串行 Skill 生成规则

## 1. 使用场景

当用户按顺序要求：

- “先改这个 Skill”；
- “一个一个分别改”；
- “先思考这个 Skill”；
- “输出这个 Skill 的文件包”；

AI 必须进入串行 Skill 生成模式。

## 2. 核心规则

- 每轮只处理一个 Skill。
- 只处理用户明确指定的 Skill。
- 不主动生成其他 Skill。
- 不把其他 Skill 的细节复制进当前 Skill。
- 不要求读者先读其他 Skill。
- 只写当前 Skill 必须自包含的内容。
- 当前 Skill 的 references 只补充当前 Skill。
- 当前 Skill 的 docs/contracts 只定义当前 Skill 对应契约。

## 3. 跨 Skill 边界写法

允许写：

```text
本 Skill 不定义数据库表结构。
本 Skill 不定义前端组件实现。
本 Skill 不定义子 Agent 内部协议。
```

不允许写：

```text
必须先阅读 xxx-skill，否则无法理解本文。
详见另一个 Skill 的全部规则。
```

## 4. 后续建议

如果发现其他 Skill 后续也要同步，应在交接中写：

```text
后续建议：下一步可重构 xxx-skill。
```

但不得自动生成或修改它。

## 5. 中文要求

用户要求中文版时：

- `SKILL.md` 正文必须中文；
- references 必须中文；
- docs/contracts 必须中文；
- PATCH_NOTES 必须中文；
- MANIFEST 必须中文；
- 代码标识符、路径、协议字段可以保留英文。
