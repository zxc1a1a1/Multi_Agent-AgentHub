# 工具注册和权限规则

## 1. 目的

本文定义子 Agent 使用工具时的注册和权限规则。

工具是子 Agent 可调用的外部能力，可能带来安全风险。

## 2. 工具注册字段

每个工具必须声明：

- name
- description
- input schema
- output schema
- permissions
- timeout
- side effects
- dangerous
- requiresConfirmation

## 3. 权限类型

工具权限包括但不限于：

```text
network
filesystem
shell
workspace_write
deploy
object_storage
external_api
```

## 4. MVP 规则

MVP 阶段 `code-agent` 默认：

```text
tools.enabled = false
```

不得在 MVP 阶段默认启用 shell、filesystem、network、deploy 等工具。

## 5. 危险工具

以下工具必须经过安全审查和用户确认策略：

- shell 命令。
- 文件写入。
- 文件删除。
- 网络请求。
- 部署操作。
- 对象存储上传 / 下载。
- 外部 API 调用。
- 权限变更。

## 6. 相关契约

危险工具必须遵守：

```text
security-boundary-contract
```

## 7. 禁止事项

不得：

- 未注册工具就调用。
- 未声明权限就调用工具。
- 无 timeout 调用工具。
- 在没有 sandbox 的情况下执行 shell。
- 在没有根目录限制的情况下访问文件系统。
- 在没有确认的情况下执行部署或删除。
- 把工具错误原样暴露给用户。
