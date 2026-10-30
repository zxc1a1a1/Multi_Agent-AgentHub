# 09 飞书接入方案

- 状态：待评审
- 目标：支持飞书群通知、机器人对话与审批卡片，不把飞书协议侵入编排层

## 1. 两种模式

| 模式 | 方向 | 能力 | 适用 |
|---|---|---|---|
| 自定义机器人 Webhook | AgentHub → 群 | 只推送，支持文本/富文本/图片等 | 通知、告警、结果 |
| 企业自建应用/机器人 | 双向 | 接收消息事件、卡片回调、发消息 | 对话、审批、任务 |

飞书自定义机器人无需管理员审核，但仅适合推送；需要接收消息与卡片交互时，必须使用自建应用的事件订阅与卡片回调。

## 2. 推荐架构

```text
Feishu
  -> Gateway /api/integrations/feishu/events
     challenge/verification + decrypt + dedup
     map open_id/chat_id -> principal/conversation
     request_id = feishu:<event_id>
  -> Orchestrator
  -> Gateway render reply/card
  -> Feishu message/card API
```

## 3. 配置项（环境变量，不含真实值）

```text
FEISHU_APP_ID=
FEISHU_APP_SECRET=
FEISHU_VERIFICATION_TOKEN=
FEISHU_ENCRYPT_KEY=
FEISHU_WEBHOOK_URL=       # 自定义机器人（可选）
FEISHU_ENABLED=false
```

`tenant_access_token` 由服务端缓存并提前刷新；不得写入日志/前端。

## 4. 事件与卡片

1. 首次配置回调需处理 URL challenge。
2. 校验 Verification Token/签名；如启用加密，使用 Encrypt Key 解密。
3. 按 event_id 去重，保证同一事件只触发一次 run。
4. 文本消息 → run；卡片按钮 → `approve/cancel/revise` 映射 Gateway HITL。
5. 卡片回调必须校验操作者与消息归属，避免越权审批。
6. 回调快速 ACK；结果异步发送，避免飞书重试。

## 5. 消息与审批

- 通知：文本/富文本/卡片，按群或用户发送。
- 审批卡：计划摘要、参与者、风险、操作按钮；按钮携带 planId/revision。
- 一次性：planId + revision + 用户 + action 绑定；重复点击返回已处理。
- 失败：卡片显示可读错误，不暴露内部堆栈/URL/密钥。

## 6. 安全

- 验证 challenge 只用于配置，不触发业务。
- 验签/解密失败直接拒绝。
- open_id/chat_id 只作为外部标识，不写内部用户主键。
- 出站域名 allowlist；不在消息中放内部地址。
- 审批人权限需在 AgentHub 侧再次校验，不能只信飞书按钮。

## 7. MVP 范围

| 阶段 | 内容 |
|---|---|
| F1 | 自定义机器人推送运行结果 |
| F2 | 自建应用接收文本消息 → run → 回复 |
| F3 | 审批卡片 + 卡片回调 |
| F4 | 群/租户配额、审计、卡片模板 |

## 8. 组件

- HTTP + 标准库签名/解密；或采用飞书官方/社区 Go SDK（采用前核验版本、许可证、维护状态）。
- 复用 Gateway auth/限流/审计；新增 `channel_events` 去重表。
- 卡片模板版本化，避免平台结构变化影响业务。

## 9. 验收

- [ ] challenge 正确响应
- [ ] 验签/解密失败 0 业务处理
- [ ] 重复 event_id 只触发 1 次
- [ ] 卡片重复点击只消费 1 次
- [ ] 越权审批被拒
- [ ] 日志无 token/encrypt key/完整消息体

## 10. 来源

- 自定义机器人：https://open.feishu.cn/document/client-docs/bot-v3/add-custom-bot
- 卡片回调：https://open.feishu.cn/document/uAjLw4CM/ukzMukzMukzM/feishu-cards/handle-card-callbacks
- 应用配置说明：https://open.feishu.cn/document/develop-a-card-interactive-bot/faqs
