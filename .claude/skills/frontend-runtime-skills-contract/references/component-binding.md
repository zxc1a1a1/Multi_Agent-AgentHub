# Component Binding 规则

## 1. 目的

本文定义 Runtime Skill 到 React Component 的绑定规则。

## 2. 绑定原则

每个 implemented Runtime Skill 必须唯一绑定一个 React Component。

Runtime Skill 使用 `toolName` 识别。

React Component 名称不得作为 `toolName`。

正确链路：

```text
toolName
→ Runtime Skill Registry
→ parametersSchema validation
→ component binding
→ React Component render / execute
```

## 3. 长期组件绑定

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

## 4. MVP 绑定

MVP 阶段只实现：

```text
code_preview → CodePreview
```

其他绑定可以出现在 contract 中，但不得执行。

## 5. 组件错误

组件 render 失败时必须：

- 不崩溃聊天 UI。
- 显示安全 fallback。
- 记录结构化调试信息。
- 不暴露 stack trace 给用户。

## 6. 禁止事项

不得：

- Agent 直接指定 React Component。
- Artifact 直接携带 React Component 名称。
- 根据字符串猜测组件名并动态执行。
- implemented=false 的 Skill 绑定后仍执行。
