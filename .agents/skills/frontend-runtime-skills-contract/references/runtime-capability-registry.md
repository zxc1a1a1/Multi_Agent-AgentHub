# Runtime Capability Registry

## 目的

Registry 是前端 Runtime Capability 的唯一入口。

前端不得在组件中通过 `agentName` 或临时分支决定渲染能力。

## 注册项字段

每个 capability 必须声明：

- `toolName`
- `description`
- `status`
- `component`
- `behavior`
- `riskLevel`
- `requiresConfirmation`
- `acceptsStreamingArgs`
- `parseAt`
- `parametersSchema`
- `resultSchema`
- `failureMode`

## 状态

| status | 含义 |
|---|---|
| `implemented` | 当前代码可执行 |
| `seeded` | 推荐注册，但实现状态以代码为准 |
| `reserved` | 已占名，不执行 |
| `disabled` | 存在但禁用 |
| `deprecated` | 兼容旧能力，不推荐新增使用 |

## 硬性规则

- `toolName` 必须唯一。
- `toolName` 必须使用 `snake_case`。
- `status !== implemented` 时不得自动执行。
- Registry 不得包含 `agentName → component` 映射。
- Registry 不得使用 `allowedInMvp` 这类历史阶段字段作为执行判断。
