# Execution Behavior 规则

## 1. 目的

本文定义 Runtime Skill 的执行行为分类。

## 2. 行为类型

Runtime Skill 行为分三类：

```text
render
interactive
side_effect
```

## 3. render

只展示内容，不改变服务端状态。

示例：

```text
code_preview
image_preview
markdown_render
terminal_output
chart_render
deploy_status
```

默认：

```text
blocking = false
requiresConfirmation = false
```

## 4. interactive

需要用户输入、上传或确认。

示例：

```text
form_input
confirm_action
file_upload
```

可能：

```text
blocking = true
requiresConfirmation = true
```

## 5. side_effect

会触发外部副作用或修改资源。

示例：

```text
future deploy_action
future file_delete
future command_execute
```

必须：

```text
requiresConfirmation = true
dangerous = true
```

## 6. deploy_status 说明

`deploy_status` 是状态展示，不等于执行部署。

如果后续需要执行部署，应另设 action 类 Skill，或通过 `confirm_action` 包裹。

## 7. 禁止事项

不得：

- 让 render-only Skill 修改服务端状态。
- 让 code_preview 执行代码。
- 让展示状态类 Skill 执行动作。
- 在没有 confirmation gate 时执行 side effect。
