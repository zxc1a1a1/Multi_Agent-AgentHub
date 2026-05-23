# Runtime Skill Registry 规则

## 1. 目的

本文定义前端 Runtime Skill Registry 的规则。

Runtime Skill Registry 是前端判断某个 `toolName` 能否被识别、校验、绑定和执行的事实源。

## 2. 注册项结构

推荐注册项：

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

## 3. 命名规则

Runtime Skill 名称必须使用小写 snake_case。

合法：

```text
code_preview
web_preview
confirm_action
file_upload
```

非法：

```text
CodePreview
codePreview
preview_code
frontend.code.preview
```

## 4. 长期预留 Skill

长期预留：

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

## 5. 状态规则

`implemented`：

```text
已实现，可执行。
```

`reserved`：

```text
已预留，不可执行。
```

`disabled`：

```text
临时关闭，不可执行。
```

`deprecated`：

```text
不推荐使用，可按兼容策略处理。
```

## 6. MVP 规则

MVP 阶段：

```text
code_preview:
  implemented = true
  status = implemented
  allowedInMvp = true

其他 Runtime Skill:
  implemented = false
  status = reserved
  allowedInMvp = false
```

## 7. 禁止事项

不得：

- 执行未注册 toolName。
- 执行 implemented=false 的 Skill。
- 把 React Component 名称当作 toolName。
- 把 Artifact type 当作 toolName。
- 用用户输入动态生成 toolName。
