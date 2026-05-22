# Frontend Runtime Skills Contract

版本：v0.2-redesign  
适用项目：AgentHub - 多 Agent 协作平台  
适用阶段：MVP + 正式开发演进  
事实源文件：

```text
<repo-root>/docs/contracts/frontend-runtime-skills.md
<repo-root>/docs/contracts/frontend-runtime-skills.schema.json
```

## 1. 目的

本文定义 AgentHub 前端 Runtime Skill 的项目级契约。

前端 Runtime Skill 是指：

```text
Agent 通过 AG-UI Tool Call 请求前端执行、展示或交互的能力。
```

核心链路：

```text
AG-UI Tool Call
→ Frontend Runtime Skill Registry
→ Args schema validation
→ React Component binding
→ Render / Interaction / ToolResult
```

本文只约束：

```text
AG-UI Tool Call → React Frontend Runtime Skill
```

## 2. 官方 / 上游协议约束

AG-UI 事件结构由 `agui-event-contract` 负责。

本文只消费 Tool Call 生命周期：

```text
TOOL_CALL_START
TOOL_CALL_ARGS
TOOL_CALL_END
```

或等价 AG-UI ToolCallStart / ToolCallArgs / ToolCallEnd 事件。

本文不重新定义 AG-UI SSE 事件字段。

## 3. Contract first 规则

任何新增或修改 Runtime Skill 行为前，必须先更新：

```text
<repo-root>/docs/contracts/frontend-runtime-skills.md
<repo-root>/docs/contracts/frontend-runtime-skills.schema.json
```

未更新 contract 的实现变更不得接受。

## 4. 边界

本文负责：

- Runtime Skill Registry。
- toolName 注册和状态。
- 参数 schema。
- React Component binding。
- 执行行为。
- blocking。
- requiresConfirmation。
- dangerous。
- ToolResult。
- fallback。

本文不负责：

- AG-UI 事件字段。
- A2A 到 AG-UI 转换。
- Artifact schema。
- Artifact 存储。
- Artifact type 到 Runtime Skill 的映射。
- REST API。
- iframe / 上传 / 下载完整安全策略。

## 5. Runtime Skill Registry

Runtime Skill 注册项：

```ts
type FrontendRuntimeSkill = {
  name: string;
  description: string;
  implemented: boolean;
  status: "implemented" | "reserved" | "disabled" | "deprecated";
  component: string;
  parametersSchema: JsonSchema;
  resultSchema?: JsonSchema;
  behavior: "render" | "interactive" | "side_effect";
  blocking: boolean;
  requiresConfirmation: boolean;
  dangerous: boolean;
  allowedInMvp: boolean;
  failureMode: "fallback_card" | "silent_ignore" | "tool_result_error";
};
```

长期预留 Skill：

```text
code_preview
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

## 6. MVP Registry

MVP 阶段只实现：

```text
code_preview
```

MVP 阶段其他 Skill 必须：

```text
implemented = false
status = reserved
allowedInMvp = false
```

## 7. Tool Call 消费流程

前端消费流程：

```text
1. TOOL_CALL_START:
   创建 pending tool call。

2. TOOL_CALL_ARGS:
   按 toolCallId 追加 args chunk。

3. TOOL_CALL_END:
   标记 args 完成。

4. parse JSON:
   只在 END 后解析完整 args。

5. schema validation:
   根据 toolName 找到 parametersSchema 校验。

6. registry validation:
   检查 implemented / disabled / dangerous / confirmation。

7. render or execute:
   选择 React Component。

8. result:
   如果 blocking / interactive，返回 ToolResult。
```

前端必须按 `toolCallId` 管理 pending calls。

## 8. 参数 schema 校验

每个 Runtime Skill 必须定义 `parametersSchema`。

参数必须在 `TOOL_CALL_END` 后校验。

必须拒绝：

- 缺少必填字段。
- 字段类型错误。
- 非法 enum。
- 未知危险字段。
- 超大 inline payload。
- 非法 URL。
- 非法 file reference。
- 违反安全策略的字段。

## 9. Component Binding

长期绑定：

```text
code_preview     → CodePreview
web_preview      → WebPreview
diff_preview     → DiffView
file_download    → FileCard
image_preview    → ImagePreview
deploy_status    → DeployCard
markdown_render  → MarkdownView
terminal_output  → TerminalOutput
chart_render     → ChartView
form_input       → FormInput
confirm_action   → ConfirmDialog
file_upload      → FileUpload
```

MVP 只实现：

```text
code_preview → CodePreview
```

## 10. Execution Behavior

行为类型：

```text
render
interactive
side_effect
```

`render`：

```text
只展示内容，不改变服务端状态。
```

`interactive`：

```text
需要用户输入、上传或确认。
```

`side_effect`：

```text
会触发外部副作用或修改资源。
```

`side_effect` 必须：

```text
requiresConfirmation = true
dangerous = true
```

## 11. ToolResult

展示型 `render` Skill 可以不返回 ToolResult。

交互型 `interactive` Skill 通常需要返回 ToolResult。

副作用型 `side_effect` Skill 必须返回 ToolResult。

推荐结构：

```ts
type FrontendToolResult = {
  toolCallId: string;
  skillName: string;
  status: "success" | "cancelled" | "failed";
  data?: unknown;
  error?: {
    code: string;
    safeMessage: string;
  };
};
```

## 12. Safety / Confirmation

以下行为必须要求用户确认：

- 外部副作用。
- 文件上传。
- 文件删除。
- 部署操作。
- 命令执行。
- 权限变更。
- 支付行为。
- 修改持久化资源。
- 访问敏感文件。
- 下载私有文件。

## 13. MVP: code_preview

`code_preview` 参数：

```ts
type CodePreviewParams = {
  code: string;
  language: string;
  filename: string;
};
```

注册项：

```json
{
  "name": "code_preview",
  "description": "Render read-only code content.",
  "implemented": true,
  "status": "implemented",
  "component": "CodePreview",
  "behavior": "render",
  "blocking": false,
  "requiresConfirmation": false,
  "dangerous": false,
  "allowedInMvp": true,
  "failureMode": "fallback_card"
}
```

`code_preview` 不得：

- 执行代码。
- eval 代码。
- 将代码插入 script。
- 修改服务端状态。
- 访问外部网络。
- 读取本地文件。
- 自动下载文件。
- 触发部署。

## 14. 正式开发扩展规则

新增或启用任何 Runtime Skill 必须明确：

- `name`
- `description`
- `implemented`
- `status`
- `component`
- `parametersSchema`
- `resultSchema`
- `behavior`
- `blocking`
- `requiresConfirmation`
- `dangerous`
- `allowedInMvp`
- `failureMode`

危险能力必须同时遵守：

```text
security-boundary-contract
```

## 15. 错误处理

必须安全处理：

- unknown toolName。
- implemented=false。
- disabled skill。
- malformed args JSON。
- schema validation failed。
- component render failed。
- user cancelled。
- dangerous operation blocked。

错误处理必须：

- 不执行该 Skill。
- 不让聊天 UI 崩溃。
- 显示安全 fallback。
- 记录结构化调试信息。
- 不暴露 stack trace、token 或内部服务地址。

## 16. 禁止事项

不得：

- 未更新 contract 就新增 Runtime Skill。
- 使用未注册 toolName。
- 执行 unknown toolName。
- 执行 implemented=false 的 Runtime Skill。
- 把 React Component 名称当作 toolName。
- 把 Artifact type 当作 toolName。
- Tool Call args 未校验就传入 React Component。
- 在 TOOL_CALL_ARGS 分片未结束前解析完整 JSON。
- 重复 TOOL_CALL_END 导致重复执行。
- 让 code_preview 执行代码。
- 让展示型 Runtime Skill 修改服务端状态。
- 在没有 confirmation gate 的情况下执行危险操作。
- 把 MVP code_preview 简化规则扩展成所有 Runtime Skill 的通用规则。

## 17. 相关契约

本文只引用以下契约，不重新定义它们：

- `agui-event-contract`
- `artifact-contract`
- `platform-api-contract`
- `security-boundary-contract`
- `testing-review-contract`
- `observability-debugging-contract`
- `code-style-and-conventions`
