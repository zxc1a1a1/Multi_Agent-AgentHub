# 08 企业微信接入方案

- 状态：待评审
- 目标：让 AgentHub 能接收企业微信消息并回推结果/审批卡片，同时保持 Gateway 为唯一外部入口

## 1. 两种接入模式

| 模式 | 方向 | 能力 | 适用 |
|---|---|---|---|
| 群机器人/消息推送 Webhook | AgentHub → 群 | 只推送，不收消息 | 通知、告警、运行结果 |
| 自建应用回调/智能机器人 | 双向 | 接收消息、回调、可回消息 | 对话、审批、任务执行 |

企业微信文档中的“回调配置”要求配置 `URL`、`Token`、`EncodingAESKey`；自建应用“接收消息”由企业微信服务器推送到开发者 URL。智能机器人文档还描述了消息推送与流式消息能力，具体以账号开通能力为准。

## 2. 推荐架构

```text
WeCom
  -> Gateway /api/integrations/wecom/callback
     verify signature + decrypt + dedup
     map external user/chat -> conversation
     create run (request_id derived from msg id)
  -> Orchestrator
  -> Gateway render reply
  -> WeCom send message API (or webhook for group push)
```

接入层不复制编排逻辑；只做协议、鉴权、去重、消息映射。

## 3. 配置项（环境变量，不含真实值）

```text
WECOM_CORP_ID=
WECOM_AGENT_ID=
WECOM_SECRET=
WECOM_CALLBACK_TOKEN=
WECOM_ENCODING_AES_KEY=
WECOM_WEBHOOK_URL=        # 群消息推送（可选）
WECOM_ENABLED=false
```

密钥只来自环境/Secret Manager；不得写入 compose、文档或日志。

## 4. 回调处理

1. 校验 `msg_signature`/timestamp/nonce 与 Token。
2. 使用 EncodingAESKey 解密，校验 CorpID。
3. 按 MsgId/EventId 去重（SQL 唯一键或 Redis SETNX + TTL）。
4. 将发送者映射为 AgentHub principal；会话按 `external_chat_id` 绑定。
5. 派生 `request_id = wecom:<msgid>` 保证幂等。
6. 异步执行 run；先 ACK，后推送结果（避免平台超时重试）。
7. 失败记录可观测事件，不返回内部错误细节。

## 5. 发送与审批

- 结果通知：文本/Markdown/卡片，走应用消息 API 或群 webhook。
- 审批：计划卡渲染为 WeCom 交互卡片；回调用 `action=approve/cancel/revise` 映射到 Gateway HITL。
- 一次性：审批 ID + 用户 + 计划修订绑定，重复回调返回已处理。
- 限流：按企业/会话维度限速；平台重试不得造成重复 run。

## 6. 安全边界

- 只接受配置 CorpID 的回调；拒绝未知来源。
- 验签失败直接 4xx，不解析业务 body。
- 日志脱敏：不记录 token、AESKey、完整消息体。
- 出站域名 allowlist；禁止把内部 URL/密钥发到企业微信。
- 外部用户不能直接触发写工具；仍需 AgentHub 审批/权限。

## 7. MVP 范围

| 阶段 | 内容 |
|---|---|
| M1 | 群 webhook 推送运行结果/告警 |
| M2 | 自建应用回调接收文本 → 触发 run → 回推结果 |
| M3 | 计划审批卡片 + 一次性回调 |
| M4 | 流式/分段消息、限流、审计与配额 |

## 8. 组件与依赖

- HTTP 客户端 + 标准库 crypto 实现验签/解密；不引入来源不明的闭源 SDK。
- 若采用社区 SDK：核验维护状态、许可证、是否读取环境密钥、是否内置遥测。
- 回调入口复用 Gateway 现有 auth/限流/审计。
- 消息去重表建议：`channel_events(channel, external_id, status, created_at)`。

## 9. 验收

- [ ] 验签失败 0 业务处理
- [ ] 重复 MsgId 只创建 1 个 run
- [ ] 回调 ACK 在平台超时前返回
- [ ] 审批重复点击只消费一次
- [ ] 日志无密钥/完整消息体
- [ ] 企业微信侧限流/重试不产生重复副作用

## 10. 来源

- 回调配置：https://developer.work.weixin.qq.com/document/path/90930
- 接收消息：https://developer.work.weixin.qq.com/document/path/100719
- 发送应用消息：https://developer.work.weixin.qq.com/document/path/90236
- 消息推送（群机器人）：https://developer.work.weixin.qq.com/document/path/91770
