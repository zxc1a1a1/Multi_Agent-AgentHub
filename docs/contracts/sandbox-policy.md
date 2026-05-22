# Sandbox Policy

## 1. 文档目的

本文档定义 AgentHub 中 sandbox、iframe、run_command、file_upload、workspace 访问和高危操作确认规则。

## 2. code_preview

MVP v0.1 只实现 `code_preview`。

规则：

- 只展示代码。
- 不执行代码。
- 不自动运行命令。
- 不访问用户文件。
- 不读取密钥。
- 不发起网络请求。

## 3. web_preview

Post-MVP 如果实现 `web_preview`，必须使用 iframe sandbox。

建议：

```html
<iframe sandbox="allow-scripts">
```

谨慎增加：

```text
allow-forms
allow-downloads
allow-same-origin
```

默认禁止：

```text
allow-top-navigation
allow-popups
allow-modals
```

规则：

- iframe 内容不得访问父页面 token。
- iframe 不得共享主站 localStorage。
- 用户生成 HTML 不得直接进入主 DOM `innerHTML`。
- 预览与主应用必须隔离。

## 4. run_command

MVP v0.1 不实现 `run_command`。

Post-MVP 如果支持，必须：

- sandbox。
- workspace 根目录限制。
- 命令白名单。
- timeout。
- 输出长度限制。
- 禁止访问系统敏感路径。
- 禁止网络扫描。
- 禁止读取密钥文件。
- 禁止持久后台进程。
- 必须用户确认或策略允许。

## 5. file_upload

MVP v0.1 不实现 `file_upload`。

Post-MVP 如果支持，必须：

- 文件大小限制。
- MIME type 校验。
- 后缀校验。
- 病毒扫描或安全扫描。
- 私有存储。
- 用户权限校验。
- 不允许直接作为可执行文件运行。
- 不允许直接注入 prompt 而不做过滤。

## 6. confirm_action

以下操作必须走 `confirm_action`：

- 部署。
- 命令执行。
- 文件覆盖。
- 删除重要资源。
- 调用外部服务产生费用。
- 访问敏感文件。
- 发布自建 Agent。

规则：

- 高危操作必须阻塞 run，等待用户确认。
- LLM 不得绕过确认直接执行。
- 审批记录 Post-MVP 应进入 APPROVAL 表。
