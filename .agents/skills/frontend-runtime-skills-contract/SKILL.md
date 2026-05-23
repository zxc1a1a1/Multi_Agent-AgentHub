---
name: frontend-runtime-skills-contract
description: 当定义、实现、修改或审查 AgentHub 中 AG-UI Tool Call 到前端 React Runtime Skill 的注册、参数校验、组件绑定、执行行为、ToolResult 或安全确认规则时，使用本 Skill。
---

# frontend-runtime-skills-contract

## 1. 目的

本 Skill 定义 AgentHub 前端 Runtime Skill 的开发契约。

前端 Runtime Skill 是指：

```text
Agent 通过 AG-UI Tool Call 请求前端执行、展示或交互的能力。
```

本 Skill 的边界是：

```text
AG-UI Tool Call
→ Frontend Runtime Skill Registry
→ Args schema validation
→ React Component binding
→ Render / Interaction / ToolResult
```

一句话定位：

```text
frontend-runtime-skills-contract 定义 AG-UI Tool Call 到前端 React Runtime Skill 的注册、参数校验、组件映射、执行边界和交互结果规则。
```

## 2. 官方 / 上游协议约束

涉及 AG-UI 事件和 Tool Call 语义时，以上游 AG-UI 协议为准。

本 Skill 不定义 AG-UI 事件结构。

本 Skill 只消费上游协议已经定义的 Tool Call 生命周期：

```text
TOOL_CALL_START
TOOL_CALL_ARGS
TOOL_CALL_END
```

或等价 AG-UI ToolCallStart / ToolCallArgs / ToolCallEnd 事件。

上游事件字段、SSE 格式、事件顺序和流式协议由：

```text
agui-event-contract
```

负责。

本 Skill 负责：

```text
前端在收到完整 Tool Call 后，如何识别 toolName、校验 args、绑定组件、执行或渲染、返回可选 ToolResult。
```

## 3. 文件位置说明

### 项目级 Contract 文件

以下文件属于 AgentHub 项目仓库，是项目级事实源：

```text
<repo-root>/docs/contracts/frontend-runtime-skills.md
<repo-root>/docs/contracts/frontend-runtime-skills.schema.json
```

### 当前 Skill 的参考文件

以下文件属于当前 Coding Agent Skill，用于补充本 Skill 的详细开发规则：

```text
<current-skill-dir>/references/registry-policy.md
<current-skill-dir>/references/tool-call-consumption.md
<current-skill-dir>/references/parameter-schema-policy.md
<current-skill-dir>/references/component-binding.md
<current-skill-dir>/references/execution-behavior.md
<current-skill-dir>/references/tool-result-policy.md
<current-skill-dir>/references/safety-confirmation-policy.md
<current-skill-dir>/references/mvp-code-preview.md
```

如果当前 Skill 安装在 Claude Code 项目目录中，则 `<current-skill-dir>` 通常是：

```text
<repo-root>/.claude/skills/frontend-runtime-skills-contract
```

## 4. 适用场景

当进行以下工作时，启用本 Skill：

- 新增前端 Runtime Skill。
- 修改前端 Runtime Skill registry。
- 修改 Tool Call 消费逻辑。
- 修改 Tool Call args 聚合逻辑。
- 修改 Runtime Skill 参数 schema。
- 修改 Runtime Skill 到 React Component 的绑定。
- 修改 Runtime Skill 的执行行为。
- 修改 blocking / interactive / side_effect 行为。
- 修改 ToolResult 结构。
- 修改用户确认和危险操作规则。
- 修改 MVP `code_preview`。
- 审查未知 toolName 的处理。
- 审查参数校验失败后的 fallback。
- 审查前端是否执行了不该执行的代码或副作用。

## 5. 本 Skill 负责

本 Skill 负责：

- Runtime Skill Registry。
- `toolName` 命名和注册规则。
- `implemented` / `reserved` / `disabled` / `deprecated` 状态。
- AG-UI Tool Call 的前端消费规则。
- Tool Call args chunk 聚合规则。
- 参数 JSON parse 和 schema validation。
- Runtime Skill 到 React Component 的绑定。
- render / interactive / side_effect 行为分类。
- blocking / non-blocking 规则。
- requiresConfirmation / dangerous 规则。
- ToolResult 规则。
- 用户安全 fallback。
- MVP `code_preview` 规则。
- 正式开发阶段启用新 Runtime Skill 的规则。

## 6. 本 Skill 不负责

本 Skill 不负责：

- AG-UI 事件名称和事件字段定义。
- SSE parser 具体实现。
- A2A 到 AG-UI 的转换。
- Artifact schema。
- Artifact 存储。
- Artifact type 到 Runtime Skill 的映射。
- REST API。
- React 视觉样式。
- Zustand / store 具体实现。
- 对象存储鉴权。
- iframe sandbox 完整策略。
- 文件上传完整安全策略。
- Docker / CI / 测试策略。
- 通用 TypeScript / React 代码风格。

涉及以上内容时，只引用相关 contract，不在本 Skill 中重新定义。

## 7. MVP Profile

MVP 阶段只实现一个前端 Runtime Skill：

```text
code_preview
```

MVP 阶段唯一可执行注册项：

```text
code_preview → CodePreview
```

MVP 阶段 `code_preview` 参数：

```ts
type CodePreviewParams = {
  code: string;
  language: string;
  filename: string;
};
```

MVP 行为：

```text
behavior = render
blocking = false
requiresConfirmation = false
dangerous = false
allowedInMvp = true
implemented = true
```

MVP 阶段其他 Runtime Skill 只能 reserved，不得执行：

```text
web_preview
diff_preview
file_download
image_preview
deploy_status
markdown_render
terminal_output
chart_render
form_input
confirm_action
file_upload
```

MVP 详细规则见：

```text
<current-skill-dir>/references/mvp-code-preview.md
```

## 8. 正式开发演进

正式开发阶段可以逐步启用新的 Runtime Skill。

启用任何新 Runtime Skill 前，必须先更新：

```text
<repo-root>/docs/contracts/frontend-runtime-skills.md
<repo-root>/docs/contracts/frontend-runtime-skills.schema.json
```

并同步检查：

```text
<current-skill-dir>/references/registry-policy.md
<current-skill-dir>/references/parameter-schema-policy.md
<current-skill-dir>/references/component-binding.md
<current-skill-dir>/references/execution-behavior.md
<current-skill-dir>/references/tool-result-policy.md
<current-skill-dir>/references/safety-confirmation-policy.md
```

新增或启用 Runtime Skill 必须明确：

- `name`
- `description`
- `implemented`
- `component`
- `parametersSchema`
- `resultSchema`
- `behavior`
- `blocking`
- `requiresConfirmation`
- `dangerous`
- `allowedInMvp`
- `failureMode`

涉及危险操作、文件上传、外部副作用、下载私有资源、iframe 或用户确认时，必须同时遵守：

```text
security-boundary-contract
```

## 9. 使用 references 的规则

修改不同区域时，应先读取对应 reference：

```text
新增或修改 registry:
  <current-skill-dir>/references/registry-policy.md

修改 Tool Call 消费:
  <current-skill-dir>/references/tool-call-consumption.md

修改参数 schema:
  <current-skill-dir>/references/parameter-schema-policy.md

修改组件绑定:
  <current-skill-dir>/references/component-binding.md

修改执行行为:
  <current-skill-dir>/references/execution-behavior.md

修改 ToolResult:
  <current-skill-dir>/references/tool-result-policy.md

修改确认和危险操作:
  <current-skill-dir>/references/safety-confirmation-policy.md

修改 MVP code_preview:
  <current-skill-dir>/references/mvp-code-preview.md
```

## 10. Contract first 规则

任何新增或修改 Runtime Skill 行为前，必须先更新项目级 contract：

```text
<repo-root>/docs/contracts/frontend-runtime-skills.md
<repo-root>/docs/contracts/frontend-runtime-skills.schema.json
```

未更新 contract 的实现变更不得接受。

不得先改前端实现，再补 contract。

## 11. 开发 / 审查 Checklist

开发或审查 Runtime Skill 时，必须确认：

```text
toolName 是否已注册？
skill 是否 implemented？
参数 schema 是否存在？
args 是否只在 TOOL_CALL_END 后解析？
args 是否通过 schema validation？
component 是否唯一绑定？
behavior 是否明确？
blocking 是否明确？
requiresConfirmation 是否明确？
dangerous 是否明确？
failureMode 是否明确？
未知 toolName 是否安全 fallback？
未实现 skill 是否安全 fallback？
参数非法是否不会执行？
```

MVP 阶段还必须确认：

```text
只有 code_preview 是 implemented=true。
其他 Runtime Skill 均为 implemented=false。
code_preview 不执行代码。
code_preview 不访问网络。
code_preview 不修改服务端状态。
```

## 12. 禁止事项

Coding Agent 不得：

- 未更新 contract 就新增 Runtime Skill。
- 使用未注册 toolName。
- 执行 unknown toolName。
- 执行 `implemented=false` 的 Runtime Skill。
- 把 React Component 名称当作 toolName。
- 把 Artifact type 当作 toolName。
- 把 Tool Call args 未校验就传入 React Component。
- 在 `TOOL_CALL_ARGS` 分片未结束前解析完整 JSON。
- 重复 `TOOL_CALL_END` 导致重复执行。
- 让参数校验失败的 Tool Call 继续执行。
- 让 `code_preview` 执行代码。
- 让 `code_preview` 插入 script。
- 让展示型 Runtime Skill 修改服务端状态。
- 在没有 confirmation gate 的情况下执行危险操作。
- 在前端错误中暴露 stack trace、token 或内部服务地址。
- 把 MVP 的 `code_preview` 简化规则扩展成所有 Runtime Skill 的通用规则。

## 13. 相关契约

本 Skill 只引用以下契约，不重新定义它们：

- `agui-event-contract`：负责 AG-UI 事件名称、事件字段和流式协议。
- `artifact-contract`：负责 Artifact schema、生命周期、存储和 artifact.type 到 Runtime Skill 的映射。
- `platform-api-contract`：负责大型 Artifact 内容查询、下载和 REST API。
- `security-boundary-contract`：负责危险操作、确认、iframe、下载、上传、sandbox 和鉴权策略。
- `testing-review-contract`：负责 Tool Call lifecycle、schema validation 和 Runtime Skill fallback 的测试要求。
- `observability-debugging-contract`：负责 toolCallId、artifactId、runId、traceId 的日志和调试规则。
- `code-style-and-conventions`：负责 TypeScript、React、lint 和格式化规则。
