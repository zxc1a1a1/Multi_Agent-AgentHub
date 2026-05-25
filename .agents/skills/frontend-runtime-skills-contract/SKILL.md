---
name: frontend-runtime-skills-contract
description: "用于定义 AgentHub 前端 Runtime Capability 的注册、Tool Call 参数聚合、schema 校验、组件绑定、执行行为、安全降级、iframe/Markdown/Code 渲染和 ToolResult 规则。该 Skill 不绑定具体 Agent 名称。"
---

# frontend-runtime-skills-contract

## 1. Skill 目的

本 Skill 定义 AgentHub 前端 Runtime Capability 的通用契约。

它约束前端在收到完整 Tool Call 后，如何：

- 识别 `toolName`。
- 查询 Runtime Capability Registry。
- 聚合 Tool Call args。
- 解析 JSON 参数。
- 校验参数 schema。
- 绑定 React 组件或前端执行器。
- 执行渲染、交互或安全降级。
- 处理未知能力、非法参数和组件异常。
- 管理 ToolResult 的返回边界。

一句话：

**前端 Runtime Capability 只绑定 `toolName + schema + component + behavior + riskLevel + failureMode`，不绑定任何具体 Agent 名称。**

---

## 2. 独立性原则

本 Skill 是独立 Skill。

阅读本文件不需要先阅读其他 Skill。

本 Skill 只定义前端 Runtime Capability 的注册与执行规则。

本 Skill 不展开：

- 实时事件协议字段。
- 后端如何生成 Tool Call。
- 子 Agent 如何生成产物。
- Artifact 持久化结构。
- 数据库表结构。
- Docker 交付方案。
- 后端 Runtime 设计。
- Agent 编排算法。

如果前端最终收到的是 Tool Call，本 Skill 只关心：

```text
工具名是什么？
参数是否完整？
参数是否合法？
前端是否注册了这个能力？
这个能力是否允许执行？
执行失败时如何降级？
```

---

## 3. 通用性原则

Frontend Runtime Capability 不绑定具体 Agent 名称。

前端不得通过 `agentName` 判断应该执行哪个 Runtime Capability。

前端只根据以下信息处理 Tool Call：

1. `toolName` 是否已注册。
2. 注册项 `status` 是否允许执行。
3. Tool Call args 是否完整。
4. args 是否通过 schema 校验。
5. Runtime Capability 的安全策略是否允许执行。
6. 对应组件是否存在且可安全渲染。

禁止：

```ts
if (agentName === 'web-agent') {
  renderWebPreview(args)
}
```

正确：

```ts
const capability = registry[toolName]
if (capability?.status === 'implemented' && validate(capability.parametersSchema, args)) {
  render(capability.component, args)
}
```

示例：

- 任意 Agent 都可以触发 `code_preview`，只要 args 符合 `CodePreviewParams`。
- 任意 Agent 都可以触发 `web_preview`，只要 args 符合 `WebPreviewParams`。
- 任意 Agent 都可以触发 `markdown_render`，只要 args 符合 `MarkdownRenderParams`。

禁止：

- 通过 Agent 名称推断参数结构。
- 通过 Agent 名称绕过 schema validation。
- 把某个 Runtime Capability 写成某个 Agent 专属能力。
- 在前端核心 runtime 中维护 `agentName → component` 映射。

---

## 4. 当前阶段识别

MVP v0.1 已完成，仅作为历史回归基线。

MVP 历史基线包括：

- `code_preview`。
- 小型代码预览。
- 不执行代码。
- 不访问网络。
- 不修改服务端状态。

这些历史基线不得限制后续 Runtime Capability 扩展。

当前通用目标：

- 支持可扩展 Runtime Capability Registry。
- 支持多个前端 runtime capability。
- 支持 Tool Call args 分片聚合。
- 支持 `TOOL_CALL_END` 后统一解析。
- 支持 schema validation。
- 支持未知能力安全降级。
- 支持非法参数安全降级。
- 支持渲染型、交互型和副作用型能力分层。
- 支持高风险渲染能力的隔离与错误恢复。

---

## 5. 本 Skill 负责什么

本 Skill 负责：

- Runtime Capability Registry。
- Tool Call 参数聚合。
- 参数 schema 校验。
- toolName 与组件绑定。
- 渲染型 capability 的安全边界。
- 交互型 capability 的用户动作边界。
- 副作用型 capability 的默认禁用规则。
- ToolResult 基础结构。
- 失败降级策略。
- 前端 runtime 安全规则。
- Review Checklist。

---

## 6. 本 Skill 不负责什么

本 Skill 不负责：

- 服务端如何选择 Agent。
- 服务端如何生成 Tool Call。
- SSE 事件 wire format。
- Artifact 后端事实源结构。
- 子 Agent 内部运行时。
- 数据库如何保存 Tool Call。
- 具体 React 样式。
- 具体 UI 设计稿。
- 生产部署方案。

---

## 7. Runtime Capability 定义

Runtime Capability 是前端可执行的能力单元。

它可以是：

- 展示一段代码。
- 预览一段 HTML。
- 渲染 Markdown。
- 显示图表。
- 展示图片。
- 请求用户确认。
- 收集表单输入。
- 触发下载。

Runtime Capability 不是：

- Agent 名称。
- Artifact 类型。
- React 组件名称。
- 后端事件类型。
- 数据库表名。

Runtime Capability 的唯一外部入口是 `toolName`。

---

## 8. Runtime Capability Registry

前端必须维护 Runtime Capability Registry。

推荐类型：

```ts
type RuntimeSkillStatus =
  | 'implemented'
  | 'seeded'
  | 'reserved'
  | 'disabled'
  | 'deprecated'

type RuntimeSkillBehavior =
  | 'render'
  | 'interactive'
  | 'side_effect'

type RuntimeSkillRiskLevel =
  | 'low'
  | 'medium'
  | 'high'

type RuntimeSkillFailureMode =
  | 'hide'
  | 'placeholder'
  | 'error_card'
  | 'text_fallback'

type FrontendRuntimeSkill = {
  toolName: string
  description: string
  status: RuntimeSkillStatus
  component?: string
  behavior: RuntimeSkillBehavior
  riskLevel: RuntimeSkillRiskLevel
  requiresConfirmation: boolean
  acceptsStreamingArgs: boolean
  parseAt: 'tool_call_end'
  parametersSchema: unknown
  resultSchema?: unknown
  failureMode: RuntimeSkillFailureMode
}
```

规则：

- `toolName` 必须唯一。
- `toolName` 使用 `snake_case`。
- `component` 不是 `toolName`。
- `artifact.type` 不是 `toolName`。
- `status = implemented` 才允许自动执行。
- `status = seeded` 表示推荐注册但不代表当前代码已经完整实现。
- `status = reserved` 表示已占名但不可执行。
- `status = disabled` 表示存在但明确禁用。
- `status = deprecated` 表示兼容旧能力，不推荐新增使用。
- `parseAt` 当前固定为 `tool_call_end`。
- 参数必须通过 schema 校验后才能传给组件。

---

## 9. 当前种子 Runtime Capabilities

以下能力是当前推荐注册的前端通用能力，不代表系统只能支持这些能力。

| toolName | 能力类型 | 是否绑定 Agent | 默认状态 |
|---|---|---:|---|
| `code_preview` | 代码预览 | 否 | `implemented` |
| `web_preview` | HTML/Web 预览 | 否 | `implemented` 或 `seeded` |
| `markdown_render` | Markdown 渲染 | 否 | `implemented` 或 `seeded` |

规则：

- 这些能力不属于任何具体 Agent。
- 任意 Agent 都可以触发这些能力，只要 Tool Call 合法。
- 新 Agent 不得要求前端增加 `if agentName === ...`。
- 新能力必须通过 registry 扩展，而不是修改 Agent 分支判断。

---

## 10. Reserved / Disabled Capabilities

未来可能注册的能力：

| toolName | 推荐状态 | 说明 |
|---|---|---|
| `image_preview` | `reserved` | 图片预览 |
| `file_download` | `reserved` | 文件下载 |
| `diff_preview` | `reserved` | Diff 预览 |
| `terminal_output` | `reserved` | 终端输出展示 |
| `chart_render` | `reserved` | 图表展示 |
| `form_input` | `disabled` | 用户表单输入 |
| `confirm_action` | `disabled` | 用户确认 |
| `file_upload` | `disabled` | 文件上传 |
| `deploy_status` | `disabled` | 部署状态或部署动作 |

规则：

- `reserved` 能力不得执行。
- `disabled` 能力不得执行。
- 交互型能力必须等待用户明确动作。
- 副作用型能力默认禁用，必须有确认门和权限控制。

---

## 11. Tool Call 消费状态机

前端必须按 `toolCallId` 管理 Tool Call 生命周期。

标准状态机：

```text
TOOL_CALL_START
  → 创建 pending buffer

TOOL_CALL_ARGS
  → 按 toolCallId 追加 delta/content

TOOL_CALL_END
  → 拼接完整 args
  → JSON.parse
  → schema validation
  → registry lookup
  → safety check
  → component render / fallback
```

规则：

- 不得在 `TOOL_CALL_ARGS` 阶段执行 capability。
- `TOOL_CALL_ARGS` 可以有多个分片。
- args 聚合必须按 `toolCallId`。
- 同一 message 可以有多个 `toolCallId`。
- 未知 `toolCallId` 的 ARGS / END 必须安全忽略或记录错误。
- END 后重复到达不得重复执行。
- JSON parse 失败必须 fallback。
- schema validation 失败必须 fallback。
- 未知 `toolName` 不得执行。

兼容规则：

- 新事件优先读取 `delta`。
- 兼容旧事件中的 `content`。
- 聚合时使用 `event.delta ?? event.content ?? ''`。

---

## 12. 参数聚合与 Schema Validation

参数处理必须分四步：

1. 按 `toolCallId` 聚合字符串。
2. 在 `TOOL_CALL_END` 后解析 JSON。
3. 使用 registry 中的 `parametersSchema` 校验。
4. 校验通过后才传给组件或执行器。

规则：

- 组件不得接收未校验参数。
- 不得信任后端一定返回合法 JSON。
- 字符串字段必须设置合理长度上限。
- URL 字段必须限制协议。
- HTML / Markdown / JS 字段必须按 capability 安全策略处理。
- 参数校验失败时不得导致页面白屏。

---

## 13. Component Binding

Runtime Capability 与组件绑定通过 registry 完成。

示例：

```ts
const registry = {
  code_preview: {
    toolName: 'code_preview',
    status: 'implemented',
    component: 'CodePreview',
    behavior: 'render',
    riskLevel: 'low',
    requiresConfirmation: false,
    acceptsStreamingArgs: true,
    parseAt: 'tool_call_end',
    parametersSchema: CodePreviewSchema,
    failureMode: 'error_card',
  },
}
```

规则：

- 组件名称不得作为外部协议字段。
- 外部只传 `toolName`。
- Registry 负责把 `toolName` 映射到组件。
- 缺失组件必须 fallback。
- 组件抛错必须被 Error Boundary 或等价机制隔离。

---

## 14. Execution Behavior

Runtime Capability 按行为分三类。

### 14.1 render

只展示，不产生服务端副作用。

示例：

- `code_preview`
- `web_preview`
- `markdown_render`
- `image_preview`
- `chart_render`

规则：

- 可以在参数合法后自动执行。
- 必须安全渲染。
- 失败时必须降级。

### 14.2 interactive

需要用户交互，但不直接改变外部世界。

示例：

- `form_input`
- `confirm_action`

规则：

- 必须等待用户动作。
- 必须明确取消路径。
- 必须有 ToolResult 或等价结果结构。

### 14.3 side_effect

可能修改服务端、文件、部署、外部系统或用户数据。

示例：

- `file_upload`
- `deploy_action`
- `run_command`

规则：

- 默认 `disabled`。
- 必须有用户确认。
- 必须有权限检查。
- 必须有清晰审计记录。

---

## 15. code_preview Capability Policy

`code_preview` 是通用代码展示能力，不绑定任何 Agent。

参数：

```ts
type CodePreviewParams = {
  code: string
  language: string
  filename?: string
  title?: string
}
```

建议注册项：

```text
behavior = render
riskLevel = low
requiresConfirmation = false
failureMode = error_card
```

规则：

- 只展示代码。
- 不执行代码。
- 不插入 `<script>`。
- 不访问网络。
- 不修改服务端状态。
- 大代码应折叠或截断展示。
- `language` 不合法时使用纯文本高亮降级。

---

## 16. web_preview Capability Policy

`web_preview` 是通用 HTML/Web 预览能力，不绑定任何 Agent。

参数：

```ts
type WebPreviewParams = {
  html: string
  css?: string
  js?: string
  title?: string
  filename?: string
}
```

建议注册项：

```text
behavior = render
riskLevel = medium
requiresConfirmation = false
failureMode = placeholder 或 error_card
```

规则：

- 不得把 HTML 直接注入主应用 DOM。
- 必须使用 iframe / sandbox / 等价隔离策略。
- 默认不得启用 `allow-same-origin`。
- 不得默认启用 `allow-forms`。
- 不得默认启用 `allow-popups`。
- 不得默认启用 `allow-downloads`。
- 不得默认启用 `allow-top-navigation`。
- 如果启用 scripts，不得同时启用 same-origin，除非有单独安全评审和隔离 origin。
- iframe 出错时显示错误卡片或占位，不得导致聊天页白屏。
- `html` 为空时不得渲染空 iframe，应降级为错误卡片。

推荐默认 sandbox：

```html
<iframe sandbox="allow-scripts" />
```

禁止默认 sandbox：

```html
<iframe sandbox="allow-scripts allow-same-origin" />
```

---

## 17. markdown_render Capability Policy

`markdown_render` 是通用 Markdown 渲染能力，不绑定任何 Agent。

参数：

```ts
type MarkdownRenderParams = {
  markdown: string
  title?: string
}
```

建议注册项：

```text
behavior = render
riskLevel = low
requiresConfirmation = false
failureMode = text_fallback
```

规则：

- 支持 GFM。
- 默认禁用 raw HTML。
- 链接协议必须限制。
- 代码块只展示，不执行。
- 渲染失败回退纯文本。
- 不得把 Markdown 中的 HTML 当可信 DOM 插入主应用。

---

## 18. ToolResult Policy

v1.0 默认 render capability 不需要向后端返回 ToolResult。

以下能力通常不返回 ToolResult：

- `code_preview`
- `web_preview`
- `markdown_render`

允许返回 ToolResult 的情况：

- 用户确认类交互。
- 表单输入。
- 文件选择。
- 手动反馈。
- 前端执行结果需要回传。

推荐结构：

```ts
type ToolResult = {
  toolCallId: string
  status: 'success' | 'cancelled' | 'failed'
  data?: unknown
  error?: {
    code: string
    message: string
  }
}
```

规则：

- ToolResult 不得包含 secret。
- ToolResult 不得回传未脱敏异常堆栈。
- ToolResult 必须关联 `toolCallId`。
- 交互型和副作用型 capability 必须定义取消路径。

---

## 19. Failure Mode Policy

Runtime Capability 必须声明失败策略。

可选失败策略：

| failureMode | 说明 |
|---|---|
| `hide` | 静默隐藏，仅记录开发日志 |
| `placeholder` | 显示占位卡片 |
| `error_card` | 显示错误卡片 |
| `text_fallback` | 回退为纯文本展示 |

推荐：

| capability | 参数非法 | 渲染异常 | 未支持 |
|---|---|---|---|
| `code_preview` | `error_card` | `error_card` | `text_fallback` |
| `web_preview` | `error_card` | `placeholder` | `text_fallback` |
| `markdown_render` | `text_fallback` | `text_fallback` | `text_fallback` |

规则：

- 单个 capability 崩溃不得导致整个聊天页面白屏。
- 高风险渲染组件必须被 Error Boundary 或等价机制隔离。
- 面向用户的错误信息必须脱敏。
- 开发日志不得打印 token、API key、完整敏感 payload。

---

## 20. Runtime Security Policy

通用安全规则：

- Runtime Capability 输入一律视为不可信。
- 参数通过 schema 校验不代表内容安全。
- 展示型 capability 不得产生服务端副作用。
- 交互型 capability 必须有明确用户动作。
- 副作用型 capability 默认禁用。
- 未知 `toolName` 不得执行。
- 未校验 args 不得传给组件。
- 错误信息不得泄漏 token、堆栈、内网地址或完整 prompt。

XSS 规则：

- 不把未知 HTML 注入主 DOM。
- Markdown 默认禁用 raw HTML。
- 文本展示默认转义。
- URL 字段默认只允许 `http`、`https`、`mailto` 等明确协议。
- 需要保留 HTML 结构时必须使用隔离或 sanitization 策略。

---

## 21. Contract-first 规则

新增 Runtime Capability 前必须先定义：

- `toolName`。
- 参数 schema。
- 注册项状态。
- 组件绑定方式。
- 行为类型。
- 风险等级。
- 是否需要确认。
- 失败策略。
- 安全边界。
- Review Checklist。

不得直接在组件中临时判断 `toolName` 并绕过 registry。

---

## 22. Review Checklist

### 通用性

- 是否没有通过 `agentName` 决定 Runtime Capability？
- 是否没有把某个 Agent 写成某个 capability 的专属来源？
- 是否只通过 `toolName` 查询 registry？
- 是否没有通过 Agent 名称绕过参数校验？

### Registry

- `toolName` 是否唯一？
- `toolName` 是否使用 `snake_case`？
- `status` 是否明确？
- `behavior` 是否明确？
- `riskLevel` 是否明确？
- `requiresConfirmation` 是否明确？
- `failureMode` 是否明确？
- 是否没有保留 `allowedInMvp` 这类历史阶段字段？

### Tool Call

- 是否按 `toolCallId` 聚合 args？
- 是否等待 `TOOL_CALL_END` 后解析？
- 是否兼容 `delta` / `content`？
- JSON parse 失败是否安全降级？
- 重复 END 是否不会重复执行？
- 未知 `toolName` 是否不会执行？

### Schema

- 是否有参数 schema？
- 是否校验必填字段？
- 是否限制字符串长度？
- 是否限制 URL 协议？
- 是否没有把未校验 args 传给组件？

### code_preview

- 是否只展示不执行？
- 是否不插入 script？
- 是否不访问网络？

### web_preview

- 是否 iframe 隔离？
- 是否没有直接注入主 DOM？
- 是否没有默认 `allow-same-origin`？
- 是否没有默认 `allow-scripts + allow-same-origin` 组合？
- 渲染失败是否不白屏？

### markdown_render

- 是否默认禁用 raw HTML？
- 链接是否安全？
- 渲染失败是否回退纯文本？

### 安全

- 是否没有暴露 token / stack trace / 内网地址？
- side_effect 是否默认 disabled？
- dangerous / high risk 能力是否有确认门？

---

## 23. 完成定义

本 Skill 完成时，必须满足：

- `SKILL.md` 独立可读。
- 不绑定具体 Agent 名称。
- 明确 Runtime Capability Registry。
- 明确 toolName / schema / component / behavior / riskLevel / failureMode。
- 明确 Tool Call 消费状态机。
- 明确 `TOOL_CALL_END` 后再解析参数。
- 明确 schema validation。
- 明确 `code_preview`、`web_preview`、`markdown_render` 是通用能力。
- 明确 reserved / disabled 能力不得执行。
- 明确 iframe / Markdown / Code 安全策略。
- 明确 ToolResult 边界。
- 明确 Review Checklist。

---

## References

- `references/runtime-capability-registry.md`
- `references/tool-call-consumption.md`
- `references/parameter-schema-policy.md`
- `references/component-binding-policy.md`
- `references/execution-behavior-policy.md`
- `references/code-preview-capability.md`
- `references/web-preview-capability.md`
- `references/markdown-render-capability.md`
- `references/failure-mode-policy.md`
- `references/tool-result-policy.md`
- `references/runtime-security-policy.md`
- `references/runtime-skill-review-checklist.md`
