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

## 三层边界

`toolName` 是前端运行时能力名，由 `preview.previewType` 决定，不由 `outputMode` 直接决定：

```text
outputMode → artifact.type → previewType → toolName
```

- `outputMode`：Agent 声明的语义类别（上游，不直接驱动前端）。
- `artifact.type`：平台归一化后的产物类型（中游）。
- `previewType`：预览意图，应优先等于 toolName。
- `toolName`：前端 Registry 查询键（下游）。

三者不得直接等同。所有转换必须通过映射表表达。

## 硬性规则

- `toolName` 必须唯一。
- `toolName` 必须使用 `snake_case`。
- `previewType` 应优先等于 Runtime `toolName`。
- `artifact.type` 不是 `toolName`。
- `outputMode` 不是 `toolName`。
- `toolName` 由 `preview.previewType` 决定，不由 `outputMode` 直接决定。
- 不得通过 `agentName` 推断 outputMode 或选择 toolName。
- 不得使用 `download` 作为 toolName（已废弃，统一使用 `file_download`）。
- `status !== implemented` 时不得自动执行。
- Registry 不得包含 `agentName → component` 映射。
- Registry 不得使用 `allowedInMvp` 这类历史阶段字段作为执行判断。
