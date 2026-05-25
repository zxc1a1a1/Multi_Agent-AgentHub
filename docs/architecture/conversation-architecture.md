# Conversation Architecture

Conversation 支持 `single` 与 `group`。

Group conversation 允许多个 Agent participant。用户输入可以包含 @mention，但 @mention 只是路由提示，最终仍由 Orchestrator 校验并生成计划。

一个 run 可以产生多条 Agent message，每条消息必须保留 senderName / senderId。
