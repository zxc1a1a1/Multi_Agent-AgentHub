# E2E Smoke 测试

## 1. 目的

本文定义 AgentHub MVP 和正式开发阶段的 E2E smoke 测试。

## 2. MVP Smoke Path

MVP 必测：

```text
docker compose up
open page
new conversation
select code-agent
send prompt
streaming reply visible
code preview visible
copy code
refresh page
history still visible
```

## 3. 推荐断言

断言：

- 页面可打开。
- 输入框可输入。
- 消息发送后出现用户消息。
- Agent 回复有流式内容。
- CodePreview 出现。
- CodePreview 显示 filename 和 language。
- Copy code 可点击。
- 刷新后历史消息恢复。
- 控制台无严重错误。
- 网络请求无 500。

## 4. 测试数据

E2E 应使用 fake LLM 或稳定 mock response。

不得依赖真实 LLM 输出。

## 5. 禁止事项

不得：

- 用固定 sleep 代替等待状态。
- 依赖不稳定 DOM selector。
- 依赖真实外部 API。
- E2E 中使用真实 API key。
