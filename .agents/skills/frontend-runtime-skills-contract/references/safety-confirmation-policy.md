# Safety / Confirmation 规则

## 1. 目的

本文定义 Runtime Skill 的用户确认和危险操作规则。

## 2. requiresConfirmation

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

## 3. dangerous

以下能力必须标记为 dangerous：

- shell / command execution。
- file write / delete。
- deploy action。
- private object storage access。
- permission change。
- credential access。
- untrusted iframe execution。
- external network side effect。

## 4. confirm_action

`confirm_action` 是用户确认能力，不应自动执行实际危险动作。

危险动作必须由明确的后续安全流程处理。

## 5. file_upload

`file_upload` 必须遵守：

```text
security-boundary-contract
platform-api-contract
```

包括文件大小、类型、权限、扫描、存储和用户授权规则。

## 6. web_preview

`web_preview` 如果使用 iframe 或渲染外部 HTML，必须遵守 sandbox 和 CSP 规则。

## 7. MVP

MVP 阶段：

```text
code_preview.requiresConfirmation = false
code_preview.dangerous = false
```

MVP 不启用：

```text
file_upload
confirm_action
side_effect action
```

## 8. 禁止事项

不得：

- 在没有用户确认时执行危险操作。
- 用 confirm_action 伪装已经执行的动作。
- file_upload 绕过安全策略。
- web_preview 绕过 sandbox。
- 将用户确认结果伪造为 success。
