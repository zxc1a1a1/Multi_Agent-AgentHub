# Execution Behavior Policy

## 行为分类

### render

只展示内容，不产生服务端副作用。

示例：`code_preview`、`web_preview`、`markdown_render`。

### interactive

需要用户动作，但不直接改变外部世界。

示例：`form_input`、`confirm_action`。

### side_effect

可能改变服务端、文件、部署、外部系统或用户数据。

示例：`file_upload`、`deploy_action`、`run_command`。

## 规则

- render 可以在参数合法后自动执行。
- interactive 必须等待用户动作。
- side_effect 默认 disabled。
- side_effect 必须有确认门、权限检查和审计记录。
