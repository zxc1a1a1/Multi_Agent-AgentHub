# Frontend Runtime Skills Contract

## 目的

本文定义 AgentHub 前端 Runtime Capability 的通用契约。

前端 Runtime Capability 不绑定具体 Agent 名称，只绑定：

```text
toolName + parametersSchema + component + behavior + riskLevel + failureMode
```

## 通用性原则

禁止通过 `agentName` 决定组件或能力。

正确流程：

1. 收到完整 Tool Call。
2. 读取 `toolName`。
3. 查询 Runtime Capability Registry。
4. 校验状态是否允许执行。
5. 解析并校验 args。
6. 执行安全策略。
7. 渲染组件或降级。

## 当前种子能力

| toolName | 能力 | 默认状态 | 绑定 Agent |
|---|---|---|---:|
| `code_preview` | 代码预览 | `implemented` | 否 |
| `web_preview` | HTML/Web 预览 | `implemented` 或 `seeded` | 否 |
| `markdown_render` | Markdown 渲染 | `implemented` 或 `seeded` | 否 |
| `image_preview` | 图片预览 | `reserved` | 否 |
| `file_download` | 文件下载 | `reserved` | 否 |
| `document_preview` | 文档预览 | `reserved` | 否 |
| `data_preview` | 数据预览 | `reserved` | 否 |

## 行为分类

| behavior | 说明 |
|---|---|
| `render` | 只展示，不产生副作用 |
| `interactive` | 需要用户动作，不直接改变外部世界 |
| `side_effect` | 可能修改外部状态，默认禁用 |

## previewType 与 toolName

`previewType` 应优先等于 Runtime `toolName`。`artifact.type` 不是 `toolName`，`outputMode` 不是 `toolName`。不得使用 `download` 作为 toolName（已废弃，统一使用 `file_download`）。`document_preview` / `data_preview` 当前为 planned/reserved，不要求实现。

### 三层映射链

`toolName` 是前端运行时能力名，由 `preview.previewType` 决定，不由 `outputMode` 直接决定：

```text
outputMode → artifact.type → previewType → toolName
```

- `outputMode`：Agent 声明的语义类别（上游，不直接驱动前端）。
- `artifact.type`：平台归一化后的产物类型（中游）。
- `previewType`：预览意图，应优先等于 toolName。
- `toolName`：前端 Registry 查询键（下游）。

三者不得直接等同。所有转换必须通过映射表表达。

规则：
- 不得通过 `outputMode` 直接选择 toolName。
- 不得通过 `agentName` 选择 toolName。
- 不得把 `outputMode` 直接当作 `artifact.type`（除非映射表显式声明兼容）。

## 安全原则

- Runtime Capability 输入一律不可信。
- 参数通过 schema 校验后才能进入组件。
- 未知 toolName 不执行。
- 高风险 HTML 必须隔离渲染。
- Markdown 默认禁用 raw HTML。
- code_preview 只展示不执行。
