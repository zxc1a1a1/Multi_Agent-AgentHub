# Authorization Object Policy

## 目的

确保用户只能访问自己有权限的资源。

## 资源类型

- Conversation
- Message
- Run
- Artifact
- ToolCall
- Agent configuration
- Uploaded file

## 规则

- 所有资源读取、更新、删除、下载都必须校验 owner、participant 或显式授权关系。
- 群聊 participant 变更必须校验操作者权限。
- Artifact 下载必须绑定 conversation / run / user 权限。
- Agent 配置不得被无权限用户读取。
- 对象级授权失败不得泄露敏感存在性信息。
