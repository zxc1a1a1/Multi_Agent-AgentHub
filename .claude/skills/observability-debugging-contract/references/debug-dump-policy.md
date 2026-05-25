# Debug Dump Policy

## 目的

定义 AgentHub 排障快照的范围和安全边界。

## Debug Dump 是什么

Debug dump 是用于排障的结构化快照。

它不是完整数据库导出，也不是完整 prompt dump。

## 允许字段

- `traceId`
- `requestId`
- `runId`
- `conversationId`
- `planId`
- `stepId`
- `agentTaskId`
- `strategy`
- `planningMode`
- `selectedAgentNames`
- `event timeline`
- `errorCode`
- `safeMessage`
- `durationMs`
- provider/model 摘要
- artifact/toolCall 引用

## 禁止字段

- 完整 prompt
- 完整用户隐私输入
- API key
- token
- 原始 Provider 响应
- 数据库连接串
- 签名 URL

## 规则

- debug dump 默认关闭或受控开启。
- debug dump 必须脱敏。
- debug dump 不得作为长期事实源。
